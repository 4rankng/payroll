package project_employee

import (
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/project"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

const checkInConfigurationMonthsBack = 24

func resolveCheckInConfigurationMonth(value string, now time.Time) (time.Time, error) {
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return currentMonth, nil
	}

	selectedMonth, err := time.ParseInLocation("2006-01", trimmed, now.Location())
	if err != nil {
		return time.Time{}, errors.New("tháng điểm danh phải có định dạng YYYY-MM")
	}
	oldestMonth := currentMonth.AddDate(0, -checkInConfigurationMonthsBack, 0)
	if selectedMonth.After(currentMonth) || selectedMonth.Before(oldestMonth) {
		return time.Time{}, errors.New("tháng điểm danh nằm ngoài phạm vi cho phép")
	}
	return selectedMonth, nil
}

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

// requireProjectModifyAccess checks that the caller may modify per-employee
// configuration for the project: admin always, adv_partner only on projects
// they can modify, everyone else 403.
func (h *Handler) requireProjectModifyAccess(c *gin.Context, projectID uint, featureDenyMsg, projectDenyMsg, logAction string) bool {
	role := c.GetString(constants.CtxUserRole)
	if role == string(domain.RoleAdmin) {
		return true
	}
	if role != string(domain.RoleAdvPartner) {
		response.Forbidden(c, featureDenyMsg)
		return false
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return false
	}
	uid, ok := userID.(uint)
	if !ok {
		response.Forbidden(c, constants.MsgInvalidUserIDVN)
		return false
	}

	canModify, err := h.projectPermissionService.CanUserModifyProject(c.Request.Context(), projectID, uid)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to check "+logAction+" project permission",
			"project_id", projectID,
			"user_id", uid,
			"error", err,
		)
		response.InternalServerError(c, constants.MsgFailedToCheckProjectAccessVN)
		return false
	}
	if !canModify {
		response.Forbidden(c, projectDenyMsg)
		return false
	}
	return true
}

func (h *Handler) requireCheckInConfigurationAccess(c *gin.Context, projectID uint) bool {
	return h.requireProjectModifyAccess(c, projectID,
		"Bạn không có quyền cấu hình điểm danh",
		"Bạn không có quyền cấu hình điểm danh cho dự án này",
		"check-in")
}

// requireAdvanceToggleAccess gates the per-employee advance payment kill
// switch ("tạm ngừng ứng lương").
func (h *Handler) requireAdvanceToggleAccess(c *gin.Context, projectID uint) bool {
	return h.requireProjectModifyAccess(c, projectID,
		"Bạn không có quyền tạm ngừng ứng lương",
		"Bạn không có quyền tạm ngừng ứng lương cho dự án này",
		"advance toggle")
}

func (h *Handler) ListCheckInConfigurableProjects(c *gin.Context) {
	role := c.GetString(constants.CtxUserRole)
	if role != string(domain.RoleAdmin) && role != string(domain.RoleAdvPartner) {
		response.Forbidden(c, "Bạn không có quyền cấu hình điểm danh")
		return
	}

	filters := domain.ProjectFilters{
		ProjectStatus: []domain.ProjectStatus{domain.ProjectStatusRunning},
		Limit:         -1,
		SortBy:        "name",
		SortOrder:     "asc",
	}
	if role == string(domain.RoleAdvPartner) {
		userID, exists := c.Get(constants.CtxUserID)
		uid, ok := userID.(uint)
		if !exists || !ok {
			response.Forbidden(c, constants.MsgInvalidUserIDVN)
			return
		}
		filters.ModifiableBy = &uid
	}

	projects, err := h.projectService.ListProjects(c.Request.Context(), filters)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to list check-in configurable projects", "error", err)
		response.InternalServerError(c, constants.MsgFailedToListProjectsVN)
		return
	}

	result := make([]dto.CheckInConfigurableProjectResponse, 0, len(projects))
	for _, project := range projects {
		if !project.IsFlexible {
			continue
		}
		result = append(result, dto.CheckInConfigurableProjectResponse{
			ID:         project.ID,
			Name:       project.Name,
			Code:       project.Code,
			Status:     project.ProjectStatus,
			IsFlexible: project.IsFlexible,
		})
	}
	response.Success(c, result, "Check-in configurable projects retrieved successfully")
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

