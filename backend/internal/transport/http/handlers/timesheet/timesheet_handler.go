package timesheet

import (
	"strconv"
	"time"

	"api-server/internal/app/dto"
	constants "api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/timeutil"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// GetTimesheet retrieves a timesheet by ID
// @Summary Get a timesheet by ID
// @Description Get timesheet information by ID
// @Tags timesheets
// @Accept json
// @Produce json
// @Param id path int true "Timesheet ID"
// @Success 200 {object} dto.TimesheetResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/{id} [get]
func (h *Handler) GetTimesheet(c *gin.Context) {
	id, ok := h.validateTimesheetID(c)
	if !ok {
		return
	}

	timesheet, err := h.timesheetService.GetTimesheet(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	timesheetResponse := h.buildTimesheetResponse(timesheet)
	response.Success(c, timesheetResponse, constants.MsgTimesheetsRetrievedSuccessfullyVN)
}

// ListTimesheets lists timesheets with pagination and filtering
// @Summary List timesheets
// @Description Get paginated list of timesheet entries
// @Tags timesheets
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Items per page" default(20)
// @Param project_id query int false "Filter by project ID"
// @Param employee_id query int false "Filter by employee ID"
// @Param from_date query string false "Filter by date from (YYYY-MM-DD)"
// @Param to_date query string false "Filter by date to (YYYY-MM-DD)"
// @Param status query []string false "Filter by status (pending_approval,approved,rejected,paid,failed,cancelled)"
// @Param sortBy query string false "Sort by field" default(date)
// @Param sortOrder query string false "Sort order (asc/desc)" default(desc)
// @Success 200 {object} dto.ListTimesheetsResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets [get]
func (h *Handler) ListTimesheets(c *gin.Context) {
	filters := h.parseTimesheetFilters(c)

	// Apply partner role filtering - show timesheets for employees created by, assigned by, or granted access to the partner
	userRole := c.GetString(constants.CtxUserRole)
	if userRole == string(domain.RolePartner) {
		userID, exists := c.Get(constants.CtxUserID)
		if !exists {
			response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
			return
		}
		if uid, ok := userID.(uint); ok {
			filters.EmployeeCreatedBy = &uid
			filters.EmployeeAssignedByPartner = &uid
		} else {
			response.Forbidden(c, constants.MsgInvalidUserIDVN)
			return
		}
	}

	timesheets, err := h.timesheetService.ListTimesheets(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListTimesheetsVN)
		return
	}

	total, err := h.timesheetService.CountTimesheets(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountTimesheetsVN)
		return
	}

	var timesheetResponses []dto.TimesheetWithDetailsResponse
	for _, timesheet := range timesheets {
		timesheetResponses = append(timesheetResponses, h.buildTimesheetWithDetailsResponse(timesheet))
	}

	// Calculate pagination
	page := (filters.Offset / filters.Limit) + 1
	totalPages := (int(total) + filters.Limit - 1) / filters.Limit

	pagination := response.Pagination{
		Page:         page,
		PageSize:     filters.Limit,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	message := constants.MsgTimesheetsRetrievedSuccessfullyVN
	if page > 1 {
		message = "Page " + strconv.Itoa(page) + " of timesheets retrieved successfully"
	}

	if len(timesheetResponses) == 0 {
		response.SuccessEmptyWithPagination(c, "No timesheets found", pagination)
		return
	}

	response.SuccessWithPagination(c, timesheetResponses, message, pagination)
}

// UpdateTimesheet updates an existing timesheet entry
func (h *Handler) UpdateTimesheet(c *gin.Context) {
	id, ok := h.validateTimesheetID(c)
	if !ok {
		return
	}

	var req dto.UpdateTimesheetRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	userID, userRole, ok := h.getUserContext(c)
	if !ok {
		return
	}

	// Get existing timesheet
	timesheet, err := h.timesheetService.GetTimesheet(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Check if timesheet has been paid - nobody can edit paid timesheets
	if timesheet.IsPaid() {
		response.Forbidden(c, constants.MsgCannotEditPaidTimesheetVN)
		return
	}

	// Convert string role to domain role
	var domainUserRole domain.UserRole
	if userRole == string(domain.RoleAdmin) {
		domainUserRole = domain.RoleAdmin
	} else {
		domainUserRole = domain.RolePartner
	}

	// Check if timesheet can be edited by this user
	if !timesheet.CanBeEditedByUser(domainUserRole) {
		// Provide specific error message for approved timesheets
		if timesheet.IsApproved() {
			response.Forbidden(c, constants.MsgOnlyAdminCanEditApprovedTimesheetVN)
		} else {
			response.Forbidden(c, constants.MsgTimesheetCannotBeEditedVN)
		}
		return
	}

	// Update fields if provided
	if req.HoursWorked != nil {
		timesheet.HoursWorked = *req.HoursWorked
	}
	if req.PayType != nil {
		timesheet.PayType = *req.PayType
		// Payrate will be recalculated by the service during update
	}

	timesheet.CreatedBy = userID

	// If partner is editing approved timesheet, reset to pending
	if userRole != "admin" && timesheet.IsApproved() {
		if err := timesheet.SubmitForApproval(); err != nil {
			response.HandleDomainError(c, err)
			return
		}
	}

	if err := h.timesheetService.UpdateTimesheet(c.Request.Context(), timesheet.ID, timesheet, userID); err != nil {
		response.HandleDomainError(c, err)
		return
	}

	timesheetResponse := h.buildTimesheetResponse(timesheet)
	response.Success(c, timesheetResponse, constants.MsgTimesheetUpdatedSuccessfullyVN)
}

// DeleteTimesheet deletes a timesheet entry
// @Summary Delete a timesheet entry
// @Description Delete a timesheet entry with business rule validations
// @Tags timesheets
// @Accept json
// @Produce json
// @Param id path int true "Timesheet ID"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/{id} [delete]
func (h *Handler) DeleteTimesheet(c *gin.Context) {
	id, ok := h.validateTimesheetID(c)
	if !ok {
		return
	}

	userID, userRole, ok := h.getUserContext(c)
	if !ok {
		return
	}

	// Check user has ADMIN or PARTNER role
	if userRole != string(domain.RoleAdmin) && userRole != string(domain.RolePartner) {
		response.Forbidden(c, constants.MsgForbiddenVN)
		return
	}

	// Get existing timesheet to validate status
	timesheet, err := h.timesheetService.GetTimesheet(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Check if timesheet has been paid - paid timesheets cannot be deleted
	if timesheet.IsPaid() {
		response.Forbidden(c, constants.MsgCannotEditPaidTimesheetVN)
		return
	}

	// Apply role-specific deletion rules
	if userRole == string(domain.RolePartner) {
		// Partners can only delete timesheets in "pending_approval" or "rejected" status
		if !timesheet.IsPendingApproval() && !timesheet.IsRejected() {
			response.BadRequest(c, constants.MsgCanOnlyDeletePendingOrRejectedTimesheetVN)
			return
		}
	}
	// Admins can delete any timesheet as long as payment_status=pending (already validated above)

	if err := h.timesheetService.DeleteTimesheet(c.Request.Context(), id, userID); err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, nil, constants.MsgTimesheetDeletedSuccessfullyVN)
}

// GetTimesheetsByProjectAndDate retrieves timesheets by project with flexible date filtering
// @Summary Get timesheets by project and date
// @Description Get timesheet entries for a specific project with date/month/range filtering and pagination
// @Tags timesheets
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Param date query string false "Filter by specific date (YYYY-MM-DD format)"
// @Param month query string false "Filter by month (YYYY-MM format)"
// @Param fromDate query string false "Start date (YYYY-MM-DD format)"
// @Param toDate query string false "End date (YYYY-MM-DD format)"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Items per page" default(20)
// @Param status query string false "Filter by status (comma-separated)"
// @Param sortBy query string false "Sort field" default(date)
// @Param sortOrder query string false "Sort order (asc/desc)" default(desc)
// @Success 200 {object} dto.ListTimesheetsResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/projects/{id} [get]
func (h *Handler) GetTimesheetsByProjectAndDate(c *gin.Context) {
	// Get project ID from path parameter
	projectIDStr := c.Param("id")
	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidProjectIDVN)
		return
	}

	// Get user context to filter by employee for partners
	userID, userRole, ok := h.getUserContext(c)
	if !ok {
		return
	}

	// Parse filters using helper
	filters := h.parseTimesheetFilters(c)

	// Set project ID filter
	projectIDUint := uint(projectID)
	filters.ProjectIDs = []uint{projectIDUint}

	// Handle date filtering modes
	dateStr := c.Query("date")
	monthStr := c.Query("month")
	fromDateStr := c.Query("fromDate")
	toDateStr := c.Query("toDate")

	// Support specific date filtering — parse in Vietnam timezone (UTC+7)
	// to avoid boundary-off-by-7h bug where fromDate=05-22 skips day 22.
	loc := time.Local

	if dateStr != "" {
		date, err := time.ParseInLocation(timeutil.DateFormat, dateStr, loc)
		if err != nil {
			response.BadRequest(c, constants.MsgInvalidDateFormatVN)
			return
		}
		date = timeutil.StartOfDay(date.UTC())
		filters.Date = &date
	}

	// Support month filtering (YYYY-MM)
	if monthStr != "" {
		monthDate, err := time.ParseInLocation("2006-01", monthStr, loc)
		if err != nil {
			response.BadRequest(c, "Invalid month format. Use YYYY-MM format")
			return
		}
		// Set fromDate to first day of month
		fromDate := time.Date(monthDate.Year(), monthDate.Month(), 1, 0, 0, 0, 0, loc)
		// Set toDate to last day of month
		toDate := fromDate.AddDate(0, 1, -1)
		filters.FromDate = &fromDate
		filters.ToDate = &toDate
	}

	// Support date range filtering (if not using date or month)
	if fromDateStr != "" && dateStr == "" && monthStr == "" {
		fromDate, err := time.ParseInLocation(timeutil.DateFormat, fromDateStr, loc)
		if err != nil {
			response.BadRequest(c, constants.MsgInvalidFromDateFormatVN)
			return
		}
		fromDate = timeutil.StartOfDay(fromDate.UTC())
		filters.FromDate = &fromDate
	}

	if toDateStr != "" && dateStr == "" && monthStr == "" {
		toDate, err := time.ParseInLocation(timeutil.DateFormat, toDateStr, loc)
		if err != nil {
			response.BadRequest(c, constants.MsgInvalidToDateFormatVN)
			return
		}
		toDate = timeutil.StartOfDay(toDate.UTC())
		filters.ToDate = &toDate
	}

	// If user is a partner, filter by employees created by, assigned by, or granted access to them
	if userRole == string(domain.RolePartner) {
		filters.EmployeeCreatedBy = &userID
		filters.EmployeeAssignedByPartner = &userID
	}

	timesheets, err := h.timesheetService.ListTimesheets(c.Request.Context(), filters)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	total, err := h.timesheetService.CountTimesheets(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountTimesheetsVN)
		return
	}

	// Build response
	var timesheetResponses []dto.TimesheetResponse
	for _, ts := range timesheets {
		timesheetResponses = append(timesheetResponses, h.buildTimesheetResponse(ts))
	}

	// Calculate pagination
	page := (filters.Offset / filters.Limit) + 1
	totalPages := (int(total) + filters.Limit - 1) / filters.Limit

	pagination := response.Pagination{
		Page:         page,
		PageSize:     filters.Limit,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	if len(timesheetResponses) == 0 {
		response.SuccessEmptyWithPagination(c, constants.MsgNoProjectTimesheetsFoundVN, pagination)
		return
	}

	response.SuccessWithPagination(c, timesheetResponses, constants.MsgProjectTimesheetsRetrievedSuccessfullyVN, pagination)
}

