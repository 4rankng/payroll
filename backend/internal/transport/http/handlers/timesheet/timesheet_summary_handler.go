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

	// Parse date parameters in local timezone to match DB storage and the
	// other timesheet endpoints (which use ParseInLocation). Using
	// time.Parse here would interpret the date as UTC, shifting the half-open
	// range [fromDate, toDate) by the UTC offset (e.g. +07:00 in HCM), which
	// silently drops boundary-day entries and produces a smaller employee /
	// amount count than the grouped endpoint and Excel weekly export.
	loc, _ := time.LoadLocation("Local")
	if fromDate := c.Query("fromDate"); fromDate != "" {
		if d, err := time.ParseInLocation("2006-01-02", fromDate, loc); err == nil {
			filters.FromDate = &d
		}
	}

	if toDate := c.Query("toDate"); toDate != "" {
		if d, err := time.ParseInLocation("2006-01-02", toDate, loc); err == nil {
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
