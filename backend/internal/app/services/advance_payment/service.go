package advance_payment

import (
	"api-server/internal/constants"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/utils"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"api-server/internal/domain"
)

type ProgressCallback func(totalRows, processedRows int)

// Service provides advance payment functionality
type Service struct {
	config     *Config
	calculator *Calculator
	logger     *slog.Logger
}

// NewService creates a new advance payment service. The calculator pulls its
// fee from config.FeeResolver, so the active schedule applies per-request
// rather than at startup.
func NewService(config *Config, logger *slog.Logger) *Service {
	return &Service{
		config:     config,
		calculator: NewCalculator(config.FeeResolver),
		logger:     logger,
	}
}

// GetConfig returns the service configuration
func (s *Service) GetConfig() *Config {
	return s.config
}

// GetEmployeeAdvanceInfoByUserID returns advance payment information for the authenticated user
func (s *Service) GetEmployeeAdvanceInfoByUserID(ctx context.Context, userID uint64) (*domain.EmployeeAdvanceInfo, error) {
	// Look up employee by user ID
	employee, err := s.config.EmployeeRepo.GetByUserID(ctx, uint(userID))
	if err != nil {
		if domain.IsNotFoundError(err) {
			return &domain.EmployeeAdvanceInfo{
				CurrentMonth:        GetCurrentMonth(),
				HasFlexibleSchedule: false,
			}, nil
		}
		return nil, errors.Wrap(err, "failed to get employee by user id")
	}

	return s.GetEmployeeAdvanceInfo(ctx, uint64(employee.ID))
}

// GetEmployeeAdvanceInfo returns advance payment information for an employee
func (s *Service) GetEmployeeAdvanceInfo(ctx context.Context, employeeID uint64) (*domain.EmployeeAdvanceInfo, error) {
	eligibility, err := s.getAdvanceEligibility(ctx, employeeID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check payment schedule")
	}

	info := &domain.EmployeeAdvanceInfo{
		CurrentMonth:        GetCurrentMonth(),
		HasFlexibleSchedule: eligibility.hasFlexible,
		Quotas:              make([]domain.AdvancePaymentQuota, 0),
	}

	if !eligibility.hasFlexible {
		info.CanRequest = false
		return info, nil
	}

	now := clock.Now()
	isBeforeCutoff := IsBeforeCutoff(now)
	isInLockedGap := IsInLockedGap(now)

	// Determine valid months for flexible employees
	// They earn quota by calendar month.
	currentCalMonth := now.Format("2006-01")
	prevCalMonth := now.AddDate(0, -1, 0).Format("2006-01")

	validMonths := []string{}

	if eligibility.hasCheckInEnabled {
		// Check-in/out employees earn quota continuously from completed shifts,
		// so the import/cutoff windows do not apply. Show every month with
		// earned quota and let the remaining budget be the only request gate.
		months, err := s.config.AdvancePaymentRepo.GetMonthsByEmployee(ctx, employeeID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get earned quota months")
		}
		validMonths = append(validMonths, months...)
	} else {
		if isBeforeCutoff {
			// Days 1-10: Can withdraw from previous month AND current month
			validMonths = append(validMonths, prevCalMonth)
			validMonths = append(validMonths, currentCalMonth)
		} else if isInLockedGap {
			// Days 11-19: show the current month quota. Requesting stays locked
			// only until admin uploads bang cham cong for that month.
			validMonths = append(validMonths, currentCalMonth)
		} else {
			// Days 20-31: Can withdraw from current month
			validMonths = append(validMonths, currentCalMonth)
		}
	}

	var totalMax, totalComp, totalPend uint64

	for _, m := range validMonths {
		maxAdv, err := s.config.AdvancePaymentRepo.SumMaxAdvByEmployeeMonth(ctx, employeeID, m)
		if err != nil {
			s.logger.Warn("failed to get max advance", "error", err, "employee_id", employeeID, "month", m)
		}
		completed, err := s.config.AdvancePaymentRequestRepo.SumCompletedByEmployeeMonth(ctx, employeeID, m)
		if err != nil {
			s.logger.Warn("failed to get completed amount", "error", err, "employee_id", employeeID, "month", m)
		}
		pending, err := s.config.AdvancePaymentRequestRepo.SumPendingByEmployeeMonth(ctx, employeeID, m)
		if err != nil {
			s.logger.Warn("failed to get pending amount", "error", err, "employee_id", employeeID, "month", m)
		}

		rem := int64(maxAdv) - int64(completed) - int64(pending)
		remUint := uint64(0)
		if rem > 0 {
			remUint = uint64(rem)
		}

		if maxAdv > 0 {
			info.Quotas = append(info.Quotas, domain.AdvancePaymentQuota{
				ForMonth:         m,
				MaxAdvanceAmount: maxAdv,
				CompletedAmount:  completed,
				PendingAmount:    pending,
				RemainingAmount:  remUint,
			})

			totalMax += maxAdv
			totalComp += completed
			totalPend += pending
		}
	}

	info.TotalMaxAdvance = totalMax
	info.CompletedAmount = totalComp
	info.PendingAmount = totalPend
	remTotal := int64(totalMax) - int64(totalComp) - int64(totalPend)
	if remTotal > 0 {
		info.RemainingAmount = uint64(remTotal)
	}

	hasCurrentMonthQuota := false
	for _, quota := range info.Quotas {
		if quota.ForMonth == currentCalMonth && quota.MaxAdvanceAmount > 0 {
			hasCurrentMonthQuota = true
			break
		}
	}

	if !eligibility.hasCheckInEnabled && isRequestWindowLocked(now, hasCurrentMonthQuota) {
		info.CanRequest = false
		info.CanRequestTitle = fmt.Sprintf(constants.MsgAdvanceCutoffTitleVN, FormatMonthDisplay(currentCalMonth))
		info.CanRequestReason = fmt.Sprintf(constants.MsgAdvanceCutoffWaitingUploadVN, FormatMonthDisplay(currentCalMonth))
	} else {
		info.CanRequest = info.RemainingAmount >= 10000
		// When there is no quota at all (salary data not yet uploaded), provide
		// an explanatory message so the employee understands why CanRequest=false.
		if !info.CanRequest && info.TotalMaxAdvance == 0 && len(info.Quotas) == 0 {
			info.CanRequestTitle = fmt.Sprintf(constants.MsgAdvanceCutoffTitleVN, FormatMonthDisplay(prevCalMonth))
			info.CanRequestReason = fmt.Sprintf(constants.MsgAdvanceCutoffWaitingUploadVN, FormatMonthDisplay(currentCalMonth))
		}
	}

	return info, nil
}

