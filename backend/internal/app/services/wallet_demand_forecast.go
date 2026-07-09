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

const (
	defaultWalletForecastLeadDays = 2
	walletDemandChartPeriodCount  = 3
)

// WalletDemandForecastService produces the advance-payment demand cohort chart
// data and the wallet-balance prediction for the Wallet page.
//
// ADVISORY ONLY: the prediction is display-only. It reads walletSvc.GetBalance
// for the current Available figure but MUST NOT call SyncBalance, CreateTopup,
// the disbursement poller, or any auto top-up. Feeding it back into balance sync
// would re-introduce the wallet-balance-inflation bug.
type WalletDemandForecastService struct {
	requestRepo domain.AdvancePaymentRequestRepository
	walletSvc   wallet.WalletService
	clock       clock.Clock
	cfg         config.WalletForecastConfig
}

// NewWalletDemandForecastService constructs the forecast service. clk defaults to
// the real clock when nil.
func NewWalletDemandForecastService(
	requestRepo domain.AdvancePaymentRequestRepository,
	walletSvc wallet.WalletService,
	clk clock.Clock,
	cfg config.WalletForecastConfig,
) *WalletDemandForecastService {
	if clk == nil {
		clk = clock.New()
	}
	return &WalletDemandForecastService{
		requestRepo: requestRepo,
		walletSvc:   walletSvc,
		clock:       clk,
		cfg:         cfg,
	}
}

// GetDemandForecast builds the cohort series for the effective period + recent
// completed periods and the just-in-time balance prediction.
func (s *WalletDemandForecastService) GetDemandForecast(ctx context.Context) (*wallet.WalletDemandForecastResponse, error) {
	now := s.clock.Now()
	currentForMonth := forecastForMonth(now)
	maxDay := maxCycleDay(currentForMonth)
	todayCycleDay := cycleDayFor(now, currentForMonth)
	leadDays := s.leadDays()
	horizonCycleDay := forecastHorizonCycleDay(now, currentForMonth, leadDays)

	historyMonths := s.cfg.HistoryMonths
	if historyMonths < 3 {
		historyMonths = 3 // current period + at least 2 historical
	}
	forMonths := make([]string, 0, historyMonths)
	for i := 0; i < historyMonths; i++ {
		forMonths = append(forMonths, addMonths(currentForMonth, -i))
	}

	rows, err := s.requestRepo.GetCohortByMonths(ctx, forMonths)
	if err != nil {
		return nil, fmt.Errorf("lấy dữ liệu cohort ứng lương: %w", err)
	}

	series := make([]cohortSeries, 0, len(forMonths))
	byMonth := make(map[string]cohortSeries, len(forMonths))
	for i, fm := range forMonths {
		ps := pivotCohort(rows, fm, i == 0)
		series = append(series, ps)
		byMonth[fm] = ps
	}

	historical := make([]cohortSeries, 0, len(series)-1)
	for _, ps := range series {
		if !ps.isCurrent {
			historical = append(historical, ps)
		}
	}

	current := byMonth[currentForMonth]
	actualSoFar := current.cumulativeAt(todayCycleDay)
	alreadyPaid := current.completedTotal
	knownUnpaid := max(int64(0), actualSoFar-alreadyPaid)

	rate := completionRate(historical)
	paidFrac := rate
	if paidFrac <= 0 {
		// No completed history: assume remaining demand will be paid (conservative
		// — never under-recommend when the completion signal is missing).
		paidFrac = 1
	}

	nSim := s.cfg.NSim
	if nSim <= 0 {
		nSim = 5000
	}
	dist := forecastDemandDistributionBetween(
		historical, todayCycleDay, horizonCycleDay, nSim,
		forecastSeed(currentForMonth, horizonCycleDay), paidFrac,
	)

	sl := serviceLevelConfig{
		Quantile:          s.cfg.ServiceLevel,
		CostUnder:         s.cfg.CostUnder,
		CostOver:          s.cfg.CostOver,
		UncertaintyFactor: s.cfg.UncertaintyFactor,
	}
	recommended, coverage := newsvendorRecommendation(dist, sl)
	recommended += knownUnpaid

	// Legacy pace-projection (cohort-median) feeds the p50 reference fields and
	// remaining-to-pay fields for API continuity. The top-up recommendation above
	// intentionally uses the lead-window distribution, not the full-cycle reserve.
	projectionCycleDay := todayCycleDay
	if projectionCycleDay < 1 {
		projectionCycleDay = horizonCycleDay
	}
	paceTotal, _, _, _ := forecastProjectedTotal(actualSoFar, historical, projectionCycleDay)
	projectedPaid := int64(float64(paceTotal) * rate)
	if dist.method == "no-history" {
		projectedPaid = paceTotal
	}
	remainingToPay := max(int64(0), projectedPaid-alreadyPaid)

	// Reference ladder + CI band, in lead-window cash units.
	p50Ref := knownUnpaid + int64(math.Round(dist.p50))
	p90Ref := knownUnpaid + int64(math.Round(dist.p90))
	p99Ref := knownUnpaid + int64(math.Round(dist.p99))
	if dist.method == "no-history" {
		p50Ref, p90Ref, p99Ref = knownUnpaid, knownUnpaid, knownUnpaid
	}

	currentAvailable := int64(0)
	if bal, err := s.walletSvc.GetBalance(ctx); err == nil && bal != nil {
		currentAvailable = bal.Available
	}

	shortfall := recommended - currentAvailable
	surplus := int64(0)
	if shortfall < 0 {
		surplus = -shortfall
		shortfall = 0
	}

	chartMonths := forMonths
	if len(chartMonths) > walletDemandChartPeriodCount {
		chartMonths = chartMonths[:walletDemandChartPeriodCount]
	}
	periods := make([]wallet.WalletDemandPeriod, 0, len(chartMonths))
	for _, fm := range chartMonths {
		periods = append(periods, buildDemandPeriod(byMonth[fm]))
	}

	return &wallet.WalletDemandForecastResponse{
		CurrentForMonth: currentForMonth,
		CurrentCycleDay: todayCycleDay,
		MaxCycleDay:     maxDay,
		Periods:         periods,
		Prediction: wallet.WalletDemandPrediction{
			ActualSoFar:         actualSoFar,
			ProjectedTotal:      paceTotal,
			ProjectedPaid:       projectedPaid,
			AlreadyPaid:         alreadyPaid,
			RemainingToPay:      remainingToPay,
			RecommendedBalance:  recommended,
			CurrentAvailable:    currentAvailable,
			Shortfall:           shortfall,
			Surplus:             surplus,
			CompletionRate:      rate,
			Method:              dist.method,
			Confidence:          confidenceLabel(dist),
			BasisPeriods:        dist.basisPeriods,
			LeadDays:            leadDays,
			HorizonCycleDay:     horizonCycleDay,
			P50Reference:        p50Ref,
			P90Reference:        p90Ref,
			P99Reference:        p99Ref,
			CoverageProbability: coverage,
			NHistory:            dist.basisPeriods,
			ConfidenceInterval: wallet.ForecastConfidenceInterval{
				Lower: p50Ref,
				Upper: p99Ref,
			},
			ServiceLevel: wallet.ForecastServiceLevel{
				Quantile:  coverage,
				CostUnder: s.cfg.CostUnder,
				CostOver:  s.cfg.CostOver,
			},
		},
		GeneratedAt: now.Format(time.RFC3339),
	}, nil
}

