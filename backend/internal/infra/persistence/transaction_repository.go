package persistence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const allFields = "id, description, transaction_type, ROUND(amount) as amount, party, status, ROUND(settled_amount) as settled_amount, url, asset_id, user_id, loan_id, reversed_transaction_id, created_by, created_at, updated_at, deleted_at, transaction_code"

type transactionRepository struct {
	db *gorm.DB
}

// NewTransactionRepository creates a new transaction repository
func NewTransactionRepository(db *gorm.DB) domain.TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) Create(ctx context.Context, txn *domain.Transaction) error {
	txn.PrepareForCreate()
	// Use getDB to respect existing transaction context
	db := r.getDB(ctx)
	if err := db.Create(txn).Error; err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	return nil
}

func (r *transactionRepository) GetByID(ctx context.Context, id uint) (*domain.Transaction, error) {
	var txn domain.Transaction
	err := r.db.WithContext(ctx).
		// Coerce DECIMAL amounts to integer VND to avoid scan errors when DB schema hasn't migrated
		Select(allFields).
		Preload("Asset").
		Preload("Creator").
		Preload("LedgerEntries").
		Preload("Settlements").
		First(&txn, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Calculate settled_amount from settlements (single source of truth)
	r.calculateSettledAmount(&txn)

	return &txn, nil
}

func (r *transactionRepository) GetByIDForUpdate(ctx context.Context, id uint) (*domain.Transaction, error) {
	db := r.getDB(ctx)

	var txn domain.Transaction
	err := db.WithContext(ctx).
		Select(allFields).
		Preload("Asset").
		Preload("Creator").
		Preload("LedgerEntries").
		Preload("Settlements").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&txn, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction for update: %w", err)
	}

	r.calculateSettledAmount(&txn)

	return &txn, nil
}

func (r *transactionRepository) GetByAssetID(ctx context.Context, assetID uint) (*domain.Transaction, error) {
	var txn domain.Transaction

	err := r.db.WithContext(ctx).
		Select(allFields).
		Preload("Asset").
		Preload("Creator").
		Preload("LedgerEntries").
		Preload("Settlements").
		Where("asset_id = ?", assetID).
		First(&txn).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	r.calculateSettledAmount(&txn)

	return &txn, nil
}

