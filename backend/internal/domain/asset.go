package domain

import (
	"context"
	"time"
)

type Asset struct {
	ID         uint      `json:"id" gorm:"primarykey;type:bigint unsigned"`
	Filename   string    `json:"filename" gorm:"type:varchar(255);not null"`
	FilePath   string    `json:"file_path" gorm:"type:varchar(500);not null"`
	UploadType string    `json:"upload_type" gorm:"type:varchar(50);not null;default:'general'"`
	Checksum   *string   `json:"checksum,omitempty" gorm:"type:varchar(64);index:idx_assets_checksum"`
	UploadedBy uint      `json:"uploaded_by" gorm:"not null;type:bigint unsigned"`
	Metadata   *string   `json:"metadata,omitempty" gorm:"type:json"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Relationships
	Uploader User `json:"uploader" gorm:"foreignKey:UploadedBy;references:ID"`
}

// GetAuditEntityType implements Auditable interface
func (a Asset) GetAuditEntityType() string {
	return "asset"
}

// GetAuditEntityID implements Auditable interface
func (a Asset) GetAuditEntityID() uint {
	return a.ID
}

type AssetRepository interface {
	Create(ctx context.Context, asset *Asset) (*Asset, error)
	GetByID(ctx context.Context, id uint) (*Asset, error)
	GetByChecksum(ctx context.Context, checksum string, uploadType string) (*Asset, error)
	CountByFilePath(ctx context.Context, filePath string) (int64, error)
	List(ctx context.Context, filters AssetFilters) ([]*Asset, error)
	Count(ctx context.Context, filters AssetFilters) (int64, error)
	Update(ctx context.Context, asset *Asset) error
	UpdateMetadata(ctx context.Context, id uint, metadata string) error
	FindOrphaned(ctx context.Context, olderThan time.Time) ([]*Asset, error)
}

type AssetFilters struct {
	UploadType    *string
	UploadTypes   []string
	UploadedBy    *uint
	FromDate      *time.Time
	ToDate        *time.Time
	MetadataQuery map[string]string // JSON path key → value for metadata filtering
	MetadataLike  map[string]string // JSON path key → substring term (LIKE %term%)
	// MetadataNotNull requires metadata IS NOT NULL (the renderer drops NULL rows).
	MetadataNotNull bool
	// GroupByProject orders by metadata project_id, then created_at DESC.
	GroupByProject bool
	Limit          int
	Offset         int
	SortBy         string
	SortOrder      string
}

type AssetUploadRequest struct {
	UploadType string `json:"upload_type" binding:"required"`
}

const (
	UploadTypeLedgerEvidence            = "ledger_evidence"
	UploadTypeBulkTransferResult        = "bulk_transfer_result"   // payroll weekly/monthly transfer results
	UploadTypeAdvancePaymentResult      = "advance_payment_result" // advance payment (ứng lương) transfer results
	UploadTypeGeneral                   = "general"
	UploadTypeDocument                  = "document"
	UploadTypeFlexPayImport             = "flex_pay_import"
	UploadTypeAdvancePaymentExport      = "advance_payment_export"        // exported advance payment transfer files
	UploadTypeAdvancePaymentSaoKeExport = "advance_payment_sao_ke_export" // sao ke reconciliation exports
	UploadTypeAdvancePaymentSaoKeResult = "advance_payment_sao_ke_result" // uploaded sao ke settlement results
	UploadTypePartnerBCCImport          = "partner_bcc_import"
	UploadTypeSaoKe                     = "sao_ke"           // sao kê reminder email attachments
	UploadTypeSettlementProof           = "settlement_proof" // settlement evidence uploads
	UploadTypeEmployeeImports           = "employee_imports" // bulk employee import files
)

// allowedUploadTypes is the closed set of upload_type values accepted for file
// storage. It guards against path traversal: upload_type is used as the first
// path segment when storing files, so only these fixed (traversal-free)
// constants are permitted.
var allowedUploadTypes = map[string]bool{
	UploadTypeLedgerEvidence:            true,
	UploadTypeBulkTransferResult:        true,
	UploadTypeAdvancePaymentResult:      true,
	UploadTypeGeneral:                   true,
	UploadTypeDocument:                  true,
	UploadTypeFlexPayImport:             true,
	UploadTypeAdvancePaymentExport:      true,
	UploadTypeAdvancePaymentSaoKeExport: true,
	UploadTypeAdvancePaymentSaoKeResult: true,
	UploadTypePartnerBCCImport:          true,
	UploadTypeSaoKe:                     true,
	UploadTypeSettlementProof:           true,
	UploadTypeEmployeeImports:           true,
}

// IsValidUploadType reports whether t is one of the recognized upload_type
// constants. Unknown values (including traversal sequences) are rejected.
func IsValidUploadType(t string) bool {
	return allowedUploadTypes[t]
}

const (
	// Import job types
	ImportJobTypeFlexPay = "flex_pay"

	// Import job statuses
	ImportJobStatusPending    = "pending"
	ImportJobStatusProcessing = "processing"
	ImportJobStatusCompleted  = "completed"
	ImportJobStatusFailed     = "failed"
)
