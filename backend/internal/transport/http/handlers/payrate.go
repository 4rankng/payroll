package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/payroll"
	"api-server/internal/app/services/project"
	"api-server/internal/pkg/clock"
)

type PayrateHandler struct {
	payrateService           *payroll.PayrateService
	projectService           *project.ProjectService
	projectPermissionService *project.ProjectPermissionService
	clock                    clock.Clock
}

func NewPayrateHandler(
	payrateService *payroll.PayrateService,
	projectService *project.ProjectService,
	projectPermissionService *project.ProjectPermissionService,
	clk clock.Clock,
) *PayrateHandler {
	if clk == nil {
		clk = clock.New()
	}
	return &PayrateHandler{
		payrateService:           payrateService,
		projectService:           projectService,
		projectPermissionService: projectPermissionService,
		clock:                    clk,
	}
}

// Helper function to convert time.Time to string pointer for responses
func formatDateToString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	dateStr := t.Format("2006-01-02")
	return &dateStr
}

// earliestEffectiveFrom returns the day after the project's most recent paid
// timesheet work date — the earliest start date a payrate update may take.
// Returns "" when the project has no paid timesheets (nothing anchors the floor). The
// date is formatted in its stored location: converting to UTC first would
// shift it back a day under a loc=Local MySQL DSN.
func (h *PayrateHandler) earliestEffectiveFrom(ctx context.Context, projectID uint) string {
	latestPaid, err := h.payrateService.GetLatestPaidTimesheetDateForProject(ctx, projectID)
	if err != nil || latestPaid == nil {
		return ""
	}
	return latestPaid.AddDate(0, 0, 1).Format("2006-01-02")
}

// Start-date fields are never load-time locked: moving a start date later now
// splits the config (update-as-create), so any stored state is correctable via
// the normal edit flow. Completed/cancelled projects are rejected by the
// validate endpoint and the update handler itself.

func (h *PayrateHandler) CreatePayrate(c *gin.Context) {
	var req dto.CreatePayrateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	userRole, roleExists := c.Get("user_role")
	if !roleExists {
		response.Forbidden(c, constants.MsgUserRoleNotFoundInContextVN)
		return
	}

	// Check if project exists and validate status
	project, err := h.projectService.GetProject(c.Request.Context(), req.ProjectID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	if userRole.(string) == string(domain.RolePartner) {
		canModify, err := h.projectPermissionService.CanUserModifyProject(c.Request.Context(), req.ProjectID, userID.(uint))
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToCheckProjectAccessVN)
			return
		}
		if !canModify {
			response.Forbidden(c, constants.MsgForbiddenVN)
			return
		}
	}

	// Check if project status allows modifications
	if project.IsCompleted() || project.IsCancelled() {
		response.BadRequest(c, constants.MsgCannotCreatePayrateForProjectVN+": "+string(project.ProjectStatus))
		return
	}

	// Convert DTO to domain entity
	payrate, err := req.ToDomain()
	if err != nil {
		response.BadRequest(c, fmt.Sprintf("%s: %s", constants.MsgInvalidPayrateConfigurationVN, err.Error()))
		return
	}

	// Validate rate structure
	if err := payrate.Payrate.Validate(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Auto-resolve overlapping shifts (wider shift wins)
	resolved, err := payrate.Payrate.ResolveFlexibleOverlaps()
	if err == nil {
		payrate.Payrate = resolved
	}

	// Parse from date string to time.Time for database
	fromDate, err := time.ParseInLocation("2006-01-02", req.EffectiveFrom, time.Local)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidEffectiveFromFormatVN)
		return
	}

	// Parse to date if provided
	var toDate *time.Time
	if req.EffectiveTo != nil && *req.EffectiveTo != "" {
		parsedToDate, err := time.ParseInLocation("2006-01-02", *req.EffectiveTo, time.Local)
		if err != nil {
			response.BadRequest(c, constants.MsgInvalidToDateFormatVN)
			return
		}
		toDate = &parsedToDate
	}

	// Set additional fields on the converted payrate
	payrate.ProjectID = req.ProjectID
	payrate.FromDate = fromDate
	payrate.ToDate = toDate
	payrate.CreatedBy = userID.(uint)

	createdPayrate, err := h.payrateService.CreatePayrate(c.Request.Context(), payrate, userID.(uint), userRole.(string))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Convert dates back to string for response
	var toDateStr *string
	if createdPayrate.ToDate != nil {
		dateStr := createdPayrate.ToDate.Format("2006-01-02")
		toDateStr = &dateStr
	}

	payrateResponse := dto.PayrateResponse{
		ID:        createdPayrate.ID,
		ProjectID: createdPayrate.ProjectID,
		Rates:     createdPayrate.Payrate,
		FromDate:  createdPayrate.FromDate.Format("2006-01-02"),
		ToDate:    toDateStr,
		CreatedBy: createdPayrate.CreatedBy,
		CreatedAt: createdPayrate.CreatedAt,
		UpdatedAt: createdPayrate.UpdatedAt,
	}

	response.SuccessCreated(c, payrateResponse, constants.MsgPayrateCreatedSuccessfullyVN)
}

