package advance_payment

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/pkg/errors"
	"github.com/xuri/excelize/v2"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
)

// dedupeAdvancePayments collapses multiple rows for the same (project, employee)
// down to a single record, keeping the LAST value seen. The flexible payroll
// template stores the freshest amounts in the trailing sheet (e.g. "UL (2)"
// supersedes "UL" within the same workbook), and within a sheet later rows
// supersede earlier ones. We collapse to one contribution per employee per
// upload event so that one upload = one top-up; if the workbook listed the
// same employee twice with different amounts we'd otherwise stack the file's
// own duplicates onto the employee's running ceiling.
func dedupeAdvancePayments(items []*domain.AdvancePayment) []*domain.AdvancePayment {
	if len(items) == 0 {
		return items
	}
	index := make(map[string]int, len(items))
	deduped := make([]*domain.AdvancePayment, 0, len(items))
	for _, ap := range items {
		key := fmt.Sprintf("%d:%d", ap.ProjectID, ap.EmployeeID)
		if pos, ok := index[key]; ok {
			deduped[pos] = ap
			continue
		}
		index[key] = len(deduped)
		deduped = append(deduped, ap)
	}
	return deduped
}

// detectColumnOffset examines the first few data rows to determine the column
// layout. Old format has no branch column (phone at index 4); new format has
// a branch/chi nhánh column at index 4 (non-numeric text like "Hải Phòng")
// which shifts all subsequent columns by 1.
//
// Returns 0 for old format, 1 for new format (with branch column).
func detectColumnOffset(rows [][]string) int {
	checkCount := 3
	if len(rows)-1 < checkCount {
		checkCount = len(rows) - 1
	}
	if checkCount <= 0 {
		return 0
	}

	nonDigitCount := 0
	for i := 1; i <= checkCount && i < len(rows); i++ {
		row := rows[i]
		if len(row) <= 4 {
			continue
		}
		col4 := strings.TrimSpace(row[4])
		if col4 == "" {
			continue
		}
		// If column E contains mostly non-digit chars, it's a branch name
		digits := 0
		for _, r := range col4 {
			if unicode.IsDigit(r) {
				digits++
			}
		}
		if digits < len([]rune(col4))/2 {
			nonDigitCount++
		}
	}

	if nonDigitCount > 0 {
		return 1
	}
	return 0
}

