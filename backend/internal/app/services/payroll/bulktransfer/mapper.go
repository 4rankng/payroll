package bulktransfer

import (
	"context"
	"fmt"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
)

// MappedRow represents a successfully mapped row with all resolved entities
type MappedRow struct {
	Detail     dto.BulkTransferResultItemDetail
	Employee   *domain.Employee
	Timesheets []*domain.Timesheet
	ProjectID  uint
}

// BulkTransferMapper maps parsed rows to domain entities
type BulkTransferMapper struct {
	employeeRepo  EmployeeRepository
	timesheetRepo TimesheetRepository
	rowParser     *RowParser
}

// NewBulkTransferMapper creates a new BulkTransferMapper instance
func NewBulkTransferMapper(
	employeeRepo EmployeeRepository,
	timesheetRepo TimesheetRepository,
	rowParser *RowParser,
) *BulkTransferMapper {
	return &BulkTransferMapper{
		employeeRepo:  employeeRepo,
		timesheetRepo: timesheetRepo,
		rowParser:     rowParser,
	}
}

// MapRow maps a parsed row to domain entities
func (m *BulkTransferMapper) MapRow(
	ctx context.Context,
	parsed ParsedRow,
	rowNum int,
) (*MappedRow, error) {
	detail := dto.BulkTransferResultItemDetail{
		Row:                   rowNum,
		Status:                ProcessingStatusSuccess,
		EmployeeAccountNumber: parsed.AccountNumber,
		EmployeeAccountName:   parsed.AccountName,
		Amount:                parsed.Amount,
		PaymentDescription:    parsed.Description,
		TransferStatus:        parsed.TransferStatus,
		TrackingData:          parsed.TrackingData,
	}

	// Find employee
	employee, err := m.findEmployee(ctx, &detail)
	if err != nil {
		return nil, err
	}
	if employee == nil {
		// Employee not found, detail already updated with error
		return &MappedRow{Detail: detail}, nil
	}

	detail.EmployeeID = &employee.ID
	detail.EmployeeFullname = employee.FormattedFullname()

	// Match timesheets
	timesheets, projectID, err := m.matchTimesheets(ctx, &detail, employee, parsed)
	if err != nil {
		return nil, err
	}
	if timesheets == nil {
		// No timesheets found, detail already updated with warning
		return &MappedRow{Detail: detail, Employee: employee}, nil
	}

	// Validate timesheets
	if err := m.validateTimesheets(timesheets, employee, projectID, &detail); err != nil {
		// Validation failed, detail already updated with error
		return &MappedRow{Detail: detail, Employee: employee}, nil
	}

	return &MappedRow{
		Detail:     detail,
		Employee:   employee,
		Timesheets: timesheets,
		ProjectID:  projectID,
	}, nil
}

// findEmployee finds an employee by account number
func (m *BulkTransferMapper) findEmployee(
	ctx context.Context,
	detail *dto.BulkTransferResultItemDetail,
) (*domain.Employee, error) {
	employee, err := m.employeeRepo.GetByBankAccountNumber(ctx, detail.EmployeeAccountNumber)
	if err != nil {
		if domain.IsNotFoundError(err) {
			detail.Status = ProcessingStatusFailed
			detail.Message = "Employee not found with this account number"
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find employee: %w", err)
	}

	return employee, nil
}

// matchTimesheets finds matching timesheets for the employee
func (m *BulkTransferMapper) matchTimesheets(
	ctx context.Context,
	detail *dto.BulkTransferResultItemDetail,
	employee *domain.Employee,
	parsed ParsedRow,
) ([]*domain.Timesheet, uint, error) {
	// Try parsing tracking data first
	parsedProjectID, timesheetIDs, err := m.rowParser.ParseTrackingData(parsed.TrackingData)
	if err == nil && len(timesheetIDs) > 0 {
		return m.matchByTrackingIDs(ctx, detail, employee, parsedProjectID, timesheetIDs)
	}

	// Fallback to description parsing
	return m.matchByDescription(ctx, detail, employee)
}

// matchByTrackingIDs matches timesheets using tracking IDs
func (m *BulkTransferMapper) matchByTrackingIDs(
	ctx context.Context,
	detail *dto.BulkTransferResultItemDetail,
	employee *domain.Employee,
	projectID uint,
	timesheetIDs []uint,
) ([]*domain.Timesheet, uint, error) {
	timesheets, err := m.timesheetRepo.GetByIDs(ctx, timesheetIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get timesheets by IDs: %w", err)
	}

	if len(timesheets) == 0 {
		detail.Status = ProcessingStatusWarning
		detail.Message = "No timesheets found with provided tracking IDs"
		detail.TimesheetsUpdated = 0
		return nil, 0, nil
	}

	return timesheets, projectID, nil
}

// matchByDescription matches timesheets using payment description
func (m *BulkTransferMapper) matchByDescription(
	ctx context.Context,
	detail *dto.BulkTransferResultItemDetail,
	employee *domain.Employee,
) ([]*domain.Timesheet, uint, error) {
	_, parsedProjectID, fromDate, toDate, err := m.rowParser.ParsePaymentPeriodWithProject(detail.PaymentDescription)
	if err != nil {
		detail.Status = ProcessingStatusFailed
		detail.Message = fmt.Sprintf("Invalid payment description and tracking data format: %v", err)
		return nil, 0, nil
	}

	timesheets, err := m.timesheetRepo.GetByProjectAndEmployee(ctx, parsedProjectID, employee.ID, fromDate, toDate)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get employee timesheets: %w", err)
	}

	if len(timesheets) == 0 {
		detail.Status = ProcessingStatusWarning
		detail.Message = "No timesheets found for this period"
		detail.TimesheetsUpdated = 0
		return nil, 0, nil
	}

	return timesheets, parsedProjectID, nil
}

