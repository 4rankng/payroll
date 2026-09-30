package persistence

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
)

// Fixture IDs live far beyond any real or integration-suite sequence so the
// cleanup can never delete rows it does not own. The shared dev database also
// contains real rows, so assertions target only these fixture employees.
const (
	fxUser     = uint64(214000000001)
	fxProjectA = uint64(214000000001)
	fxProjectB = uint64(214000000002)
	fxPayrate  = uint64(214000000001)
	fxEmpAn    = uint64(214000000101) // salary+advance, working
	fxEmpBinh  = uint64(214000000102) // has mobile
	fxEmpCuong = uint64(214000000103) // advance before window
	fxEmpDung  = uint64(214000000104) // never-paid advance
	fxEmpGiang = uint64(214000000105) // soft-deleted
	fxEmpHanh  = uint64(214000000106) // advance only
	fxEmpLan   = uint64(214000000107) // boundary dates
	fxEmpPhuc  = uint64(214000000108) // NULL mobile
	fxPE       = uint64(214000000101)
	fxTsBase   = uint64(214000000201) // +0..8
	fxAdvPay   = uint64(214000000301)
	fxAprBase  = uint64(214000000301) // +0..5
	fxBandLow  = uint64(214000000001)
	fxBandHigh = uint64(214000000999)
)

