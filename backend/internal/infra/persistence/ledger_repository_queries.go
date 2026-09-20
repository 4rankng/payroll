package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
	"api-server/internal/pkg/utils"

	"gorm.io/gorm"
)

func (r *LedgerEntryRepository) GetByID(ctx context.Context, id uint) (*domain.LedgerEntry, error) {
	var entry domain.LedgerEntry
	if err := r.SafeGetByID(ctx, &entry, id, "Creator", "Asset"); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *LedgerEntryRepository) GetByAssetID(ctx context.Context, assetID uint) ([]*domain.LedgerEntry, error) {
	var entries []*domain.LedgerEntry
	err := r.DB.WithContext(ctx).
		Preload("Creator").
		Preload("Asset").
		Where("asset_id = ?", assetID).
		Order("date DESC").
		Limit(common.DefaultMaxResults).
		Find(&entries).Error
	return entries, err
}

func (r *LedgerEntryRepository) GetByTransactionID(ctx context.Context, txnID uint) ([]*domain.LedgerEntry, error) {
	return r.getEntriesByField(ctx, "transaction_id", txnID)
}

func (r *LedgerEntryRepository) GetBySettlementID(ctx context.Context, settlementID uint) ([]*domain.LedgerEntry, error) {
	return r.getEntriesByField(ctx, "settlement_id", settlementID)
}

// ListReversalGroup returns the balanced block the entry belongs to.
//
// Why the fallback: entries written by the manual /ledger/entries endpoint get no
// transaction id, and they are inserted in one statement, so every leg of that
// block shares created_at to the millisecond and the same author. Grouping on
// those two columns recovers the block without inventing an identifier for rows
// that already exist. A single-entry block that is one-sided simply comes back as
// one row, and the reversal path refuses it because it does not balance.
func (r *LedgerEntryRepository) ListReversalGroup(ctx context.Context, entry *domain.LedgerEntry) ([]*domain.LedgerEntry, error) {
	if entry == nil {
		return nil, domain.NewValidationError("Bản ghi sổ cái không hợp lệ")
	}
	if entry.TransactionID != nil {
		return r.GetByTransactionID(ctx, *entry.TransactionID)
	}

	var entries []*domain.LedgerEntry
	err := r.DB.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("transaction_id IS NULL").
		Where("created_at = ?", entry.CreatedAt).
		Where("created_by = ?", entry.CreatedBy).
		Order("id ASC").
		Find(&entries).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load reversal group: %w", err)
	}
	return entries, nil
}

// HasReversal reports whether the entry already has a live mirror.
func (r *LedgerEntryRepository) HasReversal(ctx context.Context, entryID uint) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.LedgerEntry{}).
		Where("deleted_at IS NULL").
		Where("reversal_of_entry_id = ?", entryID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check existing reversal: %w", err)
	}
	return count > 0, nil
}

// getEntriesByField fetches ledger entries filtered by a single field value,
// with standard preloads and ascending created_at ordering.
func (r *LedgerEntryRepository) getEntriesByField(ctx context.Context, field string, value interface{}) ([]*domain.LedgerEntry, error) {
	var entries []*domain.LedgerEntry
	err := r.DB.WithContext(ctx).
		Preload("Creator").
		Preload("Asset").
		Preload("Settlement").
		Where(fmt.Sprintf("%s = ?", field), value).
		Order("created_at ASC").
		Find(&entries).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []*domain.LedgerEntry{}, nil
		}
		return nil, err
	}
	return entries, nil
}

func (r *LedgerEntryRepository) List(ctx context.Context, filters domain.LedgerFilters) ([]*domain.LedgerEntry, error) {
	var entries []*domain.LedgerEntry
	err := r.queryBuilder.BuildListQuery(filters).WithContext(ctx).Find(&entries).Error
	return entries, err
}

func (r *LedgerEntryRepository) Count(ctx context.Context, filters domain.LedgerFilters) (int64, error) {
	var count int64
	err := r.queryBuilder.BuildCountQuery(filters).WithContext(ctx).Count(&count).Error
	return count, err
}

