package domain

import (
	"context"
	"time"
)

// TransactionCode tracks transaction codes linking bank transfers to advance payment requests
type TransactionCode struct {
	ID        uint      `json:"id" gorm:"primaryKey;type:bigint unsigned"`
	Code      string    `json:"code" gorm:"size:36;uniqueIndex;not null"`
	Data      []byte    `json:"data" gorm:"type:json;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TransactionCodeData represents the JSON structure stored in Data field
// At lookup, exactly one of FlexPay, WeeklyPay, MonthlyPay, or ManualDisbursement will be populated
type TransactionCodeData struct {
	// Flex pay - advance payment requests (used by advance_payment service)
	FlexPay *FlexPayData `json:"flex_pay,omitempty"`

	// Weekly pay - timesheet IDs (used by bulk transfer export, weekly cycle)
	WeeklyPay *CyclePayData `json:"weekly_pay,omitempty"`

	// Monthly pay - timesheet IDs (used by bulk transfer export, monthly cycle)
	MonthlyPay *CyclePayData `json:"monthly_pay,omitempty"`

	// Manual disbursement - admin manual transfer
	ManualDisbursement *ManualDisbursementData `json:"manual_disbursement,omitempty"`
}

// ManualDisbursementData contains details for admin manual disbursements
type ManualDisbursementData struct {
	Amount      int64  `json:"amount"`
	BankCode    string `json:"bank_code"`
	AccountNo   string `json:"account_no"`
	AccountName string `json:"account_name"`
	Description string `json:"description"`
}

// FlexPayData contains advance payment request IDs for flex pay
type FlexPayData struct {
	RequestIDs []uint64 `json:"request_ids"`
}

// CyclePayData contains timesheet data for weekly/monthly pay cycles
type CyclePayData struct {
	TimesheetIDs []uint `json:"timesheet_ids"`
	EmployeeID   uint   `json:"employee_id"`
	ProjectID    uint   `json:"project_id"`
	Amount       int64  `json:"amount"`
	FileID       *uint  `json:"file_id"`
}

// GetTimesheetIDs returns timesheet IDs from whichever pay field is set
func (t *TransactionCodeData) GetTimesheetIDs() []uint {
	if t.WeeklyPay != nil {
		return t.WeeklyPay.TimesheetIDs
	}
	if t.MonthlyPay != nil {
		return t.MonthlyPay.TimesheetIDs
	}
	return nil
}

// GetEmployeeID returns employee ID from whichever pay field is set
func (t *TransactionCodeData) GetEmployeeID() uint {
	if t.WeeklyPay != nil {
		return t.WeeklyPay.EmployeeID
	}
	if t.MonthlyPay != nil {
		return t.MonthlyPay.EmployeeID
	}
	return 0
}

// GetAmount returns amount from whichever pay field is set
func (t *TransactionCodeData) GetAmount() int64 {
	if t.WeeklyPay != nil {
		return t.WeeklyPay.Amount
	}
	if t.MonthlyPay != nil {
		return t.MonthlyPay.Amount
	}
	return 0
}

// GetFileID returns file ID from whichever pay field is set
func (t *TransactionCodeData) GetFileID() *uint {
	if t.WeeklyPay != nil {
		return t.WeeklyPay.FileID
	}
	if t.MonthlyPay != nil {
		return t.MonthlyPay.FileID
	}
	return nil
}

// GetRequestIDs returns advance pay request IDs (backward compat)
func (t *TransactionCodeData) GetRequestIDs() []uint64 {
	if t.FlexPay != nil {
		return t.FlexPay.RequestIDs
	}
	return nil
}

// GetCycle returns the payment cycle ("flexible", "weekly", or "monthly") based on which pay field is set
func (t *TransactionCodeData) GetCycle() string {
	if t.FlexPay != nil {
		return "flexible"
	}
	if t.WeeklyPay != nil {
		return "weekly"
	}
	if t.MonthlyPay != nil {
		return "monthly"
	}
	return ""
}

// TransactionCodeRepository defines the interface for transaction code persistence operations
type TransactionCodeRepository interface {
	Create(ctx context.Context, tc *TransactionCode) error
	GetByCode(ctx context.Context, code string) (*TransactionCode, error)
	// GetAllCodes returns all existing transaction codes as a set for uniqueness checking
	GetAllCodes(ctx context.Context) (map[string]struct{}, error)
	// CreateBatch creates multiple transaction codes in a single transaction
	CreateBatch(ctx context.Context, tcs []*TransactionCode) error
	// FindByCodes returns transaction codes matching the provided codes in a single indexed query
	FindByCodes(ctx context.Context, codes []string) ([]*TransactionCode, error)
	// UpdateFileIDByCodes updates the file_id in the Data JSON for transaction codes matching the given codes
	UpdateFileIDByCodes(ctx context.Context, codes []string, fileID uint) error
}
