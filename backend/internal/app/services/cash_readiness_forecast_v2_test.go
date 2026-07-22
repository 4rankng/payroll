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

type fakeCashMeasurement struct {
	accuracy  domain.CashForecastAccuracy
	residuals []domain.CashForecastResidual
	upserts   []*domain.CashForecastSnapshot
	upsertErr error
}

func (f *fakeCashMeasurement) Upsert(_ context.Context, snapshot *domain.CashForecastSnapshot) error {
	f.upserts = append(f.upserts, snapshot)
	return f.upsertErr
}

func (f *fakeCashMeasurement) ListResolvedResiduals(_ context.Context, _ domain.CashForecastResolvedQuery) ([]domain.CashForecastResidual, error) {
	return f.residuals, nil
}

func (f *fakeCashMeasurement) GetAccuracy(_ context.Context, _ domain.CashForecastResolvedQuery) (*domain.CashForecastAccuracy, error) {
	result := f.accuracy
	return &result, nil
}

func (f *fakeCashMeasurement) ResolveWeeklyExport(context.Context, time.Time, time.Time, []domain.CashForecastOutcomeItem, string) (int64, error) {
	return 0, nil
}

func forecastRow(work, created, approved string, status domain.TimesheetStatus, employeeID, projectID uint, amount int64) domain.TimesheetAccrualDailyRow {
	parse := func(value string) time.Time {
		result, _ := time.ParseInLocation("2006-01-02", value, clock.DefaultLocation)
		return result
	}
	row := domain.TimesheetAccrualDailyRow{
		WorkDate: parse(work), CreatedAt: parse(created), Status: status,
		EmployeeID: employeeID, ProjectID: projectID, Amount: amount,
	}
	if approved != "" {
		value := parse(approved)
		row.ApprovedAt = &value
	}
	return row
}

func TestCashReadinessV2_PendingInfluencesExpectedWithoutOldBacklog(t *testing.T) {
	rows := []domain.TimesheetAccrualDailyRow{
		forecastRow("2026-07-02", "2026-07-01", "2026-07-02", domain.TimesheetStatusApproved, 1, 10, 100),
		forecastRow("2026-07-03", "2026-07-02", "", domain.TimesheetStatusPendingApproval, 2, 10, 80),
		// Old-cycle pending backlog must not enter target pending exposure.
		forecastRow("2026-06-10", "2026-06-10", "", domain.TimesheetStatusPendingApproval, 99, 99, 9_000_000),
	}
	projection := forecastCashReadinessV2(rows, july3, clock.NextTimesheetPayCycle(july3), timesheetForecastSeed(1, "2026-07", 3), config.CashForecastConfig{})
	if projection.approved != 100 || projection.pending != 80 {
		t.Fatalf("decomposition approved=%d pending=%d, want 100/80", projection.approved, projection.pending)
	}
	if projection.expectedPending <= 0 || projection.expectedPayout <= projection.approved {
		t.Fatalf("pending did not influence expectation: expected_pending=%d payout=%d", projection.expectedPending, projection.expectedPayout)
	}
	if projection.pending >= 9_000_000 {
		t.Fatal("old-cycle pending backlog contaminated target Ky")
	}
}

func TestCashReadinessV2_PoolsNormalizedObservationsAcrossKyDeterministically(t *testing.T) {
	rows := []domain.TimesheetAccrualDailyRow{
		forecastRow("2026-07-02", "2026-07-01", "2026-07-02", domain.TimesheetStatusApproved, 1, 10, 100),
		// Ky 2 and Ky 3 provide comparable future-created shapes for target Ky 1.
		forecastRow("2026-05-08", "2026-05-08", "2026-05-09", domain.TimesheetStatusApproved, 11, 20, 100),
		forecastRow("2026-05-12", "2026-05-12", "2026-05-14", domain.TimesheetStatusApproved, 11, 20, 300),
		forecastRow("2026-06-15", "2026-06-15", "2026-06-16", domain.TimesheetStatusApproved, 21, 30, 200),
		forecastRow("2026-06-20", "2026-06-20", "2026-06-22", domain.TimesheetStatusApproved, 21, 30, 400),
	}
	pc := clock.NextTimesheetPayCycle(july3)
	seed := timesheetForecastSeed(pc.Ky, pc.WorkMonth.Format("2006-01"), pc.CycleDayToday)
	a := forecastCashReadinessV2(rows, july3, pc, seed, config.CashForecastConfig{NSim: 1000})
	b := forecastCashReadinessV2(rows, july3, pc, seed, config.CashForecastConfig{NSim: 1000})
	if a.basisCycles != 2 {
		t.Fatalf("basis cycles=%d, want 2 across different Ky", a.basisCycles)
	}
	if a.expectedFuture <= 0 {
		t.Fatalf("expected future=%d, want pooled future contribution", a.expectedFuture)
	}
	if a != b {
		t.Fatalf("same seed produced different projections: %#v vs %#v", a, b)
	}
	if a.intervalLower > a.expectedPayout || a.expectedPayout > a.intervalUpper || a.recommendedReserve < a.expectedPayout || a.expectedPayout < a.approved {
		t.Fatalf("ordering/floors violated: %#v", a)
	}
}

