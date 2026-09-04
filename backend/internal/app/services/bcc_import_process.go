package services

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	excelparser "api-server/internal/app/services/excel"
	"api-server/internal/domain"
	domainservices "api-server/internal/domain/services"
	"api-server/internal/pkg/clock"

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
			slog.Error("BCCImport: metadata update failed in fail path", "asset_id", createdAsset.ID, "error", metaErr)
		}
		return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt),
			fmt.Errorf("import failed: %s", FirstErrorReason(detail))
	}

	// 4. Parse the Excel file.
	xf, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return fail("failed", fmt.Sprintf("không thể mở file Excel: %v", err))
	}
	defer func() { _ = xf.Close() }()

	// 4a. Detect format: legacy BCC vs multi-position
	formatResult, detectErr := excelparser.DetectFormat(xf)
	if detectErr != nil {
		return fail("failed", fmt.Sprintf("không nhận diện được định dạng file: %v", detectErr))
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
		// The "BCC" sheet name can carry a date-row layout (partner T09
		// template), so route by parse outcome: ordered strategies run until
		// one actually reads the file, and the format reflects the winner so
		// post-parse behavior keys to the parser that ran.
		var winner excelparser.BCCFormat
		parsed, winner, err = excelparser.ParseBCCData(xf)
		if err == nil {
			slog.Info("BCCImport: strategy routing picked parser",
				"format", winner, "filename", filename)
			formatResult.Format = winner
		}
	}
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi phân tích file BCC: %v", err))
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
		return fail("failed", err.Error())
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
	rateToTarget := make(map[int]rateTarget)
	for path, rate := range flatRates {
		if rate == 0 {
			continue
		}
		parts := strings.Split(path, ".")
		if len(parts) != 3 {
			continue
		}
		candidate := rateTarget{parts[1], parts[2]}
		candPri, candKnown := dayTypePriority[candidate.dayType]
		if !candKnown {
			continue // skip unrecognized day types
		}
		existing, exists := rateToTarget[rate]
		if !exists || candPri < dayTypePriority[existing.dayType] {
			rateToTarget[rate] = candidate
		}
	}

	// 6.5 Auto-create and assign employees from STK sheet if it exists.
	// Deduce the best position from payrate rates matching BCC shift rates.
	position := deducePosition(flatRates, parsed.ShiftRates)
	monthStartDate := time.Date(year, month, 1, 0, 0, 0, 0, loc)

	// Pre-declare importErrors so STK auto-creation failures are surfaced to the user.
	var importErrors []domain.ImportError

	var stkRows []excelparser.STKRow
	if formatResult.Format == excelparser.FormatDateRow {
		// Date-row template: the BCC sheet itself carries employee and bank
		// info, so employees are upserted from the parsed rows. A real STK
		// sheet, when present, still wins for bank/mobile corrections and
		// STK-only hires.
		stkRows = dateRowEmployeeToSTKRows(parsed.Employees)
		if stkParsed, stkErr := excelparser.ParseSTKSheet(xf); stkErr != nil {
			slog.Warn("BCCImport: failed to parse STK sheet", "error", stkErr)
		} else {
			stkRows = mergeSTKRows(stkRows, stkParsed)
		}
	} else {
		var stkErr error
		stkRows, stkErr = excelparser.ParseSTKSheet(xf)
		if stkErr != nil {
			slog.Warn("BCCImport: failed to parse STK sheet", "error", stkErr)
		}
	}

	// Build STK CCCD→Name lookup for cross-validation against BCC employee names.
	stkNameByCCCD := make(map[string]string, len(stkRows))
	for _, row := range stkRows {
		if row.CCCD != "" && row.FullName != "" {
			stkNameByCCCD[row.CCCD] = row.FullName
		}
	}

	if len(stkRows) > 0 {
		slog.Info("BCCImport: found STK sheet, processing employee auto-creation",
			"count", len(stkRows), "position", position)

		// Cache bank name → ID to avoid redundant DB queries per row.
		// Wrap STK auto-creation in a transaction to prevent orphaned records.
		stkErr := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
			bankCache := make(map[string]*uint)

			for _, row := range stkRows {
				cccd := row.CCCD
				fullName := row.FullName
				if cccd == "" || fullName == "" {
					continue
				}

				// 1. Resolve Bank ID (cached)
				var bankID *uint
				if row.BankName != "" {
					if cached, ok := bankCache[row.BankName]; ok {
						bankID = cached
					} else {
						bankID = s.employeeService.ResolveBankID(txCtx, row.BankName)
						bankCache[row.BankName] = bankID
					}
				}

				// 2. Check if employee already exists by CCCD
				existingEmp, empErr := s.employeeService.GetEmployeeByCCCD(txCtx, cccd)
				if empErr != nil && !domain.IsNotFoundError(empErr) {
					// Real DB error — don't conflate with "not found".
					slog.Error("BCCImport: DB error looking up employee by CCCD",
						"cccd", cccd, "error", empErr)
					importErrors = append(importErrors, domain.ImportError{
						Employee: fullName,
						Reason:   fmt.Sprintf("lỗi tra cứu nhân viên CCCD %s: %v", cccd, empErr),
					})
					continue
				}

				var emp *domain.Employee
				if existingEmp == nil {
					// Create new employee
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
						slog.Error("BCCImport: failed to auto-create employee",
							"cccd", cccd, "name", fullName, "error", createErr)
						importErrors = append(importErrors, domain.ImportError{
							Employee: fullName,
							Reason:   fmt.Sprintf("không thể tạo nhân viên CCCD %s: %v", cccd, createErr),
						})
						continue
					}
					emp = createdEmp
					slog.Info("BCCImport: auto-created employee profile and user account",
						"cccd", cccd, "employee_id", emp.ID)
				} else {
					emp = existingEmp

					// Apply changed bank info from STK using a targeted,
					// atomically validated update.
					if bankUpdates := buildSTKBankUpdates(emp, row, bankID, fullName); bankUpdates != nil {
						if updateErr := s.employeeService.UpdateBankInfo(txCtx, emp.ID, bankUpdates); updateErr != nil {
							slog.Error("BCCImport: failed to update bank info for employee",
								"employee_id", emp.ID, "error", updateErr)
							importErrors = append(importErrors, domain.ImportError{
								Employee: fullName,
								Reason:   "Không thể cập nhật thông tin ngân hàng",
							})
						} else {
							slog.Info("BCCImport: updated bank info for existing employee",
								"employee_id", emp.ID, "cccd", cccd)
						}
					}

					// Fill missing mobile from STK when employee has none.
					if row.Mobile != "" && emp.Mobile == "" {
						if err := s.employeeService.UpdateMobile(txCtx, emp.ID, row.Mobile); err != nil {
							slog.Error("BCCImport: failed to fill mobile for existing employee",
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

				// 3. Ensure employee is assigned to the project
				existingAssignment, assignErr := s.employeeService.GetActiveAssignment(txCtx, projectID, emp.ID)
				if assignErr != nil || existingAssignment == nil {
					// Deduce position specifically for this employee based on their timesheet entries
					empPosition := position
					var matchedParsedEmp *excelparser.BCCEmployeeData
					for idx := range parsed.Employees {
						if parsed.Employees[idx].CCCD == cccd {
							matchedParsedEmp = &parsed.Employees[idx]
							break
						}
					}
					if matchedParsedEmp != nil && len(matchedParsedEmp.Entries) > 0 {
						empRates := make(map[string]int64)
						for _, entry := range matchedParsedEmp.Entries {
							if rate, ok := parsed.ShiftRates[entry.ShiftLabel]; ok {
								empRates[entry.ShiftLabel] = rate
							}
						}
						empPosition = deducePosition(flatRates, empRates)
					}

					// Date-row templates carry each employee's position in the
					// file's "Vị trí" column; it outranks rate-based deduction,
					// which is meaningless for rateless files (always "phổ
					// thông"). Blank cells keep the deduced default.
					if formatResult.Format == excelparser.FormatDateRow &&
						matchedParsedEmp != nil && strings.TrimSpace(matchedParsedEmp.Position) != "" {
						empPosition = strings.TrimSpace(matchedParsedEmp.Position)
					}

					assignment := &domain.ProjectEmployee{
						ProjectID:       projectID,
						EmployeeID:      emp.ID,
						EmployeeName:    emp.Fullname,
						EmployeeCCCD:    emp.CCCD,
						Position:        empPosition,
						StartDate:       monthStartDate,
						PaymentSchedule: string(domain.PaymentScheduleWeekly),
						CreatedBy:       uploaderID,
					}

					if createErr := s.employeeService.CreateAssignment(txCtx, assignment); createErr != nil {
						slog.Error("BCCImport: failed to auto-assign employee to project",
							"employee_id", emp.ID, "project_id", projectID, "error", createErr)
						importErrors = append(importErrors, domain.ImportError{
							Employee: fullName,
							Reason:   fmt.Sprintf("không thể phân công nhân viên %s vào dự án: %v", fullName, createErr),
						})
					} else {
						slog.Info("BCCImport: auto-assigned employee to project",
							"employee_id", emp.ID, "project_id", projectID,
							"position", empPosition, "start_date", monthStartDate.Format("2006-01-02"))
					}
				}
			}
			return nil
		})
		if stkErr != nil {
			slog.Error("BCCImport: STK auto-creation transaction failed", "error", stkErr)
		}
	}

	// 7. Load all active project employees, build lookup maps.
	assignments, err := s.employeeService.GetActiveAssignments(ctx, projectID)
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi tải danh sách nhân viên: %v", err))
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
	// Label-keyed resolution (column code = payrate shift leaf) is a date-row
	// template concept: only there do column codes like NT/T7/CN name payrate
	// leaves. Legacy rateless files must keep the calendar path — their CN
	// means "ca ngày" while a date-row config's CN leaf means Chủ Nhật, so
	// running label-first on them silently re-prices old templates.
	labelKeyedFile := ratelessFile && formatResult.Format == excelparser.FormatDateRow
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
			importErrors = append(importErrors, domain.ImportError{
				Row:      rowNum,
				Employee: emp.FullName,
				Reason:   "nhân viên không tìm thấy trong hệ thống",
			})
			rowNum++
			continue
		}

		// STK cross-check: if the same CCCD appears in STK with a different name,
		// the BCC sheet likely has a CCCD typo (one CCCD assigned to two different people).
		if stkName, ok := stkNameByCCCD[emp.CCCD]; ok && stkName != "" {
			// Normalize both names for comparison: trim, NFC, lower-case. NFC is
			// important because Excel files from macOS can ship Vietnamese text
			// in NFD form, which would otherwise produce a spurious mismatch.
			bccNorm := bccNormName(emp.FullName)
			stkNorm := bccNormName(stkName)
			if bccNorm != stkNorm {
				// Names differ after NFC normalization. Try a diacritic-stripped
				// comparison before rejecting: a single accent typo (e.g. "Thì" vs
				// "Thị") is data-entry noise, not a real CCCD mismatch. Allow the
				// import and log a warning so the partner can correct the typo.
				if bccNormNameLoose(emp.FullName) == bccNormNameLoose(stkName) {
					slog.Warn("BCCImport: STK/BCC name differs only by diacritic, allowing import",
						"cccd", emp.CCCD, "bcc_name", emp.FullName, "stk_name", stkName)
				} else {
					cleanBCCName := strings.TrimSpace(emp.FullName)
					cleanSTKName := strings.TrimSpace(stkName)
					importErrors = append(importErrors, domain.ImportError{
						Row:      rowNum,
						Employee: cleanBCCName,
						Reason:   fmt.Sprintf("CCCD %s thuộc về %s (theo STK), không phải %s — có thể sai CCCD", emp.CCCD, cleanSTKName, cleanBCCName),
					})
					rowNum++
					continue
				}
			}
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

	// 9. Preserve reviewed rows, replace pending rows, and create missing rows.
	var staleIDs []uint
	flexibleSkippedCount := 0
	protectedSkippedCount := 0
	if len(entries) > 0 {
		monthEnd := time.Date(year, month+1, 0, 23, 59, 59, 0, loc)
		existingTS, terr := s.timesheetReader.GetByProject(ctx, projectID, monthStart, monthEnd)
		if terr != nil {
			return fail("failed", fmt.Sprintf("lỗi tải bảng chấm công hiện có: %v", terr))
		}
		entries, staleIDs, protectedSkippedCount, flexibleSkippedCount = planBCCReplacement(
			entries, existingTS, flexibleEmployeeIDs, false,
		)
	}

	// 10. Call BulkCreateTimesheets.
	if len(entries) == 0 {
		if len(importErrors) == 0 && protectedSkippedCount+flexibleSkippedCount > 0 {
			return s.completeSkippedBCCImport(ctx, createdAsset, uploaderID, BCCImportStats{
				ProjectID:    projectID,
				OriginalName: filename,
				ForMonth:     effectiveMonth,
				TotalRows:    totalRows,
				SkippedCount: protectedSkippedCount + flexibleSkippedCount,
			})
		}
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
				slog.Error("BCCImport: metadata update failed for empty-entries path", "asset_id", createdAsset.ID, "error", metaErr)
			}
			return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt),
				fmt.Errorf("import failed: %s", FirstErrorReason(detail))
		}
		return fail("failed", reason)
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
		return fail("failed", fmt.Sprintf("lỗi tạo bảng chấm công: %v", err))
	}

	// 11. Count results.
	createdCount := len(result.CreatedTimesheets)
	skippedCount := len(result.DeletedTimesheets) + protectedSkippedCount + flexibleSkippedCount + countZeroHourEntries(entries)
	importErrors = append(importErrors, importErrorsFromBulkFailures(result.FailedEntries, empNames)...)
	errorCount := len(importErrors)

	now := clock.Now()
	finalStatus := "completed"
	if createdCount == 0 && errorCount > 0 {
		finalStatus = "failed"
	}

	// 12. Update asset metadata with final results.
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
		slog.Error("BCCImport: metadata update failed for success path", "asset_id", createdAsset.ID, "error", metaErr)
	}

	return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt), nil
}
