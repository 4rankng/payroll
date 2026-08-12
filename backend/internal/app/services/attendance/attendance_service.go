package attendance

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/geo"
)

// checkInShiftWindow is the half-width of the check-in window around the
// configured shift start T: check-in is allowed in (T - checkInShiftWindow,
// T + checkInShiftWindow).
const checkInShiftWindow = 1 * time.Hour

const (
	// checkOutLowerGrace is how long before the configured shift end K a checkout
	// is still allowed.
	checkOutLowerGrace = 1 * time.Hour
	// checkOutUpperGrace is how long after the configured shift end K a checkout is
	// still allowed: checkout is valid in [K - checkOutLowerGrace, K + checkOutUpperGrace].
	checkOutUpperGrace = 4 * time.Hour
)

const confirmedNoSalaryCheckoutReason = "Nhân viên đã xác nhận tan ca không ghi nhận tiền lương cho ca này."
const employeeCancelledWrongShiftReason = "Nhân viên đã hủy ca do vào nhầm ca."

const (
	attendanceCheckInWindowCode   = "ATTENDANCE_CHECK_IN_WINDOW"
	attendanceCheckOutWindowCode  = "ATTENDANCE_CHECK_OUT_WINDOW"
	attendanceGPSInaccurateCode   = "ATTENDANCE_GPS_INACCURATE"
	attendanceOutsideGeofenceCode = "ATTENDANCE_OUTSIDE_GEOFENCE"
)

type nearestCheckpointGuidance struct {
	Name           string  `json:"name"`
	Lat            float64 `json:"lat"`
	Lng            float64 `json:"lng"`
	DistanceMeters float64 `json:"distance_meters"`
}

func newTimingValidationError(code, action, message string, earliest, latest time.Time) *domain.DomainError {
	return domain.NewValidationErrorWithCode(code, message).
		WithContext("guidance_type", "timing").
		WithContext("action", action).
		WithContext("window_start", earliest.Format("15:04")).
		WithContext("window_end", latest.Format("15:04"))
}

func newLocationValidationError(code, reason, message string, checkpoint nearestCheckpointGuidance) *domain.DomainError {
	return domain.NewValidationErrorWithCode(code, message).
		WithContext("guidance_type", "location").
		WithContext("reason", reason).
		WithContext("nearest_checkpoint", checkpoint)
}

// TaskEnqueuer schedules deferred attendance tasks. Implemented by the asynq
// client wrapper; fakes capture the calls in tests. Nil is allowed — when unset,
// CheckIn/CheckOut skip scheduling (used in lightweight tests).
type TaskEnqueuer interface {
	EnqueueAutoRejectCheckout(attendanceID uint, at time.Time) error
	// EnqueueCreditQuota schedules the deferred quota-credit task after the
	// Admin-configured post-checkout hold. The worker is idempotent (guards on
	// quota_credited_at).
	EnqueueCreditQuota(attendanceID uint, at time.Time) error
}

type selfCheckInAdvanceHoldDurationProvider interface {
	GetSelfCheckInAdvanceHoldDuration(ctx context.Context) time.Duration
}

type AttendanceService struct {
	attendanceRepo      domain.AttendanceRepository
	projectEmployeeRepo domain.ProjectEmployeeRepository
	projectRepo         domain.ProjectRepository
	payrateRepo         domain.PayrateRepository
	advancePaymentRepo  domain.AdvancePaymentRepository
	settingsConfig      interface {
		GetSelfCheckInAdvancePercentageForUpdate(ctx context.Context) (uint64, error)
	}
	transactionManager domain.TransactionManager
	taskEnqueuer       TaskEnqueuer
	clock              clock.Clock
}

