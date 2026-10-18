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

func (r *LenderRepository) HasAnyLoans(ctx context.Context, lenderID uint) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Loan{}).
		Where("lender_id = ?", lenderID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetLoanSummary aggregates live loans linked to the lender. Principal totals
// only count disbursed loans (a draft loan has borrowed nothing yet); the
// repaid figure is derived as borrowed minus outstanding.
func (r *LenderRepository) GetLoanSummary(ctx context.Context, lenderID uint) (domain.LenderLoanSummary, error) {
	var row struct {
		DisbursedPrincipal int64
		OutstandingAmount  int64
		ActiveCount        int64
		TotalCount         int64
	}
	err := r.DB.WithContext(ctx).
		Model(&domain.Loan{}).
		Select(`COALESCE(SUM(CASE WHEN disbursed_at IS NOT NULL THEN principal_amount ELSE 0 END), 0) AS disbursed_principal,
			COALESCE(SUM(CASE WHEN disbursed_at IS NOT NULL THEN outstanding_principal ELSE 0 END), 0) AS outstanding_amount,
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS active_count,
			COUNT(*) AS total_count`, domain.LoanStatusActive).
		Where("lender_id = ?", lenderID).
		Scan(&row).Error
	if err != nil {
		return domain.LenderLoanSummary{}, err
	}
	return domain.LenderLoanSummary{
		TotalPrincipalBorrowed: row.DisbursedPrincipal,
		TotalPrincipalRepaid:   row.DisbursedPrincipal - row.OutstandingAmount,
		OutstandingPrincipal:   row.OutstandingAmount,
		ActiveLoansCount:       int(row.ActiveCount),
		TotalLoansCount:        int(row.TotalCount),
	}, nil
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
