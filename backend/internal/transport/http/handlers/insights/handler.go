// Package insights is the HTTP transport for the daily ops digest: the
// admin-only Tổng Quan card reads GET /insights/digest; a dismiss stores the
// 7-day family suppression.
package insights

import (
	"net/http"
	"strconv"

	"api-server/internal/app/services/insights"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// Handler serves the insights routes.
type Handler struct {
	service *insights.Service
}

// NewHandler constructs the handler over the digest service.
func NewHandler(service *insights.Service) *Handler {
	return &Handler{service: service}
}

// GetDigest handles GET /api/v1/insights/digest?date=YYYY-MM-DD (today default).
func (h *Handler) GetDigest(c *gin.Context) {
	findings, err := h.service.GetDigest(c.Request.Context(), c.Query("date"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorResponse{
			Status:     "error",
			Message:    "Không thể tải bản tóm tắt — vui lòng thử lại.",
			HTTPStatus: http.StatusInternalServerError,
			Code:       "INSIGHTS_ERROR",
		})
		return
	}
	response.Success(c, findings, "Tóm tắt vận hành trong ngày")
}

// Dismiss handles POST /api/v1/insights/:id/dismiss.
func (h *Handler) Dismiss(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "Mã phát hiện không hợp lệ")
		return
	}
	if err := h.service.Dismiss(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorResponse{
			Status:     "error",
			Message:    "Không thể ẩn phát hiện — vui lòng thử lại.",
			HTTPStatus: http.StatusInternalServerError,
			Code:       "INSIGHTS_ERROR",
		})
		return
	}
	response.SuccessEmpty(c, "Đã ẩn phát hiện này trong 7 ngày")
}
