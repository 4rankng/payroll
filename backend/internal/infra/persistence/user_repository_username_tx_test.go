package persistence

import (
	"context"
	"testing"

	"api-server/internal/domain"
	"api-server/internal/infra/transaction"
)

// TestGetUsernamesByPrefix_SeesUncommittedTxWrites guards the import-time
// username collision check. Employees auto-created from one BCC upload share a
// single transaction; two of them can generate the same base username, and the
// collision lookup must run on the tx connection to see the first, still
// uncommitted insert — a pool connection reads a stale snapshot and the second
// INSERT fails with a duplicate-key conflict.
func TestGetUsernamesByPrefix_SeesUncommittedTxWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, err := getTestDB()
	if err != nil {
		t.Skipf("cannot connect to test DB: %v", err)
	}

	repo := &UserRepository{BaseRepository: &BaseRepository{DB: db}}
	txMgr := transaction.NewGormTransactionManager(db)
	ctx := context.Background()

	const base = "uqtxquyenlv"
	// Remove leftovers from a previous failed run before asserting availability.
	db.Unscoped().Where("username LIKE ?", base+"%").Delete(&domain.User{})

	txErr := txMgr.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := repo.Create(txCtx, &domain.User{
			Username: base,
			Fullname: "Lù Văn Quyến",
			Role:     domain.RoleEmployee,
			Password: "not-a-real-login",
		}); err != nil {
			t.Fatalf("setup: create user inside tx: %v", err)
		}

		got, err := repo.GetUsernamesByPrefix(txCtx, base)
		if err != nil {
			t.Fatalf("GetUsernamesByPrefix inside tx: %v", err)
		}
		found := false
		for _, name := range got {
			if name == base {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetUsernamesByPrefix must see the uncommitted in-tx insert; got %v, want containing %q", got, base)
		}
		return nil
	})
	if txErr != nil {
		t.Fatalf("transaction failed: %v", txErr)
	}

	// Cleanup regardless of outcome.
	t.Cleanup(func() {
		db.Unscoped().Where("username LIKE ?", base+"%").Delete(&domain.User{})
	})
}
