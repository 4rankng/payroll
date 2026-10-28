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

// newZnsMessageRepo wires the repository over an in-memory SQLite database
// with the zns_messages table created. The list behavior under test (filters,
// pagination, DESC order) is independent of the MySQL column types.
func newZnsMessageRepo(t *testing.T) *ZnsMessageRepository {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.Exec(`CREATE TABLE zns_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME NOT NULL,
		feature TEXT NOT NULL DEFAULT '',
		template_id TEXT NOT NULL DEFAULT '',
		phone TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT '',
		provider_msg_id TEXT NOT NULL DEFAULT '',
		provider_code INTEGER NOT NULL DEFAULT 0,
		error_msg TEXT NOT NULL DEFAULT '',
		duration_ms INTEGER NOT NULL DEFAULT 0
	)`).Error)
	return &ZnsMessageRepository{BaseRepository: &BaseRepository{DB: db}}
}

func seedZns(t *testing.T, repo *ZnsMessageRepository, rows ...*domain.ZnsMessage) {
	t.Helper()
	for _, r := range rows {
		require.NoError(t, repo.Record(context.Background(), r))
	}
}

// TestZnsMessageListAdmin_NewestFirstWithTotal pins AC1: the admin list is
// newest-first and the total reflects every matching row, not the page size.
func TestZnsMessageListAdmin_NewestFirstWithTotal(t *testing.T) {
	t.Parallel()
	repo := newZnsMessageRepo(t)
	base := time.Date(2026, 10, 28, 10, 0, 0, 0, time.Local)
	seedZns(t, repo,
		&domain.ZnsMessage{Feature: "salary_notification", Phone: "0911111111", Status: domain.ZnsStatusSent, CreatedAt: base.Add(-2 * time.Hour)},
		&domain.ZnsMessage{Feature: "salary_notification", Phone: "0922222222", Status: domain.ZnsStatusSent, CreatedAt: base},
		&domain.ZnsMessage{Feature: "password_reset", Phone: "0933333333", Status: domain.ZnsStatusFailed, CreatedAt: base.Add(-time.Hour)},
	)

	rows, total, err := repo.ListAdmin(context.Background(), domain.ZnsMessageFilter{Page: 1, PageSize: 2})
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, rows, 2)
	require.Equal(t, "0922222222", rows[0].Phone, "newest first")
	require.Equal(t, "0933333333", rows[1].Phone)
}

// TestZnsMessageListAdmin_FiltersCombineIndependently pins AC2: status,
// feature, date range and free-text query each filter on their own and
// combine with AND.
func TestZnsMessageListAdmin_FiltersCombineIndependently(t *testing.T) {
	t.Parallel()
	repo := newZnsMessageRepo(t)
	day := time.Date(2026, 10, 28, 0, 0, 0, 0, time.Local)
	seedZns(t, repo,
		&domain.ZnsMessage{Feature: "salary_notification", TemplateID: "tpl-a", Phone: "0912345678", Status: domain.ZnsStatusSent, ProviderMsgID: "zns-1", CreatedAt: day.Add(9 * time.Hour)},
		&domain.ZnsMessage{Feature: "salary_notification", TemplateID: "tpl-a", Phone: "0987654321", Status: domain.ZnsStatusFailed, ProviderMsgID: "zns-2", CreatedAt: day.Add(10 * time.Hour)},
		&domain.ZnsMessage{Feature: "password_reset", TemplateID: "tpl-b", Phone: "0900000001", Status: domain.ZnsStatusSent, ProviderMsgID: "zns-3", CreatedAt: day.Add(11 * time.Hour)},
		&domain.ZnsMessage{Feature: "password_reset", TemplateID: "tpl-b", Phone: "0900000002", Status: domain.ZnsStatusSent, ProviderMsgID: "zns-4", CreatedAt: day.Add(26 * time.Hour)},
	)

	ctx := context.Background()

	// status only
	rows, total, err := repo.ListAdmin(ctx, domain.ZnsMessageFilter{Statuses: []string{domain.ZnsStatusFailed}, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Equal(t, "0987654321", rows[0].Phone)

	// feature only
	rows, total, err = repo.ListAdmin(ctx, domain.ZnsMessageFilter{Feature: "password_reset", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, 2, total)

	// template only
	rows, total, err = repo.ListAdmin(ctx, domain.ZnsMessageFilter{TemplateID: "tpl-a", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, 2, total)

	// date range (day 1 only)
	from, to := day, day
	rows, total, err = repo.ListAdmin(ctx, domain.ZnsMessageFilter{FromDate: &from, ToDate: &to, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, 3, total)

	// q matches phone substring
	rows, total, err = repo.ListAdmin(ctx, domain.ZnsMessageFilter{Query: "09123", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Equal(t, "0912345678", rows[0].Phone)

	// q matches provider message id
	rows, total, err = repo.ListAdmin(ctx, domain.ZnsMessageFilter{Query: "zns-3", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)

	// combined: feature AND status
	_, total, err = repo.ListAdmin(ctx, domain.ZnsMessageFilter{
		Feature:  "salary_notification",
		Statuses: []string{domain.ZnsStatusFailed},
		Page:     1, PageSize: 10,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)

	// combined: feature AND date range AND status → empty
	_, total, err = repo.ListAdmin(ctx, domain.ZnsMessageFilter{
		Feature:  "password_reset",
		Statuses: []string{domain.ZnsStatusFailed},
		FromDate: &from, ToDate: &to,
		Page:     1, PageSize: 10,
	})
	require.NoError(t, err)
	require.EqualValues(t, 0, total)
}

// TestZnsMessageListAdmin_PaginationPagesThroughAllRows pins AC1 paging:
// total stays constant while pages advance without overlap.
func TestZnsMessageListAdmin_PaginationPagesThroughAllRows(t *testing.T) {
	t.Parallel()
	repo := newZnsMessageRepo(t)
	base := time.Date(2026, 10, 28, 8, 0, 0, 0, time.Local)
	for i := range 5 {
		seedZns(t, repo, &domain.ZnsMessage{
			Feature:   "test_send",
			Phone:     "090000000" + string(rune('1'+i)),
			Status:    domain.ZnsStatusSent,
			CreatedAt: base.Add(time.Duration(i) * time.Minute),
		})
	}

	ctx := context.Background()
	page1, total, err := repo.ListAdmin(ctx, domain.ZnsMessageFilter{Page: 1, PageSize: 2})
	require.NoError(t, err)
	require.EqualValues(t, 5, total)

	page2, total, err := repo.ListAdmin(ctx, domain.ZnsMessageFilter{Page: 2, PageSize: 2})
	require.NoError(t, err)
	require.EqualValues(t, 5, total)

	page3, _, err := repo.ListAdmin(ctx, domain.ZnsMessageFilter{Page: 3, PageSize: 2})
	require.NoError(t, err)

	require.Len(t, page1, 2)
	require.Len(t, page2, 2)
	require.Len(t, page3, 1)
	require.NotEqual(t, page1[0].ID, page2[0].ID, "pages must not overlap")
	require.True(t, page1[0].CreatedAt.After(page2[0].CreatedAt), "newest first across pages")
}