func (r *LedgerEntryRepository) GetByAccount(ctx context.Context, account domain.LedgerAccount) ([]*domain.LedgerEntry, error) {
	var entries []*domain.LedgerEntry
	err := r.DB.WithContext(ctx).
		Preload("Creator").
		Preload("Asset").
		Where("account = ?", account).
		Order("date DESC").
		Limit(common.DefaultMaxResults).
		Find(&entries).Error
	return entries, err
}

func (r *LedgerEntryRepository) GetByDateRange(ctx context.Context, start, end time.Time) ([]*domain.LedgerEntry, error) {
	var entries []*domain.LedgerEntry
	err := r.DB.WithContext(ctx).
		Preload("Creator").
		Preload("Asset").
		Where("date BETWEEN ? AND ?", start, end).
		Order("date DESC").
		Limit(common.DefaultMaxResults).
		Find(&entries).Error
	return entries, err
}

// GetAccountEntriesByDateRange fetches only the fields needed for financial projections
// (account, date, debit, credit) without preloading relations or applying a row limit.
// Called via concrete type assertion in the dashboard service.
func (r *LedgerEntryRepository) GetAccountEntriesByDateRange(ctx context.Context, start, end time.Time) ([]*domain.LedgerEntry, error) {
	var entries []*domain.LedgerEntry
	err := r.DB.WithContext(ctx).
		Select("id, account, date, debit, credit").
		Where("date BETWEEN ? AND ? AND deleted_at IS NULL", start, end).
		Order("date ASC").
		Find(&entries).Error
	return entries, err
}

// CumulativeAccountTotals holds cumulative net amounts per account before a cutoff date.
type CumulativeAccountTotals struct {
	Revenue float64
	Expense float64
	Equity  float64
	Loan    float64
}

// GetCumulativeTotalsBeforeDate returns the net (credit - debit) total for each account
// for all entries before the given date. Used as a baseline for cumulative chart calculations.
func (r *LedgerEntryRepository) GetCumulativeTotalsBeforeDate(ctx context.Context, beforeDate time.Time) (*CumulativeAccountTotals, error) {
	var results []struct {
		Account     string `gorm:"column:account"`
		TotalDebit  int64  `gorm:"column:total_debit"`
		TotalCredit int64  `gorm:"column:total_credit"`
	}

	err := r.DB.WithContext(ctx).
		Model(&domain.LedgerEntry{}).
		Select("account, COALESCE(SUM(debit), 0) as total_debit, COALESCE(SUM(credit), 0) as total_credit").
		Where("date < ? AND deleted_at IS NULL AND account IN (?, ?, ?, ?)",
			beforeDate, domain.AccountRevenue, domain.AccountExpense, domain.AccountEquity, domain.AccountLoan).
		Group("account").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	totals := &CumulativeAccountTotals{}
	for _, r := range results {
		net := float64(r.TotalCredit - r.TotalDebit)
		switch r.Account {
		case domain.AccountRevenue:
			totals.Revenue = net
		case domain.AccountExpense:
			totals.Expense = net
		case domain.AccountEquity:
			totals.Equity = net
		case domain.AccountLoan:
			totals.Loan = net
		}
	}
	return totals, nil
}

func (r *LedgerEntryRepository) SearchLedgerEntries(ctx context.Context, search string, limit int) ([]*domain.LedgerEntry, error) {
	if search == "" {
		return nil, domain.NewValidationError(constants.MsgSearchQueryRequiredVN)
	}
	if len(strings.TrimSpace(search)) < 3 {
		return nil, domain.NewValidationError(constants.MsgSearchQueryMinLengthVN)
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var entries []*domain.LedgerEntry
	err := r.DB.WithContext(ctx).
		Preload("Creator").
		Preload("Asset").
		Where("search_normalized LIKE ?", utils.NormalizeVietnameseForSearch(search)).
		Order("date DESC, created_at DESC").
		Limit(limit).
		Find(&entries).Error
	return entries, err
}
