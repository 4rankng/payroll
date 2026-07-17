package services

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"api-server/internal/config"
	"api-server/internal/domain"
	"api-server/internal/domain/wallet"
	"api-server/internal/pkg/clock"
)

// Cash-readiness forecast for the next timesheet bulk transfer.
//
// Forecasts the final approved value of the next Ky only. It combines approved
// value already observed inside the target Ky with a short-horizon projection
// of approvals still expected before that Ky's pay date, then subtracts the
// live wallet balance to surface a single "cash to prepare" gap. Outstanding
// payments from earlier cycles are intentionally excluded.
//
// ADVISORY ONLY: the service holds only READ-ONLY ports. It has no handle to
// SyncBalance, CreateTopup, or any disbursement, so feeding it back into balance
// sync is structurally impossible — this prevents re-introducing the
// wallet-balance-inflation bug.

// ForecastProvider projects additional approved-pay accrual between two
// cycle-day positions from a historical per-Ky cohort. The v1 implementation is
// a statistical baseline (newsvendor Monte-Carlo); a future Prophet/LightGBM
// provider implements the same interface with no service/handler/UI change.
type ForecastProvider interface {
	ProjectAccrual(ctx context.Context, hist []cohortSeries, fromCycleDay, throughCycleDay int, seed int64, cfg config.CashForecastConfig) (AccrualProjection, error)
}

// AccrualProjection is a provider's projection of additional accrual between the
// observed cycle-day and the horizon (pay date).
type AccrualProjection struct {
	P50         int64   // median additional accrual
	Expected    int64   // arithmetic mean additional accrual
	P95         int64   // tail additional accrual
	Method      string  // "monte-carlo" | "gamma-fit" | "no-history" | "growth-adjusted"
	Confidence  string  // "high" | "medium" | "low"
	BasisCycles int     // number of historical cycles used
	GrowthRate  float64 // EWMA growth factor applied (1.0 when not trending; >1 = upward trend)
}

// TimesheetAccrualProvider is the v1 statistical ForecastProvider. It delegates
// to the existing newsvendor Monte-Carlo engine (wallet_demand_forecast_math.go)
// — no math is reimplemented.
type TimesheetAccrualProvider struct{}

// NewTimesheetAccrualProvider returns the v1 statistical provider.
func NewTimesheetAccrualProvider() *TimesheetAccrualProvider { return &TimesheetAccrualProvider{} }

// ProjectAccrual fits the per-cycle accrual distribution between fromCycleDay
// and throughCycleDay from the historical cohort and reads its p50 / p95. With
// no usable history the distribution is a point mass at 0 (method "no-history"),
// collapsing the band to the confirmed floor.
func (p *TimesheetAccrualProvider) ProjectAccrual(_ context.Context, hist []cohortSeries, fromCycleDay, throughCycleDay int, seed int64, cfg config.CashForecastConfig) (AccrualProjection, error) {
	nSim := cfg.NSim
	if nSim <= 0 {
		nSim = 5000
	}
	// paidFrac = 1: the accrual cohort already expresses cash that will leave
	// the wallet (approved pay), so no completion-fraction scaling is needed.
	dist := forecastDemandDistributionBetween(hist, fromCycleDay, throughCycleDay, nSim, seed, 1.0)
	return AccrualProjection{
		P50:         int64(math.Round(dist.p50)),
		Expected:    int64(math.Round(meanF(dist.samples))),
		P95:         int64(math.Round(dist.p95)),
		Method:      dist.method,
		Confidence:  confidenceLabel(dist),
		BasisCycles: dist.basisPeriods,
		GrowthRate:  dist.growthFactor,
	}, nil
}

// TimesheetAccrualReader reads the historical accrual cohort. Backed by the
// timesheet repository (domain.TimesheetRepository satisfies it structurally).
type TimesheetAccrualReader interface {
	GetAccrualCohort(ctx context.Context, filters domain.TimesheetFilters) ([]domain.TimesheetAccrualDailyRow, error)
}

// WalletBalanceReader is the read-only wallet port. Defined narrow on purpose so
// the forecast can call GetBalance but can never reach SyncBalance/CreateTopup.
type WalletBalanceReader interface {
	GetBalance(ctx context.Context) (*wallet.WalletBalance, error)
}

type WeeklyPaymentPercentageReader interface {
	GetWeeklyPaymentPercentage(ctx context.Context) float64
}