// CreateRequestByUserID creates a new advance payment request using user ID
func (s *Service) CreateRequestByUserID(ctx context.Context, userID uint64, requestAmount uint64, forMonth string) (*domain.AdvancePaymentRequest, error) {
	// Look up employee by user ID
	employee, err := s.config.EmployeeRepo.GetByUserID(ctx, uint(userID))
	if err != nil {
		return nil, domain.NewValidationError(constants.MsgEmployeeInfoNotFoundVN)
	}

	return s.CreateRequest(ctx, uint64(employee.ID), requestAmount, forMonth)
}

// validateTransferLimits checks that the disbursement net amount is within the
// provider's min/max transfer limits. Shared by the admin and self-check-in
// advance request flows. No-op when no limits function is configured.
func (s *Service) validateTransferLimits(ctx context.Context, netAmount uint64) error {
	if s.config.GetTransferLimits == nil {
		return nil
	}
	limits := s.config.GetTransferLimits(ctx)
	if limits.MinAmount > 0 && int64(netAmount) < limits.MinAmount {
		return domain.NewValidationError(fmt.Sprintf(
			"Số tiền thực nhận (%s) dưới mức tối thiểu chuyển tiền (%s). Vui lòng tăng số tiền yêu cầu.",
			utils.FormatVND(int64(netAmount)), utils.FormatVND(limits.MinAmount),
		))
	}
	if limits.MaxAmount > 0 && int64(netAmount) > limits.MaxAmount {
		return domain.NewValidationError(fmt.Sprintf(
			"Số tiền thực nhận (%s) vượt mức tối đa chuyển tiền (%s). Vui lòng giảm số tiền yêu cầu.",
			utils.FormatVND(int64(netAmount)), utils.FormatVND(limits.MaxAmount),
		))
	}
	return nil
}

