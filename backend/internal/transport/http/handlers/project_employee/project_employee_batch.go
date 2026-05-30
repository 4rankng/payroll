package project_employee

import (
	"fmt"
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/project"
	"api-server/internal/constants"
	"api-server/internal/domain"
	pkgConstants "api-server/internal/pkg/constants"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) AssignEmployee(c *gin.Context) {
	h.logger.InfoContext(c.Request.Context(), "Starting employee assignment request",
		"project_id", c.Param("id"),
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
	)

	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "Invalid project ID format",
			"project_id", c.Param("id"),
			"error", err.Error(),
		)
		response.BadRequest(c, constants.MsgInvalidProjectIDVN)
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.ErrorContext(c.Request.Context(), "User ID not found in context")
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Check if project exists and validate status
	project, err := h.projectService.GetProject(c.Request.Context(), uint(projectID))
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Project not found or error accessing project",
			"project_id", projectID,
			"error", err.Error(),
		)
		response.HandleDomainError(c, err)
		return
	}

	// Check if project status allows modifications
	if project.IsCompleted() || project.IsCancelled() {
		h.logger.WarnContext(c.Request.Context(), "Attempted to assign employees to project with restricted status",
			"project_id", projectID,
			"project_status", project.ProjectStatus,
		)
		response.BadRequest(c, constants.MsgCannotAssignToProjectStatusVN+": "+string(project.ProjectStatus))
		return
	}

	h.logger.InfoContext(c.Request.Context(), "Processing employee assignment",
		"project_id", projectID,
		"user_id", userID,
	)

	// Parse request body as batch assignment array
	var batchReq dto.BatchAssignEmployeesRequest
	if err := c.ShouldBindJSON(&batchReq); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to parse request body as array",
			"error", err,
			"project_id", projectID,
		)
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	if len(batchReq) == 0 {
		h.logger.WarnContext(c.Request.Context(), "Empty assignment array received",
			"project_id", projectID,
		)
		response.BadRequest(c, constants.MsgAtLeastOneAssignmentRequiredVN)
		return
	}

	h.logger.InfoContext(c.Request.Context(), "Processing employee assignment(s)",
		"project_id", projectID,
		"employee_count", len(batchReq),
	)

	// Handle all assignments using batch handler (works for single or multiple)
	h.handleBatchAssignment(c, uint(projectID), batchReq, userID.(uint))
}