func (h *PayrateHandler) GetPayrate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidPayrateIDVN)
		return
	}

	payrate, err := h.payrateService.GetPayrate(c.Request.Context(), uint(id))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// IDOR guard: partners may only read payrates for projects they own or have
	// been granted access to. Admins bypass this check. Mirrors ListPayrates.
	userRole := c.GetString(constants.CtxUserRole)
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
		canAccess, err := h.projectPermissionService.CanUserAccessProject(c.Request.Context(), payrate.ProjectID, uid)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToCheckProjectAccessVN)
			return
		}
		if !canAccess {
			response.Forbidden(c, constants.MsgForbiddenVN)
			return
		}
	}

	payrateResponse := dto.PayrateResponse{
		ID:                    payrate.ID,
		ProjectID:             payrate.ProjectID,
		Rates:                 payrate.Payrate,
		FromDate:              payrate.FromDate.Format("2006-01-02"),
		ToDate:                formatDateToString(payrate.ToDate),
		CreatedBy:             payrate.CreatedBy,
		CreatedAt:             payrate.CreatedAt,
		UpdatedAt:             payrate.UpdatedAt,
		EarliestEffectiveFrom: h.earliestEffectiveFrom(c.Request.Context(), payrate.ProjectID),
		FromDateLocked:        false,
	}

	response.Success(c, payrateResponse, constants.MsgPayrateRetrievedSuccessfullyVN)
}

