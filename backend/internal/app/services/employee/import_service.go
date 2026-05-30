package employee

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	bankpkg "api-server/internal/pkg/bank"
	"api-server/internal/pkg/utils"
)

// ImportService handles employee import operations
type ImportService struct {
	employeeRepo        domain.EmployeeRepository
	projectRepo         domain.ProjectRepository
	projectEmployeeRepo domain.ProjectEmployeeRepository
	bankRepo            domain.BankRepository
	userService         *EmployeeUserService
	progressService     *infrastructure.EmployeeImportProgressService
	logger              *slog.Logger
}

// NewImportService creates a new employee import service
func NewImportService(
	employeeRepo domain.EmployeeRepository,
	projectRepo domain.ProjectRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	bankRepo domain.BankRepository,
	userService *EmployeeUserService,
	progressService *infrastructure.EmployeeImportProgressService,
) *ImportService {
	return &ImportService{
		employeeRepo:        employeeRepo,
		projectRepo:         projectRepo,
		projectEmployeeRepo: projectEmployeeRepo,
		bankRepo:            bankRepo,
		userService:         userService,
		progressService:     progressService,
		logger:              slog.Default().With("component", "EmployeeImportService"),
	}
}

// ParseEmployeeExcel parses an employee import Excel file and returns the rows and total count
func (s *ImportService) ParseEmployeeExcel(ctx context.Context, file *excelize.File) ([]dto.EmployeeImportRow, int, error) {
	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, 0, fmt.Errorf("no sheets found in Excel file")
	}

	rows, err := file.GetRows(sheets[0])
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read Excel file: %w", err)
	}

	if len(rows) < 2 {
		return nil, 0, fmt.Errorf("excel file is empty or has no data rows")
	}

	var employeeRows []dto.EmployeeImportRow
	rowNumber := 0

	// Start from row 2 (skip header)
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		rowNumber = i + 1

		// Skip empty rows
		if len(row) < 2 || row[0] == "" {
			continue
		}

		importRow, err := s.parseRow(row, rowNumber)
		if err != nil {
			s.logger.Warn("failed to parse row", "row", rowNumber, "error", err)
			continue
		}

		// Validate required fields
		if importRow.CCCD == "" || importRow.Fullname == "" {
			s.logger.Warn("skipping row with missing required fields", "row", rowNumber, "cccd", importRow.CCCD, "fullname", importRow.Fullname)
			continue
		}

		employeeRows = append(employeeRows, importRow)
	}

	return employeeRows, len(employeeRows), nil
}

// parseDateOfBirth parses a date string and returns a time.Time pointer
// Supports formats: "2006-01-02", "02/01/2006", "2006/01/02"
func parseDateOfBirth(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}

	// Try different date formats
	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"2006/01/02",
		"02-01-2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t
		}
	}

	// If all formats fail, return nil
	return nil
}

// parseRow parses a single Excel row into an EmployeeImportRow
// Excel format: B=CCCD, C=Project, D=Position, E=Mobile, F=Fullname, G=BankAcctName, H=BankAcctNum, I=BankName
func (s *ImportService) parseRow(row []string, rowNumber int) (dto.EmployeeImportRow, error) {
	importRow := dto.EmployeeImportRow{
		RowNumber: rowNumber,
	}

	// Excel columns (0-indexed, where 0=Column A):
	// 1: CCCD (B) | 2: Project Code (C) | 3: Position (D) | 4: Mobile (E)
	// 5: Fullname (F) | 6: Bank Account Name (G) | 7: Bank Account Number (H) | 8: Bank Name (I)

	if len(row) > 1 {
		importRow.CCCD = strings.TrimSpace(row[1])
	}
	if len(row) > 2 {
		importRow.ProjectCode = strings.TrimSpace(row[2])
	}
	if len(row) > 3 {
		importRow.Position = strings.TrimSpace(row[3])
	}
	if len(row) > 4 {
		mobile := strings.TrimSpace(row[4])
		if len(mobile) > 15 {
			mobile = mobile[:15]
		}
		importRow.Mobile = mobile
	}
	if len(row) > 5 {
		importRow.Fullname = strings.TrimSpace(row[5])
	}
	if len(row) > 6 {
		importRow.BankAccountName = strings.TrimSpace(row[6])
	}
	if len(row) > 7 {
		importRow.BankAccount = strings.TrimSpace(row[7])
	}
	if len(row) > 8 {
		importRow.BankName = strings.TrimSpace(row[8])
	}

	// Default position if not provided
	if importRow.Position == "" {
		importRow.Position = "phổ thông"
	}

	return importRow, nil
}

