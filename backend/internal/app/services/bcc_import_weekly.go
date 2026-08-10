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
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/utils"

	"github.com/xuri/excelize/v2"
)

// wbccRateKey groups rate lookups by (position, dayType) for weekly BCC imports.
type wbccRateKey struct {
	position string
	dayType  string
}

// processWeeklyBCCUpload handles the BCC-<shiftType> sheet format (e.g. BCC-HC, BCC-OT150).
// Each sheet represents one shift type for all employees doing that shift.
// Separate timesheet entries are created per (employee, date, shiftType).
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
	fail := func(status, reason string) (*BCCImportResult, error) {
		errs := []domain.ImportError{{Reason: reason}}
		detail := marshalErrors(errs)
		now := clock.Now()
		stats := BCCImportStats{
			ProjectID:    projectID,
			OriginalName: filename,
			ForMonth:     effectiveMonth,
			Status:       status,
			ErrorCount:   1,
			ErrorDetail:  detail,
			ProcessedAt:  &now,
		}
		if metaErr := s.updateAssetMetadata(ctx, createdAsset.ID, &stats); metaErr != nil {
			slog.Error("BCCImport(WBCC): metadata update failed in fail path", "asset_id", createdAsset.ID, "error", metaErr)
		}
		return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt),
			fmt.Errorf("import failed: %s", FirstErrorReason(detail))
	}

	// 1. Parse all BCC-* sheets.
	parsed, err := excelparser.ParseWeeklyBCCFile(xf, formatResult.WeeklyBCCSheets)
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi phân tích file BCC tuần: %v", err))
	}

	// 2-3. Shared setup: month parsing, lock, payrate lookup.
	ictx, releaseLock, err := s.prepareImportContext(ctx, projectID, effectiveMonth)
	if err != nil {
		return fail("failed", err.Error())
	}
	defer releaseLock()
	year, month, monthStart, flatRates := ictx.year, ictx.month, ictx.monthStart, ictx.flatRates
	loc := monthStart.Location()

	// 4. Resolve payrates for the ACTUAL timesheet dates in the file, not monthStart.
	// A payrate can change mid-month; a weekly BCC's visible days (e.g. June 8-14)
	// may fall under a different payrate than the 1st of the month. Resolving at
	// monthStart picks the wrong config (and the wrong rates), so we look the
	// payrate up per entry date and cache it.
	flatRatesByDate := make(map[string]map[string]int)
	flatRatesFor := func(d time.Time) map[string]int {
		key := d.Format("2006-01-02")
		if fr, ok := flatRatesByDate[key]; ok {
			return fr
		}
		var fr map[string]int
		if pr, err := s.payrateRepo.GetActiveByProjectAndDate(ctx, projectID, d); err == nil {
			if f, ferr := pr.Payrate.Flatten(); ferr == nil {
				fr = f
			}
		}
		flatRatesByDate[key] = fr // cache (nil if no payrate covers this date)
		return fr
	}

	// 4b. Validate that every sheet's shift type exists in the payrate config
	//     active for the dates that sheet actually has entries on.
	for _, sheet := range parsed.Sheets {
		checkedDates := make(map[string]bool)
		for _, emp := range sheet.Employees {
			for _, entry := range emp.Entries {
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
					return fail("failed", fmt.Sprintf("không tìm thấy cấu hình lương cho ngày %s", dateKey))
				}
				if !shiftInConfig(fr, sheet.ShiftType) {
					return fail("failed",
						fmt.Sprintf("ca làm \"%s\" không có trong cấu hình lương cho ngày %s. Các ca làm khả dụng: %s",
							sheet.ShiftType, dateKey, strings.Join(getShiftTypes(fr), ", ")))
				}
			}
		}
	}

	// 5. STK auto-creation (same pattern as multi-position).
	var importErrors []domain.ImportError
	blockedEmployeeCCCDs := make(map[string]struct{})

	stkRows, stkErr := excelparser.ParseSTKSheet(xf)
	if stkErr != nil {
		slog.Warn("BCCImport(WBCC): failed to parse STK sheet", "error", stkErr)
	}

	// Build STK CCCD→Name lookup for cross-validation.
	stkNameByCCCD := make(map[string]string, len(stkRows))
	for _, row := range stkRows {
		if row.CCCD != "" && row.FullName != "" {
			stkNameByCCCD[row.CCCD] = row.FullName
		}
	}

	// Collect unique CCCDs across all sheets for position deduction.
	// Default position: use the first position from payrate config (sorted for determinism).
	availablePositions := getPositions(flatRates)
	sort.Strings(availablePositions)
	defaultPosition := ""
	if len(availablePositions) > 0 {
		defaultPosition = availablePositions[0]
	}

	if len(stkRows) > 0 {
		slog.Info("BCCImport(WBCC): found STK sheet, processing employee auto-creation",
			"count", len(stkRows))

		monthStartDate := time.Date(year, month, 1, 0, 0, 0, 0, loc)

		stkErr := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
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
					assignment := &domain.ProjectEmployee{
						ProjectID:       projectID,
						EmployeeID:      emp.ID,
						EmployeeName:    emp.Fullname,
						EmployeeCCCD:    emp.CCCD,
						Position:        defaultPosition,
						StartDate:       monthStartDate,
						PaymentSchedule: string(domain.PaymentScheduleWeekly),
						CreatedBy:       uploaderID,
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
			return fail("failed", fmt.Sprintf("lỗi tự động tạo nhân viên từ STK: %v", stkErr))
		}
	}

	// 6. Load all active project assignments.
	assignments, err := s.employeeService.GetActiveAssignments(ctx, projectID)
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi tải danh sách nhân viên: %v", err))
	}
	byCCCD := make(map[string]*domain.ProjectEmployee, len(assignments))
	empNames := make(map[uint]string, len(assignments))
	for _, a := range assignments {
		byCCCD[a.EmployeeCCCD] = a
		empNames[a.EmployeeID] = a.EmployeeName
	}

	// 6.5. Auto-create employees that appear in BCC sheets but don't exist in the project yet.
	// The STK sheet may cover a different set of employees — BCC employees must also be created.
	{
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

		if len(missingCCCDs) > 0 {
			slog.Info("BCCImport(WBCC): auto-creating missing employees from BCC sheets",
				"count", len(missingCCCDs))

			monthStartDate := time.Date(year, month, 1, 0, 0, 0, 0, loc)

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

					// Ensure employee is assigned to the project.
					assignment := &domain.ProjectEmployee{
						ProjectID:       projectID,
						EmployeeID:      emp.ID,
						EmployeeName:    emp.Fullname,
						EmployeeCCCD:    emp.CCCD,
						Position:        defaultPosition,
						StartDate:       monthStartDate,
						PaymentSchedule: string(domain.PaymentScheduleWeekly),
						CreatedBy:       uploaderID,
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
				}
				return nil
			})
			if autoErr != nil {
				slog.Error("BCCImport(WBCC): BCC employee auto-creation transaction failed", "error", autoErr)
			}
		}
	}

	// 7. Build timesheet entries for each shift-type sheet.
	//
	// Rate resolution for WeeklyBCC:
	//   - The shift type (e.g. "HC") from the sheet name is a leaf key in the payrate JSON.
	//   - For each employee+date entry, we build the path:
	//     "{employee_position}.{dayType}.{shiftType}"
	//   - dayType is determined by the calendar date (weekday vs weekend only; public holidays are NOT detected).
	//   - If the exact path is not found, we fall back to any path ending with the shift type.

	var entries []domainservices.BulkCreateTimesheetEntry
	flexibleEmployeeIDs := make(map[uint]struct{})
	totalRows := 0
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
			if stkName, ok := stkNameByCCCD[emp.EmployeeCode]; ok && stkName != "" {
				bccNorm := bccNormName(emp.FullName)
				stkNorm := bccNormName(stkName)
				if bccNorm != stkNorm && bccNormNameLoose(emp.FullName) != bccNormNameLoose(stkName) {
					importErrors = append(importErrors, domain.ImportError{
						Employee: emp.FullName,
						Reason:   fmt.Sprintf("tên BCC (%s) và tên STK (%s) khác nhau cho cùng CCCD %s — có thể sai CCCD", emp.FullName, stkName, emp.EmployeeCode),
					})
					continue
				}
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

			empPosition := strings.ToLower(assignment.Position)

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

				// Resolve the rate set for THIS date's payrate, then look up
				// position + dayType + shiftType.
				shiftRates := shiftRatesFor(realDate)
				key := wbccRateKey{position: empPosition, dayType: strings.ToLower(dayType)}
				_, found := shiftRates[key]
				if !found {
					// Try weekend rate as fallback.
					fallbackKey := wbccRateKey{position: empPosition, dayType: "ngày nghỉ"}
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

	// 8. "Latest wins" overwrite (same logic as other formats).
	var staleIDs []uint
	flexibleSkippedCount := 0
	if len(entries) > 0 {
		monthEnd := time.Date(year, month+1, 0, 23, 59, 59, 0, loc)
		existingTS, terr := s.timesheetReader.GetByProject(ctx, projectID, monthStart, monthEnd)
		if terr != nil {
			return fail("failed", fmt.Sprintf("lỗi tải bảng chấm công hiện có: %v", terr))
		}

		// Dedup key includes HourType so that importing BCC-OT150 does not
		// hard-delete existing BCC-HC entries for the same employee+date.
		type dk struct {
			empID    uint
			date     string
			hourType string
		}

		importDates := make(map[dk]bool, len(entries))
		for _, e := range entries {
			importDates[dk{e.EmployeeID, e.Date, strings.ToLower(e.HourType)}] = true
		}

		blocked := make(map[dk]string)
		existingFlexible := make(map[dk]bool)
		for _, ts := range existingTS {
			// Extract hourType (last segment) from the timesheet's PayType path.
			tsHourType := ""
			if parts := strings.Split(ts.PayType, "."); len(parts) >= 1 {
				tsHourType = strings.ToLower(parts[len(parts)-1])
			}
			k := dk{ts.EmployeeID, ts.Date.Format("2006-01-02"), tsHourType}
			if !importDates[k] {
				continue
			}
			if _, isFlexible := flexibleEmployeeIDs[ts.EmployeeID]; isFlexible {
				existingFlexible[k] = true
				continue
			}

			isPaid := ts.PaymentStatus == domain.PaymentStatusPaid ||
				ts.PaymentStatus == domain.PaymentStatusFailed ||
				ts.PaymentStatus == domain.PaymentStatusCancelled

			switch {
			case isPaid:
				blocked[k] = "đã thanh toán"
			case ts.Status == domain.TimesheetStatusApproved:
				blocked[k] = "đã được phê duyệt"
			default:
				staleIDs = append(staleIDs, ts.ID)
			}
		}

		if len(blocked) > 0 {
			warned := make(map[dk]bool, len(blocked))
			var filtered []domainservices.BulkCreateTimesheetEntry
			for _, e := range entries {
				k := dk{e.EmployeeID, e.Date, strings.ToLower(e.HourType)}
				if reason, isBlocked := blocked[k]; isBlocked {
					if !warned[k] {
						warned[k] = true
						name := empNames[e.EmployeeID]
						if name == "" {
							name = fmt.Sprintf("ID %d", e.EmployeeID)
						}
						importErrors = append(importErrors, domain.ImportError{
							Employee: name,
							Reason:   fmt.Sprintf("ngày %s (%s): %s, không ghi đè", e.Date, e.HourType, reason),
						})
					}
					continue
				}
				filtered = append(filtered, e)
			}
			entries = filtered
		}
		if len(existingFlexible) > 0 {
			filtered := entries[:0]
			for _, entry := range entries {
				key := dk{entry.EmployeeID, entry.Date, strings.ToLower(entry.HourType)}
				if existingFlexible[key] {
					flexibleSkippedCount++
					continue
				}
				filtered = append(filtered, entry)
			}
			entries = filtered
		}

	}

	// 9. Bulk create or return failure.
	if len(entries) == 0 {
		reason := "không có dữ liệu hợp lệ để tạo bảng chấm công"
		if len(importErrors) > 0 {
			detail := marshalErrors(importErrors)
			now := clock.Now()
			stats := BCCImportStats{
				ProjectID:    projectID,
				OriginalName: filename,
				ForMonth:     effectiveMonth,
				Status:       "failed",
				TotalRows:    totalRows,
				ErrorCount:   len(importErrors),
				ErrorDetail:  detail,
				ProcessedAt:  &now,
			}
			if metaErr := s.updateAssetMetadata(ctx, createdAsset.ID, &stats); metaErr != nil {
				slog.Error("BCCImport(WBCC): metadata update failed in empty-entries path",
					"asset_id", createdAsset.ID, "error", metaErr)
			}
			return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt),
				fmt.Errorf("import failed: %s", FirstErrorReason(detail))
		}
		return fail("failed", reason)
	}

	result, err := s.applyTimesheetReplacement(ctx, staleIDs, entries, uploaderID, uploaderRole)
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi tạo bảng chấm công: %v", err))
	}

	// 10. Finalize.
	createdCount := len(result.CreatedTimesheets)
	skippedCount := len(result.DeletedTimesheets) + flexibleSkippedCount
	for _, f := range result.FailedEntries {
		importErrors = append(importErrors, domain.ImportError{
			Employee: fmt.Sprintf("employee_id=%d date=%s", f.Request.EmployeeID, f.Request.Date),
			Reason:   f.Error,
		})
	}
	errorCount := len(importErrors)

	now := clock.Now()
	finalStatus := "completed"
	if createdCount == 0 && errorCount > 0 {
		finalStatus = "failed"
	}

	errDetail := marshalErrors(importErrors)
	stats := BCCImportStats{
		ProjectID:    projectID,
		OriginalName: filename,
		ForMonth:     effectiveMonth,
		Status:       finalStatus,
		TotalRows:    totalRows,
		CreatedCount: createdCount,
		SkippedCount: skippedCount,
		ErrorCount:   errorCount,
		ErrorDetail:  errDetail,
		ProcessedAt:  &now,
	}
	if metaErr := s.updateAssetMetadata(ctx, createdAsset.ID, &stats); metaErr != nil {
		slog.Error("BCCImport(WBCC): metadata update failed", "asset_id", createdAsset.ID, "error", metaErr)
	}

	return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt), nil
}