func TestBuildCashCycleObservations_ExcludesApprovalsAfterPayDate(t *testing.T) {
	rows := []domain.TimesheetAccrualDailyRow{
		// Created before the analogous forecast point but approved after payday:
		// this is a failed pending conversion for that transfer.
		forecastRow("2026-05-08", "2026-05-09", "2026-05-18", domain.TimesheetStatusApproved, 1, 10, 100),
		// Created after the forecast point and also approved after payday: it must
		// not inflate the future-created amount for the May 17 transfer.
		forecastRow("2026-05-09", "2026-05-12", "2026-05-18", domain.TimesheetStatusApproved, 2, 10, 200),
	}
	observations, successes, failures := buildCashCycleObservations(
		map[string][]domain.TimesheetAccrualDailyRow{"2026-05/ky2": rows},
		"2026-07/ky1",
		july3,
		3,
	)
	if successes != 0 || failures != 1 {
		t.Fatalf("pending outcomes successes=%d failures=%d, want 0/1", successes, failures)
	}
	if len(observations) != 1 || observations[0].futurePerEmployee != 0 {
		t.Fatalf("late approvals contaminated future amount: %#v", observations)
	}
}

func TestBuildCashCycleObservations_UsesFinalApprovedTotalAsFallback(t *testing.T) {
	rows := []domain.TimesheetAccrualDailyRow{
		// Bulk-entered after the analogous forecast day. The cycle is still
		// useful as a final-total basis even though it cannot teach pending
		// conversion.
		forecastRow("2026-05-08", "2026-05-12", "2026-05-18", domain.TimesheetStatusApproved, 1, 10, 100),
		forecastRow("2026-05-09", "2026-05-12", "2026-05-18", domain.TimesheetStatusApproved, 2, 10, 300),
	}
	observations, successes, failures := buildCashCycleObservations(
		map[string][]domain.TimesheetAccrualDailyRow{"2026-05/ky2": rows},
		"2026-07/ky1",
		july3,
		3,
	)
	if successes != 0 || failures != 0 {
		t.Fatalf("fallback cycle should not teach conversion: successes=%d failures=%d", successes, failures)
	}
	if len(observations) != 1 || observations[0].futurePerEmployee != 200 {
		t.Fatalf("final-total fallback not used: %#v", observations)
	}
}

func TestBuildCashCycleObservations_IncludesApprovalDuringPayDate(t *testing.T) {
	row := forecastRow("2026-05-08", "2026-05-12", "2026-05-17", domain.TimesheetStatusApproved, 1, 10, 250)
	approvedAt, _ := time.ParseInLocation("2006-01-02 15:04", "2026-05-17 18:30", clock.DefaultLocation)
	row.ApprovedAt = &approvedAt

	observations, _, _ := buildCashCycleObservations(
		map[string][]domain.TimesheetAccrualDailyRow{"2026-05/ky2": {row}},
		"2026-07/ky1",
		july3,
		3,
	)
	if len(observations) != 1 || observations[0].futurePerEmployee != 250 {
		t.Fatalf("pay-date approval missing from future amount: %#v", observations)
	}
}