// ProcessImportChunk processes a chunk of employee import rows
// Returns: createdCount, updatedCount, errorCount, errors
func (s *ImportService) ProcessImportChunk(
	ctx context.Context,
	importID string,
	rows []dto.EmployeeImportRow,
	createdBy uint,
	startRowNumber int,
) (int, int, int, []dto.RowError, error) {
	createdCount := 0
	updatedCount := 0
	errorCount := 0
	var errors []dto.RowError

	// First, collect all unique project codes and batch get/create them
	projectMap := make(map[string]*domain.Project)
	for _, row := range rows {
		if row.ProjectCode != "" && projectMap[row.ProjectCode] == nil {
			project, err := s.getOrCreateProject(ctx, row.ProjectCode, createdBy)
			if err != nil {
				s.logger.Warn("failed to get/create project", "code", row.ProjectCode, "error", err)
				errors = append(errors, dto.RowError{
					RowNumber: row.RowNumber,
					CCCD:      row.CCCD,
					Message:   fmt.Sprintf("Lỗi tạo dự án: %v", err),
				})
				errorCount++
				continue
			}
			projectMap[row.ProjectCode] = project
		}
	}

	// Process each row
	for _, row := range rows {
		employeeCreated, employeeUpdated, rowErrors := s.processRow(
			ctx,
			row,
			projectMap[row.ProjectCode],
			createdBy,
		)

		if employeeCreated {
			createdCount++
		} else if employeeUpdated {
			updatedCount++
		}

		if len(rowErrors) > 0 {
			errors = append(errors, rowErrors...)
			errorCount++
		}
	}

	return createdCount, updatedCount, errorCount, errors, nil
}

// processRow processes a single employee import row
func (s *ImportService) processRow(
	ctx context.Context,
	row dto.EmployeeImportRow,
	project *domain.Project,
	createdBy uint,
) (bool, bool, []dto.RowError) {
	var rowErrors []dto.RowError

	// Get or create employee
	employee, employeeCreated, employeeUpdated, err := s.getOrCreateEmployee(ctx, row, createdBy)
	if err != nil {
		rowErrors = append(rowErrors, dto.RowError{
			RowNumber: row.RowNumber,
			CCCD:      row.CCCD,
			Message:   fmt.Sprintf("Lỗi tạo nhân viên: %v", err),
		})
		return false, false, rowErrors
	}

	// If project is specified, create assignment
	if project != nil {
		_, err := s.getOrCreateAssignment(ctx, project.ID, employee.ID, row, createdBy)
		if err != nil {
			rowErrors = append(rowErrors, dto.RowError{
				RowNumber: row.RowNumber,
				CCCD:      row.CCCD,
				Message:   fmt.Sprintf("Lỗi tạo phân công: %v", err),
			})
			return employeeCreated, employeeUpdated, rowErrors
		}
	}

	return employeeCreated, employeeUpdated, rowErrors
}

// getOrCreateProject gets an existing project or creates a new one
func (s *ImportService) getOrCreateProject(ctx context.Context, code string, createdBy uint) (*domain.Project, error) {
	project, err := s.projectRepo.GetByCode(ctx, code)
	if err == nil {
		return project, nil
	}

	if !domain.IsNotFoundError(err) {
		return nil, err
	}

	// Create new project
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

	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, err
	}

	s.logger.Info("created project from import", "code", code, "project_id", project.ID)
	return project, nil
}