func NewAttendanceService(
	attendanceRepo domain.AttendanceRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	projectRepo domain.ProjectRepository,
	payrateRepo domain.PayrateRepository,
	advancePaymentRepo domain.AdvancePaymentRepository,
	settingsConfig interface {
		GetSelfCheckInAdvancePercentageForUpdate(ctx context.Context) (uint64, error)
	},
	transactionManager domain.TransactionManager,
	taskEnqueuer TaskEnqueuer,
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
		settingsConfig:      settingsConfig,
		transactionManager:  transactionManager,
		taskEnqueuer:        taskEnqueuer,
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
// checkout window [shift.end - checkOutLowerGrace, shift.end + checkOutUpperGrace].
// Shared by the checkout gate (validateCheckOutWindow) and the earning match.
func checkOutFitsShift(shift *parsedShift, checkOutTime time.Time) bool {
	earliest := shift.end.Add(-checkOutLowerGrace)
	latest := shift.end.Add(checkOutUpperGrace)
	return (checkOutTime.After(earliest) || checkOutTime.Equal(earliest)) && !checkOutTime.After(latest)
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
	return newTimingValidationError(attendanceCheckInWindowCode, "check_in", fmt.Sprintf(
		"Giờ vào làm không hợp lệ. Bạn chỉ được vào làm từ %s đến %s.",
		earliest.Format("15:04"),
		latest.Format("15:04"),
	), earliest, latest)
}

// validateCheckOutWindow rejects a checkout that falls outside the allowed
// [shift.end - checkOutLowerGrace, shift.end + checkOutUpperGrace] window, where
// shift.end is the configured shift end K. The message points the employee at
// the allowed checkout period.
func validateCheckOutWindow(shift *parsedShift, checkInTime, checkOutTime time.Time) error {
	earliest := shift.end.Add(-checkOutLowerGrace)
	latest := shift.end.Add(checkOutUpperGrace)
	if checkOutTime.Before(earliest) {
		return newTimingValidationError(attendanceCheckOutWindowCode, "check_out", fmt.Sprintf(
			"Bạn mới vào làm lúc %s. Chỉ có thể tan ca từ %s đến %s.",
			checkInTime.Format("15:04"),
			earliest.Format("15:04"),
			latest.Format("15:04"),
		), earliest, latest)
	}
	if checkOutTime.After(latest) {
		return newTimingValidationError(attendanceCheckOutWindowCode, "check_out", fmt.Sprintf(
			"Đã quá giờ tan ca. Bạn chỉ được tan ca từ %s đến %s.",
			earliest.Format("15:04"),
			latest.Format("15:04"),
		), earliest, latest)
	}
	return nil
}

func isConfirmedNoSalaryCheckout(att *domain.Attendance) bool {
	return att != nil &&
		att.CheckOutTime != nil &&
		att.EarningAmount != nil &&
		*att.EarningAmount == 0 &&
		att.SalaryRejectReason != nil &&
		strings.Contains(*att.SalaryRejectReason, confirmedNoSalaryCheckoutReason)
}

func isAutoRejectedNoCheckout(att *domain.Attendance) bool {
	return att != nil && att.CheckOutTime == nil && att.SalaryRejectReason != nil
}

// parsedShift is a single configured shift for a position, resolved to absolute
// datetimes (night-shift / cross-midnight aware, anchored to the check-in day or
// one of its ±1 neighbors) together with its rate.
type parsedShift struct {
	start  time.Time
	end    time.Time
	amount int
}

// ShiftWindow is an advisory representation of a configured shift and the
// attendance windows derived from it. It is intentionally separate from the
// domain Attendance model: these times are informational and the validation
// methods remain authoritative.
type ShiftWindow struct {
	ShiftStart          time.Time
	ShiftEnd            time.Time
	CheckInWindowStart  time.Time
	CheckInWindowEnd    time.Time
	CheckOutWindowStart time.Time
	CheckOutWindowEnd   time.Time
	// Name is the admin-chosen display name for this shift's time-range (e.g.
	// "Ca làm" for "09:00-18:00"), looked up from the project's ShiftNames.
	// Empty when no name is configured — callers fall back to default labels.
	Name string
}

// AdminCheckInShift is a server-resolved shift that an administrator can use
// to record a missed check-in. Keeping these absolute timestamps on the server
// prevents clients from manufacturing their own checkout window.
type AdminCheckInShift struct {
	Index    int
	Label    string
	Start    time.Time
	End      time.Time
	Amount   int64
	Position string
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
//   - checkout valid in [shift.end - checkOutLowerGrace, shift.end + checkOutUpperGrace]
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

// validateGeofence checks whether a GPS reading is confidently inside a
// configured gate.
//
// A reading passes a gate when either:
//   - the whole uncertainty circle fits inside the radius
//     (dist + accuracy <= radius); or
//   - the point estimate is comfortably inside the zone (dist <= radius/2) and
//     the reported accuracy is no worse than the zone (0 < accuracy <= radius).
//
// The second path stops a zone-scale GPS reading from blocking an employee who
// is plainly at the gate: the worst-case rule alone rejects a point 50 m from a
// 150 m gate when the phone reports ±100 m, because 50 + 100 overshoots the
// radius by centimetres. Requiring accuracy <= radius keeps grossly inaccurate
// fixes (e.g. ±800 m) rejected via the uncertain branch below.
//
// accuracy == 0 (unknown, or not sent by a legacy client) skips the uncertainty
// terms — dist + 0 <= radius reduces to "point inside" — so legacy on-gate
// fixes still pass. A reading whose uncertainty cannot decide inside vs outside
// is "GPS inaccurate"; a reading outside every gate is "outside geofence" even
// when its accuracy is poor (a far reading is unambiguously outside, so the
// accuracy guard no longer short-circuits before the distance check). Returns an
// error if no gates are configured.
func (s *AttendanceService) validateGeofence(project *domain.Project, reading domain.GeoReading) (string, error) {
	gates := project.GeofenceGates
	if len(gates) == 0 {
		return "", domain.NewValidationError("Dự án chưa cấu hình vị trí chấm công. Vui lòng báo quản lý.")
	}

	radius := float64(project.GeofenceRadiusMeters)
	insideButUncertain := false
	var nearestGate domain.GeofenceGate
	nearestDistance := -1.0
	for _, gate := range gates {
		dist := geo.HaversineDistance(reading.Lat, reading.Lng, gate.Lat, gate.Lng)
		if nearestDistance < 0 || dist < nearestDistance {
			nearestGate = gate
			nearestDistance = dist
		}
		if dist > radius {
			continue
		}
		if reading.Accuracy <= 0 || dist+reading.Accuracy <= radius {
			return gate.Name, nil
		}
		// Point estimate is inside, but the uncertainty circle spills past the
		// radius. Trust it when GPS accuracy is zone-scale and the employee is
		// in the inner half of the zone — they are plainly at the gate, and a
		// few metres of GPS jitter should not block an otherwise valid checkout.
		if reading.Accuracy <= radius && dist <= radius/2 {
			return gate.Name, nil
		}
		insideButUncertain = true
	}
	checkpoint := nearestCheckpointGuidance{
		Name:           nearestGate.Name,
		Lat:            nearestGate.Lat,
		Lng:            nearestGate.Lng,
		DistanceMeters: nearestDistance,
	}
	if insideButUncertain {
		return "", newLocationValidationError(
			attendanceGPSInaccurateCode,
			"gps_inaccurate",
			"Tín hiệu GPS không đủ chính xác để chấm công. Hãy đứng ở nơi thoáng hơn, giữ điện thoại yên vài giây rồi thử lại.",
			checkpoint,
		)
	}
	return "", newLocationValidationError(
		attendanceOutsideGeofenceCode,
		"outside_geofence",
		"Bạn đang ở ngoài khu vực chấm công của dự án. Vui lòng di chuyển đến cổng hoặc khu vực đã được cấu hình.",
		checkpoint,
	)
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

// ResolveProjectID returns the project id a check-in for this employee would
// resolve to, best-effort. The handler uses it to attribute a failed attempt to
// the right project when project_id was omitted (auto-detected), since the
// resolved id is otherwise trapped inside CheckIn's transaction. Returns 0 when
// the project cannot be determined — errors are swallowed because this only
// feeds best-effort forensic logging.
func (s *AttendanceService) ResolveProjectID(ctx context.Context, employeeID, projectID uint) uint {
	project, err := s.resolveProject(ctx, employeeID, projectID)
	if err != nil || project == nil {
		return 0
	}
	return project.ID
}

// ResolveCheckoutProjectID returns the project attached to the attendance record
// a checkout attempt would target at attemptedAt. This is best-effort forensic
// metadata for failed-attempt logging and dashboard enrichment; errors are
// swallowed so the original checkout response path is never blocked.
func (s *AttendanceService) ResolveCheckoutProjectID(ctx context.Context, employeeID uint, attemptedAt time.Time) uint {
	if s == nil || s.attendanceRepo == nil {
		return 0
	}
	if attemptedAt.IsZero() {
		if s.clock == nil {
			return 0
		}
		attemptedAt = s.clock.Now()
	}

	attemptedAt = attemptedAt.In(clock.DefaultLocation)
	today := time.Date(attemptedAt.Year(), attemptedAt.Month(), attemptedAt.Day(), 0, 0, 0, 0, clock.DefaultLocation)

	attendance, err := s.attendanceRepo.GetByEmployeeAndDate(ctx, employeeID, today)
	if err != nil {
		return 0
	}
	if attendance == nil {
		yesterday := today.AddDate(0, 0, -1)
		attendance, err = s.attendanceRepo.GetByEmployeeAndDate(ctx, employeeID, yesterday)
		if err != nil {
			return 0
		}
	}
	if attendance == nil {
		return 0
	}
	return attendance.ProjectID
}

func (s *AttendanceService) CheckIn(ctx context.Context, employeeID, projectID uint, geo domain.GeoReading) (*domain.Attendance, error) {
	var result *domain.Attendance

	err := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		now := s.clock.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if err := s.completeLegacyApprovedOpenAttendance(txCtx, employeeID, today); err != nil {
			return err
		}

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
		gateName, err := s.validateGeofence(project, geo)
		if err != nil {
			return err
		}

		// 4. Enforce 1 check-in per day per employee
		existing, err := s.attendanceRepo.GetByEmployeeAndDate(txCtx, employeeID, today)
		if err != nil {
			return fmt.Errorf("failed to check existing attendance: %w", err)
		}
		if existing != nil && !isConfirmedNoSalaryCheckout(existing) && !isAutoRejectedNoCheckout(existing) {
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
			EmployeeID:      employeeID,
			ProjectID:       project.ID,
			Date:            today,
			CheckInTime:     now,
			CheckInLat:      geo.Lat,
			CheckInLng:      geo.Lng,
			CheckInAccuracy: geo.AccuracyPtr(),
			CheckInGpsAt:    geo.GpsAt,
			CheckInGate:     gateName,
		}

		if err := s.attendanceRepo.Create(txCtx, attendance); err != nil {
			return err
		}

		// 7. Schedule the auto-reject task at the checkout deadline K+4h. It fires
		// only after this transaction commits (RegisterAfterCommit), so the
		// attendance row is durable. When it fires, the handler rejects the record
		// iff the employee still hasn't checked out.
		if s.taskEnqueuer != nil {
			attendanceID := attendance.ID
			deadline := shift.end.Add(checkOutUpperGrace)
			enqueuer := s.taskEnqueuer
			domain.RegisterAfterCommit(txCtx, func() {
				if err := enqueuer.EnqueueAutoRejectCheckout(attendanceID, deadline); err != nil {
					observability.GetLogger().Warn("failed to enqueue auto-reject checkout task",
						"attendance_id", attendanceID, "error", err)
				}
			})
		}

		result = attendance
		return nil
	})

	return result, err
}

// completeLegacyApprovedOpenAttendance closes a record written by the former
// status-only approval path before a new check-in is inserted. Running it in
// the check-in transaction makes the repair atomic with the insert that the
// stale open row would otherwise block.
func (s *AttendanceService) completeLegacyApprovedOpenAttendance(ctx context.Context, employeeID uint, before time.Time) error {
	att, err := s.attendanceRepo.GetApprovedOpenBefore(ctx, employeeID, before)
	if err != nil {
		return fmt.Errorf("failed to find approved open attendance: %w", err)
	}
	if att == nil {
		return nil
	}

	checkOutTime := att.CheckInTime
	if shift := s.resolveShiftForAttendance(ctx, att); shift != nil {
		checkOutTime = shift.end
	} else {
		// Historical payrate configuration may have been removed after approval.
		// The prior approval is still authoritative; persist a closed record rather
		// than stranding the employee behind an obsolete open-row constraint.
		observability.GetLogger().Warn("closing approved open attendance without historical shift configuration",
			"attendance_id", att.ID, "employee_id", att.EmployeeID)
	}

	updated, err := s.attendanceRepo.CompleteApprovedOpen(ctx, att.ID, checkOutTime, "admin")
	if err != nil {
		return fmt.Errorf("failed to complete approved open attendance: %w", err)
	}
	if updated {
		observability.GetLogger().Info("completed legacy admin-approved attendance before new check-in",
			"attendance_id", att.ID, "employee_id", att.EmployeeID, "check_out_time", checkOutTime)
	}
	return nil
}

// AdminCheckInShifts returns today's configured shifts for an employee's active
// assignment. Admin-created records intentionally stay within today's live
// checkout window: the employee, not the administrator, must still checkout
// from the configured geofence to complete the shift.
func (s *AttendanceService) AdminCheckInShifts(ctx context.Context, employeeID, projectID uint, day time.Time) ([]AdminCheckInShift, error) {
	_, shifts, err := s.adminCheckInShifts(ctx, employeeID, projectID, day)
	return shifts, err
}

func (s *AttendanceService) adminCheckInShifts(ctx context.Context, employeeID, projectID uint, day time.Time) (*domain.ProjectEmployee, []AdminCheckInShift, error) {
	now := s.clock.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, now.Location())
	if !day.Equal(today) {
		return nil, nil, domain.NewValidationError("Chỉ có thể tạo check-in cho hôm nay để nhân viên tự tan ca.")
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load project for admin check-in: %w", err)
	}
	if project == nil {
		return nil, nil, domain.NewNotFoundError("Không tìm thấy dự án")
	}
	if !project.IsFlexible {
		return nil, nil, domain.NewValidationError("Dự án không hỗ trợ chấm công linh hoạt")
	}

	assignment, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(ctx, projectID, employeeID)
	if err != nil {
		return nil, nil, err
	}
	if assignment == nil || !assignment.CheckInEnabled {
		return nil, nil, domain.NewValidationError("Nhân viên chưa được cấp quyền chấm công tại dự án này.")
	}

	payrate, err := s.payrateRepo.GetActiveByProjectAndDate(ctx, projectID, day)
	if err != nil {
		if !domain.IsNotFoundError(err) {
			return nil, nil, fmt.Errorf("failed to load payrate for admin check-in: %w", err)
		}
		payrate = nil
	}
	if payrate == nil {
		return nil, nil, domain.NewValidationError("Chưa có cấu hình mức lương hiệu lực cho ngày chấm công này.")
	}

	flattened, err := payrate.Payrate.Flatten()
	if err != nil {
		return nil, nil, domain.NewValidationError("Cấu hình ca làm việc không hợp lệ. Vui lòng kiểm tra mức lương dự án.")
	}
	position, _, parsed := resolveShifts(flattened, assignment.Position, day.Add(12*time.Hour))
	options := make([]AdminCheckInShift, 0, len(parsed))
	for _, shift := range parsed {
		if shift.start.Year() != day.Year() || shift.start.Month() != day.Month() || shift.start.Day() != day.Day() {
			continue
		}
		options = append(options, AdminCheckInShift{
			Label:    fmt.Sprintf("%s - %s", shift.start.Format("15:04"), shift.end.Format("15:04")),
			Start:    shift.start,
			End:      shift.end,
			Amount:   int64(shift.amount),
			Position: position,
		})
	}
	sort.Slice(options, func(i, j int) bool { return options[i].Start.Before(options[j].Start) })
	for i := range options {
		options[i].Index = i
	}
	if len(options) == 0 {
		return nil, nil, domain.NewValidationError("Chưa cấu hình ca làm việc cho vị trí này. Vui lòng kiểm tra mức lương dự án.")
	}
	return assignment, options, nil
}

