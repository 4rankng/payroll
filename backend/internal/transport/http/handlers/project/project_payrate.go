package project

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/timeutil"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ListProjectPayrates gets payrates for a specific project
func (h *Handler) ListProjectPayrates(c *gin.Context) {
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

	// Parse query parameters
	projectIDPtr := uint(projectID)
	filters := domain.PayrateFilters{
		ProjectID: &projectIDPtr,
		Limit:     100, // default
		Offset:    0,   // default
		SortBy:    "from_date",
		SortOrder: "desc",
	}

	// Handle page/pageSize pagination (new model)
	page := 1
	if p := c.Query("page"); p != "" {
		if pageNum, err := strconv.Atoi(p); err == nil && pageNum > 0 {
			page = pageNum
		}
	}

	pageSize := 100
	if ps := c.Query("pageSize"); ps != "" {
		if size, err := strconv.Atoi(ps); err == nil && size > 0 && size <= 100 {
			pageSize = size
		}
	}

	// Convert to offset/limit for repository
	filters.Limit = pageSize
	filters.Offset = (page - 1) * pageSize

	if fromDate := c.Query("fromDate"); fromDate != "" {
		if date, err := time.Parse("2006-01-02", fromDate); err == nil {
			filters.FromDate = &date
		}
	}

	if toDate := c.Query("toDate"); toDate != "" {
		if date, err := time.Parse("2006-01-02", toDate); err == nil {
			filters.ToDate = &date
		}
	}

	// Get payrates using the injected payrate service
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

	// Convert payrates to response format
	var payrateResponses []dto.PayrateResponse
	for _, payrate := range payrates {
		var toDateStr *string
		if payrate.ToDate != nil {
			dateStr := payrate.ToDate.Format(timeutil.DateFormat)
			toDateStr = &dateStr
		}

		payrateResponses = append(payrateResponses, dto.PayrateResponse{
			ID:        payrate.ID,
			ProjectID: payrate.ProjectID,
			Rates:     payrate.Payrate,
			FromDate:  payrate.FromDate.Format(timeutil.DateFormat),
			ToDate:    toDateStr,
			CreatedBy: payrate.CreatedBy,
			CreatedAt: payrate.CreatedAt,
			UpdatedAt: payrate.UpdatedAt,
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
		response.SuccessEmptyWithPagination(c, constants.MsgNoPayratesFoundForProjectVN, pagination)
		return
	}

	pagination := response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	message := constants.MsgProjectPayratesRetrievedVN
	if page > 1 {
		message = fmt.Sprintf(constants.MsgPageOfProjectPayratesRetrievedVN, page)
	}

	response.SuccessWithPagination(c, payrateResponses, message, pagination)
}

// CreateProjectPayrate creates a new payrate for a project
func (h *Handler) CreateProjectPayrate(c *gin.Context) {
	logger := observability.GetLogger()

	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		logger.Error("Invalid project ID parameter",
			"project_id_param", c.Param("id"),
			"error", err.Error(),
			"endpoint", "CreateProjectPayrate",
		)
		response.BadRequest(c, constants.MsgInvalidProjectIDVN)
		return
	}

	logger.Info("Creating payrate for project",
		"project_id", projectID,
		"user_agent", c.GetHeader("User-Agent"),
	)

	// Check if project exists and validate status
	project, err := h.projectService.GetProject(c.Request.Context(), uint(projectID))
	if err != nil {
		logger.Error("Project not found or error accessing project",
			"project_id", projectID,
			"error", err.Error(),
		)
		response.HandleDomainError(c, err)
		return
	}

	// Check if project status allows modifications
	if project.IsCompleted() || project.IsCancelled() {
		logger.Warn("Attempted to create payrate for project with restricted status",
			"project_id", projectID,
			"project_status", project.ProjectStatus,
		)
		response.BadRequest(c, constants.MsgCannotCreatePayrateForProjectVN+": "+string(project.ProjectStatus))
		return
	}

	// Log the raw request body for debugging
	body, _ := c.GetRawData()
	logger.Info("Raw request body received",
		"project_id", projectID,
		"body_size", len(body),
		"body", string(body),
	)

	// Reset body for binding (GetRawData consumes it)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	var req dto.CreatePayrateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Failed to bind JSON request body",
			"error", err.Error(),
			"project_id", projectID,
			"endpoint", "CreateProjectPayrate",
			"raw_body", string(body),
		)
		response.BadRequest(c, fmt.Sprintf(constants.MsgInvalidRequestBodyFormatVN, err.Error()))
		return
	}

	logger.Info("Successfully parsed request body",
		"project_id", projectID,
		"rates_count", len(req.Rates),
		"effective_from", req.EffectiveFrom,
		"has_effective_to", req.EffectiveTo != nil,
	)

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	userRole, roleExists := c.Get(constants.CtxUserRole)
	if !roleExists {
		response.Forbidden(c, constants.MsgUserRoleNotFoundInContextVN)
		return
	}

	// Set project ID from URL
	req.ProjectID = uint(projectID)

	// Convert DTO to domain entity
	logger.Info("Converting DTO to domain entity",
		"project_id", projectID,
	)

	payrate, err := req.ToDomain()
	if err != nil {
		logger.Error("Failed to convert DTO to domain",
			"project_id", projectID,
			"error", err.Error(),
		)
		response.BadRequest(c, fmt.Sprintf("%s: %s", constants.MsgInvalidPayrateConfigurationVN, err.Error()))
		return
	}

	// Validate rate structure using basic JSON validation
	logger.Info("Validating payrate configuration",
		"project_id", projectID,
	)

	if err := payrate.Payrate.Validate(); err != nil {
		logger.Error("Payrate configuration validation failed",
			"project_id", projectID,
			"error", err.Error(),
		)
		response.BadRequest(c, fmt.Sprintf("%s: %s", constants.MsgInvalidPayrateConfigurationVN, err.Error()))
		return
	}

	// Parse from date string to time.Time for database
	logger.Info("Parsing effective dates",
		"project_id", projectID,
		"effective_from", req.EffectiveFrom,
		"effective_to", req.EffectiveTo,
	)

	fromDate, err := time.Parse(timeutil.DateFormat, req.EffectiveFrom)
	if err != nil {
		logger.Error("Failed to parse effective_from date",
			"project_id", projectID,
			"effective_from", req.EffectiveFrom,
			"expected_format", timeutil.DateFormat,
			"error", err.Error(),
		)
		response.BadRequest(c, fmt.Sprintf("Invalid effective_from format. Use %s format. Error: %s", timeutil.DateFormat, err.Error()))
		return
	}

	// Parse to date if provided
	var toDate *time.Time
	if req.EffectiveTo != nil && *req.EffectiveTo != "" {
		parsedToDate, err := time.Parse(timeutil.DateFormat, *req.EffectiveTo)
		if err != nil {
			response.BadRequest(c, constants.MsgInvalidEffectiveToFormatVN)
			return
		}
		toDate = &parsedToDate
	}

	// Set additional fields on the converted payrate
	payrate.FromDate = fromDate
	payrate.ToDate = toDate
	payrate.CreatedBy = userID.(uint)

	logger.Info("Creating payrate in service layer",
		"project_id", projectID,
		"user_id", userID.(uint),
		"user_role", userRole.(string),
		"from_date", fromDate.Format(timeutil.DateFormat),
		"to_date", func() string {
			if toDate != nil {
				return toDate.Format(timeutil.DateFormat)
			}
			return "null"
		}(),
	)

	createdPayrate, err := h.payrateService.CreatePayrate(c.Request.Context(), payrate, userID.(uint), userRole.(string))
	if err != nil {
		logger.Error("Failed to create payrate in service layer",
			"project_id", projectID,
			"user_id", userID.(uint),
			"error", err.Error(),
			"error_type", fmt.Sprintf("%T", err),
		)
		response.HandleDomainError(c, err)
		return
	}

	logger.Info("Successfully created payrate",
		"project_id", projectID,
		"payrate_id", createdPayrate.ID,
		"user_id", userID.(uint),
	)

	// Convert dates back to string for response
	var toDateStr *string
	if createdPayrate.ToDate != nil {
		dateStr := createdPayrate.ToDate.Format(timeutil.DateFormat)
		toDateStr = &dateStr
	}

	payrateResponse := dto.PayrateResponse{
		ID:        createdPayrate.ID,
		ProjectID: createdPayrate.ProjectID,
		Rates:     createdPayrate.Payrate,
		FromDate:  createdPayrate.FromDate.Format(timeutil.DateFormat),
		ToDate:    toDateStr,
		CreatedBy: createdPayrate.CreatedBy,
		CreatedAt: createdPayrate.CreatedAt,
		UpdatedAt: createdPayrate.UpdatedAt,
	}

	response.SuccessCreated(c, payrateResponse, constants.MsgProjectPayrateCreatedSuccessfullyVN)
}

