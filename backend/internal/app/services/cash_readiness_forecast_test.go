package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"api-server/internal/config"
	"api-server/internal/domain"
	"api-server/internal/domain/wallet"
	"api-server/internal/pkg/clock"
)

// --- fakes -----------------------------------------------------------------

type fakeTimesheetReader struct {
	summary     *domain.TimesheetSummaryStats
	cohort      []domain.TimesheetAccrualDailyRow
	cohortCalls int
}

func (f *fakeTimesheetReader) GetSummaryStats(_ context.Context, _ domain.TimesheetFilters) (*domain.TimesheetSummaryStats, error) {
	return f.summary, nil
}

func (f *fakeTimesheetReader) GetAccrualCohort(_ context.Context, _ domain.TimesheetFilters) ([]domain.TimesheetAccrualDailyRow, error) {
	f.cohortCalls++
	return f.cohort, nil
}

// fakeWallet implements only WalletBalanceReader. It deliberately has NO
// SyncBalance/CreateTopup methods — the forecast's port type makes those
// unreachable, which is the structural form of the advisory invariant.
type fakeWallet struct {
	balance     *wallet.WalletBalance
	err         error
	getBalCalls int
}

func (f *fakeWallet) GetBalance(_ context.Context) (*wallet.WalletBalance, error) {
	f.getBalCalls++
	return f.balance, f.err
}

func newSvc(t *testing.T, ts *fakeTimesheetReader, w WalletBalanceReader, now time.Time) *CashReadinessForecastService {
	t.Helper()
	return NewCashReadinessForecastService(
		ts, ts, w, NewTimesheetAccrualProvider(), clock.NewFake(now), config.CashForecastConfig{},
	)
}

// july3 = 2026-07-03: Ky 1, cycle-day 3, pay day Jul 10.
var july3 = time.Date(2026, time.July, 3, 9, 0, 0, 0, clock.DefaultLocation)

func TestGetCashReadiness_ConfirmedMatchesSummary(t *testing.T) {
	ts := &fakeTimesheetReader{
		summary: &domain.TimesheetSummaryStats{PendingPaymentAmount: 462_000_000},
	}
	svc := newSvc(t, ts, &fakeWallet{balance: &wallet.WalletBalance{Available: 100_000_000}}, july3)

	got, err := svc.GetCashReadiness(context.Background(), domain.TimesheetFilters{})
	if err != nil {
		t.Fatalf("GetCashReadiness: %v", err)
	}
	if got.ConfirmedPayable != 462_000_000 {
		t.Errorf("ConfirmedPayable = %d, want 462000000 (must match GetSummaryStats)", got.ConfirmedPayable)
	}
}

func TestGetCashReadiness_GapMath(t *testing.T) {
	// No cohort history → projection collapses to 0; CashToPrepare = confirmed.
	ts := &fakeTimesheetReader{summary: &domain.TimesheetSummaryStats{PendingPaymentAmount: 300_000_000}}
	svc := newSvc(t, ts, &fakeWallet{balance: &wallet.WalletBalance{Available: 120_000_000}}, july3)

	got, err := svc.GetCashReadiness(context.Background(), domain.TimesheetFilters{})
	if err != nil {
		t.Fatalf("GetCashReadiness: %v", err)
	}
	if got.CashToPrepare != 300_000_000 {
		t.Errorf("CashToPrepare = %d, want 300000000", got.CashToPrepare)
	}
	if got.Gap != 180_000_000 {
		t.Errorf("Gap = %d, want 180000000 (300M − 120M)", got.Gap)
	}
	// Wallet covers the need → gap clamps to 0.
	rich := &fakeWallet{balance: &wallet.WalletBalance{Available: 500_000_000}}
	svc2 := newSvc(t, ts, rich, july3)
	got2, _ := svc2.GetCashReadiness(context.Background(), domain.TimesheetFilters{})
	if got2.Gap != 0 {
		t.Errorf("Gap = %d, want 0 when wallet >= cash_to_prepare", got2.Gap)
	}
}

