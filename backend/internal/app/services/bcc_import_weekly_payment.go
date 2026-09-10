package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	excelparser "api-server/internal/app/services/excel"
	"api-server/internal/domain"
	domainservices "api-server/internal/domain/services"

	"github.com/xuri/excelize/v2"
)

// processWeeklyPaymentUpload handles the weekly payment format.
// Each sheet name is the configured payrate position (for example, "Lương 520"),
// and each row 10 column header is its configured shift code (for example, "HC" or "TCN").
// Format-specific steps live in the named methods below; the shared pipeline
// pieces are in bcc_import_pipeline.go and bcc_import_weekly_shared.go.
func (s *BCCImportService) processWeeklyPaymentUpload(
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

	// 1. Parse all weekly payment sheets.
	parsed, err := excelparser.ParseWeeklyPaymentFile(xf, formatResult.WeeklyPaymentSheets, effectiveMonth)
	if err != nil {
		return fail(safeWeeklyPaymentParseError(err))
	}
	positionByCCCD, blockedEmployeeCCCDs, positionErrors := buildWeeklyPaymentPositions(parsed.Sheets)

	// 2-3. Shared setup: month parsing, lock, payrate lookup. The earliest
	// worked day lets the lookup accept a payrate that starts mid-month but
	// is already active on the file's first worked day.
	earliestDay := 0
	for _, sh := range parsed.Sheets {
		for _, emp := range sh.Employees {
			for _, e := range emp.Entries {
				if e.Day > 0 && (earliestDay == 0 || e.Day < earliestDay) {
					earliestDay = e.Day
				}
			}
		}
	}
	ictx, releaseLock, err := s.prepareImportContext(ctx, projectID, effectiveMonth, earliestDay)
	if err != nil {
		return fail(err.Error())
	}
	defer releaseLock()
	year, month, monthStart := ictx.year, ictx.month, ictx.monthStart
	loc := monthStart.Location()

	// 4. Resolve payrates for the ACTUAL timesheet dates in the file.
	rates := newWeeklyRateResolver(s, ctx, projectID)

	// 5. STK auto-creation.
	importErrors := append([]domain.ImportError(nil), positionErrors...)

	stkNameByCCCD, stkRows, stkErrors, failReason := s.autoCreateWeeklyPaymentSTK(
		ctx, xf, positionByCCCD, blockedEmployeeCCCDs, projectID, uploaderID, year, month, loc)
	if failReason != "" {
		return fail(failReason)
	}
	importErrors = append(importErrors, stkErrors...)

	// 6. Load all active project assignments.
	assignments, err := s.employeeService.GetActiveAssignments(ctx, projectID)
	if err != nil {
		return fail(fmt.Sprintf("lỗi tải danh sách nhân viên: %v", err))
	}
	plannedPositionCorrections := planWeeklyPaymentPositionCorrections(assignments, positionByCCCD, includeFlexibleEmployees)
	byCCCD := make(map[string]*domain.ProjectEmployee, len(assignments))
	empNames := make(map[uint]string, len(assignments))
	for _, a := range assignments {
		byCCCD[a.EmployeeCCCD] = a
		empNames[a.EmployeeID] = a.EmployeeName
	}

	// 6.5. Auto-create employees from BCC sheets (same pattern as weekly BCC).
	autoErrors := s.autoCreateMissingWeeklyPaymentEmployees(
		ctx, parsed, stkRows, byCCCD, empNames, blockedEmployeeCCCDs, positionByCCCD,
		year, month, loc, projectID, uploaderID)
	importErrors = append(importErrors, autoErrors...)

	// 7. Build timesheet entries for each salary-tier sheet.
	entries, flexibleEmployeeIDs, totalRows, entryErrors := s.buildWeeklyPaymentEntries(
		ctx, parsed, year, month, loc, byCCCD, empNames, stkNameByCCCD,
		blockedEmployeeCCCDs, rates.forDate, projectID, includeFlexibleEmployees)
	importErrors = append(importErrors, entryErrors...)

	// 8. Preserve reviewed rows, replace pending rows, and create missing rows.
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

	positionCorrections := selectWeeklyPaymentPositionCorrections(entries, plannedPositionCorrections)
	result, err := s.applyTimesheetReplacementPrepared(
		ctx,
		staleIDs,
		entries,
		uploaderID,
		uploaderRole,
		func(txCtx context.Context) error {
			for _, correction := range positionCorrections {
				if updateErr := s.projectEmployeeSvc.UpdateAssignmentPositionIfCurrent(
					txCtx,
					correction.assignmentID,
					correction.oldPosition,
					correction.newPosition,
					uploaderID,
				); updateErr != nil {
					return fmt.Errorf("cập nhật vị trí cho nhân viên %s: %w", correction.employeeName, updateErr)
				}
			}
			return nil
		},
	)
	if err != nil {
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

// autoCreateWeeklyPaymentSTK upserts employees from the STK sheet with the
// position taken from the salary-tier sheets (positionByCCCD). An STK-only
// hire with no position in any tier sheet is a hard row error. Returns the
// STK name lookup, raw rows, collected errors, and a fatal fail reason when
// the transaction itself failed.
func (s *BCCImportService) autoCreateWeeklyPaymentSTK(
	ctx context.Context,
	xf *excelize.File,
	positionByCCCD map[string]string,
	blockedEmployeeCCCDs map[string]struct{},
	projectID uint,
	uploaderID uint,
	year int, month time.Month, loc *time.Location,
) (stkNameByCCCD map[string]string, stkRows []excelparser.STKRow, importErrors []domain.ImportError, failReason string) {
	stkRows, stkErr := excelparser.ParseSTKSheet(xf)
	if stkErr != nil {
		slog.Warn("BCCImport(WPayment): failed to parse STK sheet", "error", stkErr)
	}
	stkNameByCCCD = weeklySTKNameLookup(stkRows)

	if len(stkRows) == 0 {
		return stkNameByCCCD, stkRows, nil, ""
	}

	slog.Info("BCCImport(WPayment): found STK sheet, processing employee auto-creation",
		"count", len(stkRows))

	monthStartDate := time.Date(year, month, 1, 0, 0, 0, 0, loc)

	stkErr = s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		bankCache := make(map[string]*uint)

		for _, row := range stkRows {
			cccd := row.CCCD
			fullName := row.FullName
			if cccd == "" || fullName == "" {
				continue
			}
			if isWeeklyBCCEmployeeBlocked(blockedEmployeeCCCDs, cccd) {
				continue
			}
			employeePosition := positionByCCCD[cccd]

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
						slog.Error("BCCImport(WPayment): failed to update bank info",
							"employee_id", emp.ID, "error", updateErr)
						importErrors = append(importErrors, domain.ImportError{
							Employee: fullName,
							Reason:   "Không thể cập nhật thông tin ngân hàng",
						})
					}
				}
				if row.Mobile != "" && emp.Mobile == "" {
					if err := s.employeeService.UpdateMobile(txCtx, emp.ID, row.Mobile); err != nil {
						slog.Error("BCCImport(WPayment): failed to fill mobile",
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

			existingAssignment, assignErr := s.employeeService.GetActiveAssignment(txCtx, projectID, emp.ID)
			if assignErr != nil && !domain.IsNotFoundError(assignErr) {
				blockedEmployeeCCCDs[cccd] = struct{}{}
				importErrors = append(importErrors, domain.ImportError{
					Employee: fullName,
					Reason:   fmt.Sprintf("lỗi kiểm tra phân công nhân viên %s: %v", fullName, assignErr),
				})
				continue
			}
			if existingAssignment == nil && employeePosition == "" {
				blockedEmployeeCCCDs[cccd] = struct{}{}
				importErrors = append(importErrors, domain.ImportError{
					Employee: fullName,
					Reason:   "không xác định được vị trí: nhân viên có trong STK nhưng không có trong sheet lương",
				})
				continue
			}
			if existingAssignment == nil {
				assignment := &domain.ProjectEmployee{
					ProjectID:       projectID,
					EmployeeID:      emp.ID,
					EmployeeName:    emp.Fullname,
					EmployeeCCCD:    emp.CCCD,
					Position:        employeePosition,
					StartDate:       monthStartDate,
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
		slog.Error("BCCImport(WPayment): STK auto-creation transaction failed", "error", stkErr)
		return stkNameByCCCD, stkRows, importErrors, fmt.Sprintf("lỗi tự động tạo nhân viên từ STK: %v", stkErr)
	}

	return stkNameByCCCD, stkRows, importErrors, ""
}

// autoCreateMissingWeeklyPaymentEmployees creates tier-sheet employees that
// have no project assignment yet (position taken from their tier sheet; a
// missing position is a row error), filling bank info from STK when
// available. byCCCD and empNames are updated in place.
func (s *BCCImportService) autoCreateMissingWeeklyPaymentEmployees(
	ctx context.Context,
	parsed *excelparser.WeeklyPaymentImportData,
	stkRows []excelparser.STKRow,
	byCCCD map[string]*domain.ProjectEmployee,
	empNames map[uint]string,
	blockedEmployeeCCCDs map[string]struct{},
	positionByCCCD map[string]string,
	year int, month time.Month, loc *time.Location,
	projectID uint,
	uploaderID uint,
) []domain.ImportError {
	var importErrors []domain.ImportError

	var missingCCCDs []struct {
		cccd     string
		fullName string
		position string
	}
	seenMissing := make(map[string]bool)
	for _, sheet := range parsed.Sheets {
		for _, emp := range sheet.Employees {
			if emp.EmployeeCode == "" {
				continue
			}
			if _, exists := byCCCD[emp.EmployeeCode]; exists {
				continue
			}
			if isWeeklyBCCEmployeeBlocked(blockedEmployeeCCCDs, emp.EmployeeCode) {
				continue
			}
			if seenMissing[emp.EmployeeCode] {
				continue
			}
			seenMissing[emp.EmployeeCode] = true
			missingCCCDs = append(missingCCCDs, struct {
				cccd     string
				fullName string
				position string
			}{
				cccd:     emp.EmployeeCode,
				fullName: emp.FullName,
				position: positionByCCCD[emp.EmployeeCode],
			})
		}
	}

	if len(missingCCCDs) == 0 {
		return nil
	}

	slog.Info("BCCImport(WPayment): auto-creating missing employees from BCC sheets",
		"count", len(missingCCCDs))

	monthStartDate := time.Date(year, month, 1, 0, 0, 0, 0, loc)

	autoErr := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		for _, m := range missingCCCDs {
			if m.position == "" {
				blockedEmployeeCCCDs[m.cccd] = struct{}{}
				importErrors = append(importErrors, domain.ImportError{
					Employee: m.fullName,
					Reason:   "không xác định được vị trí từ sheet lương",
				})
				continue
			}
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
				emp = &domain.Employee{
					Fullname:  m.fullName,
					CCCD:      m.cccd,
					CreatedBy: uploaderID,
				}
				if stkRow := findSTKRow(stkRows, m.cccd); stkRow != nil {
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
			} else {
				emp = existingEmp
				if stkRow := findSTKRow(stkRows, m.cccd); stkRow != nil {
					var stkBankID *uint
					if stkRow.BankName != "" {
						stkBankID = s.employeeService.ResolveBankID(txCtx, stkRow.BankName)
					}
					if bankUpdates := buildSTKBankUpdates(emp, *stkRow, stkBankID, m.fullName); bankUpdates != nil {
						if updateErr := s.employeeService.UpdateBankInfo(txCtx, emp.ID, bankUpdates); updateErr != nil {
							slog.Error("BCCImport(WPayment): failed to update bank info",
								"employee_id", emp.ID, "error", updateErr)
						}
					}
					if stkRow.Mobile != "" && emp.Mobile == "" {
						if err := s.employeeService.UpdateMobile(txCtx, emp.ID, stkRow.Mobile); err != nil {
							slog.Error("BCCImport(WPayment): failed to fill mobile",
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

			assignment := &domain.ProjectEmployee{
				ProjectID:       projectID,
				EmployeeID:      emp.ID,
				EmployeeName:    emp.Fullname,
				EmployeeCCCD:    emp.CCCD,
				Position:        m.position,
				StartDate:       monthStartDate,
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
			byCCCD[emp.CCCD] = assignment
			empNames[emp.ID] = emp.Fullname
		}
		return nil
	})
	if autoErr != nil {
		slog.Error("BCCImport(WPayment): BCC employee auto-creation transaction failed", "error", autoErr)
	}

	return importErrors
}

// buildWeeklyPaymentEntries converts parsed salary-tier sheets into bulk
// timesheet entries. This template has one day-type rate set: all entries use
// ngày thường and the column header itself selects the configured shift
// rate. Zero-hour entries are deletion requests and skip rate resolution.
func (s *BCCImportService) buildWeeklyPaymentEntries(
	ctx context.Context,
	parsed *excelparser.WeeklyPaymentImportData,
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
		sheetPosition := canonicalBCCRateKeySegment(sheet.Position)

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

			// STK cross-check.
			if mismatch := weeklyCrossCheckSTKName(emp.EmployeeCode, emp.FullName, stkNameByCCCD, ""); mismatch != nil {
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

			for _, entry := range emp.Entries {
				if entry.Day < 1 || entry.Day > 31 {
					continue
				}
				realDate := time.Date(year, month, entry.Day, 0, 0, 0, 0, loc)
				if realDate.Month() != month {
					continue
				}

				dateStr := realDate.Format("2006-01-02")

				// This template has one day-type rate set: all entries use ngày thường.
				// The column header itself selects the configured shift rate.
				dayType := "ngày thường"

				// Zero-hour entries are deletion requests: no rate burden, so
				// skip payrate resolution and let the replacement plan delete
				// the matching pending row.
				if entry.Hours <= 0 {
					entries = append(entries, domainservices.BulkCreateTimesheetEntry{
						ProjectID:   projectID,
						EmployeeID:  assignment.EmployeeID,
						Date:        dateStr,
						HoursWorked: entry.Hours,
						HourType:    entry.ShiftKey,
						DayType:     &dayType,
					})
					continue
				}

				shiftRates := buildShiftRatesForShift(flatRatesFor(realDate), entry.ShiftKey)
				key := weeklyPaymentRateKey(sheetPosition)
				if _, found := shiftRates[key]; !found {
					name := empNames[assignment.EmployeeID]
					if name == "" {
						name = fmt.Sprintf("ID %d", assignment.EmployeeID)
					}
					importErrors = append(importErrors, domain.ImportError{
						Employee: name,
						Reason: fmt.Sprintf("không tìm thấy mức lương cho ca %s, vị trí %s, ngày %s",
							entry.ShiftKey, sheet.Position, dateStr),
					})
					continue
				}

				// Use bare shift code as hour type (matching WeeklyBCC convention).
				hourType := entry.ShiftKey

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
