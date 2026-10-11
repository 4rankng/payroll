package approval

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	appapproval "api-server/internal/app/services/approval"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// Handler serves the approval routes.
type Handler struct {
	service *appapproval.Service
	logger  *slog.Logger
}

// NewHandler constructs the handler over the approval engine.
func NewHandler(service *appapproval.Service) *Handler {
	return &Handler{service: service, logger: observability.GetLogger().With("component", "approval-handler")}
}

// actorFromRequest extracts the authenticated caller.
func actorFromRequest(c *gin.Context) (appapproval.Actor, bool) {
	userID, ok := helpers.GetUserIDOrRespond(c)
	if !ok {
		return appapproval.Actor{}, false
	}
	role, ok := helpers.GetUserRoleOrRespond(c)
	if !ok {
		return appapproval.Actor{}, false
	}
	return appapproval.Actor{ID: userID, Role: role}, true
}

func pathID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "Mã yêu cầu không hợp lệ")
		return 0, false
	}
	return uint(id), true
}

// List handles GET /api/v1/approvals — admins see all, partners only their own.
func (h *Handler) List(c *gin.Context) {
	actor, ok := actorFromRequest(c)
	if !ok {
		return
	}
	pagination := helpers.ParsePagination(c, 20)
	filters := domain.ApprovalRequestFilters{
		Status: parseStatusFilter(c.Query("status")),
		Limit:  pagination.PageSize,
		Offset: pagination.Offset,
	}
	if t := c.Query("type"); t != "" {
		filters.Type = []domain.ApprovalRequestType{domain.ApprovalRequestType(t)}
	}
	items, total, err := h.service.List(c.Request.Context(), actor, filters)
	if err != nil {
		writeError(c, err)
		return
	}
	views := h.service.ViewsFor(c.Request.Context(), items)
	rendered := make([]requestResponse, 0, len(items))
	for _, item := range items {
		rendered = append(rendered, newRequestResponseWithView(item, views[item.ID]))
	}
	response.SuccessWithPagination(c, rendered, "Danh sách yêu cầu phê duyệt",
		helpers.CalculatePagination(pagination.Page, pagination.PageSize, total))
}

// Get handles GET /api/v1/approvals/:id — ownership enforced in the service.
func (h *Handler) Get(c *gin.Context) {
	actor, ok := actorFromRequest(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	req, err := h.service.Get(c.Request.Context(), id, actor)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, newRequestResponseWithView(req, h.service.ViewFor(c.Request.Context(), req)), "Chi tiết yêu cầu")
}