// AdminCreateCheckIn records only a check-in at the selected configured shift
// start. It deliberately does not write checkout GPS, earning, or quota: the
// employee must use the normal checkout flow to finish the shift.
func (s *AttendanceService) AdminCreateCheckIn(ctx context.Context, employeeID, projectID uint, day time.Time, shiftIndex int) (*domain.Attendance, error) {
	var created *domain.Attendance
	err := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		now := s.clock.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if err := s.completeLegacyApprovedOpenAttendance(txCtx, employeeID, today); err != nil {
			return err
		}

		_, shifts, err := s.adminCheckInShifts(txCtx, employeeID, projectID, day)
		if err != nil {
			return err
		}
		if shiftIndex < 0 || shiftIndex >= len(shifts) {
			return domain.NewValidationError("Ca làm việc được chọn không hợp lệ.")
		}
		shift := shifts[shiftIndex]
		if shift.Start.After(now) {
			return domain.NewValidationError("Chỉ có thể tạo check-in cho ca đã bắt đầu.")
		}
		if now.After(shift.End.Add(checkOutUpperGrace)) {
			return domain.NewValidationError("Ca làm đã quá giờ tan ca; không thể tạo check-in để nhân viên tự tan ca.")
		}

		existing, err := s.attendanceRepo.GetByEmployeeAndDate(txCtx, employeeID, today)
		if err != nil {
			return fmt.Errorf("failed to check existing attendance: %w", err)
		}
		if existing != nil && !isConfirmedNoSalaryCheckout(existing) && !isAutoRejectedNoCheckout(existing) {
			return domain.NewValidationError("Nhân viên đã có check-in trong ngày hôm nay.")
		}

		attendance := &domain.Attendance{
			EmployeeID:  employeeID,
			ProjectID:   projectID,
			Date:        today,
			CheckInTime: shift.Start,
			CheckInGate: "admin",
		}
		if err := s.attendanceRepo.Create(txCtx, attendance); err != nil {
			return err
		}
		if s.taskEnqueuer != nil {
			attendanceID := attendance.ID
			deadline := shift.End.Add(checkOutUpperGrace)
			enqueuer := s.taskEnqueuer
			domain.RegisterAfterCommit(txCtx, func() {
				if err := enqueuer.EnqueueAutoRejectCheckout(attendanceID, deadline); err != nil {
					observability.GetLogger().Warn("failed to enqueue auto-reject for admin check-in",
						"attendance_id", attendanceID, "error", err)
				}
			})
		}
		created = attendance
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.attendanceRepo.GetByID(ctx, created.ID)
}

