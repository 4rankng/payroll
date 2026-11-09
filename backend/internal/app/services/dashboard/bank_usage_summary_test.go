package dashboard

import (
	"testing"

	"api-server/internal/infra/persistence/repositories"
)

// The fold must reproduce BOTH legacy queries' semantics from one scan:
//   - overall: every paid timesheet of employees with >=1 active assignment,
//     each timesheet once — even when the employee is active in several
//     projects (the legacy join is employee-level, not pair-level);
//   - per project: the (employee, project) pair must be active;
//   - employees without any active assignment drop out entirely.
func TestFoldBankUsageSummary(t *testing.T) {
	scan := []repositories.EmployeeProjectBankUsageRow{
		// e1 active only in p1; two paid timesheets there.
		{EmployeeID: 1, ProjectID: 10, BankName: "MB", TotalPaid: 200, TransferCount: 2},
		// e2 active in p1 AND p2 — its buckets feed overall once each, and both
		// project views.
		{EmployeeID: 2, ProjectID: 10, BankName: "MB", TotalPaid: 100, TransferCount: 1},
		{EmployeeID: 2, ProjectID: 20, BankName: "MB", TotalPaid: 50, TransferCount: 1},
		// e3 active only in p2.
		{EmployeeID: 3, ProjectID: 20, BankName: "VCB", TotalPaid: 200, TransferCount: 1},
		// e4 has paid timesheets but NO active assignment: legacy INNER JOIN on
		// project_employees excludes it from overall too.
		{EmployeeID: 4, ProjectID: 90, BankName: "MB", TotalPaid: 999, TransferCount: 1},
	}
	pairs := []repositories.ActiveProjectPair{
		{EmployeeID: 1, ProjectID: 10},
		{EmployeeID: 2, ProjectID: 10},
		{EmployeeID: 2, ProjectID: 20},
		{EmployeeID: 3, ProjectID: 20},
	}

	rows := foldBankUsageSummary(scan, pairs)

	type key struct {
		project uint64
		bank    string
	}
	want := map[key]struct {
		employees int
		paid      float64
		transfers int
	}{
		{0, "MB"}:  {2, 350, 4}, // e1 (200) + e2 (100+50); e4 excluded
		{0, "VCB"}: {1, 200, 1},
		{10, "MB"}: {2, 300, 3}, // e1 + e2/p1 only
		{20, "MB"}: {1, 50, 1},  // e2/p2
		{20, "VCB"}: {1, 200, 1},
	}

	got := map[key]repositories.BankUsageSummaryRow{}
	for _, row := range rows {
		got[key{row.ProjectID, row.BankName}] = row
	}
	if len(got) != len(want) {
		t.Fatalf("fold produced %d buckets, want %d: %+v", len(got), len(want), got)
	}
	for k, w := range want {
		row, ok := got[k]
		if !ok {
			t.Fatalf("missing bucket %+v", k)
		}
		if row.EmployeeCount != w.employees || row.TotalPaid != w.paid || row.TransferCount != w.transfers {
			t.Errorf(
				"bucket project=%d bank=%s: got employees=%d paid=%v transfers=%d, want %d/%v/%d",
				k.project, k.bank, row.EmployeeCount, row.TotalPaid, row.TransferCount,
				w.employees, w.paid, w.transfers,
			)
		}
	}
}
