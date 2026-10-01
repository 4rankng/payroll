package advance_payment

import (
	"testing"

	"api-server/internal/domain"
)

// A salary period may be funded by the admin workbook OR by self check-in/out.
// These tests pin the request-side backstop: a period carrying both signatures
// must never be spendable, and a check-in period belongs to the check-in flow
// alone so one period never has two request paths.

func quotaRow(projectID, employeeID uint, forMonth string, maxAdv, salary uint64, uploaded bool) *domain.AdvancePayment {
	row := &domain.AdvancePayment{
		ProjectID: projectID, EmployeeID: employeeID, ForMonth: forMonth,
		MaxAdvAmount: maxAdv, Salary: salary,
	}
	if uploaded {
		assetID := uint(55)
		row.LastAppliedAssetID = &assetID
	}
	return row
}

func TestPeriodConflicts(t *testing.T) {
	uploadOnly := quotaRow(1, 2, "2026-10", 2_000_000, 0, true)
	checkInOnly := quotaRow(1, 2, "2026-10", 2_800_000, 4_000_000, false)
	mixed := quotaRow(1, 2, "2026-10", 5_800_000, 4_000_000, true)
	empty := quotaRow(1, 2, "2026-10", 0, 0, false)

	cases := []struct {
		name string
		rows []*domain.AdvancePayment
		want bool
	}{
		{name: "workbook only", rows: []*domain.AdvancePayment{uploadOnly}},
		{name: "check-in only", rows: []*domain.AdvancePayment{checkInOnly}},
		{name: "both sources", rows: []*domain.AdvancePayment{mixed}, want: true},
		{name: "no quota at all", rows: []*domain.AdvancePayment{empty}},
		{name: "no rows", rows: nil},
		{
			name: "mixed in a second project still conflicts",
			rows: []*domain.AdvancePayment{uploadOnly, mixed},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := periodConflicts(tc.rows) != nil; got != tc.want {
				t.Fatalf("periodConflicts found a conflict = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPeriodHasCheckInEarnings(t *testing.T) {
	uploadOnly := quotaRow(1, 2, "2026-10", 2_000_000, 0, true)
	checkInOtherProject := quotaRow(9, 2, "2026-10", 2_800_000, 4_000_000, false)

	if periodHasCheckInEarnings(nil) {
		t.Error("no rows means no check-in period")
	}
	if periodHasCheckInEarnings([]*domain.AdvancePayment{uploadOnly}) {
		t.Error("a workbook-funded period is not a check-in period")
	}
	// The regular endpoint sums the budget across every project the employee is
	// active in, so any check-in earnings make the whole period check-in-funded.
	if !periodHasCheckInEarnings([]*domain.AdvancePayment{uploadOnly, checkInOtherProject}) {
		t.Error("check-in earnings in any project make the period check-in-funded")
	}
}

func TestAdvancePaymentProvenancePredicates(t *testing.T) {
	// The two writers stamp different fields; that asymmetry is the whole
	// detection mechanism, so it is pinned here.
	upload := quotaRow(1, 2, "2026-10", 2_000_000, 0, true)
	if !upload.HasUploadQuota() || upload.HasCheckInEarnings() || upload.HasMixedSources() {
		t.Errorf("workbook row misclassified: upload=%v checkin=%v mixed=%v",
			upload.HasUploadQuota(), upload.HasCheckInEarnings(), upload.HasMixedSources())
	}

	checkIn := quotaRow(1, 2, "2026-10", 2_800_000, 4_000_000, false)
	if checkIn.HasUploadQuota() || !checkIn.HasCheckInEarnings() || checkIn.HasMixedSources() {
		t.Errorf("check-in row misclassified: upload=%v checkin=%v mixed=%v",
			checkIn.HasUploadQuota(), checkIn.HasCheckInEarnings(), checkIn.HasMixedSources())
	}

	both := quotaRow(1, 2, "2026-10", 5_800_000, 4_000_000, true)
	if !both.HasMixedSources() {
		t.Error("a row carrying both signatures must report a conflict")
	}
}
