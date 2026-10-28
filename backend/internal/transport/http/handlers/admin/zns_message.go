// Package admin holds admin-only HTTP handlers that don't fit the
// per-domain handler folders. The ZNS tracking list lives here: it is a
// read-only admin surface over the zns_messages audit table.
package admin

import (
	"strconv"
	"time"

	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ZnsMessageHandler exposes GET /api/v1/admin/zns-notifications — the admin
// ZNS tracking list over zns_messages (every ZNS the system sends, whatever
// feature). Casbin-protects to the admin role at the route level.
type ZnsMessageHandler struct {
	repo domain.ZnsMessageRepository
}

func NewZnsMessageHandler(repo domain.ZnsMessageRepository) *ZnsMessageHandler {
	return &ZnsMessageHandler{repo: repo}
}

// znsMessageResponse is the wire shape of one tracking row.
type znsMessageResponse struct {
	ID            uint   `json:"id"`
	CreatedAt     string `json:"created_at"`
	Feature       string `json:"feature"`
	TemplateID    string `json:"template_id"`
	Phone         string `json:"phone"`
	Status        string `json:"status"`
	ProviderMsgID string `json:"provider_msg_id"`
	ProviderCode  int    `json:"provider_code"`
	ErrorMsg      string `json:"error_msg"`
	DurationMs    int64  `json:"duration_ms"`
}

// List handles GET /api/v1/admin/zns-notifications.
//
// Query params (combinable): page, pageSize (alias page_size), status
// (repeatable, multi-select), feature, templateId, fromDate, toDate
// (YYYY-MM-DD, inclusive day bounds), q (phone or provider message id
// substring). Rows come back newest-first with the total matching count.
func (h *ZnsMessageHandler) List(c *gin.Context) {
	filter := domain.ZnsMessageFilter{
		Feature:    c.Query("feature"),
		TemplateID: c.Query("templateId"),
		Query:      c.Query("q"),
	}
	if v := c.Query("pageSize"); v != "" {
		filter.PageSize, _ = strconv.Atoi(v)
	}
	if v := c.Query("page_size"); v != "" {
		filter.PageSize, _ = strconv.Atoi(v)
	}
	if v := c.Query("page"); v != "" {
		filter.Page, _ = strconv.Atoi(v)
	}
	filter.Statuses = c.QueryArray("status")

	if v := c.Query("fromDate"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.FromDate = &t
		}
	}
	if v := c.Query("toDate"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.ToDate = &t
		}
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	rows, total, err := h.repo.ListAdmin(c.Request.Context(), filter)
	if err != nil {
		response.InternalServerError(c, "Không thể tải danh sách ZNS")
		return
	}

	items := make([]znsMessageResponse, len(rows))
	for i, r := range rows {
		items[i] = znsMessageResponse{
			ID:            r.ID,
			CreatedAt:     r.CreatedAt.Format(time.RFC3339),
			Feature:       r.Feature,
			TemplateID:    r.TemplateID,
			Phone:         r.Phone,
			Status:        r.Status,
			ProviderMsgID: r.ProviderMsgID,
			ProviderCode:  r.ProviderCode,
			ErrorMsg:      r.ErrorMsg,
			DurationMs:    r.DurationMs,
		}
	}

	totalPages := (int(total) + filter.PageSize - 1) / filter.PageSize
	response.SuccessWithPagination(c, items, "Lấy danh sách ZNS thành công", response.Pagination{
		Page:         filter.Page,
		PageSize:     filter.PageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	})
}
