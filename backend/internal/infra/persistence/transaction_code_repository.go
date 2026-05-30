package persistence

import (
	"context"
	"encoding/json"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
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

func (r *TransactionCodeRepository) Create(ctx context.Context, tc *domain.TransactionCode) error {
	return r.DB.WithContext(ctx).Create(tc).Error
}

func (r *TransactionCodeRepository) GetByCode(ctx context.Context, code string) (*domain.TransactionCode, error) {
	var tc domain.TransactionCode
	err := r.DB.WithContext(ctx).
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
	if err := r.DB.WithContext(ctx).Model(&domain.TransactionCode{}).Pluck("code", &codes).Error; err != nil {
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
	return r.DB.WithContext(ctx).CreateInBatches(tcs, 100).Error
}

// FindByCodes returns transaction codes matching the provided codes in a single indexed query
func (r *TransactionCodeRepository) FindByCodes(ctx context.Context, codes []string) ([]*domain.TransactionCode, error) {
	if len(codes) == 0 {
		return []*domain.TransactionCode{}, nil
	}

	var tcs []*domain.TransactionCode
	err := r.DB.WithContext(ctx).
		Where("code IN ?", codes).
		Find(&tcs).Error

	if err != nil {
		return nil, r.errorHandler.HandleListError(err, "transaction_codes")
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

	return r.DB.WithContext(ctx).Save(tcs).Error
}