// ListGroupedTimesheets lists timesheets grouped by employee with server-side pagination
// @Summary List timesheets grouped by employee
// @Description Get paginated list of timesheet entries grouped by employee (for partner view)
// @Tags timesheets
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Employees per page" default(20)
// @Param project_id query int false "Filter by project ID"
// @Param from_date query string false "Filter by date from (YYYY-MM-DD)"
// @Param to_date query string false "Filter by date to (YYYY-MM-DD)"
// @Param status query []string false "Filter by status (pending_approval,approved,rejected,paid,failed,cancelled)"
// @Param sortBy query string false "Sort by field (employee_name,employee_code,total_hours,total_amount)" default(employee_code)
// @Param sortOrder query string false "Sort order (asc/desc)" default(asc)
// @Success 200 {object} dto.ListGroupedTimesheetsResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/grouped [get]
func (h *Handler) ListGroupedTimesheets(c *gin.Context) {
	filters := h.parseTimesheetFilters(c)

	// Apply partner role filtering - show timesheets for employees created by, assigned by, or granted access to the partner
	userRole := c.GetString(constants.CtxUserRole)
	if userRole == string(domain.RolePartner) {
		userID, exists := c.Get(constants.CtxUserID)
		if !exists {
			response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
			return
		}
		if uid, ok := userID.(uint); ok {
			filters.EmployeeCreatedBy = &uid
			filters.EmployeeAssignedByPartner = &uid
		} else {
			response.Forbidden(c, constants.MsgInvalidUserIDVN)
			return
		}
	}

	// Get grouped employee data, all timesheets, and total count in a single call
	employeeGroups, allTimesheets, total, err := h.timesheetService.ListGroupedByEmployee(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListTimesheetsVN)
		return
	}

	// Group timesheets by employee
	groups := h.buildGroupedTimesheetResponse(employeeGroups, allTimesheets)

	// Calculate pagination
	page := (filters.Offset / filters.Limit) + 1
	totalPages := (int(total) + filters.Limit - 1) / filters.Limit

	pagination := response.Pagination{
		Page:         page,
		PageSize:     filters.Limit,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	result := dto.ListGroupedTimesheetsResponse{
		Groups: groups,
	}

	message := constants.MsgTimesheetsRetrievedSuccessfullyVN
	if page > 1 {
		message = "Page " + strconv.Itoa(page) + " of grouped timesheets retrieved successfully"
	}

	if len(groups) == 0 {
		response.SuccessWithPagination(c, result, "No timesheets found", pagination)
		return
	}

	response.SuccessWithPagination(c, result, message, pagination)
}

