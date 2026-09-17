package services

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	excelparser "api-server/internal/app/services/excel"
	"api-server/internal/domain"
	domainservices "api-server/internal/domain/services"
	"api-server/internal/pkg/excelkit"

	"github.com/xuri/excelize/v2"
)

func (s *BCCImportService) processAssetData(
	ctx context.Context,
	data []byte,
	filename string,
	projectID uint,
	uploaderID uint,
	uploaderRole string,
	createdAsset *domain.Asset,
	forMonth string,
	includeFlexibleEmployees bool,
) (*BCCImportResult, error) {

	// effectiveMonth is captured by the fail() closure.
	effectiveMonth := forMonth

	// Helper to update asset metadata and return result.
	fail := s.bccFailer(ctx, createdAsset, uploaderID, projectID, filename, effectiveMonth)

	// 4. Parse the Excel file.
	xf, err := excelkit.OpenReader(bytes.NewReader(data))
	if err != nil {
		return fail(fmt.Sprintf("không thể mở file Excel: %v", err))
	}
	defer func() { _ = xf.Close() }()

	// 4a. Detect format: legacy BCC vs multi-position
	formatResult, detectErr := excelparser.DetectFormat(xf)
	if detectErr != nil {
		return fail(fmt.Sprintf("không nhận diện được định dạng file: %v", detectErr))
	}

	var parsed *excelparser.BCCImportData
	switch formatResult.Format {
	case excelparser.FormatMultiPosition:
		return s.processMultiPositionUpload(ctx, xf, formatResult, filename, projectID, uploaderID, uploaderRole, createdAsset, effectiveMonth, includeFlexibleEmployees)
	case excelparser.FormatWeeklyBCC:
		return s.processWeeklyBCCUpload(ctx, xf, formatResult, filename, projectID, uploaderID, uploaderRole, createdAsset, effectiveMonth, includeFlexibleEmployees)
	case excelparser.FormatWeeklyPayment:
		return s.processWeeklyPaymentUpload(ctx, xf, formatResult, filename, projectID, uploaderID, uploaderRole, createdAsset, effectiveMonth, includeFlexibleEmployees)
	case excelparser.FormatDateRow:
		// Partner date-row template (BUMHAN M1-style): entries carry exact
		// calendar dates and the file has no rate row (rateless path).
		parsed, err = excelparser.ParseDateRowBCCFile(xf, formatResult.DateRowSheets)
	default:
		parsed, err = parseLegacyRoute(xf, formatResult, filename)
	}
	if err != nil {
		return fail(fmt.Sprintf("lỗi phân tích file BCC: %v", err))
	}

	// Parse summary for import diagnostics: employees/entries as read from the
	// file (pre in-month filter) and whether the file carries a rate row.
	totalEntries := 0
	for _, emp := range parsed.Employees {
		totalEntries += len(emp.Entries)
	}
	slog.Info("BCCImport: parsed file summary",
		"employees", len(parsed.Employees), "entries", totalEntries,
		"rates", len(parsed.ShiftRates))

	// 5-6. Shared setup: month parsing, lock, payrate lookup. The earliest
	// worked day lets the lookup accept a payrate that starts mid-month but
	// is already active on the file's first worked day. Date-row files can
	// straddle months (e.g. 21/08–24/09); only days inside the import month
	// count, so the payrate probe never targets an out-of-month day.
	earliestDay := 0
	if importY, importM, mErr := parseForMonth(effectiveMonth); mErr == nil {
		earliestDay = earliestInMonthDay(parsed.Employees, importY, importM)
	}
	ictx, releaseLock, err := s.prepareImportContext(ctx, projectID, effectiveMonth, earliestDay)
	if err != nil {
		return fail(err.Error())
	}
	defer releaseLock()
	year, month, monthStart, flatRates := ictx.year, ictx.month, ictx.monthStart, ictx.flatRates
	loc := monthStart.Location()

	// Build rate -> (dayType, hourType) lookup from the project payrate.
	// Rate is king: as long as the rate matches, the entry is valid.
	// On collision (same rate, multiple paths), prefer "ngay thuong".
	// Normalize day type: accept both short names (Thường, Nghỉ, Lễ) and full names (ngày thường, ngày nghỉ, ngày lễ)
	//
	// ratelessFile: templates without any rate row (e.g. Samsung SDS) can't do
	// rate matching at all — their entries fall back to label + calendar
	// mapping below. Files that carry rates keep the strict rate path.
	ratelessFile := len(parsed.ShiftRates) == 0
	rateToTarget := buildRateToTarget(flatRates, "")

	// 6.5 Auto-create and assign employees from STK sheet if it exists.
	// Deduce the best position from payrate rates matching BCC shift rates.
	position := deducePosition(flatRates, parsed.ShiftRates)
	stkRows := collectSTKRows(xf, parsed, formatResult.Format)
	stkNameByCCCD, stkImportErrors := s.autoCreateEmployeesFromSTK(
		ctx, stkRows, parsed, formatResult.Format,
		projectID, uploaderID, position, flatRates,
	)
	importErrors := stkImportErrors

	// 7. Load all active project employees, build lookup maps.
	assignments, err := s.employeeService.GetActiveAssignments(ctx, projectID)
	if err != nil {
		return fail(fmt.Sprintf("lỗi tải danh sách nhân viên: %v", err))
	}
	byCCCD := make(map[string]*domain.ProjectEmployee, len(assignments))
	byCode := make(map[string]*domain.ProjectEmployee, len(assignments))
	empNames := make(map[uint]string, len(assignments))
	for _, a := range assignments {
		byCCCD[a.EmployeeCCCD] = a
		if a.EmployeeCode != "" {
			byCode[a.EmployeeCode] = a
		}
		empNames[a.EmployeeID] = a.EmployeeName
	}

	// 8. Build timesheet entries, collecting employee errors.
	labelKeyedFile := useLabelKeyedResolution(ratelessFile, formatResult.Format)
	var (
		entries             []domainservices.BulkCreateTimesheetEntry
		flexibleEmployeeIDs = make(map[uint]struct{})
		rowNum              = 12
	)

	for _, emp := range parsed.Employees {
		assignment := byCCCD[emp.CCCD]
		if assignment == nil && emp.EmployeeCode != "" {
			assignment = byCode[emp.EmployeeCode]
		}
		if assignment == nil {
			// Match the wording used by multi-position and weekly-rate lookup
			// paths: quote the lookup key so the partner sees whether the
			// miss is a typo or a missing project assignment. Prefer CCCD
			// (date-row / multi-position identifier) when both are present;
			// otherwise fall back to the employee code (legacy templates).
			lookupKey := emp.CCCD
			if lookupKey == "" {
				lookupKey = emp.EmployeeCode
			}
			importErrors = append(importErrors, domain.ImportError{
				Row:      rowNum,
				Employee: emp.FullName,
				Reason:   fmt.Sprintf("không tìm thấy nhân viên \"%s\" trong dự án", lookupKey),
			})
			rowNum++
			continue
		}

		// STK cross-check: if the same CCCD appears in STK with a different
		// name, the BCC sheet likely has a CCCD typo (one CCCD assigned to
		// two different people).
		if mismatch := crossCheckSTKName(emp, stkNameByCCCD); mismatch != nil {
			mismatch.Row = rowNum
			importErrors = append(importErrors, *mismatch)
			rowNum++
			continue
		}

		if assignment.PaymentSchedule == string(domain.PaymentScheduleFlexible) {
			if !includeFlexibleEmployees {
				importErrors = append(importErrors, domain.ImportError{
					Row:      rowNum,
					Employee: emp.FullName,
					Reason:   "nhân viên lương linh hoạt không áp dụng BCC import",
				})
				rowNum++
				continue
			}
			flexibleEmployeeIDs[assignment.EmployeeID] = struct{}{}
		}

		for _, entry := range emp.Entries {
			date := time.Date(year, month, entry.DayNum, 0, 0, 0, 0, loc)
			if entry.FullDate != nil {
				// Date-row templates carry exact calendar dates; the in-month
				// filter below drops days belonging to other months.
				date = *entry.FullDate
			}
			if date.Year() != year || date.Month() != month {
				continue
			}

			target, ok := resolveBCCEntryTarget(parsed.ShiftRates, rateToTarget, flatRates, entry.ShiftLabel, date, ratelessFile, labelKeyedFile)
			if !ok {
				// A zero-hour cell carries no rate burden, so an unpriced column
				// is not an error — the weekly paths skip rate resolution for
				// these outright. Templates routinely ship a spare sub-column
				// (EPE's Sunday TCNN) filled with zeros and no rate in the rate
				// row; erroring on it fails an import that has nothing to book.
				// Legacy deletion is keyed on employee+date, which this date's
				// priced cells already cover, so dropping it loses no intent.
				if entry.Hours <= 0 {
					continue
				}
				// Rateless files have no VND figure to quote — "(0 VND)" would
				// read as a zero-rate config error rather than an unmapped code.
				reason := fmt.Sprintf("không tìm thấy mức lương cho ca %s", entry.ShiftLabel)
				if rate := parsed.ShiftRates[entry.ShiftLabel]; rate > 0 {
					reason = fmt.Sprintf("%s (%d VND)", reason, rate)
				}
				importErrors = append(importErrors, domain.ImportError{
					Row:      rowNum,
					Employee: emp.FullName,
					Reason:   reason,
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
		rowNum++
	}

	totalRows := len(parsed.Employees)

	// 8b. Align assignment start dates with the entries this file proves.
	s.backdateAssignmentsForImport(ctx, assignments, entries, uploaderID, filename)

	// 9. Preserve reviewed rows, replace pending rows, and create missing rows.
	// Same-bucket entries are collapsed inside planMonthReplacement.
	entries, staleIDs, protectedSkippedCount, flexibleSkippedCount, replErr := s.planMonthReplacement(
		ctx, projectID, year, month, monthStart, loc, entries, flexibleEmployeeIDs, false)
	if replErr != nil {
		return fail(fmt.Sprintf("lỗi tải bảng chấm công hiện có: %v", replErr))
	}

	// 10. Call BulkCreateTimesheets.
	if len(entries) == 0 {
		reason := "không có dữ liệu hợp lệ để tạo bảng chấm công"
		// Date-row files carry exact calendar dates; when every entry fell
		// outside the selected month, the generic reason hides the actual fix
		// (pick the month the data belongs to). Quote the file's data range.
		if len(importErrors) == 0 {
			var minDate, maxDate time.Time
			datedEntries := 0
			for _, emp := range parsed.Employees {
				for _, e := range emp.Entries {
					if e.FullDate == nil {
						continue
					}
					datedEntries++
					if minDate.IsZero() || e.FullDate.Before(minDate) {
						minDate = *e.FullDate
					}
					if maxDate.IsZero() || e.FullDate.After(maxDate) {
						maxDate = *e.FullDate
					}
				}
			}
			if datedEntries > 0 {
				reason = fmt.Sprintf(
					"file chỉ chứa giờ công ngoài tháng %s (dữ liệu từ %s đến %s) — chọn tháng tương ứng với dữ liệu file",
					effectiveMonth, minDate.Format("02/01/2006"), maxDate.Format("02/01/2006"))
			}
		}
		return s.finishEmptyBCCImport(ctx, createdAsset, uploaderID, projectID, filename, effectiveMonth,
			totalRows, protectedSkippedCount, flexibleSkippedCount, importErrors, reason)
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

	// 11-12. Count results and persist the completed import's stats via the
	// shared tail. Skipped no longer includes zero-hour no-ops: a zero cell
	// whose day has no pending row did nothing visible to the partner and
	// would only inflate the "Skipped" number on screen.
	importErrors = append(importErrors, importErrorsFromBulkFailures(result.FailedEntries, empNames)...)
	return s.finalizeCreatedBCCImport(ctx, createdAsset, uploaderID, projectID, filename, effectiveMonth,
		totalRows, len(result.CreatedTimesheets), protectedSkippedCount, flexibleSkippedCount,
		len(result.DeletedTimesheets), importErrors)
}

// parseLegacyRoute handles the default (BCC-sheet-named) dispatch branch: the
// "BCC" sheet name can carry a date-row layout (partner T09), so route by
// parse outcome — ordered strategies run until one actually reads the file.
//
// Load-bearing contract: on success the winner's format REPLACES the
// sheet-name detection in formatResult. The downstream behavior keys
// (collectSTKRows, resolveImportPosition, useLabelKeyedResolution) read the
// winner, not what DetectFormat guessed.
func parseLegacyRoute(xf *excelize.File, formatResult *excelparser.FormatDetectionResult, filename string) (*excelparser.BCCImportData, error) {
	parsed, winner, err := excelparser.ParseBCCData(xf)
	if err == nil {
		slog.Info("BCCImport: strategy routing picked parser",
			"format", winner, "filename", filename)
		formatResult.Format = winner
	}
	return parsed, err
}

// useLabelKeyedResolution reports whether entries resolve payrate leaves by
// column code (label) first. Label-keyed resolution is a date-row template
// concept: only there do column codes like NT/T7/CN name payrate leaves.
// Legacy rateless files must keep the calendar path — their CN means "ca
// ngày" while a date-row config's CN leaf means Chủ Nhật, so running
// label-first on them silently re-prices old templates.
func useLabelKeyedResolution(ratelessFile bool, format excelparser.BCCFormat) bool {
	return ratelessFile && format == excelparser.FormatDateRow
}
