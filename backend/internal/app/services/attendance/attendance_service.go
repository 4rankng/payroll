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

// checkInShiftWindow is the half-width of the check-in window around the
// configured shift start T: check-in is allowed in (T - checkInShiftWindow,
// T + checkInShiftWindow).
const checkInShiftWindow = 1 * time.Hour

// checkOutUpperGrace is how long after the configured shift end K a checkout is
// still allowed: checkout is valid in [K, K + checkOutUpperGrace).
const checkOutUpperGrace = 1 * time.Hour

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

// checkInFitsShift reports whether checkInTime lies within the allowed check-in
// window (shift.start - checkInShiftWindow, shift.start + checkInShiftWindow).
// Shared by the check-in gate (validateCheckInWindow) and the earning match so
// the two cannot diverge.
func checkInFitsShift(shift *parsedShift, checkInTime time.Time) bool {
	return checkInTime.After(shift.start.Add(-checkInShiftWindow)) && checkInTime.Before(shift.start.Add(checkInShiftWindow))
}

// checkOutFitsShift reports whether checkOutTime lies within the allowed
// checkout window [shift.end, shift.end + checkOutUpperGrace). Shared by the
// checkout gate (validateCheckOutWindow) and the earning match.
func checkOutFitsShift(shift *parsedShift, checkOutTime time.Time) bool {
	return (checkOutTime.After(shift.end) || checkOutTime.Equal(shift.end)) && checkOutTime.Before(shift.end.Add(checkOutUpperGrace))
}

// validateCheckInWindow rejects a check-in that falls outside the allowed
// (shift.start - checkInShiftWindow, shift.start + checkInShiftWindow) window
// around the configured shift start T.
func validateCheckInWindow(shift *parsedShift, checkInTime time.Time) error {
	if checkInFitsShift(shift, checkInTime) {
		return nil
	}
	earliest := shift.start.Add(-checkInShiftWindow)
	latest := shift.start.Add(checkInShiftWindow)
	return domain.NewValidationError(fmt.Sprintf(
		"Giờ vào làm không hợp lệ. Bạn chỉ được vào làm từ %s đến %s.",
		earliest.Format("15:04"),
		latest.Format("15:04"),
	))
}

// validateCheckOutWindow rejects a checkout that falls outside the allowed
// [shift.end, shift.end + checkOutUpperGrace) window, where shift.end is the
// configured shift end K. The message points the employee at the real end of
// their shift.
func validateCheckOutWindow(shift *parsedShift, checkInTime, checkOutTime time.Time) error {
	earliest := shift.end
	latest := shift.end.Add(checkOutUpperGrace)
	if checkOutTime.Before(earliest) {
		return domain.NewValidationError(fmt.Sprintf(
			"Bạn mới vào làm lúc %s. Chỉ có thể tan ca từ %s đến %s.",
			checkInTime.Format("15:04"),
			earliest.Format("15:04"),
			latest.Format("15:04"),
		))
	}
	if !checkOutTime.Before(latest) {
		return domain.NewValidationError(fmt.Sprintf(
			"Đã quá giờ tan ca. Bạn chỉ được tan ca từ %s đến %s.",
			earliest.Format("15:04"),
			latest.Format("15:04"),
		))
	}
	return nil
}

// parsedShift is a single configured shift for a position, resolved to absolute
// datetimes (night-shift / cross-midnight aware, anchored to the check-in day or
// one of its ±1 neighbors) together with its rate.
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
//   - shifts: every parseable shift for the effective position, generated for the
//     check-in day and its ±1 neighbors so night shifts and early arrivals anchor
//     to the correct calendar day.
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

		// Generate candidate shifts starting on the day before, the day of, and
		// the day after check-in. Each candidate keeps absolute [start, end] with
		// cross-midnight handled by adding 24h to end. closestShift then picks the
		// candidate whose start is nearest the check-in, anchoring night shifts and
		// early arrivals (e.g. 19:50 for 20:00-04:00) to the correct calendar day —
		// replacing the old rollback hack that miscomputed K for early arrivals.
		for dayOffset := -1; dayOffset <= 1; dayOffset++ {
			day := ci.AddDate(0, 0, dayOffset)
			shiftStart := time.Date(day.Year(), day.Month(), day.Day(), start.Hour(), start.Minute(), 0, 0, ci.Location())
			shiftEnd := time.Date(day.Year(), day.Month(), day.Day(), end.Hour(), end.Minute(), 0, 0, ci.Location())
			if shiftEnd.Before(shiftStart) {
				shiftEnd = shiftEnd.Add(24 * time.Hour) // Night shift crosses midnight
			}
			shifts = append(shifts, parsedShift{start: shiftStart, end: shiftEnd, amount: amount})
		}
	}

	return effectivePosition, positionFound, shifts
}

