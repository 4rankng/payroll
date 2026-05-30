package domain

import (
	"context"
	"time"
)

// BulkTransferFile represents a bulk transfer export request with all data needed to recreate the Excel file
type BulkTransferFile struct {
	ID                uint       `json:"id" gorm:"primarykey;type:bigint unsigned"`
	Filename          string     `json:"filename" gorm:"type:varchar(255);not null"`
	Cycle             *string    `json:"cycle,omitempty" gorm:"type:enum('weekly','monthly','flexible')"`
	CreatedBy         uint       `json:"created_by" gorm:"not null;type:bigint unsigned"`
	FromDate          *time.Time `json:"fromDate,omitempty" gorm:"type:date"`
	ToDate            *time.Time `json:"toDate,omitempty" gorm:"type:date"`
	ForMonth          *string    `json:"for_month,omitempty" gorm:"type:char(7)"`
	TransactionsCount int        `json:"transactions_count" gorm:"not null;default:0"`
	CompletedCount    int        `json:"completed_count" gorm:"not null;default:0"`
	FailedCount       int        `json:"failed_count" gorm:"not null;default:0"`
	TransferAmount    int64      `json:"transfer_amount" gorm:"type:bigint unsigned;not null;default:0"`
	Data              string     `json:"data" gorm:"type:json;not null"`
	AssetID           *uint      `json:"asset_id,omitempty" gorm:"column:asset_id;type:bigint unsigned;index:idx_asset_id"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	Source            string     `json:"source" gorm:"type:varchar(16);not null;default:'manual'"`
	Status            string     `json:"status" gorm:"type:varchar(16);not null;default:'exported'"`
	UploadedAt        *time.Time `json:"uploaded_at,omitempty" gorm:"type:timestamp null"`

	// Relationships
	Creator User   `json:"creator,omitempty" gorm:"foreignKey:CreatedBy;references:ID"`
	Asset   *Asset `json:"asset,omitempty" gorm:"foreignKey:AssetID;references:ID"`
}

// TableName sets the table name for GORM
func (BulkTransferFile) TableName() string {
	return "bulk_transfer_files"
}

// BulkTransferFileRepository defines the interface for bulk transfer file persistence operations
type BulkTransferFileRepository interface {
	Create(ctx context.Context, file *BulkTransferFile) error
	GetByID(ctx context.Context, id uint) (*BulkTransferFile, error)
	GetByIDWithAsset(ctx context.Context, id uint) (*BulkTransferFile, error)
	GetByFilename(ctx context.Context, filename string) (*BulkTransferFile, error)
	GetByAssetID(ctx context.Context, assetID uint) (*BulkTransferFile, error)
	GetMostRecentWithoutAsset(ctx context.Context) (*BulkTransferFile, error)
	FindByTimesheetIDs(ctx context.Context, timesheetIDs []uint) (*BulkTransferFile, error)
	FindByTimesheetIDGroups(ctx context.Context, timesheetIDs []uint) ([]*BulkTransferFile, error)
	FindDataByTransactionCodes(ctx context.Context, transactionCodes []string) ([]BulkTransferFileDataEntry, string, error)
	UpdateTransactionData(ctx context.Context, tx interface{}, id uint, updatedData string, assetID uint) error
	UpdateWithLock(ctx context.Context, id uint, updates map[string]interface{}) error
	ListWithFilters(ctx context.Context, cycle string, fromDate, toDate *time.Time, limit, offset int) ([]*BulkTransferFile, int64, error)
	ListForPayrollReport(ctx context.Context, fromDate, toDate time.Time) ([]*BulkTransferFile, error)
	DeleteOrphanFiles(ctx context.Context) (int64, error)
	GetRecentHighValueTransfers(ctx context.Context, fromDate, toDate time.Time) (map[uint][]int64, error)
	CountEmployeesPaidInPeriod(ctx context.Context, fromDate, toDate time.Time) (int, error)
	FindAllInDateRange(ctx context.Context, fromDate, toDate time.Time, files *[]*BulkTransferFile) error
	GetSalaryDistribution(ctx context.Context, fromDate, toDate *time.Time) (map[string][]int64, error)
	ListResultUploads(ctx context.Context, fromDate, toDate *time.Time, limit, offset int, sortOrder string) ([]*BulkTransferFile, int64, error)
	UpdateCounts(ctx context.Context, id uint, completedCount, failedCount int) error
	GetPendingUploadsByUser(ctx context.Context, userID uint) ([]*BulkTransferFile, error)
}

// StringPtr returns a pointer to the given string value
func StringPtr(s string) *string { return &s }

type BulkTransferFileDataEntry struct {
	FileID          *uint
	TransactionID   uint   // For linking to transaction (0 if not yet linked)
	TimesheetIDs    []uint // For linking timesheets to transaction
	EmployeeID      uint
	Amount          int64
	TransactionCode string
}
