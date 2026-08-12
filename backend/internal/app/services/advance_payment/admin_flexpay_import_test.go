package advance_payment

import (
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// TestDedupeAdvancePayments_LastWriteWins reproduces the flexible payroll
// template shape the ops team uses: the same employee can appear in multiple
// sheets within one workbook (e.g. "UL" then "UL (2)"), and the trailing
// occurrence holds the freshest amount. The deduper must collapse to one
// record per (project, employee) and keep the LAST value seen — otherwise
// BatchUpsert ends up issuing conflicting INSERTs for the same unique key
// and the order of resolution becomes implementation-defined.
func TestDedupeAdvancePayments_LastWriteWins(t *testing.T) {
	first := &domain.AdvancePayment{ProjectID: 58, EmployeeID: 638, ForMonth: "2026-04", UploadDate: "2026-04-15", MaxAdvAmount: 2460000}
	second := &domain.AdvancePayment{ProjectID: 58, EmployeeID: 638, ForMonth: "2026-04", UploadDate: "2026-04-15", MaxAdvAmount: 2580000}
	other := &domain.AdvancePayment{ProjectID: 58, EmployeeID: 444, ForMonth: "2026-04", UploadDate: "2026-04-15", MaxAdvAmount: 9060000}

	got := dedupeAdvancePayments([]*domain.AdvancePayment{first, second, other})

	if len(got) != 2 {
		t.Fatalf("expected 2 deduped records, got %d", len(got))
	}

	for _, ap := range got {
		if ap.EmployeeID == 638 && ap.MaxAdvAmount != 2580000 {
			t.Errorf("employee 638: expected last-write-wins amount 2580000, got %d", ap.MaxAdvAmount)
		}
		if ap.EmployeeID == 444 && ap.MaxAdvAmount != 9060000 {
			t.Errorf("employee 444: amount should be unchanged at 9060000, got %d", ap.MaxAdvAmount)
		}
	}
}

func TestDedupeAdvancePayments_PreservesOrder(t *testing.T) {
	a := &domain.AdvancePayment{ProjectID: 1, EmployeeID: 10, MaxAdvAmount: 100}
	b := &domain.AdvancePayment{ProjectID: 1, EmployeeID: 20, MaxAdvAmount: 200}
	c := &domain.AdvancePayment{ProjectID: 1, EmployeeID: 30, MaxAdvAmount: 300}

	got := dedupeAdvancePayments([]*domain.AdvancePayment{a, b, c})

	if len(got) != 3 {
		t.Fatalf("expected 3 records, got %d", len(got))
	}
	if got[0].EmployeeID != 10 || got[1].EmployeeID != 20 || got[2].EmployeeID != 30 {
		t.Errorf("expected order [10,20,30], got [%d,%d,%d]", got[0].EmployeeID, got[1].EmployeeID, got[2].EmployeeID)
	}
}

func TestDedupeAdvancePayments_Empty(t *testing.T) {
	if got := dedupeAdvancePayments(nil); len(got) != 0 {
		t.Errorf("expected empty slice for nil input, got %d", len(got))
	}
	if got := dedupeAdvancePayments([]*domain.AdvancePayment{}); len(got) != 0 {
		t.Errorf("expected empty slice for empty input, got %d", len(got))
	}
}

// TestDedupeAdvancePayments_DifferentProjectsSameEmployee verifies that the
// dedup key is (project, employee) — the same employee assigned to two
// projects gets two separate records, not one.
func TestDedupeAdvancePayments_DifferentProjectsSameEmployee(t *testing.T) {
	p1 := &domain.AdvancePayment{ProjectID: 58, EmployeeID: 638, MaxAdvAmount: 100}
	p2 := &domain.AdvancePayment{ProjectID: 59, EmployeeID: 638, MaxAdvAmount: 200}

	got := dedupeAdvancePayments([]*domain.AdvancePayment{p1, p2})

	if len(got) != 2 {
		t.Fatalf("expected 2 records (different projects), got %d", len(got))
	}
}

func TestResolveUploadDate_PrefersAssetCreatedAtOverFrozenClock(t *testing.T) {
	fake := clock.NewAutoFake()
	clock.SetGlobal(fake)
	t.Cleanup(func() {
		clock.SetGlobal(clock.New())
	})

	fake.Set(time.Date(2026, 6, 6, 9, 0, 0, 0, clock.DefaultLocation))

	uploadedAt := time.Date(2026, 6, 20, 17, 37, 28, 0, time.FixedZone("+08", 8*60*60))

	got := resolveUploadDate(uploadedAt)
	if got != "2026-06-20" {
		t.Fatalf("expected upload date 2026-06-20 from asset created_at, got %s", got)
	}
}

func TestResolveUploadDate_FallsBackToBusinessClockWhenAssetTimestampMissing(t *testing.T) {
	fake := clock.NewAutoFake()
	clock.SetGlobal(fake)
	t.Cleanup(func() {
		clock.SetGlobal(clock.New())
	})

	fake.Set(time.Date(2026, 6, 6, 9, 0, 0, 0, clock.DefaultLocation))

	got := resolveUploadDate(time.Time{})
	if got != "2026-06-06" {
		t.Fatalf("expected fallback upload date 2026-06-06, got %s", got)
	}
}

func TestShouldNotifyFlexPayZNS_OnlyForRequestableFlexibleEmployees(t *testing.T) {
	eligible := &domain.ProjectEmployee{PaymentSchedule: string(domain.PaymentScheduleFlexible)}
	selfCheckIn := &domain.ProjectEmployee{PaymentSchedule: string(domain.PaymentScheduleFlexible), CheckInEnabled: true}
	weekly := &domain.ProjectEmployee{PaymentSchedule: string(domain.PaymentScheduleWeekly)}

	tests := []struct {
		name       string
		amount     uint64
		mobile     string
		assignment *domain.ProjectEmployee
		want       bool
	}{
		{name: "eligible flexible employee", amount: 1_000_000, mobile: "0366178061", assignment: eligible, want: true},
		{name: "self check-in employee", amount: 1_000_000, mobile: "0366178061", assignment: selfCheckIn},
		{name: "non-flexible employee", amount: 1_000_000, mobile: "0366178061", assignment: weekly},
		{name: "missing mobile", amount: 1_000_000, assignment: eligible},
		{name: "zero amount", mobile: "0366178061", assignment: eligible},
		{name: "missing assignment", amount: 1_000_000, mobile: "0366178061"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldNotifyFlexPayZNS(tt.amount, tt.mobile, tt.assignment); got != tt.want {
				t.Fatalf("shouldNotifyFlexPayZNS() = %v, want %v", got, tt.want)
			}
		})
	}
}
