package persistence

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	domaintx "api-server/internal/domain/transactions"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestRepo wires the repository over an in-memory SQLite database
// with the wallet_payments table auto-migrated. Returns the
// repository under test plus the underlying *gorm.DB so the test can
// inspect / mutate rows directly when it wants to corner-case the
// optimistic-locking path.
//
// AutoMigrate quirk: SQLite's column type mapping is lenient, so the
// MySQL DATETIME(3) / JSON columns degrade to TEXT/BLOB. That is fine
// for these tests — the repository's behavior we care about is
// version-bumping and row-affected logic, not column types.
func newTestRepo(t *testing.T) (*TxWalletPaymentRepository, *gorm.DB) {
	t.Helper()
	// Per-test shared in-memory database. cache=shared so concurrent
	// connections from the pool see the same data; SetMaxOpenConns(1)
	// linearizes writes (SQLite's writers serialize anyway, but
	// limiting the pool size means we never see "no such table"
	// from a connection that opened a fresh empty database).
	// _time_format=sqlite ensures DATETIME values round-trip through
	// the *time.Time scanner — without it, SQLite's TEXT-storage default
	// fails on Scan into *time.Time (the path StatsByErrorCode takes).
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_busy_timeout=5000&_time_format=sqlite"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	// Use a hand-rolled CREATE TABLE rather than AutoMigrate: the GORM
	// tag bigint unsigned + autoIncrement collides with SQLite's
	// requirement that AUTOINCREMENT only attach to INTEGER PRIMARY
	// KEY. Tests don't care about the column types, only the FSM
	// behavior, so we simulate the production schema with SQLite-
	// friendly types.
	require.NoError(t, db.Exec(`CREATE TABLE wallet_payments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		txn_id TEXT NOT NULL UNIQUE,
		request_id TEXT NOT NULL UNIQUE,
		invoice_no TEXT,
		provider TEXT NOT NULL DEFAULT '9pay',
		requested_amount INTEGER NOT NULL,
		fee INTEGER NOT NULL DEFAULT 0,
		recipient_name TEXT NOT NULL,
		recipient_account_no TEXT NOT NULL,
		recipient_bank TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		error_code TEXT,
		error_message TEXT,
		entity_id INTEGER,
		created_by INTEGER,
		batch_id TEXT,
		bulk_transfer_batch_id INTEGER,
		bulk_transfer_order INTEGER,
		vfic_code TEXT,
		enqueue_state TEXT DEFAULT 'enqueued',
		sweeper_retry_count INTEGER NOT NULL DEFAULT 0,
		version INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		settled_at DATETIME,
		reconciled_at DATETIME,
		resolution_source TEXT
	)`).Error)

	// Build the repository by hand — the public constructor takes
	// our internal *Database wrapper, which we don't have here.
	repo := &TxWalletPaymentRepository{
		BaseRepository: &BaseRepository{DB: db},
	}
	return repo, db
}

