package attendance

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

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