func TestCashReadinessV2_UsesCompletedCyclesWhenTargetHasNoRowsYet(t *testing.T) {
	rows := []domain.TimesheetAccrualDailyRow{
		// This completed cycle was bulk-entered after the analogous cycle day.
		// It cannot inform pending conversion, but it remains a valid final-total
		// basis for a target cycle whose timesheets have not been entered yet.
		forecastRow("2026-06-08", "2026-06-14", "2026-06-15", domain.TimesheetStatusApproved, 1, 10, 100),
		forecastRow("2026-06-09", "2026-06-14", "2026-06-15", domain.TimesheetStatusApproved, 2, 10, 300),
	}
	now, _ := time.ParseInLocation("2006-01-02", "2026-07-17", clock.DefaultLocation)
	pc := clock.NextTimesheetPayCycle(now)
	projection := forecastCashReadinessV2(rows, now, pc, 42, config.CashForecastConfig{NSim: 1000})

	if projection.basisCycles != 1 {
		t.Fatalf("basis cycles=%d, want completed-cycle fallback", projection.basisCycles)
	}
	if projection.expectedFuture != 400 || projection.expectedPayout != 400 || projection.recommendedReserve != 400 {
		t.Fatalf("completed-cycle fallback projection=%#v, want 400 VND", projection)
	}
	if projection.approved != 0 || projection.pending != 0 {
		t.Fatalf("target decomposition=%#v, want all demand classified as future", projection)
	}
	if projection.method != "completed-cycle-bootstrap" {
		t.Fatalf("method=%q, want honest completed-cycle fallback label", projection.method)
	}
}

func TestCashReadinessV2_UsesRecentWorkforceLevelWhenTargetHasNoRows(t *testing.T) {
	rows := make([]domain.TimesheetAccrualDailyRow, 0)
	addCompletedCycle := func(month time.Month, employees int) {
		work := time.Date(2026, month, 1, 0, 0, 0, 0, clock.DefaultLocation).Format("2006-01-02")
		created := time.Date(2026, month, 7, 0, 0, 0, 0, clock.DefaultLocation).Format("2006-01-02")
		approved := time.Date(2026, month, 10, 0, 0, 0, 0, clock.DefaultLocation).Format("2006-01-02")
		for employee := 1; employee <= employees; employee++ {
			rows = append(rows, forecastRow(work, created, approved, domain.TimesheetStatusApproved, uint(employee), 10, 100))
		}
	}
	// The old median-headcount fallback chose 2 and ignored the sharp, sustained
	// expansion to 10 workers in the two most recent completed cycles.
	addCompletedCycle(time.February, 2)
	addCompletedCycle(time.March, 2)
	addCompletedCycle(time.April, 2)
	addCompletedCycle(time.May, 10)
	addCompletedCycle(time.June, 10)

	now, _ := time.ParseInLocation("2006-01-02", "2026-07-17", clock.DefaultLocation)
	projection := forecastCashReadinessV2(
		rows,
		now,
		clock.NextTimesheetPayCycle(now),
		42,
		config.CashForecastConfig{NSim: 1000, GrowthEWMAlpha: 0.5},
	)

	if projection.expectedFuture != 800 {
		t.Fatalf("expected future=%d, want recent-workforce EWMA level 8 * 100 = 800", projection.expectedFuture)
	}
}

func TestCashReadinessV2_NoTargetRowsBootstrapFullCompletedTotals(t *testing.T) {
	rows := make([]domain.TimesheetAccrualDailyRow, 0, 20)
	for employee := 1; employee <= 10; employee++ {
		// These rows already existed at the analogous cycle day. A partial-cycle
		// model sees no remaining future amount, but an entirely empty target Ky
		// must bootstrap the completed total instead of returning zero.
		rows = append(rows,
			forecastRow("2026-05-01", "2026-05-01", "2026-05-02", domain.TimesheetStatusApproved, uint(employee), 10, 100),
			forecastRow("2026-06-01", "2026-06-01", "2026-06-02", domain.TimesheetStatusApproved, uint(employee), 10, 100),
		)
	}

	now, _ := time.ParseInLocation("2006-01-02", "2026-07-17", clock.DefaultLocation)
	projection := forecastCashReadinessV2(rows, now, clock.NextTimesheetPayCycle(now), 42, config.CashForecastConfig{NSim: 1000})

	if projection.expectedFuture != 1_000 || projection.expectedPayout != 1_000 {
		t.Fatalf("empty-target projection=%#v, want full completed-cycle total 1000", projection)
	}
}

