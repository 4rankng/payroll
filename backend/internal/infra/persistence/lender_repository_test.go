package persistence

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newLenderRepository wires the repository over an in-memory SQLite database
// with the minimal loans table the lender loan checks read. SQLite-friendly
// types stand in for the MySQL column types — the behavior under test
// (linked-loan detection, disbursed-only principal sums, soft-delete
// exclusion) is independent of those types.
func newLenderRepository(t *testing.T) *LenderRepository {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.Exec(`CREATE TABLE loans (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		lender_id INTEGER NOT NULL,
		principal_amount INTEGER NOT NULL DEFAULT 0,
		outstanding_principal INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'active',
		disbursed_at DATETIME,
		deleted_at DATETIME
	)`).Error)
	return &LenderRepository{BaseRepository: &BaseRepository{DB: db}}
}

func seedLenderLoan(t *testing.T, repo *LenderRepository, lenderID int, principal, outstanding int64, status string, disbursed, softDeleted bool) {
	t.Helper()
	disbursedAt := "NULL"
	if disbursed {
		disbursedAt = "'2026-01-01 00:00:00'"
	}
	deletedAt := "NULL"
	if softDeleted {
		deletedAt = "'2026-02-01 00:00:00'"
	}
	require.NoError(t, repo.DB.Exec(
		`INSERT INTO loans (lender_id, principal_amount, outstanding_principal, status, disbursed_at, deleted_at)
		 VALUES (?, ?, ?, ?, `+disbursedAt+`, `+deletedAt+`)`,
		lenderID, principal, outstanding, status,
	).Error)
}

func TestLenderRepository_HasAnyLoans(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("no loans", func(t *testing.T) {
		repo := newLenderRepository(t)
		has, err := repo.HasAnyLoans(ctx, 1)
		require.NoError(t, err)
		require.False(t, has)
	})

	t.Run("draft loan still counts as a linked record", func(t *testing.T) {
		repo := newLenderRepository(t)
		seedLenderLoan(t, repo, 1, 0, 0, "active", false, false)
		has, err := repo.HasAnyLoans(ctx, 1)
		require.NoError(t, err)
		require.True(t, has)
	})

	t.Run("other lenders and soft-deleted loans are ignored", func(t *testing.T) {
		repo := newLenderRepository(t)
		seedLenderLoan(t, repo, 2, 7_000_000, 7_000_000, "active", true, false)
		seedLenderLoan(t, repo, 1, 5_000_000, 5_000_000, "active", true, true)
		has, err := repo.HasAnyLoans(ctx, 1)
		require.NoError(t, err)
		require.False(t, has)
	})
}

func TestLenderRepository_GetLoanSummary(t *testing.T) {
	t.Parallel()
	repo := newLenderRepository(t)
	ctx := context.Background()

	// Disbursed active loan, partially repaid
	seedLenderLoan(t, repo, 1, 10_000_000, 4_000_000, "active", true, false)
	// Disbursed and fully repaid (closed)
	seedLenderLoan(t, repo, 1, 5_000_000, 0, "closed", true, false)
	// Undisbursed draft — counts toward totals but not principal sums
	seedLenderLoan(t, repo, 1, 0, 0, "active", false, false)
	// Disbursed but soft-deleted — must be excluded entirely
	seedLenderLoan(t, repo, 1, 99_000_000, 99_000_000, "active", true, true)
	// Another lender's loan — must not leak into the summary
	seedLenderLoan(t, repo, 2, 7_000_000, 7_000_000, "active", true, false)

	summary, err := repo.GetLoanSummary(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, int64(15_000_000), summary.TotalPrincipalBorrowed)
	require.Equal(t, int64(4_000_000), summary.OutstandingPrincipal)
	require.Equal(t, int64(11_000_000), summary.TotalPrincipalRepaid)
	require.Equal(t, 2, summary.ActiveLoansCount)
	require.Equal(t, 3, summary.TotalLoansCount)
}