func (h *Handler) GetCheckInConfiguration(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidProjectIDVN)
		return
	}
	if !h.requireCheckInConfigurationAccess(c, uint(projectID)) {
		return
	}

	page := 1
	if value, parseErr := strconv.Atoi(c.Query("page")); parseErr == nil && value > 0 {
		page = value
	}
	pageSize := 20
	if value, parseErr := strconv.Atoi(c.Query("pageSize")); parseErr == nil && value > 0 && value <= 100 {
		pageSize = value
	}

	status := domain.CheckInConfigurationStatus(c.DefaultQuery("status", string(domain.CheckInConfigurationStatusEnabled)))
	switch status {
	case domain.CheckInConfigurationStatusAll,
		domain.CheckInConfigurationStatusEnabled,
		domain.CheckInConfigurationStatusActive,
		domain.CheckInConfigurationStatusInactive,
		domain.CheckInConfigurationStatusPending:
	default:
		response.BadRequest(c, "Trạng thái điểm danh không hợp lệ")
		return
	}
	selectedMonth, err := resolveCheckInConfigurationMonth(c.Query("month"), h.clock.Now())
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, monthStart, _, err := h.projectEmployeeService.GetCheckInConfiguration(
		c.Request.Context(),
		uint(projectID),
		status,
		c.Query("search"),
		page,
		pageSize,
		selectedMonth,
	)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to get check-in configuration",
			"project_id", projectID,
			"error", err,
		)
		response.InternalServerError(c, "Không thể tải cấu hình điểm danh")
		return
	}

	employees := make([]dto.CheckInConfigurationEmployeeResponse, 0, len(result.Employees))
	for _, employee := range result.Employees {
		employees = append(employees, dto.CheckInConfigurationEmployeeResponse{
			AssignmentID:         employee.AssignmentID,
			ProjectID:            employee.ProjectID,
			EmployeeID:           employee.EmployeeID,
			EmployeeName:         employee.EmployeeName,
			EmployeeCCCD:         employee.EmployeeCCCD,
			EmployeeCode:         employee.EmployeeCode,
			CheckInEnabled:       employee.CheckInEnabled,
			PendingCheckInEnable: employee.PendingCheckInEnable,
			CheckInEffectiveFrom: employee.CheckInEffectiveFrom,
			AttendanceCount:      employee.AttendanceCount,
			LastCheckInAt:        employee.LastCheckInAt,
		})
	}

	totalPages := (int(result.Total) + pageSize - 1) / pageSize
	response.Success(c, dto.CheckInConfigurationResponse{
		Employees: employees,
		Summary: dto.CheckInConfigurationSummaryResponse{
			Enabled:  result.Summary.Enabled,
			Active:   result.Summary.Active,
			Inactive: result.Summary.Inactive,
			Pending:  result.Summary.Pending,
		},
		Month: monthStart.Format("2006-01"),
		Pagination: dto.CheckInConfigurationPaginationResponse{
			Page:         page,
			PageSize:     pageSize,
			TotalPages:   totalPages,
			TotalRecords: int(result.Total),
		},
	}, "Đã tải cấu hình điểm danh")
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
	if !h.requireCheckInConfigurationAccess(c, uint(projectID)) {
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

// ToggleAdvanceRequestEnabled handles PATCH /api/v1/projects/:id/employees/:employeeId/advance-request-enabled
func (h *Handler) ToggleAdvanceRequestEnabled(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidIDFormatVN)
		return
	}
	if !h.requireAdvanceToggleAccess(c, uint(projectID)) {
		return
	}

	employeeID, err := strconv.ParseUint(c.Param("employeeId"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidIDFormatVN)
		return
	}

	var req dto.ToggleAdvanceRequestEnabledRequest
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

	if err := h.projectEmployeeService.ToggleAdvanceRequestEnabled(c.Request.Context(), uint(projectID), uint(employeeID), req.AdvanceRequestEnabled, uid); err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, err.Error())
			return
		}
		if domainErr, ok := err.(*domain.DomainError); ok {
			response.BadRequest(c, domainErr.Message)
			return
		}
		response.InternalServerError(c, "Failed to toggle advance request status")
		return
	}

	response.Success(c, nil, "Advance request status updated successfully")
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

func (h *Handler) DisableInactiveCheckInEmployees(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidProjectIDVN)
		return
	}
	if !h.requireCheckInConfigurationAccess(c, uint(projectID)) {
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

	disabledCount, err := h.projectEmployeeService.DisableInactiveCheckInEmployees(
		c.Request.Context(),
		uint(projectID),
		uid,
	)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to disable inactive check-in employees",
			"project_id", projectID,
			"user_id", uid,
			"error", err,
		)
		response.InternalServerError(c, "Không thể tắt điểm danh cho nhân viên chưa điểm danh")
		return
	}

	response.Success(c, dto.DisableInactiveCheckInEmployeesResponse{
		DisabledCount: disabledCount,
	}, "Đã tắt điểm danh cho nhân viên chưa điểm danh")
}

func (h *Handler) DisablePendingCheckInEmployees(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidProjectIDVN)
		return
	}
	if !h.requireCheckInConfigurationAccess(c, uint(projectID)) {
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

	disabledCount, err := h.projectEmployeeService.DisablePendingCheckInEmployees(
		c.Request.Context(),
		uint(projectID),
		uid,
	)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to disable pending check-in employees",
			"project_id", projectID,
			"user_id", uid,
			"error", err,
		)
		response.InternalServerError(c, "Không thể hủy kích hoạt điểm danh đang chờ")
		return
	}

	response.Success(c, dto.DisablePendingCheckInEmployeesResponse{
		DisabledCount: disabledCount,
	}, "Đã hủy kích hoạt điểm danh đang chờ")
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
