// Package admin holds admin-only HTTP handlers that don't fit the
// per-domain handler folders. Cross-cutting tooling like the
// provider_transactions stats endpoint lives here.
package admin

import (
	"log/slog"
	"net/http"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// WalletPaymentStatsHandler exposes
// GET /api/v1/admin/provider-transactions/stats?from=&to=&group_by=
//
// Casbin-protected to admin role at the route level (see routes.go).
// Two group_by modes — error_code (default) and status. Date params
// are inclusive YYYY-MM-DD.
type WalletPaymentStatsHandler struct {
	stats  *disbursement.StatsService
	logger *slog.Logger
}

// NewWalletPaymentStatsHandler builds a handler over the stats
// service. stats may be nil — when it is, the route returns 503 so
// callers know the feature is dormant rather than getting a generic
// 404 from a missing route.
func NewWalletPaymentStatsHandler(stats *disbursement.StatsService, logger *slog.Logger) *WalletPaymentStatsHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &WalletPaymentStatsHandler{stats: stats, logger: logger}
}

// errorCodeStatsResponse mirrors the spec contract. The Groups slice
// is empty (not null) for empty windows so frontend code doesn't have
// to nil-check.
type errorCodeStatsResponse struct {
	From   string                        `json:"from"`
	To     string                        `json:"to"`
	Groups []disbursement.ErrorCodeGroup `json:"groups"`
}

type statusStatsResponse struct {
	From   string                     `json:"from"`
	To     string                     `json:"to"`
	Groups []disbursement.StatusGroup `json:"groups"`
}

// Stats handles GET /api/v1/admin/provider-transactions/stats.
//
// Casbin authorization is applied at the route level; this handler
// assumes the caller has already passed the admin gate.
func (h *WalletPaymentStatsHandler) Stats(c *gin.Context) {
	if h.stats == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "provider_transactions stats not configured",
		})
		return
	}

	from, to, err := disbursement.ParseDateRange(c.Query("from"), c.Query("to"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	groupBy := c.DefaultQuery("group_by", "error_code")
	const layout = "2006-01-02"

	switch groupBy {
	case "error_code":
		groups, err := h.stats.ByErrorCode(c.Request.Context(), from, to)
		if err != nil {
			h.logger.Error("provider_transactions stats by error_code failed", "error", err)
			response.InternalServerError(c, "failed to compute stats")
			return
		}
		response.Success(c, errorCodeStatsResponse{
			From:   from.Format(layout),
			To:     to.Format(layout),
			Groups: groups,
		}, "stats")
	case "status":
		groups, err := h.stats.ByStatus(c.Request.Context(), from, to)
		if err != nil {
			h.logger.Error("provider_transactions stats by status failed", "error", err)
			response.InternalServerError(c, "failed to compute stats")
			return
		}
		response.Success(c, statusStatsResponse{
			From:   from.Format(layout),
			To:     to.Format(layout),
			Groups: groups,
		}, "stats")
	default:
		response.BadRequest(c, "group_by must be one of: error_code, status")
	}
}