func (s *AttendanceService) CheckOut(ctx context.Context, employeeID uint, geo domain.GeoReading, confirmNoSalary bool) (*domain.Attendance, error) {
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
		if attendance.SalaryRejectReason != nil {
			// Auto-rejected by the checkout-window task — the shift is final/closed.
			return domain.NewValidationError("Ca làm việc đã bị tự động từ chối do quá giờ tan ca.")
		}
		if attendance.GetStatus(now) == domain.AttendanceStatusOrphaned {
			return domain.NewValidationError("Ca làm việc đã quá hạn tan ca")
		}

		// Resolve the worked shift to derive the checkout window [K-1h, K+4h] from
		// the configured shift end. Load the assignment + payrate here so they are
		// reused for earning below.
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
		var forcedNoSalaryReason *string
		if err := validateCheckOutWindow(shift, attendance.CheckInTime, now); err != nil {
			if !confirmNoSalary {
				return err
			}
			reason := fmt.Sprintf("%s %s", err.Error(), confirmedNoSalaryCheckoutReason)
			forcedNoSalaryReason = &reason
			// Salary-affecting employee self-service action: the employee explicitly
			// accepted zero pay to close a mistaken shift outside the checkout window.
			// Log it so the override is auditable (who/when/which shift) by ops/payroll.
			observability.GetLogger().Info(
				"attendance checkout override: employee confirmed no-salary checkout outside window",
				"attendance_id", attendance.ID,
				"employee_id", employeeID,
				"project_id", attendance.ProjectID,
				"check_in_time", attendance.CheckInTime,
				"check_out_time", now,
			)
		}

		// 2. Enforce the project geofence for checkout using the same validation
		// contract as check-in. A successful check-in does not grant a later
		// checkout from outside the configured area.
		project, err := s.projectRepo.GetByID(txCtx, attendance.ProjectID)
		if err != nil {
			return fmt.Errorf("failed to load project for geofence validation: %w", err)
		}
		gateName, err := s.validateGeofence(project, geo)
		if err != nil {
			return err
		}

		// 3. Update CheckOutTime and coords
		attendance.CheckOutTime = &now
		attendance.CheckOutLat = &geo.Lat
		attendance.CheckOutLng = &geo.Lng
		attendance.CheckOutAccuracy = geo.AccuracyPtr()
		attendance.CheckOutGpsAt = geo.GpsAt
		attendance.CheckOutGate = &gateName

		// 4. Calculate earning_amount (reuses the assignment + payrate loaded for
		// the earliest-checkout check above).
		if !assignment.CheckInEnabled {
			return domain.NewValidationError("Chấm công đã bị vô hiệu hóa. Vui lòng liên hệ quản lý.")
		}

		var earningAmount int64
		var salaryRejectReason *string
		if forcedNoSalaryReason != nil {
			earningAmount = 0
			salaryRejectReason = forcedNoSalaryReason
		} else if payrate == nil {
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
		var quotaCreditEligibleAt *time.Time
		if earningAmount > 0 {
			eligibleAt := now.Add(s.selfCheckInAdvanceHoldDuration(txCtx))
			quotaCreditEligibleAt = &eligibleAt
			attendance.QuotaCreditEligibleAt = quotaCreditEligibleAt
		}

		// 5. Update attendance record
		if err := s.attendanceRepo.Update(txCtx, attendance); err != nil {
			return err
		}

		// 6. Defer the quota credit by the configured post-checkout hold so the
		// earning stays pending before it becomes advanceable. The
		// earning stays parked on the attendance row (earning_amount, persisted
		// above) with quota_credited_at = NULL; the credit task banks it later via
		// CreditAttendanceQuota, which derives forMonth from the check-out date and
		// is idempotent on quota_credited_at. Enqueued after-commit so the task can
		// only fire once the attendance row is durable. A lost task is recovered by
		// the periodic CreditOverduePendingQuota sweep.
		if quotaCreditEligibleAt != nil && s.taskEnqueuer != nil {
			attendanceID := attendance.ID
			fireAt := *quotaCreditEligibleAt
			enqueuer := s.taskEnqueuer
			domain.RegisterAfterCommit(txCtx, func() {
				if err := enqueuer.EnqueueCreditQuota(attendanceID, fireAt); err != nil {
					observability.GetLogger().Warn("failed to enqueue quota credit task",
						"attendance_id", attendanceID, "error", err)
				}
			})
		}

		result = attendance
		return nil
	})

	return result, err
}

