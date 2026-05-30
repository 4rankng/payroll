package timesheet

import (
	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ApproveTimesheet approves a timesheet
// @Summary Approve a timesheet
// @Description Approve a timesheet entry
// @Tags timesheets
// @Accept json
// @Produce json
// @Param id path int true "Timesheet ID"
// @Success 200 {object} dto.TimesheetResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/{id}/approve [put]
func (h *Handler) ApproveTimesheet(c *gin.Context) {
	id, ok := h.validateTimesheetID(c)
	if !ok {
		return
	}

	userID, _, ok := h.getUserContext(c)
	if !ok {
		return
	}

	if err := h.timesheetService.ApproveTimesheet(c.Request.Context(), id, userID); err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Get updated timesheet
	timesheet, err := h.timesheetService.GetTimesheet(c.Request.Context(), id)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToGetUpdatedTimesheetVN)
		return
	}

	timesheetResponse := h.buildTimesheetResponse(timesheet)
	response.Success(c, timesheetResponse, constants.MsgTimesheetApprovedSuccessfullyVN)
}

// RejectTimesheet rejects a timesheet
// @Summary Reject a timesheet
// @Description Reject a timesheet entry with reason
// @Tags timesheets
// @Accept json
// @Produce json
// @Param id path int true "Timesheet ID"
// @Param rejection body dto.RejectTimesheetRequest true "Rejection data"
// @Success 200 {object} dto.TimesheetResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/{id}/reject [put]
func (h *Handler) RejectTimesheet(c *gin.Context) {
	id, ok := h.validateTimesheetID(c)
	if !ok {
		return
	}

	var req dto.RejectTimesheetRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	userID, _, ok := h.getUserContext(c)
	if !ok {
		return
	}

	if err := h.timesheetService.RejectTimesheet(c.Request.Context(), id, req.RejectionReason, userID); err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Get updated timesheet
	timesheet, err := h.timesheetService.GetTimesheet(c.Request.Context(), id)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToGetUpdatedTimesheetVN)
		return
	}

	timesheetResponse := h.buildTimesheetResponse(timesheet)
	response.Success(c, timesheetResponse, constants.MsgTimesheetRejectedSuccessfullyVN)
}
