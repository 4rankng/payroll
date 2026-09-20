package settings

import (
	"strconv"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/config"
)

type Handler struct {
	settingsService *config.SettingsService
}

func NewHandler(settingsService *config.SettingsService) *Handler {
	return &Handler{
		settingsService: settingsService,
	}
}

// Helper function to convert setting to response DTO
func (h *Handler) buildSettingResponse(setting *domain.Settings) dto.SettingResponse {
	return dto.NewSettingResponse(setting)
}

// CreateSetting creates a new setting (Admin only)
// @Summary Create a new setting
// @Description Create a new system setting with the provided information
// @Tags settings
// @Accept json
// @Produce json
// @Param setting body dto.CreateSettingRequest true "Setting creation data"
// @Success 201 {object} dto.SettingResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/settings [post]
func (h *Handler) CreateSetting(c *gin.Context) {
	var req dto.CreateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, "User ID not found in context")
		return
	}

	setting := &domain.Settings{
		Key:       req.Key,
		Value:     req.Value,
		ValueType: domain.SettingsValueType(req.ValueType),
	}

	createdSetting, err := h.settingsService.CreateSetting(c.Request.Context(), setting, userID.(uint))
	if err != nil {
		if domain.IsConflictError(err) {
			response.Conflict(c, err.Error())
			return
		}
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, constants.MsgFailedToCreateSettingVN)
		return
	}

	response.SuccessCreated(c, h.buildSettingResponse(createdSetting), "Setting created successfully")
}

// GetSetting retrieves a setting by ID (Admin only)
// @Summary Get a setting by ID
// @Description Get detailed information about a specific setting
// @Tags settings
// @Produce json
// @Param id path int true "Setting ID"
// @Success 200 {object} dto.SettingResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/settings/{id} [get]
func (h *Handler) GetSetting(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidSettingIDVN)
		return
	}

	setting, err := h.settingsService.GetSetting(c.Request.Context(), uint(id))
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalServerError(c, constants.MsgFailedToGetSettingVN)
		return
	}

	response.Success(c, h.buildSettingResponse(setting), "Setting retrieved successfully")
}

// GetSettingByKey retrieves a setting by key (Admin only)
// @Summary Get a setting by key
// @Description Get detailed information about a specific setting by key
// @Tags settings
// @Produce json
// @Param key path string true "Setting key"
// @Success 200 {object} dto.SettingResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/settings/key/{key} [get]
func (h *Handler) GetSettingByKey(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.BadRequest(c, "Setting key is required")
		return
	}

	setting, err := h.settingsService.GetSettingByKey(c.Request.Context(), key)
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalServerError(c, constants.MsgFailedToGetSettingVN)
		return
	}

	response.Success(c, h.buildSettingResponse(setting), "Setting retrieved successfully")
}

// UpdateSetting updates a setting by ID (Admin only)
// @Summary Update a setting
// @Description Update an existing setting's information
// @Tags settings
// @Accept json
// @Produce json
// @Param id path int true "Setting ID"
// @Param setting body dto.UpdateSettingRequest true "Setting update data"
// @Success 200 {object} dto.SettingResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/settings/{id} [put]
func (h *Handler) UpdateSetting(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidSettingIDVN)
		return
	}

	var req dto.UpdateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Convert request to map for service
	updateData := make(map[string]any)
	if req.Key != nil {
		updateData["key"] = *req.Key
	}
	if req.Value != nil {
		updateData["value"] = *req.Value
	} else if req.Value == nil && c.Request.Header.Get("Content-Type") == "application/json" {
		// Handle explicit null value
		updateData["value"] = nil
	}
	if req.ValueType != nil {
		updateData["value_type"] = *req.ValueType
	}

	updatedSetting, err := h.settingsService.UpdateSetting(c.Request.Context(), uint(id), updateData, userID.(uint))
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, err.Error())
			return
		}
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		if domain.IsConflictError(err) {
			response.Conflict(c, err.Error())
			return
		}
		response.InternalServerError(c, constants.MsgFailedToUpdateSettingVN)
		return
	}

	response.Success(c, h.buildSettingResponse(updatedSetting), "Setting updated successfully")
}

// DeleteSetting deletes a setting by ID (Admin only)
// @Summary Delete a setting
// @Description Delete an existing setting (soft delete by setting status to inactive)
// @Tags settings
// @Produce json
// @Param id path int true "Setting ID"
// @Success 200 {object} dto.MessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/settings/{id} [delete]
func (h *Handler) DeleteSetting(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidSettingIDVN)
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	if err := h.settingsService.DeleteSetting(c.Request.Context(), uint(id), userID.(uint)); err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalServerError(c, constants.MsgFailedToDeleteSettingVN)
		return
	}

	response.Success(c, nil, "Setting deleted successfully")
}

// ListSettings lists settings with optional filtering (Admin only)
// @Summary List settings
// @Description Get a paginated list of settings with optional filtering
// @Tags settings
// @Produce json
// @Param limit query int false "Number of items per page" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Param search query string false "Search term for setting key"
// @Param status query string false "Filter by status" Enums(active, inactive)
// @Param value_type query string false "Filter by value type" Enums(string, number, boolean, json)
// @Param sort_by query string false "Field to sort by" default(updated_at)
// @Param sort_order query string false "Sort order" Enums(asc, desc) default(desc)
// @Success 200 {object} dto.ListSettingsResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/settings [get]
func (h *Handler) ListSettings(c *gin.Context) {
	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	search := c.Query("search")
	valueType := c.Query("value_type")
	sortBy := c.DefaultQuery("sort_by", "updated_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	// Build filters
	filters := domain.SettingsFilters{
		Search:    search,
		Limit:     limit,
		Offset:    offset,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	// Add value type filter if provided
	if valueType != "" {
		filters.ValueType = []domain.SettingsValueType{domain.SettingsValueType(valueType)}
	}

	settings, total, err := h.settingsService.ListSettings(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListSettingsVN)
		return
	}

	// Convert to response DTOs
	settingResponses := make([]dto.SettingResponse, len(settings))
	for i, setting := range settings {
		settingResponses[i] = h.buildSettingResponse(setting)
	}

	// Calculate pagination
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	page := (offset / limit) + 1

	response.SuccessWithPagination(c, settingResponses, "Settings retrieved successfully", response.Pagination{
		Page:         page,
		PageSize:     limit,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	})
}

// GetActiveSettings retrieves all active settings (Admin only)
// @Summary Get all active settings
// @Description Get all settings that are currently active
// @Tags settings
// @Produce json
// @Success 200 {object} dto.GetActiveSettingsResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/settings/active [get]
func (h *Handler) GetActiveSettings(c *gin.Context) {
	settings, _, err := h.settingsService.ListSettings(c.Request.Context(), domain.SettingsFilters{})
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToGetActiveSettingsVN)
		return
	}

	// Convert to response DTOs
	settingResponses := make([]dto.SettingResponse, len(settings))
	for i, setting := range settings {
		settingResponses[i] = h.buildSettingResponse(setting)
	}

	response.Success(c, dto.GetActiveSettingsResponse{
		Settings: settingResponses,
	}, "Active settings retrieved successfully")
}
