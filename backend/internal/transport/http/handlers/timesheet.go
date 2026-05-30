package handlers

import (
	timesheetHandler "api-server/internal/transport/http/handlers/timesheet"

	"api-server/internal/app/services/config"
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/payroll"
	"api-server/internal/app/services/project"
	"api-server/internal/app/services/settlement"
	"api-server/internal/app/services/timesheet"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

type TimesheetHandler struct {
	*timesheetHandler.Handler
}

func NewTimesheetHandler(
	timesheetService *timesheet.TimesheetService,
	payrollReportService *payroll.PayrollReportService,
	settingsConfigService *config.SettingsConfigService,
	payrollReportExporter *payroll.PayrollReportExporter,
	payrollReportByProjectService *payroll.PayrollReportByProjectService,
	payrollReportByProjectExporter *payroll.PayrollReportByProjectExporter,
	projectService *project.ProjectService,
	projectEmployeeService *project.ProjectEmployeeService,
	payrateService *payroll.PayrateService,
	projectPermissionService *project.ProjectPermissionService,
	employeePermissionService *employee.EmployeePermissionService,
	settlementUploadService *settlement.SettlementUploadService,
	timesheetRepo domain.TimesheetRepository,
	auditService interface{},
	clk clock.Clock,
) *TimesheetHandler {
	return &TimesheetHandler{
		Handler: timesheetHandler.NewHandler(
			timesheetService,
			payrollReportService,
			settingsConfigService,
			payrollReportExporter,
			payrollReportByProjectService,
			payrollReportByProjectExporter,
			projectService,
			projectEmployeeService,
			payrateService,
			projectPermissionService,
			employeePermissionService,
			settlementUploadService,
			timesheetRepo,
			auditService,
			clk,
		),
	}
}
