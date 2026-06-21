package attendance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/geo"
)

type AttendanceService struct {
	attendanceRepo      domain.AttendanceRepository
	projectEmployeeRepo domain.ProjectEmployeeRepository
	projectRepo         domain.ProjectRepository
	payrateRepo         domain.PayrateRepository
	advancePaymentRepo  domain.AdvancePaymentRepository
	transactionManager  *infrastructure.TransactionManager
	clock               clock.Clock
}

func NewAttendanceService(
	attendanceRepo domain.AttendanceRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	projectRepo domain.ProjectRepository,
	payrateRepo domain.PayrateRepository,
	advancePaymentRepo domain.AdvancePaymentRepository,
	transactionManager *infrastructure.TransactionManager,
	clk clock.Clock,
) *AttendanceService {
	if clk == nil {
		clk = clock.New()
	}
	return &AttendanceService{
		attendanceRepo:      attendanceRepo,
		projectEmployeeRepo: projectEmployeeRepo,
		projectRepo:         projectRepo,
		payrateRepo:         payrateRepo,
		advancePaymentRepo:  advancePaymentRepo,
		transactionManager:  transactionManager,
		clock:               clk,
	}
}

// validateGeofence checks if coordinates are within configured gates for the given project.
// Returns an error if no gates are configured.
func (s *AttendanceService) validateGeofence(project *domain.Project, lat, lng float64) (string, error) {
	gates := project.GeofenceGates
	if len(gates) == 0 {
		return "", domain.NewValidationError("Chưa cấu hình vị trí vào làm cho dự án")
	}

	radius := float64(project.GeofenceRadiusMeters)
	for _, gate := range gates {
		dist := geo.HaversineDistance(lat, lng, gate.Lat, gate.Lng)
		if dist <= radius {
			return gate.Name, nil
		}
	}
	return "", domain.NewValidationError("Bạn đang ở ngoài khu vực chấm công. Vui lòng di chuyển đến gần cổng nhà máy.")
}

// resolveProject determines the project for a check-in request.
// If projectID is provided (non-zero), it validates the project exists and is flexible.
// If projectID is 0 (omitted), it auto-detects the employee's active flexible project.
// Returns the resolved project to avoid redundant DB queries downstream.
func (s *AttendanceService) resolveProject(ctx context.Context, employeeID, projectID uint) (*domain.Project, error) {
	if projectID != 0 {
		project, err := s.projectRepo.GetByID(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to get project: %w", err)
		}
		if !project.IsFlexible {
			return nil, domain.NewValidationError("Dự án không hỗ trợ chấm công linh hoạt")
		}
		return project, nil
	}

	// Auto-detect: find active flexible projects for this employee
	projects, err := s.projectEmployeeRepo.GetActiveProjectsForEmployee(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active projects: %w", err)
	}

	var flexibleProjects []*domain.Project
	for _, p := range projects {
		if p.IsFlexible {
			flexibleProjects = append(flexibleProjects, p)
		}
	}

	switch len(flexibleProjects) {
	case 0:
		return nil, domain.NewValidationError("Bạn không thuộc dự án linh hoạt nào. Vui lòng liên hệ quản lý.")
	case 1:
		return flexibleProjects[0], nil
	default:
		return nil, domain.NewValidationError("Bạn thuộc nhiều dự án linh hoạt. Vui lòng chọn dự án trước khi vào làm.")
	}
}