func (h *Handler) handleBatchAssignment(c *gin.Context, projectID uint, batchReq dto.BatchAssignEmployeesRequest, userID uint) {
	var assignments []*domain.ProjectEmployee

	for _, req := range batchReq {
		// Set default start date to today if not provided
		var startDate time.Time
		if req.StartDate != nil && *req.StartDate != "" {
			parsedDate, err := time.Parse("2006-01-02", *req.StartDate)
			if err != nil {
				h.logger.ErrorContext(c.Request.Context(), "Invalid start date format",
					"start_date", *req.StartDate,
					"employee_id", req.EmployeeID,
					"error", err,
				)
				response.BadRequest(c, fmt.Sprintf("Định dạng ngày bắt đầu không hợp lệ cho nhân viên %d: %s. Mong đợi YYYY-MM-DD", req.EmployeeID, *req.StartDate))
				return
			}
			startDate = parsedDate
		} else {
			startDate = h.clock.NowUTC().Truncate(24 * time.Hour) // Today at 00:00:00
		}

		// Parse end date if provided
		var endDate *time.Time
		if req.EndDate != nil && *req.EndDate != "" {
			parsedEndDate, err := time.Parse("2006-01-02", *req.EndDate)
			if err != nil {
				h.logger.ErrorContext(c.Request.Context(), "Invalid end date format",
					"end_date", *req.EndDate,
					"employee_id", req.EmployeeID,
					"error", err,
				)
				response.BadRequest(c, fmt.Sprintf("Định dạng ngày kết thúc không hợp lệ cho nhân viên %d: %s. Mong đợi YYYY-MM-DD", req.EmployeeID, *req.EndDate))
				return
			}
			endDate = &parsedEndDate
		}

		// Position is now required - no default value needed
		position := req.Position

		// Set employee code, will use CCCD as fallback via BeforeCreate hook if empty
		employeeCode := req.EmployeeCode

		// Set payment schedule with default to "weekly" if not provided
		paymentSchedule := pkgConstants.PaymentScheduleWeekly
		if req.PaymentSchedule != nil && *req.PaymentSchedule != "" {
			paymentSchedule = *req.PaymentSchedule
		}

		assignment := &domain.ProjectEmployee{
			ProjectID:       projectID,
			EmployeeID:      req.EmployeeID,
			EmployeeCode:    employeeCode,
			Position:        position,
			StartDate:       startDate,
			LastDate:        endDate,
			PaymentSchedule: paymentSchedule,
			CreatedBy:       userID,
		}

		assignments = append(assignments, assignment)
	}

	createdAssignments, err := h.projectEmployeeService.AssignEmployeesBatch(c.Request.Context(), assignments, userID)
	if err != nil {
		// Log detailed information about the failure
		h.logger.Error("Failed to assign employees to project",
			"project_id", projectID,
			"employee_count", len(assignments),
			"user_id", userID,
			"error", err,
			"error_type", fmt.Sprintf("%T", err),
		)

		// Log individual assignment attempts for debugging
		for i, assignment := range assignments {
			h.logger.Info("Assignment attempt details",
				"index", i,
				"employee_id", assignment.EmployeeID,
				"project_id", assignment.ProjectID,
				"start_date", assignment.StartDate,
				"end_date", assignment.LastDate,
			)
		}

		response.HandleDomainError(c, err)
		return
	}

	var assignmentResponses []dto.ProjectEmployeeAssignmentResponse
	for _, assignment := range createdAssignments {
		assignmentResponses = append(assignmentResponses, dto.ProjectEmployeeAssignmentResponse{
			ID:                     assignment.ID,
			ProjectID:              assignment.ProjectID,
			EmployeeID:             assignment.EmployeeID,
			EmployeeCode:           assignment.EmployeeCode,
			Position:               assignment.Position,
			StartDate:              assignment.StartDate,
			LastDate:               assignment.LastDate,
			PaymentSchedule:        assignment.PaymentSchedule,
			PendingPaymentSchedule: assignment.PendingPaymentSchedule,
			ScheduleEffectiveFrom:  assignment.ScheduleEffectiveFrom,
			CreatedBy:              assignment.CreatedBy,
			CreatedAt:              assignment.CreatedAt,
			UpdatedAt:              assignment.UpdatedAt,
		})
	}

	message := constants.MsgAssignmentSuccessVN
	if len(assignmentResponses) > 1 {
		message = fmt.Sprintf(constants.MsgMultipleAssignmentsSuccessVN, len(assignmentResponses))
	}

	response.SuccessCreated(c, assignmentResponses, message)
}

