package handlers

import (
	"strconv"
	"time"

	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// AuditHandler handles HTTP requests for audit log endpoints
type AuditHandler struct {
	auditRepo domain.AuditLogRepository
}

// NewAuditHandler creates a new AuditHandler
func NewAuditHandler(auditRepo domain.AuditLogRepository) *AuditHandler {
	return &AuditHandler{auditRepo: auditRepo}
}

// AuditLogResponse is the HTTP response shape for a single audit log entry.
// Metadata is kept as a raw JSON string so the frontend can parse it without
// any loss from Go's float64 JSON number coercion.
type AuditLogResponse struct {
	ID           uint    `json:"id"`
	UserID       uint    `json:"user_id"`
	UserFullname string  `json:"user_fullname"`
	UserUsername string  `json:"user_username"`
	UserRole     string  `json:"user_role"`
	Action       string  `json:"action"`
	EntityType   string  `json:"entity_type"`
	EntityID     *uint   `json:"entity_id"`
	Message      string  `json:"message"`
	IPAddress    *string `json:"ip_address"`
	Browser      *string `json:"browser"`
	Platform     *string `json:"platform"`
	Metadata     *string `json:"metadata"` // raw JSON string
	CreatedAt    string  `json:"created_at"`
}

func toAuditLogResponse(log *domain.AuditLog) AuditLogResponse {
	return AuditLogResponse{
		ID:           log.ID,
		UserID:       log.UserID,
		UserFullname: log.UserFullname,
		UserUsername: log.UserUsername,
		UserRole:     log.UserRole,
		Action:       string(log.Action),
		EntityType:   string(log.EntityType),
		EntityID:     log.EntityID,
		Message:      log.Message,
		IPAddress:    log.IPAddress,
		Browser:      log.Browser,
		Platform:     log.Platform,
		Metadata:     log.Metadata,
		CreatedAt:    log.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// ListAuditLogsRequest holds query parameters for listing audit logs
type ListAuditLogsRequest struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
	UserID    uint   `form:"userId"`
	FromDate  string `form:"fromDate"`
	ToDate    string `form:"toDate"`
	IPAddress string `form:"ipAddress"`
	SortBy    string `form:"sortBy"`
	SortOrder string `form:"sortOrder"`
}

// ListAuditLogs handles GET /audit/logs
// @Summary List audit logs
// @Description Get paginated audit logs with optional filters
// @Tags audit
// @Accept json
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param pageSize query int false "Page size (default 20, max 100)"
// @Param userId query int false "Filter by user ID"
// @Param action query []string false "Filter by action(s)"
// @Param entityType query []string false "Filter by entity type(s)"
// @Param fromDate query string false "Filter from date (RFC3339)"
// @Param toDate query string false "Filter to date (RFC3339)"
// @Param ipAddress query string false "Filter by IP address"
// @Param sortBy query string false "Sort field (default: created_at)"
// @Param sortOrder query string false "Sort order: asc or desc (default: desc)"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/audit/logs [get]
func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	var req ListAuditLogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}

	// Pagination defaults
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	filters := domain.AuditFilters{
		Limit:     pageSize,
		Offset:    (page - 1) * pageSize,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	// Optional user filter
	if req.UserID > 0 {
		filters.UserID = &req.UserID
	}

	// Multi-value action filter: ?action=CREATE&action=UPDATE
	for _, a := range c.QueryArray("action") {
		filters.Action = append(filters.Action, domain.AuditAction(a))
	}

	// Multi-value entity type filter: ?entityType=user&entityType=employee
	for _, et := range c.QueryArray("entityType") {
		filters.EntityType = append(filters.EntityType, domain.EntityType(et))
	}

	// Optional IP address filter
	if req.IPAddress != "" {
		filters.IPAddress = &req.IPAddress
	}

	// Date range filters
	if req.FromDate != "" {
		t, err := time.Parse(time.RFC3339, req.FromDate)
		if err != nil {
			// Try date-only format
			t, err = time.Parse("2006-01-02", req.FromDate)
			if err != nil {
				response.BadRequest(c, "Invalid fromDate format, use RFC3339 or YYYY-MM-DD")
				return
			}
		}
		filters.FromDate = &t
	}

	if req.ToDate != "" {
		t, err := time.Parse(time.RFC3339, req.ToDate)
		if err != nil {
			t, err = time.Parse("2006-01-02", req.ToDate)
			if err != nil {
				response.BadRequest(c, "Invalid toDate format, use RFC3339 or YYYY-MM-DD")
				return
			}
			// End of day for date-only input
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		filters.ToDate = &t
	}

	logs, err := h.auditRepo.List(c.Request.Context(), filters)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	total, err := h.auditRepo.Count(c.Request.Context(), filters)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	result := make([]AuditLogResponse, len(logs))
	for i, log := range logs {
		result[i] = toAuditLogResponse(log)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	response.SuccessWithPagination(c, result, "audit logs retrieved successfully", response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	})
}

// GetAuditLog handles GET /audit/logs/:id
// @Summary Get audit log detail
// @Description Get a single audit log entry by ID
// @Tags audit
// @Accept json
// @Produce json
// @Param id path int true "Audit log ID"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/audit/logs/{id} [get]
func (h *AuditHandler) GetAuditLog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "Invalid audit log ID")
		return
	}

	log, err := h.auditRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, toAuditLogResponse(log), "audit log retrieved successfully")
}
