package project_employee

import (
	"log/slog"
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/project"
	"api-server/internal/pkg/clock"
)

type Handler struct {
	projectEmployeeService    *project.ProjectEmployeeService
	employeeService           *employee.EmployeeService
	projectService            *project.ProjectService
	projectPermissionService  *project.ProjectPermissionService
	employeePermissionService *employee.EmployeePermissionService
	logger                    *slog.Logger
	clock                     clock.Clock
}

func NewHandler(projectEmployeeService *project.ProjectEmployeeService, employeeService *employee.EmployeeService, projectService *project.ProjectService, projectPermissionService *project.ProjectPermissionService, employeePermissionService *employee.EmployeePermissionService, logger *slog.Logger, clk clock.Clock) *Handler {
	if clk == nil {
		clk = clock.New()
	}
	return &Handler{
		projectEmployeeService:    projectEmployeeService,
		employeeService:           employeeService,
		projectService:            projectService,
		projectPermissionService:  projectPermissionService,
		employeePermissionService: employeePermissionService,
		logger:                    logger,
		clock:                     clk,
	}
}

func (h *Handler) ListProjectEmployees(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "Project ID must be a valid number")
		return
	}

	projectIDUint := uint(projectID)
	filters := domain.ProjectEmployeeFilters{
		ProjectID: &projectIDUint,
		Limit:     50,
		Offset:    0,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	// Handle page/pageSize pagination (new model)
	page := 1
	if p := c.Query("page"); p != "" {
		if pageNum, err := strconv.Atoi(p); err == nil && pageNum > 0 {
			page = pageNum
		}
	}

	pageSize := 50
	if ps := c.Query("pageSize"); ps != "" {
		if size, err := strconv.Atoi(ps); err == nil && size > 0 && size <= 10000 {
			pageSize = size
		}
	}

	// Add status filter
	if status := c.Query("status"); status != "" {
		filters.Status = status
	}

	// Add check-in enabled filter
	if checkIn := c.Query("check_in_enabled"); checkIn != "" {
		val := checkIn == "true" || checkIn == "1"
		filters.CheckInEnabled = &val
	}

	// Add free-text search across denormalized employee fields
	filters.Search = c.Query("search")

	// For partner users, check if they have limited employee-based access
	userRole := c.GetString(constants.CtxUserRole)
	var accessibleEmployeeIDs map[uint]bool // Use map for O(1) lookup
	var hasLimitedAccess bool

	if userRole == string(domain.RolePartner) {
		userID, exists := c.Get(constants.CtxUserID)
		if !exists {
			response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
			return
		}

		uid, ok := userID.(uint)
		if !ok {
			response.Forbidden(c, constants.MsgInvalidUserIDVN)
			return
		}

		// Check if partner has full project access (creator or explicit project_users access)
		// If they do, they can see all employees. If not, they only have employee-based access.
		canModify, err := h.projectPermissionService.CanUserModifyProject(c.Request.Context(), projectIDUint, uid)
		if err != nil {
			h.logger.ErrorContext(c.Request.Context(), "Failed to check project modify permission",
				"project_id", projectIDUint,
				"user_id", uid,
				"error", err,
			)
			response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
			return
		}

		// If user doesn't have full project access, they only have employee-based access
		if !canModify {
			hasLimitedAccess = true
			// Build a set of accessible employee IDs
			accessibleEmployeeIDs = make(map[uint]bool)

			// Get employees this partner user has been granted access to
			grantedEmployeeIDs, err := h.employeePermissionService.GetAccessibleEmployeeIDs(c.Request.Context(), uid)
			if err != nil {
				h.logger.ErrorContext(c.Request.Context(), "Failed to get accessible employee IDs",
					"user_id", uid,
					"error", err,
				)
				response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
				return
			}

			for _, empID := range grantedEmployeeIDs {
				accessibleEmployeeIDs[empID] = true
			}
		}
	}

	// If user has limited access, we need to get all assignments and filter them
	// Otherwise, we can use pagination at the DB level
	if hasLimitedAccess {
		// Get all assignments for the project without pagination
		allFilters := filters
		allFilters.Limit = 10000 // Large limit to get all
		allFilters.Offset = 0

		assignments, err := h.projectEmployeeService.ListAssignments(c.Request.Context(), allFilters)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
			return
		}

		// Filter assignments to only those the user can access
		var filteredAssignments []*domain.ProjectEmployee
		for _, assignment := range assignments {
			if accessibleEmployeeIDs[assignment.EmployeeID] {
				filteredAssignments = append(filteredAssignments, assignment)
			}
		}

		// Apply pagination to filtered results
		total := int64(len(filteredAssignments))
		startIdx := (page - 1) * pageSize
		endIdx := startIdx + pageSize

		if startIdx >= len(filteredAssignments) {
			startIdx = len(filteredAssignments)
			endIdx = len(filteredAssignments)
		} else if endIdx > len(filteredAssignments) {
			endIdx = len(filteredAssignments)
		}

		paginatedAssignments := filteredAssignments[startIdx:endIdx]

		// Build response
		var assignmentResponses []dto.ProjectEmployeeResponse
		for _, assignment := range paginatedAssignments {
			assignmentResponses = append(assignmentResponses, dto.ProjectEmployeeResponse{
				ID:                    assignment.ID,
				ProjectID:             assignment.ProjectID,
				EmployeeID:            assignment.EmployeeID,
				EmployeeName:          assignment.EmployeeName,
				EmployeeCCCD:          assignment.EmployeeCCCD,
				EmployeeCode:          assignment.EmployeeCode,
				Position:              assignment.Position,
				StartDate:             assignment.StartDate,
				LastDate:              assignment.LastDate,
				CheckInEnabled:        assignment.CheckInEnabled,
				PendingCheckInEnabled: assignment.PendingCheckInEnabled,
				CheckInEffectiveFrom:  assignment.CheckInEffectiveFrom,
				CreatedBy:             assignment.CreatedBy,
				CreatedAt:             assignment.CreatedAt,
				UpdatedAt:             assignment.UpdatedAt,
			})
		}

		// Calculate pagination
		totalPages := (int(total) + pageSize - 1) / pageSize

		if len(assignmentResponses) == 0 {
			pagination := response.Pagination{
				Page:         page,
				PageSize:     pageSize,
				TotalPages:   totalPages,
				TotalRecords: int(total),
			}
			response.SuccessEmptyWithPagination(c, "No project employees found", pagination)
			return
		}

		pagination := response.Pagination{
			Page:         page,
			PageSize:     pageSize,
			TotalPages:   totalPages,
			TotalRecords: int(total),
		}

		message := "Project employees retrieved successfully"
		if page > 1 {
			message = "Page " + strconv.Itoa(page) + " of project employees retrieved successfully"
		}

		response.SuccessWithPagination(c, assignmentResponses, message, pagination)
		return
	}

	// User has full access - use normal pagination at DB level
	filters.Limit = pageSize
	filters.Offset = (page - 1) * pageSize

	assignments, err := h.projectEmployeeService.ListAssignments(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
		return
	}

	total, err := h.projectEmployeeService.CountAssignments(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountEmployeesVN)
		return
	}

	var assignmentResponses []dto.ProjectEmployeeResponse
	for _, assignment := range assignments {
		employeeName := assignment.Employee.Fullname
		if employeeName == "" {
			employeeName = assignment.EmployeeName
		}

		employeeCCCD := assignment.Employee.CCCD
		if employeeCCCD == "" {
			employeeCCCD = assignment.EmployeeCCCD
		}

		assignmentResponses = append(assignmentResponses, dto.ProjectEmployeeResponse{
			ID:                     assignment.ID,
			ProjectID:              assignment.ProjectID,
			EmployeeID:             assignment.EmployeeID,
			EmployeeName:           employeeName,
			EmployeeCCCD:           employeeCCCD,
			EmployeeCode:           assignment.EmployeeCode,
			Position:               assignment.Position,
			StartDate:              assignment.StartDate,
			LastDate:               assignment.LastDate,
			PaymentSchedule:        assignment.PaymentSchedule,
			PendingPaymentSchedule: assignment.PendingPaymentSchedule,
			ScheduleEffectiveFrom:  assignment.ScheduleEffectiveFrom,
			CheckInEnabled:         assignment.CheckInEnabled,
			PendingCheckInEnabled:  assignment.PendingCheckInEnabled,
			CheckInEffectiveFrom:   assignment.CheckInEffectiveFrom,
			CreatedBy:              assignment.CreatedBy,
			CreatedAt:              assignment.CreatedAt,
			UpdatedAt:              assignment.UpdatedAt,
		})
	}

	// Calculate pagination
	totalPages := (int(total) + pageSize - 1) / pageSize

	if len(assignmentResponses) == 0 {
		pagination := response.Pagination{
			Page:         page,
			PageSize:     pageSize,
			TotalPages:   totalPages,
			TotalRecords: int(total),
		}
		response.SuccessEmptyWithPagination(c, "No project employees found", pagination)
		return
	}

	pagination := response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	message := "Project employees retrieved successfully"
	if page > 1 {
		message = "Page " + strconv.Itoa(page) + " of project employees retrieved successfully"
	}

	response.SuccessWithPagination(c, assignmentResponses, message, pagination)
}

