package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Lender represents a lender (bank or individual) who provides loans to the company
type Lender struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	Name              string         `gorm:"type:varchar(255);not null" json:"name"`
	CCCD              *string        `gorm:"type:varchar(50)" json:"cccd"` // Nullable for organizations
	Email             *string        `gorm:"type:varchar(255)" json:"email"`
	Mobile            *string        `gorm:"type:varchar(50)" json:"mobile"`
	Notes             *string        `gorm:"type:text" json:"notes"`
	BankID            *uint          `gorm:"type:bigint unsigned" json:"bank_id"`         // Foreign key reference to banks table
	BankAccountNumber *string        `gorm:"type:varchar(30)" json:"bank_account_number"` // Lender's bank account number
	BankAccountName   *string        `gorm:"type:varchar(255)" json:"bank_account_name"`  // Lender's bank account name
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// Validate checks business rules for lender
func (l *Lender) Validate() error {
	if l.Name == "" {
		return NewValidationError("Tên người cho vay không được để trống")
	}
	if len(l.Name) > 255 {
		return NewValidationError("Tên người cho vay không được vượt quá 255 ký tự")
	}
	if l.BankAccountNumber != nil && len(*l.BankAccountNumber) > 30 {
		return NewValidationError("Số tài khoản ngân hàng không được vượt quá 30 ký tự")
	}
	if l.BankAccountName != nil && len(*l.BankAccountName) > 255 {
		return NewValidationError("Tên tài khoản ngân hàng không được vượt quá 255 ký tự")
	}
	return nil
}

// LenderLoanSummary aggregates the live (non soft-deleted) loans linked to a lender
type LenderLoanSummary struct {
	TotalPrincipalBorrowed int64
	TotalPrincipalRepaid   int64
	OutstandingPrincipal   int64
	ActiveLoansCount       int
	TotalLoansCount        int
}

// LenderRepository defines the interface for lender persistence operations
type LenderRepository interface {
	Create(ctx context.Context, lender *Lender) error
	GetByID(ctx context.Context, id uint) (*Lender, error)
	Update(ctx context.Context, lender *Lender) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filters LenderFilters) ([]*Lender, int64, error)
	// HasAnyLoans returns true if any live loan references the lender
	HasAnyLoans(ctx context.Context, lenderID uint) (bool, error)
	// GetLoanSummary aggregates live loans linked to the lender
	GetLoanSummary(ctx context.Context, lenderID uint) (LenderLoanSummary, error)
}

// LenderFilters represents filtering options for lender queries
type LenderFilters struct {
	Search    *string
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
}
