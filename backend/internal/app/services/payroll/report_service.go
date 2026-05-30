package payroll

import (
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
)

// PayrollReportService orchestrates payroll report operations
type PayrollReportService struct {
	payrollReportDomainService *domainServices.PayrollReportService

	timesheetRepo        domain.TimesheetRepository
	employeeRepo         domain.EmployeeRepository
	projectEmployeeRepo  domain.ProjectEmployeeRepository
	projectRepo          domain.ProjectRepository
	bulkTransferFileRepo domain.BulkTransferFileRepository
}

// NewPayrollReportService creates a new payroll report application service
func NewPayrollReportService(
	timesheetRepo domain.TimesheetRepository,
	employeeRepo domain.EmployeeRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	projectRepo domain.ProjectRepository,
	bulkTransferFileRepo domain.BulkTransferFileRepository,
) *PayrollReportService {
	return &PayrollReportService{
		payrollReportDomainService: domainServices.NewPayrollReportService(
			timesheetRepo,
			projectEmployeeRepo,
			bulkTransferFileRepo,
			employeeRepo,
			projectRepo,
		),
		timesheetRepo:        timesheetRepo,
		employeeRepo:         employeeRepo,
		projectEmployeeRepo:  projectEmployeeRepo,
		projectRepo:          projectRepo,
		bulkTransferFileRepo: bulkTransferFileRepo,
	}
}
