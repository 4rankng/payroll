package handlers

import (
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/project"
	"api-server/internal/app/services/timesheet"
	"api-server/internal/pkg/clock"
	employeeHandler "api-server/internal/transport/http/handlers/employee"
)

type EmployeeHandler struct {
	*employeeHandler.Handler
}

func NewEmployeeHandlerWithServices(employeeService *employee.EmployeeService, timesheetService *timesheet.TimesheetService, projectEmployeeService *project.ProjectEmployeeService, auditService interface{}, clk clock.Clock) *EmployeeHandler {
	return &EmployeeHandler{
		Handler: employeeHandler.NewHandlerWithServices(employeeService, timesheetService, projectEmployeeService, auditService, clk),
	}
}
