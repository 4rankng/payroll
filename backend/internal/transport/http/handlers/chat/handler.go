// Package chat is the HTTP transport for the MiMo assistant chat surface:
// conversations, paginated history, and message turns. Ownership is enforced
// in the service on every access; this layer only parses, maps, and renders.
package chat

import (
	"errors"
	"net/http"
	"strconv"

	"api-server/internal/app/services/llmchat"
	"api-server/internal/domain"
	llmport "api-server/internal/infra/llm"
	auditctx "api-server/internal/pkg/context"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// Handler serves the chat routes.
type Handler struct {
	service *llmchat.Service
}

func NewHandler(service *llmchat.Service) *Handler {
	return &Handler{service: service}
}

// callerFromRequest extracts the authenticated caller. The JWT role gates the
// surface (admin + partner only — adv_partner is excluded even though it reuses
// the admin layout).
func callerFromRequest(c *gin.Context) (llmchat.Caller, bool) {
	userID, ok := helpers.GetUserIDOrRespond(c)
	if !ok {
		return llmchat.Caller{}, false
	}
	role, ok := helpers.GetUserRoleOrRespond(c)
	if !ok {
		return llmchat.Caller{}, false
	}
	if !llmchat.RoleAllowed(role) {
		response.Forbidden(c, "Trợ lý AI chỉ dành cho quản trị viên và đối tác")
		return llmchat.Caller{}, false
	}
	return llmchat.Caller{
		UserID:   userID,
		Role:     role,
		FullName: auditctx.GetFullName(c.Request.Context()),
	}, true
}

// CreateConversation handles POST /api/v1/chat/conversations.
func (h *Handler) CreateConversation(c *gin.Context) {
	caller, ok := callerFromRequest(c)
	if !ok {
		return
	}
	var req createConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// An empty body is a valid "no title" create.
		req = createConversationRequest{}
	}
	conv, err := h.service.CreateConversation(c.Request.Context(), caller, req.Title)
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessCreated(c, newConversationResponse(conv), "Đã tạo hội thoại")
}

// ListConversations handles GET /api/v1/chat/conversations — the caller's own
// conversations only (admin included; cross-reads go through GET by id).
func (h *Handler) ListConversations(c *gin.Context) {
	caller, ok := callerFromRequest(c)
	if !ok {
		return
	}
	pagination := helpers.ParsePagination(c, 20)
	convs, total, err := h.service.ListConversations(c.Request.Context(), caller, pagination.Offset, pagination.Limit)
	if err != nil {
		writeError(c, err)
		return
	}
	items := make([]conversationResponse, 0, len(convs))
	for _, conv := range convs {
		items = append(items, newConversationResponse(conv))
	}
	response.SuccessWithPagination(c, items, "Danh sách hội thoại",
		helpers.CalculatePagination(pagination.Page, pagination.PageSize, total))
}

// GetConversation handles GET /api/v1/chat/conversations/:id — polling endpoint.
// Messages are paginated (messagePage / messagePageSize) — unbounded thread
// fetches are refused by construction.
func (h *Handler) GetConversation(c *gin.Context) {
	caller, ok := callerFromRequest(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	page, pageSize := messagePagination(c)
	offset := (page - 1) * pageSize

	view, err := h.service.GetConversation(c.Request.Context(), caller, id, offset, pageSize)
	if err != nil {
		writeError(c, err)
		return
	}

	data := conversationDetailResponse{
		Conversation:  newConversationResponse(view.Conversation),
		Messages:      make([]messageResponse, 0, len(view.Messages)),
		ToolCalls:     make([]toolCallResponse, 0, len(view.ToolCalls)),
		TotalMessages: view.TotalMessages,
	}
	for _, msg := range view.Messages {
		data.Messages = append(data.Messages, newMessageResponse(msg))
	}
	for _, call := range view.ToolCalls {
		data.ToolCalls = append(data.ToolCalls, newToolCallResponse(call))
	}
	response.SuccessWithPagination(c, data, "Chi tiết hội thoại",
		helpers.CalculatePagination(page, pageSize, view.TotalMessages))
}

// SendMessage handles POST /api/v1/chat/conversations/:id/messages — runs one
// assistant turn and returns the answer plus the tool-call trail.
func (h *Handler) SendMessage(c *gin.Context) {
	caller, ok := callerFromRequest(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Nội dung tin nhắn không hợp lệ")
		return
	}

	result, err := h.service.SendMessage(c.Request.Context(), caller, id, req.Content)
	if err != nil {
		writeError(c, err)
		return
	}

	data := turnResponse{
		Message:   newMessageResponse(result.AssistantMessage),
		ToolCalls: make([]toolCallResponse, 0, len(result.ToolCalls)),
		Timing: turnTimingResponse{
			DurationMS: result.Timing.DurationMS,
			ModelMS:    result.Timing.ModelMS,
			ToolMS:     result.Timing.ToolMS,
			RoundTrips: result.Timing.RoundTrips,
			ToolCalls:  result.Timing.ToolCalls,
			TimedOut:   result.Timing.TimedOut,
		},
	}
	for _, call := range result.ToolCalls {
		data.ToolCalls = append(data.ToolCalls, newToolCallResponse(call))
	}
	response.Success(c, data, "Đã trả lời")
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "Mã hội thoại không hợp lệ")
		return 0, false
	}
	return uint(id), true
}

// messagePagination parses the message page params with the same normalisation
// rules as helpers.ParsePagination (separate names so conversation and message
// pages can be fetched independently).
func messagePagination(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("messagePage", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(c.DefaultQuery("messagePageSize", "20"))
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

// writeError maps service failures to designed HTTP responses. LLM provider
// errors already carry stable Vietnamese copy (llm.APIError).
func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, llmchat.ErrForbiddenRole):
		response.Forbidden(c, "Trợ lý AI chỉ dành cho quản trị viên và đối tác")
	case errors.Is(err, llmchat.ErrChatDisabled):
		response.Forbidden(c, "Trợ lý AI đã bị tắt cho vai trò của bạn")
	case errors.Is(err, llmchat.ErrRateLimited):
		response.TooManyRequests(c, "Bạn gửi tin nhắn quá nhanh — vui lòng chờ ít giây rồi thử lại.", 30)
	case errors.Is(err, llmchat.ErrTurnTimeout):
		c.JSON(http.StatusGatewayTimeout, response.ErrorResponse{
			Status:     "error",
			Message:    "Trợ lý phản hồi quá lâu — thao tác đã dừng lại, vui lòng thử lại.",
			HTTPStatus: http.StatusGatewayTimeout,
			Code:       "CHAT_TURN_TIMEOUT",
		})
	default:
		var apiErr *llmport.APIError
		if errors.As(err, &apiErr) {
			status := http.StatusServiceUnavailable
			code := "LLM_UNAVAILABLE"
			switch apiErr.Kind {
			case llmport.KindRateLimited:
				status = http.StatusTooManyRequests
				code = "LLM_RATE_LIMITED"
			case llmport.KindInternal:
				status = http.StatusInternalServerError
				code = "LLM_ERROR"
			}
			c.JSON(status, response.ErrorResponse{
				Status:     "error",
				Message:    apiErr.Message,
				HTTPStatus: status,
				Code:       code,
			})
			return
		}
		var domainErr *domain.DomainError
		if errors.As(err, &domainErr) {
			response.HandleDomainError(c, err)
			return
		}
		response.InternalServerError(c, "Không thể xử lý yêu cầu chat — vui lòng thử lại sau.")
	}
}