// CancelCurrentAttendance lets an employee void their current open check-in when
// they entered the wrong shift. It records zero earning and a reject reason, then
// the existing check-in path allows a corrected check-in afterward because the
// record is now a no-checkout rejection.
func (s *AttendanceService) CancelCurrentAttendance(ctx context.Context, employeeID uint) (*domain.Attendance, error) {
	var result *domain.Attendance

	err := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		now := s.clock.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

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
		}
		if attendance == nil {
			return domain.NewValidationError("Không tìm thấy ca đang làm để hủy")
		}
		if attendance.EmployeeID != employeeID {
			return domain.NewValidationError("Không thể hủy ca của nhân viên khác")
		}
		if attendance.IsCompleted() {
			return domain.NewValidationError("Ca này đã tan ca, không thể hủy")
		}
		if attendance.SalaryRejectReason != nil {
			return domain.NewValidationError("Ca này đã được hủy hoặc từ chối")
		}
		if attendance.GetStatus(now) == domain.AttendanceStatusOrphaned {
			return domain.NewValidationError("Ca làm việc đã quá hạn tan ca")
		}

		ok, err := s.attendanceRepo.MarkAutoRejected(txCtx, attendance.ID, employeeCancelledWrongShiftReason)
		if err != nil {
			return fmt.Errorf("failed to cancel attendance: %w", err)
		}
		if !ok {
			return domain.NewValidationError("Ca này đã được cập nhật, vui lòng tải lại")
		}

		zero := int64(0)
		reason := employeeCancelledWrongShiftReason
		attendance.EarningAmount = &zero
		attendance.SalaryRejectReason = &reason
		result = attendance
		return nil
	})

	return result, err
}

// autoRejectExpiredReasonFallback is the salary_reject_reason recorded when the
// configured shift cannot be resolved for an expired-checkout attendance (e.g.,
// payrate deleted or assignment ended after check-in). The normal path uses
// formatAutoRejectReason with the actual check-in time, configured shift end
// (K), and grace-window upper bound (K+4h, "hạn chót").
const autoRejectExpiredReasonFallback = "Đã hết hạn tan ca — bạn đã quá giờ checkout cho ca này. Vui lòng liên hệ quản lý."

// formatAutoRejectReason builds the salary_reject_reason recorded when a
// checkout window [K-1h, K+4h] closes with no checkout. It points the employee at
// their actual check-in time, the configured shift end (K), and the grace
// deadline (K+4h) so they can see exactly when they should have ended the shift.
func formatAutoRejectReason(checkInTime, shiftEnd time.Time) string {
	deadline := shiftEnd.Add(checkOutUpperGrace)
	return fmt.Sprintf(
		"Đã hết hạn tan ca (Vào làm: %s; Tan ca: %s (hạn chót %s))",
		checkInTime.Format("15:04"),
		shiftEnd.Format("15:04"),
		deadline.Format("15:04"),
	)
}

// autoRejectReasonFor resolves the configured shift for an attendance and
// formats the reject reason with the actual check-in / shift-end / deadline
// times. Falls back to the generic message when the shift cannot be resolved
// (payrate missing, assignment ended, no shift configured) so the rejection
// is still informative. Errors from the lookup are logged and swallowed —
// the rejection itself is more important than the reason text.
func (s *AttendanceService) autoRejectReasonFor(ctx context.Context, att *domain.Attendance) string {
	shift := s.resolveShiftForAttendance(ctx, att)
	if shift == nil {
		return autoRejectExpiredReasonFallback
	}
	return formatAutoRejectReason(att.CheckInTime, shift.end)
}

// resolveShiftForAttendance loads the payrate and active assignment for the
// attendance and resolves the shift. Returns nil on any miss (payrate missing,
// assignment ended, no shift configured) — the caller should fall back to a
// generic reason in that case.
func (s *AttendanceService) resolveShiftForAttendance(ctx context.Context, att *domain.Attendance) *parsedShift {
	payrate, err := s.payrateRepo.GetActiveByProjectAndDate(ctx, att.ProjectID, att.Date)
	if err != nil {
		if !domain.IsNotFoundError(err) {
			observability.GetLogger().Warn("auto-reject: failed to load payrate",
				"attendance_id", att.ID, "error", err)
		}
		return nil
	}
	if payrate == nil {
		return nil
	}
	assignment, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(ctx, att.ProjectID, att.EmployeeID)
	if err != nil {
		if !domain.IsNotFoundError(err) {
			observability.GetLogger().Warn("auto-reject: failed to load assignment",
				"attendance_id", att.ID, "error", err)
		}
		return nil
	}
	if assignment == nil {
		return nil
	}
	return s.resolveShift(payrate, assignment.Position, att.CheckInTime)
}

// AutoRejectIfExpired finalizes an attendance whose checkout window [K-1h, K+4h]
// has closed with no checkout: it sets earning to 0 and records a reject
// reason, making the shift final. Idempotent — the underlying MarkAutoRejected
// is a conditional UPDATE (WHERE check_out_time IS NULL AND
// salary_reject_reason IS NULL AND review_action IS NULL), so it is a safe
// no-op if the employee checked out, the record was rejected/reviewed, or a
// concurrent CheckOut/Admin review beats it. Invoked by the asynq auto-reject
// task scheduled at K+4h at check-in.
func (s *AttendanceService) AutoRejectIfExpired(ctx context.Context, attendanceID uint) error {
	att, err := s.attendanceRepo.GetByID(ctx, attendanceID)
	if err != nil {
		return fmt.Errorf("failed to load attendance: %w", err)
	}
	if att == nil || att.CheckOutTime != nil || att.SalaryRejectReason != nil || att.IsReviewed() {
		return nil
	}

	reason := s.autoRejectReasonFor(ctx, att)
	updated, err := s.attendanceRepo.MarkAutoRejected(ctx, attendanceID, reason)
	if err != nil {
		return fmt.Errorf("failed to auto-reject attendance: %w", err)
	}
	if updated {
		observability.GetLogger().Info("Auto-rejected attendance after checkout window expired",
			"attendance_id", attendanceID)
	}
	return nil
}

