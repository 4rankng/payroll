package timesheet

import (
	"api-server/internal/app/services/config"
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/app/services/payroll"
	"api-server/internal/app/services/project"
	"api-server/internal/app/services/settlement"
	"api-server/internal/app/services/timesheet"
	"api-server/internal/domain"
)

// HandlerConfig consolidates all handler dependencies
type HandlerConfig struct {
	// Core Services
	TimesheetService *timesheet.TimesheetService

	// Report Services
	PayrollReportService           *payroll.PayrollReportService
	PayrollReportExporter          *payroll.PayrollReportExporter
	PayrollReportByProjectService  *payroll.PayrollReportByProjectService
	PayrollReportByProjectExporter *payroll.PayrollReportByProjectExporter

	// Supporting Services
	TimesheetResponseService  *timesheet.TimesheetResponseService
	SettingsConfigService     *config.SettingsConfigService
	ProjectService            *project.ProjectService
	ProjectEmployeeService    *project.ProjectEmployeeService
	PayrateService            *payroll.PayrateService
	ProjectPermissionService  *project.ProjectPermissionService
	EmployeePermissionService *employee.EmployeePermissionService
	SettlementUploadService   *settlement.SettlementUploadService

	// Repositories
	TimesheetRepo domain.TimesheetRepository

	// Audit
	AuditService *infrastructure.AuditService
}

// NewHandlerFromConfig creates a new handler from config
func NewHandlerFromConfig(cfg *HandlerConfig) *Handler {
	return &Handler{
		timesheetService:               cfg.TimesheetService,
		payrollReportService:           cfg.PayrollReportService,
		settingsConfigService:          cfg.SettingsConfigService,
		payrollReportExporter:          cfg.PayrollReportExporter,
		payrollReportByProjectService:  cfg.PayrollReportByProjectService,
		payrollReportByProjectExporter: cfg.PayrollReportByProjectExporter,
		timesheetResponseService:       cfg.TimesheetResponseService,
		projectService:                 cfg.ProjectService,
		projectEmployeeService:         cfg.ProjectEmployeeService,
		payrateService:                 cfg.PayrateService,
		projectPermissionService:       cfg.ProjectPermissionService,
		employeePermissionService:      cfg.EmployeePermissionService,
		settlementUploadService:        cfg.SettlementUploadService,
		timesheetRepo:                  cfg.TimesheetRepo,
		auditService:                   cfg.AuditService,
	}
}