func forecastForMonth(t time.Time) string {
	if clock.IsInLockedGap(t) {
		return clock.NextAdvanceMonthFromTime(t)
	}
	return clock.AdvanceMonthFromTime(t)
}

func (s *WalletDemandForecastService) leadDays() int {
	if s.cfg.LeadDays <= 0 {
		return defaultWalletForecastLeadDays
	}
	return s.cfg.LeadDays
}

// buildDemandPeriod turns a pivoted series into the JSON period payload: one
// point per cycle day in [1, maxCycleDay] with cumulative + daily amounts and a
// Vietnamese day label ("20/6", "9/7", ...).
func buildDemandPeriod(s cohortSeries) wallet.WalletDemandPeriod {
	pts := make([]wallet.WalletDemandPoint, 0, s.maxCycleDay)
	for d := 1; d <= s.maxCycleDay; d++ {
		pts = append(pts, wallet.WalletDemandPoint{
			CycleDay:    d,
			DayLabel:    dayLabel(s.forMonth, d),
			Amount:      s.cumulative[d],
			DailyAmount: s.dailyAmount[d],
		})
	}
	return wallet.WalletDemandPeriod{
		ForMonth:  s.forMonth,
		Label:     "Kỳ " + clock.FormatMonthDisplay(s.forMonth),
		IsCurrent: s.isCurrent,
		Series:    pts,
	}
}

// addMonths returns forMonth shifted by n months ("2006-01" format).
func addMonths(forMonth string, n int) string {
	t, err := clock.ParseMonth(forMonth)
	if err != nil {
		return forMonth
	}
	return t.AddDate(0, n, 0).Format("2006-01")
}

// dayLabel converts (forMonth, cycleDay) into a compact Vietnamese day label
// such as "20/6" (day/month). cycleDay 1 = day 20 of forMonth.
func dayLabel(forMonth string, cycleDay int) string {
	m, err := clock.ParseMonth(forMonth)
	if err != nil {
		return fmt.Sprintf("%d", cycleDay)
	}
	year, month := m.Year(), m.Month()
	daysInStartMonth := daysInMonth(year, month) - clock.PeriodCycleStartDay + 1

	var label time.Time
	if cycleDay <= daysInStartMonth {
		day := clock.PeriodCycleStartDay + cycleDay - 1
		label = time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	} else {
		// Into the next calendar month: start from day 20 of forMonth and roll
		// forward (cycleDay - daysInStartMonth) days into the next month.
		start := time.Date(year, month, clock.PeriodCycleStartDay, 0, 0, 0, 0, time.UTC)
		label = start.AddDate(0, 0, daysInStartMonth+(cycleDay-daysInStartMonth)-1)
	}
	return fmt.Sprintf("%d/%d", label.Day(), int(label.Month()))
}