// ImportFlexPayFile applies one upload of the flexible payroll template.
// One file = one round. Multiple rounds per period live as separate rows
// keyed on (employee, project, for_month, upload_date) where upload_date
// is the calendar DAY the file was uploaded — e.g. a late-April upload
// gets upload_date='2026-04-28', an early-May upload gets '2026-05-02'.
// The rounds sum at read time via SumMaxAdvByEmployeeMonth.
//
// The flexpay workbook has a "UL" sheet with the actual data for this
// upload, plus auxiliary sheets ("ds" = employee directory, "UL (2)" =
// historical reference copy from a prior round). Only "UL" feeds the
// import — the others are ignored.
func (s *Service) ImportFlexPayFile(ctx context.Context, file *excelize.File, forMonth string, assetID uint, createdBy uint, uploadedAt time.Time, progressCB ProgressCallback) (*dto.ImportFlexPayFileResult, error) {
	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("no sheets found in Excel file")
	}

	result := &dto.ImportFlexPayFileResult{
		ForMonth: forMonth,
	}

	// upload_date is the CALENDAR DAY of the upload, not the period the
	// data applies to. That's what makes two rounds for the same for_month
	// (even within the same calendar month, on different days) land in
	// different rows under the (employee, project, for_month, upload_date)
	// unique key — collapsing this to forMonth (as f3e8f33b did) was the
	// regression that caused the ghost-duplicates bug the user originally
	// reported.
	uploadDate := resolveUploadDate(uploadedAt)

	// Detect importable sheets by header structure. Any sheet whose column B
	// header contains "Mã nhân viên" is an importable data sheet — this covers
	// "UL", date-pattern sheets like "26.5", and any future naming convention.
	// Skip auxiliary sheets: "ds" (passwords, different columns), "UL (2)"
	// (prior round snapshot with different header).
	rowsBySheet := make(map[string][][]string, len(sheets))
	importSheets := make([]string, 0, len(sheets))
	for _, sheetName := range sheets {
		rows, err := file.GetRows(sheetName)
		if err != nil || len(rows) == 0 || len(rows[0]) < 2 {
			continue
		}
		colB := strings.ToLower(strings.TrimSpace(rows[0][1]))
		if strings.Contains(colB, "mã nhân viên") {
			importSheets = append(importSheets, sheetName)
			rowsBySheet[sheetName] = rows
		}
	}
	if len(importSheets) == 0 {
		return nil, errors.New("flexpay workbook has no importable sheets (expected column B header containing \"Mã nhân viên\")")
	}

	// Count total rows across the import sheets for progress tracking
	totalRows := 0
	for _, sheetName := range importSheets {
		for i := 1; i < len(rowsBySheet[sheetName]); i++ {
			row := rowsBySheet[sheetName][i]
			if len(row) > 1 && strings.TrimSpace(row[1]) != "" {
				totalRows++
			}
		}
	}

	if progressCB != nil {
		progressCB(totalRows, 0)
	}

	processedRows := 0
	var advancePayments []*domain.AdvancePayment
	// Collect eligible employees for salary ZNS notifications. The last row for
	// an assignment wins, matching the advance-payment import dedupe rule.
	var employeeZNSData []dto.EmployeeZNSData
	employeeZNSIndex := make(map[string]int)

	for _, sheetName := range importSheets {
		rows := rowsBySheet[sheetName]

		// Detect column layout by sniffing first data rows.
		// Old: E(4)=Phone(digits), ..., I(8)=Bank, J(9)=Amount
		// New: E(4)=Branch(text), F(5)=Phone, ..., J(9)=Bank, K(10)=Amount
		offset := detectColumnOffset(rows)
		s.logger.Info("detected column format", "sheet", sheetName, "has_branch_column", offset == 1)

		for i := 1; i < len(rows); i++ {
			row := rows[i]

			if len(row) < 2 {
				continue
			}

			cccd := strings.TrimSpace(row[1])
			if cccd == "" {
				break // stop at empty row
			}

			result.TotalRows++

			minCols := 9 + offset
			if len(row) < minCols {
				s.logger.Warn("row has insufficient columns", "sheet", sheetName, "row", i+1)
				processedRows++
				if progressCB != nil {
					progressCB(totalRows, processedRows)
				}
				continue
			}

			projectCode := strings.TrimSpace(row[2])
			position := strings.TrimSpace(row[3])
			// row[4] is phone (old, offset=0) or branch (new, offset=1)
			mobile := strings.TrimSpace(row[4+offset])
			fullName := strings.TrimSpace(row[5+offset])
			bankAccountName := strings.TrimSpace(row[6+offset])
			bankAccountNumber := strings.TrimSpace(row[7+offset])
			bankName := strings.TrimSpace(row[8+offset])

			amountIdx := 9 + offset
			var hanMuc uint64
			if len(row) > amountIdx {
				hanMucStr := strings.TrimSpace(row[amountIdx])
				if hanMucStr != "" && hanMucStr != "0" {
					parsed, err := parseCommaNumber(hanMucStr)
					if err == nil {
						hanMuc = uint64(parsed)
					}
				}
			}

			if position == "" {
				position = "phổ thông"
			}

			if projectCode == "" || fullName == "" {
				processedRows++
				if progressCB != nil {
					progressCB(totalRows, processedRows)
				}
				continue
			}

			project, projectCreated, err := s.getOrCreateProject(ctx, projectCode, createdBy)
			if err != nil {
				s.logger.Warn("failed to get/create project", "code", projectCode, "error", err)
				processedRows++
				if progressCB != nil {
					progressCB(totalRows, processedRows)
				}
				continue
			}
			if projectCreated {
				result.ProjectsCreated++
			} else {
				result.ProjectsSkipped++
			}

			employee, employeeCreated, _, err := s.getOrCreateEmployee(ctx, cccd, fullName, bankAccountNumber, bankAccountName, bankName, mobile, createdBy)
			if err != nil {
				s.logger.Warn("failed to get/create employee", "cccd", cccd, "error", err)
				processedRows++
				if progressCB != nil {
					progressCB(totalRows, processedRows)
				}
				continue
			}
			if employeeCreated {
				result.EmployeesCreated++
			} else {
				result.EmployeesSkipped++
			}

			assignment, assignmentCreated, err := s.getOrCreateAssignment(ctx, project.ID, employee.ID, cccd, fullName, position, createdBy)
			if err != nil {
				s.logger.Warn("failed to get/create assignment", "project_id", project.ID, "employee_id", employee.ID, "error", err)
				processedRows++
				if progressCB != nil {
					progressCB(totalRows, processedRows)
				}
				continue
			}
			if assignmentCreated {
				result.AssignmentsCreated++
			} else {
				result.AssignmentsSkipped++
			}

			if shouldNotifyFlexPayZNS(hanMuc, mobile, assignment) {
				// The card's "Hạn cuối" must match the ENFORCED request window:
				// requests for payroll month M are accepted through
				// RequestCutoffDay of month M+1 (inclusive) — not the end of
				// month M. Invalid months cannot be made requestable, so they
				// must not produce a notification.
				expiryDate, err := time.Parse("2006-01", forMonth)
				if err == nil {
					key := fmt.Sprintf("%d:%d", project.ID, employee.ID)
					notification := dto.EmployeeZNSData{
						ProjectID:    project.ID,
						EmployeeID:   employee.ID,
						EmployeeName: fullName,
						Mobile:       mobile,
						Amount:       int64(hanMuc),
						ExpiryDate:   expiryDate.AddDate(0, 1, clock.RequestCutoffDay-1),
					}
					if pos, exists := employeeZNSIndex[key]; exists {
						employeeZNSData[pos] = notification
					} else {
						employeeZNSIndex[key] = len(employeeZNSData)
						employeeZNSData = append(employeeZNSData, notification)
					}
				}
			}

			// Only create advance payment record if amount column is provided and > 0
			if hanMuc > 0 {
				// Self-check-in employees earn their advance quota from check-in/out
				// (advance_payments.salary → configured max_adv_amount). Exclude them from
				// the admin FlexPay import so BatchUpsert never overwrites their
				// salary-derived quota. This is a SEPARATE flow (AC5).
				if assignment.CheckInEnabled {
					s.logger.Info("skip flexpay advance import for check-in-enabled employee",
						"project_id", project.ID, "employee_id", employee.ID, "for_month", forMonth)
				} else {
					assetIDCopy := assetID
					ap := &domain.AdvancePayment{
						ProjectID:          project.ID,
						EmployeeID:         employee.ID,
						ForMonth:           forMonth,
						UploadDate:         uploadDate,
						MaxAdvAmount:       hanMuc,
						LastAppliedAssetID: &assetIDCopy,
					}
					advancePayments = append(advancePayments, ap)
				}
			}

			processedRows++
			if progressCB != nil {
				progressCB(totalRows, processedRows)
			}
		}
	}

	if len(advancePayments) > 0 {
		// Per-row idempotency lives in the SQL: BatchUpsert stamps
		// last_applied_asset_id on every row it touches, and the
		// ON DUPLICATE KEY UPDATE clause skips the salary / max_adv_amount
		// addition when the incoming asset matches the row's last applied
		// one. So a re-enqueued duplicate upload reaches BatchUpsert and
		// the SQL turns it into a no-op per row, with row-level locking
		// keeping concurrent runs safe.
		deduped := dedupeAdvancePayments(advancePayments)

		if err := s.config.AdvancePaymentRepo.BatchUpsert(ctx, deduped); err != nil {
			return nil, errors.Wrap(err, "failed to top up advance payments")
		}

		result.AdvancePaymentsCreated = len(deduped)
		result.AdvancePaymentsSkipped = len(advancePayments) - len(deduped)

		s.logger.Info("topped up advance payments from flex pay import",
			"asset_id", assetID,
			"for_month", forMonth,
			"rows_in_file", len(advancePayments),
			"unique_employees", len(deduped))
	}

	s.logger.Info("imported flexible payroll file",
		"for_month", forMonth,
		"total_rows", result.TotalRows,
		"employees_created", result.EmployeesCreated,
		"projects_created", result.ProjectsCreated,
		"assignments_created", result.AssignmentsCreated,
	)

	if s.config.CacheService != nil {
		if err := s.config.CacheService.DeletePattern(ctx, "dashboard:employees_summary*"); err != nil {
			s.logger.Warn("failed to invalidate employee summary cache", "error", err)
		}
		if err := s.config.CacheService.DeletePattern(ctx, "dashboard:employees_summary_creator*"); err != nil {
			s.logger.Warn("failed to invalidate employee summary creator cache", "error", err)
		}
	}

	// Attach employee ZNS data to result for notification sending
	result.EmployeeZNSData = employeeZNSData
	s.logger.Info("collected employee data for ZNS", "count", len(employeeZNSData))

	return result, nil
}