// Approve records a manual admin approval of a disputed attendance and
// recomputes the earning for the full configured shift. It clears any prior
// salary_reject_reason (restoring the row to a payable state) and stamps the
// review audit. Idempotent — a consistent approved row is a no-op, while a
// legacy row contradicted by a delayed auto-reject is recomputed and repaired.
// No transaction: a single conditional UPDATE is atomic by itself, mirroring
// AutoRejectIfExpired.
//
// The earning recompute uses the configured shift end K as the effective
// checkout. K is always inside the checkout window [K-grace, K+grace], so the
// payrate shift-match in calculateEarningAmount resolves deterministically and
// pays the full shift the employee was disputed out of.
func (s *AttendanceService) Approve(ctx context.Context, attendanceID, adminID uint, note string) (*domain.Attendance, error) {
	att, err := s.attendanceRepo.GetByID(ctx, attendanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load attendance: %w", err)
	}
	if att == nil {
		return nil, domain.NewNotFoundError("Không tìm thấy bản ghi chấm công")
	}
	if att.IsApproved() && att.CheckOutTime != nil && att.SalaryRejectReason == nil && att.EarningAmount != nil && *att.EarningAmount > 0 {
		// Already approved. The earning may still be un-credited if the credit
		// step failed on a prior call (CreditAttendanceQuota is idempotent and
		// skips already-credited rows), so re-attempt it before returning.
		if _, err := s.CreditAttendanceQuota(ctx, attendanceID); err != nil {
			return nil, fmt.Errorf("failed to credit quota on re-approve: %w", err)
		}
		// Already in the requested state — return the current record.
		return s.attendanceRepo.GetByID(ctx, attendanceID)
	}

	// Recompute earning for the full configured shift.
	shift := s.resolveShiftForAttendance(ctx, att)
	if shift == nil {
		return nil, domain.NewValidationError("Không xác định được ca làm việc để tính lại lương. Vui lòng kiểm tra cấu hình mức lương/ca cho dự án này.")
	}
	payrate, err := s.payrateRepo.GetActiveByProjectAndDate(ctx, att.ProjectID, att.Date)
	if err != nil {
		return nil, fmt.Errorf("failed to load payrate for approve: %w", err)
	}
	if payrate == nil {
		return nil, domain.NewValidationError("Chưa có cấu hình mức lương hiệu lực cho ngày chấm công này.")
	}
	assignment, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(ctx, att.ProjectID, att.EmployeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to load assignment for approve: %w", err)
	}
	if assignment == nil {
		return nil, domain.NewValidationError("Không tìm thấy phân công của nhân viên trên dự án này.")
	}

	earning, reason, err := s.calculateEarningAmount(payrate, assignment.Position, att.CheckInTime, shift.end)
	if err != nil {
		return nil, fmt.Errorf("failed to recompute earning on approve: %w", err)
	}
	if earning <= 0 {
		// The shift couldn't be matched even at its own configured end — surface
		// the payrate-engine reason verbatim so the admin understands why pay
		// can't be restored.
		msg := reason
		if msg == "" {
			msg = "Không thể tính lại lương cho ca này. Vui lòng kiểm tra cấu hình ca làm việc."
		}
		return nil, domain.NewValidationError(msg)
	}

	adminGate := "admin"
	updated, err := s.attendanceRepo.MarkAdminReviewed(
		ctx, attendanceID, domain.AttendanceReviewActionApproved, note, adminID, s.clock.Now(), &earning, &shift.end, &adminGate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mark attendance approved: %w", err)
	}
	if !updated {
		return nil, domain.NewNotFoundError("Không tìm thấy bản ghi chấm công")
	}

	// Admin approval credits the quota immediately, bypassing the configured hold that
	// the self-check-out path applies. CreditAttendanceQuota is idempotent on
	// quota_credited_at, so a re-approve (caught by the IsApproved guard above)
	// will not double-bank the earning.
	if _, err := s.CreditAttendanceQuota(ctx, attendanceID); err != nil {
		return nil, fmt.Errorf("failed to credit approved attendance: %w", err)
	}

	// Reload with associations so the response mapper has Employee/Project.
	return s.attendanceRepo.GetByID(ctx, attendanceID)
}

// Reject records a manual admin rejection of an attendance, zeroing the earning
// and storing the reason in salary_reject_reason (so GetStatus stays consistent)
// plus the review audit. Idempotent — re-rejecting an already-rejected row is a
// no-op. note is required.
func (s *AttendanceService) Reject(ctx context.Context, attendanceID, adminID uint, note string) (*domain.Attendance, error) {
	att, err := s.attendanceRepo.GetByID(ctx, attendanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load attendance: %w", err)
	}
	if att == nil {
		return nil, domain.NewNotFoundError("Không tìm thấy bản ghi chấm công")
	}
	if att.IsRejectedByAdmin() {
		return att, nil
	}
	if att.IsApproved() && att.SalaryRejectReason == nil {
		return nil, domain.NewValidationError("Ca làm đã được duyệt và hoàn thành, không thể từ chối lại.")
	}
	if att.QuotaCreditedAt != nil {
		return nil, domain.NewValidationError("Thu nhập của ca làm đã được cộng vào hạn mức ứng lương, không thể từ chối lại.")
	}

	zero := int64(0)
	updated, err := s.attendanceRepo.MarkAdminReviewed(
		ctx, attendanceID, domain.AttendanceReviewActionRejected, note, adminID, s.clock.Now(), &zero, nil, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mark attendance rejected: %w", err)
	}
	if !updated {
		latest, reloadErr := s.attendanceRepo.GetByID(ctx, attendanceID)
		if reloadErr != nil {
			return nil, fmt.Errorf("failed to reload attendance after rejected transition conflict: %w", reloadErr)
		}
		if latest == nil {
			return nil, domain.NewNotFoundError("Không tìm thấy bản ghi chấm công")
		}
		return nil, domain.NewValidationError("Ca làm đã hoàn thành hoặc thu nhập đã được cộng vào hạn mức ứng lương, không thể từ chối lại.")
	}

	return s.attendanceRepo.GetByID(ctx, attendanceID)
}

// autoRejectSweepLookback bounds the fallback sweep to recent records so it does
// not backfill ancient history. autoRejectSweepMinAge matches the orphan
// threshold (18h) so only records past any plausible shift window are finalized.
const (
	autoRejectSweepLookback = 7 * 24 * time.Hour
	autoRejectSweepMinAge   = 18 * time.Hour
)