// CreateRequest creates a new advance payment request
func (s *Service) CreateRequest(ctx context.Context, employeeID uint64, requestAmount uint64, forMonth string) (*domain.AdvancePaymentRequest, error) {
	// Validate minimum amount
	if requestAmount < 10000 {
		return nil, domain.NewValidationError(constants.MsgMinAdvanceAmountVN)
	}

	eligibility, err := s.getAdvanceEligibility(ctx, employeeID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check payment schedule")
	}
	if !eligibility.hasFlexible {
		return nil, domain.NewValidationError(constants.MsgEmployeeNoFlexiblePayScheduleVN)
	}

	// Enforce the three-phase window rule for flexible employees
	now := clock.Now()

	currentCalMonth := now.Format("2006-01")

	if !eligibility.hasCheckInEnabled && !isRequestMonthAllowed(now, forMonth) {
		return nil, domain.NewValidationError(constants.MsgSalaryInfoNotFoundForMonthVN)
	}

	// Get advance payment record for this employee
	advPayments, err := s.config.AdvancePaymentRepo.GetByEmployeeAndMonth(ctx, employeeID, forMonth)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get advance payments")
	}
	if len(advPayments) == 0 {
		if !eligibility.hasCheckInEnabled && IsInLockedGap(now) && forMonth == currentCalMonth {
			return nil, domain.NewValidationError(constants.MsgAdvanceRequestCutoffVN)
		}
		return nil, domain.NewValidationError(constants.MsgSalaryInfoNotFoundForMonthVN)
	}

	// Use first advance payment record (assuming one project per employee for now)
	advPay := advPayments[0]

	// Calculate fee and net amount using the schedule active right now.
	fee, netAmount := s.calculator.CalculateFee(ctx, requestAmount, clock.Now())

	// Validate that net amount respects disbursement provider limits.
	if err := s.validateTransferLimits(ctx, netAmount); err != nil {
		return nil, err
	}

	// Create request atomically with budget check (prevents TOCTOU race)
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

	s.logger.Info("advance payment request created",
		"request_id", req.ID,
		"employee_id", employeeID,
		"amount", requestAmount,
		"fee", fee,
		"net_amount", netAmount,
		"effective_month", forMonth,
	)

	return req, nil
}

func isRequestWindowLocked(now time.Time, hasCurrentMonthQuota bool) bool {
	return IsInLockedGap(now) && !hasCurrentMonthQuota
}

func isRequestMonthAllowed(now time.Time, forMonth string) bool {
	currentCalMonth := now.Format("2006-01")
	prevCalMonth := now.AddDate(0, -1, 0).Format("2006-01")

	if IsBeforeCutoff(now) {
		return forMonth == prevCalMonth || forMonth == currentCalMonth
	}

	return forMonth == currentCalMonth
}

// GetRequestHistoryByUserID returns the request history for the authenticated user
func (s *Service) GetRequestHistoryByUserID(ctx context.Context, userID uint64, limit, offset int) ([]domain.AdvancePaymentHistoryItem, int64, error) {
	// Look up employee by user ID
	employee, err := s.config.EmployeeRepo.GetByUserID(ctx, uint(userID))
	if err != nil {
		if domain.IsNotFoundError(err) {
			return []domain.AdvancePaymentHistoryItem{}, 0, nil
		}
		return nil, 0, errors.Wrap(err, "failed to get employee by user id")
	}

	return s.GetRequestHistory(ctx, uint64(employee.ID), limit, offset)
}

// GetRequestHistory returns the request history for an employee
func (s *Service) GetRequestHistory(ctx context.Context, employeeID uint64, limit, offset int) ([]domain.AdvancePaymentHistoryItem, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	requests, total, err := s.config.AdvancePaymentRequestRepo.GetByEmployee(ctx, employeeID, limit, offset)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to get request history")
	}

	// Convert to history items
	items := make([]domain.AdvancePaymentHistoryItem, len(requests))
	for i, req := range requests {
		projectName := ""
		projectCode := ""
		if req.Project.ID != 0 {
			projectName = req.Project.Name
			projectCode = req.Project.Code
		}

		items[i] = domain.AdvancePaymentHistoryItem{
			ID:            req.ID,
			RequestAmount: req.RequestAmount,
			Fee:           req.Fee,
			NetAmount:     req.NetAmount,
			Status:        req.Status,
			CreatedAt:     req.CreatedAt,
			PaidAt:        req.PaidAt,
			ProjectName:   projectName,
			ProjectCode:   projectCode,
		}
	}

	return items, total, nil
}

// CalculateFeePreview calculates the fee for a given amount without creating
// a request. The preview uses the schedule active right now — admin-staged
// future schedules do NOT affect the preview until their effective date.
func (s *Service) CalculateFeePreview(ctx context.Context, requestAmount uint64) (fee, netAmount uint64) {
	return s.calculator.CalculateFee(ctx, requestAmount, clock.Now())
}

type advanceEligibility struct {
	hasFlexible       bool
	hasCheckInEnabled bool
}

func (s *Service) getAdvanceEligibility(ctx context.Context, employeeID uint64) (advanceEligibility, error) {
	// Get project employee assignments
	assignments, err := s.config.ProjectEmployeeRepo.GetByEmployee(ctx, uint(employeeID))
	if err != nil {
		return advanceEligibility{}, err
	}

	var result advanceEligibility
	for _, assignment := range assignments {
		if assignment.PaymentSchedule == string(domain.PaymentScheduleFlexible) && assignment.LastDate == nil {
			result.hasFlexible = true
			if assignment.CheckInEnabled {
				result.hasCheckInEnabled = true
			}
		}
	}

	return result, nil
}

