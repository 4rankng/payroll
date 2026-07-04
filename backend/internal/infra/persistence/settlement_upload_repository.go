package persistence

import (
	"context"
	"errors"
	"fmt"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

// SettlementUploadRepository persists settlement_uploads idempotency rows.
type SettlementUploadRepository struct {
	*BaseRepository
}

// NewSettlementUploadRepository wires the repository onto the shared database,
// returning the domain interface so callers depend on the port.
func NewSettlementUploadRepository(db *Database) domain.SettlementUploadRepository {
	return &SettlementUploadRepository{BaseRepository: NewBaseRepository(db)}
}

// GetByFileHash returns the prior upload row for the given hash. A missing row
// (first settlement of this content) returns (nil, nil) so the caller can treat
// absence as "not previously processed" without branching on a sentinel error.
func (r *SettlementUploadRepository) GetByFileHash(ctx context.Context, fileHash string) (*domain.SettlementUpload, error) {
	var row domain.SettlementUpload
	err := r.DB.WithContext(ctx).Where("file_hash = ?", fileHash).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("settlement_uploads: get by file_hash: %w", err)
	}
	return &row, nil
}

// Create inserts a new settlement_uploads row. The unique index on file_hash
// makes a concurrent double-insert return an error rather than duplicating.
func (r *SettlementUploadRepository) Create(ctx context.Context, upload *domain.SettlementUpload) error {
	if err := r.DB.WithContext(ctx).Create(upload).Error; err != nil {
		return fmt.Errorf("settlement_uploads: create: %w", err)
	}
	return nil
}