func newRow(t *testing.T) *domaintx.WalletPayment {
	t.Helper()
	return &domaintx.WalletPayment{
		TxnID:              uuid.New(),
		RequestID:          "req-" + uuid.NewString()[:8],
		RequestedAmount:    10_000,
		Fee:                200,
		RecipientName:      "Test User",
		RecipientAccountNo: "0888523111",
		RecipientBank:      "9PAY",
		Status:             domaintx.StatePending,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}

func TestRepository_CreateAndGet(t *testing.T) {
	t.Parallel()
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	row := newRow(t)
	require.NoError(t, repo.Create(ctx, row))
	require.NotZero(t, row.ID, "Create should populate ID")

	got, err := repo.GetByID(ctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, row.RequestID, got.RequestID)
	require.Equal(t, domaintx.StatePending, got.Status)
	require.Equal(t, int64(0), got.Version, "fresh row should be version 0")
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newTestRepo(t)
	_, err := repo.GetByID(context.Background(), 99999)
	require.ErrorIs(t, err, domaintx.ErrNotFound)
}

// TestRepository_MarkReconciled_StatusPrecondition pins H7: MarkReconciled
// (the FSM-bypassing reconcile path) must only override rows currently in a
// reconcilable prior state — failed/completed/authorised/verified, sourced
// from WalletPaymentService.ReconcilePayment's switch. A pending row must be
// refused so the reconcile path cannot silently flip non-terminal rows.
func TestRepository_MarkReconciled_StatusPrecondition(t *testing.T) {
	t.Parallel()
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	reconcilable := []domaintx.State{
		domaintx.StateFailed,
		domaintx.StateCompleted,
		domaintx.StateAuthorised,
		domaintx.StateVerified,
	}
	for _, prior := range reconcilable {
		t.Run("override_from_"+string(prior), func(t *testing.T) {
			t.Parallel()
			row := newRow(t)
			row.Status = prior
			require.NoError(t, repo.Create(ctx, row))

			require.NoError(t, repo.MarkReconciled(ctx, row.ID, domaintx.StateCompleted, time.Now()))

			got, err := repo.GetByID(ctx, row.ID)
			require.NoError(t, err)
			require.Equal(t, domaintx.StateCompleted, got.Status,
				"reconcilable prior state %s should be overridden to completed", prior)
			require.Positive(t, got.Version, "version should be bumped by the override")
		})
	}

	// Non-reconcilable prior state must be refused (the H7 guard).
	t.Run("block_from_pending", func(t *testing.T) {
		t.Parallel()
		row := newRow(t) // Status defaults to StatePending
		require.NoError(t, repo.Create(ctx, row))

		err := repo.MarkReconciled(ctx, row.ID, domaintx.StateCompleted, time.Now())
		require.ErrorIs(t, err, domaintx.ErrNotFound,
			"MarkReconciled must refuse to override a pending row")

		got, err := repo.GetByID(ctx, row.ID)
		require.NoError(t, err)
		require.Equal(t, domaintx.StatePending, got.Status,
			"pending row must be left untouched by the reconcile path")
	})
}

func TestRepository_UpdateExpected_HappyPath(t *testing.T) {
	t.Parallel()
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	row := newRow(t)
	require.NoError(t, repo.Create(ctx, row))

	completed := domaintx.StateCompleted
	invoiceNo := "PN-12345"
	code := "000"
	msg := "Thành công"

	err := repo.UpdateExpected(ctx, row.ID, 0, domaintx.UpdatePatch{
		Status:       &completed,
		InvoiceNo:    &invoiceNo,
		ErrorCode:    &code,
		ErrorMessage: &msg,
	})
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, domaintx.StateCompleted, got.Status)
	require.Equal(t, "PN-12345", got.GetInvoiceNo())
	require.Equal(t, int64(200), got.Fee, "fee stamped at INSERT, untouched on update")
	require.Equal(t, int64(1), got.Version, "version should bump 0→1")
	require.NotNil(t, got.SettledAt, "settled_at should be set on terminal transition")
}

func TestRepository_UpdateExpected_VersionMismatch(t *testing.T) {
	t.Parallel()
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	row := newRow(t)
	require.NoError(t, repo.Create(ctx, row))

	// First update bumps version 0 → 1.
	completed := domaintx.StateCompleted
	require.NoError(t, repo.UpdateExpected(ctx, row.ID, 0, domaintx.UpdatePatch{Status: &completed}))

	// Second update with stale expected version (still 0) must be
	// rejected with ErrConcurrentModification.
	failed := domaintx.StateFailed
	err := repo.UpdateExpected(ctx, row.ID, 0, domaintx.UpdatePatch{Status: &failed})
	require.ErrorIs(t, err, domaintx.ErrConcurrentModification)
}

// TestRepository_UpdateExpected_ConcurrentRace fires N goroutines at the
// same row, each trying to perform the version-0 → version-1 transition.
// Exactly one should win; the rest should hit ErrConcurrentModification.
// This is the database-level half of the concurrency contract — the FSM
// is the legality guard, the version check is the safety net.
func TestRepository_UpdateExpected_ConcurrentRace(t *testing.T) {
	t.Parallel()
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	row := newRow(t)
	require.NoError(t, repo.Create(ctx, row))

	const racers = 25
	var wg sync.WaitGroup
	var wins, misses int32
	completed := domaintx.StateCompleted

	wg.Add(racers)
	for i := 0; i < racers; i++ {
		go func() {
			defer wg.Done()
			err := repo.UpdateExpected(ctx, row.ID, 0, domaintx.UpdatePatch{Status: &completed})
			switch {
			case err == nil:
				atomic.AddInt32(&wins, 1)
			case errors.Is(err, domaintx.ErrConcurrentModification):
				atomic.AddInt32(&misses, 1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	require.Equal(t, int32(1), atomic.LoadInt32(&wins), "exactly one writer wins")
	require.Equal(t, int32(racers-1), atomic.LoadInt32(&misses), "everyone else gets ErrConcurrentModification")

	got, err := repo.GetByID(ctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), got.Version, "row version should be 1")
}

func TestRepository_StatsByStatus(t *testing.T) {
	t.Parallel()
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		row := newRow(t)
		require.NoError(t, repo.Create(ctx, row))
	}
	rows, err := repo.StatsByStatus(ctx, time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 1))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, domaintx.StatePending, rows[0].Status)
	require.Equal(t, int64(3), rows[0].Count)
	require.Equal(t, int64(30_000), rows[0].TotalRequestedAmount)
	require.Equal(t, int64(600), rows[0].TotalFee, "3 rows × 200 fee each = 600")
}