// getOrCreateEmployee gets an existing employee or creates a new one
func (s *ImportService) getOrCreateEmployee(ctx context.Context, row dto.EmployeeImportRow, createdBy uint) (*domain.Employee, bool, bool, error) {
	employee, err := s.employeeRepo.GetByCCCD(ctx, row.CCCD)
	if err == nil {
		// Employee exists, check if update is needed
		needsUpdate := s.shouldUpdateEmployee(employee, row)

		if needsUpdate {
			s.updateEmployeeFields(employee, row)
			if err := s.employeeRepo.Update(ctx, employee); err != nil {
				s.logger.Warn("failed to update employee", "employee_id", employee.ID, "error", err)
			} else {
				s.logger.Info("updated employee from import", "employee_id", employee.ID, "cccd", row.CCCD)
				return employee, false, true, nil
			}
		}

		// Create user account if employee doesn't have one
		if employee.UserID == nil && s.userService != nil {
			s.createUserAccountForEmployee(ctx, employee)
		}

		return employee, false, false, nil
	}

	// Create new employee
	bankID := s.resolveBankID(ctx, row.BankName)
	var dateOfBirth *time.Time
	if row.DateOfBirth != nil {
		dateOfBirth = parseDateOfBirth(*row.DateOfBirth)
	}

	employee = &domain.Employee{
		Fullname:          row.Fullname,
		CCCD:              row.CCCD,
		BankAccountNumber: row.BankAccount,
		BankAccountName:   row.BankAccountName,
		BankID:            bankID,
		Mobile:            row.Mobile,
		Address:           row.Address,
		Email:             row.Email,
		DateOfBirth:       dateOfBirth,
		CreatedBy:         createdBy,
	}

	if err := s.employeeRepo.Create(ctx, employee); err != nil {
		return nil, false, false, err
	}

	s.logger.Info("created employee from import", "employee_id", employee.ID, "cccd", row.CCCD)

	// Create user account
	if s.userService != nil {
		s.createUserAccountForEmployee(ctx, employee)
	}

	return employee, true, false, nil
}

// shouldUpdateEmployee checks if an employee needs to be updated based on import data
func (s *ImportService) shouldUpdateEmployee(employee *domain.Employee, row dto.EmployeeImportRow) bool {
	if row.Fullname != "" && employee.Fullname != row.Fullname {
		return true
	}
	if row.Email != nil && (employee.Email == nil || *employee.Email != *row.Email) {
		return true
	}
	if row.Mobile != "" && employee.Mobile != row.Mobile {
		return true
	}
	if row.Address != "" && employee.Address != row.Address {
		return true
	}
	if row.BankAccount != "" && employee.BankAccountNumber != row.BankAccount {
		return true
	}
	if row.BankAccountName != "" && employee.BankAccountName != row.BankAccountName {
		return true
	}
	// Check if bank name has changed
	if row.BankName != "" {
		bankID := s.resolveBankID(context.Background(), row.BankName)
		if employee.BankID == nil && bankID != nil {
			return true
		}
		if employee.BankID != nil && bankID == nil {
			return true
		}
		if employee.BankID != nil && bankID != nil && *employee.BankID != *bankID {
			return true
		}
	}
	if row.DateOfBirth != nil {
		rowDateOfBirth := parseDateOfBirth(*row.DateOfBirth)
		if employee.DateOfBirth == nil || (rowDateOfBirth != nil && *employee.DateOfBirth != *rowDateOfBirth) {
			return true
		}
	}
	return false
}

// updateEmployeeFields updates employee fields from import data
func (s *ImportService) updateEmployeeFields(employee *domain.Employee, row dto.EmployeeImportRow) {
	if row.Fullname != "" {
		employee.Fullname = row.Fullname
	}
	if row.Email != nil {
		employee.Email = row.Email
	}
	if row.Mobile != "" {
		employee.Mobile = row.Mobile
	}
	if row.Address != "" {
		employee.Address = row.Address
	}
	if row.BankAccount != "" {
		employee.BankAccountNumber = row.BankAccount
	}
	if row.BankAccountName != "" {
		employee.BankAccountName = row.BankAccountName
	}
	if row.BankName != "" {
		bankID := s.resolveBankID(context.Background(), row.BankName)
		employee.BankID = bankID
	}
	if row.DateOfBirth != nil {
		employee.DateOfBirth = parseDateOfBirth(*row.DateOfBirth)
	}
}

