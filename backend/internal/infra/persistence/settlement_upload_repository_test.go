package persistence

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newSettlementUploadRepo wires the repository over an in-memory SQLite
// database with the settlement_uploads table created. SQLite-friendly types
// stand in for the MySQL CHAR(64)/JSON/DATETIME columns — the repository
// behavior under test (nil-on-miss lookup, round-trip, unique constraint) is
// independent of those column types.
func newSettlementUploadRepo(t *testing.T) *SettlementUploadRepository {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.Exec(`CREATE TABLE settlement_uploads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_hash TEXT NOT NULL UNIQUE,
		uploaded_at DATETIME NOT NULL,
		request_ids_json TEXT,
		settled_count INTEGER NOT NULL DEFAULT 0
	)`).Error)
	return &SettlementUploadRepository{BaseRepository: &BaseRepository{DB: db}}
}

// TestSettlementUploadRepository_GetByFileHash_MissingReturnsNil pins the
// contract ProcessSettlementFile relies on: a first-time upload (no prior row)
// returns (nil, nil), not an error, so the caller branches on a nil pointer.
func TestSettlementUploadRepository_GetByFileHash_MissingReturnsNil(t *testing.T) {
	t.Parallel()
	repo := newSettlementUploadRepo(t)
	ctx := context.Background()

	got, err := repo.GetByFileHash(ctx, "no-such-hash")
	require.NoError(t, err)
	require.Nil(t, got, "missing row must return (nil, nil)")
}

// TestSettlementUploadRepository_CreateThenFetch verifies the idempotency row
// round-trips and that a re-lookup by file_hash hits the prior record.
func TestSettlementUploadRepository_CreateThenFetch(t *testing.T) {
	t.Parallel()
	repo := newSettlementUploadRepo(t)
	ctx := context.Background()

	upload := &domain.SettlementUpload{
		FileHash:       "deadbeef",
		UploadedAt:     time.Now(),
		RequestIDsJSON: "[1,2,3]",
		SettledCount:   3,
	}
	require.NoError(t, repo.Create(ctx, upload))
	require.NotZero(t, upload.ID, "Create should populate ID")

	got, err := repo.GetByFileHash(ctx, "deadbeef")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "deadbeef", got.FileHash)
	require.Equal(t, int64(3), got.SettledCount)
	require.Equal(t, "[1,2,3]", got.RequestIDsJSON)

	// A second lookup for a different hash is still a nil-on-miss.
	other, err := repo.GetByFileHash(ctx, "cafebabe")
	require.NoError(t, err)
	require.Nil(t, other)
}
