package payroll

import (
	"context"
	"time"

	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
)

// PayrollReportByProjectService orchestrates payroll report by project operations
type PayrollReportByProjectService struct {
	domainService *domainServices.PayrollReportByProjectService
}

// NewPayrollReportByProjectService creates a new payroll report by project application service
func NewPayrollReportByProjectService(
	timesheetRepo domain.TimesheetRepository,
	projectRepo domain.ProjectRepository,
	employeeRepo domain.EmployeeRepository,
) *PayrollReportByProjectService {
	return &PayrollReportByProjectService{
		domainService: domainServices.NewPayrollReportByProjectService(
			timesheetRepo,
			projectRepo,
			employeeRepo,
		),
	}
}

// GetProjectsForPayrollReport orchestrates the retrieval of project payroll report data
func (s *PayrollReportByProjectService) GetProjectsForPayrollReport(ctx context.Context, atDate time.Time) ([]*domainServices.ProjectReportData, error) {
	return s.domainService.GetProjectsForPayrollReport(ctx, atDate)
}

// GetPayrollReportForProjects generates payroll report for specific projects and date range
func (s *PayrollReportByProjectService) GetPayrollReportForProjects(ctx context.Context, projectIDs []uint, fromDate, toDate time.Time) ([]*domainServices.ProjectReportData, error) {
	return s.domainService.GetPayrollReportForProjects(ctx, projectIDs, fromDate, toDate)
}
