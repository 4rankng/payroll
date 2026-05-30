package services

import (
	"context"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
)

// ConflictAction defines the action to take when resolving assignment conflicts
type ConflictAction string

const (
	ConflictActionReplace   ConflictAction = "REPLACE"
	ConflictActionTerminate ConflictAction = "TERMINATE"
	ConflictActionReject    ConflictAction = "REJECT"
)

// ConflictResolution contains the resolution decision for an assignment conflict
type ConflictResolution struct {
	Action      ConflictAction `json:"action"`
	TargetID    uint           `json:"target_id"`         // Assignment to replace/terminate
	TerminateAt *time.Time     `json:"terminate_at"`      // Termination date if TERMINATE
	Reason      string         `json:"reason"`            // Error message
	Message     string         `json:"message,omitempty"` // Optional human-readable message
}

// AssignmentConflictResolver handles conflict resolution for project employee assignments
type AssignmentConflictResolver interface {
	// ResolveConflict determines how to handle a conflict between a new assignment and existing assignment
	ResolveConflict(ctx context.Context,
		newAssignment *domain.ProjectEmployee,
		existingAssignment *domain.ProjectEmployee,
		timesheetCount int64,
		latestTimesheetDate *time.Time) (*ConflictResolution, error)
}

// DefaultAssignmentConflictResolver implements the business rules for assignment conflicts
type DefaultAssignmentConflictResolver struct{}

// NewAssignmentConflictResolver creates a new default assignment conflict resolver
func NewAssignmentConflictResolver() AssignmentConflictResolver {
	return &DefaultAssignmentConflictResolver{}
}

// ResolveConflict implements the business logic for resolving assignment conflicts
func (r *DefaultAssignmentConflictResolver) ResolveConflict(
	ctx context.Context,
	newAssignment *domain.ProjectEmployee,
	existingAssignment *domain.ProjectEmployee,
	timesheetCount int64,
	latestTimesheetDate *time.Time) (*ConflictResolution, error) {

	// Special Rule: Same project assignment - treat as an update/modification
	if newAssignment.ProjectID == existingAssignment.ProjectID {

		// For same project, we're more lenient with conflicts
		if timesheetCount == 0 {

			return &ConflictResolution{
				Action:   ConflictActionReplace,
				TargetID: existingAssignment.ID,
				Reason:   "ASSIGNMENT_UPDATED",
				Message:  "Cập nhật phân công cho cùng dự án (không có bảng chấm công)",
			}, nil
		}

		// If has timesheets but new start date is reasonable relative to existing assignment
		if latestTimesheetDate != nil && newAssignment.StartDate.After(*latestTimesheetDate) {

			// Terminate the existing assignment at the last timesheet date
			terminateDate := *latestTimesheetDate

			return &ConflictResolution{
				Action:      ConflictActionTerminate,
				TargetID:    existingAssignment.ID,
				TerminateAt: &terminateDate,
				Reason:      "ASSIGNMENT_EXTENDED",
				Message:     "Gia hạn phân công với ngày bắt đầu mới sau ngày chấm công cuối",
			}, nil
		}

		// For same project, even if there are overlapping timesheets, we can still allow replacement
		// as it's considered an update to the assignment details (position, dates, etc.)
		return &ConflictResolution{
			Action:   ConflictActionReplace,
			TargetID: existingAssignment.ID,
			Reason:   "ASSIGNMENT_UPDATED",
			Message:  "Cập nhật phân công cho cùng dự án",
		}, nil
	}

	// Rule 1: No timesheets exist - can replace the existing assignment
	if timesheetCount == 0 {

		return &ConflictResolution{
			Action:   ConflictActionReplace,
			TargetID: existingAssignment.ID,
			Reason:   "ASSIGNMENT_REPLACED",
			Message:  "Phân công hiện tại không có bảng chấm công, thay thế bằng phân công mới",
		}, nil
	}

	// Rule 2: Has timesheets but new start date is after the last timesheet
	// In this case, we can terminate the existing assignment and create new one
	if latestTimesheetDate != nil && newAssignment.StartDate.After(*latestTimesheetDate) {

		// Terminate the existing assignment one day before the new start date
		terminateDate := newAssignment.StartDate.AddDate(0, 0, -1)

		return &ConflictResolution{
			Action:      ConflictActionTerminate,
			TargetID:    existingAssignment.ID,
			TerminateAt: &terminateDate,
			Reason:      "ASSIGNMENT_TERMINATED",
			Message:     "Phân công hiện tại được kết thúc tự động vì ngày bắt đầu mới sau ngày chấm công cuối cùng",
		}, nil
	}

	// Rule 3: Cannot resolve conflict - existing assignment has timesheets that overlap

	return &ConflictResolution{
		Action:  ConflictActionReject,
		Reason:  "ASSIGNMENT_OVERLAP",
		Message: constants.MsgAssignmentOverlapVN,
	}, nil
}
