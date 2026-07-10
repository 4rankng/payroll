package services

import (
	"context"
	"testing"
	"time"

	"api-server/internal/config"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// TestBacktest_ProjectionVsActual demonstrates the backtest methodology the
// standalone CLI will run: forecast each historical cycle from the cycles that
// preceded it, then compare projected p50 to that cycle's actual grand total.
// This is the offline accuracy signal that feeds the statistical-vs-ML decision.
func TestBacktest_ProjectionVsActual(t *testing.T) {
	d := func(s string) time.Time {
		t, _ := time.ParseInLocation("2006-01-02", s, clock.DefaultLocation)
		return t
	}
	// Three Ky-1 cycles. We forecast each from the prior ones and compare to its
	// own grand total (the "actual" the cycle reached).
	cycles := []struct {
		forMonth   string
		rows       []domain.TimesheetAccrualDailyRow
		actualTotal int64
	}{
		{
			forMonth: "2026-04",
			rows: []domain.TimesheetAccrualDailyRow{
				{WorkDate: d("2026-04-03"), ApprovedDate: d("2026-04-01"), Amount: 100},
				{WorkDate: d("2026-04-03"), ApprovedDate: d("2026-04-05"), Amount: 200},
				{WorkDate: d("2026-04-03"), ApprovedDate: d("2026-04-10"), Amount: 700},
			},
			actualTotal: 1000,
		},
		{
			forMonth: "2026-05",
			rows: []domain.TimesheetAccrualDailyRow{
				{WorkDate: d("2026-05-03"), ApprovedDate: d("2026-05-01"), Amount: 120},
				{WorkDate: d("2026-05-03"), ApprovedDate: d("2026-05-05"), Amount: 180},
				{WorkDate: d("2026-05-03"), ApprovedDate: d("2026-05-10"), Amount: 800},
			},
			actualTotal: 1100,
		},
		{
			forMonth: "2026-06",
			rows: []domain.TimesheetAccrualDailyRow{
				{WorkDate: d("2026-06-02"), ApprovedDate: d("2026-06-01"), Amount: 110},
				{WorkDate: d("2026-06-02"), ApprovedDate: d("2026-06-05"), Amount: 190},
				{WorkDate: d("2026-06-02"), ApprovedDate: d("2026-06-10"), Amount: 750},
			},
			actualTotal: 1050,
		},
	}

	provider := NewTimesheetAccrualProvider()
	cfg := config.CashForecastConfig{}
	ctx := context.Background()

	// Forecast cycle index i (i >= 1) from cycles [0..i-1] observed at cycle-day 3,
	// projected through the pay date (cycle-day 10). Compare p50 of the FULL
	// remaining accrual + observed-so-far to the actual total.
	type result struct {
		forMonth string
		projected int64
		actual    int64
	}
	var results []result
	for i := 1; i < len(cycles); i++ {
		// Basis = earlier cycles only (the backtest must not peek at the target).
		var basis []domain.TimesheetAccrualDailyRow
		for j := 0; j < i; j++ {
			basis = append(basis, cycles[j].rows...)
		}
		hist := buildTimesheetCohort(basis, 1, cycles[i].forMonth)

		// Observed-so-far in the target cycle at cycle-day 3.
		target := buildTimesheetCohort(cycles[i].rows, 1, "9999-99")
		var observedSoFar int64
		if len(target) == 1 {
			observedSoFar = target[0].cumulativeAt(3)
		}
		// Project accrual from cycle-day 3 through 10 and add observed-so-far.
		proj, err := provider.ProjectAccrual(ctx, hist, 3, 10, timesheetForecastSeed(1, cycles[i].forMonth, 3), cfg)
		if err != nil {
			t.Fatalf("ProjectAccrual cycle %s: %v", cycles[i].forMonth, err)
		}
		results = append(results, result{
			forMonth:  cycles[i].forMonth,
			projected: observedSoFar + proj.P50,
			actual:    cycles[i].actualTotal,
		})
	}

	// With 2 similar prior cycles, the projected p50 should land within ~50% of
	// actual for each forecasted cycle (a loose, honest bound for a 2-basis
	// gamma-fit; the real decision threshold is ~12% over >=4 cycles).
	for _, r := range results {
		if r.actual == 0 {
			continue
		}
		errPct := float64(r.projected-r.actual) / float64(r.actual)
		if errPct < 0 {
			errPct = -errPct
		}
		t.Logf("cycle %s: projected=%d actual=%d abs%%err=%.1f%%", r.forMonth, r.projected, r.actual, errPct*100)
		if errPct > 0.5 {
			t.Errorf("cycle %s: projection %d off by %.1f%% from actual %d (bound 50%%)", r.forMonth, r.projected, errPct*100, r.actual)
		}
	}
}