// two Ky-1 historical cycles → non-degenerate gamma-fit projection.
func twoCyclesCohort() []domain.TimesheetAccrualDailyRow {
	d := func(s string) time.Time {
		t, _ := time.ParseInLocation("2006-01-02", s, clock.DefaultLocation)
		return t
	}
	return []domain.TimesheetAccrualDailyRow{
		// 2026-05 Ky1: approvals at cycle-days 1, 5, 10 → grandTotal 1000.
		{WorkDate: d("2026-05-03"), ApprovedDate: d("2026-05-01"), Amount: 100},
		{WorkDate: d("2026-05-03"), ApprovedDate: d("2026-05-05"), Amount: 200},
		{WorkDate: d("2026-05-03"), ApprovedDate: d("2026-05-10"), Amount: 700},
		// 2026-06 Ky1: approvals at cycle-days 2, 8 → grandTotal 500.
		{WorkDate: d("2026-06-02"), ApprovedDate: d("2026-06-02"), Amount: 150},
		{WorkDate: d("2026-06-02"), ApprovedDate: d("2026-06-08"), Amount: 350},
		// 2026-07 Ky1 (current cycle) — MUST be excluded from the historical basis.
		{WorkDate: d("2026-07-02"), ApprovedDate: d("2026-07-02"), Amount: 999_999},
	}
}

func TestGetCashReadiness_BandMonotonicAndProjects(t *testing.T) {
	ts := &fakeTimesheetReader{
		summary: &domain.TimesheetSummaryStats{PendingPaymentAmount: 100_000_000},
		cohort:  twoCyclesCohort(),
	}
	svc := newSvc(t, ts, &fakeWallet{balance: &wallet.WalletBalance{Available: 50_000_000}}, july3)

	got, err := svc.GetCashReadiness(context.Background(), domain.TimesheetFilters{})
	if err != nil {
		t.Fatalf("GetCashReadiness: %v", err)
	}
	if got.BandLower > got.BandUpper {
		t.Errorf("band not monotonic: Lower=%d > Upper=%d", got.BandLower, got.BandUpper)
	}
	if got.ProjectedP50 > got.ProjectedP95 {
		t.Errorf("p50 > p95: %d > %d", got.ProjectedP50, got.ProjectedP95)
	}
	if got.ProjectedExpected < got.ProjectedP50 || got.ProjectedExpected > got.ProjectedP95 {
		t.Errorf("expected projection %d is outside p50–p95 interval [%d, %d]", got.ProjectedExpected, got.ProjectedP50, got.ProjectedP95)
	}
	if got.ExpectedTotal != got.ConfirmedPayable+got.ProjectedExpected {
		t.Errorf("ExpectedTotal = %d, want confirmed + expected projection = %d", got.ExpectedTotal, got.ConfirmedPayable+got.ProjectedExpected)
	}
	if got.ProjectedP50 <= 0 {
		t.Errorf("ProjectedP50 = %d, want > 0 with 2 historical cycles", got.ProjectedP50)
	}
	if got.BasisCycles != 2 {
		t.Errorf("BasisCycles = %d, want 2 (current cycle excluded)", got.BasisCycles)
	}
	if got.Method == "" || got.Method == "no-history" {
		t.Errorf("Method = %q, want a real projection method", got.Method)
	}
	if got.Confidence == "" {
		t.Errorf("Confidence empty")
	}
}

