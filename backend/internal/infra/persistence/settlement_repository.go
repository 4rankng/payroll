package persistence

import (
	"context"
	"fmt"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

type settlementRepository struct {
	db *gorm.DB
}

// NewSettlementRepository creates a new settlement repository
func NewSettlementRepository(db *gorm.DB) domain.SettlementRepository {
	return &settlementRepository{db: db}
}

// getDB gets the appropriate DB instance from context or falls back to default
func (r *settlementRepository) getDB(ctx context.Context) *gorm.DB {
	// Check if there's a transaction context
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *settlementRepository) Create(ctx context.Context, settlement *domain.Settlement) error {
	if err := r.getDB(ctx).Create(settlement).Error; err != nil {
		return fmt.Errorf("failed to create settlement: %w", err)
	}
	return nil
}

func (r *settlementRepository) GetByID(ctx context.Context, id uint) (*domain.Settlement, error) {
	var settlement domain.Settlement
	err := r.getDB(ctx).
		// Coerce DECIMAL amounts to integer VND in query results
		Select("id, transaction_id, ROUND(amount) as amount, settlement_date, proof_url, proof_asset_id, payment_method, notes, settlement_uuid, created_by, created_at, updated_at, deleted_at").
		Preload("Transaction").
		Preload("ProofAsset").
		Preload("Creator").
		First(&settlement, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("settlement not found")
		}
		return nil, fmt.Errorf("failed to get settlement: %w", err)
	}

	return &settlement, nil
}

func (r *settlementRepository) GetByTransactionID(ctx context.Context, transactionID uint) ([]*domain.Settlement, error) {
	var settlements []*domain.Settlement
	err := r.getDB(ctx).
		// Coerce DECIMAL amounts to integer VND in query results
		Select("id, transaction_id, ROUND(amount) as amount, settlement_date, proof_url, proof_asset_id, payment_method, notes, settlement_uuid, created_by, created_at, updated_at, deleted_at").
		Where("transaction_id = ?", transactionID).
		Preload("ProofAsset").
		Preload("Creator").
		Order("settlement_date DESC, created_at DESC").
		Find(&settlements).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get settlements by transaction ID: %w", err)
	}

	return settlements, nil
}

func (r *settlementRepository) GetTotalSettledAmount(ctx context.Context, transactionID uint) (int64, error) {
	var total int64
	err := r.getDB(ctx).
		Model(&domain.Settlement{}).
		Where("transaction_id = ? AND deleted_at IS NULL", transactionID).
		// Round SUM(amount) to integer VND before scanning into int64
		Select("COALESCE(ROUND(SUM(amount)), 0)").
		Scan(&total).Error

	if err != nil {
		return 0, fmt.Errorf("failed to get total settled amount: %w", err)
	}

	return total, nil
}

func (r *settlementRepository) List(ctx context.Context, filters domain.SettlementFilters) ([]*domain.Settlement, error) {
	var settlements []*domain.Settlement

	query := r.getDB(ctx).Model(&domain.Settlement{})
	// Coerce DECIMAL amounts to integer VND in query results
	query = query.Select("id, transaction_id, ROUND(amount) as amount, settlement_date, proof_url, proof_asset_id, payment_method, notes, settlement_uuid, created_by, created_at, updated_at, deleted_at")

	// Apply filters
	if filters.TransactionID != nil {
		query = query.Where("transaction_id = ?", *filters.TransactionID)
	}

	if filters.PaymentMethod != nil && *filters.PaymentMethod != "" {
		query = query.Where("payment_method = ?", *filters.PaymentMethod)
	}

	if filters.FromDate != nil {
		query = query.Where("settlement_date >= ?", *filters.FromDate)
	}

	if filters.ToDate != nil {
		query = query.Where("settlement_date <= ?", *filters.ToDate)
	}

	if filters.CreatedBy != nil {
		query = query.Where("created_by = ?", *filters.CreatedBy)
	}

	// Apply sorting
	sortBy := filters.SortBy
	if sortBy == "" {
		sortBy = "settlement_date"
	}

	sortOrder := filters.SortOrder
	if sortOrder == "" {
		sortOrder = "DESC"
	}

	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	// Preload relationships
	query = query.Preload("Transaction").
		Preload("ProofAsset").
		Preload("Creator")

	err := query.Find(&settlements).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list settlements: %w", err)
	}

	return settlements, nil
}

func (r *settlementRepository) Count(ctx context.Context, filters domain.SettlementFilters) (int64, error) {
	var count int64

	query := r.getDB(ctx).Model(&domain.Settlement{})

	// Apply same filters as List
	if filters.TransactionID != nil {
		query = query.Where("transaction_id = ?", *filters.TransactionID)
	}

	if filters.PaymentMethod != nil && *filters.PaymentMethod != "" {
		query = query.Where("payment_method = ?", *filters.PaymentMethod)
	}

	if filters.FromDate != nil {
		query = query.Where("settlement_date >= ?", *filters.FromDate)
	}

	if filters.ToDate != nil {
		query = query.Where("settlement_date <= ?", *filters.ToDate)
	}

	if filters.CreatedBy != nil {
		query = query.Where("created_by = ?", *filters.CreatedBy)
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count settlements: %w", err)
	}

	return count, nil
}
