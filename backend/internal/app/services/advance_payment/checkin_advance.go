package advance_payment

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

	"github.com/pkg/errors"
)

// Self-check-in advance flow — DEDICATED PATH (Option B).
//
// This flow is separate from the admin-upload (BCC) advance flow:
//   - Quota source: check-out earnings (advance_payments.salary = 100% earned;
//     max_adv_amount = floor(salary * configured_percent / 100); default 70%).
//   - Salary period: calendar month (project.SalaryPeriodFrom), NOT the admin
//     day-20 three-phase model.
//   - Request window: opens on SelfCheckInAdvanceWindowOpenDay (day 10) and closes
//     at month end; the previous salary period is locked after month rollover.
//
// The admin GetEmployeeAdvanceInfo / CreateRequest are untouched.

// SelfCheckInAdvanceWindowOpenDay is the day of the calendar month from which a
// self-check-in employee may request an advance for the current salary period.
// Before this day the employee may still check-in/out and earn, but cannot submit
// an advance request. Centralized alongside the configurable self-check-in
// advance percentage.
const SelfCheckInAdvanceWindowOpenDay = 10

// CheckInAdvanceDisclaimerVN is the in-app note clarifying that the displayed
// check-in wages exclude overtime and company allowances.
const CheckInAdvanceDisclaimerVN = "Ngày công này chưa bao gồm tính tăng ca, và các khoản phụ cấp khác của công ty."

// minCheckInAdvanceRequest is the minimum advance amount (VND), matching the
// admin flow's floor.
const minCheckInAdvanceRequest uint64 = 10000

// CheckInAdvanceInfo is the self-check-in advance summary returned to the employee
// app. Salary = tiền công thực tế (100% earned); MaxAdvanceAmount = tiền công được
// ứng (the configured self-check-in percentage, default 70%). Fee/transfer
// fields are filled by the handler from the service config.
// CheckInAdvanceInfo is an INTERNAL service-level value — it is NOT serialized to the
// client; the handler maps it field-by-field into dto.AdvancePaymentInfoResponse (which
// carries the canonical camelCase json tags). Do not add json tags here.
type CheckInAdvanceInfo struct {
	ForMonth string
	Salary   uint64 // tiền công thực tế (100% earned, already credited)
	// PendingEarnings is the total earning held in the configured credit window this
	// month — checked-out attendances whose earning has not yet been banked into
	// the quota pool. Shown separately so the worker sees money is coming; it is
	// NOT part of MaxAdvanceAmount / the configured advanceable cap.
	PendingEarnings           uint64
	MaxAdvanceAmount          uint64 // tiền công được ứng (configured percent of credited salary)
	CompletedAmount           uint64
	PendingAmount             uint64
	RemainingAmount           uint64
	CanRequest                bool
	CanRequestTitle           string
	CanRequestReason          string
	WindowOpenDay             int
	Disclaimer                string
	HasFlexible               bool
	FeePercentage             float64
	MinFee                    uint64
	ProviderMinTransferAmount uint64
	ProviderMaxTransferAmount uint64
}

// checkInRequestPeriod returns the salary period a self-check-in employee may
// request at time now, and whether that period's window is open.
//
//   - days 1-8 (the previous period's tail): the PREVIOUS month, open — the
//     same tail the regular endpoint serves, so a check-in period keeps exactly
//     one request path whichever side of the month we are on.
//   - day 9: locked, matching the regular flow's gap day.
//   - days 10+: the CURRENT month, open from SelfCheckInAdvanceWindowOpenDay
//     through month end.
//
// Prior months are always locked: a period cannot be requested after it rolls
// past its window.
func checkInRequestPeriod(now time.Time) (forMonth string, open bool) {
	currentCalMonth := now.Format("2006-01")
	if IsBeforeCutoff(now) {
		return now.AddDate(0, -1, 0).Format("2006-01"), true
	}
	return currentCalMonth, now.Day() >= SelfCheckInAdvanceWindowOpenDay
}

// isCheckInRequestWindowOpen reports whether a self-check-in employee may
// request an advance for forMonth at time now.
func isCheckInRequestWindowOpen(now time.Time, forMonth string) bool {
	period, open := checkInRequestPeriod(now)
	return open && forMonth == period
}