// RemoveEmployeesFromProject removes employees from a project
// @Summary Remove employees from project
// @Description Removes one or more employees from a project by setting a last date
// @Tags project-employees
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Param request body dto.BatchRemoveEmployeesRequest true "Remove employees request data"
// @Success 200 {object} []dto.ProjectEmployeeResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /projects/{id}/employees/remove [post]
func (h *Handler) RemoveEmployeesFromProject(c *gin.Context) {
	h.logger.InfoContext(c.Request.Context(), "Starting employee removal from project request",
		"project_id", c.Param("id"),
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
	)

	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "Invalid project ID format",
			"project_id", c.Param("id"),
			"error", err.Error(),
		)
		response.BadRequest(c, constants.MsgInvalidProjectIDVN)
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.ErrorContext(c.Request.Context(), "User ID not found in context")
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Check if project exists and validate status
	project, err := h.projectService.GetProject(c.Request.Context(), uint(projectID))
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Project not found or error accessing project",
			"project_id", projectID,
			"error", err.Error(),
		)
		response.HandleDomainError(c, err)
		return
	}

	// Check if project status allows modifications
	if project.IsCompleted() || project.IsCancelled() {
		h.logger.WarnContext(c.Request.Context(), "Attempted to remove employees from project with restricted status",
			"project_id", projectID,
			"project_status", project.ProjectStatus,
		)
		response.BadRequest(c, constants.MsgCannotRemoveEmployeesProjectStatusVN+string(project.ProjectStatus))
		return
	}

	// Parse request body as batch removal array
	var batchReq dto.BatchRemoveEmployeesRequest
	if err := c.ShouldBindJSON(&batchReq); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to parse request body as array",
			"error", err,
			"project_id", projectID,
		)
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	if len(batchReq) == 0 {
		h.logger.WarnContext(c.Request.Context(), "Empty removal array received",
			"project_id", projectID,
		)
		response.BadRequest(c, "At least one employee removal is required")
		return
	}

	h.logger.InfoContext(c.Request.Context(), "Processing employee removal(s)",
		"project_id", projectID,
		"employee_count", len(batchReq),
	)

	// Handle all removals using batch handler
	h.handleBatchRemoval(c, uint(projectID), batchReq, userID.(uint))
}

func (h *Handler) handleBatchRemoval(c *gin.Context, projectID uint, batchReq dto.BatchRemoveEmployeesRequest, userID uint) {
	// Convert DTO removals to service requests
	var removals []project.RemoveEmployeeRequest
	for _, req := range batchReq {
		// Parse last date if provided
		var lastDate *time.Time
		if req.LastDate != nil && *req.LastDate != "" {
			parsedLastDate, err := time.Parse("2006-01-02", *req.LastDate)
			if err != nil {
				h.logger.ErrorContext(c.Request.Context(), "Invalid last date format",
					"last_date", *req.LastDate,
					"employee_id", req.EmployeeID,
					"error", err,
				)
				response.BadRequest(c, fmt.Sprintf("Định dạng ngày kết thúc không hợp lệ cho nhân viên %d: %s. Mong đợi YYYY-MM-DD", req.EmployeeID, *req.LastDate))
				return
			}
			lastDate = &parsedLastDate
		}

		removals = append(removals, project.RemoveEmployeeRequest{
			EmployeeID: req.EmployeeID,
			LastDate:   lastDate,
		})
	}

	err := h.projectEmployeeService.RemoveEmployeesBatch(c.Request.Context(), projectID, removals, userID)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to remove employees from project",
			"project_id", projectID,
			"employee_count", len(removals),
			"user_id", userID,
			"error", err,
		)
		response.HandleDomainError(c, err)
		return
	}

	// Log successful removal for audit
	h.logger.InfoContext(c.Request.Context(), "Employees removed from project",
		"project_id", projectID,
		"employee_count", len(batchReq),
		"removed_by", userID,
	)

	// Return simple success response
	message := fmt.Sprintf("%d employee(s) removed from project successfully", len(batchReq))
	response.Success(c, nil, message)
}