func TestGetCashReadiness_NoHistoryFallback(t *testing.T) {
	ts := &fakeTimesheetReader{
		summary: &domain.TimesheetSummaryStats{PendingPaymentAmount: 200_000_000},
		cohort:  nil, // no historical data
	}
	svc := newSvc(t, ts, &fakeWallet{balance: &wallet.WalletBalance{Available: 0}}, july3)

	got, err := svc.GetCashReadiness(context.Background(), domain.TimesheetFilters{})
	if err != nil {
		t.Fatalf("GetCashReadiness: %v", err)
	}
	if got.Method != "no-history" {
		t.Errorf("Method = %q, want no-history", got.Method)
	}
	if got.Confidence != "low" {
		t.Errorf("Confidence = %q, want low", got.Confidence)
	}
	if got.ProjectedP50 != 0 || got.ProjectedExpected != 0 || got.ProjectedP95 != 0 {
		t.Errorf("no-history projection must be 0, got p50=%d expected=%d p95=%d", got.ProjectedP50, got.ProjectedExpected, got.ProjectedP95)
	}
	if got.ExpectedTotal != 200_000_000 {
		t.Errorf("ExpectedTotal = %d, want confirmed-only 200000000", got.ExpectedTotal)
	}
	if got.CashToPrepare != 200_000_000 {
		t.Errorf("CashToPrepare = %d, want confirmed-only 200000000", got.CashToPrepare)
	}
	if got.BandLower != got.BandUpper {
		t.Errorf("no-history band must collapse: Lower=%d Upper=%d", got.BandLower, got.BandUpper)
	}
}

func TestGetCashReadiness_Deterministic(t *testing.T) {
	ts := &fakeTimesheetReader{
		summary: &domain.TimesheetSummaryStats{PendingPaymentAmount: 100_000_000},
		cohort:  twoCyclesCohort(),
	}
	mk := func() *CashReadinessForecastService {
		return newSvc(t, ts, &fakeWallet{balance: &wallet.WalletBalance{Available: 40_000_000}}, july3)
	}
	a, err := mk().GetCashReadiness(context.Background(), domain.TimesheetFilters{})
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	b, err := mk().GetCashReadiness(context.Background(), domain.TimesheetFilters{})
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if a.ProjectedP50 != b.ProjectedP50 || a.ProjectedExpected != b.ProjectedExpected || a.ProjectedP95 != b.ProjectedP95 || a.CashToPrepare != b.CashToPrepare || a.ExpectedTotal != b.ExpectedTotal {
		t.Errorf("non-deterministic: p50 a=%d b=%d, expected a=%d b=%d, p95 a=%d b=%d", a.ProjectedP50, b.ProjectedP50, a.ProjectedExpected, b.ProjectedExpected, a.ProjectedP95, b.ProjectedP95)
	}
}

func TestGetCashReadiness_AdvisoryInvariant(t *testing.T) {
	// The service depends on WalletBalanceReader (GetBalance only). It has no
	// type-level access to SyncBalance/CreateTopup, so the invariant is enforced
	// by the port. This test confirms the wallet is read via GetBalance and the
	// forecast still returns when the wallet read fails (defensive).
	ts := &fakeTimesheetReader{summary: &domain.TimesheetSummaryStats{PendingPaymentAmount: 100_000_000}}
	w := &fakeWallet{balance: &wallet.WalletBalance{Available: 30_000_000}}
	svc := newSvc(t, ts, w, july3)

	got, err := svc.GetCashReadiness(context.Background(), domain.TimesheetFilters{})
	if err != nil {
		t.Fatalf("GetCashReadiness: %v", err)
	}
	if w.getBalCalls == 0 {
		t.Error("wallet GetBalance was never called")
	}
	if !got.WalletAvailableOK || got.WalletAvailable != 30_000_000 {
		t.Errorf("WalletAvailable = %d (ok=%v), want 30000000 ok=true", got.WalletAvailable, got.WalletAvailableOK)
	}

	// Wallet read failure must not blank the card — confirmed + projection still shown.
	wErr := &fakeWallet{err: errFailedWallet}
	svc2 := newSvc(t, ts, wErr, july3)
	got2, err := svc2.GetCashReadiness(context.Background(), domain.TimesheetFilters{})
	if err != nil {
		t.Fatalf("GetCashReadiness with wallet error: %v", err)
	}
	if got2.WalletAvailableOK {
		t.Error("WalletAvailableOK should be false on read failure")
	}
	if got2.ConfirmedPayable != 100_000_000 {
		t.Errorf("confirmed must still populate on wallet failure: %d", got2.ConfirmedPayable)
	}
}

