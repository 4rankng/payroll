package timesheet

import (
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// GetSummary gets summary statistics for timesheets
func (h *Handler) GetSummary(c *gin.Context) {
	// 1. Validate user context
	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	userRole := c.GetString(constants.CtxUserRole)
	filters := domain.TimesheetFilters{}

	// 2. Apply role-based filtering for partner users - only show stats for employees they created
	if userRole == string(domain.RolePartner) {
		if uid, ok := userID.(uint); ok {
			filters.EmployeeCreatedBy = &uid
		} else {
			response.Forbidden(c, constants.MsgInvalidUserIDVN)
			return
		}
	}

	// 3. Parse optional filters
	if projectID := c.Query("project_id"); projectID != "" {
		if id, err := strconv.ParseUint(projectID, 10, 32); err == nil {
			uid := uint(id)
			filters.ProjectIDs = []uint{uid}
		}
	}

	if employeeID := c.Query("employee_id"); employeeID != "" {
		if id, err := strconv.ParseUint(employeeID, 10, 32); err == nil {
			uid := uint(id)
			filters.EmployeeID = &uid
		}
	}

	// Parse date parameters
	if fromDate := c.Query("fromDate"); fromDate != "" {
		if d, err := time.Parse("2006-01-02", fromDate); err == nil {
			filters.FromDate = &d
		}
	}

	if toDate := c.Query("toDate"); toDate != "" {
		if d, err := time.Parse("2006-01-02", toDate); err == nil {
			filters.ToDate = &d
		}
	}

	stats, err := h.timesheetService.GetSummaryStats(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToGetTimesheetSummaryVN)
		return
	}

	summaryResponse := dto.TimesheetSummaryResponse{
		TotalEntries:         stats.TotalEntries,
		TotalEmployees:       stats.TotalEmployees,
		PendingApproval:      stats.PendingApproval,
		PendingEmployees:     stats.PendingEmployees,
		PendingPaymentAmount: stats.PendingPaymentAmount,
		ApprovedEntries:      stats.ApprovedEntries,
		PaidEntries:          stats.PaidEntries,
		PaidAmount:           stats.PaidAmount,
		PaidEmployees:        stats.PaidEmployees,
		RejectedEntries:      stats.RejectedEntries,
		LastUpdated:          stats.LastUpdated,
	}

	response.Success(c, summaryResponse, constants.MsgTimesheetSummaryRetrievedVN)
}
