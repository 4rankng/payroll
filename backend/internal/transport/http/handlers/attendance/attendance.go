package attendance

import (
	"log/slog"
	"strconv"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/attendance"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	attendanceService *attendance.AttendanceService
	employeeRepo      domain.EmployeeRepository
	clk               clock.Clock
	logger            *slog.Logger
}

func NewHandler(attendanceService *attendance.AttendanceService, employeeRepo domain.EmployeeRepository, clk clock.Clock, logger *slog.Logger) *Handler {
	return &Handler{
		attendanceService: attendanceService,
		employeeRepo:      employeeRepo,
		clk:               clk,
		logger:            logger,
	}
}

// resolveEmployeeID resolves the Employee.ID from the JWT's User.ID via the employee repository.
// The JWT stores User.ID (from the users table), but the attendance service needs Employee.ID.
func (h *Handler) resolveEmployeeID(c *gin.Context) (uint, bool) {
	uid, ok := helpers.GetUserIDOrRespond(c)
	if !ok {
		return 0, false
	}
	employee, err := h.employeeRepo.GetByUserID(c.Request.Context(), uid)
	if err != nil {
		response.HandleDomainError(c, err)
		return 0, false
	}
	return employee.ID, true
}

func (h *Handler) mapToResponse(att *domain.Attendance) *dto.AttendanceResponse {
	return &dto.AttendanceResponse{
		ID:            att.ID,
		ProjectID:     att.ProjectID,
		EmployeeID:    att.EmployeeID,
		Date:          att.Date,
		CheckInTime:   att.CheckInTime,
		CheckInGate:   att.CheckInGate,
		CheckOutTime:  att.CheckOutTime,
		CheckOutGate:  att.CheckOutGate,
		EarningAmount: att.EarningAmount,
		Status:        string(att.GetStatus(h.clk.Now())),
	}
}

// CheckIn handles POST /api/v1/mobile/attendance/check-in
func (h *Handler) CheckIn(c *gin.Context) {
	var req dto.CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ: "+err.Error())
		return
	}

	employeeID, ok := h.resolveEmployeeID(c)
	if !ok {
		return
	}

	att, err := h.attendanceService.CheckIn(c.Request.Context(), employeeID, req.ProjectID, req.Lat, req.Lng)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, h.mapToResponse(att), "Check-in thành công")
}

// CheckOut handles POST /api/v1/mobile/attendance/check-out
func (h *Handler) CheckOut(c *gin.Context) {
	var req dto.CheckOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ: "+err.Error())
		return
	}

	employeeID, ok := h.resolveEmployeeID(c)
	if !ok {
		return
	}

	att, err := h.attendanceService.CheckOut(c.Request.Context(), employeeID, req.Lat, req.Lng)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, h.mapToResponse(att), "Check-out thành công")
}

// GetToday handles GET /api/v1/mobile/attendance/today
func (h *Handler) GetToday(c *gin.Context) {
	employeeID, ok := h.resolveEmployeeID(c)
	if !ok {
		return
	}

	att, err := h.attendanceService.GetTodayAttendance(c.Request.Context(), employeeID)
	if err != nil {
		h.logger.Error("failed to get today attendance", "error", err)
		response.InternalServerError(c, "Lỗi hệ thống khi lấy thông tin chấm công")
		return
	}

	if att == nil {
		response.Success(c, nil, "Chưa có dữ liệu chấm công hôm nay")
		return
	}

	response.Success(c, h.mapToResponse(att), "Lấy thông tin chấm công thành công")
}

// List handles GET /api/v1/mobile/attendance/history
func (h *Handler) List(c *gin.Context) {
	employeeID, ok := h.resolveEmployeeID(c)
	if !ok {
		return
	}

	filters := domain.AttendanceFilters{
		EmployeeID: &employeeID,
		Limit:      20,
	}

	// Allow client to override limit (capped at 100)
	if limitStr := c.Query("limit"); limitStr != "" {
		if lim, err := strconv.Atoi(limitStr); err == nil && lim > 0 && lim <= 100 {
			filters.Limit = lim
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if off, err := strconv.Atoi(offsetStr); err == nil && off >= 0 {
			filters.Offset = off
		}
	}

	attendances, _, err := h.attendanceService.List(c.Request.Context(), filters)
	if err != nil {
		h.logger.Error("failed to get attendance history", "error", err)
		response.InternalServerError(c, "Lỗi hệ thống khi lấy lịch sử chấm công")
		return
	}

	res := make([]*dto.AttendanceResponse, len(attendances))
	for i, att := range attendances {
		res[i] = h.mapToResponse(att)
	}

	response.Success(c, res, "Lấy lịch sử chấm công thành công")
}
