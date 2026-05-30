package services

import (
	"context"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
)

// PayrollReportService handles payroll report domain logic
type PayrollReportService struct {
	timesheetRepo        domain.TimesheetRepository
	projectEmployeeRepo  domain.ProjectEmployeeRepository
	bulkTransferFileRepo domain.BulkTransferFileRepository
	employeeRepo         domain.EmployeeRepository
	projectRepo          domain.ProjectRepository
}

// NewPayrollReportService creates a new payroll report domain service
func NewPayrollReportService(
	timesheetRepo domain.TimesheetRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	bulkTransferFileRepo domain.BulkTransferFileRepository,
	employeeRepo domain.EmployeeRepository,
	projectRepo domain.ProjectRepository,
) *PayrollReportService {
	return &PayrollReportService{
		timesheetRepo:        timesheetRepo,
		projectEmployeeRepo:  projectEmployeeRepo,
		bulkTransferFileRepo: bulkTransferFileRepo,
		employeeRepo:         employeeRepo,
		projectRepo:          projectRepo,
	}
}

// PayrollReportEntry represents a single entry in the payroll report
type PayrollReportEntry struct {
	RowNumber    int
	PaymentDate  *time.Time
	EmployeeName string
	EmployeeCCCD string
	ProjectName  string
	PaidAmount   int64
	TimesheetIDs []uint
}

// GetPaidTimesheetsForPayrollReport retrieves paid timesheets within date range and enriches them with employee codes
func (s *PayrollReportService) GetPaidTimesheetsForPayrollReport(ctx context.Context, fromDate, toDate time.Time) ([]*PayrollReportEntry, error) {
	// Get paid timesheets within date range based on timesheet date (not payment date)
	filters := domain.TimesheetFilters{
		PaymentStatus: []domain.PaymentStatus{domain.PaymentStatusPaid},
		FromDate:      &fromDate,
		ToDate:        &toDate,
		Limit:         10000, // Large number to get all paid timesheets in date range
		Offset:        0,
		SortBy:        "date", // Sort by work date instead of payment date
		SortOrder:     "asc",
	}

	timesheets, err := s.timesheetRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Enrich with employee codes
	var reportEntries []*PayrollReportEntry
	rowNumber := 1

	for _, ts := range timesheets {
		entry := &PayrollReportEntry{
			RowNumber:    rowNumber,
			PaymentDate:  ts.PaidAt,
			PaidAmount:   ts.PaidAmount,
			TimesheetIDs: []uint{ts.ID},
		}

		// Extract employee information
		if ts.Employee != nil {
			entry.EmployeeName = ts.Employee.FormattedFullname()
			entry.EmployeeCCCD = ts.Employee.CCCD
		}

		// Extract project information
		if ts.Project != nil {
			entry.ProjectName = ts.Project.Name
		}

		reportEntries = append(reportEntries, entry)
		rowNumber++
	}

	return reportEntries, nil
}

// GeneratePayrollReportFromBulkTransferFiles generates payroll report from bulk transfer files
func (s *PayrollReportService) GeneratePayrollReportFromBulkTransferFiles(ctx context.Context, fromDate, toDate time.Time) ([]*PayrollReportEntry, error) {
	// Get processed bulk transfer files within date range
	files, err := s.bulkTransferFileRepo.ListForPayrollReport(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	// Pre-collect all unique employee and project IDs across all files
	employeeIDSet := make(map[uint]struct{})
	projectIDSet := make(map[uint]struct{})
	for _, file := range files {
		fileDataArray, err := dto.ParseBulkTransferFileData(file.Data)
		if err != nil {
			continue
		}
		for _, data := range fileDataArray {
			employeeIDSet[data.EmployeeID] = struct{}{}
			projectIDSet[data.ProjectID] = struct{}{}
		}
	}

	// Batch-fetch all employees
	employeeIDs := make([]int64, 0, len(employeeIDSet))
	for id := range employeeIDSet {
		employeeIDs = append(employeeIDs, int64(id))
	}
	employeeList, err := s.employeeRepo.GetByIDs(ctx, employeeIDs)
	if err != nil {
		return nil, err
	}
	employeeMap := make(map[uint]*domain.Employee, len(employeeList))
	for _, e := range employeeList {
		employeeMap[e.ID] = e
	}

	// Batch-fetch all projects
	projectIDs := make([]uint, 0, len(projectIDSet))
	for id := range projectIDSet {
		projectIDs = append(projectIDs, id)
	}
	projectsById, err := s.projectRepo.GetByIDs(ctx, projectIDs)
	if err != nil {
		return nil, err
	}

	var reportEntries []*PayrollReportEntry
	rowNumber := 1

	for _, file := range files {
		fileDataArray, err := dto.ParseBulkTransferFileData(file.Data)
		if err != nil {
			continue
		}

		for _, data := range fileDataArray {
			employee, ok := employeeMap[data.EmployeeID]
			if !ok {
				continue
			}

			project, ok := projectsById[data.ProjectID]
			if !ok {
				continue
			}

			entry := &PayrollReportEntry{
				RowNumber:    rowNumber,
				PaymentDate:  &file.CreatedAt,
				EmployeeName: employee.FormattedFullname(),
				EmployeeCCCD: employee.CCCD,
				ProjectName:  project.Name,
				PaidAmount:   data.Amount,
				TimesheetIDs: data.TimesheetIDs,
			}

			reportEntries = append(reportEntries, entry)
			rowNumber++
		}
	}

	return reportEntries, nil
}
