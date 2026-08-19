package timesheet

import (
	"context"
	"fmt"
	"net/http"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/excel"
	"api-server/internal/constants"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ExportTimesheets exports timesheets for a date range to Excel format
func (h *Handler) ExportTimesheets(c *gin.Context) {
	var req dto.ExportTimesheetsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	if err := req.ParseProjectIDs(); err != nil {
		response.BadRequest(c, fmt.Sprintf("Invalid project IDs: %v", err))
		return
	}

	if err := req.ParseStatus(); err != nil {
		response.BadRequest(c, fmt.Sprintf("Invalid status: %v", err))
		return
	}

	if err := req.ValidateDateRange(); err != nil {
		response.BadRequest(c, fmt.Sprintf("Invalid date range: %v", err))
		return
	}

	fromDate, toDate, err := req.GetParsedDates()
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidDateFormatVN)
		return
	}

	var statusFilter []domain.TimesheetStatus
	if len(req.Statuses) > 0 {
		for _, status := range req.Statuses {
			statusFilter = append(statusFilter, domain.TimesheetStatus(status))
		}
	}

	filters := domain.TimesheetFilters{
		FromDate:        &fromDate,
		ToDate:          &toDate,
		ProjectIDs:      req.ProjectIDs,
		EmployeeID:      req.EmployeeID,
		TimesheetStatus: statusFilter,
		Limit:           10000,
		Offset:          0,
		SortBy:          "date",
		SortOrder:       "asc",
	}

	timesheets, err := h.timesheetService.ListTimesheets(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToGetTimesheetsVN)
		return
	}

	headers := []string{
		"STT", "Ngày công", "Họ tên", "Mã NV", "Dự án", "Ca làm việc",
		"Loại thanh toán", "Mức lương", "Tổng tiền", "Tiền đã trả", "Trạng thái", "Duyệt bởi", "Ngày duyệt",
	}
	currencyColumns := []int{7, 8, 9}

	data := h.buildTimesheetExportData(timesheets)
	filename := fmt.Sprintf("bang_cong_%s_%s.xlsx", req.FromDate, req.ToDate)

	if auditSvc, ok := h.auditService.(interface {
		LogFileExport(ctx context.Context, dataType string, recordCount int, fileName string) error
	}); ok && auditSvc != nil {
		go func() {
			_ = auditSvc.LogFileExport(c.Request.Context(), "timesheets", len(timesheets), filename)
		}()
	}

	h.generateExcelResponse(c, "Timesheets", headers, data, currencyColumns, filename)
}

func translateStatusToVietnamese(status string) string {
	statusTranslations := map[string]string{
		"pending_approval": "Chờ phê duyệt",
		"approved":         "Đã phê duyệt",
		"rejected":         "Bị từ chối",
		"pending":          "Chờ thanh toán",
		"paid":             "Đã thanh toán",
		"failed":           "Thanh toán thất bại",
		"cancelled":        "Đã hủy",
	}

	if translated, exists := statusTranslations[status]; exists {
		return translated
	}
	return status
}