// CashReadinessForecastService composes target-Ky observed approvals + projected
// target-Ky approvals − wallet balance into the cash-prep gap.
type CashReadinessForecastService struct {
	timesheetRepo TimesheetAccrualReader
	walletSvc     WalletBalanceReader
	provider      ForecastProvider
	clock         clock.Clock
	cfg           config.CashForecastConfig
	paymentConfig WeeklyPaymentPercentageReader
	measurement   domain.CashForecastSnapshotRepository
}

// NewCashReadinessForecastService constructs the service. clk and provider
// default to the real clock / statistical provider when nil.
func NewCashReadinessForecastService(
	timesheetRepo TimesheetAccrualReader,
	walletSvc WalletBalanceReader,
	provider ForecastProvider,
	clk clock.Clock,
	cfg config.CashForecastConfig,
	paymentConfig WeeklyPaymentPercentageReader,
	measurement ...domain.CashForecastSnapshotRepository,
) *CashReadinessForecastService {
	if clk == nil {
		clk = clock.New()
	}
	if provider == nil {
		provider = NewTimesheetAccrualProvider()
	}
	service := &CashReadinessForecastService{
		timesheetRepo: timesheetRepo,
		walletSvc:     walletSvc,
		provider:      provider,
		clock:         clk,
		cfg:           cfg,
		paymentConfig: paymentConfig,
	}
	if len(measurement) > 0 {
		service.measurement = measurement[0]
	}
	return service
}

// GetCashReadiness builds the target-Ky forecast for the next pay date. Filters
// scope the cohort by project/employee/role, but the current outstanding-payment
// summary is deliberately not an input.
func (s *CashReadinessForecastService) GetCashReadiness(ctx context.Context, filters domain.TimesheetFilters) (*domain.CashReadiness, error) {
	now := s.clock.Now()
	pc := clock.NextTimesheetPayCycle(now)
	leadDays := s.leadDays()

	// Read row-level point-in-time facts. Status is deliberately not constrained:
	// pending and rejected outcomes are needed for beta-smoothed conversion.
	cohortFilters := s.cohortFilters(filters, now)
	rows, err := s.timesheetRepo.GetAccrualCohort(ctx, cohortFilters)
	if err != nil {
		return nil, fmt.Errorf("lấy cohort accrual: %w", err)
	}
	currentForMonth := pc.WorkMonth.Format("2006-01")
	seed := timesheetForecastSeed(pc.Ky, currentForMonth, pc.CycleDayToday)
	projection := forecastCashReadinessV2(rows, now, pc, seed, s.cfg)
	scaleCashReadinessProjection(&projection, s.weeklyPaymentPercentage(ctx))
	horizonDays := daysBetweenDates(now, pc.NextPayDate)
	accuracy := s.calibrate(ctx, &projection, horizonDays, isScopedCashForecast(filters))

	walletAvailable, walletOK := s.walletAvailable(ctx)
	legacyCashToPrepare := projection.legacyMedian
	gap := max(int64(0), legacyCashToPrepare-walletAvailable)
	reliability, confidence := reliabilityLabels(accuracy, s.cfg)

	result := &domain.CashReadiness{
		ObservedApproved:      projection.approved,
		ProjectedP50:          max(int64(0), projection.legacyMedian-projection.approved),
		ProjectedExpected:     max(int64(0), projection.legacyExpected-projection.approved),
		ProjectedP95:          max(int64(0), projection.legacyP95-projection.approved),
		BandLower:             projection.legacyMedian,
		BandUpper:             projection.legacyP95,
		ExpectedTotal:         projection.legacyExpected,
		PendingTargetAmount:   projection.pending,
		ExpectedPendingAmount: projection.expectedPending,
		ExpectedFutureAmount:  projection.expectedFuture,
		ExpectedPayout:        projection.expectedPayout,
		RecommendedReserve:    projection.recommendedReserve,
		IntervalLower:         projection.intervalLower,
		IntervalUpper:         projection.intervalUpper,
		ModelVersion:          cashReadinessModelVersion,
		CalibrationSamples:    accuracy.SampleCount,
		ReliabilityState:      reliability,
		AccuracyWAPE:          accuracy.WAPE,
		AccuracyBias:          accuracy.Bias,
		IntervalCoverage:      accuracy.IntervalCoverage,
		ReserveShortfallRate:  accuracy.ReserveShortfallRate,
		WalletAvailable:       walletAvailable,
		WalletAvailableOK:     walletOK,
		CashToPrepare:         legacyCashToPrepare,
		Gap:                   gap,
		PrepareByDate:         pc.PrepareByDate(leadDays),
		NextPayDate:           pc.NextPayDate,
		LeadDays:              leadDays,
		Ky:                    pc.Ky,
		CycleDayToday:         pc.CycleDayToday,
		Method:                projection.method,
		Confidence:            confidence,
		BasisCycles:           projection.basisCycles,
		GrowthRate:            1,
		GeneratedAt:           now,
	}
	s.persistMeasurement(ctx, filters, result, pc, horizonDays)
	return result, nil
}

