package attendance

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

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
