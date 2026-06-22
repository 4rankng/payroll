package persistence

import (
	"context"
	"fmt"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"

	"gorm.io/gorm"
)

type LenderRepository struct {
	*BaseRepository
}

func NewLenderRepository(db *Database) domain.LenderRepository {
	return &LenderRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *LenderRepository) Create(ctx context.Context, lender *domain.Lender) error {
	return r.DB.WithContext(ctx).Create(lender).Error
}

func (r *LenderRepository) GetByID(ctx context.Context, id uint) (*domain.Lender, error) {
	var lender domain.Lender
	err := r.DB.WithContext(ctx).
		First(&lender, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("lender not found")
		}
		return nil, err
	}

	return &lender, nil
}

func (r *LenderRepository) Update(ctx context.Context, lender *domain.Lender) error {
	return r.DB.WithContext(ctx).Save(lender).Error
}

func (r *LenderRepository) Delete(ctx context.Context, id uint) error {
	return r.DB.WithContext(ctx).Delete(&domain.Lender{}, id).Error
}

func (r *LenderRepository) List(ctx context.Context, filters domain.LenderFilters) ([]*domain.Lender, int64, error) {
	var lenders []*domain.Lender
	var total int64

	query := r.DB.WithContext(ctx).Model(&domain.Lender{})

	// Apply filters
	query = r.applyFilters(query, filters)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortBy := "created_at"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}

	sortOrder := "DESC"
	if filters.SortOrder != "" {
		sortOrder = filters.SortOrder
	}

	query = query.Order(fmt.Sprintf("%s %s", common.SanitizeSortColumn(sortBy, "created_at"), common.SanitizeSortOrder(sortOrder, "DESC")))

	// Apply pagination
	if filters.PageSize > 0 {
		offset := (filters.Page - 1) * filters.PageSize
		query = query.Limit(filters.PageSize).Offset(offset)
	}

	err := query.Find(&lenders).Error
	return lenders, total, err
}

func (r *LenderRepository) HasActiveLoans(ctx context.Context, lenderID uint) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Loan{}).
		Where("lender_id = ? AND status = ?", lenderID, domain.LoanStatusActive).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// applyFilters applies the common filters to the query
func (r *LenderRepository) applyFilters(query *gorm.DB, filters domain.LenderFilters) *gorm.DB {
	// Search functionality
	if filters.Search != nil && *filters.Search != "" {
		searchTerm := "%" + *filters.Search + "%"
		query = query.Where("LOWER(name) LIKE LOWER(?)", searchTerm)
	}

	return query
}

// HasDisbursedLoans checks if any loan with non-null disbursed_at exists for lender
func (r *LenderRepository) HasDisbursedLoans(ctx context.Context, lenderID uint) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Loan{}).
		Where("lender_id = ? AND disbursed_at IS NOT NULL", lenderID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeleteUndisbursedLoansByLender soft-deletes loans that were never disbursed for the given lender
func (r *LenderRepository) DeleteUndisbursedLoansByLender(ctx context.Context, lenderID uint) error {
	return r.DB.WithContext(ctx).
		Where("lender_id = ? AND disbursed_at IS NULL", lenderID).
		Delete(&domain.Loan{}).Error
}