func weeklyBCCMissingAssignmentError(
	cccd string,
	fullName string,
	blockedCCCDs map[string]struct{},
	reportedCCCDs map[string]struct{},
) *domain.ImportError {
	if isWeeklyBCCEmployeeBlocked(blockedCCCDs, cccd) {
		return nil
	}
	if _, reported := reportedCCCDs[cccd]; reported {
		return nil
	}
	reportedCCCDs[cccd] = struct{}{}
	return &domain.ImportError{
		Employee: fullName,
		Reason:   fmt.Sprintf("không tìm thấy nhân viên với CCCD \"%s\" trong dự án", cccd),
	}
}

func isWeeklyBCCEmployeeBlocked(blockedCCCDs map[string]struct{}, cccd string) bool {
	_, blocked := blockedCCCDs[cccd]
	return blocked
}

// shiftInConfig reports whether shiftType appears as a leaf key in the flattened
// payrate (case-insensitive), regardless of its rate value. Used to validate that
// a BCC sheet's shift exists in the payrate active for a given date.
func shiftInConfig(flatRates map[string]int, shiftType string) bool {
	if len(flatRates) == 0 {
		return false
	}
	target := strings.ToLower(shiftType)
	for path := range flatRates {
		parts := strings.Split(path, ".")
		if len(parts) >= 3 && strings.ToLower(parts[len(parts)-1]) == target {
			return true
		}
	}
	return false
}