func (s *CashReadinessForecastService) weeklyPaymentPercentage(ctx context.Context) float64 {
	if s.paymentConfig == nil {
		return 1
	}
	percentage := s.paymentConfig.GetWeeklyPaymentPercentage(ctx)
	if math.IsNaN(percentage) || math.IsInf(percentage, 0) || percentage <= 0 || percentage > 1 {
		slog.Warn("cash forecast weekly payment percentage invalid; using full amount", "value", percentage)
		return 1
	}
	return percentage
}

func scaleCashReadinessProjection(projection *cashReadinessV2Projection, percentage float64) {
	if projection == nil || percentage == 1 {
		return
	}
	scale := func(value int64) int64 { return int64(float64(value) * percentage) }
	projection.approved = scale(projection.approved)
	projection.pending = scale(projection.pending)
	projection.expectedPending = scale(projection.expectedPending)
	projection.expectedFuture = scale(projection.expectedFuture)
	projection.expectedPayout = scale(projection.expectedPayout)
	projection.recommendedReserve = scale(projection.recommendedReserve)
	projection.intervalLower = scale(projection.intervalLower)
	projection.intervalUpper = scale(projection.intervalUpper)
	projection.legacyMedian = scale(projection.legacyMedian)
	projection.legacyExpected = scale(projection.legacyExpected)
	projection.legacyP95 = scale(projection.legacyP95)
}

// cohortFilters preserves role/project/employee scope but overrides the status
// (approved-only) and date window (lookback) for the cohort basis. Payment
// status is intentionally NOT filtered — see GetAccrualCohort.
func (s *CashReadinessForecastService) cohortFilters(filters domain.TimesheetFilters, now time.Time) domain.TimesheetFilters {
	months := s.historyMonths()
	from := now.AddDate(0, -months, 0)
	return domain.TimesheetFilters{
		EmployeeCreatedBy:         filters.EmployeeCreatedBy,
		EmployeeAssignedByPartner: filters.EmployeeAssignedByPartner,
		ProjectIDs:                filters.ProjectIDs,
		EmployeeID:                filters.EmployeeID,
		EmployeeIDs:               filters.EmployeeIDs,
		FromDate:                  &from,
		ToDate:                    &now,
	}
}