// GetCurrentProjectPayrate gets the currently active or nearest upcoming payrate for a project
func (h *Handler) GetCurrentProjectPayrate(c *gin.Context) {
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

	// Get current or upcoming payrate
	payrate, err := h.payrateService.GetCurrentOrUpcomingPayrateByProject(c.Request.Context(), uint(projectID))
	if err != nil {
		// No payrate is not an error — return 200 with null data
		if domain.IsNotFoundError(err) {
			response.Success(c, nil, "No active payrate found for this project")
			return
		}
		response.HandleDomainError(c, err)
		return
	}

	// Convert to response format
	var toDateStr *string
	if payrate.ToDate != nil {
		dateStr := payrate.ToDate.Format(timeutil.DateFormat)
		toDateStr = &dateStr
	}

	payrateResponse := dto.PayrateResponse{
		ID:        payrate.ID,
		ProjectID: payrate.ProjectID,
		Rates:     payrate.Payrate,
		FromDate:  payrate.FromDate.Format(timeutil.DateFormat),
		ToDate:    toDateStr,
		CreatedBy: payrate.CreatedBy,
		CreatedAt: payrate.CreatedAt,
		UpdatedAt: payrate.UpdatedAt,
	}

	response.Success(c, payrateResponse, constants.MsgCurrentProjectPayrateRetrievedVN)
}
