package domain

import (
	"context"
	"time"

	"api-server/internal/pkg/clock"
	"gorm.io/gorm"
)

// Settlement represents a payment event that settles (partially or fully) a transaction
type Settlement struct {
	ID             uint           `json:"id" gorm:"primarykey;type:bigint unsigned"`
	SettlementUUID *string        `json:"settlement_uuid" gorm:"type:varchar(36);uniqueIndex;comment:'Idempotency key for async processing'"`
	TransactionID  uint           `json:"transaction_id" gorm:"not null;type:bigint unsigned"`
	Amount         int64          `json:"amount" gorm:"type:bigint;not null"`
	SettlementDate time.Time      `json:"settlement_date" gorm:"type:date;not null"`
	ProofURL       string         `json:"proof_url" gorm:"type:varchar(500)"`
	ProofAssetID   *uint          `json:"proof_asset_id" gorm:"type:bigint unsigned"`
	PaymentMethod  string         `json:"payment_method" gorm:"type:varchar(50);default:'cash'"`
	Notes          string         `json:"notes" gorm:"type:text"`
	CreatedBy      uint           `json:"created_by" gorm:"not null;type:bigint unsigned"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	Transaction *Transaction `json:"transaction,omitempty" gorm:"foreignKey:TransactionID;references:ID"`
	ProofAsset  *Asset       `json:"proof_asset,omitempty" gorm:"foreignKey:ProofAssetID;references:ID"`
	Creator     User         `json:"creator" gorm:"foreignKey:CreatedBy;references:ID"`
}

// SettlementRepository defines the interface for settlement persistence operations
type SettlementRepository interface {
	Create(ctx context.Context, settlement *Settlement) error
	GetByID(ctx context.Context, id uint) (*Settlement, error)
	GetByTransactionID(ctx context.Context, transactionID uint) ([]*Settlement, error)
	GetTotalSettledAmount(ctx context.Context, transactionID uint) (int64, error)
	List(ctx context.Context, filters SettlementFilters) ([]*Settlement, error)
	Count(ctx context.Context, filters SettlementFilters) (int64, error)
}

// SettlementFilters represents filtering options for settlement queries
type SettlementFilters struct {
	TransactionID *uint
	PaymentMethod *string
	FromDate      *time.Time
	ToDate        *time.Time
	CreatedBy     *uint
	Limit         int
	Offset        int
	SortBy        string
	SortOrder     string
}

// Validate validates the settlement
func (s *Settlement) Validate() error {
	if s.TransactionID == 0 {
		return NewValidationError("transaction_id là bắt buộc")
	}

	if s.Amount <= 0 {
		return NewValidationError("số tiền phải lớn hơn 0")
	}

	if s.SettlementDate.IsZero() {
		return NewValidationError("ngày thanh toán là bắt buộc")
	}

	// Validate settlement date is not in the future
	// Compare date-only to avoid timezone mismatch issues around midnight
	now := clock.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	settlementDay := time.Date(s.SettlementDate.Year(), s.SettlementDate.Month(), s.SettlementDate.Day(), 0, 0, 0, 0, now.Location())
	if settlementDay.After(today) {
		return NewValidationError("ngày thanh toán không được ở tương lai")
	}

	if s.CreatedBy == 0 {
		return NewValidationError("người tạo là bắt buộc")
	}

	return nil
}

// TableName specifies the table name for GORM
func (Settlement) TableName() string {
	return "settlements"
}

// Removed FormatAmount: prefer utils.FormatVND at call sites
