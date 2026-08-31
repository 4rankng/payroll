package attendance

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

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

// CreditAttendanceQuotaNow banks a completed self-check-out's earning into the
// advance-payment quota pool immediately, skipping the configured post-checkout
// hold. It is the admin "credit now" action for healthy completed shifts: unlike
// Approve it does not recompute the earning or overwrite the employee's
// check-out. Rows with nothing payable are rejected; already-credited rows are
// an idempotent success. Returns the reloaded attendance so the response mapper
// sees the stamped quota_credited_at.
func (s *AttendanceService) CreditAttendanceQuotaNow(ctx context.Context, attendanceID, adminID uint) (*domain.Attendance, error) {
	att, err := s.attendanceRepo.GetByID(ctx, attendanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load attendance: %w", err)
	}
	if att == nil {
		return nil, domain.NewNotFoundError("Không tìm thấy bản ghi chấm công")
	}
	if att.CheckOutTime == nil || att.EarningAmount == nil || *att.EarningAmount <= 0 {
		return nil, domain.NewValidationError("Ca này chưa có thu nhập để cộng hạn mức")
	}
	if att.QuotaCreditedAt != nil {
		// Already banked — idempotent success; return the current record.
		return att, nil
	}
	if _, err := s.CreditAttendanceQuota(ctx, attendanceID); err != nil {
		return nil, fmt.Errorf("failed to credit attendance quota: %w", err)
	}
	observability.GetLogger().Info("admin credited attendance quota before hold elapsed",
		"attendance_id", attendanceID,
		"employee_id", att.EmployeeID,
		"admin_id", adminID,
		"earning_amount", *att.EarningAmount,
	)
	// Reload with associations so the response mapper has Employee/Project.
	return s.attendanceRepo.GetByID(ctx, attendanceID)
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
