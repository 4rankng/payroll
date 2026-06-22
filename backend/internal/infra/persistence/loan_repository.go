package persistence

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"strings"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LoanRepository struct {
	*BaseRepository
}

func NewLoanRepository(db *Database) domain.LoanRepository {
	return &LoanRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *LoanRepository) Create(ctx context.Context, loan *domain.Loan) error {
	return r.db(ctx).Create(loan).Error
}

func (r *LoanRepository) GetByID(ctx context.Context, id uint) (*domain.Loan, error) {
	var loan domain.Loan
	err := r.DB.WithContext(ctx).
		Preload("Lender").
		Preload("Creator").
		First(&loan, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("loan not found")
		}
		return nil, err
	}

	return &loan, nil
}

func (r *LoanRepository) Update(ctx context.Context, loan *domain.Loan) error {
	updates := map[string]interface{}{
		"disbursed_at":          loan.DisbursedAt,
		"outstanding_principal": loan.OutstandingPrincipal,
		"status":                loan.Status,
		"total_interest_paid":   loan.TotalInterestPaid,
		"payment_day_of_month":  loan.PaymentDayOfMonth,
		"description":           loan.Description,
		"updated_at":            clock.Now(),
	}

	return r.db(ctx).
		Model(&domain.Loan{}).
		Where("id = ?", loan.ID).
		Updates(updates).Error
}

func (r *LoanRepository) List(ctx context.Context, filters domain.LoanFilters) ([]*domain.Loan, int64, error) {
	var loans []*domain.Loan
	var total int64

	query := r.DB.WithContext(ctx).Model(&domain.Loan{})

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

	// Preload relationships
	query = query.Preload("Lender").Preload("Creator")

	err := query.Find(&loans).Error
	return loans, total, err
}

func (r *LoanRepository) GetNextSequenceForYear(ctx context.Context, year int) (int, error) {
	var loan domain.Loan
	pattern := fmt.Sprintf("LOAN-%d-%%", year)

	// Use transaction-aware DB connection so SELECT FOR UPDATE works within the caller's transaction
	err := r.db(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("loan_code LIKE ?", pattern).
		Order("loan_code DESC").
		First(&loan).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 1, nil // First loan of the year
		}
		return 0, err
	}

	// Extract sequence number from loan code (LOAN-YYYY-NNN)
	parts := strings.Split(loan.LoanCode, "-")
	if len(parts) != 3 {
		return 1, nil
	}

	var sequence int
	_, err = fmt.Sscanf(parts[2], "%d", &sequence)
	if err != nil {
		return 1, nil
	}

	return sequence + 1, nil
}

func (r *LoanRepository) GetActiveLoans(ctx context.Context) ([]*domain.Loan, error) {
	var loans []*domain.Loan
	err := r.DB.WithContext(ctx).
		Where("status = ?", domain.LoanStatusActive).
		Preload("Lender").
		Preload("Creator").
		Order("created_at DESC").
		Find(&loans).Error

	return loans, err
}

func (r *LoanRepository) Delete(ctx context.Context, id uint) error {
	return r.db(ctx).Delete(&domain.Loan{}, id).Error
}

// applyFilters applies the common filters to the query
func (r *LoanRepository) applyFilters(query *gorm.DB, filters domain.LoanFilters) *gorm.DB {
	// Filter by lender
	if filters.LenderID != nil {
		query = query.Where("lender_id = ?", *filters.LenderID)
	}

	// Filter by status
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	return query
}

// db returns the appropriate DB instance for the context
func (r *LoanRepository) db(ctx context.Context) *gorm.DB {
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.DB.WithContext(ctx)
}
