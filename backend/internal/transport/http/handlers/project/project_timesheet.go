package project

import (
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/timesheet"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ListProjectTimesheets lists timesheets for a specific project with pagination and filtering
// @Summary List project timesheets
// @Description Get a list of timesheets for a specific project with pagination and filtering options
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Items per page" default(20)
// @Param from_date query string false "Filter by date from (YYYY-MM-DD)"
// @Param to_date query string false "Filter by date to (YYYY-MM-DD)"
// @Param status query []string false "Filter by status (approved,pending_approval,rejected)"
// @Success 200 {object} dto.ListTimesheetsResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /projects/{id}/timesheets [get]
func (h *Handler) ListProjectTimesheets(c *gin.Context) {
	// Validate project ID
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidProjectIDVN)
		return
	}

	// Check if project exists
	_, err = h.projectService.GetProject(c.Request.Context(), uint(projectID))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Access control handled by middleware (ProjectPermissionService.CanUserAccessProject)

	// Parse pagination parameters
	page := 1
	if p := c.Query("page"); p != "" {
		if pageNum, err := strconv.Atoi(p); err == nil && pageNum > 0 {
			page = pageNum
		}
	}

	pageSize := 20
	if ps := c.Query("pageSize"); ps != "" {
		if size, err := strconv.Atoi(ps); err == nil && size > 0 && size <= 100 {
			pageSize = size
		}
	}

	// Build timesheet filters
	filters := domain.TimesheetFilters{
		ProjectIDs: []uint{uint(projectID)},
		Limit:      pageSize,
		Offset:     (page - 1) * pageSize,
		SortBy:     "date",
		SortOrder:  "desc",
	}

	// Parse date filters
	if fromDate := c.Query("fromDate"); fromDate != "" {
		if d, err := time.Parse("2006-01-02", fromDate); err == nil {
			filters.FromDate = &d
		} else {
			response.BadRequest(c, constants.MsgInvalidFromDateFormatVN)
			return
		}
	}

	if toDate := c.Query("toDate"); toDate != "" {
		if d, err := time.Parse("2006-01-02", toDate); err == nil {
			filters.ToDate = &d
		} else {
			response.BadRequest(c, constants.MsgInvalidToDateFormatVN)
			return
		}
	}

	// Parse status filters
	if statusList := c.QueryArray("status"); len(statusList) > 0 {
		var statuses []domain.TimesheetStatus
		for _, s := range statusList {
			switch s {
			case "approved":
				statuses = append(statuses, domain.TimesheetStatusApproved)
			case "pending_approval":
				statuses = append(statuses, domain.TimesheetStatusPendingApproval)
			case "rejected":
				statuses = append(statuses, domain.TimesheetStatusRejected)
			default:
				response.BadRequest(c, constants.MsgInvalidStatusFilterVN)
				return
			}
		}
		filters.TimesheetStatus = statuses
	}

	// Get timesheets
	timesheets, err := h.timesheetService.ListTimesheets(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListTimesheetsVN)
		return
	}

	// Get total count for pagination
	total, err := h.timesheetService.CountTimesheets(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountTimesheetsVN)
		return
	}

	// Build response data using the service to properly extract data
	var timesheetResponses []dto.TimesheetWithDetailsResponse
	timesheetResponseService := timesheet.NewTimesheetResponseService(h.timesheetService)
	for _, timesheet := range timesheets {
		timesheetResponse := timesheetResponseService.BuildTimesheetWithDetailsResponse(timesheet)
		timesheetResponses = append(timesheetResponses, timesheetResponse)
	}

	// Calculate pagination
	totalPages := (int(total) + pageSize - 1) / pageSize

	pagination := response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	if len(timesheetResponses) == 0 {
		response.SuccessEmptyWithPagination(c, constants.MsgNoProjectTimesheetsFoundVN, pagination)
		return
	}

	response.SuccessWithPagination(c, timesheetResponses, constants.MsgProjectTimesheetsRetrievedSuccessfullyVN, pagination)
}
