package persistence

import (
	"context"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"

	"gorm.io/gorm"
)

type BankRepository struct {
	*BaseRepository
	filterBuilder *common.FilterBuilder
	errorHandler  *common.RepoErrorHandler
}

func NewBankRepository(db *Database) domain.BankRepository {
	return &BankRepository{
		BaseRepository: NewBaseRepository(db),
		filterBuilder:  common.NewFilterBuilder(db.DB),
		errorHandler:   common.NewRepoErrorHandler(),
	}
}

func (r *BankRepository) Create(ctx context.Context, bank *domain.Bank) error {
	return r.DB.WithContext(ctx).Create(bank).Error
}

func (r *BankRepository) GetByID(ctx context.Context, id uint) (*domain.Bank, error) {
	var bank domain.Bank
	err := r.DB.WithContext(ctx).
		First(&bank, id).Error

	if err != nil {
		return nil, r.errorHandler.HandleGetError(err, "bank", id)
	}

	return &bank, nil
}

func (r *BankRepository) Update(ctx context.Context, bank *domain.Bank) error {
	return r.DB.WithContext(ctx).Save(bank).Error
}

func (r *BankRepository) Delete(ctx context.Context, id uint) error {
	return r.DB.WithContext(ctx).Delete(&domain.Bank{}, id).Error
}

func (r *BankRepository) List(ctx context.Context, filters domain.BankFilters) ([]*domain.Bank, error) {
	var banks []*domain.Bank
	query := r.DB.WithContext(ctx).Model(&domain.Bank{})

	// Apply filters
	query = r.applyFilters(query, filters)

	// Apply sorting and pagination using FilterBuilder
	query = r.filterBuilder.ApplySorting(query, filters.SortBy, filters.SortOrder, "bank_code")
	query = r.filterBuilder.ApplyPagination(query, filters.Limit, filters.Offset)

	err := query.Find(&banks).Error
	if err != nil {
		return nil, r.errorHandler.HandleListError(err, "bank")
	}

	return banks, nil
}

func (r *BankRepository) Count(ctx context.Context, filters domain.BankFilters) (int64, error) {
	var count int64
	query := r.DB.WithContext(ctx).Model(&domain.Bank{})

	// Apply filters
	query = r.applyFilters(query, filters)

	err := query.Count(&count).Error
	return count, err
}

func (r *BankRepository) SearchByBranchName(ctx context.Context, searchTerm string, limit int) ([]*domain.Bank, error) {
	var banks []*domain.Bank
	searchPattern := "%" + searchTerm + "%"

	query := r.DB.WithContext(ctx).
		Where("LOWER(branch_name) LIKE LOWER(?)", searchPattern).
		Order("branch_name ASC")

	if limit <= 0 || limit > 100 {
		limit = 100 // Set to max 100 records for search
	}
	query = query.Limit(limit)

	err := query.Find(&banks).Error
	return banks, err
}

// applyFilters applies the common filters to the query
func (r *BankRepository) applyFilters(query *gorm.DB, filters domain.BankFilters) *gorm.DB {
	// Search functionality
	if filters.Search != "" {
		searchTerm := "%" + filters.Search + "%"
		query = query.Where("LOWER(branch_name) LIKE LOWER(?)", searchTerm)
	}

	return query
}

// GetActiveBanks returns all active banks
func (r *BankRepository) GetActiveBanks(ctx context.Context) ([]*domain.Bank, error) {
	var banks []*domain.Bank
	err := r.DB.WithContext(ctx).
		Order("branch_name ASC").
		Find(&banks).Error

	return banks, err
}

func (r *BankRepository) FindByBankCode(ctx context.Context, bankCode string) (*domain.Bank, error) {
	var bank domain.Bank
	err := r.DB.WithContext(ctx).
		Where("bank_code = ?", bankCode).
		First(&bank).Error
	if err != nil {
		return nil, err
	}
	return &bank, nil
}

func (r *BankRepository) FindBySwiftCode(ctx context.Context, swiftCode string) (*domain.Bank, error) {
	var bank domain.Bank
	err := r.DB.WithContext(ctx).
		Where("swift_code = ?", swiftCode).
		First(&bank).Error
	if err != nil {
		return nil, err
	}
	return &bank, nil
}