// GetCheckInAdvanceInfo returns the self-check-in advance summary for an employee.
func (s *Service) GetCheckInAdvanceInfo(ctx context.Context, employeeID uint64) (*CheckInAdvanceInfo, error) {
	eligibility, err := s.getAdvanceEligibility(ctx, employeeID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check payment schedule")
	}

	now := clock.Now()
	// The screen must describe the period the employee can actually request
	// right now: the previous month during the days 1-8 tail, the current month
	// from day 10, nothing in between.
	servedMonth, windowOpen := checkInRequestPeriod(now)

	info := &CheckInAdvanceInfo{
		ForMonth:      servedMonth,
		WindowOpenDay: SelfCheckInAdvanceWindowOpenDay,
		Disclaimer:    CheckInAdvanceDisclaimerVN,
		HasFlexible:   eligibility.hasFlexible,
	}

	// Kill switch ("tạm ngừng ứng lương") wins over every other CanRequest
	// reason, but data population still runs so the employee keeps seeing
	// their real salary/remaining figures.
	applyPaused := func() {
		info.CanRequest = false
		info.CanRequestTitle = constants.MsgAdvanceRequestPausedTitleVN
		info.CanRequestReason = constants.MsgAdvanceRequestPausedReasonVN
	}

	if !eligibility.hasCheckInEnabled {
		info.CanRequest = false
		info.CanRequestTitle = "Chưa bật dịch vụ tự chấm công"
		info.CanRequestReason = "Tài khoản chưa được cấp quyền sử dụng dịch vụ tự chấm công. Vui lòng liên hệ quản lý."
		if eligibility.advanceRequestDisabled {
			applyPaused()
		}
		return info, nil
	}

	// Salary (100% earned) + advanceable cap for the served salary period,
	// summed across projects in a single query.
	salary, maxAdv, err := s.config.AdvancePaymentRepo.SumSalaryAndMaxAdvByEmployeeMonth(ctx, employeeID, servedMonth)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get salary and max advance")
	}
	// Earnings still inside the configured credit window this month — displayed
	// separately; not part of the advanceable cap.
	pendingEarnings, err := s.config.AdvancePaymentRepo.SumPendingEarningsByEmployeeMonth(ctx, employeeID, servedMonth)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pending earnings")
	}
	completed, err := s.config.AdvancePaymentRequestRepo.SumCompletedByEmployeeMonth(ctx, employeeID, servedMonth)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get completed amount")
	}
	pending, err := s.config.AdvancePaymentRequestRepo.SumPendingByEmployeeMonth(ctx, employeeID, servedMonth)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pending amount")
	}

	info.Salary = salary
	info.PendingEarnings = pendingEarnings
	info.MaxAdvanceAmount = maxAdv
	info.CompletedAmount = completed
	info.PendingAmount = pending
	if rem := int64(maxAdv) - int64(completed) - int64(pending); rem > 0 {
		info.RemainingAmount = uint64(rem)
	}

	if !windowOpen {
		info.CanRequest = false
		info.CanRequestTitle = "Chưa đến kỳ xin ứng"
		info.CanRequestReason = fmt.Sprintf("Bạn có thể xin ứng lương từ ngày %d đến cuối tháng.", SelfCheckInAdvanceWindowOpenDay)
		if eligibility.advanceRequestDisabled {
			applyPaused()
		}
		return info, nil
	}

	info.CanRequest = info.RemainingAmount >= minCheckInAdvanceRequest
	if !info.CanRequest {
		info.CanRequestTitle = "Chưa đủ hạn mức"
		info.CanRequestReason = "Bạn chưa có tiền công được ứng còn lại. Vui lòng chấm công thêm ca làm việc."
	}
	if eligibility.advanceRequestDisabled {
		applyPaused()
	}
	return info, nil
}

// GetCheckInAdvanceInfoByUserID resolves the employee from the JWT user id, then
// delegates to GetCheckInAdvanceInfo.
func (s *Service) GetCheckInAdvanceInfoByUserID(ctx context.Context, userID uint64) (*CheckInAdvanceInfo, error) {
	employee, err := s.config.EmployeeRepo.GetByUserID(ctx, uint(userID))
	if err != nil {
		if domain.IsNotFoundError(err) {
			return &CheckInAdvanceInfo{
				ForMonth:      clock.Now().Format("2006-01"),
				WindowOpenDay: SelfCheckInAdvanceWindowOpenDay,
				Disclaimer:    CheckInAdvanceDisclaimerVN,
			}, nil
		}
		return nil, errors.Wrap(err, "failed to get employee by user id")
	}
	return s.GetCheckInAdvanceInfo(ctx, uint64(employee.ID))
}

