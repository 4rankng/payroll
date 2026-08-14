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

// LoanRepaymentReminder is the read model used by the one-day-ahead reminder.
// It keeps notification fan-out independent from persistence entities and
// allows the repository to fetch loan and lender details without N+1 queries.
type LoanRepaymentReminder struct {
	ScheduleID uint
	LoanCode   string
	LenderName string
	Period     int
	DueDate    time.Time
	Amount     int64
}

// LoanRepaymentScheduleRepository defines the interface for loan repayment schedule persistence operations
type LoanRepaymentScheduleRepository interface {
	Create(ctx context.Context, schedule *LoanRepaymentSchedule) error
	GetByID(ctx context.Context, id uint) (*LoanRepaymentSchedule, error)
	GetByLoanID(ctx context.Context, loanID uint) ([]*LoanRepaymentSchedule, error)
	// GetByLoanIDs batch-fetches schedules for multiple loans in a single query,
	// grouped by loan id. Use this in list endpoints to avoid an N+1 of
	// GetByLoanID per loan. Loans with no schedules are absent from the map.
	GetByLoanIDs(ctx context.Context, loanIDs []uint) (map[uint][]*LoanRepaymentSchedule, error)
	GetByTransactionID(ctx context.Context, transactionID uint) (*LoanRepaymentSchedule, error)
	ListByTransactionIDs(ctx context.Context, transactionIDs []uint) ([]*LoanRepaymentSchedule, error)
	GetPendingSchedulesByLoan(ctx context.Context, loanID uint) ([]*LoanRepaymentSchedule, error)
	GetDueSchedules(ctx context.Context, dueDate time.Time) ([]*LoanRepaymentSchedule, error)
	ListPendingForReminder(ctx context.Context, start, end time.Time) ([]*LoanRepaymentReminder, error)
	Update(ctx context.Context, schedule *LoanRepaymentSchedule) error
	Delete(ctx context.Context, id uint) error
}