// shouldNotifyFlexPayZNS keeps the notification recipient cohort identical to
// the import path that creates requestable FlexPay quota. Self-check-in
// employees earn quota from attendance and must not receive this upload notice.
func shouldNotifyFlexPayZNS(amount uint64, mobile string, assignment *domain.ProjectEmployee) bool {
	return amount > 0 && mobile != "" && assignment != nil &&
		assignment.PaymentSchedule == string(domain.PaymentScheduleFlexible) &&
		!assignment.CheckInEnabled
}

func resolveUploadDate(uploadedAt time.Time) string {
	if uploadedAt.IsZero() {
		return clock.Now().Format("2006-01-02")
	}
	return uploadedAt.In(clock.DefaultLocation).Format("2006-01-02")
}

// getOrCreateProject mirrors employee.ImportService.getOrCreateProject (the
// only difference is the created-flag return); the two deliberately stay
// separate — the employee variant's signature and logging are load-bearing
// for its audit trail. The getOrCreateEmployee/getOrCreateAssignment pairs
// below have genuinely diverged (bank validation, StartDate/PaymentSchedule
// parsing live only in the employee variant) and must not be merged.
func (s *Service) getOrCreateProject(ctx context.Context, code string, createdBy uint) (*domain.Project, bool, error) {
	project, err := s.config.ProjectRepo.GetByCode(ctx, code)
	if err == nil {
		return project, false, nil
	}

	if !domain.IsNotFoundError(err) {
		return nil, false, err
	}

	now := clock.NowUTC()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	project = &domain.Project{
		Code:          code,
		Name:          code,
		ClientName:    code,
		ProjectStatus: domain.ProjectStatusRunning,
		StartDate:     &startDate,
		CreatedBy:     createdBy,
	}

	if err := s.config.ProjectRepo.Create(ctx, project); err != nil {
		return nil, false, err
	}

	return project, true, nil
}

