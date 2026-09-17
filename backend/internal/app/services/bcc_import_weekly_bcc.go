package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	excelparser "api-server/internal/app/services/excel"
	"api-server/internal/domain"
	domainservices "api-server/internal/domain/services"

	"github.com/xuri/excelize/v2"
)

// processWeeklyBCCUpload handles the BCC-<shiftType> sheet format (e.g.
// BCC-HC, BCC-OT150). Each sheet represents one shift type for all employees
// doing that shift. Separate timesheet entries are created per (employee,
// date, shiftType). Format-specific steps live in the named methods below;
// the shared pipeline pieces are in bcc_import_pipeline.go.
func (s *BCCImportService) processWeeklyBCCUpload(
	ctx context.Context,
	xf *excelize.File,
	formatResult *excelparser.FormatDetectionResult,
	filename string,
	projectID uint,
	uploaderID uint,
	uploaderRole string,
	createdAsset *domain.Asset,
	effectiveMonth string,
	includeFlexibleEmployees bool,
) (*BCCImportResult, error) {
	fail := s.bccFailer(ctx, createdAsset, uploaderID, projectID, filename, effectiveMonth)

	// 1. Parse all BCC-* sheets.
	parsed, err := excelparser.ParseWeeklyBCCFile(xf, formatResult.WeeklyBCCSheets)
	if err != nil {
		return fail(fmt.Sprintf("lỗi phân tích file BCC tuần: %v", err))
	}

	// 2-3. Shared setup: month parsing, lock, payrate lookup. The earliest
	// worked day lets the lookup accept a payrate that starts mid-month but
	// is already active on the file's first worked day.
	earliestDay := 0
	for _, sh := range parsed.Sheets {
		for _, emp := range sh.Employees {
			for _, e := range emp.Entries {
				if d := e.Date.Day(); d > 0 && (earliestDay == 0 || d < earliestDay) {
					earliestDay = d
				}
			}
		}
	}
	ictx, releaseLock, err := s.prepareImportContext(ctx, projectID, effectiveMonth, earliestDay)
	if err != nil {
		return fail(err.Error())
	}
	defer releaseLock()
	year, month, monthStart, flatRates := ictx.year, ictx.month, ictx.monthStart, ictx.flatRates
	loc := monthStart.Location()

	// 4. Resolve payrates for the ACTUAL timesheet dates in the file, not monthStart.
	rates := newWeeklyRateResolver(s, ctx, projectID)

	// 4b. Every sheet's shift type must exist in the payrate config active
	// for the dates that sheet actually has entries on. Zero-hour entries
	// (deletion requests) carry no rate burden.
	if reason := weeklyBCCConfigError(parsed, year, month, loc, rates.forDate); reason != "" {
		return fail(reason)
	}

	// 5. STK auto-creation.
	blockedEmployeeCCCDs := make(map[string]struct{})
	availablePositions := getPositions(flatRates)
	sort.Strings(availablePositions)
	defaultPosition := ""
	if len(availablePositions) > 0 {
		defaultPosition = availablePositions[0]
	}

	stkNameByCCCD, stkRows, importErrors, failReason := s.autoCreateWeeklyBCCSTK(
		ctx, xf, defaultPosition, projectID, uploaderID, year, month, loc, blockedEmployeeCCCDs)
	if failReason != "" {
		return fail(failReason)
	}

	// 6. Load all active project assignments.
	assignments, err := s.employeeService.GetActiveAssignments(ctx, projectID)
	if err != nil {
		return fail(fmt.Sprintf("lỗi tải danh sách nhân viên: %v", err))
	}
	byCCCD := make(map[string]*domain.ProjectEmployee, len(assignments))
	empNames := make(map[uint]string, len(assignments))
	for _, a := range assignments {
		byCCCD[a.EmployeeCCCD] = a
		empNames[a.EmployeeID] = a.EmployeeName
	}

	// 6.5. Auto-create employees that appear in BCC sheets but don't exist in
	// the project yet. The STK sheet may cover a different set of employees —
	// BCC employees must also be created.
	autoErrors, createdAssignments := s.autoCreateMissingWeeklyBCCEmployees(
		ctx, parsed, stkRows, byCCCD, empNames, blockedEmployeeCCCDs,
		defaultPosition, year, month, loc, projectID, uploaderID)
	importErrors = append(importErrors, autoErrors...)
	assignments = append(assignments, createdAssignments...)

	// 7. Build timesheet entries for each shift-type sheet.
	entries, flexibleEmployeeIDs, totalRows, entryErrors := s.buildWeeklyBCCEntries(
		ctx, parsed, year, month, loc, byCCCD, empNames, stkNameByCCCD,
		blockedEmployeeCCCDs, rates.forDate, projectID, includeFlexibleEmployees)
	importErrors = append(importErrors, entryErrors...)

	// 7b. Align assignment start dates with the entries this file proves.
	s.backdateAssignmentsForImport(ctx, assignments, entries, uploaderID, filename)

	// 8. Preserve reviewed rows, replace pending rows, and create missing rows.
	// HourType is part of the key so an OT import cannot replace HC.
	entries, staleIDs, protectedSkippedCount, flexibleSkippedCount, replErr := s.planMonthReplacement(
		ctx, projectID, year, month, monthStart, loc, entries, flexibleEmployeeIDs, true)
	if replErr != nil {
		return fail(fmt.Sprintf("lỗi tải bảng chấm công hiện có: %v", replErr))
	}

	// 9. Bulk create or return failure.
	if len(entries) == 0 {
		return s.finishEmptyBCCImport(ctx, createdAsset, uploaderID, projectID, filename, effectiveMonth,
			totalRows, protectedSkippedCount, flexibleSkippedCount, importErrors,
			"không có dữ liệu hợp lệ để tạo bảng chấm công")
	}

	result, err := s.applyTimesheetReplacement(ctx, staleIDs, entries, uploaderID, uploaderRole)
	if err != nil {
		// Row-level failures must reach the uploader (employee + date + safe
		// reason); the generic "N dòng không hợp lệ" gives nothing to correct.
		if result != nil && len(result.FailedEntries) > 0 {
			importErrors = append(importErrors, importErrorsFromBulkFailures(result.FailedEntries, empNames)...)
			return s.failWithImportErrors(ctx, createdAsset, uploaderID, BCCImportStats{
				ProjectID:    projectID,
				OriginalName: filename,
				ForMonth:     effectiveMonth,
				TotalRows:    totalRows,
			}, importErrors)
		}
		return fail(fmt.Sprintf("lỗi tạo bảng chấm công: %v", err))
	}

	// 10. Finalize.
	// Map bulk-create failures through the shared helper so each error carries
	// the employee's name and the affected date instead of a technical
	// "employee_id=…" string the UI strips as unsafe detail.
	importErrors = append(importErrors, importErrorsFromBulkFailures(result.FailedEntries, empNames)...)
	return s.finalizeCreatedBCCImport(ctx, createdAsset, uploaderID, projectID, filename, effectiveMonth,
		totalRows, len(result.CreatedTimesheets), protectedSkippedCount, flexibleSkippedCount,
		len(result.DeletedTimesheets), importErrors)
}

