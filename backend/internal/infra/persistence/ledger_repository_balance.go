package persistence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

func (r *LedgerEntryRepository) GetBalance(ctx context.Context) (int64, error) {
	var rows []accountAgg
	err := r.DB.WithContext(ctx).
		Model(&domain.LedgerEntry{}).
		Select("account, CAST(COALESCE(SUM(debit), 0) AS SIGNED) as total_debit, CAST(COALESCE(SUM(credit), 0) AS SIGNED) as total_credit").
		Where("deleted_at IS NULL").
		Group("account").
		Scan(&rows).Error
	if err != nil {
		return 0, err
	}

	return calculateNetWorthFromAccountTotals(rows), nil
}

// calculateNetWorthFromAccountTotals computes assets less liabilities from
// account-level debit and credit totals.
func calculateNetWorthFromAccountTotals(rows []accountAgg) int64 {
	var total int64
	for _, acc := range rows {
		switch acc.Account {
		case domain.AccountCash, domain.AccountReceivable:
			total += acc.TotalDebit - acc.TotalCredit
		case domain.AccountPayable, domain.AccountLoan:
			total -= acc.TotalCredit - acc.TotalDebit
		}
	}
	return total
}

func (r *LedgerEntryRepository) GetBalanceByAccount(ctx context.Context, account domain.LedgerAccount) (int64, error) {
	type result struct {
		Balance int64 `gorm:"column:balance"`
	}
	var res result
	err := r.DB.WithContext(ctx).
		Model(&domain.LedgerEntry{}).
		Select("CAST(COALESCE(SUM(debit - credit), 0) AS SIGNED) as balance").
		Where("account = ?", account).
		Scan(&res).Error
	return res.Balance, err
}

// RecalculateAllBalances recalculates the balance field for every ledger entry in sequence.
// This is the canonical implementation — all other balance logic must produce the same results.
func (r *LedgerEntryRepository) RecalculateAllBalances(ctx context.Context) error {
	return r.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		var entries []domain.LedgerEntry
		if err := tx.Order("date ASC, id ASC").Find(&entries).Error; err != nil {
			return fmt.Errorf("failed to fetch all entries: %w", err)
		}
		if len(entries) == 0 {
			return nil
		}
		return r.batchUpdateBalances(tx, entries)
	})
}

// updateSubsequentBalances recalculates balances for all entries after fromID.
func (r *LedgerEntryRepository) updateSubsequentBalances(tx *gorm.DB, fromID uint) error {
	runningBalance, err := r.calculateRunningBalanceAtID(tx, fromID+1)
	if err != nil {
		return fmt.Errorf("failed to calculate starting balance: %w", err)
	}

	var entries []domain.LedgerEntry
	if err := tx.Where("id > ?", fromID).Order("date ASC, id ASC").Find(&entries).Error; err != nil {
		return fmt.Errorf("failed to fetch subsequent entries: %w", err)
	}
	if len(entries) == 0 {
		return nil
	}

	const batchSize = 500
	caseClauses := make([]string, 0, batchSize)
	ids := make([]uint, 0, batchSize)

	for i := range entries {
		runningBalance = domain.CalculateNextBalance(runningBalance, &entries[i])
		caseClauses = append(caseClauses, fmt.Sprintf("WHEN %d THEN %d", entries[i].ID, runningBalance))
		ids = append(ids, entries[i].ID)

		if len(caseClauses) >= batchSize || i == len(entries)-1 {
			caseExpr := "CASE id " + strings.Join(caseClauses, " ") + " END"
			if err := tx.Model(&domain.LedgerEntry{}).
				Where("id IN ?", ids).
				Update("balance", gorm.Expr(caseExpr)).Error; err != nil {
				return fmt.Errorf("failed to batch update subsequent balances: %w", err)
			}
			caseClauses = caseClauses[:0]
			ids = ids[:0]
		}
	}
	return nil
}

// calculateRunningBalanceAtID returns the cumulative balance for all entries with id < entryID.
func (r *LedgerEntryRepository) calculateRunningBalanceAtID(tx *gorm.DB, entryID uint) (int64, error) {
	var balance int64
	err := tx.Model(&domain.LedgerEntry{}).
		Select(`COALESCE(SUM(
			CASE
				WHEN account IN ('cash', 'receivable') THEN credit - debit
				WHEN account IN ('payable', 'loan') THEN debit - credit
				ELSE 0
			END
		), 0)`).
		Where("id < ?", entryID).
		Scan(&balance).Error
	if err != nil {
		return 0, fmt.Errorf("failed to calculate running balance at ID %d: %w", entryID, err)
	}
	return balance, nil
}

// batchUpdateBalances applies CASE WHEN batch updates for a slice of entries,
// computing running balances from zero. Used by RecalculateAllBalances.
func (r *LedgerEntryRepository) batchUpdateBalances(tx *gorm.DB, entries []domain.LedgerEntry) error {
	const batchSize = 500
	caseClauses := make([]string, 0, batchSize)
	ids := make([]uint, 0, batchSize)
	var runningBalance int64

	for i := range entries {
		runningBalance = domain.CalculateNextBalance(runningBalance, &entries[i])
		caseClauses = append(caseClauses, fmt.Sprintf("WHEN %d THEN %d", entries[i].ID, runningBalance))
		ids = append(ids, entries[i].ID)

		if len(caseClauses) >= batchSize || i == len(entries)-1 {
			caseExpr := "CASE id " + strings.Join(caseClauses, " ") + " END"
			if err := tx.Model(&domain.LedgerEntry{}).
				Where("id IN ?", ids).
				Update("balance", gorm.Expr(caseExpr)).Error; err != nil {
				return fmt.Errorf("failed to batch update balances: %w", err)
			}
			caseClauses = caseClauses[:0]
			ids = ids[:0]
		}
	}
	return nil
}

// calculateNetWorthBefore computes Cash + Receivable - Payable - Loan
// for all non-deleted entries strictly before the given date.
func calculateNetWorthBefore(db *gorm.DB, before time.Time) (int64, error) {
	var rows []accountAgg
	if err := db.Model(&domain.LedgerEntry{}).
		Select("account, CAST(COALESCE(SUM(debit), 0) AS SIGNED) as total_debit, CAST(COALESCE(SUM(credit), 0) AS SIGNED) as total_credit").
		Where("date < ? AND deleted_at IS NULL", before).
		Group("account").
		Scan(&rows).Error; err != nil {
		return 0, err
	}

	var netWorth int64
	for _, acc := range rows {
		switch acc.Account {
		case domain.AccountCash, domain.AccountReceivable:
			netWorth += acc.TotalDebit - acc.TotalCredit
		case domain.AccountPayable, domain.AccountLoan:
			netWorth -= acc.TotalCredit - acc.TotalDebit
		}
	}
	return netWorth, nil
}