// PayrollReportExport exports payroll report by project and salary period.
// Supports two modes:
//  1. fromDate + toDate (+ optional projectIds) — export by date range
//  2. atDate — existing behavior: auto-select projects by salary period
func (h *Handler) PayrollReportExport(c *gin.Context) {
	var req dto.PayrollReportByProjectRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	userID, userRole, ok := h.getUserContext(c)
	if !ok {
		return
	}

	if userRole != string(domain.RoleAdmin) && userRole != string(domain.RolePartner) {
		response.Forbidden(c, constants.MsgForbiddenVN)
		return
	}

	var reportData []*domainServices.ProjectReportData
	var buffer []byte
	var filename string

	if req.IsExportByProjectRange() {
		fromDate, toDate, err := req.GetParsedDateRange()
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		projectIDs, err := req.GetParsedProjectIDs()
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		// Partner: restrict to accessible projects only
		if userRole == string(domain.RolePartner) {
			projectIDs, err = h.filterProjectsForPartner(c.Request.Context(), userID, projectIDs)
			if err != nil {
				response.InternalServerError(c, constants.MsgFailedToCheckProjectAccessVN)
				return
			}
			if len(projectIDs) == 0 {
				response.Forbidden(c, constants.MsgForbiddenVN)
				return
			}
		}

		reportData, err = h.payrollReportByProjectService.GetPayrollReportForProjects(
			c.Request.Context(), projectIDs, fromDate, toDate,
		)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToGetPayrollReportDataVN)
			return
		}

		var buf []byte
		buf, _, err = h.payrollReportByProjectExporter.GenerateExcel(c.Request.Context(), reportData, fromDate)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToGeneratePayrollReportVN)
			return
		}
		buffer = buf
		filename = fmt.Sprintf("sao_ke_%s_%s.xlsx", fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))
	} else {
		if req.AtDate == "" {
			response.BadRequest(c, "atDate hoặc (fromDate, toDate) là bắt buộc")
			return
		}

		if err := req.ValidateDate(); err != nil {
			response.BadRequest(c, constants.MsgInvalidAtDateFormatVN)
			return
		}

		atDate, err := req.GetParsedDate()
		if err != nil {
			response.BadRequest(c, constants.MsgInvalidDateFormatVN)
			return
		}

		reportData, err = h.payrollReportByProjectService.GetProjectsForPayrollReport(c.Request.Context(), atDate)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToGetPayrollReportDataVN)
			return
		}

		var buf []byte
		buf, _, err = h.payrollReportByProjectExporter.GenerateExcel(c.Request.Context(), reportData, atDate)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToGeneratePayrollReportVN)
			return
		}
		buffer = buf
		filename = fmt.Sprintf("sao_ke_tt_%s.xlsx", atDate.Format("2006-01-02"))
	}

	if auditSvc, ok := h.auditService.(interface {
		LogFileExport(ctx context.Context, dataType string, recordCount int, fileName string) error
	}); ok && auditSvc != nil {
		go func() {
			totalRecords := 0
			for _, project := range reportData {
				totalRecords += len(project.EmployeeData)
			}
			_ = auditSvc.LogFileExport(c.Request.Context(), "payroll_report_by_project", totalRecords, filename)
		}()
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(buffer)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer)
}

func (h *Handler) buildTimesheetExportData(timesheets []*domain.Timesheet) [][]interface{} {
	excelService := excel.NewExportService()
	var data [][]interface{}

	for i, ts := range timesheets {
		employeeName, employeeCode := h.extractEmployeeInfo(ts)
		projectName := h.extractProjectName(ts)
		approvedBy, approvedDate := h.extractApprovalInfo(ts, excelService)

		rowData := []interface{}{
			i + 1, excelService.FormatDateValue(ts.Date), employeeName, employeeCode,
			projectName, ts.HoursWorked, ts.PayType, ts.PayRate, ts.Amount,
			ts.PaidAmount, translateStatusToVietnamese(string(ts.PaymentStatus)), approvedBy, approvedDate,
		}
		data = append(data, rowData)
	}
	return data
}

func (h *Handler) generateExcelResponse(c *gin.Context, sheetName string, headers []string, data [][]interface{}, currencyColumns []int, filename string) {
	excelService := excel.NewExportService()
	f, err := excelService.CreateStyledWorkbook(sheetName)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			observability.GetLogger().Error("Error closing Excel file", "error", err)
		}
	}()

	if err := excelService.WriteHeaders(f, sheetName, headers); err != nil {
		response.InternalServerError(c, constants.MsgFailedToWriteExcelHeadersVN)
		return
	}

	for i, rowData := range data {
		rowNum := i + 2
		if err := excelService.WriteDataRow(f, sheetName, rowNum, rowData, currencyColumns); err != nil {
			response.InternalServerError(c, constants.MsgFailedToWriteExcelDataVN)
			return
		}
	}

	if err := excelService.AutoSizeColumns(f, sheetName, headers, data); err != nil {
		response.InternalServerError(c, constants.MsgFailedToAutoSizeColumnsVN)
		return
	}

	buffer, err := excelService.SaveToBuffer(f)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(buffer)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer)
}

func (h *Handler) extractEmployeeInfo(ts *domain.Timesheet) (string, string) {
	if ts.Employee != nil {
		return ts.Employee.FormattedFullname(), ts.Employee.CCCD
	}
	return "", ""
}

func (h *Handler) extractProjectName(ts *domain.Timesheet) string {
	if ts.Project != nil {
		return ts.Project.Name
	}
	return ""
}

func (h *Handler) extractApprovalInfo(ts *domain.Timesheet, excelService *excel.ExportService) (string, string) {
	approvedBy := ""
	if ts.ApprovedBy != nil && ts.ApprovedUser != nil {
		approvedBy = ts.ApprovedUser.Fullname
	}

	approvedDate := ""
	if ts.ApprovedAt != nil {
		approvedDate = excelService.FormatDate(ts.ApprovedAt)
	}

	return approvedBy, approvedDate
}