// AutoRejectSweep is the safety-net backstop for AutoRejectIfExpired. It
// finalizes attendance records whose checkout window closed with no checkout but
// were never auto-rejected — which happens when the per-attendance task
// scheduled at check-in was lost (Redis unavailable at commit time, or a process
// crash between commit and the after-commit enqueue). Each finalization uses the
// race-free MarkAutoRejected, so in-flight checkouts and prior rejections are
// safe no-ops. Returns the count of records finalized this pass.
func (s *AttendanceService) AutoRejectSweep(ctx context.Context) (int, error) {
	now := s.clock.Now()
	candidates, err := s.attendanceRepo.GetOrphanCandidates(ctx, now.Add(-autoRejectSweepLookback), now.Add(-autoRejectSweepMinAge))
	if err != nil {
		return 0, fmt.Errorf("failed to load auto-reject sweep candidates: %w", err)
	}
	rejected := 0
	for _, att := range candidates {
		reason := s.autoRejectReasonFor(ctx, att)
		updated, err := s.attendanceRepo.MarkAutoRejected(ctx, att.ID, reason)
		if err != nil {
			// One bad row must not abort the whole sweep; the next pass retries it.
			observability.GetLogger().Warn("auto-reject sweep: failed to reject attendance",
				"attendance_id", att.ID, "error", err)
			continue
		}
		if updated {
			rejected++
		}
	}
	if rejected > 0 {
		observability.GetLogger().Info("Auto-reject sweep finalized attendances",
			"count", rejected, "candidates", len(candidates))
	}
	return rejected, nil
}

// CreditAttendanceQuota banks an attendance's earning into the advance-payment
// quota pool: it bumps advance_payments.salary and recomputes max_adv_amount
// (= floor(salary * configured_percent / 100)). It is the single entry point
// for crediting and
// is shared by the deferred self-check-out task, the admin Approve
// path (immediate credit), and the safety-net sweep.
//
// Idempotent and race-free: it claims the credit via a conditional
// MarkQuotaCredited (still payable and quota_credited_at IS NULL) inside a transaction, and
// only banks the earning if the claim succeeded. Two concurrent credits for the
// same attendance serialize on the row lock; the loser's MarkQuotaCredited
// returns RowsAffected=0 and skips the bank, so the earning is never double-
// counted. forMonth is derived from the check-out DATE (not credit-time now) so
// a check-out near month-end credits to the correct month even when its hold
// crosses into the next month. Returns true when this call banked a
// credit, false for a no-op (already credited / nothing to credit / not found).
func (s *AttendanceService) CreditAttendanceQuota(ctx context.Context, attendanceID uint) (bool, error) {
	return s.creditAttendanceQuota(ctx, attendanceID, false)
}

// CreditScheduledAttendanceQuota credits an earning only after the immutable
// deadline persisted at checkout. It is used by the deferred worker and sweep;
// manual admin approval deliberately uses CreditAttendanceQuota to bypass it.
func (s *AttendanceService) CreditScheduledAttendanceQuota(ctx context.Context, attendanceID uint) (bool, error) {
	return s.creditAttendanceQuota(ctx, attendanceID, true)
}

func (s *AttendanceService) creditAttendanceQuota(ctx context.Context, attendanceID uint, requireEligibility bool) (bool, error) {
	credited := false
	err := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		advancePercent := domain.DefaultSelfCheckInAdvancePercentage
		if s.settingsConfig != nil {
			var err error
			advancePercent, err = s.settingsConfig.GetSelfCheckInAdvancePercentageForUpdate(txCtx)
			if err != nil {
				return fmt.Errorf("failed to lock self check-in advance percentage: %w", err)
			}
		}

		att, err := s.attendanceRepo.GetByID(txCtx, attendanceID)
		if err != nil {
			return fmt.Errorf("failed to load attendance for quota credit: %w", err)
		}
		if att == nil {
			return nil
		}
		// Nothing earnable to bank, or already banked by a concurrent credit.
		if att.EarningAmount == nil || *att.EarningAmount <= 0 {
			return nil
		}
		if att.QuotaCreditedAt != nil {
			return nil
		}
		if requireEligibility {
			eligibleAt := att.QuotaCreditEligibleAt
			if eligibleAt == nil && att.CheckOutTime != nil {
				legacyEligibleAt := att.CheckOutTime.Add(domain.QuotaCreditHoldDuration)
				eligibleAt = &legacyEligibleAt
			}
			if eligibleAt == nil || s.clock.Now().Before(*eligibleAt) {
				return nil
			}
		}

		// Claim first. The conditional UPDATE also verifies the current row is
		// still payable, so a concurrent admin rejection blocks stale credit.
		claimed, err := s.attendanceRepo.MarkQuotaCredited(txCtx, attendanceID, s.clock.Now())
		if err != nil {
			return fmt.Errorf("failed to claim quota credit: %w", err)
		}
		if !claimed {
			return nil
		}

		// Derive the salary month from the work date, not credit-time now: a
		// A check-out near month end can credit in the next month but belongs to
		// its original quota month.
		effective := att.Date
		if att.CheckOutTime != nil {
			effective = *att.CheckOutTime
		}
		forMonth := effective.Format("2006-01")
		uploadDate := effective.Format("2006-01-02")

		aps, err := s.advancePaymentRepo.GetByEmployeeAndMonth(txCtx, uint64(att.EmployeeID), forMonth)
		if err != nil {
			return fmt.Errorf("failed to load advance payment row: %w", err)
		}
		var ap *domain.AdvancePayment
		for _, v := range aps {
			if v.ProjectID == att.ProjectID {
				ap = v
				break
			}
		}

		earning := uint64(*att.EarningAmount)
		if ap == nil {
			ap = &domain.AdvancePayment{
				ProjectID:    att.ProjectID,
				EmployeeID:   att.EmployeeID,
				ForMonth:     forMonth,
				UploadDate:   uploadDate,
				Salary:       earning,
				MaxAdvAmount: (earning * advancePercent) / 100,
			}
			if err := s.advancePaymentRepo.Create(txCtx, ap); err != nil {
				return fmt.Errorf("failed to create advance payment row: %w", err)
			}
		} else {
			if err := s.advancePaymentRepo.AccumulateSalary(txCtx, uint64(ap.ID), *att.EarningAmount, advancePercent); err != nil {
				return fmt.Errorf("failed to accumulate salary: %w", err)
			}
		}
		credited = true
		return nil
	})
	return credited, err
}

func (s *AttendanceService) selfCheckInAdvanceHoldDuration(ctx context.Context) time.Duration {
	if settingsConfig, ok := s.settingsConfig.(selfCheckInAdvanceHoldDurationProvider); ok {
		return settingsConfig.GetSelfCheckInAdvanceHoldDuration(ctx)
	}
	return domain.QuotaCreditHoldDuration
}

// quotaCreditSweepBatch caps the number of records one sweep pass finalizes, so
// the safety net stays bounded even if a long outage leaves many pending credits.
const quotaCreditSweepBatch = 500

