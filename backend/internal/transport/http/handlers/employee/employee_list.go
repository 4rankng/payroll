package employee

import (
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// buildBankInfo creates bank info for employee responses
func buildBankInfo(bank *domain.Bank) *dto.EmployeeBankInfo {
	if bank == nil {
		return nil
	}
	return &dto.EmployeeBankInfo{
		ID:         bank.ID,
		BranchName: bank.BranchName,
	}
}

func formatDatePointer(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format("2006-01-02")
	return &formatted
}

func buildProjectInfo(project domain.CurrentProject) dto.EmployeeProjectInfo {
	startDate := ""
	if !project.StartDate.IsZero() {
		startDate = project.StartDate.Format("2006-01-02")
	}

	return dto.EmployeeProjectInfo{
		ProjectID:              project.ProjectID,
		ProjectEmployeeID:      project.ProjectEmployeeID,
		Name:                   project.Name,
		Code:                   project.Code,
		ClientName:             project.ClientName,
		Position:               project.Position,
		StartDate:              startDate,
		LastDate:               formatDatePointer(project.LastDate),
		PaymentSchedule:        project.PaymentSchedule,
		PendingPaymentSchedule: project.PendingPaymentSchedule,
		ScheduleEffectiveFrom:  formatDatePointer(project.ScheduleEffectiveFrom),
		IsFlexible:             project.IsFlexible,
		CheckInEnabled:         project.CheckInEnabled,
	}
}

// allowedEmployeeSortFields is a set of valid sort fields to prevent SQL injection.
var allowedEmployeeSortFields = map[string]struct{}{
	"created_at": {}, "updated_at": {}, "fullname": {}, "email": {}, "cccd": {},
	"mobile": {}, "address": {}, "bank_account_number": {}, "bank_account_name": {},
}

// parseEmployeeFilters builds EmployeeFilters from query params, applying pagination,
// sorting, search, date range, and role-based access control.
func (h *Handler) parseEmployeeFilters(c *gin.Context) (domain.EmployeeFilters, int, int, bool) {
	pg := helpers.ParsePagination(c, 100)

	filters := domain.EmployeeFilters{
		Limit:     pg.Limit,
		Offset:    pg.Offset,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	// Support both camelCase and snake_case sort params
	for _, key := range []string{"sortBy", "sort_by"} {
		if v := c.Query(key); v != "" {
			if _, ok := allowedEmployeeSortFields[v]; ok {
				filters.SortBy = v
				break
			}
		}
	}
	for _, key := range []string{"sortOrder", "sort_order"} {
		if v := c.Query(key); v != "" {
			switch v {
			case "asc", "desc", "ASC", "DESC":
				filters.SortOrder = v
			}
			break
		}
	}

	if search := c.Query("search"); search != "" {
		filters.Search = search
	}

	if fromDate := c.Query("fromDate"); fromDate != "" {
		if parsed, err := time.Parse("2006-01-02", fromDate); err == nil {
			filters.FromDate = &parsed
		} else {
			response.BadRequest(c, constants.MsgInvalidFromDateFormatVN)
			return filters, 0, 0, false
		}
	}

	if toDate := c.Query("toDate"); toDate != "" {
		if parsed, err := time.Parse("2006-01-02", toDate); err == nil {
			filters.ToDate = &parsed
		} else {
			response.BadRequest(c, constants.MsgInvalidToDateFormatVN)
			return filters, 0, 0, false
		}
	}

	// Role-based access control for partner users
	userRole := c.GetString(constants.CtxUserRole)
	if userRole == string(domain.RolePartner) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
			return filters, 0, 0, false
		}
		uid, ok := userID.(uint)
		if !ok {
			response.Forbidden(c, constants.MsgInvalidUserIDVN)
			return filters, 0, 0, false
		}
		filters.AccessibleBy = &uid
	}

	return filters, pg.Page, pg.PageSize, true
}

// buildEmployeeListResponse converts a slice of EmployeeWithProjects to response DTOs.
func buildEmployeeListResponse(employees []*domain.EmployeeWithProjects) []dto.EmployeeResponse {
	result := make([]dto.EmployeeResponse, 0, len(employees))
	for _, emp := range employees {
		var dobStr *string
		if emp.DateOfBirth != nil {
			s := emp.DateOfBirth.Format("2006-01-02")
			dobStr = &s
		}
		r := dto.EmployeeResponse{
			ID:                emp.ID,
			Fullname:          emp.Fullname,
			Email:             emp.Email,
			CCCD:              emp.CCCD,
			Address:           emp.Address,
			Mobile:            emp.Mobile,
			Bank:              buildBankInfo(emp.Bank),
			BankAccountNumber: emp.BankAccountNumber,
			BankAccountName:   emp.BankAccountName,
			DateOfBirth:       dobStr,
			CreatedBy:         emp.CreatedBy,
			CreatedAt:         emp.CreatedAt,
			UpdatedAt:         emp.UpdatedAt,
			CurrentProjects:   make([]dto.EmployeeProjectInfo, 0, len(emp.CurrentProjects)),
		}
		for _, p := range emp.CurrentProjects {
			r.CurrentProjects = append(r.CurrentProjects, buildProjectInfo(p))
		}
		result = append(result, r)
	}
	return result
}