// weeklyBCCConfigError returns a failure reason when a sheet's shift type is
// missing from the payrate config active on its worked dates, or "".
func weeklyBCCConfigError(
	parsed *excelparser.WeeklyBCCImportData,
	year int, month time.Month, loc *time.Location,
	flatRatesFor func(time.Time) map[string]int,
) string {
	for _, sheet := range parsed.Sheets {
		checkedDates := make(map[string]bool)
		for _, emp := range sheet.Employees {
			for _, entry := range emp.Entries {
				if entry.Hours <= 0 {
					continue
				}
				realDate := time.Date(year, month, entry.Date.Day(), 0, 0, 0, 0, loc)
				if realDate.Month() != month {
					continue
				}
				dateKey := realDate.Format("2006-01-02")
				if checkedDates[dateKey] {
					continue
				}
				checkedDates[dateKey] = true
				fr := flatRatesFor(realDate)
				if len(fr) == 0 {
					return fmt.Sprintf("không tìm thấy cấu hình lương cho ngày %s", dateKey)
				}
				if !shiftInConfig(fr, sheet.ShiftType) {
					return fmt.Sprintf("ca làm \"%s\" không có trong cấu hình lương cho ngày %s. Các ca làm khả dụng: %s",
						sheet.ShiftType, dateKey, strings.Join(getShiftTypes(fr), ", "))
				}
			}
		}
	}
	return ""
}

