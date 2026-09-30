package employee

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/excel"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ExportEmployeesPaidWithoutMobile exports employees who received salary (paid
// timesheets) or a completed FlexPay advance in the last N months but have no
// mobile number on file. Payment recency is the activity proxy: the last-paid
// columns let the admin judge how recently each unreachable employee was paid.
// Admin-only — no partner/accountant Casbin policy exists for this route.
func (h *Handler) ExportEmployeesPaidWithoutMobile(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			observability.GetLogger().Error("panic in ExportEmployeesPaidWithoutMobile",
				"panic", r,
				"stack", string(debug.Stack()),
			)
			response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		}
	}()

	if userRole := c.GetString(constants.CtxUserRole); userRole != string(domain.RoleAdmin) {
		response.Forbidden(c, constants.MsgExportAdminOnlyVN)
		return
	}

	var req dto.ExportEmployeesPaidWithoutMobileRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	// Business window uses the injected clock (ADR-006); [from, to] inclusive.
	now := h.clock.Now()
	from := now.AddDate(0, -req.Months, 0)
	to := now

	activities, err := h.employeeService.ListEmployeesPaidWithoutMobile(c.Request.Context(), from, to)
	if err != nil {
		observability.GetLogger().Error("failed to list paid employees without mobile", "error", err)
		response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
		return
	}

	excelService := excel.NewExportService()
	f, err := excelService.CreateStyledWorkbook("Employees")
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			observability.GetLogger().Error("Error closing Excel file", "error", err)
		}
	}()

	headers := []string{
		"STT",
		"Họ và tên",
		"CCCD",
		"Trạng thái",
		"Tổng lương đã nhận (VND)",
		"Số đợt trả lương",
		"Lần trả lương cuối",
		"Tổng tạm ứng đã nhận (VND)",
		"Số lần tạm ứng",
		"Lần tạm ứng cuối",
		"Hoạt động gần nhất",
		"Dự án",
	}

	if err := excelService.WriteHeaders(f, "Employees", headers); err != nil {
		response.InternalServerError(c, constants.MsgFailedToWriteExcelHeadersVN)
		return
	}

	data := make([][]interface{}, 0, len(activities))
	for i, a := range activities {
		data = append(data, []interface{}{
			i + 1,
			a.Fullname,
			a.CCCD,
			statusLabel(a.IsWorking),
			a.SalaryTotal,
			a.SalaryCount,
			excelService.FormatDate(a.LastSalaryPaidAt),
			a.AdvanceTotal,
			a.AdvanceCount,
			excelService.FormatDate(a.LastAdvancePaidAt),
			excelService.FormatDate(a.LastActivityAt),
			a.Projects,
		})
	}

	for i, rowData := range data {
		if err := excelService.WriteDataRow(f, "Employees", i+2, rowData, []int{}); err != nil {
			response.InternalServerError(c, constants.MsgFailedToWriteExcelDataVN)
			return
		}
	}

	if err := excelService.AutoSizeColumns(f, "Employees", headers, data); err != nil {
		response.InternalServerError(c, constants.MsgFailedToAutoSizeColumnsVN)
		return
	}

	buffer, err := excelService.SaveToBuffer(f)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		return
	}

	filename := fmt.Sprintf("employees_paid_without_mobile_%s.xlsx", h.clock.Now().Format("2006-01-02"))

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(buffer)))

	// Emit audit event for file export (non-blocking), same pattern as
	// ExportEmployees: background context so handler return doesn't cancel it.
	if auditSvc, ok := h.auditService.(interface {
		LogFileExport(ctx context.Context, dataType string, recordCount int, fileName string) error
	}); ok && auditSvc != nil {
		recordCount := len(activities)
		go func() {
			_ = auditSvc.LogFileExport(context.Background(), "employees_paid_without_mobile", recordCount, filename)
		}()
	}

	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer)
}

// statusLabel maps the working-assignment check to its Vietnamese label,
// matching the labels used on the admin Employees page.
func statusLabel(isWorking bool) string {
	if isWorking {
		return "Đang làm việc"
	}
	return "Chưa phân công"
}