// validateTimesheets validates that all timesheets meet the criteria
func (m *BulkTransferMapper) validateTimesheets(
	timesheets []*domain.Timesheet,
	employee *domain.Employee,
	projectID uint,
	detail *dto.BulkTransferResultItemDetail,
) error {
	for _, ts := range timesheets {
		if ts.EmployeeID != employee.ID {
			detail.Status = ProcessingStatusFailed
			detail.Message = fmt.Sprintf("Timesheet %d does not belong to employee %s", ts.ID, employee.FormattedFullname())
			return fmt.Errorf("timesheet validation failed")
		}
		if ts.ProjectID != projectID {
			detail.Status = ProcessingStatusFailed
			detail.Message = fmt.Sprintf("Timesheet %d does not belong to project %d", ts.ID, projectID)
			return fmt.Errorf("timesheet validation failed")
		}
		if ts.Status != domain.TimesheetStatusApproved {
			detail.Status = ProcessingStatusFailed
			detail.Message = fmt.Sprintf("Timesheet %d is not approved (status: %s)", ts.ID, ts.Status)
			return fmt.Errorf("timesheet validation failed")
		}
		if ts.PaymentStatus != domain.PaymentStatusPending {
			detail.Status = ProcessingStatusWarning
			detail.Message = fmt.Sprintf("Timesheet %d already has payment status: %s", ts.ID, ts.PaymentStatus)
			return fmt.Errorf("timesheet validation failed")
		}
	}

	return nil
}

// BatchMapRows maps multiple rows concurrently with a worker pool
func (m *BulkTransferMapper) BatchMapRows(
	ctx context.Context,
	rows [][]string,
	parser BankResultParser,
	maxWorkers int,
) ([]*MappedRow, error) {
	startRow := parser.StartRow()
	dataRows := rows[startRow:]

	type mapResult struct {
		index int
		row   *MappedRow
		err   error
	}

	results := make(chan mapResult, len(dataRows))
	semaphore := make(chan struct{}, maxWorkers)

	// Process rows concurrently
	var launchedCount int
	for i, rawRow := range dataRows {
		// Check for empty row
		rowParser := NewRowParser()
		if rowParser.IsEmptyRow(rawRow) {
			// Stop processing at first empty row
			break
		}

		semaphore <- struct{}{} // Acquire semaphore
		launchedCount++
		go func(idx int, raw []string) {
			defer func() { <-semaphore }() // Release semaphore

			rowNum := startRow + idx + 1
			parsed := parser.ParseRow(raw)

			// Validate row
			if err := parser.ValidateRow(parsed); err != nil {
				detail := dto.BulkTransferResultItemDetail{
					Row:                   rowNum,
					Status:                ProcessingStatusFailed,
					Message:               err.Error(),
					EmployeeAccountNumber: parsed.AccountNumber,
					EmployeeAccountName:   parsed.AccountName,
					Amount:                parsed.Amount,
					PaymentDescription:    parsed.Description,
					TransferStatus:        parsed.TransferStatus,
					TrackingData:          parsed.TrackingData,
				}
				results <- mapResult{index: idx, row: &MappedRow{Detail: detail}}
				return
			}

			// Map row
			mapped, err := m.MapRow(ctx, parsed, rowNum)
			results <- mapResult{index: idx, row: mapped, err: err}
		}(i, rawRow)
	}

	// Collect results - only read as many as we launched
	mappedRows := make([]*MappedRow, len(dataRows))
	var collectErr error
	for i := 0; i < launchedCount; i++ {
		result := <-results
		if result.err != nil && collectErr == nil {
			collectErr = result.err
		}
		if result.row != nil {
			mappedRows[result.index] = result.row
		}
	}

	if collectErr != nil {
		return nil, collectErr
	}

	// Filter out nil entries (empty rows)
	filtered := make([]*MappedRow, 0, len(mappedRows))
	for _, row := range mappedRows {
		if row != nil {
			filtered = append(filtered, row)
		}
	}

	return filtered, nil
}