func (s *AttendanceService) CheckIn(ctx context.Context, employeeID, projectID uint, lat, lng float64) (*domain.Attendance, error) {
	var result *domain.Attendance

	err := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// 1. Resolve and validate project (auto-detect if not provided)
		project, err := s.resolveProject(txCtx, employeeID, projectID)
		if err != nil {
			return err
		}

		// 2. Validate employee assignment is active and check_in_enabled
		assignment, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(txCtx, project.ID, employeeID)
		if err != nil {
			return err
		}
		if !assignment.CheckInEnabled {
			return domain.NewValidationError("Bạn chưa được cấp quyền chấm công")
		}

		// 3. Geofence validation
		gateName, err := s.validateGeofence(project, lat, lng)
		if err != nil {
			return err
		}

		// 4. Enforce 1 check-in per day per employee
		now := s.clock.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		existing, err := s.attendanceRepo.GetByEmployeeAndDate(txCtx, employeeID, today)
		if err != nil {
			return fmt.Errorf("failed to check existing attendance: %w", err)
		}
		if existing != nil {
			return domain.NewValidationError("Bạn đã vào làm trong ngày hôm nay rồi")
		}

		// 5. Create attendance record
		attendance := &domain.Attendance{
			EmployeeID:  employeeID,
			ProjectID:   project.ID,
			Date:        today,
			CheckInTime: now,
			CheckInLat:  lat,
			CheckInLng:  lng,
			CheckInGate: gateName,
		}

		if err := s.attendanceRepo.Create(txCtx, attendance); err != nil {
			return err
		}

		result = attendance
		return nil
	})

	return result, err
}

func (s *AttendanceService) CheckOut(ctx context.Context, employeeID uint, lat, lng float64) (*domain.Attendance, error) {
	var result *domain.Attendance

	err := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		now := s.clock.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		// 1. Load active check-in record for today (or yesterday if night shift)
		attendance, err := s.attendanceRepo.GetByEmployeeAndDate(txCtx, employeeID, today)
		if err != nil {
			return fmt.Errorf("failed to get today's attendance: %w", err)
		}
		if attendance == nil {
			yesterday := today.AddDate(0, 0, -1)
			attendance, err = s.attendanceRepo.GetByEmployeeAndDate(txCtx, employeeID, yesterday)
			if err != nil {
				return fmt.Errorf("failed to get yesterday's attendance: %w", err)
			}
			if attendance == nil {
				return domain.NewValidationError("Không tìm thấy thông tin vào làm hợp lệ")
			}
		}

		if attendance.IsCompleted() {
			return domain.NewValidationError("Bạn đã tan ca rồi")
		}
		if attendance.GetStatus(now) == domain.AttendanceStatusOrphaned {
			return domain.NewValidationError("Ca làm việc đã quá hạn tan ca")
		}

		// 2. Geofence validation
		project, err := s.projectRepo.GetByID(txCtx, attendance.ProjectID)
		if err != nil {
			return fmt.Errorf("failed to load project for geofence validation: %w", err)
		}
		gateName, err := s.validateGeofence(project, lat, lng)
		if err != nil {
			return err
		}

		// 3. Update CheckOutTime and coords
		attendance.CheckOutTime = &now
		attendance.CheckOutLat = &lat
		attendance.CheckOutLng = &lng
		attendance.CheckOutGate = &gateName

		// 4. Calculate earning_amount
		assignment, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(txCtx, attendance.ProjectID, attendance.EmployeeID)
		if err != nil {
			return err
		}
		if !assignment.CheckInEnabled {
			return domain.NewValidationError("Chấm công đã bị vô hiệu hóa. Vui lòng liên hệ quản lý.")
		}

		payrate, err := s.payrateRepo.GetActiveByProjectAndDate(txCtx, attendance.ProjectID, attendance.Date)
		if err != nil {
			return err
		}

		earningAmount, err := s.calculateEarningAmount(payrate, assignment.Position, attendance.CheckInTime, now)
		if err != nil {
			observability.GetLogger().Warn("Failed to calculate earning amount", "error", err)
			earningAmount = 0
		}

		attendance.EarningAmount = &earningAmount

		// 5. Update attendance record
		if err := s.attendanceRepo.Update(txCtx, attendance); err != nil {
			return err
		}

		// 6. Accumulate advance payment quota
		if earningAmount > 0 {
			currentMonth := now.Format("2006-01")
			currentDate := now.Format("2006-01-02")

			aps, err := s.advancePaymentRepo.GetByEmployeeAndMonth(txCtx, uint64(employeeID), currentMonth)
			if err != nil {
				return err
			}

			var ap *domain.AdvancePayment
			for _, v := range aps {
				if v.ProjectID == attendance.ProjectID {
					ap = v
					break
				}
			}

			if ap == nil {
				ap = &domain.AdvancePayment{
					ProjectID:    attendance.ProjectID,
					EmployeeID:   attendance.EmployeeID,
					ForMonth:     currentMonth,
					UploadDate:   currentDate,
					MaxAdvAmount: uint64(earningAmount),
				}
				if err := s.advancePaymentRepo.Create(txCtx, ap); err != nil {
					return err
				}
			} else {
				if err := s.advancePaymentRepo.IncrementMaxAdvAmount(txCtx, uint64(ap.ID), earningAmount); err != nil {
					return err
				}
			}
		}

		result = attendance
		return nil
	})

	return result, err
}