// CreditOverduePendingQuota is the safety-net backstop for the deferred
// quota-credit task. It banks earnings whose persisted checkout deadline has
// elapsed but were never credited — which happens when the per-attendance task scheduled at check-out
// was lost (Redis unavailable at commit time, or a process crash between commit
// and the after-commit enqueue). Each credit uses the idempotent
// CreditAttendanceQuota, so in-flight tasks and overlapping runs are safe
// no-ops. Returns the count of records banked this pass.
func (s *AttendanceService) CreditOverduePendingQuota(ctx context.Context) (int, error) {
	ids, err := s.attendanceRepo.GetOverdueQuotaCreditCandidates(ctx, s.clock.Now(), quotaCreditSweepBatch)
	if err != nil {
		return 0, fmt.Errorf("failed to load overdue quota-credit candidates: %w", err)
	}
	credited := 0
	for _, id := range ids {
		banked, err := s.CreditScheduledAttendanceQuota(ctx, id)
		if err != nil {
			// One bad row must not abort the whole sweep; the next pass retries it.
			observability.GetLogger().Warn("quota-credit sweep: failed to credit attendance",
				"attendance_id", id, "error", err)
			continue
		}
		if banked {
			credited++
		}
	}
	if credited > 0 {
		observability.GetLogger().Info("Quota-credit sweep banked pending earnings",
			"count", credited, "candidates", len(ids))
	}
	return credited, nil
}

// GetTodayAttendance returns the attendance record for the employee for the
// current day. For night shifts checked in before midnight, it falls back to an
// open previous-day attendance so the mobile card can still show "Tan ca" after
// midnight. It returns nil if no current-day or open previous-day record exists.
func (s *AttendanceService) GetTodayAttendance(ctx context.Context, employeeID uint) (*domain.Attendance, error) {
	now := s.clock.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	att, err := s.attendanceRepo.GetByEmployeeAndDate(ctx, employeeID, today)
	if err != nil {
		return nil, err
	}
	if att != nil {
		s.normalizeLegacySalaryRejectReason(ctx, att)
		if isAutoRejectedNoCheckout(att) {
			return nil, nil
		}
		return att, nil
	}

	yesterday := today.AddDate(0, 0, -1)
	att, err = s.attendanceRepo.GetByEmployeeAndDate(ctx, employeeID, yesterday)
	if err != nil {
		return nil, err
	}
	if att == nil || att.CheckOutTime != nil || att.SalaryRejectReason != nil {
		return nil, nil
	}
	if att.IsApproved() {
		// Older approvals were stored as a logical completion only. Repair that
		// physical open row before returning the mobile read model so the employee
		// can immediately begin today's shift and the database no longer retains
		// a stale open check-in.
		if err := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
			return s.completeLegacyApprovedOpenAttendance(txCtx, employeeID, today)
		}); err != nil {
			return nil, err
		}
		return nil, nil
	}
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
		closest.end.Add(-checkOutLowerGrace).Format("15:04"),
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

// ResolveShiftWindow is retained for callers that only need the check-in
// advisory window. New callers that also need checkout guidance should use
// ResolveShiftWindows.
func ResolveShiftWindow(flattened map[string]int, position string, now time.Time) (shiftStart, shiftEnd, windowStart, windowEnd time.Time, ok bool) {
	shiftStart, shiftEnd, windowStart, windowEnd, _, _, ok = ResolveShiftWindows(flattened, position, now)
	return shiftStart, shiftEnd, windowStart, windowEnd, ok
}

// ResolveShiftWindows derives both advisory attendance windows for display on
// the employee portal. It reuses the same resolveShifts → closestShift logic as
// CheckIn and CheckOut, including cross-midnight shift resolution. The returned
// checkout bounds are [shift.end - 1h, shift.end + 4h], matching
// validateCheckOutWindow exactly.
func ResolveShiftWindows(flattened map[string]int, position string, now time.Time) (shiftStart, shiftEnd, checkInWindowStart, checkInWindowEnd, checkOutWindowStart, checkOutWindowEnd time.Time, ok bool) {
	_, _, shifts := resolveShifts(flattened, position, now)
	if len(shifts) == 0 {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, time.Time{}, time.Time{}, false
	}
	shift := closestShift(shifts, now)
	if shift == nil {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, time.Time{}, time.Time{}, false
	}
	return shift.start,
		shift.end,
		shift.start.Add(-checkInShiftWindow),
		shift.start.Add(checkInShiftWindow),
		shift.end.Add(-checkOutLowerGrace),
		shift.end.Add(checkOutUpperGrace),
		true
}

// ResolveAllShiftWindows returns one advisory window for every configured shift
// for the resolved position. The absolute date is anchored to now, but callers
// should display the time-of-day values; duplicate ±1-day resolver candidates
// are removed. Cross-midnight end and checkout times retain their correct
// following-day instants.
//
// names optionally maps a "HH:MM-HH:MM" time-range to an admin-chosen display
// name; when present, the matching ShiftWindow.Name is populated. Pass nil when
// the caller has no name config (windows come back with empty Name).
func ResolveAllShiftWindows(flattened map[string]int, position string, now time.Time, names map[string]string) []ShiftWindow {
	_, _, shifts := resolveShifts(flattened, position, now)
	windowsByTimeRange := make(map[string]ShiftWindow)
	for _, shift := range shifts {
		if shift.start.Year() != now.Year() || shift.start.YearDay() != now.YearDay() {
			continue
		}
		key := shift.start.Format("15:04") + "-" + shift.end.Format("15:04")
		window := ShiftWindow{
			ShiftStart:          shift.start,
			ShiftEnd:            shift.end,
			CheckInWindowStart:  shift.start.Add(-checkInShiftWindow),
			CheckInWindowEnd:    shift.start.Add(checkInShiftWindow),
			CheckOutWindowStart: shift.end.Add(-checkOutLowerGrace),
			CheckOutWindowEnd:   shift.end.Add(checkOutUpperGrace),
		}
		if names != nil {
			window.Name = names[key]
		}
		windowsByTimeRange[key] = window
	}

	windows := make([]ShiftWindow, 0, len(windowsByTimeRange))
	for _, window := range windowsByTimeRange {
		windows = append(windows, window)
	}
	sort.Slice(windows, func(i, j int) bool {
		return windows[i].ShiftStart.Before(windows[j].ShiftStart)
	})
	return windows
}

// ExtractShiftRanges returns the set of distinct "HH:MM-HH:MM" shift time-ranges
// configured in the flattened payrate, across every position. Used by project
// create/update handlers to validate that admin-named shifts (Project.ShiftNames)
// correspond to shifts the payrate actually defines. The returned slice is
// deduplicated but unordered.
func ExtractShiftRanges(flattened map[string]int) []string {
	seen := make(map[string]struct{})
	ranges := make([]string, 0)
	for key := range flattened {
		parts := strings.Split(key, ".")
		if len(parts) < 3 {
			continue
		}
		timeRange := parts[len(parts)-1]
		if _, ok := seen[timeRange]; ok {
			continue
		}
		if domain.IsValidShiftRange(timeRange) {
			seen[timeRange] = struct{}{}
			ranges = append(ranges, timeRange)
		}
	}
	return ranges
}