func (h *PayrateHandler) ListPayrates(c *gin.Context) {
	filters := domain.PayrateFilters{
		Limit:     10,
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

	pageSize := 10
	if ps := c.Query("pageSize"); ps != "" {
		if size, err := strconv.Atoi(ps); err == nil && size > 0 && size <= 100 {
			pageSize = size
		}
	}

	// Convert to offset/limit for repository
	filters.Limit = pageSize
	filters.Offset = (page - 1) * pageSize

	if projectID := c.Query("project_id"); projectID != "" {
		if id, err := strconv.ParseUint(projectID, 10, 32); err == nil {
			uid := uint(id)
			filters.ProjectID = &uid
		}
	}

	// Partner role: payrates are project-level configuration (often created by
	// admins), so we gate by project access rather than creator ownership.
	// - For a specific project: require access to that project, then return all
	//   of its payrates (no CreatedBy filter) so partners see admin-created rates.
	// - For a global list (no project_id): fall back to payrates the partner created.
	userRole := c.GetString(constants.CtxUserRole)
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

		if filters.ProjectID != nil {
			canAccess, err := h.projectPermissionService.CanUserAccessProject(c.Request.Context(), *filters.ProjectID, uid)
			if err != nil {
				response.InternalServerError(c, constants.MsgFailedToCheckProjectAccessVN)
				return
			}
			if !canAccess {
				response.Forbidden(c, constants.MsgForbiddenVN)
				return
			}
			// Access granted: return all payrates for this project.
		} else {
			// No project scope: only show payrates the partner created.
			filters.CreatedBy = &uid
		}
	}

	payrates, err := h.payrateService.ListPayrates(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListPayratesVN)
		return
	}

	total, err := h.payrateService.CountPayrates(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountPayratesVN)
		return
	}

	var payrateResponses []dto.PayrateResponse
	for _, payrate := range payrates {
		payrateResponses = append(payrateResponses, dto.PayrateResponse{
			ID:                    payrate.ID,
			ProjectID:             payrate.ProjectID,
			Rates:                 payrate.Payrate,
			FromDate:              payrate.FromDate.Format("2006-01-02"),
			ToDate:                formatDateToString(payrate.ToDate),
			CreatedBy:             payrate.CreatedBy,
			CreatedAt:             payrate.CreatedAt,
			UpdatedAt:             payrate.UpdatedAt,
			EarliestEffectiveFrom: h.earliestEffectiveFrom(c.Request.Context(), payrate.ProjectID),
			FromDateLocked:        false,
		})
	}

	// Calculate pagination
	totalPages := (int(total) + pageSize - 1) / pageSize

	if len(payrateResponses) == 0 {
		pagination := response.Pagination{
			Page:         page,
			PageSize:     pageSize,
			TotalPages:   totalPages,
			TotalRecords: int(total),
		}
		response.SuccessEmptyWithPagination(c, constants.MsgNoPayratesFoundVN, pagination)
		return
	}

	pagination := response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	message := constants.MsgPayratesRetrievedSuccessfullyVN
	if page > 1 {
		message = fmt.Sprintf(constants.MsgPageOfPayratesRetrievedVN, page)
	}

	response.SuccessWithPagination(c, payrateResponses, message, pagination)
}