// buildShiftRatesForShift collects every non-zero rate path whose leaf matches
// shiftType (case-insensitive), grouped by (position, dayType) for quick lookup.
func buildShiftRatesForShift(flatRates map[string]int, shiftType string) map[wbccRateKey]int {
	shiftRates := make(map[wbccRateKey]int)
	target := strings.ToLower(shiftType)
	for path, rate := range flatRates {
		if rate == 0 {
			continue
		}
		parts := strings.Split(path, ".")
		if len(parts) < 3 {
			continue
		}
		if strings.ToLower(parts[len(parts)-1]) != target {
			continue
		}
		position := parts[0]
		dayType := strings.Join(parts[1:len(parts)-1], ".")
		key := wbccRateKey{position: strings.ToLower(position), dayType: strings.ToLower(dayType)}
		shiftRates[key] = rate
	}
	return shiftRates
}

// getShiftTypes extracts unique leaf-level shift type names from flattened payrate paths.
// For paths like "pho thong.ngay thuong.HC" → returns ["HC"].
func getShiftTypes(flatRates map[string]int) []string {
	seen := make(map[string]bool)
	var shiftTypes []string
	for path := range flatRates {
		parts := strings.Split(path, ".")
		if len(parts) >= 3 {
			leaf := parts[len(parts)-1]
			if !seen[leaf] {
				seen[leaf] = true
				shiftTypes = append(shiftTypes, leaf)
			}
		}
	}
	return shiftTypes
}

