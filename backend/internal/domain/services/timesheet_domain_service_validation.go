package services

import (
	"context"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// ValidateTimesheetModification validates if a timesheet can be modified
func (s *TimesheetDomainService) ValidateTimesheetModification(ctx context.Context, timesheetID uint, modifierID uint) error {
	timesheet, err := s.timesheetRepo.GetByID(ctx, timesheetID)
	if err != nil {
		return err
	}

	// Cannot modify approved or rejected timesheets
	if timesheet.Status == domain.TimesheetStatusApproved {
		return domain.NewValidationError(constants.MsgCannotModifyApprovedTimesheetVN)
	}

	if timesheet.Status == domain.TimesheetStatusRejected {
		return domain.NewValidationError(constants.MsgCannotModifyRejectedTimesheetVN)
	}

	// Check if timesheet is not too old for modification (business rule: 7 days)
	if time.Since(timesheet.Date) > 7*24*time.Hour && timesheet.Status != domain.TimesheetStatusPendingApproval {
		return domain.NewValidationError(constants.MsgTimesheetTooOldForModificationVN)
	}

	return nil
}

// ValidateTimesheetDeletion validates if a timesheet can be deleted
func (s *TimesheetDomainService) ValidateTimesheetDeletion(ctx context.Context, timesheetID uint, deleterID uint) error {
	timesheet, err := s.timesheetRepo.GetByID(ctx, timesheetID)
	if err != nil {
		return err
	}

	// Cannot delete approved timesheets
	if timesheet.Status == domain.TimesheetStatusApproved {
		return domain.NewValidationError(constants.MsgCannotDeleteApprovedTimesheetVN)
	}

	// Cannot delete paid timesheets
	if timesheet.PaymentStatus == domain.PaymentStatusPaid {
		return domain.NewValidationError(constants.MsgCannotDeletePaidTimesheetVN)
	}

	return nil
}

// SetInitialTimesheetStatus sets the initial status of a timesheet based on user role and business rules
func (s *TimesheetDomainService) SetInitialTimesheetStatus(ctx context.Context, timesheet *domain.Timesheet, createdBy uint, userRole string) error {
	return s.SetInitialTimesheetStatusForBulk(ctx, timesheet, createdBy, userRole, false)
}

// SetInitialTimesheetStatusForBulk sets the initial status for a bulk-created
// timesheet. Imports that must be reviewed can require approval even when an
// administrator submitted the file.
func (s *TimesheetDomainService) SetInitialTimesheetStatusForBulk(
	ctx context.Context,
	timesheet *domain.Timesheet,
	createdBy uint,
	userRole string,
	requireApproval bool,
) error {
	if requireApproval {
		timesheet.Status = domain.TimesheetStatusPendingApproval
		timesheet.ApprovedBy = nil
		timesheet.ApprovedAt = nil
		return nil
	}

	// Business rule: Admin submissions are auto-approved
	if userRole == "admin" {
		// Set status directly to approved for admin users
		now := clock.Now()
		timesheet.Status = domain.TimesheetStatusApproved
		timesheet.ApprovedBy = &createdBy
		timesheet.ApprovedAt = &now
		return nil
	}

	// Business rule: Partner submissions require approval
	timesheet.Status = domain.TimesheetStatusPendingApproval
	return nil
}
