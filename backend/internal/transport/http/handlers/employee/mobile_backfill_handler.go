package employee

import (
	"context"
	"runtime/debug"

	"api-server/internal/app/services/employee"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/response"
	"api-server/internal/transport/http/uploadguard"

	"github.com/gin-gonic/gin"
)

// ImportMobiles receives the "thiếu số điện thoại" workbook back after the
// admin filled the Mobile column in: employees are matched by CCCD and only
// EMPTY mobile fields are written — an existing number is never overwritten.
// Synchronous by design: the monthly file is ~130 rows, so the response
// carries the full per-row summary directly, with no import job to poll.
//
// Admin-only, matching the sibling export-paid-without-mobile that produces
// the workbook in the first place.
func (h *Handler) ImportMobiles(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			observability.GetLogger().Error("panic in ImportMobiles",
				"panic", r,
				"stack", string(debug.Stack()),
			)
			response.InternalServerError(c, constants.MsgFailedToBackfillMobilesVN)
		}
	}()

	if userRole := c.GetString(constants.CtxUserRole); userRole != string(domain.RoleAdmin) {
		response.Forbidden(c, constants.MsgMobileBackfillAdminOnlyVN)
		return
	}

	// Guarded multipart receive: body cap, size cap, .xlsx magic sniff.
	body, filename, ok := uploadguard.Receive(c, false)
	if !ok {
		return
	}

	rows, parseErrors, err := employee.ParseMobileBackfillExcel(body)
	if err != nil {
		observability.GetLogger().Error("failed to parse mobile backfill workbook", "error", err, "filename", filename)
		response.BadRequest(c, constants.MsgFailedToParseMobileFileVN)
		return
	}

	result, err := h.employeeService.BackfillMobiles(c.Request.Context(), rows, parseErrors)
	if err != nil {
		observability.GetLogger().Error("failed to backfill employee mobiles", "error", err, "filename", filename)
		response.InternalServerError(c, constants.MsgFailedToBackfillMobilesVN)
		return
	}

	// Audit the import with the number of rows actually written, same shape
	// as the async employee import (nil-safe via the interface assert — the
	// handler's auditService field is intentionally loosely typed).
	if auditSvc, ok := h.auditService.(interface {
		LogFileImport(ctx context.Context, dataType string, recordCount int, fileName string) error
	}); ok && auditSvc != nil {
		_ = auditSvc.LogFileImport(c.Request.Context(), "employee_mobiles", result.Updated, filename)
	}

	response.Success(c, result, "Nhập số điện thoại hoàn tất")
}