func TestGetCashReadiness_PayCycleFraming(t *testing.T) {
	ts := &fakeTimesheetReader{summary: &domain.TimesheetSummaryStats{PendingPaymentAmount: 0}}
	svc := newSvc(t, ts, &fakeWallet{balance: &wallet.WalletBalance{}}, july3)

	got, _ := svc.GetCashReadiness(context.Background(), domain.TimesheetFilters{})
	if got.Ky != 1 {
		t.Errorf("Ky = %d, want 1", got.Ky)
	}
	wantPay := time.Date(2026, time.July, 10, 0, 0, 0, 0, clock.DefaultLocation)
	if !got.NextPayDate.Equal(wantPay) {
		t.Errorf("NextPayDate = %v, want %v", got.NextPayDate, wantPay)
	}
	wantPrepare := time.Date(2026, time.July, 8, 0, 0, 0, 0, clock.DefaultLocation) // Jul 10 − 2
	if !got.PrepareByDate.Equal(wantPrepare) {
		t.Errorf("PrepareByDate = %v, want %v", got.PrepareByDate, wantPrepare)
	}
	if got.CycleDayToday != 3 {
		t.Errorf("CycleDayToday = %d, want 3", got.CycleDayToday)
	}
}

func TestBuildTimesheetCohort_PartitionsByKy(t *testing.T) {
	d := func(s string) time.Time {
		t, _ := time.ParseInLocation("2006-01-02", s, clock.DefaultLocation)
		return t
	}
	// A Ky-2 row (work day 10) and a Ky-1 row (work day 3); requesting Ky 1 keeps only the latter.
	rows := []domain.TimesheetAccrualDailyRow{
		{WorkDate: d("2026-06-10"), ApprovedDate: d("2026-06-12"), Amount: 500}, // Ky 2
		{WorkDate: d("2026-06-03"), ApprovedDate: d("2026-06-03"), Amount: 200}, // Ky 1, cycle-day 3
	}
	got := buildTimesheetCohort(rows, 1, "2026-07")
	if len(got) != 1 {
		t.Fatalf("got %d series, want 1 (Ky-partitioned)", len(got))
	}
	if got[0].grandTotal != 200 {
		t.Errorf("grandTotal = %d, want 200 (Ky-2 row excluded)", got[0].grandTotal)
	}
}

// errFailedWallet is a sentinel for wallet-read failure in tests.
var errFailedWallet = errors.New("wallet unavailable")

// TestBuildTimesheetCohort_ChronologicallySorted verifies the sort that the
// growth EWMA depends on — Go map iteration is randomized, so without this
// sort the same request could return different growth factors across restarts.
func TestBuildTimesheetCohort_ChronologicallySorted(t *testing.T) {
	d := func(s string) time.Time {
		t, _ := time.ParseInLocation("2006-01-02", s, clock.DefaultLocation)
		return t
	}
	// 3 historical Ky-1 cycles (work day 3) across different months.
	rows := []domain.TimesheetAccrualDailyRow{
		{WorkDate: d("2026-06-03"), ApprovedDate: d("2026-06-10"), Amount: 300},
		{WorkDate: d("2026-04-03"), ApprovedDate: d("2026-04-10"), Amount: 100},
		{WorkDate: d("2026-05-03"), ApprovedDate: d("2026-05-10"), Amount: 200},
	}
	got := buildTimesheetCohort(rows, 1, "2026-07")
	if len(got) != 3 {
		t.Fatalf("got %d series, want 3", len(got))
	}
	want := []string{"2026-04", "2026-05", "2026-06"}
	for i, w := range want {
		if got[i].forMonth != w {
			t.Errorf("series[%d].forMonth = %q, want %q (chronological)", i, got[i].forMonth, w)
		}
	}
	// Grand totals must also be in chronological order (100, 200, 300).
	wantTotals := []int64{100, 200, 300}
	for i, w := range wantTotals {
		if got[i].grandTotal != w {
			t.Errorf("series[%d].grandTotal = %d, want %d", i, got[i].grandTotal, w)
		}
	}
}
