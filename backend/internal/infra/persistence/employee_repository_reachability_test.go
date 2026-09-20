package persistence

import (
	"context"
	"testing"

	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// The reachability report is an operator worklist: it must point at the
// employees whose contact data is actually unusable, must not flag employees it
// cannot prove are unreachable (a later successful Zalo send, a soft-deleted
// twin), and must attach the money exposure so the worklist can be prioritised.
func TestEmployeeRepository_ReachabilityReport(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/reachability.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
	CREATE TABLE employees (id INTEGER PRIMARY KEY, fullname TEXT, cccd TEXT, mobile TEXT, deleted_at DATETIME);
	CREATE TABLE projects (id INTEGER PRIMARY KEY, name TEXT, deleted_at DATETIME);
	CREATE TABLE project_employees (id INTEGER PRIMARY KEY, project_id INTEGER, employee_id INTEGER,
		start_date DATETIME, last_date DATETIME, deleted_at DATETIME);
	CREATE TABLE timesheets (id INTEGER PRIMARY KEY, employee_id INTEGER, amount INTEGER, paid_amount INTEGER,
		timesheet_status TEXT, payment_status TEXT, deleted_at DATETIME);
	CREATE TABLE flexpay_salary_notifications (id INTEGER PRIMARY KEY, employee_id INTEGER, status TEXT,
		provider_code INTEGER, created_at DATETIME);

	INSERT INTO employees (id, fullname, cccd, mobile, deleted_at) VALUES
		(1,  'Không có số',        '031000000001', '',           NULL),
		(2,  'Đã gửi lại được',    '031000000002', '0901234567', NULL),
		(3,  'Chưa liên kết Zalo', '031000000003', '0907654321', NULL),
		(4,  'Bị tắt thông báo',   '031000000004', '0901111222', NULL),
		(5,  'Đã xoá',             '031000000005', '',           '2026-01-01 00:00:00'),
		(6,  'Số bị chia sẻ A',    '031000000006', '0902222333', NULL),
		(7,  'Số bị chia sẻ B',    '031000000007', '0902222333', NULL),
		(8,  'Số sai định dạng',   '031000000008', '03289220248', NULL),
		(10, 'Vừa sai vừa chưa LK','031000000010', '090333',     NULL),
		(11, 'Dự án bắt đầu sau',  '031000000011', '',           NULL),
		(12, 'Bản đã xoá trùng số','031000000012', '0912345678', NULL),
		(13, 'Bản ghi đã xoá',     '031000000013', '0912345678', '2026-01-01 00:00:00');

	INSERT INTO projects (id, name, deleted_at) VALUES
		(100, 'Dự án Một', NULL),
		(200, 'Dự án Hai', NULL),
		(300, 'Dự án Đã Xoá', '2026-01-01 00:00:00');

	INSERT INTO project_employees (id, project_id, employee_id, start_date, last_date, deleted_at) VALUES
		(1, 100, 1,  '2024-01-01', NULL,         NULL),
		(2, 100, 6,  '2024-01-01', NULL,         NULL),
		(3, 200, 3,  '2024-01-01', NULL,         NULL),
		(4, 200, 3,  '2024-01-01', '2025-06-30', NULL),
		(5, 300, 3,  '2024-01-01', NULL,         NULL),
		(6, 100, 11, '2999-01-01', NULL,         NULL);

	INSERT INTO timesheets (id, employee_id, amount, paid_amount, timesheet_status, payment_status, deleted_at) VALUES
		(1, 1, 500000,  500000, 'approved',         'paid',    NULL),
		(2, 1, 200000,  0,      'approved',         'pending', NULL),
		(3, 1, 999999,  0,      'rejected',         'pending', NULL),
		(4, 1, 111111,  0,      'approved',         'cancelled', NULL),
		(5, 1, 777777,  0,      'approved',         'pending', '2026-01-01 00:00:00'),
		(6, 10, 90000,  90000,  'approved',         'paid',    NULL),
		(7, 10, 70000,  0,      'approved',         'failed',  NULL),
		(8, 10, 300000, 0,      'pending_approval', 'pending', NULL);

	INSERT INTO flexpay_salary_notifications (id, employee_id, status, provider_code, created_at) VALUES
		(1, 2,  'suppressed', -118, '2026-01-01 00:00:00'),
		(2, 2,  'sent',       0,    '2026-06-01 00:00:00'),
		(3, 3,  'suppressed', -118, '2026-05-01 00:00:00'),
		(4, 4,  'suppressed', 0,    '2026-05-01 00:00:00'),
		(5, 10, 'suppressed', -118, '2026-05-01 00:00:00'),
		(6, 10, 'sent',       0,    '2026-04-01 00:00:00');
	`).Error)

	repo := &EmployeeRepository{BaseRepository: &BaseRepository{DB: db}}
	ctx := context.Background()

	ids := func(rows []*domain.EmployeeReachability) []uint {
		out := make([]uint, 0, len(rows))
		for _, row := range rows {
			out = append(out, row.EmployeeID)
		}
		return out
	}
	rowByID := func(rows []*domain.EmployeeReachability, id uint) *domain.EmployeeReachability {
		for _, row := range rows {
			if row.EmployeeID == id {
				return row
			}
		}
		t.Fatalf("employee %d missing from report", id)
		return nil
	}

	// listPage returns the classified page for the filters. The page carries the
	// cohort counters alongside its rows, so one read serves both.
	listPage := func(t *testing.T, filters domain.EmployeeReachabilityFilters) *domain.EmployeeReachabilityPage {
		t.Helper()
		page, err := repo.ListEmployeeReachability(ctx, filters)
		require.NoError(t, err)
		return page
	}

	t.Run("classifies every state and skips reachable employees", func(t *testing.T) {
		rows := listPage(t, domain.EmployeeReachabilityFilters{}).Employees
		require.Equal(t, []uint{1, 3, 6, 7, 8, 10, 11}, ids(rows),
			"ordered by employee id, excluding reachable/soft-deleted employees")

		require.Equal(t, []string{domain.EmployeeReachabilityStateNoPhone}, rowByID(rows, 1).States)
		require.Equal(t, []string{domain.EmployeeReachabilityStateNotLinked}, rowByID(rows, 3).States)
		require.Equal(t, []string{domain.EmployeeReachabilityStateDuplicateMobile}, rowByID(rows, 6).States)
		require.Equal(t, []string{domain.EmployeeReachabilityStateDuplicateMobile}, rowByID(rows, 7).States)
		require.Equal(t, []string{domain.EmployeeReachabilityStateInvalidMobile}, rowByID(rows, 8).States)
		require.Equal(t, []string{domain.EmployeeReachabilityStateNoPhone}, rowByID(rows, 11).States)
		require.Equal(t, []string{
			domain.EmployeeReachabilityStateNotLinked,
			domain.EmployeeReachabilityStateInvalidMobile,
		}, rowByID(rows, 10).States, "an employee can be unreachable for more than one reason")
	})

	// Employee 2 was suppressed once but the most recent notification was sent:
	// reporting them would send ops after a number that already works. Employee 4
	// was suppressed because the feature was off, which says nothing about Zalo.
	t.Run("only the latest notification decides not_linked", func(t *testing.T) {
		rows := listPage(t, domain.EmployeeReachabilityFilters{
			State: domain.EmployeeReachabilityStateNotLinked,
		}).Employees
		require.Equal(t, []uint{3, 10}, ids(rows))
	})

	t.Run("state filter narrows the page", func(t *testing.T) {
		cases := map[string][]uint{
			domain.EmployeeReachabilityStateNoPhone:         {1, 11},
			domain.EmployeeReachabilityStateDuplicateMobile: {6, 7},
			domain.EmployeeReachabilityStateInvalidMobile:   {8, 10},
		}
		for state, want := range cases {
			rows := listPage(t, domain.EmployeeReachabilityFilters{State: state}).Employees
			require.Equal(t, want, ids(rows), "state %s", state)
		}
	})

	t.Run("project filter uses current assignments only", func(t *testing.T) {
		projectID := uint(100)
		rows := listPage(t, domain.EmployeeReachabilityFilters{ProjectID: &projectID}).Employees
		require.Equal(t, []uint{1, 6}, ids(rows),
			"the future-dated assignment (11) and the deleted project (300) must not count")
	})

	t.Run("projects and money exposure", func(t *testing.T) {
		rows := listPage(t, domain.EmployeeReachabilityFilters{}).Employees

		noPhone := rowByID(rows, 1)
		require.Equal(t, []string{"Dự án Một"}, noPhone.Projects)
		require.Equal(t, int64(500000), noPhone.PaidAmount, "only paid rows count, and only their paid_amount")
		require.Equal(t, int64(200000), noPhone.OutstandingAmount,
			"rejected, cancelled and soft-deleted rows are not payable")

		require.Equal(t, []string{"Dự án Hai"}, rowByID(rows, 3).Projects,
			"ended assignments and soft-deleted projects are excluded")

		overlapping := rowByID(rows, 10)
		require.Equal(t, int64(90000), overlapping.PaidAmount)
		require.Equal(t, int64(370000), overlapping.OutstandingAmount,
			"a failed payment stays payable; pending approval counts as accrued")

		require.Empty(t, rowByID(rows, 8).Projects, "employees with no current assignment keep an empty list")
	})

	t.Run("counters cover the cohort and ignore the state filter", func(t *testing.T) {
		summary := listPage(t, domain.EmployeeReachabilityFilters{}).Summary
		require.Equal(t, domain.EmployeeReachabilitySummary{
			NoPhone: 2, NotLinked: 2, InvalidMobile: 2, DuplicateMobile: 2, Employees: 7,
		}, summary)

		// The page is narrowed by the state filter while the counters are not,
		// because both are derived from the same classified cohort.
		filtered := listPage(t, domain.EmployeeReachabilityFilters{
			State: domain.EmployeeReachabilityStateNoPhone,
		})
		require.Equal(t, summary, filtered.Summary, "counters size the whole worklist, not the filtered page")
		require.Equal(t, []uint{1, 11}, ids(filtered.Employees), "the page is the filtered cohort")

		projectID := uint(100)
		scoped := listPage(t, domain.EmployeeReachabilityFilters{ProjectID: &projectID}).Summary
		require.Equal(t, domain.EmployeeReachabilitySummary{
			NoPhone: 1, NotLinked: 0, InvalidMobile: 0, DuplicateMobile: 1, Employees: 2,
		}, scoped, "the project filter does scope the counters")
	})

	t.Run("pagination applies to classified rows", func(t *testing.T) {
		rows := listPage(t, domain.EmployeeReachabilityFilters{Limit: 2, Offset: 1}).Employees
		require.Equal(t, []uint{3, 6}, ids(rows))

		past := listPage(t, domain.EmployeeReachabilityFilters{Offset: 99})
		require.Empty(t, past.Employees)
	})
}
