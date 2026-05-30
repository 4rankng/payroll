package domain

import (
	"context"
	"time"
)

// ScheduleStatus represents the status of a repayment schedule
type ScheduleStatus string

const (
	ScheduleStatusPending ScheduleStatus = "pending"
	ScheduleStatusPaid    ScheduleStatus = "paid"
)

// LoanRepaymentSchedule represents a single repayment schedule item
type LoanRepaymentSchedule struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	LoanID        uint           `gorm:"not null;index" json:"loan_id"`
	Period        int            `gorm:"not null" json:"period"`
	DueDate       time.Time      `gorm:"type:date;not null" json:"due_date"`
	Amount        int64          `gorm:"not null" json:"amount"`
	Status        ScheduleStatus `gorm:"type:enum('pending','paid');not null;default:'pending'" json:"status"`
	PaidAt        *time.Time     `json:"paid_at,omitempty"`
	PaymentRef    *string        `gorm:"type:varchar(255)" json:"payment_reference,omitempty"`
	TransactionID *uint          `gorm:"type:bigint unsigned" json:"transaction_id,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// LoanRepaymentScheduleRepository defines the interface for loan repayment schedule persistence operations
type LoanRepaymentScheduleRepository interface {
	Create(ctx context.Context, schedule *LoanRepaymentSchedule) error
	GetByID(ctx context.Context, id uint) (*LoanRepaymentSchedule, error)
	GetByLoanID(ctx context.Context, loanID uint) ([]*LoanRepaymentSchedule, error)
	GetByTransactionID(ctx context.Context, transactionID uint) (*LoanRepaymentSchedule, error)
	ListByTransactionIDs(ctx context.Context, transactionIDs []uint) ([]*LoanRepaymentSchedule, error)
	GetPendingSchedulesByLoan(ctx context.Context, loanID uint) ([]*LoanRepaymentSchedule, error)
	GetDueSchedules(ctx context.Context, dueDate time.Time) ([]*LoanRepaymentSchedule, error)
	Update(ctx context.Context, schedule *LoanRepaymentSchedule) error
	Delete(ctx context.Context, id uint) error
}
