package domain

import (
	"context"
)

// Bank represents a bank branch in the domain
type Bank struct {
	ID         uint   `json:"id" gorm:"primarykey;type:bigint unsigned"`
	BranchName string `json:"branch_name" gorm:"type:varchar(255);uniqueIndex;not null"`
	Bin        string `json:"bin" gorm:"type:varchar(6);null"`
	BankCode   string `json:"bank_code" gorm:"type:varchar(10);null"`
	SwiftCode  string `json:"swift_code" gorm:"type:varchar(11);null"`
}

// BankRepository defines the interface for bank persistence operations
type BankRepository interface {
	Create(ctx context.Context, bank *Bank) error
	GetByID(ctx context.Context, id uint) (*Bank, error)
	Update(ctx context.Context, bank *Bank) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filters BankFilters) ([]*Bank, error)
	Count(ctx context.Context, filters BankFilters) (int64, error)
	SearchByBranchName(ctx context.Context, searchTerm string, limit int) ([]*Bank, error)
	FindByBankCode(ctx context.Context, bankCode string) (*Bank, error)
	FindBySwiftCode(ctx context.Context, swiftCode string) (*Bank, error)
}

// BankFilters represents filtering options for bank queries
type BankFilters struct {
	Search    string // Search in branch_name
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// ValidateBranchName validates the bank's branch name
func (b *Bank) ValidateBranchName() error {
	if b.BranchName == "" {
		return NewValidationError("branch name is required")
	}
	if len(b.BranchName) > 255 {
		return NewValidationError("branch name must be less than 255 characters")
	}
	return nil
}

// IsValid validates the entire bank entity
func (b *Bank) IsValid() error {
	if err := b.ValidateBranchName(); err != nil {
		return err
	}
	return nil
}

// CanBeUsed returns true if bank can be used for transactions
func (b *Bank) CanBeUsed() bool {
	return b.BranchName != ""
}
