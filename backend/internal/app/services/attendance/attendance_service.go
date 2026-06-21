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

// minimumCheckoutDuration is the fallback earliest-checkout floor used only when
// no payrate shift can be resolved for the employee's position (e.g. payrate not
// yet configured). When a configured shift IS resolved, its end hour is used as
// the earliest checkout instead — see resolveEarliestCheckout.
const minimumCheckoutDuration = 4 * time.Hour

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

// validateEarliestCheckout rejects a checkout that happens before the earliest
// allowed time. The earliest time is derived from the configured shift end hour
// (see resolveEarliestCheckout), so the message points the employee at the real
// end of their shift rather than a fixed check-in + N hours.
func validateEarliestCheckout(checkInTime, earliestCheckout, checkOutTime time.Time) error {
	if checkOutTime.Before(earliestCheckout) {
		return domain.NewValidationError(fmt.Sprintf(
			"Bạn mới vào làm lúc %s. Chỉ có thể tan ca sau %s",
			checkInTime.Format("15:04"),
			earliestCheckout.Format("15:04"),
		))
	}
	return nil
}

// parsedShift is a single configured shift for a position, resolved to absolute
// check-in-day datetimes (night-shift / cross-midnight aware) together with its rate.
type parsedShift struct {
	start  time.Time
	end    time.Time
	amount int
}

// resolveShifts parses the flattened payrate for the given position and returns:
//   - effectivePosition: the configured position matching `position` (case- and
//     diacritic-insensitive), falling back to the only configured position when the
//     requested one is absent.
//   - positionFound: whether the effective position exists in the configuration.
//   - shifts: every parseable shift for the effective position, on the check-in day.
//
// The caller derives shiftFound as len(shifts) > 0.
func resolveShifts(flattened map[string]int, position string, ci time.Time) (effectivePosition string, positionFound bool, shifts []parsedShift) {
	configuredPositions := make(map[string]string)
	for key := range flattened {
		parts := strings.Split(key, ".")
		if len(parts) < 3 {
			continue
		}
		pos := strings.Join(parts[:len(parts)-2], ".")
		configuredPositions[strings.ToLower(pos)] = pos
	}

	effectivePosition = position
	for _, configuredPosition := range configuredPositions {
		if strings.EqualFold(configuredPosition, position) {
			effectivePosition = configuredPosition
			break
		}
	}
	if !hasConfiguredPosition(configuredPositions, effectivePosition) && len(configuredPositions) == 1 {
		for _, onlyPosition := range configuredPositions {
			effectivePosition = onlyPosition
		}
	}
	positionFound = hasConfiguredPosition(configuredPositions, effectivePosition)

	for key, amount := range flattened {
		// key is position.dayType.HH:MM-HH:MM
		parts := strings.Split(key, ".")
		if len(parts) < 3 {
			continue
		}

		pos := strings.Join(parts[:len(parts)-2], ".")
		if !strings.EqualFold(pos, effectivePosition) {
			continue
		}

		timeRange := parts[len(parts)-1]
		timeParts := strings.Split(timeRange, "-")
		if len(timeParts) != 2 {
			continue
		}

		start, startErr := time.Parse("15:04", timeParts[0])
		end, endErr := time.Parse("15:04", timeParts[1])
		if startErr != nil || endErr != nil {
			continue
		}

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

		shifts = append(shifts, parsedShift{start: shiftStart, end: shiftEnd, amount: amount})
	}

	return effectivePosition, positionFound, shifts
}

// closestShift returns the shift whose start is nearest to the check-in time, or
// nil when there are no shifts. Used to report the expected shift boundaries.
func closestShift(shifts []parsedShift, ci time.Time) *parsedShift {
	var closest *parsedShift
	var closestDistance time.Duration
	for i := range shifts {
		distance := ci.Sub(shifts[i].start).Abs()
		if closest == nil || distance < closestDistance {
			closest = &shifts[i]
			closestDistance = distance
		}
	}
	return closest
}

