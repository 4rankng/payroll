package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Account represents a chart of accounts entry for proper account classification
type Account struct {
	ID        uint            `json:"id" gorm:"primarykey;type:bigint unsigned"`
	Code      string          `json:"code" gorm:"type:varchar(20);uniqueIndex;not null;comment:'Account code (e.g., 1000, 1100)'"`
	Name      string          `json:"name" gorm:"type:varchar(100);not null;comment:'Account name'"`
	Type      AccountCategory `json:"type" gorm:"type:enum('asset','liability','equity','revenue','expense');not null;comment:'Account type for classification'"`
	ParentID  *uint           `json:"parent_id" gorm:"type:bigint unsigned;comment:'Parent account for hierarchy'"`
	DeletedAt gorm.DeletedAt  `json:"-" gorm:"index"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`

	// Relationships
	Parent   *Account  `json:"parent,omitempty" gorm:"foreignKey:ParentID;references:ID"`
	Children []Account `json:"children,omitempty" gorm:"foreignKey:ParentID;references:ID"`
}

// TableName specifies the table name for the Account model
func (Account) TableName() string {
	return "accounts"
}

// AccountRepository defines the interface for account persistence operations
type AccountRepository interface {
	Create(ctx context.Context, account *Account) error
	GetByID(ctx context.Context, id uint) (*Account, error)
	GetByCode(ctx context.Context, code string) (*Account, error)
	List(ctx context.Context) ([]*Account, error)
	Update(ctx context.Context, account *Account) error
	Delete(ctx context.Context, id uint) error
	GetByType(ctx context.Context, accountType AccountCategory) ([]*Account, error)
	GetChildren(ctx context.Context, parentID uint) ([]*Account, error)
}