func (s *CashReadinessForecastService) calibrate(ctx context.Context, projection *cashReadinessV2Projection, horizonDays int, scoped bool) domain.CashForecastAccuracy {
	// Company-wide exports are not a valid outcome for filtered forecasts.
	if s.measurement == nil || scoped {
		return domain.CashForecastAccuracy{}
	}
	query := domain.CashForecastResolvedQuery{
		ScopeKey: domain.CashForecastCompanyScope, ModelVersion: cashReadinessModelVersion,
		HorizonDays: horizonDays, Limit: 52,
	}
	accuracy, err := s.measurement.GetAccuracy(ctx, query)
	if err != nil {
		slog.Warn("cash forecast accuracy unavailable", "error", err, "horizon_days", horizonDays)
		return domain.CashForecastAccuracy{}
	}
	if accuracy == nil {
		accuracy = &domain.CashForecastAccuracy{}
	}
	if accuracy.SampleCount < 12 {
		return *accuracy
	}
	residuals, err := s.measurement.ListResolvedResiduals(ctx, query)
	if err != nil || len(residuals) == 0 {
		if err != nil {
			slog.Warn("cash forecast residual calibration unavailable", "error", err, "horizon_days", horizonDays)
		}
		return *accuracy
	}
	values := make([]int64, 0, len(residuals))
	var total int64
	for _, residual := range residuals {
		values = append(values, residual.ResidualAmount)
		total += residual.ResidualAmount
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	meanResidual := int64(math.Round(float64(total) / float64(len(values))))
	projection.expectedFuture = max(int64(0), projection.expectedFuture+meanResidual)
	projection.expectedPayout = max(projection.approved, projection.approved+projection.expectedPending+projection.expectedFuture)
	projection.intervalLower = max(projection.approved, projection.intervalLower+cashQuantile(values, 0.05))
	projection.intervalUpper = max(projection.intervalLower, projection.intervalUpper+cashQuantile(values, 0.95))
	serviceLevel := s.cfg.ServiceLevel
	if serviceLevel <= 0 {
		serviceLevel = 0.95
	}
	projection.recommendedReserve = max(projection.expectedPayout, projection.recommendedReserve+cashQuantile(values, serviceLevel))
	projection.intervalUpper = max(projection.intervalUpper, projection.expectedPayout)
	return *accuracy
}

func (s *CashReadinessForecastService) persistMeasurement(ctx context.Context, filters domain.TimesheetFilters, result *domain.CashReadiness, pc clock.TimesheetPayCycle, horizonDays int) {
	if s.measurement == nil || isScopedCashForecast(filters) {
		return
	}
	fromDate := time.Date(pc.WorkMonth.Year(), pc.WorkMonth.Month(), clock.WorkStartDay(pc.Ky), 0, 0, 0, 0, clock.DefaultLocation)
	snapshot := &domain.CashForecastSnapshot{
		ScopeKey: domain.CashForecastCompanyScope,
		CycleKey: cashCycleKey(pc.WorkMonth.Format("2006-01"), pc.Ky),
		CycleDay: pc.CycleDayToday, ModelVersion: cashReadinessModelVersion,
		TargetFromDate: fromDate, TargetToDate: fromDate.AddDate(0, 0, 6), HorizonDays: horizonDays,
		ObservedApprovedAmount: result.ObservedApproved, PendingAmount: result.PendingTargetAmount,
		ExpectedPendingAmount: result.ExpectedPendingAmount, ExpectedFutureAmount: result.ExpectedFutureAmount,
		ExpectedPayoutAmount: result.ExpectedPayout, RecommendedReserveAmount: result.RecommendedReserve,
		IntervalLowerAmount: result.IntervalLower, IntervalUpperAmount: result.IntervalUpper,
		GeneratedAt: result.GeneratedAt,
	}
	if err := s.measurement.Upsert(ctx, snapshot); err != nil {
		slog.Warn("cash forecast snapshot write failed", "error", err, "cycle_key", snapshot.CycleKey, "cycle_day", snapshot.CycleDay)
	}
}

func isScopedCashForecast(filters domain.TimesheetFilters) bool {
	return filters.EmployeeCreatedBy != nil || filters.EmployeeAssignedByPartner != nil ||
		len(filters.ProjectIDs) > 0 || filters.EmployeeID != nil || len(filters.EmployeeIDs) > 0
}

func daysBetweenDates(from, to time.Time) int {
	fromDate := time.Date(from.In(clock.DefaultLocation).Year(), from.In(clock.DefaultLocation).Month(), from.In(clock.DefaultLocation).Day(), 0, 0, 0, 0, time.UTC)
	toDate := time.Date(to.In(clock.DefaultLocation).Year(), to.In(clock.DefaultLocation).Month(), to.In(clock.DefaultLocation).Day(), 0, 0, 0, 0, time.UTC)
	return max(0, int(toDate.Sub(fromDate)/(24*time.Hour)))
}

func reliabilityLabels(accuracy domain.CashForecastAccuracy, cfg config.CashForecastConfig) (string, string) {
	if accuracy.SampleCount < 12 {
		return "uncalibrated", "low"
	}
	if accuracy.SampleCount < 24 {
		return "learning", "low"
	}
	wapeThreshold := defaultPositive(cfg.WAPEThreshold, 0.10)
	biasThreshold := defaultPositive(cfg.AbsBiasThreshold, 0.03)
	shortfallThreshold := defaultPositive(cfg.ReserveShortfallThreshold, 0.10)
	coverageMin := defaultPositive(cfg.CoverageMin, 0.85)
	coverageMax := defaultPositive(cfg.CoverageMax, 0.95)
	reliable := accuracy.WAPE <= wapeThreshold && math.Abs(accuracy.Bias) <= biasThreshold &&
		accuracy.ReserveShortfallRate <= shortfallThreshold &&
		accuracy.IntervalCoverage >= coverageMin && accuracy.IntervalCoverage <= coverageMax
	if reliable {
		return "measured", "high"
	}
	return "measured", "low"
}

func defaultPositive(value, fallback float64) float64 {
	if value > 0 {
		return value
	}
	return fallback
}

// observedApprovedForCycle sums approvals already recorded for the target Ky up
// to the current cycle-day. The repository input is approved-only, so pending
// approvals and outstanding payments from other cycles cannot enter this floor.
func observedApprovedForCycle(rows []domain.TimesheetAccrualDailyRow, targetKy int, currentForMonth string, throughCycleDay int) int64 {
	var total int64
	for _, r := range rows {
		if clock.KyFromWorkDay(r.WorkDate.Day()) != targetKy || r.WorkDate.Format("2006-01") != currentForMonth {
			continue
		}
		cycleDay := clock.CycleDayForApproval(targetKy, r.WorkDate.Year(), r.WorkDate.Month(), r.ApprovedDate)
		if cycleDay >= 1 && cycleDay <= throughCycleDay {
			total += r.Amount
		}
	}
	return total
}

func (s *CashReadinessForecastService) walletAvailable(ctx context.Context) (int64, bool) {
	bal, err := s.walletSvc.GetBalance(ctx)
	if err != nil || bal == nil {
		return 0, false
	}
	return bal.Available, true
}

func (s *CashReadinessForecastService) leadDays() int {
	if s.cfg.LeadDays > 0 {
		return s.cfg.LeadDays
	}
	return 2
}

func (s *CashReadinessForecastService) historyMonths() int {
	if s.cfg.HistoryMonths >= 3 {
		return s.cfg.HistoryMonths
	}
	return 6
}

// buildTimesheetCohort pivots daily accrual rows into per-Ky historical
// cohortSeries for the target Ky, excluding the in-progress current cycle. It
// mirrors pivotCohort's cumulative-building but derives the cycle-day from the
// approval date via the clock pay-cycle model (cycle math stays a single
// testable source of truth in pkg/clock).
func buildTimesheetCohort(rows []domain.TimesheetAccrualDailyRow, targetKy int, currentForMonth string) []cohortSeries {
	type acc struct {
		daily map[int]int64
		grand int64
		year  int
		month time.Month
	}
	byMonth := make(map[string]*acc)
	for _, r := range rows {
		ky := clock.KyFromWorkDay(r.WorkDate.Day())
		if ky != targetKy {
			continue
		}
		forMonth := r.WorkDate.Format("2006-01")
		if forMonth == currentForMonth {
			continue
		}
		cycleDay := clock.CycleDayForApproval(ky, r.WorkDate.Year(), r.WorkDate.Month(), r.ApprovedDate)
		if cycleDay < 1 {
			continue
		}
		a, ok := byMonth[forMonth]
		if !ok {
			a = &acc{daily: make(map[int]int64), year: r.WorkDate.Year(), month: r.WorkDate.Month()}
			byMonth[forMonth] = a
		}
		a.daily[cycleDay] += r.Amount
		a.grand += r.Amount
	}

	series := make([]cohortSeries, 0, len(byMonth))
	for forMonth, a := range byMonth {
		maxDay := clock.MaxCycleDay(targetKy, a.year, a.month)
		if maxDay < 1 {
			maxDay = 10
		}
		s := cohortSeries{
			forMonth:    forMonth,
			isCurrent:   false,
			maxCycleDay: maxDay,
			dailyAmount: a.daily,
			grandTotal:  a.grand,
		}
		s.cumulative = make(map[int]int64, maxDay)
		var run int64
		for d := 1; d <= maxDay; d++ {
			run += s.dailyAmount[d]
			s.cumulative[d] = run
		}
		series = append(series, s)
	}
	// Chronological sort is REQUIRED: the growth EWMA (growthEWMA) and any
	// index-based trend math depend on series[0] = oldest month. Go map
	// iteration is randomized, so without this sort the same request could
	// return different growth factors across process restarts.
	sort.Slice(series, func(i, j int) bool {
		return series[i].forMonth < series[j].forMonth
	})
	return series
}

// timesheetForecastSeed derives a deterministic seed from the Ky, work month,
// and cycle day so the same request returns stable numbers within a cycle day
// (no UI flicker on refetch). Mirrors forecastSeed's role for the advance cycle.
func timesheetForecastSeed(ky int, forMonth string, cycleDay int) int64 {
	m, err := clock.ParseMonth(forMonth)
	if err != nil {
		return int64(cycleDay)
	}
	return int64(m.Year())*100000 + int64(m.Month())*1000 + int64(ky)*100 + int64(cycleDay)
}