// GetPaidWithoutMobile feeds the admin data-quality export: employees who
// received salary (paid timesheets) or a completed FlexPay advance inside the
// window but have no mobile on file. These tests pin the union semantics
// (either source qualifies), the mobile/soft-delete filters, per-source
// aggregates, the working-status flag, and window boundaries. Runs against
// MySQL because the aggregate timestamps scan as engine datetimes.
func TestEmployeeRepository_GetPaidWithoutMobile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB test in short mode")
	}
	db, err := getTestDB()
	if err != nil {
		t.Skipf("cannot connect to test DB: %v", err)
	}

	cleanup := func() {
		for _, stmt := range []string{
			`DELETE FROM advance_payment_requests WHERE id BETWEEN ? AND ?`,
			`DELETE FROM advance_payments WHERE id BETWEEN ? AND ?`,
			`DELETE FROM timesheets WHERE id BETWEEN ? AND ?`,
			`DELETE FROM payrates WHERE id BETWEEN ? AND ?`,
			`DELETE FROM project_employees WHERE id BETWEEN ? AND ?`,
			`DELETE FROM employees WHERE id BETWEEN ? AND ?`,
			`DELETE FROM projects WHERE id BETWEEN ? AND ?`,
			`DELETE FROM users WHERE id BETWEEN ? AND ?`,
		} {
			_ = db.Exec(stmt, fxBandLow, fxBandHigh).Error
		}
	}
	cleanup()
	t.Cleanup(cleanup)

	loc := time.Local
	mustExec := func(q string, args ...any) {
		require.NoError(t, db.Exec(q, args...).Error)
	}

	// Window [2026-07-01, 2026-09-30 18:00]: both boundaries inclusive.
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, loc)
	to := time.Date(2026, 9, 30, 18, 0, 0, 0, loc)

	// Creator satisfies the created_by foreign keys.
	mustExec(`INSERT INTO users (id, username, password, fullname, role) VALUES (?, 'paidnomobile-fixture', 'x', 'Người tạo test', 'partner')`, fxUser)
	mustExec(`INSERT INTO projects (id, name, created_by) VALUES (?, 'Dự án Alpha', ?), (?, 'Dự án Beta', ?)`, fxProjectA, fxUser, fxProjectB, fxUser)
	mustExec(`INSERT INTO payrates (id, project_id, payrate_json, from_date, created_by) VALUES (?, ?, '{"normal": 10000}', '2026-01-01', ?)`, fxPayrate, fxProjectA, fxUser)

	// An: salary+advance, working; Bình: has mobile; Cường: advance before
	// window; Dũng: never-paid advance; Giang: soft-deleted; Hạnh: advance
	// only; Lan: boundary dates; Phúc: NULL mobile.
	mustExec(`INSERT INTO employees (id, fullname, cccd, mobile, deleted_at, created_by) VALUES
		(?, 'Nguyễn Văn An', '214000000101', '', NULL, ?),
		(?, 'Trần Thị Bình', '214000000102', '0900000002', NULL, ?),
		(?, 'Lê Văn Cường', '214000000103', '', NULL, ?),
		(?, 'Phạm Văn Dũng', '214000000104', '', NULL, ?),
		(?, 'Đỗ Thị Giang', '214000000105', '', '2026-08-01 00:00:00', ?),
		(?, 'Vũ Thị Hạnh', '214000000106', '', NULL, ?),
		(?, 'Đặng Văn Lan', '214000000107', '', NULL, ?),
		(?, 'Bùi Thị Phúc', '214000000108', NULL, NULL, ?)`,
		fxEmpAn, fxUser, fxEmpBinh, fxUser, fxEmpCuong, fxUser, fxEmpDung, fxUser,
		fxEmpGiang, fxUser, fxEmpHanh, fxUser, fxEmpLan, fxUser, fxEmpPhuc, fxUser)

	// Only An has an active assignment => working.
	mustExec(`INSERT INTO project_employees (id, project_id, employee_id, employee_name, employee_cccd, start_date, last_date, created_by) VALUES
		(?, ?, ?, 'Nguyễn Văn An', '214000000101', '2026-01-01', NULL, ?)`,
		fxPE, fxProjectA, fxEmpAn, fxUser)

	mustExec(`INSERT INTO timesheets (id, project_id, employee_id, payrate_id, date, hours_worked, paytype, payrate, amount, timesheet_status, payment_status, paid_amount, paid_at, created_by) VALUES
		(?, ?, ?, ?, '2026-08-10', 8, 'test', 0, 0, 'approved', 'paid', 2000000, ?, ?),
		(?, ?, ?, ?, '2026-08-20', 8, 'test', 0, 0, 'approved', 'paid', 1500000, ?, ?),
		(?, ?, ?, ?, '2026-08-21', 8, 'test', 0, 0, 'approved', 'pending', 900000, ?, ?),
		(?, ?, ?, ?, '2026-08-11', 8, 'test', 0, 0, 'approved', 'paid', 500000, ?, ?),
		(?, ?, ?, ?, '2026-08-12', 8, 'test', 0, 0, 'approved', 'paid', 700000, ?, ?),
		(?, ?, ?, ?, '2026-09-30', 8, 'test', 0, 0, 'approved', 'paid', 800000, ?, ?),
		(?, ?, ?, ?, '2026-08-01', 8, 'test', 0, 0, 'approved', 'paid', 400000, ?, ?),
		(?, ?, ?, ?, '2026-08-02', 8, 'test', 0, 0, 'approved', 'paid', 600000, ?, ?),
		(?, ?, ?, ?, '2026-08-03', 8, 'test', 0, 0, 'approved', 'paid', 200000, ?, ?)`,
		fxTsBase, fxProjectA, fxEmpAn, fxPayrate, d(2026, 8, 10, loc), fxUser,
		fxTsBase+1, fxProjectA, fxEmpAn, fxPayrate, d(2026, 8, 20, loc), fxUser,
		fxTsBase+2, fxProjectA, fxEmpAn, fxPayrate, d(2026, 8, 21, loc), fxUser,
		fxTsBase+3, fxProjectA, fxEmpBinh, fxPayrate, d(2026, 8, 11, loc), fxUser,
		fxTsBase+4, fxProjectA, fxEmpGiang, fxPayrate, d(2026, 8, 12, loc), fxUser,
		fxTsBase+5, fxProjectA, fxEmpLan, fxPayrate, to, fxUser,
		fxTsBase+6, fxProjectA, fxEmpPhuc, fxPayrate, d(2026, 8, 1, loc), fxUser,
		fxTsBase+7, fxProjectA, fxEmpPhuc, fxPayrate, d(2026, 8, 2, loc), fxUser,
		fxTsBase+8, fxProjectB, fxEmpPhuc, fxPayrate, d(2026, 8, 3, loc), fxUser)

	mustExec(`INSERT INTO advance_payments (id, project_id, employee_id, for_month, upload_date, max_adv_amount, salary) VALUES
		(?, ?, ?, '2099-09', '2099-09-01', 0, 0)`, fxAdvPay, fxProjectA, fxEmpAn)

	mustExec(`INSERT INTO advance_payment_requests (id, adv_pay_id, project_id, employee_id, request_amount, fee, provider_fee, net_amount, status, paid_at) VALUES
		(?, ?, ?, ?, 4000000, 1000000, 0, 3000000, 'COMPLETED', ?),
		(?, ?, ?, ?, 1000000, 100000, 0, 900000, 'FAILED', ?),
		(?, ?, ?, ?, 1000000, 100000, 0, 900000, 'COMPLETED', ?),
		(?, ?, ?, ?, 1000000, 100000, 0, 900000, 'PENDING', NULL),
		(?, ?, ?, ?, 3000000, 500000, 0, 2500000, 'COMPLETED', ?),
		(?, ?, ?, ?, 1300000, 100000, 0, 1200000, 'COMPLETED', ?)`,
		fxAprBase, fxAdvPay, fxProjectA, fxEmpAn, d(2026, 9, 15, loc),
		fxAprBase+1, fxAdvPay, fxProjectA, fxEmpAn, d(2026, 9, 16, loc),
		fxAprBase+2, fxAdvPay, fxProjectA, fxEmpCuong, d(2026, 6, 30, loc),
		fxAprBase+3, fxAdvPay, fxProjectA, fxEmpDung,
		fxAprBase+4, fxAdvPay, fxProjectA, fxEmpHanh, d(2026, 9, 5, loc),
		fxAprBase+5, fxAdvPay, fxProjectA, fxEmpLan, from)

	repo := &EmployeeRepository{
		BaseRepository: &BaseRepository{DB: db},
	}
	ctx := context.Background()

	rows, err := repo.GetPaidWithoutMobile(ctx, from, to)
	require.NoError(t, err)

	// Included: An (salary+advance), Hạnh (advance only), Lan (both boundary
	// dates), Phúc (NULL mobile, salary only). Excluded: Bình (has mobile),
	// Cường (advance before window), Dũng (advance never paid), Giang
	// (soft-deleted), plus An's un-paid and failed rows.
	byID := make(map[uint]*domain.EmployeePaidActivity, len(rows))
	for _, row := range rows {
		byID[row.EmployeeID] = row
	}

	an := byID[uint(fxEmpAn)]
	require.NotNil(t, an)
	require.Equal(t, int64(3500000), an.SalaryTotal)
	require.Equal(t, int64(2), an.SalaryCount)
	require.Equal(t, int64(3000000), an.AdvanceTotal)
	require.Equal(t, int64(1), an.AdvanceCount)
	require.True(t, an.LastSalaryPaidAt.Equal(time.Date(2026, 8, 20, 0, 0, 0, 0, loc)))
	require.True(t, an.LastAdvancePaidAt.Equal(time.Date(2026, 9, 15, 0, 0, 0, 0, loc)))
	require.True(t, an.LastActivityAt.Equal(time.Date(2026, 9, 15, 0, 0, 0, 0, loc)))
	require.True(t, an.IsWorking, "active assignment => working")
	require.Equal(t, "Dự án Alpha", an.Projects)

	hanh := byID[uint(fxEmpHanh)]
	require.NotNil(t, hanh, "advance-only employee qualifies through the union")
	require.Equal(t, int64(0), hanh.SalaryTotal)
	require.Equal(t, int64(0), hanh.SalaryCount)
	require.Equal(t, int64(2500000), hanh.AdvanceTotal)
	require.False(t, hanh.IsWorking, "no assignment => not working")
	require.True(t, hanh.LastActivityAt.Equal(time.Date(2026, 9, 5, 0, 0, 0, 0, loc)))
	require.Nil(t, hanh.LastSalaryPaidAt)

	lan := byID[uint(fxEmpLan)]
	require.NotNil(t, lan, "paid_at == from and paid_at == to are both inclusive")
	require.Equal(t, int64(800000), lan.SalaryTotal)
	require.Equal(t, int64(1200000), lan.AdvanceTotal)
	require.True(t, lan.LastActivityAt.Equal(to))

	phuc := byID[uint(fxEmpPhuc)]
	require.NotNil(t, phuc, "NULL mobile counts as missing")
	require.Equal(t, int64(1200000), phuc.SalaryTotal)
	require.Equal(t, "Dự án Alpha,Dự án Beta", phuc.Projects)

	// Excluded fixture employees must never appear.
	for _, excluded := range []uint{uint(fxEmpBinh), uint(fxEmpCuong), uint(fxEmpDung), uint(fxEmpGiang)} {
		require.NotContains(t, byID, excluded)
	}
}

// d builds a fixture datetime in the given location.
func d(year int, month time.Month, day int, loc *time.Location) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}
