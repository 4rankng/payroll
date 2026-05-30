package disbursement

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain/ports/infrastructure"
	"api-server/internal/transport/http/response"
)

const (
	reconDateLayout   = "02/01/2006"
	reconMaxRangeDays = 31
)

// ReconciliationExportHandler exposes a single synchronous endpoint that
// proxies the admin "Tải file đối soát" download through the active
// disbursement provider's two-step export API. The endpoint blocks
// until the provider returns the CSV bytes (typically 1-3 seconds) —
// no queue, no polling, no DB row.
//
//	GET /api/v1/admin/manual-disbursement/reconciliation/download
//	    ?date_from=DD/MM/YYYY&date_to=DD/MM/YYYY
//
// Provider resolution goes through disbursement.Registry — the handler
// type-asserts the active provider against infrastructure.ReportExporter
// and refuses the request when the active provider doesn't implement
// the capability.
type ReconciliationExportHandler struct {
	registry *disbursement.Registry
	logger   *slog.Logger
}

// NewReconciliationExportHandler wires the handler. registry must be
// non-nil; provider availability is checked at request time so a
// dormant registry returns a clean 503 rather than a nil-route panic.
func NewReconciliationExportHandler(registry *disbursement.Registry, logger *slog.Logger) *ReconciliationExportHandler {
	if registry == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &ReconciliationExportHandler{registry: registry, logger: logger}
}

// Download handles GET /admin/manual-disbursement/reconciliation/download.
func (h *ReconciliationExportHandler) Download(c *gin.Context) {
	dateFromStr := c.Query("date_from")
	dateToStr := c.Query("date_to")
	if dateFromStr == "" || dateToStr == "" {
		response.BadRequest(c, "date_from và date_to là bắt buộc (DD/MM/YYYY)")
		return
	}
	from, err := time.ParseInLocation(reconDateLayout, dateFromStr, time.Local)
	if err != nil {
		response.BadRequest(c, "date_from phải có định dạng DD/MM/YYYY")
		return
	}
	to, err := time.ParseInLocation(reconDateLayout, dateToStr, time.Local)
	if err != nil {
		response.BadRequest(c, "date_to phải có định dạng DD/MM/YYYY")
		return
	}
	if from.After(to) {
		response.BadRequest(c, "date_from phải nhỏ hơn hoặc bằng date_to")
		return
	}
	if to.Sub(from) > time.Duration(reconMaxRangeDays-1)*24*time.Hour {
		response.BadRequest(c, fmt.Sprintf("Khoảng ngày tối đa là %d ngày", reconMaxRangeDays))
		return
	}

	ctx := c.Request.Context()

	provider, err := h.registry.Active(ctx)
	if err != nil {
		if errors.Is(err, disbursement.ErrNoActiveProvider) {
			response.InternalServerError(c, "Chưa cấu hình nhà cung cấp chi hộ")
			return
		}
		h.logger.Warn("reconciliation: active provider lookup failed", "error", err)
		response.InternalServerError(c, "Không thể tải báo cáo đối soát")
		return
	}

	exporter, ok := provider.(infrastructure.ReportExporter)
	if !ok {
		h.logger.Warn("reconciliation: active provider does not support export",
			"provider", provider.Name())
		response.BadRequest(c, "Nhà cung cấp hiện tại không hỗ trợ xuất báo cáo đối soát")
		return
	}

	h.logger.Info("reconciliation: requesting export",
		"provider", provider.Name(),
		"date_from", dateFromStr, "date_to", dateToStr)

	csvBytes, fileName, err := exporter.ExportReconciliation(ctx, from, to)
	if err != nil {
		h.logger.Warn("reconciliation: export failed",
			"provider", provider.Name(),
			"date_from", dateFromStr, "date_to", dateToStr,
			"error", err)
		response.InternalServerError(c, "Không thể tải báo cáo đối soát")
		return
	}
	if len(csvBytes) == 0 {
		response.InternalServerError(c, "Báo cáo đối soát trả về rỗng")
		return
	}

	if fileName == "" {
		fileName = fmt.Sprintf("reconciliation_%s_%s", from.Format("02012006"), to.Format("02012006"))
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, fileName))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", csvBytes)
}
