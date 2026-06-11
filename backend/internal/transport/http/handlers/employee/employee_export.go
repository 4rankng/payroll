package employee

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/excel"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ExportEmployees exports all employees to Excel format
func (h *Handler) ExportEmployees(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			observability.GetLogger().Error("panic in ExportEmployees",
				"panic", r,
				"stack", string(debug.Stack()),
			)
			response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		}
	}()

	var req dto.ExportEmployeesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	filters := domain.EmployeeFilters{
		Limit:  10000,
		Offset: 0,
	}

	if err := req.ParseProjectIDs(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if len(req.ProjectIDs) > 0 {
		filters.ProjectIDs = req.ProjectIDs
	}

	// Role-based access control for partner users
	userRole := c.GetString(constants.CtxUserRole)
	if userRole == string(domain.RolePartner) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
			return
		}
		uid, ok := userID.(uint)
		if !ok {
			response.Forbidden(c, constants.MsgInvalidUserIDVN)
			return
		}
		filters.AccessibleBy = &uid
	}

	// Get all employees with their projects
	employees, err := h.employeeService.ListEmployeesWithAllProjects(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
		return
	}

	// Create Excel file
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

	// Define headers in Vietnamese as per specification
	headers := []string{
		"STT",
		"Username",
		"Họ và tên",
		"Email",
		"CCCD",
		"Mobile",
		"Địa chỉ",
		"Tên ngân hàng",
		"Số tài khoản ngân hàng",
		"Tên tài khoản ngân hàng",
		"Ngày sinh",
		"Dự án hiện tại",
		"Ngày tạo",
	}

	// Write headers
	if err := excelService.WriteHeaders(f, "Employees", headers); err != nil {
		response.InternalServerError(c, constants.MsgFailedToWriteExcelHeadersVN)
		return
	}

	// Prepare data
	var data [][]interface{}
	for i, emp := range employees {
		// Format username
		username := ""
		if emp.User != nil {
			username = emp.User.Username
		}

		// Format bank name
		bankName := ""
		if emp.Bank != nil {
			bankName = emp.Bank.BranchName
		}

		// Format email
		email := ""
		if emp.Email != nil {
			email = *emp.Email
		}

		// Format date of birth
		dateOfBirth := ""
		if emp.DateOfBirth != nil {
			dateOfBirth = excelService.FormatDate(emp.DateOfBirth)
		}

		// Format current projects
		var projectNames []string
		for _, project := range emp.CurrentProjects {
			projectNames = append(projectNames, project.Name)
		}
		currentProjects := strings.Join(projectNames, ", ")

		// Create row data with STT (row number) as first column
		rowData := []interface{}{
			i + 1, // STT - Row number starting from 1
			username,
			emp.FormattedFullname(),
			email,
			emp.CCCD,
			emp.Mobile,
			emp.Address,
			bankName,
			emp.BankAccountNumber,
			emp.BankAccountName,
			dateOfBirth,
			currentProjects,
			excelService.FormatDateValue(emp.CreatedAt),
		}

		data = append(data, rowData)
	}

	// Write data rows
	for i, rowData := range data {
		rowNum := i + 2 // Start from row 2 (row 1 is headers)
		if err := excelService.WriteDataRow(f, "Employees", rowNum, rowData, []int{}); err != nil {
			response.InternalServerError(c, constants.MsgFailedToWriteExcelDataVN)
			return
		}
	}

	// Auto-size columns
	if err := excelService.AutoSizeColumns(f, "Employees", headers, data); err != nil {
		response.InternalServerError(c, constants.MsgFailedToAutoSizeColumnsVN)
		return
	}

	// Save to buffer
	buffer, err := excelService.SaveToBuffer(f)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		return
	}

	// Generate filename with current date
	filename := fmt.Sprintf("employees_export_%s.xlsx", h.clock.Now().Format("2006-01-02"))

	// Set headers for XLSX file download
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(buffer)))

	// Emit audit event for file export (non-blocking)
	// Use context.Background() because c.Request.Context() is cancelled when the handler returns,
	// which would race with the goroutine and silently drop the audit log.
	if auditSvc, ok := h.auditService.(interface {
		LogFileExport(ctx context.Context, dataType string, recordCount int, fileName string) error
	}); ok && auditSvc != nil {
		recordCount := len(employees)
		go func() {
			_ = auditSvc.LogFileExport(context.Background(), "employees", recordCount, filename)
		}()
	}

	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer)
}