// createUserAccountForEmployee creates a user account for an employee
func (s *ImportService) createUserAccountForEmployee(ctx context.Context, employee *domain.Employee) {
	baseUsername := utils.GenerateUsername(employee.Fullname)
	if baseUsername == "" {
		s.logger.Warn("failed to generate username from name", "employee_id", employee.ID, "name", employee.Fullname)
		return
	}

	username := s.userService.EnsureUniqueUsername(ctx, baseUsername)
	userID, err := s.userService.CreateUserForEmployee(ctx, employee, username)
	if err != nil {
		s.logger.Error("failed to create user for employee", "employee_id", employee.ID, "username", username, "error", err)
		return
	}

	employee.UserID = &userID
	if err := s.employeeRepo.Update(ctx, employee); err != nil {
		s.logger.Error("failed to update employee with user_id", "employee_id", employee.ID, "user_id", userID, "error", err)
		return
	}

	s.logger.Info("created user account for employee", "employee_id", employee.ID, "username", username)
}

// getOrCreateAssignment gets an existing assignment or creates a new one
func (s *ImportService) getOrCreateAssignment(ctx context.Context, projectID, employeeID uint, row dto.EmployeeImportRow, createdBy uint) (bool, error) {
	existing, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(ctx, projectID, employeeID)
	if err == nil && existing != nil {
		return false, nil
	}

	now := clock.NowUTC()
	startDate := now

	// Parse start date if provided
	if row.StartDate != "" {
		parsedDate, err := time.Parse("2006-01-02", row.StartDate)
		if err == nil {
			startDate = parsedDate
		}
	}

	position := row.Position
	if position == "" {
		position = "phổ thông"
	}

	paymentSchedule := string(domain.PaymentScheduleFlexible)
	if row.PaymentSchedule != "" {
		paymentSchedule = row.PaymentSchedule
	}

	assignment := &domain.ProjectEmployee{
		ProjectID:       projectID,
		EmployeeID:      employeeID,
		EmployeeName:    row.Fullname,
		EmployeeCCCD:    row.CCCD,
		Position:        position,
		StartDate:       startDate,
		PaymentSchedule: paymentSchedule,
		CreatedBy:       createdBy,
	}

	if err := s.projectEmployeeRepo.Create(ctx, assignment); err != nil {
		return false, err
	}

	s.logger.Info("created assignment from import", "project_id", projectID, "employee_id", employeeID)
	return true, nil
}

// resolveBankID resolves a bank name to a bank ID
func (s *ImportService) resolveBankID(ctx context.Context, bankName string) *uint {
	if bankName == "" {
		return nil
	}

	// Map to exact branch_name using keyword-based matching
	// e.g., "Ngân hàng TMCP An Bình - ABBANK" -> "An Bình (ABBANK)"
	mappedName := bankpkg.MapName(bankName)
	if mappedName == "" {
		return nil
	}

	// Search using the mapped exact branch_name
	banks, err := s.bankRepo.SearchByBranchName(ctx, mappedName, 1)
	if err != nil {
		s.logger.Warn("failed to search bank", "name", mappedName, "error", err)
		return nil
	}

	if len(banks) == 0 {
		s.logger.Warn("bank not found", "search_name", mappedName, "original_name", bankName)
		return nil
	}

	id := banks[0].ID
	return &id
}

// GetProgress retrieves the current import progress
func (s *ImportService) GetProgress(ctx context.Context, importID string) (*dto.EmployeeImportStatus, error) {
	return s.progressService.GetImportStatus(ctx, importID)
}