func (r *transactionRepository) GetByIDs(ctx context.Context, ids []uint) ([]*domain.Transaction, error) {
	if len(ids) == 0 {
		return []*domain.Transaction{}, nil
	}

	var transactions []*domain.Transaction
	err := r.db.WithContext(ctx).
		Select(allFields).
		Preload("Asset").
		Preload("Creator").
		Preload("LedgerEntries").
		Preload("Settlements").
		Where("id IN ?", ids).
		Find(&transactions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	// Calculate settled_amount from settlements for each transaction
	for i := range transactions {
		r.calculateSettledAmount(transactions[i])
	}

	return transactions, nil
}

func (r *transactionRepository) List(ctx context.Context, filters domain.TransactionFilters) ([]*domain.Transaction, error) {
	var transactions []*domain.Transaction

	// Build the CTE query for proper pagination with filtering
	query := r.buildCTEQuery(ctx, filters)

	if err := query.Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	// Calculate settled_amount for all transactions from settlements (single source of truth)
	for _, txn := range transactions {
		r.calculateSettledAmount(txn)
	}

	return transactions, nil
}

func (r *transactionRepository) Count(ctx context.Context, filters domain.TransactionFilters) (int64, error) {
	var count int64

	// Build WHERE conditions (same as buildCTEQuery)
	var whereConditions []string
	var args []any

	// Add filters
	if filters.TransactionType != nil {
		whereConditions = append(whereConditions, "transaction_type = ?")
		args = append(args, *filters.TransactionType)
	}

	if filters.Status != nil {
		whereConditions = append(whereConditions, "status = ?")
		args = append(args, *filters.Status)
	}

	if filters.Party != nil {
		whereConditions = append(whereConditions, "party = ?")
		args = append(args, *filters.Party)
	}

	if filters.Search != "" {
		whereConditions = append(whereConditions, "(LOWER(description) LIKE ? OR LOWER(party) LIKE ?)")
		search := "%" + strings.ToLower(filters.Search) + "%"
		args = append(args, search, search)
	}

	if filters.CreatedBy != nil {
		whereConditions = append(whereConditions, "created_by = ?")
		args = append(args, *filters.CreatedBy)
	}

	if filters.FromDate != nil {
		whereConditions = append(whereConditions, "created_at >= ?")
		args = append(args, *filters.FromDate)
	}

	if filters.ToDate != nil {
		endOfDay := filters.ToDate.AddDate(0, 0, 1)
		whereConditions = append(whereConditions, "created_at < ?")
		args = append(args, endOfDay)
	}

	// Build WHERE clause
	whereClause := "WHERE deleted_at IS NULL"
	if len(whereConditions) > 0 {
		whereClause += " AND " + strings.Join(whereConditions, " AND ")
	}

	// Build count CTE query. NOT EXISTS (correlated) instead of NOT IN — lets
	// MySQL use idx_transactions_reversed_txn_id (migration 088) and avoids
	// materializing the inner subquery. ck:debug 2026-07-04.
	countQuery := fmt.Sprintf(`
		WITH filtered_transactions AS (
			SELECT id FROM transactions
			%s AND reversed_transaction_id IS NULL
			AND NOT EXISTS (
				SELECT 1 FROM transactions r
				WHERE r.reversed_transaction_id = transactions.id
			)
		)
		SELECT COUNT(*) FROM filtered_transactions`, whereClause)

	if err := r.db.WithContext(ctx).Raw(countQuery, args...).Scan(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	return count, nil
}

func (r *transactionRepository) Update(ctx context.Context, txn *domain.Transaction) error {
	// Use getDB to respect existing transaction context
	db := r.getDB(ctx)
	if err := db.Save(txn).Error; err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}
	return nil
}

func (r *transactionRepository) GetByParty(ctx context.Context, party string) ([]*domain.Transaction, error) {
	var transactions []*domain.Transaction

	err := r.db.WithContext(ctx).
		// Coerce DECIMAL amounts to integer VND in query results
		Select(allFields).
		Where("party = ?", party).
		Preload("Asset").
		Preload("Creator").
		Preload("Settlements").
		Order("created_at DESC").
		Find(&transactions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by party: %w", err)
	}

	// Calculate settled_amount for all transactions from settlements (single source of truth)
	for _, txn := range transactions {
		r.calculateSettledAmount(txn)
	}

	return transactions, nil
}

// GetPendingRevenueByParty retrieves all pending revenue transactions for a party
func (r *transactionRepository) GetPendingRevenueByParty(ctx context.Context, party string) ([]*domain.Transaction, error) {
	var transactions []*domain.Transaction

	err := r.db.WithContext(ctx).
		Select(allFields).
		Where("party = ?", party).
		Where("status = ?", domain.TransactionStatusPending).
		Where("transaction_type = ?", domain.TransactionTypeRevenue).
		Preload("Asset").
		Preload("Creator").
		Preload("Settlements").
		Order("created_at DESC").
		Find(&transactions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get pending revenue transactions by party: %w", err)
	}

	// Calculate settled_amount for all transactions from settlements (single source of truth)
	for _, txn := range transactions {
		r.calculateSettledAmount(txn)
	}

	return transactions, nil
}

// GetSettledTransactionsByLoan retrieves all settled transactions for a loan
func (r *transactionRepository) GetSettledTransactionsByLoan(ctx context.Context, loanID uint) ([]*domain.Transaction, error) {
	var transactions []*domain.Transaction

	err := r.db.WithContext(ctx).
		Select(allFields).
		Where("loan_id = ?", loanID).
		Where("status = ?", domain.TransactionStatusSettled).
		Preload("Settlements").
		Order("created_at ASC").
		Find(&transactions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get settled transactions by loan: %w", err)
	}

	// Calculate settled_amount for all transactions from settlements (single source of truth)
	for _, txn := range transactions {
		r.calculateSettledAmount(txn)
	}

	return transactions, nil
}

// GetWithRoundingImbalance retrieves transactions with rounding imbalances
func (r *transactionRepository) GetWithRoundingImbalance(ctx context.Context, tolerance int64, since time.Time) ([]*domain.Transaction, error) {
	var transactions []*domain.Transaction

	err := r.db.WithContext(ctx).
		Select(allFields).
		Where("updated_at >= ?", since).
		Where("ABS(amount - settled_amount) > 0").
		Where("ABS(amount - settled_amount) <= ?", tolerance).
		Order("updated_at DESC").
		Find(&transactions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get transactions with rounding imbalance: %w", err)
	}

	return transactions, nil
}

// buildCTEQuery builds the CTE query for proper pagination with filtering
func (r *transactionRepository) buildCTEQuery(ctx context.Context, filters domain.TransactionFilters) *gorm.DB {
	// Build WHERE conditions
	var whereConditions []string
	var args []any

	// Add filters
	if filters.TransactionType != nil {
		whereConditions = append(whereConditions, "transaction_type = ?")
		args = append(args, *filters.TransactionType)
	}

	if filters.Status != nil {
		whereConditions = append(whereConditions, "status = ?")
		args = append(args, *filters.Status)
	}

	if filters.Party != nil {
		whereConditions = append(whereConditions, "party = ?")
		args = append(args, *filters.Party)
	}

	if filters.Search != "" {
		whereConditions = append(whereConditions, "(LOWER(description) LIKE ? OR LOWER(party) LIKE ?)")
		search := "%" + strings.ToLower(filters.Search) + "%"
		args = append(args, search, search)
	}

	if filters.CreatedBy != nil {
		whereConditions = append(whereConditions, "created_by = ?")
		args = append(args, *filters.CreatedBy)
	}

	if filters.FromDate != nil {
		whereConditions = append(whereConditions, "created_at >= ?")
		args = append(args, *filters.FromDate)
	}

	if filters.ToDate != nil {
		endOfDay := filters.ToDate.AddDate(0, 0, 1)
		whereConditions = append(whereConditions, "created_at < ?")
		args = append(args, endOfDay)
	}

	// Build WHERE clause
	whereClause := "WHERE deleted_at IS NULL"
	if len(whereConditions) > 0 {
		whereClause += " AND " + strings.Join(whereConditions, " AND ")
	}

	// Build CTE query. NOT EXISTS (correlated) instead of NOT IN — uses
	// idx_transactions_reversed_txn_id. ck:debug 2026-07-04.
	cteQuery := fmt.Sprintf(`
		WITH filtered_transactions AS (
			SELECT %s FROM transactions
			%s AND reversed_transaction_id IS NULL
			AND NOT EXISTS (
				SELECT 1 FROM transactions r
				WHERE r.reversed_transaction_id = transactions.id
			)
		)
		SELECT * FROM filtered_transactions`, allFields, whereClause)

	// Apply sorting
	sortBy := filters.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}

	sortOrder := filters.SortOrder
	if sortOrder == "" {
		sortOrder = "DESC"
	}

	cteQuery += fmt.Sprintf(" ORDER BY %s %s", common.SanitizeSortColumn(sortBy, "created_at"), common.SanitizeSortOrder(sortOrder, "DESC"))

	// Apply pagination
	if filters.Limit > 0 {
		cteQuery += fmt.Sprintf(" LIMIT %d", filters.Limit)
	}
	if filters.Offset > 0 {
		cteQuery += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	// Execute the raw CTE query with preloads
	query := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Settlements").
		Raw(cteQuery, args...)

	return query
}

// Delete soft-deletes a transaction and all its associated ledger entries atomically.
// Without the cascade, settlement debit entries survive while the creation credit entry
// is never created (outbox worker gets "not found" on the deleted transaction), leaving
// the payable account with unmatched debits and a negative net balance.
func (r *transactionRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Collect settlement IDs before any deletion
		var settlementIDs []uint
		if err := tx.Model(&domain.Settlement{}).
			Where("transaction_id = ?", id).
			Pluck("id", &settlementIDs).Error; err != nil {
			return fmt.Errorf("failed to fetch settlement IDs: %w", err)
		}

		// Soft-delete ledger entries created directly by this transaction
		if err := tx.Where("transaction_id = ? AND settlement_id IS NULL", id).
			Delete(&domain.LedgerEntry{}).Error; err != nil {
			return fmt.Errorf("failed to delete transaction ledger entries: %w", err)
		}

		// Soft-delete ledger entries created by each settlement
		if len(settlementIDs) > 0 {
			if err := tx.Where("settlement_id IN ?", settlementIDs).
				Delete(&domain.LedgerEntry{}).Error; err != nil {
				return fmt.Errorf("failed to delete settlement ledger entries: %w", err)
			}
		}

		// Soft-delete the transaction itself
		if err := tx.Delete(&domain.Transaction{}, id).Error; err != nil {
			return fmt.Errorf("failed to delete transaction: %w", err)
		}

		return nil
	})
}

