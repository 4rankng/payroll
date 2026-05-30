package employee

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	pkgConstants "api-server/internal/pkg/constants"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/utils"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func formatDateVN(t time.Time) string {
	return t.Format("02/01/2006")
}

func formatOptionalDateVN(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatDateVN(*t)
}

func paymentScheduleVN(schedule string) string {
	switch schedule {
	case "weekly":
		return "Lương tuần"
	case "monthly":
		return "Lương tháng"
	case "flexible":
		return "Linh động"
	default:
		return schedule
	}
}

// ExportEmployeeDetail exports a single employee's detail to Excel using the HoSoNhanSu template
func (h *Handler) ExportEmployeeDetail(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			observability.GetLogger().Error("panic in ExportEmployeeDetail",
				"panic", r,
				"stack", string(debug.Stack()),
			)
			response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		}
	}()

	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidEmployeeIDVN)
	if !ok {
		return
	}

	employee, err := h.employeeService.GetEmployee(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Get creator fullname — use helpers.GetUserID for safe type assertion (was: userID.(uint) panic vector)
	creatorName := ""
	if creatorID, err := helpers.GetUserID(c); err == nil {
		if user, err := h.employeeService.GetUserByID(c.Request.Context(), creatorID); err == nil && user != nil {
			creatorName = user.Fullname
		}
	}

	f, err := excelize.OpenFile(pkgConstants.EmployeeProfileTemplatePath)
	if err != nil {
		logger := observability.GetLogger()
		logger.Error("Failed to open employee profile template", "error", err)
		response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		return
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			observability.GetLogger().Error("Failed to close Excel file", "error", closeErr)
		}
	}()

	sheetName := "Sheet1"
	if len(f.GetSheetList()) > 0 {
		sheetName = f.GetSheetList()[0]
	}

	// B3: Người tạo
	if creatorName != "" {
		_ = f.SetCellValue(sheetName, "B3", fmt.Sprintf("Người tạo: %s", creatorName))
	}

	// Personal info
	_ = f.SetCellValue(sheetName, "C6", employee.Fullname)
	_ = f.SetCellValue(sheetName, "E6", employee.CCCD)

	if employee.User != nil {
		_ = f.SetCellValue(sheetName, "C7", employee.User.Username)
	} else if employee.UserID != nil {
		if user, err := h.employeeService.GetUserByID(c.Request.Context(), *employee.UserID); err == nil {
			_ = f.SetCellValue(sheetName, "C7", user.Username)
		}
	}

	_ = f.SetCellValue(sheetName, "E7", formatOptionalDateVN(employee.DateOfBirth))
	_ = f.SetCellValue(sheetName, "C8", employee.Mobile)
	_ = f.SetCellValue(sheetName, "E8", formatDateVN(employee.CreatedAt))
	_ = f.SetCellValue(sheetName, "C9", employee.Address)

	// Bank info
	if employee.Bank != nil {
		_ = f.SetCellValue(sheetName, "C12", employee.Bank.BranchName)
	}
	_ = f.SetCellValue(sheetName, "E12", employee.BankAccountNumber)
	_ = f.SetCellValue(sheetName, "C13", employee.BankAccountName)

	// Project info - fetch active assignments
	if h.projectEmployeeService != nil {
		assignments, err := h.projectEmployeeService.ListAssignments(c.Request.Context(), domain.ProjectEmployeeFilters{
			EmployeeID: &employee.ID,
			ActiveOnly: true,
			SortBy:     "created_at",
			SortOrder:  "desc",
		})
		if err == nil && len(assignments) > 0 {
			a := assignments[0]
			if a.Project.ID > 0 {
				_ = f.SetCellValue(sheetName, "C16", a.Project.Name)
				_ = f.SetCellValue(sheetName, "E16", a.Project.Code)
			}
			_ = f.SetCellValue(sheetName, "C17", a.Position)
			_ = f.SetCellValue(sheetName, "E17", paymentScheduleVN(a.PaymentSchedule))
			_ = f.SetCellValue(sheetName, "C18", formatDateVN(a.StartDate))
		}
	}

	buffer, err := f.WriteToBuffer()
	if err != nil {
		logger := observability.GetLogger()
		logger.Error("Failed to write Excel to buffer", "error", err)
		response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		return
	}

	// Simple ASCII filename: strip diacritics, remove spaces
	asciiName := strings.ReplaceAll(utils.NormalizeVietnamese(employee.Fullname), " ", "")
	if asciiName == "" {
		asciiName = "employee"
	}
	filename := fmt.Sprintf("%s.xlsx", asciiName)

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", buffer.Len()))

	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer.Bytes())
}