// RequestPaymentScheduleChange handles POST /api/v1/project-employees/:id/payment-schedule
func (h *Handler) RequestPaymentScheduleChange(c *gin.Context) {
	assignmentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidIDFormatVN)
		return
	}

	var req dto.UpdatePaymentScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	// Determine effective date automatically (first day of next month)
	now := h.clock.Now()
	effectiveDate := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())

	// Convert schedule string to domain type
	var newSchedule domain.PaymentSchedule
	switch req.NewSchedule {
	case string(domain.PaymentScheduleWeekly):
		newSchedule = domain.PaymentScheduleWeekly
	case string(domain.PaymentScheduleMonthly):
		newSchedule = domain.PaymentScheduleMonthly
	case string(domain.PaymentScheduleFlexible):
		newSchedule = domain.PaymentScheduleFlexible
	default:
		response.BadRequest(c, constants.MsgInvalidPaymentScheduleValueVN)
		return
	}

	// Get user ID from context
	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		response.Forbidden(c, constants.MsgInvalidUserIDVN)
		return
	}

	// Request the schedule change
	if err := h.projectEmployeeService.RequestPaymentScheduleChange(
		c.Request.Context(),
		uint(assignmentID),
		newSchedule,
		effectiveDate,
		uid,
	); err != nil {
		if domainErr, ok := err.(*domain.DomainError); ok {
			switch domainErr.Type {
			case "VALIDATION_ERROR":
				response.BadRequest(c, domainErr.Message)
			case "NOT_FOUND":
				response.NotFound(c, domainErr.Message)
			default:
				response.InternalServerError(c, constants.MsgFailedToRequestScheduleChangeVN)
			}
			return
		}
		response.InternalServerError(c, constants.MsgFailedToRequestScheduleChangeVN)
		return
	}

	response.Success(c, nil, "Payment schedule change requested successfully")
}