// GetTodayAttendance returns the attendance record for the employee for the current day.
// It returns nil if no record exists for today.
func (s *AttendanceService) GetTodayAttendance(ctx context.Context, employeeID uint) (*domain.Attendance, error) {
	now := s.clock.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	att, err := s.attendanceRepo.GetByEmployeeAndDate(ctx, employeeID, today)
	if err != nil {
		return nil, err
	}

	return att, nil
}

// GetByID returns an attendance record by its ID.
func (s *AttendanceService) GetByID(ctx context.Context, id uint) (*domain.Attendance, error) {
	return s.attendanceRepo.GetByID(ctx, id)
}

// List returns a list of attendance records based on filters.
func (s *AttendanceService) List(ctx context.Context, filters domain.AttendanceFilters) ([]*domain.Attendance, int64, error) {
	attendances, err := s.attendanceRepo.List(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.attendanceRepo.Count(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	return attendances, count, nil
}

func (s *AttendanceService) calculateEarningAmount(payrate *domain.Payrate, position string, ci, co time.Time) (int64, error) {
	flattened, err := payrate.Payrate.Flatten()
	if err != nil {
		return 0, err
	}

	for key, amount := range flattened {
		// key is position.dayType.HH:MM-HH:MM
		parts := strings.Split(key, ".")
		if len(parts) < 3 {
			continue
		}

		pos := strings.Join(parts[:len(parts)-2], ".")
		if !strings.EqualFold(pos, position) {
			continue
		}

		timeRange := parts[len(parts)-1]
		timeParts := strings.Split(timeRange, "-")
		if len(timeParts) != 2 {
			continue
		}

		start, _ := time.Parse("15:04", timeParts[0])
		end, _ := time.Parse("15:04", timeParts[1])

		// Build real time equivalents for the shift on the day of check-in
		shiftStart := time.Date(ci.Year(), ci.Month(), ci.Day(), start.Hour(), start.Minute(), 0, 0, ci.Location())
		shiftEnd := time.Date(ci.Year(), ci.Month(), ci.Day(), end.Hour(), end.Minute(), 0, 0, ci.Location())

		crossesMidnight := shiftEnd.Before(shiftStart)
		if crossesMidnight {
			shiftEnd = shiftEnd.Add(24 * time.Hour) // Night shift crosses midnight
		}

		// For night shifts, if check-in is before shiftStart on the check-in day,
		// the shift started yesterday (e.g., shift 22:00-06:00, check-in at 00:30).
		// Only apply rollback for night shifts to avoid breaking early-arrival day shifts
		// (e.g., 07:50 arrival for 08:00-17:00 shift).
		if crossesMidnight && ci.Before(shiftStart) {
			shiftStart = shiftStart.Add(-24 * time.Hour)
			shiftEnd = shiftEnd.Add(-24 * time.Hour)
		}

		// Match: checkIn <= shiftStart && checkOut >= shiftEnd
		if (ci.Before(shiftStart) || ci.Equal(shiftStart)) && (co.After(shiftEnd) || co.Equal(shiftEnd)) {
			return int64(amount), nil
		}
	}

	return 0, nil
}
