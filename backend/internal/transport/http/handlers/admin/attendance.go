package admin

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/attendance"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/geo"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type AttendanceHandler struct {
	attendanceService *attendance.AttendanceService
	failedAttemptRepo domain.AttendanceFailedAttemptRepository
	projectRepo       domain.ProjectRepository
	clk               clock.Clock
	logger            *slog.Logger
}

func NewAttendanceHandler(
	attendanceService *attendance.AttendanceService,
	failedAttemptRepo domain.AttendanceFailedAttemptRepository,
	projectRepo domain.ProjectRepository,
	clk clock.Clock,
	logger *slog.Logger,
) *AttendanceHandler {
	return &AttendanceHandler{
		attendanceService: attendanceService,
		failedAttemptRepo: failedAttemptRepo,
		projectRepo:       projectRepo,
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
	loc := time.Local // business timezone — match clock.Now() and DSN loc=Local
	if fromStr := c.Query("from_date"); fromStr != "" {
		if t, err := time.ParseInLocation("2006-01-02", fromStr, loc); err == nil {
			filters.FromDate = &t
		}
	}
	if toStr := c.Query("to_date"); toStr != "" {
		if t, err := time.ParseInLocation("2006-01-02", toStr, loc); err == nil {
			filters.ToDate = &t
		}
	}
	if statusStr := c.Query("status"); statusStr != "" {
		status := domain.AttendanceStatus(statusStr)
		filters.Status = &status
	}
	if c.Query("date_field") == "check_in_time" {
		filters.UseCheckInTimeWindow = true
	}
	if successStr := c.Query("successful_checkout"); successStr != "" {
		if successful, err := strconv.ParseBool(successStr); err == nil {
			filters.SuccessfulCheckout = &successful
		}
	}
	if zeroStr := c.Query("zero_earning"); zeroStr != "" {
		if zero, err := strconv.ParseBool(zeroStr); err == nil {
			filters.ZeroEarning = &zero
		}
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
		data = append(data, mapAdminAttendanceResponse(att, now))
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

	res := mapAdminAttendanceResponse(att, h.clk.Now())

	response.Success(c, res, "Lấy thông tin thành công")
}

func (h *AttendanceHandler) Approve(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", "ID không hợp lệ")
	if !ok {
		return
	}

	var req dto.AdminReviewAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Body is optional on approve (note may be blank); tolerate empty/invalid JSON.
		req = dto.AdminReviewAttendanceRequest{}
	}

	adminID, ok := helpers.GetUserIDOrRespond(c)
	if !ok {
		return
	}

	att, err := h.attendanceService.Approve(c.Request.Context(), id, adminID, strings.TrimSpace(req.Note))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, mapAdminAttendanceResponse(att, h.clk.Now()), "Duyệt chấm công thành công")
}

// Reject handles POST /api/v1/admin/attendances/:id/reject. It zeroes the
// earning, stores the reason in salary_reject_reason, and stamps the admin
// review. The note is required and returned as the reject reason.
func (h *AttendanceHandler) Reject(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", "ID không hợp lệ")
	if !ok {
		return
	}

	var req dto.AdminReviewAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ: "+err.Error())
		return
	}
	note := strings.TrimSpace(req.Note)
	if note == "" {
		response.BadRequest(c, "Vui lòng nhập lý do từ chối")
		return
	}

	adminID, ok := helpers.GetUserIDOrRespond(c)
	if !ok {
		return
	}

	att, err := h.attendanceService.Reject(c.Request.Context(), id, adminID, note)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, mapAdminAttendanceResponse(att, h.clk.Now()), "Đã từ chối chấm công")
}

