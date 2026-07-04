package employee

import (
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/project"
	"api-server/internal/app/services/timesheet"
	"api-server/internal/transport/http/validation"
)

// HandlerConfig consolidates all handler dependencies
type HandlerConfig struct {
	EmployeeService        *employee.EmployeeService
	TimesheetService       *timesheet.TimesheetService
	ProjectEmployeeService *project.ProjectEmployeeService
	Validator              *validation.RequestValidator
}

// NewHandlerFromConfig creates a new handler from config
func NewHandlerFromConfig(cfg *HandlerConfig) *Handler {
	return &Handler{
		employeeService:        cfg.EmployeeService,
		timesheetService:       cfg.TimesheetService,
		projectEmployeeService: cfg.ProjectEmployeeService,
		validator:              cfg.Validator,
	}
}
