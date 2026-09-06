package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	excelparser "api-server/internal/app/services/excel"
	"api-server/internal/domain"

	"github.com/xuri/excelize/v2"
)

// STK-sheet processing for the legacy/date-row BCC pipeline, extracted from
// processAssetData. The two production contracts embedded here:
//   - a "BCC"-named sheet whose content is date-row (T09) upserts employees
//     from the BCC rows themselves, merged with any real STK sheet;
//   - a date-row file's "Vị trí" column outranks rate-deduced position
//     (rateless files would otherwise always deduce "phổ thông").

// collectSTKRows builds the effective STK row set for auto-creation. For
// date-row files the BCC sheet itself carries employee and bank info, so
// employees are upserted from the parsed rows; a real STK sheet, when
// present, still wins for bank/mobile corrections and STK-only hires.
func collectSTKRows(xf *excelize.File, parsed *excelparser.BCCImportData, format excelparser.BCCFormat) []excelparser.STKRow {
	if format == excelparser.FormatDateRow {
		stkRows := dateRowEmployeeToSTKRows(parsed.Employees)
		if stkParsed, stkErr := excelparser.ParseSTKSheet(xf); stkErr != nil {
			slog.Warn("BCCImport: failed to parse STK sheet", "error", stkErr)
		} else {
			stkRows = mergeSTKRows(stkRows, stkParsed)
		}
		return stkRows
	}
	stkRows, stkErr := excelparser.ParseSTKSheet(xf)
	if stkErr != nil {
		slog.Warn("BCCImport: failed to parse STK sheet", "error", stkErr)
	}
	return stkRows
}

// autoCreateEmployeesFromSTK upserts employee profiles, bank info, user
// accounts, and project assignments from the effective STK rows, inside one
// transaction. Per-row failures are collected as ImportErrors instead of
// aborting the import. Returns the STK CCCD→Name lookup used later for
// cross-validation.
func (s *BCCImportService) autoCreateEmployeesFromSTK(
	ctx context.Context,
	stkRows []excelparser.STKRow,
	parsed *excelparser.BCCImportData,
	format excelparser.BCCFormat,
	projectID uint,
	uploaderID uint,
	position string,
	monthStartDate time.Time,
	flatRates map[string]int,
) (map[string]string, []domain.ImportError) {
	stkNameByCCCD := make(map[string]string, len(stkRows))
	for _, row := range stkRows {
		if row.CCCD != "" && row.FullName != "" {
			stkNameByCCCD[row.CCCD] = row.FullName
		}
	}

	if len(stkRows) == 0 {
		return stkNameByCCCD, nil
	}

	var importErrors []domain.ImportError
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
				empPosition := resolveImportPosition(parsed, format, cccd, position, flatRates)

				assignment := &domain.ProjectEmployee{
					ProjectID:       projectID,
					EmployeeID:      emp.ID,
					EmployeeName:    emp.Fullname,
					EmployeeCCCD:    emp.CCCD,
					Position:        empPosition,
					StartDate:       monthStartDate,
					PaymentSchedule: string(domain.PaymentScheduleWeekly),
					// New assignments start advance-request enabled (explicit;
					// guards later full-row Saves from persisting the zero value).
					AdvanceRequestEnabled: true,
					CreatedBy:             uploaderID,
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

	return stkNameByCCCD, importErrors
}

// resolveImportPosition picks the assignment position for one STK row.
// Rate-based deduction from the employee's own BCC entries beats the
// workbook-wide default; a date-row file's "Vị trí" column beats both
// (rateless files would otherwise always deduce "phổ thông"). Blank cells
// keep the deduced value.
func resolveImportPosition(
	parsed *excelparser.BCCImportData,
	format excelparser.BCCFormat,
	cccd string,
	defaultPosition string,
	flatRates map[string]int,
) string {
	position := defaultPosition
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
		position = deducePosition(flatRates, empRates)
	}

	if format == excelparser.FormatDateRow &&
		matchedParsedEmp != nil && strings.TrimSpace(matchedParsedEmp.Position) != "" {
		position = strings.TrimSpace(matchedParsedEmp.Position)
	}

	return position
}

// crossCheckSTKName rejects a BCC row whose CCCD maps to a different person
// in STK — usually a CCCD typo (one CCCD assigned to two people). Names are
// compared NFC-normalized; a single diacritic typo is tolerated with a
// warning so partners can fix the typo without blocking the import.
// Returns nil when the row may proceed.
func crossCheckSTKName(emp excelparser.BCCEmployeeData, stkNameByCCCD map[string]string) *domain.ImportError {
	stkName, ok := stkNameByCCCD[emp.CCCD]
	if !ok || stkName == "" {
		return nil
	}
	// Normalize both names for comparison: trim, NFC, lower-case. NFC is
	// important because Excel files from macOS can ship Vietnamese text
	// in NFD form, which would otherwise produce a spurious mismatch.
	bccNorm := bccNormName(emp.FullName)
	stkNorm := bccNormName(stkName)
	if bccNorm == stkNorm {
		return nil
	}
	// Names differ after NFC normalization. Try a diacritic-stripped
	// comparison before rejecting: a single accent typo (e.g. "Thì" vs
	// "Thị") is data-entry noise, not a real CCCD mismatch. Allow the
	// import and log a warning so the partner can correct the typo.
	if bccNormNameLoose(emp.FullName) == bccNormNameLoose(stkName) {
		slog.Warn("BCCImport: STK/BCC name differs only by diacritic, allowing import",
			"cccd", emp.CCCD, "bcc_name", emp.FullName, "stk_name", stkName)
		return nil
	}
	cleanBCCName := strings.TrimSpace(emp.FullName)
	cleanSTKName := strings.TrimSpace(stkName)
	return &domain.ImportError{
		Employee: cleanBCCName,
		Reason: fmt.Sprintf("CCCD %s thuộc về %s (theo STK), không phải %s — có thể sai CCCD",
			emp.CCCD, cleanSTKName, cleanBCCName),
	}
}
