package persistence

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Resolving a reversal group has to work for both kinds of block the ledger
// holds: entries written by the payroll workers carry a transaction id, while the
// manual /ledger/entries endpoint writes a block in one INSERT and leaves the
// transaction id NULL. The second case is grouped on the write timestamp and
// author, which is exactly what these tests pin down: if the grouping were wrong
// the reversal would mirror the wrong rows and unbalance the books.
func TestLedgerEntryRepository_ListReversalGroup(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/ledger_reversal.db"), &gorm.Config{})
	require.NoError(t, err)
	// Hand-rolled DDL (AutoMigrate drags the Creator association into a MySQL-only
	// schema) and parameterised inserts, so the stored timestamps come from the
	// same driver encoding the repository binds against when it filters. Literal
	// SQL timestamps would make this test pass or fail on formatting instead of on
	// the grouping rule.
	require.NoError(t, db.Exec(`
	CREATE TABLE ledger_entries (
		id INTEGER PRIMARY KEY,
		date DATETIME,
		account TEXT,
		party TEXT,
		debit INTEGER,
		credit INTEGER,
		balance INTEGER,
		asset_id INTEGER,
		transaction_id INTEGER,
		reversal_of_entry_id INTEGER,
		reversal_reason TEXT,
		settlement_id INTEGER,
		created_by INTEGER,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);
	CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		username TEXT,
		email TEXT,
		fullname TEXT,
		role TEXT,
		deleted_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	);
	INSERT INTO users (id, username, fullname, role) VALUES (9, 'probe', 'Người đảo', 'admin');`).Error)
	repo := &LedgerEntryRepository{BaseRepository: NewBaseRepository(&Database{DB: db})}
	ctx := context.Background()

	blockAt := func(id uint, account domain.LedgerAccount, party string, debit, credit int64, author uint, written time.Time, txnID *uint) *domain.LedgerEntry {
		return &domain.LedgerEntry{
			ID: id, Date: written, Account: account, Party: party,
			Debit: debit, Credit: credit, CreatedBy: author, CreatedAt: written,
			TransactionID: txnID,
		}
	}

	manualWritten := time.Date(2026, 9, 20, 10, 30, 0, 123000000, time.UTC)
	laterWritten := time.Date(2026, 9, 20, 10, 30, 1, 123000000, time.UTC)
	workerWritten := time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)
	workerTxnID := uint(42)

	manualBlock := []*domain.LedgerEntry{
		blockAt(1, "cash", "VFIC", 1000000, 0, 7, manualWritten, nil),
		blockAt(2, "payable", "VFIC", 0, 1000000, 7, manualWritten, nil),
	}
	laterBlock := []*domain.LedgerEntry{
		blockAt(3, "cash", "LG", 250000, 0, 7, laterWritten, nil),
		blockAt(4, "payable", "LG", 0, 250000, 7, laterWritten, nil),
	}
	otherAuthor := blockAt(5, "cash", "OTHER", 500000, 0, 8, manualWritten, nil)
	deletedLeg := blockAt(6, "revenue", "VFIC", 0, 1000000, 7, manualWritten, nil)
	deletedLeg.DeletedAt = gorm.DeletedAt{Time: time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC), Valid: true}
	workerBlock := []*domain.LedgerEntry{
		blockAt(7, "cash", "TT", 700000, 0, 7, workerWritten, &workerTxnID),
		blockAt(8, "revenue", "TT", 0, 700000, 7, workerWritten, &workerTxnID),
	}

	insert := func(e *domain.LedgerEntry, deletedAt *time.Time) {
		require.NoError(t, db.Exec(
			`INSERT INTO ledger_entries (id, date, account, party, debit, credit, created_by, created_at, transaction_id, deleted_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			e.ID, e.Date, string(e.Account), e.Party, e.Debit, e.Credit, e.CreatedBy, e.CreatedAt, e.TransactionID, deletedAt,
		).Error)
	}
	for _, entry := range manualBlock {
		insert(entry, nil)
	}
	for _, entry := range laterBlock {
		insert(entry, nil)
	}
	insert(otherAuthor, nil)
	deleted := deletedLeg.DeletedAt.Time
	insert(deletedLeg, &deleted)
	for _, entry := range workerBlock {
		insert(entry, nil)
	}

	byID := func(id uint) *domain.LedgerEntry {
		entry, err := repo.GetByID(ctx, id)
		require.NoError(t, err)
		return entry
	}
	ids := func(entries []*domain.LedgerEntry) []uint {
		out := make([]uint, 0, len(entries))
		for _, e := range entries {
			out = append(out, e.ID)
		}
		return out
	}

	t.Run("manual block groups by write timestamp and author", func(t *testing.T) {
		group, err := repo.ListReversalGroup(ctx, byID(1))
		require.NoError(t, err)
		require.Equal(t, []uint{1, 2}, ids(group),
			"only the legs written in the same insert, excluding other authors and deleted rows")
	})

	t.Run("a later block by the same author is its own group", func(t *testing.T) {
		group, err := repo.ListReversalGroup(ctx, byID(3))
		require.NoError(t, err)
		require.Equal(t, []uint{3, 4}, ids(group))
	})

	t.Run("transaction id wins when present", func(t *testing.T) {
		group, err := repo.ListReversalGroup(ctx, byID(7))
		require.NoError(t, err)
		require.Equal(t, []uint{7, 8}, ids(group))
	})

	t.Run("has reversal only sees live mirrors", func(t *testing.T) {
		reversed, err := repo.HasReversal(ctx, 1)
		require.NoError(t, err)
		require.False(t, reversed, "nothing mirrors entry 1 yet")

		mirror := &domain.LedgerEntry{
			Date: time.Now(), Account: "cash", Party: "VFIC", Credit: 1000000,
			CreatedBy: 9, ReversalReason: "sai sót",
		}
		originalID := uint(1)
		mirror.ReversalOfEntryID = &originalID
		require.NoError(t, repo.Create(ctx, mirror))

		reversed, err = repo.HasReversal(ctx, 1)
		require.NoError(t, err)
		require.True(t, reversed)

		reversed, err = repo.HasReversal(ctx, 2)
		require.NoError(t, err)
		require.False(t, reversed, "the check is per entry, not per group")
	})
}
