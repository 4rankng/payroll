package handlers

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/excel"
	"api-server/internal/constants"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// userRoleLabel mirrors the role badges on the Nguoi dung table so the export
// reads the same as the screen.
func userRoleLabel(role string) string {
	switch role {
	case "admin":
		return "Quản trị viên"
	case "partner":
		return "Quản lý"
	case "employee":
		return "Nhân viên"
	case "adv_partner":
		return "Quản lý ứng lương"
	case "accountant":
		return "Kế toán"
	default:
		return "Người dùng"
	}
}

// ExportUsers exports the current user list to a styled XLSX workbook. It
// mirrors the employee export path: same filters as GET /users (the "current
// list"), Vietnamese headers, auto-sized columns, an audit record, and an
// attachment download.
func (h *UserHandler) ExportUsers(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			observability.GetLogger().Error("panic in ExportUsers",
				"panic", r,
				"stack", string(debug.Stack()),
			)
			response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		}
	}()

	var req dto.ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	// The export is the current list, unbound by the UI pagination: page
	// through the same filtered query at the service's maximum page size.
	const maxPageSize = 500
	var rows []dto.UserResponse
	for req.Page = 1; ; req.Page++ {
		req.PageSize = maxPageSize
		users, total, err := h.userService.ListUsers(c.Request.Context(), req)
		if err != nil {
			response.InternalServerError(c, constants.MsgInternalServerErrorVN)
			return
		}
		rows = append(rows, users...)
		if len(users) == 0 || int64(len(rows)) >= total {
			break
		}
	}

	excelService := excel.NewExportService()
	f, err := excelService.CreateStyledWorkbook("Users")
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
		"Username",
		"Email",
		"Vai trò",
		"Hoạt động cuối",
		"Ngày tạo",
	}
	if err := excelService.WriteHeaders(f, "Users", headers); err != nil {
		response.InternalServerError(c, constants.MsgFailedToWriteExcelHeadersVN)
		return
	}

	for i, u := range rows {
		lastLogin := "Chưa đăng nhập"
		if u.LastLogin != nil {
			lastLogin = u.LastLogin.Format("02/01/2006 15:04")
		}
		rowData := []interface{}{
			i + 1,
			u.Fullname,
			u.Username,
			u.Email,
			userRoleLabel(u.Role),
			lastLogin,
			excelService.FormatDateValue(u.CreatedAt),
		}
		if err := excelService.WriteDataRow(f, "Users", i+2, rowData, []int{}); err != nil {
			response.InternalServerError(c, constants.MsgFailedToWriteExcelDataVN)
			return
		}
	}

	if err := excelService.AutoSizeColumns(f, "Users", headers, nil); err != nil {
		response.InternalServerError(c, constants.MsgFailedToAutoSizeColumnsVN)
		return
	}

	buffer, err := excelService.SaveToBuffer(f)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		return
	}

	filename := fmt.Sprintf("users_export_%s.xlsx", clock.Now().Format("2006-01-02"))

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(buffer)))

	// Emit audit event for file export (non-blocking). Use
	// context.Background() because c.Request.Context() is cancelled when the
	// handler returns, which would race with the goroutine.
	if auditSvc, ok := h.auditService.(interface {
		LogFileExport(ctx context.Context, dataType string, recordCount int, fileName string) error
	}); ok && auditSvc != nil {
		recordCount := len(rows)
		go func() {
			_ = auditSvc.LogFileExport(context.Background(), "users", recordCount, filename)
		}()
	}

	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer)
}