// UpdatePayrate updates an existing payrate
func (h *PayrateHandler) UpdatePayrate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidPayrateIDVN)
		return
	}

	var req dto.UpdatePayrateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	logger := observability.GetLogger()
	logger.Info("UpdatePayrate handler called",
		"payrate_id", uint(id),
		"user_id", userID.(uint),
		"effective_from", req.EffectiveFrom)

	// Get existing payrate
	payrate, err := h.payrateService.GetPayrate(c.Request.Context(), uint(id))
	if err != nil {
		logger.Error("Failed to get existing payrate",
			"payrate_id", uint(id),
			"error", err)
		response.HandleDomainError(c, err)
		return
	}

	// Role-based access control for partner users
	userRole := c.GetString(constants.CtxUserRole)
	if userRole == string(domain.RolePartner) && payrate.CreatedBy != userID {
		response.Forbidden(c, constants.MsgForbiddenVN)
		return
	}

	// Check if project exists and validate status using the existing payrate's project_id
	project, err := h.projectService.GetProject(c.Request.Context(), payrate.ProjectID)
	if err != nil {
		logger.Error("Failed to get project for payrate validation",
			"payrate_id", uint(id),
			"project_id", payrate.ProjectID,
			"error", err)
		response.HandleDomainError(c, err)
		return
	}

	// Check if project status allows modifications
	if project.IsCompleted() || project.IsCancelled() {
		logger.Warn("Attempted to update payrate for project with restricted status",
			"payrate_id", uint(id),
			"project_id", payrate.ProjectID,
			"project_status", project.ProjectStatus,
		)
		response.BadRequest(c, constants.MsgCannotUpdatePayrateForProjectVN+": "+string(project.ProjectStatus))
		return
	}

	logger.Info("Found existing payrate",
		"payrate_id", payrate.ID,
		"current_project_id", payrate.ProjectID,
		"from_date", payrate.FromDate.Format("2006-01-02"))

	// Convert DTO to get the payrate configuration
	updatedPayrate, err := req.ToDomain()
	if err != nil {
		logger.Error("Failed to convert DTO to domain",
			"payrate_id", uint(id),
			"error", err)
		response.BadRequest(c, fmt.Sprintf("%s: %s", constants.MsgInvalidPayrateConfigurationVN, err.Error()))
		return
	}

	// Validate rate structure
	if err := updatedPayrate.Payrate.Validate(); err != nil {
		logger.Error("Payrate validation failed",
			"payrate_id", uint(id),
			"error", err)
		response.BadRequest(c, err.Error())
		return
	}

	// Auto-resolve overlapping shifts (wider shift wins)
	resolved, resolveErr := updatedPayrate.Payrate.ResolveFlexibleOverlaps()
	if resolveErr == nil {
		updatedPayrate.Payrate = resolved
	}

	logger.Info("Payrate validation passed successfully",
		"payrate_id", uint(id))

	// Parse effective from date string to time.Time for database
	fromDate, err := time.ParseInLocation("2006-01-02", req.EffectiveFrom, time.Local)
	if err != nil {
		logger.Error("Invalid effective_from date format",
			"payrate_id", uint(id),
			"effective_from", req.EffectiveFrom,
			"error", err)
		response.BadRequest(c, constants.MsgInvalidEffectiveFromFormatVN)
		return
	}

	// Parse effective to date if provided
	var toDate *time.Time
	if req.EffectiveTo != nil && *req.EffectiveTo != "" {
		parsedToDate, err := time.ParseInLocation("2006-01-02", *req.EffectiveTo, time.Local)
		if err != nil {
			logger.Error("Invalid effective_to date format",
				"payrate_id", uint(id),
				"effective_to", *req.EffectiveTo,
				"error", err)
			response.BadRequest(c, constants.MsgInvalidToDateFormatVN)
			return
		}
		toDate = &parsedToDate
	}

	logger.Info("Date parsing successful",
		"payrate_id", uint(id),
		"from_date", fromDate.Format("2006-01-02"),
		"to_date", toDate)

	// Update fields (keep existing project_id, don't change it).
	// effective_to is not part of the model: create drops it at the service
	// level and configs close only via the timeline (update-as-create split /
	// predecessor sync / EndActivePayrateForProject). A supplied effective_to
	// is parsed above for a clear 400 on malformed input but deliberately
	// ignored — mirroring create. Persisting it would mark the config "ended"
	// by our own guard and brick it from further edits.
	payrate.Payrate = updatedPayrate.Payrate
	payrate.FromDate = fromDate

	logger.Info("Calling payrate service to update payrate",
		"payrate_id", payrate.ID)
	if err := h.payrateService.UpdatePayrate(c.Request.Context(), payrate, userID.(uint)); err != nil {
		logger.Error("Payrate service update failed",
			"payrate_id", payrate.ID,
			"error", err)
		response.HandleDomainError(c, err)
		return
	}

	logger.Info("Payrate update completed successfully",
		"payrate_id", payrate.ID)

	payrateResponse := dto.PayrateResponse{
		ID:        payrate.ID,
		ProjectID: payrate.ProjectID,
		Rates:     payrate.Payrate,
		FromDate:  payrate.FromDate.Format("2006-01-02"),
		ToDate:    formatDateToString(payrate.ToDate),
		CreatedBy: payrate.CreatedBy,
		CreatedAt: payrate.CreatedAt,
		UpdatedAt: payrate.UpdatedAt,
	}

	response.Success(c, payrateResponse, constants.MsgPayrateUpdatedSuccessfullyVN)
}

// DeletePayrate marks a payrate as inactive (soft delete)
func (h *PayrateHandler) DeletePayrate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidPayrateIDVN)
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Role-based access control for partner users
	userRole := c.GetString(constants.CtxUserRole)
	if userRole == string(domain.RolePartner) {
		// Get payrate to check ownership
		payrate, err := h.payrateService.GetPayrate(c.Request.Context(), uint(id))
		if err != nil {
			response.HandleDomainError(c, err)
			return
		}
		if payrate.CreatedBy != userID {
			response.Forbidden(c, constants.MsgForbiddenVN)
			return
		}
	}

	if err := h.payrateService.DeletePayrate(c.Request.Context(), uint(id), userID.(uint)); err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, nil, "Payrate deleted successfully")
}