// UpdateProjectEmployee updates an employee assignment in a project
func (h *Handler) UpdateProjectEmployee(c *gin.Context) {
	h.logger.InfoContext(c.Request.Context(), "Starting update project employee assignment request",
		"project_id", c.Param("id"),
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
	)

	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "Invalid project ID format",
			"project_id", c.Param("id"),
			"error", err.Error(),
		)
		response.BadRequest(c, constants.MsgInvalidProjectIDVN)
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.ErrorContext(c.Request.Context(), "User ID not found in context")
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Parse request body
	var req dto.UpdateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to parse request body",
			"error", err,
			"project_id", projectID,
		)
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	// Check if project exists and validate status
	project, err := h.projectService.GetProject(c.Request.Context(), uint(projectID))
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Project not found or error accessing project",
			"project_id", projectID,
			"error", err.Error(),
		)
		response.HandleDomainError(c, err)
		return
	}

	// Check if project status allows modifications
	if project.IsCompleted() || project.IsCancelled() {
		h.logger.WarnContext(c.Request.Context(), "Attempted to update employee assignment in project with restricted status",
			"project_id", projectID,
			"project_status", project.ProjectStatus,
		)
		response.BadRequest(c, constants.MsgCannotUpdateAssignmentsProjectStatusVN+string(project.ProjectStatus))
		return
	}

	// Find existing assignment
	assignment, err := h.projectEmployeeService.GetAssignmentByProjectAndEmployee(c.Request.Context(), uint(projectID), req.EmployeeID)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Assignment not found",
			"project_id", projectID,
			"employee_id", req.EmployeeID,
			"error", err.Error(),
		)
		response.HandleDomainError(c, err)
		return
	}

	// Update assignment fields
	if req.EmployeeCode != nil {
		assignment.EmployeeCode = *req.EmployeeCode
	}
	if req.Position != nil {
		assignment.Position = *req.Position
	}
	if req.StartDate != nil && *req.StartDate != "" {
		parsedDate, err := time.Parse("2006-01-02", *req.StartDate)
		if err != nil {
			h.logger.ErrorContext(c.Request.Context(), "Invalid start date format",
				"start_date", *req.StartDate,
				"error", err,
			)
			response.BadRequest(c, fmt.Sprintf("%s: %s. Mong đợi YYYY-MM-DD", constants.MsgInvalidStartDateFormatVN, *req.StartDate))
			return
		}
		assignment.StartDate = parsedDate
	}
	if req.EndDate != nil {
		if *req.EndDate == "" {
			assignment.LastDate = nil
		} else {
			parsedEndDate, err := time.Parse("2006-01-02", *req.EndDate)
			if err != nil {
				h.logger.ErrorContext(c.Request.Context(), "Invalid end date format",
					"end_date", *req.EndDate,
					"error", err,
				)
				response.BadRequest(c, fmt.Sprintf("%s: %s. Mong đợi YYYY-MM-DD", constants.MsgInvalidEndDateFormatVN, *req.EndDate))
				return
			}
			assignment.LastDate = &parsedEndDate
		}
	}
	if req.PaymentSchedule != nil {
		assignment.PaymentSchedule = *req.PaymentSchedule
	}

	// Update the assignment
	if err := h.projectEmployeeService.UpdateAssignment(c.Request.Context(), assignment, userID.(uint)); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to update project employee assignment",
			"assignment_id", assignment.ID,
			"project_id", projectID,
			"employee_id", req.EmployeeID,
			"error", err,
		)
		response.HandleDomainError(c, err)
		return
	}

	// Return updated assignment
	assignmentResponse := dto.ProjectEmployeeAssignmentResponse{
		ID:                     assignment.ID,
		ProjectID:              assignment.ProjectID,
		EmployeeID:             assignment.EmployeeID,
		EmployeeCode:           assignment.EmployeeCode,
		Position:               assignment.Position,
		StartDate:              assignment.StartDate,
		LastDate:               assignment.LastDate,
		PaymentSchedule:        assignment.PaymentSchedule,
		PendingPaymentSchedule: assignment.PendingPaymentSchedule,
		ScheduleEffectiveFrom:  assignment.ScheduleEffectiveFrom,
		CreatedBy:              assignment.CreatedBy,
		CreatedAt:              assignment.CreatedAt,
		UpdatedAt:              assignment.UpdatedAt,
	}

	h.logger.InfoContext(c.Request.Context(), "Successfully updated project employee assignment",
		"assignment_id", assignment.ID,
		"project_id", projectID,
		"employee_id", req.EmployeeID,
	)

	response.Success(c, assignmentResponse, "Project employee assignment updated successfully")
}
