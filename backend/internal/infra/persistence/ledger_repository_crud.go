package persistence

import (
	"api-server/internal/pkg/clock"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

func (r *LedgerEntryRepository) Create(ctx context.Context, entry *domain.LedgerEntry) error {
	r.balanceMutex.Lock()
	defer r.balanceMutex.Unlock()

	return r.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		runningBalance, err := r.calculateRunningBalanceAtID(tx, entry.ID)
		if err != nil {
			return fmt.Errorf("failed to calculate running balance: %w", err)
		}
		entry.Balance = runningBalance + domain.ComputeBalanceDelta(entry.Account, entry.Debit, entry.Credit)
		if err := tx.Create(entry).Error; err != nil {
			return fmt.Errorf("failed to create ledger entry: %w", err)
		}
		return r.updateSubsequentBalances(tx, entry.ID)
	})
}

// CreateTransaction creates multiple ledger entries within a single transaction.
// Reuses an existing DB transaction from context if present.
func (r *LedgerEntryRepository) CreateTransaction(ctx context.Context, entries []*domain.LedgerEntry) error {
	r.balanceMutex.Lock()
	defer r.balanceMutex.Unlock()

	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return r.createEntriesInTx(ctx, txCtx.TX, entries)
	}
	return r.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		return r.createEntriesInTx(ctx, tx, entries)
	})
}

// createEntriesInTx creates ledger entries within an existing transaction.
func (r *LedgerEntryRepository) createEntriesInTx(ctx context.Context, tx *gorm.DB, entries []*domain.LedgerEntry) error {
	if len(entries) == 0 {
		return nil
	}

	r.queryBuilder.SortEntriesByID(entries)

	startingBalance, err := r.calculateRunningBalanceAtID(tx, entries[0].ID)
	if err != nil {
		return fmt.Errorf("failed to calculate starting balance: %w", err)
	}

	domain.CalculateRunningBalances(startingBalance, entries)

	const batchSize = 100
	if len(entries) <= batchSize {
		if err := tx.Create(entries).Error; err != nil {
			return fmt.Errorf("failed to create ledger entries: %w", err)
		}
	} else {
		if err := tx.CreateInBatches(entries, batchSize).Error; err != nil {
			return fmt.Errorf("failed to create ledger entries in batches: %w", err)
		}
	}
	return r.updateSubsequentBalances(tx, entries[0].ID)
}

// DeleteByTransactionID soft-deletes all ledger entries linked to a transaction,
// then recalculates balances to maintain consistency.
func (r *LedgerEntryRepository) DeleteByTransactionID(ctx context.Context, txnID uint) error {
	if err := r.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		var ids []uint
		if err := tx.Model(&domain.LedgerEntry{}).
			Where("transaction_id = ? AND deleted_at IS NULL", txnID).
			Order("id ASC").
			Pluck("id", &ids).Error; err != nil {
			return fmt.Errorf("failed to find ledger entries by transaction_id: %w", err)
		}
		if len(ids) == 0 {
			return nil
		}

		base := clock.NowUTC().Truncate(time.Millisecond)
		caseClauses := make([]string, len(ids))
		caseArgs := make([]interface{}, 0, len(ids))
		for i, id := range ids {
			ts := base.Add(time.Duration(i) * time.Millisecond)
			caseClauses[i] = fmt.Sprintf("WHEN %d THEN ?", id)
			caseArgs = append(caseArgs, ts)
		}
		caseExpr := "CASE id " + strings.Join(caseClauses, " ") + " END"
		if err := tx.Model(&domain.LedgerEntry{}).
			Where("id IN ? AND deleted_at IS NULL", ids).
			Update("deleted_at", gorm.Expr(caseExpr, caseArgs...)).Error; err != nil {
			return fmt.Errorf("failed to batch soft-delete ledger entries: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := r.RecalculateAllBalances(ctx); err != nil {
		return fmt.Errorf("failed to recalculate balances after delete: %w", err)
	}
	return nil
}

// CheckDuplicate checks for a duplicate ledger entry by asset_id first, then by field tuple.
func (r *LedgerEntryRepository) CheckDuplicate(ctx context.Context, entry *domain.LedgerEntry) (*domain.LedgerEntry, error) {
	if entry.AssetID != nil {
		var existing domain.LedgerEntry
		err := r.DB.WithContext(ctx).Where("asset_id = ?", *entry.AssetID).First(&existing).Error
		if err == nil {
			return &existing, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to check asset_id duplicate: %w", err)
		}
	}

	var existing domain.LedgerEntry
	err := r.DB.WithContext(ctx).
		Where("date = ? AND account = ? AND party = ? AND debit = ? AND credit = ?",
			entry.Date, entry.Account, entry.Party, entry.Debit, entry.Credit).
		First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.NewNotFoundError("no duplicate entry found")
	}
	return nil, fmt.Errorf("failed to check duplicate: %w", err)
}

