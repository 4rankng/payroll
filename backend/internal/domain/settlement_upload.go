package domain

import (
	"context"
	"time"
)

// SettlementUpload is an idempotency / audit row for a FlexPay reconciliation
// settlement run. A row is written only after ProcessSettlementFile completes
// with no per-iteration failures, keyed by FileHash (SHA-256 of the settled
// request-ID set) so a re-upload of the same recon content short-circuits to
// the prior result instead of re-touching the books.
type SettlementUpload struct {
	ID             uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	FileHash       string    `gorm:"column:file_hash;type:char(64);uniqueIndex"`
	UploadedAt     time.Time `gorm:"column:uploaded_at"`
	RequestIDsJSON string    `gorm:"column:request_ids_json;type:json"`
	SettledCount   int64     `gorm:"column:settled_count"`
}

// TableName pins the wallet_payments-adjacent audit table name.
func (SettlementUpload) TableName() string { return "settlement_uploads" }

// SettlementUploadRepository persists settlement idempotency rows.
type SettlementUploadRepository interface {
	// GetByFileHash returns the prior upload row for the given hash, or
	// (nil, nil) when no row exists (first settlement of this content).
	GetByFileHash(ctx context.Context, fileHash string) (*SettlementUpload, error)
	Create(ctx context.Context, upload *SettlementUpload) error
}