func mapAdminAttendanceResponse(att *domain.Attendance, now time.Time) dto.AdminAttendanceResponse {
	status := string(att.GetStatus(now))
	res := dto.AdminAttendanceResponse{
		ID:                 att.ID,
		ProjectID:          att.ProjectID,
		ProjectName:        att.Project.Name,
		EmployeeID:         att.EmployeeID,
		EmployeeName:       att.Employee.Fullname,
		Date:               att.Date,
		CheckInTime:        att.CheckInTime,
		CheckInLat:         att.CheckInLat,
		CheckInLng:         att.CheckInLng,
		CheckInAccuracy:    att.CheckInAccuracy,
		CheckInGpsAt:       att.CheckInGpsAt,
		CheckInGate:        att.CheckInGate,
		CheckOutTime:       att.CheckOutTime,
		CheckOutLat:        att.CheckOutLat,
		CheckOutLng:        att.CheckOutLng,
		CheckOutAccuracy:   att.CheckOutAccuracy,
		CheckOutGpsAt:      att.CheckOutGpsAt,
		CheckOutGate:       att.CheckOutGate,
		EarningAmount:      att.EarningAmount,
		SalaryRejectReason: att.SalaryRejectReason,
		ReviewAction:       att.ReviewAction,
		ReviewNote:         att.ReviewNote,
		ReviewedBy:         att.ReviewedBy,
		ReviewedAt:         att.ReviewedAt,
		Status:             status,
	}

	if status == string(domain.AttendanceStatusRejected) {
		rejectedAt := att.UpdatedAt
		res.RejectedAt = &rejectedAt
	}

	if nearest := nearestCheckpointForCoordinates(att.CheckInLat, att.CheckInLng, &att.Project); nearest != nil {
		distance := nearest.distanceMeters
		res.NearestCheckpointDistanceMeters = &distance
		if nearest.name != "" {
			name := nearest.name
			res.NearestCheckpointName = &name
		}
		res.NearestCheckpointLat = &nearest.lat
		res.NearestCheckpointLng = &nearest.lng
		if nearest.geofenceRadiusMeters > 0 {
			radius := nearest.geofenceRadiusMeters
			res.GeofenceRadiusMeters = &radius
		}
	}

	if att.CheckOutLat != nil && att.CheckOutLng != nil {
		if nearest := nearestCheckpointForCoordinates(*att.CheckOutLat, *att.CheckOutLng, &att.Project); nearest != nil {
			distance := nearest.distanceMeters
			res.CheckOutNearestCheckpointDistanceMeters = &distance
			if nearest.name != "" {
				name := nearest.name
				res.CheckOutNearestCheckpointName = &name
			}
			res.CheckOutNearestCheckpointLat = &nearest.lat
			res.CheckOutNearestCheckpointLng = &nearest.lng
			if nearest.geofenceRadiusMeters > 0 && res.GeofenceRadiusMeters == nil {
				radius := nearest.geofenceRadiusMeters
				res.GeofenceRadiusMeters = &radius
			}
		}
	}

	return res
}