// File handles POST /api/v1/approvals — identifying params only.
func (h *Handler) File(c *gin.Context) {
	actor, ok := actorFromRequest(c)
	if !ok {
		return
	}
	var body fileRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "Thiếu loại yêu cầu (type)")
		return
	}
	requestType := domain.ApprovalRequestType(body.Type)
	if !domain.ValidApprovalRequestType(requestType) {
		response.BadRequest(c, "Loại yêu cầu không hợp lệ")
		return
	}
	params, err := marshalParams(body.Params)
	if err != nil {
		response.BadRequest(c, "Tham số yêu cầu không hợp lệ")
		return
	}
	req, err := h.service.File(c.Request.Context(), appapproval.CreateParams{
		Type:      requestType,
		Params:    params,
		ActorID:   actor.ID,
		ActorRole: actor.Role,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessCreated(c, newRequestResponseWithView(req, h.service.ViewFor(c.Request.Context(), req)), "Đã gửi yêu cầu phê duyệt")
}

// Approve handles POST /api/v1/approvals/:id/approve (admin-only).
func (h *Handler) Approve(c *gin.Context) {
	actor, ok := actorFromRequest(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	req, err := h.service.Approve(c.Request.Context(), id, actor)
	if err != nil {
		h.writeDecisionError(c, err, req)
		return
	}
	response.Success(c, newRequestResponseWithView(req, h.service.ViewFor(c.Request.Context(), req)), "Đã duyệt và thực thi yêu cầu")
}

// Reject handles POST /api/v1/approvals/:id/reject — "Thử lại ngày mai".
func (h *Handler) Reject(c *gin.Context) {
	actor, ok := actorFromRequest(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	var body decisionRequest
	_ = c.ShouldBindJSON(&body) // note optional
	req, err := h.service.Reject(c.Request.Context(), id, actor, body.Note)
	if err != nil {
		h.writeDecisionError(c, err, req)
		return
	}
	response.Success(c, newRequestResponseWithView(req, h.service.ViewFor(c.Request.Context(), req)), "Đã từ chối — sẽ chuẩn bị lại theo lịch")
}

// Cancel handles POST /api/v1/approvals/:id/cancel — terminal "Huỷ".
func (h *Handler) Cancel(c *gin.Context) {
	actor, ok := actorFromRequest(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	var body decisionRequest
	_ = c.ShouldBindJSON(&body)
	req, err := h.service.Cancel(c.Request.Context(), id, actor, body.Note)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, newRequestResponseWithView(req, h.service.ViewFor(c.Request.Context(), req)), "Đã huỷ yêu cầu")
}

func marshalParams(params map[string]any) ([]byte, error) {
	if params == nil {
		return []byte(`{}`), nil
	}
	return json.Marshal(params)
}

func parseStatusFilter(raw string) []domain.ApprovalRequestStatus {
	if raw == "" {
		return nil
	}
	out := make([]domain.ApprovalRequestStatus, 0, 4)
	for _, part := range splitComma(raw) {
		s := domain.ApprovalRequestStatus(part)
		for _, valid := range domain.ValidApprovalRequestStatuses {
			if s == valid {
				out = append(out, s)
				break
			}
		}
	}
	return out
}

func splitComma(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			if i > start {
				out = append(out, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// writeErrorReq is writeError for decision paths where the engine may return
// a nil request together with the error (e.g. not found): render the row only
// when it exists.

// writeError maps engine errors onto HTTP responses. extra (may be nil-valued
// response) lets a decision that executed-with-failure still return the row.
func writeError(c *gin.Context, err error, partial ...requestResponse) {
	switch {
	case errors.Is(err, appapproval.ErrAlreadyFiled):
		c.JSON(http.StatusConflict, response.ErrorResponse{
			Status:     "error",
			Message:    "Yêu cầu này đã được gửi trước đó — không tạo bản trùng.",
			HTTPStatus: http.StatusConflict,
			Code:       "APPROVAL_ALREADY_FILED",
		})
	case errors.Is(err, appapproval.ErrForbiddenRole):
		response.Forbidden(c, "Vai trò của bạn không được thực hiện hành động này")
	default:
		var de *domain.DomainError
		if errors.As(err, &de) {
			response.BadRequest(c, de.Error())
			return
		}
		h_logger(c).Error("approval handler: internal error", "error", err)
		c.JSON(http.StatusInternalServerError, response.ErrorResponse{
			Status:     "error",
			Message:    "Không thể xử lý yêu cầu — vui lòng thử lại.",
			HTTPStatus: http.StatusInternalServerError,
			Code:       "APPROVAL_ERROR",
		})
	}
}

func h_logger(c *gin.Context) *slog.Logger {
	return observability.GetLogger().With("component", "approval-handler")
}

// writeDecisionError renders a decision failure. The engine returns a nil
// request together with validation errors (e.g. deciding a closed request),
// so the view is computed only when the row exists — never eagerly on nil.
func (h *Handler) writeDecisionError(c *gin.Context, err error, req *domain.ApprovalRequest) {
	if req == nil {
		writeError(c, err)
		return
	}
	writeError(c, err, newRequestResponseWithView(req, h.service.ViewFor(c.Request.Context(), req)))
}