// GetPendingByDescription retrieves a pending transaction by exact description match
func (r *transactionRepository) GetPendingByDescription(ctx context.Context, description string) (*domain.Transaction, error) {
	var txn domain.Transaction

	err := r.db.WithContext(ctx).
		Select(allFields).
		Where("description = ?", description).
		Where("status = ?", domain.TransactionStatusPending).
		Preload("Asset").
		Preload("Creator").
		Preload("LedgerEntries").
		Preload("Settlements").
		First(&txn).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("pending transaction not found")
		}
		return nil, fmt.Errorf("failed to get pending transaction by description: %w", err)
	}

	r.calculateSettledAmount(&txn)

	return &txn, nil
}

// calculateSettledAmount calculates the settled amount from settlements (single source of truth)
func (r *transactionRepository) calculateSettledAmount(txn *domain.Transaction) {
	var total int64
	for _, settlement := range txn.Settlements {
		total += settlement.Amount
	}
	txn.SettledAmount = total
	txn.NormalizeSettledAmount(len(txn.Settlements) > 0)
}

// getDB gets the appropriate DB instance from context or falls back to default
func (r *transactionRepository) getDB(ctx context.Context) *gorm.DB {
	// Check if there's a transaction context
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

// IncrementAmount atomically adds delta to a transaction's amount. Honors tx
// context so it commits/rolls back with the surrounding settlement transaction.
func (r *transactionRepository) IncrementAmount(ctx context.Context, id uint, delta int64) error {
	result := r.getDB(ctx).
		Model(&domain.Transaction{}).
		Where("id = ?", id).
		UpdateColumn("amount", gorm.Expr("amount + ?", delta))
	if result.Error != nil {
		return fmt.Errorf("failed to increment transaction amount: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.NewNotFoundError(fmt.Sprintf("transaction %d not found", id))
	}
	return nil
}