// determineDayType returns the Vietnamese day type string based on the calendar date.
// NOTE: Only distinguishes weekday vs weekend; public holidays are NOT detected.
func determineDayType(date time.Time) string {
	if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
		return "ngày nghỉ"
	}
	return "ngày thường"
}

// findSTKRow finds an STK row by CCCD for bank info lookup.
func findSTKRow(stkRows []excelparser.STKRow, cccd string) *excelparser.STKRow {
	for i := range stkRows {
		if stkRows[i].CCCD == cccd {
			return &stkRows[i]
		}
	}
	return nil
}

// processWeeklyPaymentUpload handles the weekly payment format (e.g. 520, 700, 750 sheets).
// Each sheet represents one salary tier, with row 10 containing per-cell shift codes.
// Combined rate key = sheet prefix + shift code (e.g., 520HC, 700TCN).
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
	fail := func(status, reason string) (*BCCImportResult, error) {
		errs := []domain.ImportError{{Reason: reason}}
		detail := marshalErrors(errs)
		now := clock.Now()
		stats := BCCImportStats{
			ProjectID:     projectID,
			OriginalName:  filename,
			ForMonth:      effectiveMonth,
			Status:        status,
			ErrorCount:    1,
			ErrorDetail:   detail,
			ProcessedAt:   &now,
		}
		if metaErr := s.updateAssetMetadata(ctx, createdAsset.ID, &stats); metaErr != nil {
			slog.Error("BCCImport(WPayment): metadata update failed in fail path", "asset_id", createdAsset.ID, "error", metaErr)
		}
		return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt),
			fmt.Errorf("import failed: %s", FirstErrorReason(detail))
	}

	// 1. Parse all weekly payment sheets.
	parsed, err := excelparser.ParseWeeklyPaymentFile(xf, formatResult.WeeklyPaymentSheets, effectiveMonth)
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi phân tích file weekly payment: %v", err))
	}

	// 2-3. Shared setup: month parsing, lock, payrate lookup.
	ictx, releaseLock, err := s.prepareImportContext(ctx, projectID, effectiveMonth)
	if err != nil {
		return fail("failed", err.Error())
	}
	defer releaseLock()
	year, month, monthStart, flatRates := ictx.year, ictx.month, ictx.monthStart, ictx.flatRates
	loc := monthStart.Location()

	// 4. Resolve payrates for the ACTUAL timesheet dates in the file.
	flatRatesByDate := make(map[string]map[string]int)
	flatRatesFor := func(d time.Time) map[string]int {
		key := d.Format("2006-01-02")
		if fr, ok := flatRatesByDate[key]; ok {
			return fr
		}
		var fr map[string]int
		if pr, err := s.payrateRepo.GetActiveByProjectAndDate(ctx, projectID, d); err == nil {
			if f, ferr := pr.Payrate.Flatten(); ferr == nil {
				fr = f
			}
		}
		flatRatesByDate[key] = fr
		return fr
	}

	// 5. STK auto-creation (same pattern as multi-position and weekly BCC).
	var importErrors []domain.ImportError
	blockedEmployeeCCCDs := make(map[string]struct{})

	stkRows, stkErr := excelparser.ParseSTKSheet(xf)
	if stkErr != nil {
		slog.Warn("BCCImport(WPayment): failed to parse STK sheet", "error", stkErr)
	}

	stkNameByCCCD := make(map[string]string, len(stkRows))
	for _, row := range stkRows {
		if row.CCCD != "" && row.FullName != "" {
			stkNameByCCCD[row.CCCD] = row.FullName
		}
	}

	// Collect unique CCCDs and determine default position.
	availablePositions := getPositions(flatRates)
	sort.Strings(availablePositions)
	defaultPosition := ""
	if len(availablePositions) > 0 {
		defaultPosition = availablePositions[0]
	}

	if len(stkRows) > 0 {
		slog.Info("BCCImport(WPayment): found STK sheet, processing employee auto-creation",
			"count", len(stkRows))

		monthStartDate := time.Date(year, month, 1, 0, 0, 0, 0, loc)

		stkErr := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
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
				if existingAssignment == nil {
					assignment := &domain.ProjectEmployee{
						ProjectID:       projectID,
						EmployeeID:      emp.ID,
						EmployeeName:    emp.Fullname,
						EmployeeCCCD:    emp.CCCD,
						Position:        defaultPosition,
						StartDate:       monthStartDate,
						PaymentSchedule: string(domain.PaymentScheduleWeekly),
						CreatedBy:       uploaderID,
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
			return fail("failed", fmt.Sprintf("lỗi tự động tạo nhân viên từ STK: %v", stkErr))
		}
	}

	// 6. Load all active project assignments.
	assignments, err := s.employeeService.GetActiveAssignments(ctx, projectID)
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi tải danh sách nhân viên: %v", err))
	}
	byCCCD := make(map[string]*domain.ProjectEmployee, len(assignments))
	empNames := make(map[uint]string, len(assignments))
	for _, a := range assignments {
		byCCCD[a.EmployeeCCCD] = a
		empNames[a.EmployeeID] = a.EmployeeName
	}

	// 6.5. Auto-create employees from BCC sheets (same pattern as weekly BCC).
	{
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
				}{
					cccd:     emp.EmployeeCode,
					fullName: emp.FullName,
				})
			}
		}

		if len(missingCCCDs) > 0 {
			slog.Info("BCCImport(WPayment): auto-creating missing employees from BCC sheets",
				"count", len(missingCCCDs))

			monthStartDate := time.Date(year, month, 1, 0, 0, 0, 0, loc)

			autoErr := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
				for _, m := range missingCCCDs {
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
						Position:        defaultPosition,
						StartDate:       monthStartDate,
						PaymentSchedule: string(domain.PaymentScheduleWeekly),
						CreatedBy:       uploaderID,
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
		}
	}

	// 7. Build timesheet entries for each salary-tier sheet.
	var entries []domainservices.BulkCreateTimesheetEntry
	flexibleEmployeeIDs := make(map[uint]struct{})
	totalRows := 0
	reportedMissingCCCDs := make(map[string]struct{})

	for _, sheet := range parsed.Sheets {
		prefix := sheet.Prefix

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
			if stkName, ok := stkNameByCCCD[emp.EmployeeCode]; ok && stkName != "" {
				bccNorm := bccNormName(emp.FullName)
				stkNorm := bccNormName(stkName)
				if bccNorm != stkNorm && bccNormNameLoose(emp.FullName) != bccNormNameLoose(stkName) {
					importErrors = append(importErrors, domain.ImportError{
						Employee: emp.FullName,
						Reason:   fmt.Sprintf("tên BCC (%s) và tên STK (%s) khác nhau cho cùng CCCD %s",
							emp.FullName, stkName, emp.EmployeeCode),
					})
					continue
				}
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

			empPosition := strings.ToLower(assignment.Position)

			for _, entry := range emp.Entries {
				if entry.Day < 1 || entry.Day > 31 {
					continue
				}
				realDate := time.Date(year, month, entry.Day, 0, 0, 0, 0, loc)
				if realDate.Month() != month {
					continue
				}

				dateStr := realDate.Format("2006-01-02")

				// Build combined rate key: prefix + shift code (e.g., 520HC)
				combinedShiftKey := excelparser.RateKeyFor(prefix, entry.ShiftKey)

				// Determine day type from shift code.
				dayType := excelparser.DayTypeFor(entry.ShiftKey)

				// Resolve the rate set for THIS date's payrate using combined shift key.
				shiftRates := buildShiftRatesForShift(flatRatesFor(realDate), combinedShiftKey)
				key := wbccRateKey{position: empPosition, dayType: strings.ToLower(dayType)}
				if _, found := shiftRates[key]; !found {
					name := empNames[assignment.EmployeeID]
					if name == "" {
						name = fmt.Sprintf("ID %d", assignment.EmployeeID)
					}
					importErrors = append(importErrors, domain.ImportError{
						Employee: name,
						Reason:   fmt.Sprintf("không tìm thấy mức lương cho ca %s, vị trí %s, ngày %s",
							combinedShiftKey, assignment.Position, dateStr),
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

	// 8. "Latest wins" overwrite (same logic as other formats).
	var staleIDs []uint
	flexibleSkippedCount := 0
	if len(entries) > 0 {
		monthEnd := time.Date(year, month+1, 0, 23, 59, 59, 0, loc)
		existingTS, terr := s.timesheetReader.GetByProject(ctx, projectID, monthStart, monthEnd)
		if terr != nil {
			return fail("failed", fmt.Sprintf("lỗi tải bảng chấm công hiện có: %v", terr))
		}

		type dk struct {
			empID    uint
			date     string
			hourType string
		}

		importDates := make(map[dk]bool, len(entries))
		for _, e := range entries {
			importDates[dk{e.EmployeeID, e.Date, strings.ToLower(e.HourType)}] = true
		}

		blocked := make(map[dk]string)
		existingFlexible := make(map[dk]bool)
		for _, ts := range existingTS {
			tsHourType := ""
			if parts := strings.Split(ts.PayType, "."); len(parts) >= 1 {
				tsHourType = strings.ToLower(parts[len(parts)-1])
			}
			k := dk{ts.EmployeeID, ts.Date.Format("2006-01-02"), tsHourType}
			if !importDates[k] {
				continue
			}
			if _, isFlexible := flexibleEmployeeIDs[ts.EmployeeID]; isFlexible {
				existingFlexible[k] = true
				continue
			}

			isPaid := ts.PaymentStatus == domain.PaymentStatusPaid ||
				ts.PaymentStatus == domain.PaymentStatusFailed ||
				ts.PaymentStatus == domain.PaymentStatusCancelled

			switch {
			case isPaid:
				blocked[k] = "đã thanh toán"
			case ts.Status == domain.TimesheetStatusApproved:
				blocked[k] = "đã được phê duyệt"
			default:
				staleIDs = append(staleIDs, ts.ID)
			}
		}

		if len(blocked) > 0 {
			warned := make(map[dk]bool, len(blocked))
			var filtered []domainservices.BulkCreateTimesheetEntry
			for _, e := range entries {
				k := dk{e.EmployeeID, e.Date, strings.ToLower(e.HourType)}
				if reason, isBlocked := blocked[k]; isBlocked {
					if !warned[k] {
						warned[k] = true
						name := empNames[e.EmployeeID]
						if name == "" {
							name = fmt.Sprintf("ID %d", e.EmployeeID)
						}
						importErrors = append(importErrors, domain.ImportError{
							Employee: name,
							Reason:   fmt.Sprintf("ngày %s (%s): %s, không ghi đè", e.Date, e.HourType, reason),
						})
					}
					continue
				}
				filtered = append(filtered, e)
			}
			entries = filtered
		}
		if len(existingFlexible) > 0 {
			filtered := entries[:0]
			for _, entry := range entries {
				key := dk{entry.EmployeeID, entry.Date, strings.ToLower(entry.HourType)}
				if existingFlexible[key] {
					flexibleSkippedCount++
					continue
				}
				filtered = append(filtered, entry)
			}
			entries = filtered
		}
	}

	// 9. Bulk create or return failure.
	if len(entries) == 0 {
		reason := "không có dữ liệu hợp lệ để tạo bảng chấm công"
		if len(importErrors) > 0 {
			detail := marshalErrors(importErrors)
			now := clock.Now()
			stats := BCCImportStats{
				ProjectID:     projectID,
				OriginalName:  filename,
				ForMonth:      effectiveMonth,
				Status:        "failed",
				TotalRows:     totalRows,
				ErrorCount:    len(importErrors),
				ErrorDetail:   detail,
				ProcessedAt:   &now,
			}
			if metaErr := s.updateAssetMetadata(ctx, createdAsset.ID, &stats); metaErr != nil {
				slog.Error("BCCImport(WPayment): metadata update failed in empty-entries path",
					"asset_id", createdAsset.ID, "error", metaErr)
			}
			return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt),
				fmt.Errorf("import failed: %s", FirstErrorReason(detail))
		}
		return fail("failed", reason)
	}

	result, err := s.applyTimesheetReplacement(ctx, staleIDs, entries, uploaderID, uploaderRole)
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi tạo bảng chấm công: %v", err))
	}

	// 10. Finalize.
	createdCount := len(result.CreatedTimesheets)
	skippedCount := len(result.DeletedTimesheets) + flexibleSkippedCount
	for _, f := range result.FailedEntries {
		importErrors = append(importErrors, domain.ImportError{
			Employee: fmt.Sprintf("employee_id=%d date=%s", f.Request.EmployeeID, f.Request.Date),
			Reason:   f.Error,
		})
	}
	errorCount := len(importErrors)

	now := clock.Now()
	finalStatus := "completed"
	if createdCount == 0 && errorCount > 0 {
		finalStatus = "failed"
	}

	errDetail := marshalErrors(importErrors)
	stats := BCCImportStats{
		ProjectID:     projectID,
		OriginalName:  filename,
		ForMonth:      effectiveMonth,
		Status:        finalStatus,
		TotalRows:     totalRows,
		CreatedCount:  createdCount,
		SkippedCount:  skippedCount,
		ErrorCount:    errorCount,
		ErrorDetail:   errDetail,
		ProcessedAt:   &now,
	}
	if metaErr := s.updateAssetMetadata(ctx, createdAsset.ID, &stats); metaErr != nil {
		slog.Error("BCCImport(WPayment): metadata update failed", "asset_id", createdAsset.ID, "error", metaErr)
	}

	return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt), nil
}
