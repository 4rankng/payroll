package handlers

import (
	timesheetHandler "api-server/internal/transport/http/handlers/timesheet"

	"api-server/internal/app/services/timesheet"
)

type TimesheetEditRequestHandler struct {
	*timesheetHandler.EditRequestHandler
}

func NewTimesheetEditRequestHandler(editRequestService *timesheet.TimesheetEditRequestService) *TimesheetEditRequestHandler {
	return &TimesheetEditRequestHandler{
		EditRequestHandler: timesheetHandler.NewEditRequestHandler(editRequestService),
	}
}
