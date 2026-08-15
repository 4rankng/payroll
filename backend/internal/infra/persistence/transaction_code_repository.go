package persistence

import (
	"context"
	"encoding/json"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"

	"gorm.io/gorm"
)

type TransactionCodeRepository struct {
	*BaseRepository
	errorHandler *common.RepoErrorHandler
}

func NewTransactionCodeRepository(db *Database) domain.TransactionCodeRepository {
	return &TransactionCodeRepository{
		BaseRepository: NewBaseRepository(db),
		errorHandler:   common.NewRepoErrorHandler(),
	}
}

// getDB keeps transaction-code reads and writes in the transaction propagated
// by application services. This is required by result uploads, where the new
// bulk-transfer history ID must commit or roll back together with its asset.
func (r *TransactionCodeRepository) getDB(ctx context.Context) *gorm.DB {
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.DB.WithContext(ctx)
}

func (r *TransactionCodeRepository) Create(ctx context.Context, tc *domain.TransactionCode) error {
	return r.getDB(ctx).Create(tc).Error
}

func (r *TransactionCodeRepository) GetByCode(ctx context.Context, code string) (*domain.TransactionCode, error) {
	var tc domain.TransactionCode
	err := r.getDB(ctx).
		Where("code = ?", code).
		First(&tc).Error

	if err != nil {
		return nil, r.errorHandler.HandleGetError(err, "transaction_code", code)
	}

	return &tc, nil
}

// GetAllCodes returns all existing transaction codes as a set for uniqueness checking
func (r *TransactionCodeRepository) GetAllCodes(ctx context.Context) (map[string]struct{}, error) {
	var codes []string
	if err := r.getDB(ctx).Model(&domain.TransactionCode{}).Pluck("code", &codes).Error; err != nil {
		return nil, r.errorHandler.HandleListError(err, "transaction_codes")
	}

	result := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		result[code] = struct{}{}
	}
	return result, nil
}

// CreateBatch creates multiple transaction codes in a single transaction
func (r *TransactionCodeRepository) CreateBatch(ctx context.Context, tcs []*domain.TransactionCode) error {
	if len(tcs) == 0 {
		return nil
	}
	return r.getDB(ctx).CreateInBatches(tcs, 100).Error
}

// FindByCodes returns transaction codes matching the provided codes.
// Chunked at DefaultChunkSize to stay under MySQL's packet limit for large
// code sets (e.g. bank-transfer-histories with hundreds of codes per month).
func (r *TransactionCodeRepository) FindByCodes(ctx context.Context, codes []string) ([]*domain.TransactionCode, error) {
	if len(codes) == 0 {
		return []*domain.TransactionCode{}, nil
	}

	tcs, err := common.ChunkStrings(ctx, r.getDB(ctx), codes, common.DefaultChunkSize,
		func(tx *gorm.DB, batch []string) ([]*domain.TransactionCode, error) {
			var batchTCs []*domain.TransactionCode
			if err := tx.Where("code IN ?", batch).Find(&batchTCs).Error; err != nil {
				return nil, r.errorHandler.HandleListError(err, "transaction_codes")
			}
			return batchTCs, nil
		})
	if err != nil {
		return nil, err
	}

	return tcs, nil
}

// UpdateFileIDByCodes updates the file_id in the Data JSON for transaction codes matching the given codes
func (r *TransactionCodeRepository) UpdateFileIDByCodes(ctx context.Context, codes []string, fileID uint) error {
	if len(codes) == 0 {
		return nil
	}

	// Fetch existing codes
	tcs, err := r.FindByCodes(ctx, codes)
	if err != nil {
		return err
	}

	for _, tc := range tcs {
		var tcData domain.TransactionCodeData
		if err := json.Unmarshal(tc.Data, &tcData); err != nil {
			continue
		}

		if tcData.WeeklyPay != nil {
			tcData.WeeklyPay.FileID = &fileID
		}
		if tcData.MonthlyPay != nil {
			tcData.MonthlyPay.FileID = &fileID
		}

		updatedBytes, err := json.Marshal(tcData)
		if err != nil {
			continue
		}
		tc.Data = updatedBytes
	}

	return r.getDB(ctx).Save(tcs).Error
}