func TestRepository_StatsByErrorCode(t *testing.T) {
	t.Parallel()
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		row := newRow(t)
		require.NoError(t, repo.Create(ctx, row))
		code := "000"
		if i%2 == 1 {
			code = "1004"
		}
		require.NoError(t, repo.UpdateExpected(ctx, row.ID, 0, domaintx.UpdatePatch{ErrorCode: &code}))
	}

	rows, err := repo.StatsByErrorCode(ctx, time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 1))
	require.NoError(t, err)
	// Two distinct codes — sorted by count DESC then code ASC.
	require.Len(t, rows, 2)
	// 3 rows are "000", 2 rows are "1004"
	require.Equal(t, "000", rows[0].ErrorCode)
	require.Equal(t, int64(3), rows[0].Count)
	require.Equal(t, "1004", rows[1].ErrorCode)
	require.Equal(t, int64(2), rows[1].Count)
}

// TestRepository_StatsByStatus_ZeroFeeOnWaive models the zero-fee-on-preflight
// rule: of N failed transfers, only those that actually reached the transfer
// endpoint keep their fee; pre-flight rejections get their stamped fee zeroed
// (FeeWaived → patch.Fee = 0), so SUM(fee) is exactly the fees charged/lost.
func TestRepository_StatsByStatus_ZeroFeeOnWaive(t *testing.T) {
	t.Parallel()
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	failed := domaintx.StateFailed
	zero := int64(0)

	// 3 failed transfers, scheduled fee 200 each. Two reached the transfer
	// endpoint (fee charged/lost); one was rejected at pre-flight (fee waived → 0).
	for i := 0; i < 3; i++ {
		row := newRow(t)
		require.NoError(t, repo.Create(ctx, row))
		patch := domaintx.UpdatePatch{Status: &failed}
		if i == 2 {
			patch.Fee = &zero // pre-flight waive
		}
		require.NoError(t, repo.UpdateExpected(ctx, row.ID, 0, patch))
	}

	rows, err := repo.StatsByStatus(ctx, time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 1))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, domaintx.StateFailed, rows[0].Status)
	require.Equal(t, int64(3), rows[0].Count)
	require.Equal(t, int64(400), rows[0].TotalFee, "2 charged (200 each) + 1 waived (0) = 400 actually lost")
}

// TestRepository_HasNonTerminalByEntityID pins H6: the retry-disbursement
// in-flight guard must flag only rows still mid-flight for the given advance
// request. Terminal rows (completed/failed/reversed), rows linked to a
// different entity_id, rows with a NULL entity_id, and the empty table must
// all report not-in-flight; each non-terminal state must report in-flight.
func TestRepository_HasNonTerminalByEntityID(t *testing.T) {
	t.Parallel()
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	const entityID = uint64(42)
	const otherEntity = uint64(99)

	// Empty table → not in-flight.
	got, err := repo.HasNonTerminalByEntityID(ctx, entityID)
	require.NoError(t, err)
	require.False(t, got, "empty table should report not-in-flight")

	// Terminal rows for this entity → still not in-flight.
	for _, st := range []domaintx.State{domaintx.StateCompleted, domaintx.StateFailed, domaintx.StateReversed} {
		row := newRow(t)
		row.Status = st
		e := entityID
		row.EntityID = &e
		require.NoError(t, repo.Create(ctx, row))
	}
	got, err = repo.HasNonTerminalByEntityID(ctx, entityID)
	require.NoError(t, err)
	require.False(t, got, "terminal rows must not count as in-flight")

	// NULL entity_id non-terminal row must not match the equality lookup.
	nullRow := newRow(t) // EntityID left nil
	nullRow.Status = domaintx.StatePending
	require.NoError(t, repo.Create(ctx, nullRow))
	got, err = repo.HasNonTerminalByEntityID(ctx, entityID)
	require.NoError(t, err)
	require.False(t, got, "NULL entity_id row must not match")

	// Non-terminal row for a different entity must not match.
	otherRow := newRow(t)
	otherRow.Status = domaintx.StatePending
	other := otherEntity
	otherRow.EntityID = &other
	require.NoError(t, repo.Create(ctx, otherRow))
	got, err = repo.HasNonTerminalByEntityID(ctx, entityID)
	require.NoError(t, err)
	require.False(t, got, "non-terminal row for another entity must not match")

	// Each non-terminal state for THIS entity → in-flight.
	for _, st := range []domaintx.State{domaintx.StatePending, domaintx.StateVerified, domaintx.StateAuthorised} {
		row := newRow(t)
		row.Status = st
		e := entityID
		row.EntityID = &e
		require.NoError(t, repo.Create(ctx, row))

		got, err := repo.HasNonTerminalByEntityID(ctx, entityID)
		require.NoError(t, err)
		require.True(t, got, "%s row for this entity should be in-flight", st)
	}
}
