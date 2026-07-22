package repositories

import (
	"fmt"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTimesheetAnalyticsTestRepository(t *testing.T) (*TimesheetAnalyticsRepository, *gorm.DB) {
	t.Helper()

	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_busy_timeout=5000&_time_format=sqlite"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	require.NoError(t, db.Exec(`CREATE TABLE timesheets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		employee_id INTEGER NOT NULL,
		date DATE NOT NULL,
		amount INTEGER NOT NULL,
		timesheet_status TEXT NOT NULL,
		payment_status TEXT NOT NULL,
		paid_amount INTEGER NOT NULL DEFAULT 0,
		updated_at DATETIME NOT NULL,
		deleted_at DATETIME
	)`).Error)

	return NewTimesheetAnalyticsRepository(db), db
}

func TestPendingPaymentSummaryIncludesPendingApprovalInOperationalBacklog(t *testing.T) {
	repo, db := newTimesheetAnalyticsTestRepository(t)
	now := time.Date(2026, time.July, 22, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 123; i++ {
		require.NoError(t, db.Exec(`
			INSERT INTO timesheets (
				project_id, employee_id, date, amount, timesheet_status,
				payment_status, paid_amount, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, 0, ?)
		`, 1, (i%3)+1, now, 10_000, domain.TimesheetStatusPendingApproval,
			domain.PaymentStatusPending, now).Error, fmt.Sprintf("insert pending row %d", i+1))
	}
	require.NoError(t, db.Exec(`
		INSERT INTO timesheets (
			project_id, employee_id, date, amount, timesheet_status,
			payment_status, paid_amount, updated_at
		) VALUES
			(1, 4, ?, 20000, 'approved', 'pending', 0, ?),
			(1, 5, ?, 30000, 'pending_approval', 'paid', 30000, ?),
			(1, 6, ?, 40000, 'approved', 'cancelled', 0, ?)
	`, now, now, now, now, now, now).Error)

	stats, err := repo.getPendingPaymentSummary(domain.TimesheetFilters{})
	require.NoError(t, err)
	require.Equal(t, int64(1_250_000), stats.PendingPaymentAmount)
	require.Equal(t, 4, stats.PendingPaymentEmployees)
}
