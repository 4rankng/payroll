package handlers

import (
	projectHandler "api-server/internal/transport/http/handlers/project"

	"api-server/internal/app/services/payroll"
	"api-server/internal/app/services/project"
	"api-server/internal/app/services/timesheet"
	"api-server/internal/pkg/clock"
)

type ProjectHandler struct {
	*projectHandler.Handler
}

func NewProjectHandlerWithServices(
	projectService *project.ProjectService,
	payrateService *payroll.PayrateService,
	timesheetService *timesheet.TimesheetService,
	projectEmployeeService *project.ProjectEmployeeService,
	projectPermissionService *project.ProjectPermissionService,
	clk clock.Clock,
) *ProjectHandler {
	return &ProjectHandler{
		Handler: projectHandler.NewHandlerWithServices(
			projectService,
			payrateService,
			timesheetService,
			projectEmployeeService,
			projectPermissionService,
			clk,
		),
	}
}