// autoCreateWeeklyBCCSTK upserts employees from the STK sheet with the
// default position (first configured position, sorted). Rows whose CCCD
// already recorded a primary error are skipped so a duplicate STK row cannot
// repeat it. Returns the STK name lookup, the raw rows (used later to fill
// bank info for BCC-only hires), collected errors, and a fatal fail reason
// when the transaction itself failed.
func (s *BCCImportService) autoCreateWeeklyBCCSTK(
	ctx context.Context,
	xf *excelize.File,
	defaultPosition string,
	projectID uint,
	uploaderID uint,
	year int, month time.Month, loc *time.Location,
	blockedEmployeeCCCDs map[string]struct{},
) (stkNameByCCCD map[string]string, stkRows []excelparser.STKRow, importErrors []domain.ImportError, failReason string) {
	stkRows, stkErr := excelparser.ParseSTKSheet(xf)
	if stkErr != nil {
		slog.Warn("BCCImport(WBCC): failed to parse STK sheet", "error", stkErr)
	}
	stkNameByCCCD = weeklySTKNameLookup(stkRows)

	if len(stkRows) == 0 {
		return stkNameByCCCD, stkRows, nil, ""
	}

	slog.Info("BCCImport(WBCC): found STK sheet, processing employee auto-creation",
		"count", len(stkRows))

	stkErr = s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		bankCache := make(map[string]*uint)

		for _, row := range stkRows {
			cccd := row.CCCD
			fullName := row.FullName
			if cccd == "" || fullName == "" {
				continue
			}
			if isWeeklyBCCEmployeeBlocked(blockedEmployeeCCCDs, cccd) {
				continue // a duplicate STK row must not repeat the primary error
			}

			var bankID *uint
			if row.BankName != "" {
				if cached, ok := bankCache[row.BankName]; ok {
					bankID = cached
				} else {
					bankID = s.employeeService.ResolveBankID(txCtx, row.BankName)
					bankCache[row.BankName] = bankID
				}
			}

			existingEmp, empErr := s.employeeService.GetEmployeeByCCCD(txCtx, cccd)
			if empErr != nil && !domain.IsNotFoundError(empErr) {
				blockedEmployeeCCCDs[cccd] = struct{}{}
				importErrors = append(importErrors, domain.ImportError{
					Employee: fullName,
					Reason:   fmt.Sprintf("lỗi tra cứu nhân viên CCCD %s: %v", cccd, empErr),
				})
				continue
			}

			var emp *domain.Employee
			if existingEmp == nil {
				emp = &domain.Employee{
					Fullname:  fullName,
					CCCD:      cccd,
					Mobile:    row.Mobile,
					CreatedBy: uploaderID,
				}
				// Attach bank fields only when STK provides enough info to
				// build a complete record; otherwise the profile is created
				// without banking info and can be filled in later.
				applySTKBankFields(emp, row, bankID, fullName)
				createdEmp, createErr := s.employeeService.CreateEmployeeFromImport(txCtx, emp, uploaderID)
				if createErr != nil {
					blockedEmployeeCCCDs[cccd] = struct{}{}
					importErrors = append(importErrors, domain.ImportError{
						Employee: fullName,
						Reason:   fmt.Sprintf("không thể tạo nhân viên CCCD %s: %v", cccd, createErr),
					})
					continue
				}
				emp = createdEmp
			} else {
				emp = existingEmp
				if bankUpdates := buildSTKBankUpdates(emp, row, bankID, fullName); bankUpdates != nil {
					if updateErr := s.employeeService.UpdateBankInfo(txCtx, emp.ID, bankUpdates); updateErr != nil {
						slog.Error("BCCImport(WBCC): failed to update bank info for employee",
							"employee_id", emp.ID, "error", updateErr)
						importErrors = append(importErrors, domain.ImportError{
							Employee: fullName,
							Reason:   "Không thể cập nhật thông tin ngân hàng",
						})
					}
				}

				// Fill missing mobile from STK when employee has none.
				if row.Mobile != "" && emp.Mobile == "" {
					if err := s.employeeService.UpdateMobile(txCtx, emp.ID, row.Mobile); err != nil {
						slog.Error("BCCImport(WBCC): failed to fill mobile for existing employee",
							"employee_id", emp.ID, "error", err)
					}
				}
				if emp.UserID == nil {
					if err := s.ensureEmployeeUserAccount(txCtx, emp.ID); err != nil {
						blockedEmployeeCCCDs[cccd] = struct{}{}
						importErrors = append(importErrors, domain.ImportError{
							Employee: fullName,
							Reason:   fmt.Sprintf("không thể tạo tài khoản nhân viên CCCD %s: %v", cccd, err),
						})
						continue
					}
				}
			}

			// Ensure employee is assigned to the project.
			existingAssignment, assignErr := s.employeeService.GetActiveAssignment(txCtx, projectID, emp.ID)
			if assignErr != nil && !domain.IsNotFoundError(assignErr) {
				blockedEmployeeCCCDs[cccd] = struct{}{}
				importErrors = append(importErrors, domain.ImportError{
					Employee: fullName,
					Reason:   fmt.Sprintf("lỗi kiểm tra phân công nhân viên %s: %v", fullName, assignErr),
				})
				continue
			}
			if existingAssignment == nil {
				// Continue coverage from the employee's last recorded timesheet
				// instead of the import month start, so uploads covering earlier
				// days still pass assignment validation.
				startDate, suggestErr := s.employeeService.SuggestAssignmentStart(txCtx, emp.ID)
				if suggestErr != nil {
					blockedEmployeeCCCDs[cccd] = struct{}{}
					importErrors = append(importErrors, domain.ImportError{
						Employee: fullName,
						Reason:   fmt.Sprintf("lỗi xác định ngày bắt đầu cho %s: %v", fullName, suggestErr),
					})
					continue
				}
				assignment := &domain.ProjectEmployee{
					ProjectID:       projectID,
					EmployeeID:      emp.ID,
					EmployeeName:    emp.Fullname,
					EmployeeCCCD:    emp.CCCD,
					Position:        defaultPosition,
					StartDate:       startDate,
					PaymentSchedule: string(domain.PaymentScheduleWeekly),
					// New assignments start advance-request enabled (explicit;
					// guards later full-row Saves from persisting the zero value).
					AdvanceRequestEnabled: true,
					CreatedBy:             uploaderID,
				}
				if createErr := s.employeeService.CreateAssignment(txCtx, assignment); createErr != nil {
					blockedEmployeeCCCDs[cccd] = struct{}{}
					importErrors = append(importErrors, domain.ImportError{
						Employee: fullName,
						Reason:   fmt.Sprintf("không thể phân công nhân viên %s: %v", fullName, createErr),
					})
				}
			}
		}
		return nil
	})
	if stkErr != nil {
		slog.Error("BCCImport(WBCC): STK auto-creation transaction failed", "error", stkErr)
		// Propagate as import error so the user sees a clear message.
		return stkNameByCCCD, stkRows, importErrors, fmt.Sprintf("lỗi tự động tạo nhân viên từ STK: %v", stkErr)
	}

	return stkNameByCCCD, stkRows, importErrors, ""
}

