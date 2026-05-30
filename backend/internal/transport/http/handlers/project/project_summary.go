package project

import (
	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// GetProjectSummary gets aggregated project metrics
// @Summary Get project summary
// @Description Get aggregated metrics for all projects
// @Tags projects
// @Accept json
// @Produce json
// @Success 200 {object} dto.ProjectSummaryResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /projects/summary [get]
func (h *Handler) GetProjectSummary(c *gin.Context) {
	summary, err := h.projectService.GetProjectSummary(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToGetProjectSummaryVN)
		return
	}

	summaryResponse := dto.ProjectSummaryResponse{
		TotalActiveProjects:       summary.TotalActiveProjects,
		TotalReceivedVND:          summary.TotalReceivedVND,
		TotalPayoutVND:            summary.TotalPayoutVND,
		TotalPendingPayableVND:    summary.TotalPendingPayableVND,
		TotalPendingReceivableVND: summary.TotalPendingReceivableVND,
	}

	response.Success(c, summaryResponse, constants.MsgProjectSummaryRetrievedVN)
}

// GetPartnerProjectSummary gets project statistics for partner users
// @Summary Get partner project summary
// @Description Get project statistics data for partner dashboard (only for projects created by the current user)
// @Tags projects
// @Accept json
// @Produce json
// @Success 200 {object} dto.PartnerProjectSummaryResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /projects/partner-summary [get]
func (h *Handler) GetPartnerProjectSummary(c *gin.Context) {
	// 1. Validate user context
	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	if uid, ok := userID.(uint); ok {
		summary, err := h.projectService.GetPartnerProjectSummary(c.Request.Context(), uid)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToGetPartnerProjectSummaryVN)
			return
		}

		response.Success(c, summary, constants.MsgPartnerProjectSummaryRetrievedVN)
	} else {
		response.Forbidden(c, constants.MsgInvalidUserIDVN)
		return
	}
}
