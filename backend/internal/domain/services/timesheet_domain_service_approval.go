package services

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
)

// ValidateTimesheetApproval validates business rules for timesheet approval
func (s *TimesheetDomainService) ValidateTimesheetApproval(ctx context.Context, timesheetID uint, approverID uint) error {
	timesheet, err := s.timesheetRepo.GetByID(ctx, timesheetID)
	if err != nil {
		return err
	}

	// Check timesheet status
	if timesheet.Status != domain.TimesheetStatusPendingApproval {
		return domain.NewValidationError(constants.MsgTimesheetNotPendingApprovalVN)
	}

	// Check if timesheet is not too old (business rule: cannot approve timesheets older than 30 days)
	if time.Since(timesheet.Date) > 30*24*time.Hour {
		return domain.NewValidationError(constants.MsgTimesheetTooOldForApprovalVN)
	}

	// Check if approver is not the same as creator (business rule)
	if timesheet.CreatedBy == approverID {
		return domain.NewValidationError(constants.MsgCannotApproveSelfCreatedTimesheetVN)
	}

	return nil
}

// ValidateTimesheetRejection validates business rules for timesheet rejection
func (s *TimesheetDomainService) ValidateTimesheetRejection(ctx context.Context, timesheetID uint, rejectorID uint, reason string) error {
	timesheet, err := s.timesheetRepo.GetByID(ctx, timesheetID)
	if err != nil {
		return err
	}

	// Check timesheet status
	if timesheet.Status != domain.TimesheetStatusPendingApproval {
		return domain.NewValidationError(constants.MsgTimesheetNotPendingApprovalVN)
	}

	// Check if rejection reason is provided
	if reason == "" {
		return domain.NewValidationError(constants.MsgRejectionReasonRequiredVN)
	}

	// Check if rejector is not the same as creator
	if timesheet.CreatedBy == rejectorID {
		return domain.NewValidationError(constants.MsgCannotRejectSelfCreatedTimesheetVN)
	}

	return nil
}

// ApproveTimesheet handles timesheet approval with business rules
func (s *TimesheetDomainService) ApproveTimesheet(ctx context.Context, timesheetID uint, approverID uint) error {
	// Validate approval
	if err := s.ValidateTimesheetApproval(ctx, timesheetID, approverID); err != nil {
		return err
	}

	// Perform approval - delegate to repository for data persistence
	return s.timesheetRepo.Approve(ctx, timesheetID, approverID)
}

// RejectTimesheet handles timesheet rejection with business rules
func (s *TimesheetDomainService) RejectTimesheet(ctx context.Context, timesheetID uint, rejectorID uint, reason string) error {
	// Validate rejection
	if err := s.ValidateTimesheetRejection(ctx, timesheetID, rejectorID, reason); err != nil {
		return err
	}

	// Perform rejection - delegate to repository for data persistence
	return s.timesheetRepo.Reject(ctx, timesheetID, reason)
}

// ResetTimesheet handles timesheet reset back to pending approval with business rules
func (s *TimesheetDomainService) ResetTimesheet(ctx context.Context, timesheetID uint) error {
	// Get timesheet to validate
	timesheet, err := s.timesheetRepo.GetByID(ctx, timesheetID)
	if err != nil {
		return err
	}

	// Only approved or rejected timesheets can be reset
	if timesheet.Status != domain.TimesheetStatusApproved && timesheet.Status != domain.TimesheetStatusRejected {
		return domain.NewValidationError(fmt.Sprintf("bảng chấm công không thể được đặt lại: trạng thái hiện tại là '%s', chỉ các bảng chấm công có trạng thái 'đã phê duyệt' hoặc 'đã từ chối' mới có thể được đặt lại", timesheet.Status))
	}

	// Cannot reset paid timesheets
	if timesheet.PaymentStatus == domain.PaymentStatusPaid {
		return domain.NewValidationError("không thể đặt lại bảng chấm công đã được thanh toán")
	}

	// Perform reset - delegate to repository for data persistence
	return s.timesheetRepo.Reset(ctx, timesheetID)
}

// ValidatePaymentStatusUpdate validates business rules for payment status updates
func (s *TimesheetDomainService) ValidatePaymentStatusUpdate(ctx context.Context, updates []domain.PaymentStatusUpdate) error {
	for _, update := range updates {
		// Get timesheet to validate
		timesheet, err := s.timesheetRepo.GetByID(ctx, update.TimesheetID)
		if err != nil {
			return err
		}

		// Only approved timesheets can have payment status updated
		if timesheet.Status != domain.TimesheetStatusApproved {
			return domain.NewValidationError(constants.MsgCanOnlyPayApprovedTimesheetsVN)
		}

		// Validate payment status transitions
		if err := s.validatePaymentStatusTransition(timesheet.PaymentStatus, update.PaymentStatus); err != nil {
			return err
		}

		// If marking as paid, require payment details
		if update.PaymentStatus == domain.PaymentStatusPaid {
			if update.PaidAmount == nil || *update.PaidAmount <= 0 {
				return domain.NewValidationError(constants.MsgPaidAmountRequiredVN)
			}
			if update.PaymentReference == nil || *update.PaymentReference == "" {
				return domain.NewValidationError(constants.MsgPaymentReferenceRequiredVN)
			}
		}
	}

	return nil
}

// UpdatePaymentStatus updates payment status with business rule validation
func (s *TimesheetDomainService) UpdatePaymentStatus(ctx context.Context, updates []domain.PaymentStatusUpdate) error {
	// Validate all updates first
	if err := s.ValidatePaymentStatusUpdate(ctx, updates); err != nil {
		return err
	}

	// Perform updates - delegate to repository for data persistence
	return s.timesheetRepo.BulkUpdatePaymentStatus(ctx, updates)
}

// validatePaymentStatusTransition validates that a payment status transition is allowed
func (s *TimesheetDomainService) validatePaymentStatusTransition(currentStatus, newStatus domain.PaymentStatus) error {
	// Define valid payment status transitions
	validTransitions := map[domain.PaymentStatus][]domain.PaymentStatus{
		domain.PaymentStatusPending:   {domain.PaymentStatusPaid, domain.PaymentStatusFailed, domain.PaymentStatusCancelled},
		domain.PaymentStatusPaid:      {}, // No transitions allowed from paid
		domain.PaymentStatusFailed:    {domain.PaymentStatusPending, domain.PaymentStatusCancelled},
		domain.PaymentStatusCancelled: {domain.PaymentStatusPending},
	}

	allowedTransitions, exists := validTransitions[currentStatus]
	if !exists {
		return domain.NewValidationError(constants.MsgInvalidPaymentStatusVN)
	}

	for _, allowed := range allowedTransitions {
		if newStatus == allowed {
			return nil
		}
	}

	return domain.NewValidationError(constants.MsgInvalidPaymentStatusTransitionVN)
}