func TestCashReadinessV2_SparseTargetKeepsCompletedCycleScale(t *testing.T) {
	rows := make([]domain.TimesheetAccrualDailyRow, 0, 21)
	for employee := 1; employee <= 10; employee++ {
		rows = append(rows,
			forecastRow("2026-05-15", "2026-05-23", "2026-05-24", domain.TimesheetStatusApproved, uint(employee), 10, 100),
			forecastRow("2026-06-15", "2026-06-23", "2026-06-24", domain.TimesheetStatusApproved, uint(employee), 10, 100),
		)
	}
	// The first current-cycle import must not collapse the forecast from the
	// recent ten-person completed-cycle scale to this one observed employee.
	rows = append(rows,
		forecastRow("2026-07-15", "2026-07-22", "", domain.TimesheetStatusPendingApproval, 1, 10, 100),
	)

	now, _ := time.ParseInLocation("2006-01-02", "2026-07-22", clock.DefaultLocation)
	projection := forecastCashReadinessV2(
		rows,
		now,
		clock.NextTimesheetPayCycle(now),
		42,
		config.CashForecastConfig{NSim: 1000},
	)

	if projection.expectedFuture != 900 || projection.expectedPayout != 950 || projection.recommendedReserve != 1_000 {
		t.Fatalf("sparse-target projection=%#v, want future=900 expected=950 reserve=1000", projection)
	}
	if projection.method != "completed-cycle-bootstrap" {
		t.Fatalf("method=%q, want completed-cycle-bootstrap", projection.method)
	}
}

func TestCashReadinessV2_SparseTargetUsesCompletedTotalsWithMixedHistory(t *testing.T) {
	rows := make([]domain.TimesheetAccrualDailyRow, 0, 21)
	for employee := 1; employee <= 10; employee++ {
		rows = append(rows,
			// This cycle has a valid partial shape at the analogous forecast day.
			forecastRow("2026-05-15", "2026-05-15", "2026-05-16", domain.TimesheetStatusApproved, uint(employee), 10, 100),
			// This cycle was entered only after the analogous forecast day.
			forecastRow("2026-06-15", "2026-06-23", "2026-06-24", domain.TimesheetStatusApproved, uint(employee), 10, 100),
		)
	}
	rows = append(rows,
		forecastRow("2026-07-15", "2026-07-22", "", domain.TimesheetStatusPendingApproval, 1, 10, 100),
	)

	now, _ := time.ParseInLocation("2006-01-02", "2026-07-22", clock.DefaultLocation)
	projection := forecastCashReadinessV2(
		rows,
		now,
		clock.NextTimesheetPayCycle(now),
		42,
		config.CashForecastConfig{NSim: 1000},
	)

	if projection.expectedFuture != 900 || projection.expectedPayout != 950 || projection.recommendedReserve != 1_000 {
		t.Fatalf("mixed-history sparse projection=%#v, want future=900 expected=950 reserve=1000", projection)
	}
	if projection.method != "completed-cycle-bootstrap" {
		t.Fatalf("method=%q, want completed-cycle-bootstrap while target participation is sparse", projection.method)
	}
}

func TestGetCashReadinessV2_CalibrationAndSnapshotAreAdvisory(t *testing.T) {
	measurement := &fakeCashMeasurement{
		accuracy:  domain.CashForecastAccuracy{SampleCount: 24, WAPE: .08, Bias: .02, IntervalCoverage: .90, ReserveShortfallRate: .05},
		upsertErr: errors.New("measurement unavailable"),
	}
	for i := 0; i < 24; i++ {
		measurement.residuals = append(measurement.residuals, domain.CashForecastResidual{ResidualAmount: 10})
	}
	ts := &fakeTimesheetReader{cohort: []domain.TimesheetAccrualDailyRow{
		forecastRow("2026-07-02", "2026-07-01", "2026-07-02", domain.TimesheetStatusApproved, 1, 10, 100),
	}}
	svc := NewCashReadinessForecastService(ts, &fakeWallet{balance: &wallet.WalletBalance{}}, nil, clock.NewFake(july3), config.CashForecastConfig{}, nil, measurement)
	got, err := svc.GetCashReadiness(context.Background(), domain.TimesheetFilters{})
	if err != nil {
		t.Fatalf("snapshot failure blanked advisory forecast: %v", err)
	}
	if got.ReliabilityState != "measured" || got.Confidence != "high" || got.CalibrationSamples != 24 {
		t.Fatalf("calibration labels=%q/%q samples=%d", got.ReliabilityState, got.Confidence, got.CalibrationSamples)
	}
	if len(measurement.upserts) != 1 {
		t.Fatalf("company forecast snapshot upserts=%d, want 1", len(measurement.upserts))
	}
	if got.RecommendedReserve < got.ExpectedPayout || got.ExpectedPayout < got.ObservedApproved {
		t.Fatalf("calibrated floors violated: reserve=%d expected=%d approved=%d", got.RecommendedReserve, got.ExpectedPayout, got.ObservedApproved)
	}

	projectID := uint(10)
	_, err = svc.GetCashReadiness(context.Background(), domain.TimesheetFilters{ProjectIDs: []uint{projectID}})
	if err != nil {
		t.Fatalf("filtered advisory: %v", err)
	}
	if len(measurement.upserts) != 1 {
		t.Fatalf("filtered forecast must not persist company snapshot; got %d upserts", len(measurement.upserts))
	}
}

