package domain

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// TimesheetEditRequestStatus represents the status of an edit request
type TimesheetEditRequestStatus string

const (
	EditRequestStatusPending  TimesheetEditRequestStatus = "pending"
	EditRequestStatusApproved TimesheetEditRequestStatus = "approved"
	EditRequestStatusRejected TimesheetEditRequestStatus = "rejected"
)

// TimesheetEditRequest represents a request to edit an approved timesheet
type TimesheetEditRequest struct {
	ID          uint                       `json:"id" gorm:"primarykey;type:bigint unsigned"`
	TimesheetID uint                       `json:"timesheet_id" gorm:"not null;type:bigint unsigned;comment:'Foreign key to timesheets table'"`
	Status      TimesheetEditRequestStatus `json:"status" gorm:"type:enum('pending','approved','rejected');not null;default:'pending';comment:'Request status'"`
	RequestedBy uint                       `json:"requested_by" gorm:"not null;type:bigint unsigned;comment:'User ID who requested the edit'"`
	ApprovedBy  *uint                      `json:"approved_by" gorm:"type:bigint unsigned;comment:'Admin user ID who approved the request'"`
	RejectedBy  *uint                      `json:"rejected_by" gorm:"type:bigint unsigned;comment:'Admin user ID who rejected the request'"`
	DeletedAt   gorm.DeletedAt             `json:"-" gorm:"index"`
	CreatedAt   time.Time                  `json:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`

	// Relationships
	Timesheet     *Timesheet `json:"timesheet,omitempty" gorm:"foreignKey:TimesheetID;references:ID"`
	RequestedUser *User      `json:"requested_user,omitempty" gorm:"foreignKey:RequestedBy;references:ID"`
	ApprovedUser  *User      `json:"approved_user,omitempty" gorm:"foreignKey:ApprovedBy;references:ID"`
	RejectedUser  *User      `json:"rejected_user,omitempty" gorm:"foreignKey:RejectedBy;references:ID"`
}

// TimesheetEditRequestRepository defines the interface for timesheet edit request persistence operations
type TimesheetEditRequestRepository interface {
	Create(ctx context.Context, request *TimesheetEditRequest) error
	GetByID(ctx context.Context, id uint) (*TimesheetEditRequest, error)
	Update(ctx context.Context, request *TimesheetEditRequest) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filters TimesheetEditRequestFilters) ([]*TimesheetEditRequest, error)
	Count(ctx context.Context, filters TimesheetEditRequestFilters) (int64, error)
	GetPendingForTimesheet(ctx context.Context, timesheetID uint) (*TimesheetEditRequest, error)
	Approve(ctx context.Context, id uint, approvedBy uint) error
	Reject(ctx context.Context, id uint, rejectedBy uint) error
}

// TimesheetEditRequestFilters represents filtering options for edit request queries
type TimesheetEditRequestFilters struct {
	Status      []TimesheetEditRequestStatus
	RequestedBy *uint      // Filter by requester
	TimesheetID *uint      // Filter by timesheet
	FromDate    *time.Time // Created date range start
	ToDate      *time.Time // Created date range end
	Limit       int
	Offset      int
	SortBy      string
	SortOrder   string
}

// IsPending returns true if the request is pending
func (r *TimesheetEditRequest) IsPending() bool {
	return r.Status == EditRequestStatusPending
}

// IsApproved returns true if the request is approved
func (r *TimesheetEditRequest) IsApproved() bool {
	return r.Status == EditRequestStatusApproved
}

// IsRejected returns true if the request is rejected
func (r *TimesheetEditRequest) IsRejected() bool {
	return r.Status == EditRequestStatusRejected
}

// CanBeApproved returns true if the request can be approved
func (r *TimesheetEditRequest) CanBeApproved() bool {
	return r.IsPending()
}

// CanBeRejected returns true if the request can be rejected
func (r *TimesheetEditRequest) CanBeRejected() bool {
	return r.IsPending()
}

// CanBeCancelled returns true if the request can be cancelled by the requester
func (r *TimesheetEditRequest) CanBeCancelled() bool {
	return r.IsPending()
}

// Approve marks the request as approved
func (r *TimesheetEditRequest) Approve(approvedBy uint) error {
	if !r.CanBeApproved() {
		return NewValidationError(fmt.Sprintf("edit request cannot be approved: current status is '%s', only 'pending' requests can be approved", r.Status))
	}

	r.Status = EditRequestStatusApproved
	r.ApprovedBy = &approvedBy
	r.RejectedBy = nil

	return nil
}

// Reject marks the request as rejected
func (r *TimesheetEditRequest) Reject(rejectedBy uint) error {
	if !r.CanBeRejected() {
		return NewValidationError(fmt.Sprintf("edit request cannot be rejected: current status is '%s', only 'pending' requests can be rejected", r.Status))
	}

	r.Status = EditRequestStatusRejected
	r.RejectedBy = &rejectedBy
	r.ApprovedBy = nil

	return nil
}

// ValidateTimesheetID validates the timesheet ID
func (r *TimesheetEditRequest) ValidateTimesheetID() error {
	if r.TimesheetID == 0 {
		return NewValidationError("ID bảng chấm công là bắt buộc")
	}
	return nil
}

// ValidateRequestedBy validates the requested by user ID
func (r *TimesheetEditRequest) ValidateRequestedBy() error {
	if r.RequestedBy == 0 {
		return NewValidationError("ID người yêu cầu là bắt buộc")
	}
	return nil
}

// IsValid validates the entire edit request entity
func (r *TimesheetEditRequest) IsValid() error {
	if err := r.ValidateTimesheetID(); err != nil {
		return err
	}
	if err := r.ValidateRequestedBy(); err != nil {
		return err
	}
	return nil
}