func (s *Service) getOrCreateEmployee(ctx context.Context, cccd, name, accountNumber, accountName, bankName, mobile string, createdBy uint) (*domain.Employee, bool, bool, error) {
	employee, err := s.config.EmployeeService.GetEmployeeByCCCD(ctx, cccd)
	if err == nil {
		needsUpdate := false

		// Update fullname if different
		if name != "" && employee.Fullname != name {
			employee.Fullname = name
			needsUpdate = true
		}

		// Update bank info if account number is different or provided
		if accountNumber != "" {
			employee.BankAccountNumber = accountNumber
			employee.BankAccountName = accountName
			bankID := s.resolveBankID(ctx, bankName)
			if bankID != nil {
				employee.BankID = bankID
			}
			needsUpdate = true
		}

		// Update mobile if different
		if mobile != "" && employee.Mobile != mobile {
			employee.Mobile = mobile
			needsUpdate = true
		}

		if needsUpdate {
			if err := s.config.EmployeeRepo.Update(ctx, employee); err != nil {
				return nil, false, false, fmt.Errorf("failed to update employee info: %w", err)
			}
		}

		if employee.UserID == nil {
			if err := s.config.EmployeeService.EnsureEmployeeUserAccount(ctx, employee.ID); err != nil {
				return nil, false, false, fmt.Errorf("failed to create employee user account: %w", err)
			}
		}

		return employee, false, needsUpdate, nil
	}

	bankID := s.resolveBankID(ctx, bankName)
	employee = &domain.Employee{
		Fullname:          name,
		CCCD:              cccd,
		BankAccountNumber: accountNumber,
		BankAccountName:   accountName,
		BankID:            bankID,
		Mobile:            mobile,
		CreatedBy:         createdBy,
	}

	createdEmployee, err := s.config.EmployeeService.CreateEmployeeFromImport(ctx, employee, createdBy)
	if err != nil {
		return nil, false, false, err
	}

	return createdEmployee, true, false, nil
}

func (s *Service) resolveBankID(ctx context.Context, bankName string) *uint {
	return s.config.EmployeeService.ResolveBankID(ctx, bankName)
}

func parseCommaNumber(s string) (uint64, error) {
	cleaned := strings.ReplaceAll(s, ",", "")
	cleaned = strings.TrimSpace(cleaned)

	var result uint64
	_, err := fmt.Sscanf(cleaned, "%d", &result)
	return result, err
}

// getOrCreateAssignment returns the existing active assignment (creating one if
// absent), so callers can read fields like CheckInEnabled without a second
// lookup. The bool is true when a new assignment was created.
func (s *Service) getOrCreateAssignment(ctx context.Context, projectID, employeeID uint, employeeCCCD, employeeName, position string, createdBy uint) (*domain.ProjectEmployee, bool, error) {
	existing, err := s.config.ProjectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(ctx, projectID, employeeID)
	if err == nil && existing != nil {
		return existing, false, nil
	}

	if position == "" {
		position = "phổ thông"
	}

	now := clock.NowUTC()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	assignment := &domain.ProjectEmployee{
		ProjectID:       projectID,
		EmployeeID:      employeeID,
		EmployeeName:    employeeName,
		EmployeeCCCD:    employeeCCCD,
		Position:        position,
		StartDate:       startDate,
		PaymentSchedule: string(domain.PaymentScheduleFlexible),
		// Explicit so the returned struct matches the DB default and a later
		// full-row Save cannot persist the zero value as paused.
		AdvanceRequestEnabled: true,
		CreatedBy:             createdBy,
	}

	if err := s.config.ProjectEmployeeRepo.Create(ctx, assignment); err != nil {
		return nil, false, err
	}

	return assignment, true, nil
}