// BatchCheckDuplicates checks for duplicate ledger entries for a batch in a single query.
// Returns a map of input index → existing duplicate entry.
func (r *LedgerEntryRepository) BatchCheckDuplicates(ctx context.Context, entries []*domain.LedgerEntry) (map[int]*domain.LedgerEntry, error) {
	if len(entries) == 0 {
		return nil, nil
	}

	result := make(map[int]*domain.LedgerEntry)

	// --- Pass 1: check by asset_id ---
	type assetCheck struct {
		idx     int
		assetID uint
	}
	var assetChecks []assetCheck
	var assetIDs []uint
	for i, e := range entries {
		if e.AssetID != nil {
			assetChecks = append(assetChecks, assetCheck{i, *e.AssetID})
			assetIDs = append(assetIDs, *e.AssetID)
		}
	}
	if len(assetIDs) > 0 {
		var found []domain.LedgerEntry
		if err := r.DB.WithContext(ctx).Where("asset_id IN ?", assetIDs).Find(&found).Error; err != nil {
			return nil, fmt.Errorf("failed to batch check asset_id duplicates: %w", err)
		}
		byAsset := make(map[uint]*domain.LedgerEntry, len(found))
		for i := range found {
			byAsset[*found[i].AssetID] = &found[i]
		}
		for _, ac := range assetChecks {
			if dup, ok := byAsset[ac.assetID]; ok {
				result[ac.idx] = dup
			}
		}
	}

	// --- Pass 2: check by field tuple for unmatched entries without asset_id ---
	type fieldKey struct {
		Date          string
		Account       string
		Party         string
		Debit         int64
		Credit        int64
		TransactionID uint
	}
	type fieldCheck struct {
		idx int
		key fieldKey
	}
	var fieldChecks []fieldCheck
	for i, e := range entries {
		if _, matched := result[i]; matched {
			continue
		}
		if e.AssetID != nil {
			continue
		}
		fieldChecks = append(fieldChecks, fieldCheck{
			idx: i,
			key: fieldKey{
				Date:    e.Date.Format("2006-01-02"),
				Account: e.Account,
				Party:   e.Party,
				Debit:   e.Debit,
				Credit:  e.Credit,
				TransactionID: func() uint {
					if e.TransactionID == nil {
						return 0
					}
					return *e.TransactionID
				}(),
			},
		})
	}
	if len(fieldChecks) > 0 {
		query := r.DB.WithContext(ctx).Model(&domain.LedgerEntry{})
		for _, fc := range fieldChecks {
			if fc.key.TransactionID > 0 {
				query = query.Or("date = ? AND account = ? AND party = ? AND debit = ? AND credit = ? AND transaction_id = ?",
					fc.key.Date, fc.key.Account, fc.key.Party, fc.key.Debit, fc.key.Credit, fc.key.TransactionID)
			} else {
				query = query.Or("date = ? AND account = ? AND party = ? AND debit = ? AND credit = ?",
					fc.key.Date, fc.key.Account, fc.key.Party, fc.key.Debit, fc.key.Credit)
			}
		}
		var found []domain.LedgerEntry
		if err := query.Find(&found).Error; err != nil {
			return nil, fmt.Errorf("failed to batch check field duplicates: %w", err)
		}
		byField := make(map[fieldKey]*domain.LedgerEntry, len(found))
		for i := range found {
			k := fieldKey{
				Date:    found[i].Date.Format("2006-01-02"),
				Account: found[i].Account,
				Party:   found[i].Party,
				Debit:   found[i].Debit,
				Credit:  found[i].Credit,
			}
			if found[i].TransactionID != nil {
				k.TransactionID = *found[i].TransactionID
			}
			byField[k] = &found[i]
			// Preserve the legacy field-only duplicate behavior for callers that
			// do not associate entries with a transaction. Transaction-linked
			// callers use the exact key above, so equal payroll amounts in two
			// different transactions are not false positives.
			k.TransactionID = 0
			byField[k] = &found[i]
		}
		for _, fc := range fieldChecks {
			if dup, ok := byField[fc.key]; ok {
				result[fc.idx] = dup
			}
		}
	}

	return result, nil
}