// closestShift returns the shift the check-in belongs to, or nil when there are
// no shifts. It prefers a shift whose [start, end] actually contains the
// check-in (the worker is mid-shift) and only falls back to nearest start when
// none contains ci. The "contains" preference stops a short neighboring shift
// from stealing the anchor near a long shift's end (e.g. a 04:00-05:00 shift
// winning over a 20:00-04:00 night shift for a 03:55 check-in) and avoids
// resolving off-hours check-ins to a future shift. Used to anchor the
// check-in/checkout/earning windows and to report expected shift boundaries.
func closestShift(shifts []parsedShift, ci time.Time) *parsedShift {
	contains := func(sh *parsedShift) bool {
		return (ci.After(sh.start) || ci.Equal(sh.start)) && (ci.Before(sh.end) || ci.Equal(sh.end))
	}
	var closest *parsedShift
	var closestDistance time.Duration
	for i := range shifts {
		sh := &shifts[i]
		distance := ci.Sub(sh.start).Abs()
		switch {
		case closest == nil:
			closest, closestDistance = sh, distance
		case contains(sh) && !contains(closest):
			closest, closestDistance = sh, distance
		case contains(sh) == contains(closest) && distance < closestDistance:
			closest, closestDistance = sh, distance
		}
	}
	return closest
}

// resolveShift returns the configured shift whose start is closest to checkInTime
// (resolved across ±1 day so night shifts and early arrivals anchor to the correct
// calendar day), or nil when no payrate/position/shift can be resolved. Callers
// derive the check-in/checkout windows from the returned shift:
//   - check-in valid in (shift.start - checkInShiftWindow, shift.start + checkInShiftWindow)
//   - checkout valid in [shift.end, shift.end + checkOutUpperGrace)
//
// A nil result means the project has no valid shift configuration for the position,
// so the check-in/checkout is rejected rather than falling back to a fixed duration.
func (s *AttendanceService) resolveShift(payrate *domain.Payrate, position string, checkInTime time.Time) *parsedShift {
	if payrate == nil {
		return nil
	}
	flattened, err := payrate.Payrate.Flatten()
	if err != nil {
		// A malformed payrate is operationally distinct from "no shift configured";
		// log it so ops can tell a broken config from a missing one (the user-facing
		// message is the same generic "not configured" either way).
		observability.GetLogger().Warn("failed to flatten payrate; cannot resolve shift", "error", err)
		return nil
	}
	_, _, shifts := resolveShifts(flattened, position, checkInTime)
	return closestShift(shifts, checkInTime)
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

		// 5. Validate check-in falls within ±1h of the configured shift start (T).
		// No resolvable shift (missing payrate/position) => reject — there is no
		// valid attendance window for this position.
		payrate, err := s.payrateRepo.GetActiveByProjectAndDate(txCtx, project.ID, today)
		if err != nil {
			if !domain.IsNotFoundError(err) {
				return fmt.Errorf("failed to load payrate: %w", err)
			}
			payrate = nil
		}
		shift := s.resolveShift(payrate, assignment.Position, now)
		if shift == nil {
			return domain.NewValidationError("Chưa cấu hình ca làm việc cho vị trí này. Vui lòng liên hệ quản lý.")
		}
		if err := validateCheckInWindow(shift, now); err != nil {
			return err
		}

		// 6. Create attendance record
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

		// Resolve the worked shift to derive the checkout window [K, K+1h) from the
		// configured shift end. Load the assignment + payrate here so they are reused
		// for earning below.
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
		shift := s.resolveShift(payrate, assignment.Position, attendance.CheckInTime)
		if shift == nil {
			return domain.NewValidationError("Chưa cấu hình ca làm việc cho vị trí này. Vui lòng liên hệ quản lý.")
		}
		if err := validateCheckOutWindow(shift, attendance.CheckInTime, now); err != nil {
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

	// A shift earns its flat amount when the attendance fits BOTH the check-in
	// and checkout windows — the same predicates the gates use, so a checkout the
	// gate accepts is guaranteed to earn. Among matching shifts pick the one
	// nearest the check-in start (tie-broken by earlier start) so the payout is
	// deterministic regardless of map iteration order.
	var match *parsedShift
	var matchDistance time.Duration
	for i := range shifts {
		sh := &shifts[i]
		if !checkInFitsShift(sh, ci) || !checkOutFitsShift(sh, co) {
			continue
		}
		distance := ci.Sub(sh.start).Abs()
		if match == nil || distance < matchDistance || (distance == matchDistance && sh.start.Before(match.start)) {
			match, matchDistance = sh, distance
		}
	}
	if match != nil {
		if match.amount <= 0 {
			return 0, "Mức lương ca được cấu hình là 0đ. Vui lòng liên hệ quản lý.", nil
		}
		return int64(match.amount), "", nil
	}

	if !positionFound {
		return 0, fmt.Sprintf("Chưa có mức lương cho vị trí \"%s\".", position), nil
	}
	closest := closestShift(shifts, ci)
	if closest == nil {
		return 0, fmt.Sprintf("Chưa có ca làm hợp lệ trong cấu hình mức lương cho vị trí \"%s\".", effectivePosition), nil
	}
	return 0, fmt.Sprintf(
		"Thời gian vào %s và tan %s không hợp lệ. Bạn phải vào làm từ %s đến %s và tan ca từ %s đến %s.",
		ci.Format("15:04"),
		co.Format("15:04"),
		closest.start.Add(-checkInShiftWindow).Format("15:04"),
		closest.start.Add(checkInShiftWindow).Format("15:04"),
		closest.end.Format("15:04"),
		closest.end.Add(checkOutUpperGrace).Format("15:04"),
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
