package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newLoanRepaymentScheduleRepo wires the repository over an in-memory SQLite
// database with the lenders, loans, and loan_repayment_schedules tables the
// reminder join reads. SQLite-friendly types stand in for the MySQL
// enum/date columns — the behavior under test (window bounds, status filter,
// soft-delete exclusion, join projection) is independent of those types.
func newLoanRepaymentScheduleRepo(t *testing.T) *LoanRepaymentScheduleRepository {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.Exec(`CREATE TABLE lenders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE loans (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		lender_id INTEGER NOT NULL,
		loan_code TEXT NOT NULL,
		deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE loan_repayment_schedules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		loan_id INTEGER NOT NULL,
		period INTEGER NOT NULL,
		due_date DATETIME NOT NULL,
		amount INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		paid_at DATETIME,
		payment_ref TEXT,
		transaction_id INTEGER,
		created_at DATETIME,
		updated_at DATETIME
	)`).Error)
	return &LoanRepaymentScheduleRepository{BaseRepository: &BaseRepository{DB: db}}
}

func seedLoanReminderRow(t *testing.T, repo *LoanRepaymentScheduleRepository, loanCode, lenderName string, period int, dueDate time.Time, amount int64, status string, loanDeleted bool) {
	t.Helper()
	deletedAt := "NULL"
	if loanDeleted {
		deletedAt = "'2026-01-01 00:00:00'"
	}
	require.NoError(t, repo.DB.Exec(
		`INSERT INTO lenders (name, deleted_at) VALUES (?, NULL)`, lenderName,
	).Error)
	var lenderID int
	require.NoError(t, repo.DB.Raw(`SELECT last_insert_rowid()`).Scan(&lenderID).Error)
	require.NoError(t, repo.DB.Exec(
		`INSERT INTO loans (lender_id, loan_code, deleted_at) VALUES (?, ?, `+deletedAt+`)`, lenderID, loanCode,
	).Error)
	var loanID int
	require.NoError(t, repo.DB.Raw(`SELECT last_insert_rowid()`).Scan(&loanID).Error)
	require.NoError(t, repo.DB.Exec(
		`INSERT INTO loan_repayment_schedules (loan_id, period, due_date, amount, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, '2026-01-01 00:00:00', '2026-01-01 00:00:00')`,
		loanID, period, dueDate.UTC(), amount, status,
	).Error)
}

// TestLoanRepaymentScheduleRepository_ListPendingForReminder pins the
// exact-window contract of the daily reminder: only pending schedules whose
// due_date falls inside [start, end), joined to live loans and lenders.
func TestLoanRepaymentScheduleRepository_ListPendingForReminder(t *testing.T) {
	t.Parallel()
	repo := newLoanRepaymentScheduleRepo(t)
	ctx := context.Background()
	location := time.FixedZone("ICT", 7*3600)

	start := time.Date(2026, 2, 1, 0, 0, 0, 0, location)
	end := start.AddDate(0, 0, 1)

	seedLoanReminderRow(t, repo, "LOAN-2026-001", "Ngân hàng A", 1, start.Add(2*time.Hour), 1000000, "pending", false)
	seedLoanReminderRow(t, repo, "LOAN-2026-001", "Ngân hàng A", 2, start.Add(3*time.Hour), 2000000, "pending", false)
	seedLoanReminderRow(t, repo, "LOAN-2026-002", "Ngân hàng B", 1, start.Add(1*time.Hour), 500000, "paid", false)
	seedLoanReminderRow(t, repo, "LOAN-2026-003", "Ngân hàng C", 1, start.Add(-24*time.Hour), 700000, "pending", false) // due today: out
	seedLoanReminderRow(t, repo, "LOAN-2026-004", "Ngân hàng D", 1, start.AddDate(0, 0, 3), 800000, "pending", false)   // later: out
	seedLoanReminderRow(t, repo, "LOAN-2026-005", "Ngân hàng E", 1, end, 900000, "pending", false)                      // end boundary: out
	seedLoanReminderRow(t, repo, "LOAN-2026-006", "Ngân hàng F", 1, start.Add(time.Hour), 950000, "pending", true)      // deleted loan: out

	// SQLite compares DATETIME values as strings, so the query window is
	// normalized to UTC to match the UTC-normalized seed rows. MySQL compares
	// real DATETIMEs, where the location of the parameter is irrelevant.
	reminders, err := repo.ListPendingForReminder(ctx, start.UTC(), end.UTC())
	require.NoError(t, err)
	// LOAN-2026-005 sits exactly on the end boundary and must be excluded by
	// the [start, end) window; only the two LOAN-2026-001 periods qualify.
	require.Len(t, reminders, 2)

	require.Equal(t, "LOAN-2026-001", reminders[0].LoanCode)
	require.Equal(t, "Ngân hàng A", reminders[0].LenderName)
	require.Equal(t, 1, reminders[0].Period)
	require.Equal(t, int64(1000000), reminders[0].Amount)

	// Rows arrive ordered by loan code then period, ready for consolidation.
	require.Equal(t, "LOAN-2026-001", reminders[1].LoanCode)
	require.Equal(t, 2, reminders[1].Period)

	for _, reminder := range reminders {
		require.NotZero(t, reminder.ScheduleID, "projection must carry the schedule id for traceability")
	}
}

// TestLoanRepaymentScheduleRepository_ListPendingForReminder_MonthRolloverWindow
// verifies a window computed across a month boundary still selects pending
// rows dated on the last day of the month.
func TestLoanRepaymentScheduleRepository_ListPendingForReminder_MonthRolloverWindow(t *testing.T) {
	t.Parallel()
	repo := newLoanRepaymentScheduleRepo(t)
	ctx := context.Background()
	location := time.FixedZone("ICT", 7*3600)

	// Reminder computed on Jan 30 looks at [Jan 31, Feb 1).
	start := time.Date(2026, 1, 31, 0, 0, 0, 0, location)
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, location)

	seedLoanReminderRow(t, repo, "LOAN-2026-100", "Ngân hàng Z", 5, start.Add(5*time.Hour), 12345000, "pending", false)

	reminders, err := repo.ListPendingForReminder(ctx, start.UTC(), end.UTC())
	require.NoError(t, err)
	require.Len(t, reminders, 1)
	require.Equal(t, "LOAN-2026-100", reminders[0].LoanCode)
	require.Equal(t, 5, reminders[0].Period)
	require.Equal(t, int64(12345000), reminders[0].Amount)
}
