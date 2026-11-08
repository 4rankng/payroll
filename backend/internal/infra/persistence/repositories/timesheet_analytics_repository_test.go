package repositories

import (
	"context"
	"fmt"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

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

func TestGetPartnerEmployeeStatsWindow(t *testing.T) {
	repo, db := newTimesheetAnalyticsTestRepository(t)

	now := time.Date(2026, time.November, 8, 10, 0, 0, 0, clock.DefaultLocation)
	clock.SetGlobal(clock.NewFake(now))
	t.Cleanup(func() { clock.SetGlobal(clock.New()) })

	day := func(n int) time.Time { return now.AddDate(0, 0, n) }
	insert := func(employeeID int, date time.Time, paymentStatus string, paidAmount int64, deleted bool) {
		t.Helper()
		var deletedAt interface{}
		if deleted {
			deletedAt = date
		}
		require.NoError(t, db.Exec(`
			INSERT INTO timesheets (
				project_id, employee_id, date, amount, timesheet_status,
				payment_status, paid_amount, updated_at, deleted_at
			) VALUES (1, ?, ?, 10000, 'approved', ?, ?, ?, ?)
		`, employeeID, date, paymentStatus, paidAmount, date, deletedAt).Error)
	}

	// Employee 1: active (timesheet yesterday), paid inside the selected month.
	insert(1, day(-1), "paid", 100, false)
	// Employee 2: dropped — last timesheet 30d ago (inside the 60d window,
	// before the 14d cutoff), paid row OUTSIDE the selected month.
	insert(2, day(-30), "paid", 500, false)
	// Employee 3: churned — last timesheet 90d ago, outside the 60d window;
	// not counted anywhere (Phase C2 bound).
	insert(3, day(-90), "paid", 900, false)
	// Employee 4: old timesheet AND a recent one — active only, never dropped
	// (pins the old correlated-NOT-EXISTS semantics).
	insert(4, day(-30), "pending", 0, false)
	insert(4, day(-2), "approved", 0, false)
	// Employee 5: soft-deleted rows are invisible even when in scope.
	insert(5, day(-1), "paid", 1000, true)

	scope := &PartnerScope{PartnerID: 7, EmployeeIDs: []uint{1, 2, 3, 4, 5}}
	monthStart := time.Date(2026, time.November, 1, 0, 0, 0, 0, clock.DefaultLocation)
	monthEnd := monthStart.AddDate(0, 1, 0)

	stats, err := repo.GetPartnerEmployeeStats(context.Background(), scope, &monthStart, &monthEnd)
	require.NoError(t, err)
	require.Equal(t, 2, stats.ActiveEmployees, "employees 1 and 4 have a timesheet in the last 14 days")
	require.Equal(t, 1, stats.DroppedEmployees, "employee 2 only; employee 4 is active, employee 3 churned out of the 60d window")
	require.Equal(t, 1, stats.PaidEmployees, "employee 1 only — employee 2's paid row predates the month window")
	require.Equal(t, 100.0, stats.TotalPaidAmount)
}