// buildGroupedTimesheetResponse builds grouped timesheet response from employee groups and timesheets
func (h *Handler) buildGroupedTimesheetResponse(employeeGroups []domain.EmployeeGroupResult, allTimesheets []*domain.Timesheet) []dto.EmployeeGroupedTimesheetResponse {
	// Create a map of employee_id to their timesheets
	employeeTimesheets := make(map[uint][]*domain.Timesheet)
	for _, ts := range allTimesheets {
		employeeTimesheets[ts.EmployeeID] = append(employeeTimesheets[ts.EmployeeID], ts)
	}

	var groups []dto.EmployeeGroupedTimesheetResponse

	for _, empGroup := range employeeGroups {
		// Get timesheets for this employee
		timesheets := employeeTimesheets[empGroup.EmployeeID]

		// Build entry responses
		var entries []dto.TimesheetWithDetailsResponse
		for _, ts := range timesheets {
			entries = append(entries, h.buildTimesheetWithDetailsResponse(ts))
		}

		// Calculate aggregated status
		aggregatedStatus := h.calculateAggregatedStatus(timesheets)

		group := dto.EmployeeGroupedTimesheetResponse{
			EmployeeID:       empGroup.EmployeeID,
			EmployeeName:     empGroup.EmployeeName,
			EmployeeCode:     empGroup.EmployeeCode,
			Entries:          entries,
			TotalHours:       empGroup.TotalHours,
			TotalAmount:      empGroup.TotalAmount,
			AggregatedStatus: aggregatedStatus,
		}

		groups = append(groups, group)
	}

	return groups
}

// calculateAggregatedStatus determines the overall status for a group of timesheets
func (h *Handler) calculateAggregatedStatus(timesheets []*domain.Timesheet) string {
	if len(timesheets) == 0 {
		return "draft"
	}

	// Count statuses
	statusCounts := make(map[string]int)
	paidCount := 0

	for _, ts := range timesheets {
		if ts.PaymentStatus == domain.PaymentStatusPaid && ts.Status == domain.TimesheetStatusApproved {
			paidCount++
		} else {
			statusCounts[string(ts.Status)]++
		}
	}

	totalEntries := len(timesheets)

	// All entries are paid
	if paidCount == totalEntries {
		return "approved" // Will be displayed as "Đã thanh toán" via payment status in frontend
	}

	// Check for mixed statuses
	if len(statusCounts) > 1 {
		return "mixed"
	}

	// All same status
	for status, count := range statusCounts {
		if count == totalEntries {
			return status
		}
	}

	return "mixed"
}