// GetPendingScheduleChanges handles GET /api/v1/project-employees/pending-schedule-changes
func (h *Handler) GetPendingScheduleChanges(c *gin.Context) {
	// Get all employees with pending schedule changes
	employees, err := h.projectEmployeeService.GetEmployeesWithPendingScheduleChanges(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToRetrievePendingSchedulesVN)
		return
	}

	var pendingChanges []dto.ProjectEmployeeResponse
	for _, emp := range employees {
		pendingChanges = append(pendingChanges, dto.ProjectEmployeeResponse{
			ID:                     emp.ID,
			ProjectID:              emp.ProjectID,
			EmployeeID:             emp.EmployeeID,
			EmployeeName:           emp.EmployeeName,
			EmployeeCCCD:           emp.EmployeeCCCD,
			EmployeeCode:           emp.EmployeeCode,
			Position:               emp.Position,
			StartDate:              emp.StartDate,
			LastDate:               emp.LastDate,
			PaymentSchedule:        string(emp.PaymentSchedule),
			PendingPaymentSchedule: stringPtr(string(*emp.PendingPaymentSchedule)),
			ScheduleEffectiveFrom:  emp.ScheduleEffectiveFrom,
			CheckInEnabled:         emp.CheckInEnabled,
			PendingCheckInEnabled:  emp.PendingCheckInEnabled,
			CheckInEffectiveFrom:   emp.CheckInEffectiveFrom,
			CreatedBy:              emp.CreatedBy,
			CreatedAt:              emp.CreatedAt,
			UpdatedAt:              emp.UpdatedAt,
		})
	}

	if len(pendingChanges) == 0 {
		response.SuccessEmpty(c, "No pending payment schedule changes found")
		return
	}

	response.Success(c, pendingChanges, "Pending payment schedule changes retrieved successfully")
}