// ListEmployees lists all employees with pagination and filtering
// @Summary List employees
// @Description Get a paginated list of employees with filtering options. Inactive employees are not returned by default.
// @Tags employees
// @Accept json
// @Produce json
// @Param page query int false "Page number, starts from 1" default(1)
// @Param pageSize query int false "Items per page (max: 100)" default(100)
// @Param sortBy query string false "Field to sort by" default(created_at)
// @Param sortOrder query string false "Sort direction asc or desc" default(desc)
// @Param status query string false "Filter by assignment status: working (assigned to projects) or unassigned"
// @Param search query string false "Search by name, email, or CCCD"
// @Param projectId query int false "Filter employees assigned to specific project"
// @Param fromDate query string false "Filter employees created from this date (YYYY-MM-DD)"
// @Param toDate query string false "Filter employees created until this date (YYYY-MM-DD)"
// @Success 200 {object} dto.ListEmployeesResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees [get]
func (h *Handler) ListEmployees(c *gin.Context) {
	filters, page, pageSize, ok := h.parseEmployeeFilters(c)
	if !ok {
		return
	}

	if status := c.Query("status"); status != "" {
		if status != "working" && status != "unassigned" {
			response.BadRequest(c, constants.MsgInvalidEmployeeStatusFilterVN)
			return
		}
		filters.Status = status
	}

	if projectID := c.Query("projectId"); projectID != "" {
		if id, err := strconv.ParseUint(projectID, 10, 32); err == nil {
			uid := uint(id)
			filters.ProjectID = &uid
		}
	}

	if schedule := c.Query("paymentSchedule"); schedule != "" {
		switch schedule {
		case "weekly", "monthly", "flexible":
			filters.PaymentSchedule = schedule
		default:
			response.BadRequest(c, "paymentSchedule phải là weekly, monthly hoặc flexible")
			return
		}
	}

	employees, err := h.employeeService.ListEmployeesWithAllProjects(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
		return
	}

	total, err := h.employeeService.CountEmployees(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountEmployeesVN)
		return
	}

	employeeResponses := buildEmployeeListResponse(employees)
	pagination := helpers.CalculatePagination(page, pageSize, total)

	if len(employeeResponses) == 0 {
		response.SuccessEmptyWithPagination(c, "No records found", pagination)
		return
	}

	message := constants.MsgEmployeesRetrievedSuccessfully
	if page > 1 {
		message = "Page " + strconv.Itoa(page) + " of employees retrieved successfully"
	}
	response.SuccessWithPagination(c, employeeResponses, message, pagination)
}

// GetEmployeesMissingBankDetails retrieves employees with missing banking information
// @Summary Get employees with missing bank details
// @Description Get employees who have incomplete banking information (missing bank account details)
// @Tags employees
// @Accept json
// @Produce json
// @Param page query int false "Page number, starts from 1" default(1)
// @Param pageSize query int false "Items per page (max: 100)" default(100)
// @Param sortBy query string false "Field to sort by" default(created_at)
// @Param sortOrder query string false "Sort direction asc or desc" default(desc)
// @Param search query string false "Search by name, email, or CCCD"
// @Success 200 {object} dto.ListEmployeesResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/missing-bank-details [get]
func (h *Handler) GetEmployeesMissingBankDetails(c *gin.Context) {
	filters, page, pageSize, ok := h.parseEmployeeFilters(c)
	if !ok {
		return
	}

	employees, err := h.employeeService.GetEmployeesWithMissingBankDetails(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
		return
	}

	total, err := h.employeeService.CountEmployeesWithMissingBankDetails(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountEmployeesVN)
		return
	}

	employeeResponses := buildEmployeeListResponse(employees)
	pagination := helpers.CalculatePagination(page, pageSize, total)

	if len(employeeResponses) == 0 {
		response.SuccessEmptyWithPagination(c, "No records found", pagination)
		return
	}

	message := "Employees with missing bank details retrieved successfully"
	if page > 1 {
		message = "Page " + strconv.Itoa(page) + " of employees with missing bank details retrieved successfully"
	}
	response.SuccessWithPagination(c, employeeResponses, message, pagination)
}

// GetUnassignedEmployees retrieves employees not assigned to any project at a given date
// @Summary Get unassigned employees
// @Description Get employees not assigned to any project at a specific date
// @Tags employees
// @Accept json
// @Produce json
// @Param at_date query string false "Date to check assignments (YYYY-MM-DD format, defaults to today)" default(today)
// @Param page query int false "Page number, starts from 1" default(1)
// @Param pageSize query int false "Items per page (max: 100)" default(100)
// @Param sortBy query string false "Field to sort by" default(created_at)
// @Param sortOrder query string false "Sort direction asc or desc" default(desc)
// @Param search query string false "Search by name, email, or CCCD"
// @Success 200 {object} dto.ListEmployeesResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/unassigned [get]
func (h *Handler) GetUnassignedEmployees(c *gin.Context) {
	atDate := h.clock.Now()
	if atDateStr := c.Query("at_date"); atDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", atDateStr); err == nil {
			atDate = parsed
		} else {
			response.BadRequest(c, constants.MsgInvalidAtDateFormatVN)
			return
		}
	}

	filters, page, pageSize, ok := h.parseEmployeeFilters(c)
	if !ok {
		return
	}

	employees, err := h.employeeService.GetUnassignedEmployeesAtDate(c.Request.Context(), atDate, filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToRetrieveUnassignedEmployeesVN)
		return
	}

	total, err := h.employeeService.CountUnassignedEmployeesAtDate(c.Request.Context(), atDate, filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountUnassignedEmployeesVN)
		return
	}

	var employeeResponses []dto.EmployeeResponse
	for _, emp := range employees {
		employeeResponses = append(employeeResponses, h.buildEmployeeResponse(c.Request.Context(), emp))
	}

	pagination := helpers.CalculatePagination(page, pageSize, total)

	if len(employeeResponses) == 0 {
		response.SuccessEmptyWithPagination(c, "No unassigned employees found", pagination)
		return
	}

	message := "Unassigned employees retrieved successfully"
	if page > 1 {
		message = "Page " + strconv.Itoa(page) + " of unassigned employees retrieved successfully"
	}
	response.SuccessWithPagination(c, employeeResponses, message, pagination)
}
