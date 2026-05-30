package advance_payment

import (
	"fmt"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ExportFlexPayEmployees exports the flex pay employee list to Excel
// GET /api/v1/advance-payments/employees/export
func (h *AdvancePaymentHandler) ExportFlexPayEmployees(c *gin.Context) {
	forMonth := c.Query("forMonth")

	// If forMonth is provided, validate it
	if forMonth != "" {
		if len(forMonth) != 7 {
			response.BadRequest(c, "Tháng (forMonth) không hợp lệ với định dạng YYYY-MM")
			return
		}

		if _, err := time.Parse("2006-01", forMonth); err != nil {
			response.BadRequest(c, "Tháng (forMonth) không hợp lệ với định dạng YYYY-MM")
			return
		}
	}
	// Note: If forMonth is empty, service will use the latest available month from database

	ctx := c.Request.Context()
	logger := observability.GetLogger()

	f, err := h.service.ExportFlexPayEmployees(ctx, forMonth)
	if err != nil {
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		logger.Error("failed to export flex pay employees", "forMonth", forMonth, "error", err)
		response.InternalServerError(c, "Không thể xuất danh sách nhân viên")
		return
	}

	// Use the actual month for filename, or default if empty
	filenameMonth := forMonth
	if filenameMonth == "" {
		filenameMonth = "gan_nhat"
	}
	filename := fmt.Sprintf("danh_sach_nhan_vien_%s.xlsx", filenameMonth)

	// Emit audit event for file export (non-blocking)
	if h.auditService != nil {
		employeeCount := 0
		if rows, rowsErr := f.GetRows(f.GetSheetName(f.GetActiveSheetIndex())); rowsErr == nil && len(rows) > 0 {
			employeeCount = len(rows) - 1 // exclude header row
		}
		go func() {
			_ = h.auditService.LogFileExport(c.Request.Context(), "flexpay_employees", employeeCount, filename)
		}()
	}

	defer func() { _ = f.Close() }()

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	if _, err := f.WriteTo(c.Writer); err != nil {
		logger.Error("failed to write Excel response", "error", err)
	}
}