// autoCreateMissingWeeklyBCCEmployees creates employees that appear in the
// BCC sheets but have no project assignment yet, filling bank info from STK
// when available. byCCCD and empNames are updated in place so subsequent
// entry building finds the new hires. Returns the created assignments so the
// import's backdate pass can align their start dates too.
func (s *BCCImportService) autoCreateMissingWeeklyBCCEmployees(
	ctx context.Context,
	parsed *excelparser.WeeklyBCCImportData,
	stkRows []excelparser.STKRow,
	byCCCD map[string]*domain.ProjectEmployee,
	empNames map[uint]string,
	blockedEmployeeCCCDs map[string]struct{},
	defaultPosition string,
	year int, month time.Month, loc *time.Location,
	projectID uint,
	uploaderID uint,
) ([]domain.ImportError, []*domain.ProjectEmployee) {
	var importErrors []domain.ImportError
	var createdAssignments []*domain.ProjectEmployee

	var missingCCCDs []struct {
		cccd     string
		fullName string
	}
	seenMissing := make(map[string]bool)
	for _, sheet := range parsed.Sheets {
		for _, emp := range sheet.Employees {
			if emp.EmployeeCode == "" {
				continue
			}
			if _, exists := byCCCD[emp.EmployeeCode]; exists {
				continue // already assigned
			}
			if isWeeklyBCCEmployeeBlocked(blockedEmployeeCCCDs, emp.EmployeeCode) {
				continue // a primary creation/assignment error was already recorded
			}
			if seenMissing[emp.EmployeeCode] {
				continue
			}
			seenMissing[emp.EmployeeCode] = true
			missingCCCDs = append(missingCCCDs, struct {
				cccd     string
				fullName string
			}{emp.EmployeeCode, emp.FullName})
		}
	}

	if len(missingCCCDs) == 0 {
		return nil, nil
	}

	slog.Info("BCCImport(WBCC): auto-creating missing employees from BCC sheets",
		"count", len(missingCCCDs))

	autoErr := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		for _, m := range missingCCCDs {
			// Check if employee exists globally by CCCD.
			existingEmp, empErr := s.employeeService.GetEmployeeByCCCD(txCtx, m.cccd)
			if empErr != nil && !domain.IsNotFoundError(empErr) {
				blockedEmployeeCCCDs[m.cccd] = struct{}{}
				importErrors = append(importErrors, domain.ImportError{
					Employee: m.fullName,
					Reason:   fmt.Sprintf("lỗi tra cứu nhân viên CCCD %s: %v", m.cccd, empErr),
				})
				continue
			}

			var emp *domain.Employee
			if existingEmp == nil {
				// Create new employee.
				emp = &domain.Employee{
					Fullname:  m.fullName,
					CCCD:      m.cccd,
					CreatedBy: uploaderID,
				}
				// Fill bank info from STK if available.
				if stkRow := findSTKRow(stkRows, m.cccd); stkRow != nil {
					// Resolve bank id first so applySTKBankFields can decide
					// whether the row has enough info for a complete record.
					var stkBankID *uint
					if stkRow.BankName != "" {
						stkBankID = s.employeeService.ResolveBankID(txCtx, stkRow.BankName)
					}
					applySTKBankFields(emp, *stkRow, stkBankID, m.fullName)
					if stkRow.Mobile != "" {
						emp.Mobile = stkRow.Mobile
					}
				}
				createdEmp, createErr := s.employeeService.CreateEmployeeFromImport(txCtx, emp, uploaderID)
				if createErr != nil {
					blockedEmployeeCCCDs[m.cccd] = struct{}{}
					importErrors = append(importErrors, domain.ImportError{
						Employee: m.fullName,
						Reason:   fmt.Sprintf("không thể tạo nhân viên CCCD %s: %v", m.cccd, createErr),
					})
					continue
				}
				emp = createdEmp
				slog.Info("BCCImport(WBCC): auto-created employee from BCC sheet",
					"cccd", m.cccd, "employee_id", emp.ID)

			} else {
				emp = existingEmp
				// Apply changed bank info from STK if available.
				if stkRow := findSTKRow(stkRows, m.cccd); stkRow != nil {
					var stkBankID *uint
					if stkRow.BankName != "" {
						stkBankID = s.employeeService.ResolveBankID(txCtx, stkRow.BankName)
					}
					if bankUpdates := buildSTKBankUpdates(emp, *stkRow, stkBankID, m.fullName); bankUpdates != nil {
						if updateErr := s.employeeService.UpdateBankInfo(txCtx, emp.ID, bankUpdates); updateErr != nil {
							slog.Error("BCCImport(WBCC): failed to update bank info",
								"employee_id", emp.ID, "error", updateErr)
							importErrors = append(importErrors, domain.ImportError{
								Employee: m.fullName,
								Reason:   "Không thể cập nhật thông tin ngân hàng",
							})
						}
					}
					if stkRow.Mobile != "" && emp.Mobile == "" {
						if err := s.employeeService.UpdateMobile(txCtx, emp.ID, stkRow.Mobile); err != nil {
							slog.Error("BCCImport(WBCC): failed to fill mobile",
								"employee_id", emp.ID, "error", err)
						}
					}
				}
			}

			if emp.UserID == nil {
				if err := s.ensureEmployeeUserAccount(txCtx, emp.ID); err != nil {
					blockedEmployeeCCCDs[m.cccd] = struct{}{}
					importErrors = append(importErrors, domain.ImportError{
						Employee: m.fullName,
						Reason:   fmt.Sprintf("không thể tạo tài khoản nhân viên CCCD %s: %v", m.cccd, err),
					})
					continue
				}
			}

			// Ensure employee is assigned to the project. Continue coverage
			// from the employee's last recorded timesheet instead of the import
			// month start, so uploads covering earlier days still validate.
			startDate, suggestErr := s.employeeService.SuggestAssignmentStart(txCtx, emp.ID)
			if suggestErr != nil {
				blockedEmployeeCCCDs[m.cccd] = struct{}{}
				importErrors = append(importErrors, domain.ImportError{
					Employee: m.fullName,
					Reason:   fmt.Sprintf("lỗi xác định ngày bắt đầu cho %s: %v", m.fullName, suggestErr),
				})
				continue
			}
			assignment := &domain.ProjectEmployee{
				ProjectID:       projectID,
				EmployeeID:      emp.ID,
				EmployeeName:    emp.Fullname,
				EmployeeCCCD:    emp.CCCD,
				Position:        defaultPosition,
				StartDate:       startDate,
				PaymentSchedule: string(domain.PaymentScheduleWeekly),
				// New assignments start advance-request enabled (explicit;
				// guards later full-row Saves from persisting the zero value).
				AdvanceRequestEnabled: true,
				CreatedBy:             uploaderID,
			}
			if createErr := s.employeeService.CreateAssignment(txCtx, assignment); createErr != nil {
				blockedEmployeeCCCDs[m.cccd] = struct{}{}
				importErrors = append(importErrors, domain.ImportError{
					Employee: m.fullName,
					Reason:   fmt.Sprintf("không thể phân công nhân viên %s: %v", m.fullName, createErr),
				})
				continue
			}
			// Update lookup maps so subsequent processing finds this employee.
			byCCCD[emp.CCCD] = assignment
			empNames[emp.ID] = emp.Fullname
			createdAssignments = append(createdAssignments, assignment)
		}
		return nil
	})
	if autoErr != nil {
		slog.Error("BCCImport(WBCC): BCC employee auto-creation transaction failed", "error", autoErr)
	}

	return importErrors, createdAssignments
}

