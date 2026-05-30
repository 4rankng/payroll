package timesheet

import (
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// AddToPayroll marks a timesheet to be included in the next payroll run
// @Summary Add timesheet to next payroll
// @Description Mark a timesheet so it will be included in the next payroll run
// @Tags timesheets
// @Accept json
// @Produce json
// @Param id path int true "Timesheet ID"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/{id}/add-to-payroll [post]
func (h *Handler) AddToPayroll(c *gin.Context) {
	id, ok := h.validateTimesheetID(c)
	if !ok {
		return
	}

	userID, userRole, ok := h.getUserContext(c)
	if !ok {
		return
	}

	// Only admins can add a timesheet to payroll
	if userRole != string(domain.RoleAdmin) {
		response.Forbidden(c, constants.MsgForbiddenVN)
		return
	}

	if err := h.timesheetService.AddTimesheetToNextPayroll(c.Request.Context(), id, userID); err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, nil, constants.MsgTimesheetMarkedForNextPayrollVN)
}