func TestGetCashReadinessV2_PreservesLegacyPercentileContract(t *testing.T) {
	rows := []domain.TimesheetAccrualDailyRow{
		forecastRow("2026-07-02", "2026-07-01", "2026-07-02", domain.TimesheetStatusApproved, 1, 10, 100),
		forecastRow("2026-06-08", "2026-06-14", "2026-06-15", domain.TimesheetStatusApproved, 2, 10, 300),
	}
	svc := NewCashReadinessForecastService(
		&fakeTimesheetReader{cohort: rows},
		&fakeWallet{balance: &wallet.WalletBalance{}},
		nil,
		clock.NewFake(july3),
		config.CashForecastConfig{NSim: 1000},
		nil,
	)
	got, err := svc.GetCashReadiness(context.Background(), domain.TimesheetFilters{})
	if err != nil {
		t.Fatalf("forecast: %v", err)
	}
	if got.BandLower != got.ObservedApproved+got.ProjectedP50 || got.BandUpper != got.ObservedApproved+got.ProjectedP95 {
		t.Fatalf("legacy band contract changed: %#v", got)
	}
	if got.CashToPrepare != got.BandLower {
		t.Fatalf("legacy cash_to_prepare=%d, want p50 total %d", got.CashToPrepare, got.BandLower)
	}
	if got.RecommendedReserve < got.ExpectedPayout {
		t.Fatalf("new reserve contract violated: %#v", got)
	}
}

type fixedWeeklyPaymentPercentage float64

func (p fixedWeeklyPaymentPercentage) GetWeeklyPaymentPercentage(context.Context) float64 {
	return float64(p)
}

func TestGetCashReadinessV2_UsesWeeklyPayableCashPercentage(t *testing.T) {
	rows := []domain.TimesheetAccrualDailyRow{
		forecastRow("2026-07-02", "2026-07-01", "2026-07-02", domain.TimesheetStatusApproved, 1, 10, 100),
	}
	svc := NewCashReadinessForecastService(
		&fakeTimesheetReader{cohort: rows},
		&fakeWallet{balance: &wallet.WalletBalance{}},
		nil,
		clock.NewFake(july3),
		config.CashForecastConfig{},
		fixedWeeklyPaymentPercentage(0.7),
	)

	got, err := svc.GetCashReadiness(context.Background(), domain.TimesheetFilters{})
	if err != nil {
		t.Fatalf("forecast: %v", err)
	}
	if got.ObservedApproved != 70 || got.CashToPrepare != 70 || got.ExpectedPayout != 70 {
		t.Fatalf("payable cash amounts = approved %d, prepare %d, expected %d; want 70 each", got.ObservedApproved, got.CashToPrepare, got.ExpectedPayout)
	}
	if got.ModelVersion != "cash-readiness-v5" {
		t.Fatalf("model version=%q, want v5 after mixed-history sparse-target correction", got.ModelVersion)
	}
}

func TestReliabilityLabels_LearningDoesNotClaimMediumConfidence(t *testing.T) {
	reliability, confidence := reliabilityLabels(domain.CashForecastAccuracy{SampleCount: 12}, config.CashForecastConfig{})
	if reliability != "learning" || confidence != "low" {
		t.Fatalf("labels=%q/%q, want learning/low", reliability, confidence)
	}
}
