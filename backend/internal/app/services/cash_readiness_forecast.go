package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"api-server/internal/config"
	"api-server/internal/domain"
	"api-server/internal/domain/wallet"
	"api-server/internal/pkg/clock"
)

// Cash-readiness forecast for the next timesheet bulk transfer.
//
// Composes the deterministic confirmed-payable floor (already-approved +
// pending-approval, not-yet-paid — identical to GetSummaryStats) with a
// short-horizon projected-accrual band derived from a per-Ky timesheet cohort,
// then subtracts the live wallet balance to surface a single "cash to prepare"
// gap. Reuses the I/O-free newsvendor Monte-Carlo engine (the same one the
// wallet demand forecast uses) on a NEW timesheet-accrual cohort — the wallet
// forecast's advance-payment cohort is the wrong data source for this problem.
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
	P50         int64  // expected additional accrual
	P95         int64  // tail additional accrual
	Method      string // "monte-carlo" | "gamma-fit" | "no-history"
	Confidence  string // "high" | "medium" | "low"
	BasisCycles int    // number of historical cycles used
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
		P95:         int64(math.Round(dist.p95)),
		Method:      dist.method,
		Confidence:  confidenceLabel(dist),
		BasisCycles: dist.basisPeriods,
	}, nil
}

// TimesheetSummaryReader reads the confirmed-payable figure. Backed by
// TimesheetService.GetSummaryStats (the CACHED path) so the forecast shares the
// exact value — and the same mutation invalidation — the "Chờ thanh toán" card
// shows. Routing through the cached service (not the raw repo) is what keeps the
// two figures from diverging inside the summary cache TTL.
type TimesheetSummaryReader interface {
	GetSummaryStats(ctx context.Context, filters domain.TimesheetFilters) (*domain.TimesheetSummaryStats, error)
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

// CashReadinessForecastService composes confirmed-payable + projected accrual −
// wallet balance into the cash-prep gap.
type CashReadinessForecastService struct {
	summaryReader TimesheetSummaryReader
	timesheetRepo TimesheetAccrualReader
	walletSvc     WalletBalanceReader
	provider      ForecastProvider
	clock         clock.Clock
	cfg           config.CashForecastConfig
}

// NewCashReadinessForecastService constructs the service. summaryReader should be
// the cached TimesheetService (so confirmed-payable matches the summary card
// exactly); timesheetRepo is the cohort reader. clk and provider default to the
// real clock / statistical provider when nil.
func NewCashReadinessForecastService(
	summaryReader TimesheetSummaryReader,
	timesheetRepo TimesheetAccrualReader,
	walletSvc WalletBalanceReader,
	provider ForecastProvider,
	clk clock.Clock,
	cfg config.CashForecastConfig,
) *CashReadinessForecastService {
	if clk == nil {
		clk = clock.New()
	}
	if provider == nil {
		provider = NewTimesheetAccrualProvider()
	}
	return &CashReadinessForecastService{
		summaryReader: summaryReader,
		timesheetRepo: timesheetRepo,
		walletSvc:     walletSvc,
		provider:      provider,
		clock:         clk,
		cfg:           cfg,
	}
}

// GetCashReadiness builds the cash-prep forecast for the next pay cycle, scoped
// to the same filters as the summary endpoint (so confirmed-payable matches the
// "Chờ thanh toán" card exactly).
func (s *CashReadinessForecastService) GetCashReadiness(ctx context.Context, filters domain.TimesheetFilters) (*domain.CashReadiness, error) {
	now := s.clock.Now()
	pc := clock.NextTimesheetPayCycle(now)
	leadDays := s.leadDays()

	// 1. Confirmed-payable: the SAME cached source as the summary card.
	summary, err := s.summaryReader.GetSummaryStats(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("lấy confirmed-payable: %w", err)
	}
	confirmed := summary.PendingPaymentAmount

	// 2. Build the per-Ky historical cohort (excluding the in-progress cycle).
	cohortFilters := s.cohortFilters(filters, now)
	rows, err := s.timesheetRepo.GetAccrualCohort(ctx, cohortFilters)
	if err != nil {
		return nil, fmt.Errorf("lấy cohort accrual: %w", err)
	}
	currentForMonth := pc.WorkMonth.Format("2006-01")
	historical := buildTimesheetCohort(rows, pc.Ky, currentForMonth)

	// 3. Project accrual between today's cycle-day and the pay date.
	seed := timesheetForecastSeed(pc.Ky, currentForMonth, pc.CycleDayToday)
	proj, err := s.provider.ProjectAccrual(ctx, historical, pc.CycleDayToday, pc.MaxCycleDay, seed, s.cfg)
	if err != nil {
		return nil, fmt.Errorf("dự báo accrual: %w", err)
	}

	// 4. Compose the band + gap.
	cashToPrepare := confirmed + proj.P50
	bandLower := confirmed + proj.P50
	bandUpper := confirmed + proj.P95

	walletAvailable, walletOK := s.walletAvailable(ctx)
	gap := max(int64(0), cashToPrepare-walletAvailable)

	return &domain.CashReadiness{
		ConfirmedPayable:  confirmed,
		ProjectedP50:      proj.P50,
		ProjectedP95:      proj.P95,
		BandLower:         bandLower,
		BandUpper:         bandUpper,
		WalletAvailable:   walletAvailable,
		WalletAvailableOK: walletOK,
		CashToPrepare:     cashToPrepare,
		Gap:               gap,
		PrepareByDate:     pc.PrepareByDate(leadDays),
		NextPayDate:       pc.NextPayDate,
		LeadDays:          leadDays,
		Ky:                pc.Ky,
		CycleDayToday:     pc.CycleDayToday,
		Method:            proj.Method,
		Confidence:        proj.Confidence,
		BasisCycles:       proj.BasisCycles,
		GeneratedAt:       now,
	}, nil
}

// cohortFilters mirrors GetSummary's role/project/employee scope but overrides
// the status (approved-only) and date window (lookback) for the cohort basis.
// Payment status is intentionally NOT filtered — see GetAccrualCohort.
func (s *CashReadinessForecastService) cohortFilters(filters domain.TimesheetFilters, now time.Time) domain.TimesheetFilters {
	months := s.historyMonths()
	from := now.AddDate(0, -months, 0)
	return domain.TimesheetFilters{
		EmployeeCreatedBy:         filters.EmployeeCreatedBy,
		EmployeeAssignedByPartner: filters.EmployeeAssignedByPartner,
		ProjectIDs:                filters.ProjectIDs,
		EmployeeID:                filters.EmployeeID,
		EmployeeIDs:               filters.EmployeeIDs,
		TimesheetStatus:           []domain.TimesheetStatus{domain.TimesheetStatusApproved},
		FromDate:                  &from,
		ToDate:                    &now,
	}
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
