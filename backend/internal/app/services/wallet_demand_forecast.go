package services

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
	"api-server/internal/domain/wallet"
	"api-server/internal/pkg/clock"
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
}

// NewWalletDemandForecastService constructs the forecast service. clk defaults to
// the real clock when nil.
func NewWalletDemandForecastService(
	requestRepo domain.AdvancePaymentRequestRepository,
	walletSvc wallet.WalletService,
	clk clock.Clock,
) *WalletDemandForecastService {
	if clk == nil {
		clk = clock.New()
	}
	return &WalletDemandForecastService{
		requestRepo: requestRepo,
		walletSvc:   walletSvc,
		clock:       clk,
	}
}

// GetDemandForecast builds the cohort series for the current period + last two
// completed periods and the balance prediction for the current period.
func (s *WalletDemandForecastService) GetDemandForecast(ctx context.Context) (*wallet.WalletDemandForecastResponse, error) {
	now := s.clock.Now()
	currentForMonth := clock.AdvanceMonthFromTime(now)
	maxDay := maxCycleDay(currentForMonth)
	todayCycleDay := max(cycleDayFor(now, currentForMonth), 1)

	forMonths := []string{
		currentForMonth,
		addMonths(currentForMonth, -1),
		addMonths(currentForMonth, -2),
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

	projectedTotal, method, basis, confidence := forecastProjectedTotal(actualSoFar, historical, todayCycleDay)
	rate := completionRate(historical)
	projectedPaid := int64(float64(projectedTotal) * rate)
	if method == "no-history" {
		// No completion-rate basis: assume every projected request will be paid
		// (conservative for treasury — never under-recommend with no history).
		projectedPaid = projectedTotal
	}
	alreadyPaid := current.completedTotal
	remainingToPay := max(int64(0), projectedPaid-alreadyPaid)
	recommendedBalance := remainingToPay

	currentAvailable := int64(0)
	if bal, err := s.walletSvc.GetBalance(ctx); err == nil && bal != nil {
		currentAvailable = bal.Available
	}

	shortfall := recommendedBalance - currentAvailable
	surplus := int64(0)
	if shortfall < 0 {
		surplus = -shortfall
		shortfall = 0
	}

	periods := make([]wallet.WalletDemandPeriod, 0, len(forMonths))
	for _, fm := range forMonths {
		periods = append(periods, buildDemandPeriod(byMonth[fm]))
	}

	return &wallet.WalletDemandForecastResponse{
		CurrentForMonth: currentForMonth,
		CurrentCycleDay: todayCycleDay,
		MaxCycleDay:     maxDay,
		Periods:         periods,
		Prediction: wallet.WalletDemandPrediction{
			ActualSoFar:        actualSoFar,
			ProjectedTotal:     projectedTotal,
			ProjectedPaid:      projectedPaid,
			AlreadyPaid:        alreadyPaid,
			RemainingToPay:     remainingToPay,
			RecommendedBalance: recommendedBalance,
			CurrentAvailable:   currentAvailable,
			Shortfall:          shortfall,
			Surplus:            surplus,
			CompletionRate:     rate,
			Method:             method,
			Confidence:         confidence,
			BasisPeriods:       basis,
		},
		GeneratedAt: now.Format(time.RFC3339),
	}, nil
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