// resolveEarliestCheckout determines the earliest allowed checkout time for an
// active attendance. When the payrate resolves a configured shift for the
// employee's position, the configured shift end (night-shift aware) is returned.
// Otherwise it falls back to check-in + minimumCheckoutDuration. It never errors:
// a malformed/empty payrate simply falls back to the minimum duration so checkout
// is not blocked by a broken rate config.
func (s *AttendanceService) resolveEarliestCheckout(payrate *domain.Payrate, position string, checkInTime time.Time) time.Time {
	if payrate != nil {
		if flattened, err := payrate.Payrate.Flatten(); err == nil {
			_, _, shifts := resolveShifts(flattened, position, checkInTime)
			if closest := closestShift(shifts, checkInTime); closest != nil {
				return closest.end
			}
		}
	}
	return checkInTime.Add(minimumCheckoutDuration)
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

		// Determine the earliest allowed checkout from the configured shift end
		// hour (payrate), falling back to a minimum duration when unconfigured.
		// Load the assignment + payrate here so they are reused for earning below.
		assignment, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(txCtx, attendance.ProjectID, attendance.EmployeeID)
		if err != nil {
			return err
		}
		payrate, err := s.payrateRepo.GetActiveByProjectAndDate(txCtx, attendance.ProjectID, attendance.Date)
		if err != nil {
			if !domain.IsNotFoundError(err) {
				return fmt.Errorf("failed to load payrate: %w", err)
			}
			payrate = nil
		}
		earliest := s.resolveEarliestCheckout(payrate, assignment.Position, attendance.CheckInTime)
		if err := validateEarliestCheckout(attendance.CheckInTime, earliest, now); err != nil {
			return err
		}

		// 2. Best-effort geofence validation for checkout.
		// Check-in already proves the worker started from a valid gate. Do not
		// block checkout on a second GPS read, because mobile location can drift
		// and leave the employee stuck in an active shift.
		project, err := s.projectRepo.GetByID(txCtx, attendance.ProjectID)
		if err != nil {
			return fmt.Errorf("failed to load project for geofence validation: %w", err)
		}
		gateName, err := s.validateGeofence(project, lat, lng)
		if err != nil {
			observability.GetLogger().Warn(
				"Checkout geofence validation failed; allowing checkout for active attendance",
				"employee_id", employeeID,
				"attendance_id", attendance.ID,
				"project_id", attendance.ProjectID,
				"lat", lat,
				"lng", lng,
				"error", err,
			)
			gateName = attendance.CheckInGate
			if strings.TrimSpace(gateName) == "" {
				gateName = "Không xác định"
			}
		}

		// 3. Update CheckOutTime and coords
		attendance.CheckOutTime = &now
		attendance.CheckOutLat = &lat
		attendance.CheckOutLng = &lng
		attendance.CheckOutGate = &gateName

		// 4. Calculate earning_amount (reuses the assignment + payrate loaded for
		// the earliest-checkout check above).
		if !assignment.CheckInEnabled {
			return domain.NewValidationError("Chấm công đã bị vô hiệu hóa. Vui lòng liên hệ quản lý.")
		}

		var earningAmount int64
		var salaryRejectReason *string
		if payrate == nil {
			reason := "Chưa có cấu hình mức lương hiệu lực cho ngày chấm công này."
			salaryRejectReason = &reason
		} else {
			var reason string
			earningAmount, reason, err = s.calculateEarningAmount(payrate, assignment.Position, attendance.CheckInTime, now)
			if err != nil {
				observability.GetLogger().Warn("Failed to calculate earning amount", "error", err)
				reason = "Cấu hình mức lương chưa hợp lệ, chưa thể ghi lương ca này."
				earningAmount = 0
			}
			if earningAmount <= 0 {
				if reason == "" {
					reason = "Thời gian vào/tan ca không hợp lệ. Vui lòng liên hệ quản lý."
				}
				salaryRejectReason = &reason
			}
		}
		attendance.EarningAmount = &earningAmount
		attendance.SalaryRejectReason = salaryRejectReason

		// 5. Update attendance record
		if err := s.attendanceRepo.Update(txCtx, attendance); err != nil {
			return err
		}

		// 6. Accumulate advance payment quota (self-check-in flow): salary = 100%
		// earned; max_adv_amount = floor(salary * SelfCheckInAdvanceablePercent / 100) = 70%.
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

			earning := uint64(earningAmount)
			if ap == nil {
				ap = &domain.AdvancePayment{
					ProjectID:    attendance.ProjectID,
					EmployeeID:   attendance.EmployeeID,
					ForMonth:     currentMonth,
					UploadDate:   currentDate,
					Salary:       earning,
					MaxAdvAmount: (earning * domain.SelfCheckInAdvanceablePercent) / 100,
				}
				if err := s.advancePaymentRepo.Create(txCtx, ap); err != nil {
					return err
				}
			} else {
				if err := s.advancePaymentRepo.AccumulateSalary(txCtx, uint64(ap.ID), earningAmount); err != nil {
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
	s.normalizeLegacySalaryRejectReason(ctx, att)

	return att, nil
}

// GetByID returns an attendance record by its ID.
func (s *AttendanceService) GetByID(ctx context.Context, id uint) (*domain.Attendance, error) {
	att, err := s.attendanceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.normalizeLegacySalaryRejectReason(ctx, att)
	return att, nil
}

// List returns a list of attendance records based on filters.
func (s *AttendanceService) List(ctx context.Context, filters domain.AttendanceFilters) ([]*domain.Attendance, int64, error) {
	attendances, err := s.attendanceRepo.List(ctx, filters)
	if err != nil {
		return nil, 0, err
	}
	for _, att := range attendances {
		s.normalizeLegacySalaryRejectReason(ctx, att)
	}

	count, err := s.attendanceRepo.Count(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	return attendances, count, nil
}

func (s *AttendanceService) normalizeLegacySalaryRejectReason(ctx context.Context, att *domain.Attendance) {
	if att == nil || att.SalaryRejectReason == nil || att.CheckOutTime == nil {
		return
	}
	if !strings.Contains(*att.SalaryRejectReason, "không khớp trọn ca") {
		return
	}

	assignment, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(ctx, att.ProjectID, att.EmployeeID)
	if err != nil {
		observability.GetLogger().Warn("Failed to load assignment for salary reject reason normalization", "attendance_id", att.ID, "error", err)
		return
	}

	payrate, err := s.payrateRepo.GetActiveByProjectAndDate(ctx, att.ProjectID, att.Date)
	if err != nil {
		observability.GetLogger().Warn("Failed to load payrate for salary reject reason normalization", "attendance_id", att.ID, "error", err)
		return
	}

	amount, reason, err := s.calculateEarningAmount(payrate, assignment.Position, att.CheckInTime, *att.CheckOutTime)
	if err != nil {
		observability.GetLogger().Warn("Failed to normalize salary reject reason", "attendance_id", att.ID, "error", err)
		return
	}
	if amount == 0 && reason != "" {
		att.SalaryRejectReason = &reason
	}
}

func (s *AttendanceService) calculateEarningAmount(payrate *domain.Payrate, position string, ci, co time.Time) (int64, string, error) {
	flattened, err := payrate.Payrate.Flatten()
	if err != nil {
		return 0, "Cấu hình mức lương chưa hợp lệ, chưa thể ghi lương ca này.", err
	}

	effectivePosition, positionFound, shifts := resolveShifts(flattened, position, ci)

	for i := range shifts {
		sh := &shifts[i]
		// Match: checkIn <= shiftStart && checkOut >= shiftEnd
		if (ci.Before(sh.start) || ci.Equal(sh.start)) && (co.After(sh.end) || co.Equal(sh.end)) {
			if sh.amount <= 0 {
				return 0, "Mức lương ca được cấu hình là 0đ. Vui lòng liên hệ quản lý.", nil
			}
			return int64(sh.amount), "", nil
		}
	}

	if !positionFound {
		return 0, fmt.Sprintf("Chưa có mức lương cho vị trí \"%s\".", position), nil
	}
	if len(shifts) == 0 {
		return 0, fmt.Sprintf("Chưa có ca làm hợp lệ trong cấu hình mức lương cho vị trí \"%s\".", effectivePosition), nil
	}

	closest := closestShift(shifts, ci)
	return 0, fmt.Sprintf(
		"Thời gian vào %s và tan %s không hợp lệ, bạn phải vào làm trước %s và tan ca sau %s.",
		ci.Format("15:04"),
		co.Format("15:04"),
		closest.start.Format("15:04"),
		closest.end.Format("15:04"),
	), nil
}

func hasConfiguredPosition(configuredPositions map[string]string, position string) bool {
	for _, configuredPosition := range configuredPositions {
		if strings.EqualFold(configuredPosition, position) {
			return true
		}
	}
	return false
}