// CreateCheckInAdvanceRequest creates an advance request under the self-check-in flow.
// The window is the served period from checkInRequestPeriod (previous month on
// days 1-8, current month from day 10); the budget is checked atomically by
// CreateWithBudgetCheck against the stored configured max_adv_amount; fee via
// the flexible-pay schedule (unchanged).
func (s *Service) CreateCheckInAdvanceRequest(ctx context.Context, employeeID uint64, requestAmount uint64, forMonth string) (*domain.AdvancePaymentRequest, error) {
	if requestAmount < minCheckInAdvanceRequest {
		return nil, domain.NewValidationError(constants.MsgMinAdvanceAmountVN)
	}

	eligibility, err := s.getAdvanceEligibility(ctx, employeeID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check payment schedule")
	}
	if eligibility.advanceRequestDisabled {
		return nil, domain.NewValidationError(constants.MsgAdvanceRequestsDisabledVN)
	}
	if !eligibility.hasCheckInEnabled {
		return nil, domain.NewValidationError("Tài khoản chưa được cấp quyền sử dụng dịch vụ tự chấm công")
	}

	now := clock.Now()
	if forMonth == "" {
		forMonth, _ = checkInRequestPeriod(now)
	}
	if !isCheckInRequestWindowOpen(now, forMonth) {
		return nil, domain.NewValidationError(fmt.Sprintf(
			"Bạn có thể xin ứng lương cho kỳ %s: từ ngày 1 đến ngày %d của tháng sau, hoặc từ ngày %d đến cuối tháng của kỳ hiện tại.",
			FormatMonthDisplay(now.AddDate(0, -1, 0).Format("2006-01")),
			clock.RequestCutoffDay,
			SelfCheckInAdvanceWindowOpenDay,
		))
	}

	advPayments, err := s.config.AdvancePaymentRepo.GetByEmployeeAndMonth(ctx, employeeID, forMonth)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get advance payments")
	}
	if len(advPayments) == 0 {
		return nil, domain.NewValidationError("Chưa có tiền công trong kỳ này. Vui lòng chấm công trước khi xin ứng.")
	}

	// A period carrying both the workbook amount and check-in earnings would let
	// the employee withdraw against the sum of two independent sources. Refuse
	// rather than pick a winner — only the admin can resolve which one stands.
	if periodConflicts(advPayments) != nil {
		return nil, domain.NewValidationError(fmt.Sprintf(
			constants.MsgAdvancePeriodMixedSourcesVN, FormatMonthDisplay(forMonth),
		))
	}
	// The budget check (CreateWithBudgetCheck) SUMs max_adv_amount across ALL of the
	// employee's projects for the month, so the request's adv_pay_id just needs a valid
	// row. Pick the row with the highest advanceable so multi-project employees
	// deterministically attach to their primary earning project instead of arbitrary [0].
	advPay := advPayments[0]
	for _, ap := range advPayments[1:] {
		if ap.MaxAdvAmount > advPay.MaxAdvAmount {
			advPay = ap
		}
	}

	fee, netAmount := s.calculator.CalculateFee(ctx, requestAmount, now)

	if err := s.validateTransferLimits(ctx, netAmount); err != nil {
		return nil, err
	}

	req := &domain.AdvancePaymentRequest{
		AdvPayID:      advPay.ID,
		ProjectID:     advPay.ProjectID,
		EmployeeID:    uint(employeeID),
		RequestAmount: requestAmount,
		Fee:           fee,
		NetAmount:     netAmount,
		Status:        domain.AdvancePaymentStatusPending,
	}
	if err := s.config.AdvancePaymentRequestRepo.CreateWithBudgetCheck(ctx, req, employeeID, forMonth); err != nil {
		return nil, err
	}

	s.logger.Info("check-in advance request created",
		"request_id", req.ID,
		"employee_id", employeeID,
		"amount", requestAmount,
		"fee", fee,
		"net_amount", netAmount,
		"for_month", forMonth,
	)
	return req, nil
}

// CreateCheckInAdvanceRequestByUserID resolves the employee from the JWT user id, then
// delegates to CreateCheckInAdvanceRequest.
func (s *Service) CreateCheckInAdvanceRequestByUserID(ctx context.Context, userID uint64, requestAmount uint64, forMonth string) (*domain.AdvancePaymentRequest, error) {
	employee, err := s.config.EmployeeRepo.GetByUserID(ctx, uint(userID))
	if err != nil {
		return nil, domain.NewValidationError(constants.MsgEmployeeInfoNotFoundVN)
	}
	return s.CreateCheckInAdvanceRequest(ctx, uint64(employee.ID), requestAmount, forMonth)
}