// GetPendingByDateRange returns pending requests within a date range
func (s *Service) GetPendingByDateRange(ctx context.Context, fromDate, toDate string) ([]*domain.AdvancePaymentRequest, error) {
	from, err := ParseMonth(fromDate)
	if err != nil {
		return nil, domain.NewValidationError(constants.MsgInvalidFromDateFormatVN)
	}
	to, err := ParseMonth(toDate)
	if err != nil {
		return nil, domain.NewValidationError(constants.MsgInvalidToDateFormatVN)
	}

	return s.config.AdvancePaymentRequestRepo.GetPendingByDateRange(ctx, from, to)
}

// GenerateTransactionCode generates a transaction code for payroll.
// Uses "VFIC" prefix for weekly/monthly, "tt" prefix for flexible payment.
// OnePay limits request_id to 20 chars; short form is prefix+14 = 16-18 chars.
// Format is alphanumeric only (no hyphen): 9pay rejects hyphens with error 318.
func GenerateTransactionCode(isFlexible bool, isShort bool) string {
	txUUID := strings.ReplaceAll(uuid.New().String(), "-", "")
	txShort := strings.ReplaceAll(txUUID, "-", "")[:14]
	prefix := "VFIC"
	if isFlexible {
		prefix = "tt"
	}
	if isShort {
		return fmt.Sprintf("%s%s", prefix, txShort)
	}
	return fmt.Sprintf("%s%s", prefix, txUUID)
}

// GenerateUniqueTransactionCode generates a transaction code that doesn't exist in the provided set
// Uses "VFIC" prefix for weekly/monthly, "tt" prefix for flexible payment
func GenerateUniqueTransactionCode(existingCodes map[string]struct{}, isFlexible bool) string {
	for i := 0; i < 100; i++ {
		code := GenerateTransactionCode(isFlexible, true)
		if _, exists := existingCodes[code]; !exists {
			return code
		}
	}
	return GenerateTransactionCode(isFlexible, false)
}

// CancelMyRequestByUserID cancels a user's own pending request using user ID
func (s *Service) CancelMyRequestByUserID(ctx context.Context, requestID, userID uint64) error {
	// Look up employee by user ID
	employee, err := s.config.EmployeeRepo.GetByUserID(ctx, uint(userID))
	if err != nil {
		return domain.NewValidationError(constants.MsgEmployeeInfoNotFoundVN)
	}

	return s.CancelMyRequest(ctx, requestID, uint64(employee.ID))
}

// CancelMyRequest cancels a user's own pending request
func (s *Service) CancelMyRequest(ctx context.Context, requestID, employeeID uint64) error {
	req, err := s.config.AdvancePaymentRequestRepo.GetByID(ctx, requestID)
	if err != nil {
		return errors.Wrap(err, "failed to get request")
	}

	if req.EmployeeID != uint(employeeID) {
		return domain.NewValidationError(constants.MsgCannotCancelRequestVN)
	}

	if !req.IsPending() {
		return domain.NewValidationError(constants.MsgCanOnlyCancelPendingRequestVN)
	}

	if err := s.config.AdvancePaymentRequestRepo.Cancel(ctx, requestID); err != nil {
		return errors.Wrap(err, "failed to cancel request")
	}

	s.logger.Info("advance payment request cancelled by user",
		"request_id", requestID,
		"employee_id", employeeID,
	)

	return nil
}

// CancelRequest cancels any pending or approved request (admin only)
func (s *Service) CancelRequest(ctx context.Context, requestID uint64) error {
	req, err := s.config.AdvancePaymentRequestRepo.GetByID(ctx, requestID)
	if err != nil {
		return errors.Wrap(err, "failed to get request")
	}

	// Admin can cancel PENDING or APPROVED requests
	if !req.IsPending() && !req.IsApproved() {
		return domain.NewValidationError(constants.MsgCanOnlyCancelPendingOrApprovedVN)
	}

	if err := s.config.AdvancePaymentRequestRepo.Cancel(ctx, requestID); err != nil {
		return errors.Wrap(err, "failed to cancel request")
	}

	s.logger.Info("advance payment request cancelled by admin",
		"request_id", requestID,
		"previous_status", req.Status,
	)

	return nil
}

// GetAvailableAmount returns the available amount for an employee
func (s *Service) GetAvailableAmount(ctx context.Context, employeeID uint64, forMonth string) (uint64, error) {
	totalMaxAdv, err := s.config.AdvancePaymentRepo.SumMaxAdvByEmployeeMonth(ctx, employeeID, forMonth)
	if err != nil {
		return 0, err
	}

	usedAmount, err := s.config.AdvancePaymentRequestRepo.SumPendingAndCompletedByEmployeeMonth(ctx, employeeID, forMonth)
	if err != nil {
		return 0, err
	}

	available := int64(totalMaxAdv) - int64(usedAmount)
	if available < 0 {
		return 0, nil
	}

	return uint64(available), nil
}
