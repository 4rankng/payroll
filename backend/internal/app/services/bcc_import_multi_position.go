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
	"golang.org/x/text/unicode/norm"

	"api-server/internal/pkg/utils"
)

// processMultiPositionUpload handles the multi-position BCC format where each
// sheet (beyond STK) represents a position. Orchestrates the shared pipeline
// (context, STK provisioning, entry building, replacement, finalize); the
// format-specific pieces live in the named methods below.
func (s *BCCImportService) processMultiPositionUpload(
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

	// 1. Parse the multi-position sheets.
	parsed, err := excelparser.ParseMultiPositionFile(xf, formatResult.PositionSheets)
	if err != nil {
		return fail(fmt.Sprintf("lỗi phân tích file BCC đa vị trí: %v", err))
	}

	// 2-4. Shared setup: month parsing, lock, payrate lookup. The earliest
	// worked day lets the lookup accept a payrate that starts mid-month but
	// is already active on the file's first worked day.
	earliestDay := 0
	for _, sh := range parsed.Sheets {
		for _, emp := range sh.Employees {
			for _, e := range emp.Entries {
				if e.DayNum > 0 && (earliestDay == 0 || e.DayNum < earliestDay) {
					earliestDay = e.DayNum
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

	// All sheet positions must exist in the payrate config.
	if reason := validateSheetPositions(parsed, flatRates); reason != "" {
		return fail(reason)
	}

	// 5. Build CCCD→position map from all position sheets (for STK auto-creation).
	// Note: PositionEmployeeData.EmployeeCode holds CCCD in the multi-position template.
	importErrors, cccdToPosition := buildCCCDToPositionMap(parsed)

	// 6. STK auto-creation with CCCD→position lookup.
	positionCorrections, stkErrors := s.autoCreateMultiPositionSTK(
		ctx, xf, cccdToPosition, getPositions(flatRates), projectID, uploaderID, monthStart)
	importErrors = append(importErrors, stkErrors...)

	// 6.5. Apply position corrections via ProjectEmployeeService (outside STK transaction)
	// to get cache invalidation, event publishing, and timesheet recalculation with
	// targeted column update (no full-row Save() overwrite risk).
	importErrors = append(importErrors, s.applyPositionCorrections(ctx, positionCorrections, uploaderID)...)

	// 7. Load assignments.
	assignments, err := s.employeeService.GetActiveAssignments(ctx, projectID)
	if err != nil {
		return fail(fmt.Sprintf("lỗi tải danh sách nhân viên: %v", err))
	}
	byCCCD := make(map[string]*domain.ProjectEmployee, len(assignments))
	byCode := make(map[string]*domain.ProjectEmployee, len(assignments))
	byCCCDAndPosition := make(map[string]*domain.ProjectEmployee, len(assignments))
	empNames := make(map[uint]string, len(assignments))
	for _, a := range assignments {
		byCCCD[a.EmployeeCCCD] = a
		byCCCDAndPosition[a.EmployeeCCCD+"|"+strings.ToLower(a.Position)] = a
		if a.EmployeeCode != "" {
			byCode[a.EmployeeCode] = a
		}
		empNames[a.EmployeeID] = a.EmployeeName
	}

	// 8. Build timesheet entries for each position sheet.
	entries, flexibleEmployeeIDs, totalRows, entryErrors := s.buildMultiPositionEntries(
		ctx, parsed, flatRates, year, month, loc,
		byCCCD, byCode, byCCCDAndPosition, projectID, uploaderID, includeFlexibleEmployees)
	importErrors = append(importErrors, entryErrors...)

	// 9. Preserve reviewed rows, replace pending rows, and create missing rows.
	entries, staleIDs, protectedSkippedCount, flexibleSkippedCount, perr := s.planMonthReplacement(
		ctx, projectID, year, month, monthStart, loc, entries, flexibleEmployeeIDs, false)
	if perr != nil {
		return fail(fmt.Sprintf("lỗi tải bảng chấm công hiện có: %v", perr))
	}

	// 10. Bulk create or return failure.
	if len(entries) == 0 {
		return s.finishEmptyBCCImport(ctx, createdAsset, uploaderID, projectID, filename, effectiveMonth,
			totalRows, protectedSkippedCount, flexibleSkippedCount, importErrors,
			"không có dữ liệu hợp lệ để tạo bảng chấm công")
	}

	result, err := s.applyTimesheetReplacement(ctx, staleIDs, entries, uploaderID, uploaderRole)
	if err != nil {
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

	// 11. Finalize.
	importErrors = append(importErrors, importErrorsFromBulkFailures(result.FailedEntries, empNames)...)
	return s.finalizeCreatedBCCImport(ctx, createdAsset, uploaderID, projectID, filename, effectiveMonth,
		totalRows, len(result.CreatedTimesheets), protectedSkippedCount, flexibleSkippedCount,
		len(result.DeletedTimesheets), importErrors)
}

// validateSheetPositions returns a failure reason when any sheet's position is
// absent from the payrate config, or "" when all positions are configured.
func validateSheetPositions(parsed *excelparser.MultiPositionImportData, flatRates map[string]int) string {
	availablePositions := getPositions(flatRates)
	availableSet := make(map[string]bool, len(availablePositions))
	for _, p := range availablePositions {
		availableSet[strings.ToLower(p)] = true
	}
	for _, sheet := range parsed.Sheets {
		if !availableSet[strings.ToLower(sheet.Position)] {
			return fmt.Sprintf("vị trí \"%s\" không có trong cấu hình lương. Các vị trí khả dụng: %s",
				sheet.Position, strings.Join(availablePositions, ", "))
		}
	}
	return ""
}

// buildCCCDToPositionMap maps every parsed employee (keyed by the template's
// EmployeeCode column, which holds CCCD) to their sheet's position. An
// employee appearing in two different position sheets is an error.
func buildCCCDToPositionMap(parsed *excelparser.MultiPositionImportData) ([]domain.ImportError, map[string]string) {
	var importErrors []domain.ImportError
	cccdToPosition := make(map[string]string)
	for _, sheet := range parsed.Sheets {
		for _, emp := range sheet.Employees {
			if emp.EmployeeCode == "" { // EmployeeCode is CCCD in multi-position template
				continue
			}
			if existing, dup := cccdToPosition[emp.EmployeeCode]; dup && existing != sheet.Position {
				importErrors = append(importErrors, domain.ImportError{
					Employee: emp.FullName,
					Reason:   fmt.Sprintf("nhân viên \"%s\" xuất hiện ở nhiều sheet vị trí (%s, %s)", emp.FullName, existing, sheet.Position),
				})
				continue
			}
			cccdToPosition[emp.EmployeeCode] = sheet.Position
		}
	}
	return importErrors, cccdToPosition
}

// autoCreateMultiPositionSTK upserts employees from the STK sheet with the
// position taken from the CCCD→position map (falling back to the
// alphabetically-first configured position). Assignments whose position
// drifted from the map are collected for post-transaction correction via
// ProjectEmployeeService (cache invalidation + events + recalculation).
func (s *BCCImportService) autoCreateMultiPositionSTK(
	ctx context.Context,
	xf *excelize.File,
	cccdToPosition map[string]string,
	availablePositions []string,
	projectID uint,
	uploaderID uint,
	monthStart time.Time,
) ([]posCorrection, []domain.ImportError) {
	var positionCorrections []posCorrection
	var importErrors []domain.ImportError

	stkRows, stkErr := excelparser.ParseSTKSheet(xf)
	if stkErr != nil {
		slog.Warn("BCCImport(MP): failed to parse STK sheet", "error", stkErr)
		return nil, nil
	}
	if len(stkRows) == 0 {
		return nil, nil
	}

	slog.Info("BCCImport(MP): found STK sheet, processing employee auto-creation", "count", len(stkRows))

	sort.Strings(availablePositions)

	stkErr = s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		bankCache := make(map[string]*uint)

		for _, row := range stkRows {
			cccd := row.CCCD
			fullName := row.FullName
			if cccd == "" || fullName == "" {
				continue
			}

			// Resolve bank ID.
			var bankID *uint
			if row.BankName != "" {
				if cached, ok := bankCache[row.BankName]; ok {
					bankID = cached
				} else {
					bankID = s.employeeService.ResolveBankID(txCtx, row.BankName)
					bankCache[row.BankName] = bankID
				}
			}

			// Check if employee exists.
			existingEmp, empErr := s.employeeService.GetEmployeeByCCCD(txCtx, cccd)
			if empErr != nil && !domain.IsNotFoundError(empErr) {
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
						slog.Error("BCCImport(MP): failed to update bank info for employee",
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
						slog.Error("BCCImport(MP): failed to fill mobile for existing employee",
							"employee_id", emp.ID, "error", err)
					}
				}
				if emp.UserID == nil {
					if err := s.ensureEmployeeUserAccount(txCtx, emp.ID); err != nil {
						importErrors = append(importErrors, domain.ImportError{
							Employee: fullName,
							Reason:   fmt.Sprintf("không thể tạo tài khoản nhân viên CCCD %s: %v", cccd, err),
						})
						continue
					}
				}
			}

			// Ensure assignment with position from CCCD→position map.
			existingAssignment, assignErr := s.employeeService.GetActiveAssignment(txCtx, projectID, emp.ID)
			if assignErr != nil && !domain.IsNotFoundError(assignErr) {
				importErrors = append(importErrors, domain.ImportError{
					Employee: fullName,
					Reason:   fmt.Sprintf("lỗi kiểm tra phân công nhân viên %s: %v", fullName, assignErr),
				})
				continue
			}
			if existingAssignment == nil {
				empPosition := cccdToPosition[cccd]
				if empPosition == "" && len(availablePositions) > 0 {
					empPosition = availablePositions[0]
				}
				assignment := &domain.ProjectEmployee{
					ProjectID:       projectID,
					EmployeeID:      emp.ID,
					EmployeeName:    emp.Fullname,
					EmployeeCCCD:    emp.CCCD,
					Position:        empPosition,
					StartDate:       monthStart,
					PaymentSchedule: string(domain.PaymentScheduleWeekly),
					// New assignments start advance-request enabled (explicit;
					// guards later full-row Saves from persisting the zero value).
					AdvanceRequestEnabled: true,
					CreatedBy:             uploaderID,
				}
				if createErr := s.employeeService.CreateAssignment(txCtx, assignment); createErr != nil {
					importErrors = append(importErrors, domain.ImportError{
						Employee: fullName,
						Reason:   fmt.Sprintf("không thể phân công nhân viên %s: %v", fullName, createErr),
					})
				}
			} else if expectedPos := cccdToPosition[cccd]; expectedPos != "" && !strings.EqualFold(existingAssignment.Position, expectedPos) {
				// Collect position correction to apply after transaction via ProjectEmployeeService
				// (which handles cache invalidation, events, validation, and timesheet recalculation).
				positionCorrections = append(positionCorrections, posCorrection{
					assignmentID: existingAssignment.ID,
					employeeID:   emp.ID,
					oldPosition:  existingAssignment.Position,
					newPosition:  expectedPos,
					employeeName: fullName,
				})
			}
		}
		return nil
	})
	if stkErr != nil {
		slog.Error("BCCImport(MP): STK auto-creation transaction failed", "error", stkErr)
	}

	return positionCorrections, importErrors
}

// applyPositionCorrections moves drifted assignments to the position the file
// declares, through ProjectEmployeeService so cache invalidation, events, and
// timesheet recalculation run.
func (s *BCCImportService) applyPositionCorrections(ctx context.Context, corrections []posCorrection, uploaderID uint) []domain.ImportError {
	var importErrors []domain.ImportError
	for _, corr := range corrections {
		if updateErr := s.projectEmployeeSvc.UpdateAssignmentPosition(ctx, corr.assignmentID, corr.newPosition, uploaderID); updateErr != nil {
			slog.Error("BCCImport(MP): failed to update assignment position",
				"assignment_id", corr.assignmentID, "position", corr.newPosition, "error", updateErr)
			importErrors = append(importErrors, domain.ImportError{
				Employee: corr.employeeName,
				Reason:   fmt.Sprintf("lỗi cập nhật vị trí cho nhân viên %s: %v", corr.employeeName, updateErr),
			})
		} else {
			slog.Info("BCCImport(MP): updated assignment position",
				"employee_id", corr.employeeID, "old", corr.oldPosition, "new", corr.newPosition)
		}
	}
	return importErrors
}

// buildMultiPositionEntries converts parsed position sheets into bulk timesheet
// entries, matching employees by CCCD+position first. Non-STK employees whose
// assignment position doesn't match their sheet get corrected in place.
func (s *BCCImportService) buildMultiPositionEntries(
	ctx context.Context,
	parsed *excelparser.MultiPositionImportData,
	flatRates map[string]int,
	year int,
	month time.Month,
	loc *time.Location,
	byCCCD map[string]*domain.ProjectEmployee,
	byCode map[string]*domain.ProjectEmployee,
	byCCCDAndPosition map[string]*domain.ProjectEmployee,
	projectID uint,
	uploaderID uint,
	includeFlexibleEmployees bool,
) (entries []domainservices.BulkCreateTimesheetEntry, flexibleEmployeeIDs map[uint]struct{}, totalRows int, importErrors []domain.ImportError) {
	flexibleEmployeeIDs = make(map[uint]struct{})

	// Normalize day type: accept both short names (Thường, Nghỉ, Lễ) and full names
	// (ngày thường, ngày nghỉ, ngày lễ) — dayTypePriority and rateTarget are the
	// shared package-level declarations.
	for _, sheet := range parsed.Sheets {
		// Build position-scoped rate-to-target map.
		rateToTarget := buildRateToTarget(flatRates, strings.ToLower(sheet.Position)+".")

		for _, emp := range sheet.Employees {
			totalRows++

			// Match by CCCD+position first, then fall back.
			// Note: emp.EmployeeCode holds CCCD in the multi-position template, so it matches
			// the byCCCDAndPosition map keyed by assignment.EmployeeCCCD.
			empLookupKey := emp.EmployeeCode // CCCD in multi-position template
			assignment := byCCCDAndPosition[empLookupKey+"|"+strings.ToLower(sheet.Position)]
			if assignment == nil {
				assignment = byCCCD[empLookupKey]
			}
			if assignment == nil {
				assignment = byCode[empLookupKey]
			}
			if assignment == nil {
				importErrors = append(importErrors, domain.ImportError{
					Employee: emp.FullName,
					Reason:   fmt.Sprintf("không tìm thấy nhân viên với mã \"%s\" trong dự án", emp.EmployeeCode),
				})
				continue
			}

			// Correct position for non-STK employees whose assignment position doesn't match this sheet.
			if !strings.EqualFold(assignment.Position, sheet.Position) {
				oldPos := assignment.Position
				if updateErr := s.projectEmployeeSvc.UpdateAssignmentPosition(ctx, assignment.ID, sheet.Position, uploaderID); updateErr != nil {
					slog.Error("BCCImport(MP): failed to correct position in BCC sheet",
						"assignment_id", assignment.ID, "position", sheet.Position, "error", updateErr)
					importErrors = append(importErrors, domain.ImportError{
						Employee: emp.FullName,
						Reason:   fmt.Sprintf("lỗi cập nhật vị trí cho nhân viên %s: %v", emp.FullName, updateErr),
					})
				} else {
					slog.Info("BCCImport(MP): corrected assignment position from BCC sheet",
						"employee_id", assignment.EmployeeID, "old", oldPos, "new", sheet.Position)
					assignment.Position = sheet.Position
					// Update lookup maps so subsequent sheets find the corrected position.
					byCCCDAndPosition[assignment.EmployeeCCCD+"|"+strings.ToLower(sheet.Position)] = assignment
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

			for _, entry := range emp.Entries {
				date := time.Date(year, month, entry.DayNum, 0, 0, 0, 0, loc)
				if date.Month() != month {
					continue
				}

				target, ok := rateToTarget[entry.RateVND]
				if !ok || entry.RateVND == 0 {
					importErrors = append(importErrors, domain.ImportError{
						Employee: emp.FullName,
						Reason:   fmt.Sprintf("không tìm thấy mức lương cho ngày %d (%d VND) ở vị trí %s", entry.DayNum, entry.RateVND, sheet.Position),
					})
					continue
				}

				// Normalize day type for consistent storage (Thường → ngày thường, etc.)
				normalizedDayType := normalizeDayType(target.dayType)
				entries = append(entries, domainservices.BulkCreateTimesheetEntry{
					ProjectID:   projectID,
					EmployeeID:  assignment.EmployeeID,
					Date:        date.Format("2006-01-02"),
					HoursWorked: entry.Hours,
					HourType:    target.hourType,
					DayType:     &normalizedDayType,
				})
			}
		}
	}

	return entries, flexibleEmployeeIDs, totalRows, importErrors
}

// getPositions extracts unique position names from flattened payrate paths.
func getPositions(flatRates map[string]int) []string {
	seen := make(map[string]bool)
	var positions []string
	for path := range flatRates {
		parts := strings.Split(path, ".")
		if len(parts) >= 1 && parts[0] != "" && !seen[parts[0]] {
			seen[parts[0]] = true
			positions = append(positions, parts[0])
		}
	}
	return positions
}

// bccNormName normalizes a Vietnamese name for case/whitespace/Unicode-form
// insensitive comparison: trim, NFC, lower-case. Excel files from macOS can
// ship Vietnamese diacritics in NFD (decomposed) form, which would otherwise
// produce false mismatches against NFC text from other tools.
func bccNormName(s string) string {
	return strings.ToLower(strings.TrimSpace(norm.NFC.String(s)))
}

// normalizeDayType converts short day type names to full names for consistent storage.
// Maps: Thường → ngày thường, Nghỉ → ngày nghỉ, Lễ → ngày lễ
// Full names are returned as-is (case-insensitive).
func normalizeDayType(dayType string) string {
	normalized := strings.ToLower(strings.TrimSpace(dayType))
	switch normalized {
	case "thường":
		return "ngày thường"
	case "nghỉ":
		return "ngày nghỉ"
	case "lễ":
		return "ngày lễ"
	default:
		// Return original if it's already a full name or unknown
		if strings.HasPrefix(normalized, "ngày ") {
			return dayType // Return as-is to preserve case
		}
		return dayType // Return unknown values as-is
	}
}

// bccNormNameLoose normalizes a Vietnamese name the same way as bccNormName,
// then strips diacritics (e.g. "Lò Thì Dương" → "lo thi duong"). Used to
// detect names that differ only by an accent mark (a common data-entry typo
// where the same person is recorded with a slightly different diacritic in
// different sheets — STK vs BCC, or BCC vs the employee profile). Two names
// whose loose-normalized forms are equal almost certainly refer to the
// same person, even when the strict-normalized forms differ.
//
// The diacritic fold delegates to the canonical utils.NormalizeVietnamese
// (unidecode-based; folds Đ/đ automatically). The leading NFC guards against
// macOS-Excel NFD input and the TrimSpace mirrors bccNormName.
func bccNormNameLoose(s string) string {
	return strings.TrimSpace(utils.NormalizeVietnamese(norm.NFC.String(s)))
}
