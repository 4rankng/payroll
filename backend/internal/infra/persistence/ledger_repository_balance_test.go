package persistence

import (
	"context"
	"testing"

	"api-server/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetAccountTotalForTransactionsScopesAccountAndTransactionIDs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:ledger_scope_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE ledger_entries (
			id INTEGER PRIMARY KEY,
			account TEXT NOT NULL,
			debit INTEGER NOT NULL,
			credit INTEGER NOT NULL,
			transaction_id INTEGER,
			deleted_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create ledger_entries: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO ledger_entries (id, account, debit, credit, transaction_id, deleted_at) VALUES
			(1, 'receivable', 1000, 0, 10, NULL),
			(2, 'receivable', 0, 100, 10, NULL),
			(3, 'receivable', 0, 900000, 20, NULL),
			(4, 'cash', 500000, 0, 10, NULL),
			(5, 'receivable', 700000, 0, 10, '2026-01-01')
	`).Error; err != nil {
		t.Fatalf("insert ledger entries: %v", err)
	}

	repository := &LedgerEntryRepository{
		BaseRepository: NewBaseRepository(&Database{DB: db}),
	}

	total, err := repository.GetAccountTotalForTransactions(
		context.Background(),
		domain.AccountReceivable,
		[]uint{10},
	)
	if err != nil {
		t.Fatalf("GetAccountTotalForTransactions() error = %v", err)
	}
	if total != 900 {
		t.Fatalf("scoped total = %d, want 900", total)
	}

	emptyTotal, err := repository.GetAccountTotalForTransactions(
		context.Background(),
		domain.AccountReceivable,
		nil,
	)
	if err != nil {
		t.Fatalf("empty GetAccountTotalForTransactions() error = %v", err)
	}
	if emptyTotal != 0 {
		t.Fatalf("empty scoped total = %d, want 0", emptyTotal)
	}
}

func TestCalculateNetWorthFromAccountTotals(t *testing.T) {
	rows := []accountAgg{
		{Account: domain.AccountCash, TotalDebit: 100, TotalCredit: 20},
		{Account: domain.AccountReceivable, TotalDebit: 40, TotalCredit: 0},
		{Account: domain.AccountPayable, TotalDebit: 5, TotalCredit: 30},
		{Account: domain.AccountLoan, TotalDebit: 0, TotalCredit: 20},
		{Account: domain.AccountRevenue, TotalDebit: 0, TotalCredit: 500},
	}

	if got, want := calculateNetWorthFromAccountTotals(rows), int64(75); got != want {
		t.Fatalf("calculateNetWorthFromAccountTotals() = %d, want %d", got, want)
	}
}
