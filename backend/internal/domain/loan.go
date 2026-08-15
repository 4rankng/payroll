package domain

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// LoanStatus represents the status of a loan
type LoanStatus string

const (
	LoanStatusActive LoanStatus = "active"
	LoanStatusClosed LoanStatus = "closed"
)

// Loan represents a loan from a lender to the company
type Loan struct {
	ID                   uint           `gorm:"primaryKey" json:"id"`
	LenderID             uint           `gorm:"not null;index" json:"lender_id"`
	Lender               *Lender        `gorm:"foreignKey:LenderID" json:"lender,omitempty"`
	LoanCode             string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"loan_code"`
	PrincipalAmount      int64          `gorm:"not null;default:0" json:"principal_amount"`
	InterestRateBps      int            `gorm:"not null" json:"interest_rate_bps"` // Basis points (1200 = 12%)
	TermMonths           int            `gorm:"not null" json:"term_months"`
	StartDate            time.Time      `gorm:"type:date;not null" json:"start_date"`
	EndDate              time.Time      `gorm:"type:date;not null" json:"end_date"`
	PaymentDayOfMonth    uint8          `gorm:"not null;default:1" json:"payment_day_of_month"`
	Status               LoanStatus     `gorm:"type:enum('active','closed');not null;default:'active'" json:"status"`
	Description          *string        `gorm:"type:text" json:"description"`
	DisbursedAt          *time.Time     `json:"disbursed_at"`
	OutstandingPrincipal int64          `gorm:"not null;default:0" json:"outstanding_principal"`
	TotalInterestPaid    int64          `gorm:"not null;default:0" json:"total_interest_paid"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy            uint           `gorm:"not null;index" json:"created_by"`
	Creator              *User          `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

// Validate checks business rules for loan
func (l *Loan) Validate() error {
	if l.PrincipalAmount <= 0 {
		return NewValidationError("Số tiền vay phải lớn hơn 0")
	}

	if l.InterestRateBps < 0 || l.InterestRateBps > 100000 {
		return NewValidationError("Lãi suất phải từ 0% đến 1000%")
	}

	if l.TermMonths < 0 {
		return NewValidationError("Thời hạn vay không được âm")
	}

	if !l.EndDate.IsZero() && l.StartDate.After(l.EndDate) {
		return NewValidationError("Ngày bắt đầu không được sau ngày kết thúc")
	}

	return nil
}

// CalculateMonthlyInterest returns monthly interest amount in VND
func (l *Loan) CalculateMonthlyInterest() int64 {
	if l.InterestRateBps == 0 || l.OutstandingPrincipal == 0 {
		return 0
	}

	// Interest = OutstandingPrincipal × (Rate / 10000) / 12
	annualInterest := (l.OutstandingPrincipal * int64(l.InterestRateBps)) / 10000
	return annualInterest / 12
}

// CalculateEndDate computes maturity date from start date and term
func CalculateEndDate(startDate time.Time, termMonths int) time.Time {
	return startDate.AddDate(0, termMonths, 0)
}

// CanDisburse checks if loan can be disbursed
func (l *Loan) CanDisburse() error {
	if l.DisbursedAt != nil {
		return NewValidationError("Khoản vay đã được giải ngân")
	}
	if l.Status != LoanStatusActive {
		return NewValidationError("Chỉ có thể giải ngân khoản vay đang hoạt động")
	}
	return nil
}

// CanRepay checks if repayment is allowed
func (l *Loan) CanRepay(amount int64) error {
	if l.DisbursedAt == nil {
		return NewValidationError("Khoản vay chưa được giải ngân")
	}
	if amount <= 0 {
		return NewValidationError("Số tiền trả nợ phải lớn hơn 0")
	}
	if amount > l.OutstandingPrincipal {
		return NewValidationError("Số tiền trả nợ vượt quá nợ gốc còn lại")
	}
	return nil
}

// ApplyScheduledPayment updates the aggregate only with the corresponding
// principal and interest components of a settled schedule installment.
func (l *Loan) ApplyScheduledPayment(principalAmount, interestAmount int64) error {
	if principalAmount < 0 || interestAmount < 0 {
		return NewValidationError("Phân bổ gốc và lãi không được âm")
	}
	if principalAmount > l.OutstandingPrincipal {
		return NewValidationError("Số tiền gốc trả vượt quá nợ gốc còn lại")
	}

	l.OutstandingPrincipal -= principalAmount
	l.TotalInterestPaid += interestAmount
	if l.OutstandingPrincipal == 0 {
		l.Status = LoanStatusClosed
	}
	return nil
}

// GenerateLoanCode creates unique loan code: LOAN-YYYY-NNN
func GenerateLoanCode(createdAt time.Time, sequence int) string {
	return fmt.Sprintf("LOAN-%d-%03d", createdAt.Year(), sequence)
}

// LoanRepository defines the interface for loan persistence operations
type LoanRepository interface {
	Create(ctx context.Context, loan *Loan) error
	GetByID(ctx context.Context, id uint) (*Loan, error)
	GetByIDForUpdate(ctx context.Context, id uint) (*Loan, error)
	Update(ctx context.Context, loan *Loan) error
	List(ctx context.Context, filters LoanFilters) ([]*Loan, int64, error)
	GetNextSequenceForYear(ctx context.Context, year int) (int, error)
	GetActiveLoans(ctx context.Context) ([]*Loan, error)
	Delete(ctx context.Context, id uint) error
}

// LoanFilters represents filtering options for loan queries
type LoanFilters struct {
	LenderID  *uint
	Status    *LoanStatus
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
}

// ScheduleItem represents a single payment schedule item
type ScheduleItem struct {
	Period  int       `json:"period"`
	DueDate time.Time `json:"due_date"`
	Type    string    `json:"type"` // descriptive label for the installment
	Amount  int64     `json:"amount"`
	Status  string    `json:"status"` // "pending" or "paid"
}
