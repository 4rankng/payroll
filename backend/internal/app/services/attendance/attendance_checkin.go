package attendance

import (
	"context"
	"fmt"
	"sort"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

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
			if assignment.HasPendingCheckInEnable() && assignment.CheckInEffectiveFrom != nil {
				return domain.NewValidationError(fmt.Sprintf(
					"Dịch vụ tự chấm công sẽ kích hoạt từ %s.",
					assignment.CheckInEffectiveFrom.Format("02/01"),
				))
			}
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