// AdminListFailedAttempts handles GET /api/v1/admin/attendances/failed-attempts.
// It returns the drill-down list behind the "failed attempts today" health tile.
func (h *AttendanceHandler) AdminListFailedAttempts(c *gin.Context) {
	var filters domain.FailedAttemptFilters

	pg := helpers.ParsePagination(c, 50)
	filters.Limit = pg.Limit
	filters.Offset = pg.Offset

	if t := c.Query("type"); t != "" {
		filters.AttemptType = &t
	}
	if cat := c.Query("category"); cat != "" {
		filters.ReasonCategory = &cat
	}
	if empIDStr := c.Query("employee_id"); empIDStr != "" {
		if empID, err := strconv.ParseUint(empIDStr, 10, 32); err == nil {
			id := uint(empID)
			filters.EmployeeID = &id
		}
	}
	loc := time.Local // business timezone — match clock.Now() and DSN loc=Local
	if fromStr := c.Query("from"); fromStr != "" {
		if t, err := time.ParseInLocation("2006-01-02", fromStr, loc); err == nil {
			filters.FromDate = &t
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		// Inclusive upper bound: shift to the start of the next day so the
		// repo's created_at < ? half-open interval covers the entire "to" day.
		if t, err := time.ParseInLocation("2006-01-02", toStr, loc); err == nil {
			next := t.AddDate(0, 0, 1)
			filters.ToDate = &next
		}
	}

	attempts, total, err := h.listFailedAttempts(c.Request.Context(), filters)
	if err != nil {
		h.logger.Error("failed to list failed attempts", "error", err)
		response.InternalServerError(c, "Lỗi hệ thống khi lấy danh sách lần thử thất bại")
		return
	}

	pagination := helpers.CalculatePagination(pg.Page, pg.PageSize, total)
	res := dto.PaginatedFailedAttemptResponse{
		Data:  attempts,
		Total: total,
	}

	response.SuccessWithPagination(c, res, "Lấy danh sách lần thử thất bại thành công", pagination)
}

// listFailedAttempts fetches the rows and total count and maps them to the admin DTO.
func (h *AttendanceHandler) listFailedAttempts(ctx context.Context, filters domain.FailedAttemptFilters) ([]dto.AdminFailedAttemptResponse, int64, error) {
	rows, err := h.failedAttemptRepo.List(ctx, filters)
	if err != nil {
		return nil, 0, err
	}
	total, err := h.failedAttemptRepo.Count(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	h.inferCheckoutProjectIDs(ctx, rows)

	projectsByID := map[uint]*domain.Project{}
	if h.projectRepo != nil {
		ids := projectIDsForFailedAttempts(rows)
		if len(ids) > 0 {
			projectsByID, err = h.projectRepo.GetByIDs(ctx, ids)
			if err != nil {
				return nil, 0, err
			}
		}
	}

	data := make([]dto.AdminFailedAttemptResponse, 0, len(rows))
	for _, a := range rows {
		data = append(data, mapFailedAttemptResponse(a, projectsByID[a.ProjectID]))
	}
	return data, total, nil
}

func (h *AttendanceHandler) inferCheckoutProjectIDs(ctx context.Context, rows []*domain.AttendanceFailedAttempt) {
	if h.attendanceService == nil {
		return
	}
	for _, row := range rows {
		if row == nil || row.ProjectID != 0 || row.AttemptType != "check_out" {
			continue
		}
		projectID := h.attendanceService.ResolveCheckoutProjectID(ctx, row.EmployeeID, row.CreatedAt)
		if projectID != 0 {
			row.ProjectID = projectID
		}
	}
}

func projectIDsForFailedAttempts(rows []*domain.AttendanceFailedAttempt) []uint {
	seen := make(map[uint]struct{})
	ids := make([]uint, 0)
	for _, row := range rows {
		if row == nil || row.ProjectID == 0 || row.Lat == nil || row.Lng == nil {
			continue
		}
		if _, ok := seen[row.ProjectID]; ok {
			continue
		}
		seen[row.ProjectID] = struct{}{}
		ids = append(ids, row.ProjectID)
	}
	return ids
}

func mapFailedAttemptResponse(a *domain.AttendanceFailedAttempt, project *domain.Project) dto.AdminFailedAttemptResponse {
	res := dto.AdminFailedAttemptResponse{
		ID:             a.ID,
		EmployeeID:     a.EmployeeID,
		EmployeeName:   a.Employee.Fullname,
		AttemptType:    a.AttemptType,
		ReasonCategory: a.ReasonCategory,
		ProjectID:      a.ProjectID,
		Lat:            a.Lat,
		Lng:            a.Lng,
		Accuracy:       a.Accuracy,
		GpsAt:          a.GpsAt,
		ErrorMessage:   a.ErrorMessage,
		ResolvedAt:     a.ResolvedAt,
		ResolvedBy:     a.ResolvedBy,
		ResolvedReason: a.ResolvedReason,
		CreatedAt:      a.CreatedAt,
	}

	nearest := nearestCheckpointForAttempt(a, project)
	if nearest == nil {
		return res
	}

	distance := nearest.distanceMeters
	res.NearestCheckpointDistanceMeters = &distance

	if nearest.name != "" {
		name := nearest.name
		res.NearestCheckpointName = &name
	}
	res.NearestCheckpointLat = &nearest.lat
	res.NearestCheckpointLng = &nearest.lng
	if nearest.geofenceRadiusMeters > 0 {
		radius := nearest.geofenceRadiusMeters
		res.GeofenceRadiusMeters = &radius
	}

	return res
}

type nearestCheckpoint struct {
	name                 string
	lat                  float64
	lng                  float64
	distanceMeters       float64
	geofenceRadiusMeters uint
}

func nearestCheckpointForAttempt(a *domain.AttendanceFailedAttempt, project *domain.Project) *nearestCheckpoint {
	if a == nil || project == nil || a.Lat == nil || a.Lng == nil || len(project.GeofenceGates) == 0 {
		return nil
	}

	return nearestCheckpointForCoordinates(*a.Lat, *a.Lng, project)
}

func nearestCheckpointForCoordinates(lat, lng float64, project *domain.Project) *nearestCheckpoint {
	if project == nil || len(project.GeofenceGates) == 0 {
		return nil
	}

	nearestIndex := -1
	nearestDistance := 0.0
	for i := range project.GeofenceGates {
		gate := project.GeofenceGates[i]
		distance := geo.HaversineDistance(lat, lng, gate.Lat, gate.Lng)
		if nearestIndex == -1 || distance < nearestDistance {
			nearestIndex = i
			nearestDistance = distance
		}
	}

	if nearestIndex == -1 {
		return nil
	}

	gate := project.GeofenceGates[nearestIndex]
	return &nearestCheckpoint{
		name:                 gate.Name,
		lat:                  gate.Lat,
		lng:                  gate.Lng,
		distanceMeters:       nearestDistance,
		geofenceRadiusMeters: project.GeofenceRadiusMeters,
	}
}
