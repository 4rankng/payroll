package domain

import (
	"fmt"
	"time"

	"api-server/internal/constants"
	"api-server/internal/pkg/clock"

	"gorm.io/gorm"
)

// TimesheetStatus represents timesheet entry status
type TimesheetStatus string

const (
	TimesheetStatusPendingApproval TimesheetStatus = "pending_approval"
	TimesheetStatusApproved        TimesheetStatus = "approved"
	TimesheetStatusRejected        TimesheetStatus = "rejected"
)

// PaymentStatus represents payment status for timesheets
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusPaid      PaymentStatus = "paid"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

// EmployeeDateCombo represents a combination of employee, project, and date for bulk queries
type EmployeeDateCombo struct {
	EmployeeID uint
	ProjectID  uint
	Date       time.Time
}

// Timesheet represents a timesheet entry in the domain
type Timesheet struct {
	ID                uint            `json:"id" gorm:"primarykey;type:bigint unsigned"`
	ProjectID         uint            `json:"project_id" gorm:"not null;type:bigint unsigned"`
	EmployeeID        uint            `json:"employee_id" gorm:"not null;type:bigint unsigned"`
	PayrateID         uint            `json:"payrate_id" gorm:"column:payrate_id;not null;type:bigint unsigned;comment:'Reference to payrate config used'"`
	Date              time.Time       `json:"date" gorm:"type:date;not null"`
	HoursWorked       float64         `json:"hours_worked" gorm:"type:decimal(4,2);not null;default:0;comment:'Hours worked (0.00-99.99)'"`
	PayType           string          `json:"paytype" gorm:"column:paytype;type:varchar(100);not null;comment:'Flexible paytype from payrate config'"`
	PayRate           int64           `json:"payrate" gorm:"column:payrate;type:bigint;not null;comment:'VND amount from payrates config'"`
	Amount            int64           `json:"amount" gorm:"type:bigint;not null;default:0"`
	Status            TimesheetStatus `json:"status" gorm:"column:timesheet_status;type:enum('pending_approval','approved','rejected');not null;default:'pending_approval'"`
	PaymentStatus     PaymentStatus   `json:"payment_status" gorm:"type:enum('pending','paid','failed','cancelled');not null;default:'pending'"`
	AllowedEdit       bool            `json:"allowed_edit" gorm:"column:allowed_edit;type:tinyint(1);not null;default:0;comment:'Flag indicating if timesheet is allowed to be edited after approval'"`
	RequestEditID     *uint           `json:"request_edit_id" gorm:"column:request_edit_id;type:bigint unsigned;comment:'Pending edit request id'"`
	PaymentReference  *string         `json:"payment_reference" gorm:"type:varchar(255);comment:'Bank transaction reference'"`
	PaymentDate       *time.Time      `json:"payment_date" gorm:"type:date"`
	PaidAmount        int64           `json:"paid_amount" gorm:"type:bigint;default:0;comment:'Total amount paid for this timesheet'"`
	PaidAt            *time.Time      `json:"paid_at" gorm:"type:datetime(3)"`
	RevenueReceivable int64           `json:"revenue_receivable" gorm:"type:bigint unsigned;default:0;comment:'Revenue to receive from client for advance cash service fee'"`
	RevenuePaid       bool            `json:"revenue_paid" gorm:"type:tinyint(1);default:0;comment:'Whether revenue has been received from client'"`
	TransactionID     *uint           `json:"transaction_id,omitempty" gorm:"column:transaction_id;type:bigint unsigned;index:idx_timesheets_transaction_id;comment:'Linked revenue transaction for this timesheet'"`
	DeletedAt         gorm.DeletedAt  `json:"-" gorm:"index"`
	CreatedBy         uint            `json:"created_by" gorm:"not null;type:bigint unsigned"`
	ApprovedBy        *uint           `json:"approved_by" gorm:"type:bigint unsigned;comment:'Admin who approved the timesheet'"`
	ApprovedAt        *time.Time      `json:"approved_at"`
	RejectionReason   string          `json:"rejection_reason" gorm:"type:text"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	ForcePayroll      bool            `json:"force_payroll" gorm:"column:force_payroll;type:tinyint(1);not null;default:0"`

	// Relationships
	Project      *Project  `json:"project,omitempty" gorm:"foreignKey:ProjectID;references:ID"`
	Employee     *Employee `json:"employee,omitempty" gorm:"foreignKey:EmployeeID;references:ID"`
	Payrate      *Payrate  `json:"payrate_config,omitempty" gorm:"foreignKey:PayrateID;references:ID"`
	CreatedUser  *User     `json:"created_user,omitempty" gorm:"foreignKey:CreatedBy;references:ID"`
	ApprovedUser *User     `json:"approved_user,omitempty" gorm:"foreignKey:ApprovedBy;references:ID"`
}

// ValidateProjectID validates the project ID
func (t *Timesheet) ValidateProjectID() error {
	if t.ProjectID == 0 {
		return NewValidationError("ID dự án là bắt buộc")
	}
	return nil
}

// ValidateEmployeeID validates the employee ID
func (t *Timesheet) ValidateEmployeeID() error {
	if t.EmployeeID == 0 {
		return NewValidationError("ID nhân viên là bắt buộc")
	}
	return nil
}

// ValidateDate validates the timesheet date
func (t *Timesheet) ValidateDate() error {
	if t.Date.IsZero() {
		return NewValidationError("ngày là bắt buộc và không thể trống")
	}

	now := clock.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	timesheetDate := time.Date(t.Date.Year(), t.Date.Month(), t.Date.Day(), 0, 0, 0, 0, now.Location())

	if timesheetDate.After(today) {
		return NewValidationError(fmt.Sprintf("ngày không thể ở tương lai: ngày cung cấp %s sau hôm nay %s", timesheetDate.Format("2006-01-02"), today.Format("2006-01-02")))
	}

	return nil
}

// ValidateHoursWorked validates hours worked
func (t *Timesheet) ValidateHoursWorked() error {
	if t.HoursWorked < 0 {
		return NewValidationError(fmt.Sprintf("giờ làm việc không thể âm: cung cấp %.2f giờ", t.HoursWorked))
	}

	if t.HoursWorked > 24 {
		return NewValidationError(fmt.Sprintf("giờ làm việc không thể vượt quá 24 giờ mỗi ngày: cung cấp %.2f giờ", t.HoursWorked))
	}

	return nil
}

// ValidatePayType validates the pay type
func (t *Timesheet) ValidatePayType() error {
	// Allow any non-empty pay type as rates are now flexible
	if t.PayType == "" {
		return NewValidationError("loại thanh toán không thể trống")
	}
	return nil
}

// ValidatePayrateID validates the payrate ID reference
func (t *Timesheet) ValidatePayrateID() error {
	if t.PayrateID == 0 {
		return NewValidationError("ID mức lương là bắt buộc")
	}
	return nil
}

// ValidatePayRate validates the pay rate
func (t *Timesheet) ValidatePayRate() error {
	if t.PayRate < 0 {
		return NewValidationError("mức lương không thể âm")
	}
	return nil
}

// ValidateStatus validates the timesheet status
func (t *Timesheet) ValidateStatus() error {
	validStatuses := map[TimesheetStatus]bool{
		TimesheetStatusPendingApproval: true,
		TimesheetStatusApproved:        true,
		TimesheetStatusRejected:        true,
	}

	if !validStatuses[t.Status] {
		return NewValidationError("trạng thái bảng chấm công không hợp lệ")
	}
	return nil
}

// IsValid validates the entire timesheet entity
func (t *Timesheet) IsValid() error {
	if err := t.ValidateProjectID(); err != nil {
		return err
	}
	if err := t.ValidateEmployeeID(); err != nil {
		return err
	}
	if err := t.ValidateDate(); err != nil {
		return err
	}
	if err := t.ValidateHoursWorked(); err != nil {
		return err
	}
	if err := t.ValidatePayType(); err != nil {
		return err
	}
	if err := t.ValidatePayrateID(); err != nil {
		return err
	}
	if err := t.ValidatePayRate(); err != nil {
		return err
	}
	return nil
}

// IsPendingApproval returns true if the timesheet is pending approval
func (t *Timesheet) IsPendingApproval() bool {
	return t.Status == TimesheetStatusPendingApproval
}

// IsApproved returns true if the timesheet is approved
func (t *Timesheet) IsApproved() bool {
	return t.Status == TimesheetStatusApproved
}

// IsRejected returns true if the timesheet is rejected
func (t *Timesheet) IsRejected() bool {
	return t.Status == TimesheetStatusRejected
}

// IsPaid returns true if the timesheet has been paid, failed, or cancelled
// Timesheets with these payment statuses cannot be edited
func (t *Timesheet) IsPaid() bool {
	return t.PaymentStatus == PaymentStatusPaid ||
		t.PaymentStatus == PaymentStatusFailed ||
		t.PaymentStatus == PaymentStatusCancelled
}

// CanBeEdited returns true if timesheet can be edited
func (t *Timesheet) CanBeEdited() bool {
	if t.AllowedEdit {
		return true
	}

	return !t.IsApproved()
}

// CanBeEditedByUser returns true if timesheet can be edited by the specified user role
func (t *Timesheet) CanBeEditedByUser(userRole UserRole) bool {
	// Nobody can edit paid timesheets
	if t.IsPaid() {
		return false
	}

	// Explicitly allowed edits override status-based restrictions
	if t.AllowedEdit {
		return true
	}

	// Admin can edit approved (but not paid) timesheets
	if userRole == RoleAdmin {
		return true
	}

	// Partner cannot edit approved timesheet
	if userRole == RolePartner && t.IsApproved() {
		return false
	}

	// For other cases, use default edit rules (draft, rejected)
	return t.CanBeEdited()
}

// CanBeApproved returns true if timesheet can be approved
func (t *Timesheet) CanBeApproved() bool {
	return t.IsPendingApproval()
}

// CanBeRejected returns true if timesheet can be rejected
func (t *Timesheet) CanBeRejected() bool {
	return t.IsPendingApproval()
}

// CanTransitionTo checks if status transition is allowed
func (t *Timesheet) CanTransitionTo(newStatus TimesheetStatus) bool {
	currentStatus := t.Status

	// Define allowed transitions
	allowedTransitions := map[TimesheetStatus][]TimesheetStatus{
		TimesheetStatusPendingApproval: {
			TimesheetStatusApproved,
			TimesheetStatusRejected,
		},
		TimesheetStatusApproved: {
			// Approved entries are immutable
		},
		TimesheetStatusRejected: {
			TimesheetStatusPendingApproval,
		},
	}

	allowedTargets, exists := allowedTransitions[currentStatus]
	if !exists {
		return false
	}

	for _, allowed := range allowedTargets {
		if allowed == newStatus {
			return true
		}
	}

	return false
}

// Approve marks the timesheet as approved
func (t *Timesheet) Approve(approvedBy uint) error {
	if !t.CanBeApproved() {
		return NewValidationError(fmt.Sprintf("bảng chấm công không thể được phê duyệt: trạng thái hiện tại là '%s', chỉ các bảng chấm công có trạng thái 'chờ phê duyệt' mới có thể được phê duyệt", t.Status))
	}

	now := clock.Now()
	t.Status = TimesheetStatusApproved
	t.ApprovedBy = &approvedBy
	t.ApprovedAt = &now
	t.RejectionReason = "" // Clear any previous rejection reason
	t.AllowedEdit = false
	t.RequestEditID = nil

	return nil
}

// Reject marks the timesheet as rejected
func (t *Timesheet) Reject(rejectionReason string) error {
	if !t.CanBeRejected() {
		return NewValidationError(fmt.Sprintf("bảng chấm công không thể bị từ chối: trạng thái hiện tại là '%s', chỉ các bảng chấm công có trạng thái 'chờ phê duyệt' mới có thể bị từ chối", t.Status))
	}

	if rejectionReason == "" {
		return NewValidationError("lý do từ chối là bắt buộc và không thể trống")
	}

	t.Status = TimesheetStatusRejected
	t.RejectionReason = rejectionReason
	t.ApprovedBy = nil
	t.ApprovedAt = nil
	t.AllowedEdit = false
	t.RequestEditID = nil

	return nil
}

// SubmitForApproval changes status to pending approval
func (t *Timesheet) SubmitForApproval() error {
	if !t.CanTransitionTo(TimesheetStatusPendingApproval) {
		return NewValidationError(fmt.Sprintf("bảng chấm công không thể được gửi để phê duyệt: trạng thái hiện tại là '%s', không thể chuyển sang 'chờ phê duyệt'", t.Status))
	}

	t.Status = TimesheetStatusPendingApproval
	t.RejectionReason = "" // Clear any previous rejection reason

	return nil
}

// CalculateAmount calculates the amount based on hours worked and pay rate
func (t *Timesheet) CalculateAmount() int64 {
	return int64(t.HoursWorked * float64(t.PayRate))
}

// RequiresApproval checks if this timesheet requires approval
func (t *Timesheet) RequiresApproval(userRole UserRole) bool {
	// Admin submissions are auto-approved
	if userRole == RoleAdmin {
		return false
	}

	// Partner submissions require approval
	return true
}

// GetWeekOfYear returns the week number of the year for this timesheet
func (t *Timesheet) GetWeekOfYear() (int, int) {
	year, week := t.Date.ISOWeek()
	return year, week
}

// GetMonthYear returns the month and year for this timesheet
func (t *Timesheet) GetMonthYear() (int, int) {
	return int(t.Date.Month()), t.Date.Year()
}

// IsWeekend returns true if the date falls on a weekend
func (t *Timesheet) IsWeekend() bool {
	weekday := t.Date.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

// SetPayRateFromConfig sets pay rate from payrate configuration based on paytype
func (t *Timesheet) SetPayRateFromConfig(payrate *Payrate, skillLevel string) error {
	if payrate == nil {
		return NewValidationError(constants.MsgPayrateConfigurationRequiredVN)
	}

	if !payrate.IsActiveOnDate(t.Date) {
		return NewValidationError(constants.MsgNoActivePayrateConfigVN)
	}

	// Use the paytype directly as path (3-level structure: position.dayType.hourType)
	// The paytype should already be properly formatted
	if t.PayType == "" {
		return NewValidationError(constants.MsgPayTypeRequiredVN)
	}

	vndAmount, err := payrate.GetRateValue(t.PayType)
	if err != nil {
		return err
	}

	// Set the payrate reference and amount
	t.PayrateID = payrate.ID
	t.PayRate = int64(vndAmount)
	return nil
}

// CalculateRevenueReceivable calculates the revenue receivable based on paid amount and fee percentage
func (t *Timesheet) CalculateRevenueReceivable(feePercentage float64) int64 {
	return int64(float64(t.PaidAmount) * feePercentage)
}