// CancelPaymentScheduleChange handles DELETE /api/v1/project-employees/:id/payment-schedule
func (h *Handler) CancelPaymentScheduleChange(c *gin.Context) {
	assignmentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidIDFormatVN)
		return
	}

	// Get user ID from context
	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		response.Forbidden(c, constants.MsgInvalidUserIDVN)
		return
	}

	// Cancel the pending schedule change
	if err := h.projectEmployeeService.CancelPaymentScheduleChange(
		c.Request.Context(),
		uint(assignmentID),
		uid,
	); err != nil {
		if domainErr, ok := err.(*domain.DomainError); ok {
			switch domainErr.Type {
			case "VALIDATION_ERROR":
				response.BadRequest(c, domainErr.Message)
			case "NOT_FOUND":
				response.NotFound(c, domainErr.Message)
			default:
				response.InternalServerError(c, constants.MsgFailedToCancelScheduleChangeVN)
			}
			return
		}
		response.InternalServerError(c, constants.MsgFailedToCancelScheduleChangeVN)
		return
	}

	response.Success(c, nil, "Payment schedule change cancelled successfully")
}

// Helper function to convert string to pointer
func stringPtr(s string) *string {
	return &s
}

// ToggleCheckInEnabled handles PATCH /api/v1/projects/:id/employees/:employeeId/checkin-enabled
func (h *Handler) ToggleCheckInEnabled(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidIDFormatVN)
		return
	}

	employeeID, err := strconv.ParseUint(c.Param("employeeId"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidIDFormatVN)
		return
	}

	var req dto.ToggleCheckInEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Unauthorized(c, constants.MsgInvalidUserIDVN)
		return
	}
	uid, ok := userID.(uint)
	if !ok {
		response.Forbidden(c, constants.MsgInvalidUserIDVN)
		return
	}

	if err := h.projectEmployeeService.ToggleCheckInEnabled(c.Request.Context(), uint(projectID), uint(employeeID), req.CheckInEnabled, uid); err != nil {
		if domainErr, ok := err.(*domain.DomainError); ok {
			response.BadRequest(c, domainErr.Message)
			return
		}
		response.InternalServerError(c, "Failed to toggle check-in status")
		return
	}

	response.Success(c, nil, "Check-in status updated successfully")
}

// BulkToggleCheckInEnabled handles PATCH /api/v1/projects/:id/employees/checkin-enabled/bulk
func (h *Handler) BulkToggleCheckInEnabled(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidIDFormatVN)
		return
	}

	var req dto.BulkToggleCheckInEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Unauthorized(c, constants.MsgInvalidUserIDVN)
		return
	}
	uid, ok := userID.(uint)
	if !ok {
		response.Forbidden(c, constants.MsgInvalidUserIDVN)
		return
	}

	if err := h.projectEmployeeService.BulkToggleCheckInEnabled(c.Request.Context(), uint(projectID), req.EmployeeIDs, req.CheckInEnabled, uid); err != nil {
		if domainErr, ok := err.(*domain.DomainError); ok {
			response.BadRequest(c, domainErr.Message)
			return
		}
		response.InternalServerError(c, "Failed to bulk toggle check-in status")
		return
	}

	response.Success(c, nil, "Check-in statuses updated successfully")
}

// CancelPendingCheckInEnable handles DELETE /api/v1/projects/:id/employees/:employeeId/checkin-enabled
func (h *Handler) CancelPendingCheckInEnable(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidIDFormatVN)
		return
	}

	employeeID, err := strconv.ParseUint(c.Param("employeeId"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidIDFormatVN)
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Unauthorized(c, constants.MsgInvalidUserIDVN)
		return
	}
	uid, ok := userID.(uint)
	if !ok {
		response.Forbidden(c, constants.MsgInvalidUserIDVN)
		return
	}

	assignment, err := h.projectEmployeeService.GetAssignmentByProjectAndEmployee(c.Request.Context(), uint(projectID), uint(employeeID))
	if err != nil {
		if domainErr, ok := err.(*domain.DomainError); ok && domainErr.Type == "NOT_FOUND" {
			response.NotFound(c, domainErr.Message)
			return
		}
		response.InternalServerError(c, "Failed to find assignment")
		return
	}
	if assignment == nil {
		response.NotFound(c, "Không tìm thấy phân công của nhân viên trong dự án")
		return
	}

	if err := h.projectEmployeeService.CancelPendingCheckInEnable(c.Request.Context(), assignment.ID, uid); err != nil {
		if domainErr, ok := err.(*domain.DomainError); ok {
			response.BadRequest(c, domainErr.Message)
			return
		}
		response.InternalServerError(c, "Failed to cancel pending check-in enable")
		return
	}

	response.Success(c, nil, "Đã hủy yêu cầu bật chấm công đang chờ kích hoạt")
}