// buildWeeklyBCCEntries converts parsed shift-type sheets into bulk timesheet
// entries. Rate resolution: the sheet's shift type is a payrate leaf key; the
// path is "{employee_position}.{ngày thường}.{shiftType}" (all entries are
// treated as weekday — the shift type already encodes the rate), with a
// weekend-rate fallback when the weekday path is missing. Zero-hour entries
// are deletion requests and skip rate resolution entirely.
func (s *BCCImportService) buildWeeklyBCCEntries(
	ctx context.Context,
	parsed *excelparser.WeeklyBCCImportData,
	year int, month time.Month, loc *time.Location,
	byCCCD map[string]*domain.ProjectEmployee,
	empNames map[uint]string,
	stkNameByCCCD map[string]string,
	blockedEmployeeCCCDs map[string]struct{},
	flatRatesFor func(time.Time) map[string]int,
	projectID uint,
	includeFlexibleEmployees bool,
) (entries []domainservices.BulkCreateTimesheetEntry, flexibleEmployeeIDs map[uint]struct{}, totalRows int, importErrors []domain.ImportError) {
	flexibleEmployeeIDs = make(map[uint]struct{})
	reportedMissingCCCDs := make(map[string]struct{})

	for _, sheet := range parsed.Sheets {
		shiftType := sheet.ShiftType

		// The rate set depends on the payrate active for each entry's date
		// (a payrate can change mid-month), so build it per date and cache it.
		shiftRatesByDate := make(map[string]map[wbccRateKey]int)
		shiftRatesFor := func(d time.Time) map[wbccRateKey]int {
			k := d.Format("2006-01-02")
			if sr, ok := shiftRatesByDate[k]; ok {
				return sr
			}
			sr := buildShiftRatesForShift(flatRatesFor(d), shiftType)
			shiftRatesByDate[k] = sr
			return sr
		}

		for _, emp := range sheet.Employees {
			totalRows++

			if isWeeklyBCCEmployeeBlocked(blockedEmployeeCCCDs, emp.EmployeeCode) {
				continue
			}
			assignment := byCCCD[emp.EmployeeCode]
			if assignment == nil {
				if missingErr := weeklyBCCMissingAssignmentError(
					emp.EmployeeCode,
					emp.FullName,
					blockedEmployeeCCCDs,
					reportedMissingCCCDs,
				); missingErr != nil {
					importErrors = append(importErrors, *missingErr)
				}
				continue
			}

			// STK cross-check (same as legacy).
			if mismatch := weeklyCrossCheckSTKName(emp.EmployeeCode, emp.FullName, stkNameByCCCD, " — có thể sai CCCD"); mismatch != nil {
				importErrors = append(importErrors, *mismatch)
				continue
			}

			if assignment.PaymentSchedule == string(domain.PaymentScheduleFlexible) {
				if !includeFlexibleEmployees {
					importErrors = append(importErrors, domain.ImportError{
						Employee: emp.FullName,
						Reason:   "nhân viên lương linh hoạt không áp dụng BCC import",
					})
					continue
				}
				flexibleEmployeeIDs[assignment.EmployeeID] = struct{}{}
			}

			empPosition := canonicalBCCRateKeySegment(assignment.Position)

			for _, entry := range emp.Entries {
				// Excel dates carry the template's month/year — we only
				// use the day number and combine with user-selected forMonth.
				dayNum := entry.Date.Day()
				realDate := time.Date(year, month, dayNum, 0, 0, 0, 0, loc)
				if realDate.Month() != month {
					continue // e.g. day 31 in a 30-day month
				}

				dateStr := realDate.Format("2006-01-02")

				// WeeklyBCC: all entries are treated as "ngày thường" regardless
				// of day-of-week — the shift type already encodes the rate.
				dayType := "ngày thường"

				// Zero-hour entries are deletion requests: they carry no rate
				// burden, so skip the payrate resolution entirely and let the
				// replacement plan delete the matching pending row.
				if entry.Hours <= 0 {
					entries = append(entries, domainservices.BulkCreateTimesheetEntry{
						ProjectID:   projectID,
						EmployeeID:  assignment.EmployeeID,
						Date:        dateStr,
						HoursWorked: entry.Hours,
						HourType:    shiftType,
						DayType:     &dayType,
					})
					continue
				}

				// Resolve the rate set for THIS date's payrate, then look up
				// position + dayType + shiftType.
				shiftRates := shiftRatesFor(realDate)
				key := wbccRateKey{position: empPosition, dayType: canonicalBCCRateKeySegment(dayType)}
				_, found := shiftRates[key]
				if !found {
					// Try weekend rate as fallback.
					fallbackKey := weeklyBCCWeekendFallbackKey(empPosition)
					if _, f := shiftRates[fallbackKey]; f {
						slog.Warn("BCCImport(WBCC): using weekend rate fallback for weekday entry",
							"employee_id", assignment.EmployeeID, "shift_type", shiftType,
							"position", empPosition, "date", dateStr)
						found = true
					}
				}

				if !found {
					name := empNames[assignment.EmployeeID]
					if name == "" {
						name = fmt.Sprintf("ID %d", assignment.EmployeeID)
					}
					importErrors = append(importErrors, domain.ImportError{
						Employee: name,
						Reason:   fmt.Sprintf("không tìm thấy mức lương cho ca %s, vị trí %s, ngày %s", shiftType, assignment.Position, dateStr),
					})
					continue
				}

				// Use shiftType as hourType — it maps directly to the payrate config key.
				hourType := shiftType

				entries = append(entries, domainservices.BulkCreateTimesheetEntry{
					ProjectID:   projectID,
					EmployeeID:  assignment.EmployeeID,
					Date:        dateStr,
					HoursWorked: entry.Hours,
					HourType:    hourType,
					DayType:     &dayType,
				})
			}
		}
	}

	return entries, flexibleEmployeeIDs, totalRows, importErrors
}
