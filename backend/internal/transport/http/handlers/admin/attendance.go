package admin

import (
	"log/slog"
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/attendance"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type AttendanceHandler struct {
	attendanceService *attendance.AttendanceService
	clk               clock.Clock
	logger            *slog.Logger
}

func NewAttendanceHandler(attendanceService *attendance.AttendanceService, clk clock.Clock, logger *slog.Logger) *AttendanceHandler {
	return &AttendanceHandler{
		attendanceService: attendanceService,
		clk:               clk,
		logger:            logger,
	}
}

// List handles GET /api/v1/admin/attendances
func (h *AttendanceHandler) List(c *gin.Context) {
	var filters domain.AttendanceFilters

	pg := helpers.ParsePagination(c, 50)
	filters.Limit = pg.Limit
	filters.Offset = pg.Offset

	if empIDStr := c.Query("employee_id"); empIDStr != "" {
		if empID, err := strconv.ParseUint(empIDStr, 10, 32); err == nil {
			id := uint(empID)
			filters.EmployeeID = &id
		}
	}
	if projIDStr := c.Query("project_id"); projIDStr != "" {
		if projID, err := strconv.ParseUint(projIDStr, 10, 32); err == nil {
			id := uint(projID)
			filters.ProjectID = &id
		}
	}
	if fromStr := c.Query("from_date"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			filters.FromDate = &t
		}
	}
	if toStr := c.Query("to_date"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			filters.ToDate = &t
		}
	}
	if statusStr := c.Query("status"); statusStr != "" {
		status := domain.AttendanceStatus(statusStr)
		filters.Status = &status
	}

	attendances, total, err := h.attendanceService.List(c.Request.Context(), filters)
	if err != nil {
		h.logger.Error("failed to list attendances", "error", err)
		response.InternalServerError(c, "Lỗi hệ thống khi lấy danh sách chấm công")
		return
	}

	now := h.clk.Now()
	data := make([]dto.AdminAttendanceResponse, 0, len(attendances))
	for _, att := range attendances {
		data = append(data, dto.AdminAttendanceResponse{
			ID:            att.ID,
			ProjectID:     att.ProjectID,
			ProjectName:   att.Project.Name,
			EmployeeID:    att.EmployeeID,
			EmployeeName:  att.Employee.Fullname,
			Date:          att.Date,
			CheckInTime:   att.CheckInTime,
			CheckInGate:   att.CheckInGate,
			CheckOutTime:  att.CheckOutTime,
			CheckOutGate:  att.CheckOutGate,
			EarningAmount: att.EarningAmount,
			Status:        string(att.GetStatus(now)),
		})
	}

	pagination := helpers.CalculatePagination(pg.Page, pg.PageSize, total)
	res := dto.PaginatedAttendanceResponse{
		Data:  data,
		Total: total,
	}

	response.SuccessWithPagination(c, res, "Lấy danh sách thành công", pagination)
}

// Get handles GET /api/v1/admin/attendances/:id
func (h *AttendanceHandler) Get(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", "ID không hợp lệ")
	if !ok {
		return
	}

	att, err := h.attendanceService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to get attendance", "error", err, "id", id)
		response.InternalServerError(c, "Lỗi hệ thống khi lấy thông tin chấm công")
		return
	}

	if att == nil {
		response.NotFound(c, "Không tìm thấy bản ghi")
		return
	}

	res := dto.AdminAttendanceResponse{
		ID:            att.ID,
		ProjectID:     att.ProjectID,
		ProjectName:   att.Project.Name,
		EmployeeID:    att.EmployeeID,
		EmployeeName:  att.Employee.Fullname,
		Date:          att.Date,
		CheckInTime:   att.CheckInTime,
		CheckInGate:   att.CheckInGate,
		CheckOutTime:  att.CheckOutTime,
		CheckOutGate:  att.CheckOutGate,
		EarningAmount: att.EarningAmount,
		Status:        string(att.GetStatus(h.clk.Now())),
	}

	response.Success(c, res, "Lấy thông tin thành công")
}
