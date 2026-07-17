package services

import (
	"math"
	"math/rand/v2"
	"sort"
	"time"

	"api-server/internal/config"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

const cashReadinessModelVersion = "cash-readiness-v3"

type cashReadinessV2Projection struct {
	approved           int64
	pending            int64
	expectedPending    int64
	expectedFuture     int64
	expectedPayout     int64
	recommendedReserve int64
	intervalLower      int64
	intervalUpper      int64
	legacyMedian       int64
	legacyExpected     int64
	legacyP95          int64
	pendingRate        float64
	basisCycles        int
	method             string
}

type cashCycleObservation struct {
	key                   string
	payDate               time.Time
	observedEmployees     int
	finalEmployees        int
	futurePerEmployee     float64
	finalApprovedPerHead  float64
	projectFuturePerHead  map[uint]float64
	projectObservedHeads  map[uint]int
	completedOnlyFallback bool
}

// forecastCashReadinessV2 performs a deterministic, non-parametric bootstrap.
// It pools comparable cycle shapes across all Ky, but never pools raw amounts:
// future-created pay is normalized by the employee count observed at the same
// cycle-day. Project-specific shapes are used only when every target project
// has a sufficient basis; otherwise the company-normalized sample is used.
func forecastCashReadinessV2(rows []domain.TimesheetAccrualDailyRow, now time.Time, pc clock.TimesheetPayCycle, seed int64, cfg config.CashForecastConfig) cashReadinessV2Projection {
	now = now.In(clock.DefaultLocation)
	targetMonth := pc.WorkMonth.Format("2006-01")
	targetKey := cashCycleKey(targetMonth, pc.Ky)

	clean := make([]domain.TimesheetAccrualDailyRow, 0, len(rows))
	for _, raw := range rows {
		if raw.Amount <= 0 || raw.WorkDate.IsZero() {
			continue
		}
		clean = append(clean, normalizeForecastRow(raw))
	}

	var approved, pending int64
	pendingRows := make([]int64, 0)
	targetHasRows := false
	targetEmployees := make(map[uint]struct{})
	targetProjectEmployees := make(map[uint]map[uint]struct{})
	byCycle := make(map[string][]domain.TimesheetAccrualDailyRow)
	for _, row := range clean {
		month := row.WorkDate.In(clock.DefaultLocation).Format("2006-01")
		ky := clock.KyFromWorkDay(row.WorkDate.In(clock.DefaultLocation).Day())
		key := cashCycleKey(month, ky)
		byCycle[key] = append(byCycle[key], row)
		if key != targetKey || row.CreatedAt.After(now) {
			continue
		}
		targetHasRows = true
		if row.EmployeeID != 0 {
			targetEmployees[row.EmployeeID] = struct{}{}
			if targetProjectEmployees[row.ProjectID] == nil {
				targetProjectEmployees[row.ProjectID] = make(map[uint]struct{})
			}
			targetProjectEmployees[row.ProjectID][row.EmployeeID] = struct{}{}
		}
		if approvedBy(row, now) {
			approved += row.Amount
		} else if row.Status != domain.TimesheetStatusRejected {
			pending += row.Amount
			pendingRows = append(pendingRows, row.Amount)
		}
	}

	observations, pendingSuccesses, pendingFailures := buildCashCycleObservations(byCycle, targetKey, now, pc.CycleDayToday)
	if len(observations) > 52 {
		observations = observations[len(observations)-52:]
	}
	pendingRate := betaSmoothedRate(pendingSuccesses, pendingFailures)
	if len(observations) == 0 {
		expectedPending := int64(math.Round(float64(pending) * pendingRate))
		expected := approved + expectedPending
		return cashReadinessV2Projection{
			approved: approved, pending: pending, expectedPending: expectedPending,
			expectedPayout: expected, recommendedReserve: expected,
			intervalLower: approved, intervalUpper: max(approved, expected),
			legacyMedian: approved, legacyExpected: approved, legacyP95: approved,
			pendingRate: pendingRate, method: "no-history",
		}
	}

	targetHeadcount := len(targetEmployees)
	if !targetHasRows {
		targetHeadcount = recentWorkforceLevel(observations, 0, cfg.GrowthEWMAlpha)
	} else if targetHeadcount == 0 {
		// Compatibility for legacy aggregate rows without employee identity.
		targetHeadcount = medianObservedHeadcount(observations)
	}
	if targetHeadcount < 1 {
		targetHeadcount = 1
	}

	nSim := cfg.NSim
	if nSim <= 0 {
		nSim = 5000
	}
	rng := rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0x9e3779b97f4a7c15))
	samples := make([]int64, nSim)
	var futureSampleTotal float64
	useProjects := targetHasRows && projectBasisSufficient(observations, targetProjectEmployees, 6)
	method := "normalized-bootstrap"
	completedFallbacks := 0
	for _, observation := range observations {
		if observation.completedOnlyFallback {
			completedFallbacks++
		}
	}
	if !targetHasRows || completedFallbacks == len(observations) {
		method = "completed-cycle-bootstrap"
	}
	for i := range samples {
		var pendingSample int64
		for _, amount := range pendingRows {
			if rng.Float64() < pendingRate {
				pendingSample += amount
			}
		}

		var futureSample int64
		if !targetHasRows {
			obs := observations[rng.IntN(len(observations))]
			futureSample = int64(math.Round(obs.finalApprovedPerHead * float64(targetHeadcount)))
		} else if useProjects {
			for projectID, employees := range targetProjectEmployees {
				choices := projectRates(observations, projectID)
				if len(choices) == 0 {
					continue
				}
				futureSample += int64(math.Round(choices[rng.IntN(len(choices))] * float64(max(1, len(employees)))))
			}
		} else {
			obs := observations[rng.IntN(len(observations))]
			futureSample = int64(math.Round(obs.futurePerEmployee * float64(targetHeadcount)))
		}
		futureSampleTotal += float64(futureSample)
		samples[i] = max(int64(0), approved+pendingSample+futureSample)
	}

	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	expectedPending := int64(math.Round(float64(pending) * pendingRate))
	expectedFuture := int64(math.Round(futureSampleTotal / float64(nSim)))
	expectedFuture = max(int64(0), expectedFuture)
	expected := max(approved, approved+expectedPending+expectedFuture)
	lower := max(approved, cashQuantile(samples, 0.05))
	upper := max(lower, cashQuantile(samples, 0.95))
	serviceLevel := cfg.ServiceLevel
	if serviceLevel <= 0 {
		serviceLevel = 0.95
	}
	reserve := cashQuantile(samples, serviceLevel)
	reserve = max(reserve, expected)
	reserve = max(reserve, approved)
	upper = max(upper, expected)

	return cashReadinessV2Projection{
		approved: approved, pending: pending, expectedPending: expectedPending,
		expectedFuture: expectedFuture, expectedPayout: expected,
		recommendedReserve: reserve, intervalLower: lower, intervalUpper: upper,
		legacyMedian: cashQuantile(samples, 0.50), legacyExpected: expected,
		legacyP95:   cashQuantile(samples, 0.95),
		pendingRate: pendingRate, basisCycles: len(observations), method: method,
	}
}

func normalizeForecastRow(row domain.TimesheetAccrualDailyRow) domain.TimesheetAccrualDailyRow {
	if row.ApprovedAt == nil && !row.ApprovedDate.IsZero() {
		approved := row.ApprovedDate
		row.ApprovedAt = &approved
	}
	if row.CreatedAt.IsZero() {
		// Compatibility for the former approved-day aggregate contract: its only
		// observable arrival timestamp was ApprovedDate.
		if row.ApprovedAt != nil {
			row.CreatedAt = *row.ApprovedAt
		} else {
			row.CreatedAt = row.WorkDate
		}
	}
	if row.Status == "" {
		if row.ApprovedAt != nil {
			row.Status = domain.TimesheetStatusApproved
		} else {
			row.Status = domain.TimesheetStatusPendingApproval
		}
	}
	return row
}

func approvedBy(row domain.TimesheetAccrualDailyRow, asOf time.Time) bool {
	return row.Status == domain.TimesheetStatusApproved && row.ApprovedAt != nil && !row.ApprovedAt.After(asOf)
}

func buildCashCycleObservations(byCycle map[string][]domain.TimesheetAccrualDailyRow, targetKey string, now time.Time, cycleDay int) ([]cashCycleObservation, int, int) {
	observations := make([]cashCycleObservation, 0, len(byCycle))
	var successes, failures int
	for key, cycleRows := range byCycle {
		if key == targetKey || len(cycleRows) == 0 {
			continue
		}
		first := cycleRows[0].WorkDate.In(clock.DefaultLocation)
		ky := clock.KyFromWorkDay(first.Day())
		payDate := clock.PayDate(ky, first.Year(), first.Month())
		if payDate.After(now) {
			continue
		}
		maxDay := clock.MaxCycleDay(ky, first.Year(), first.Month())
		analogDay := min(max(1, cycleDay), maxDay)
		asOf := time.Date(first.Year(), first.Month(), clock.WorkStartDay(ky), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), clock.DefaultLocation).
			AddDate(0, 0, analogDay-1)

		employees := make(map[uint]struct{})
		finalEmployees := make(map[uint]struct{})
		observedAny := false
		projectEmployees := make(map[uint]map[uint]struct{})
		finalProjectEmployees := make(map[uint]map[uint]struct{})
		projectFuture := make(map[uint]int64)
		projectFinal := make(map[uint]int64)
		var future int64
		var finalApproved int64
		for _, row := range cycleRows {
			if row.EmployeeID != 0 {
				finalEmployees[row.EmployeeID] = struct{}{}
				if finalProjectEmployees[row.ProjectID] == nil {
					finalProjectEmployees[row.ProjectID] = make(map[uint]struct{})
				}
				finalProjectEmployees[row.ProjectID][row.EmployeeID] = struct{}{}
			}
			if row.Status == domain.TimesheetStatusApproved {
				finalApproved += row.Amount
				projectFinal[row.ProjectID] += row.Amount
			}
			if !row.CreatedAt.After(asOf) {
				observedAny = true
				if row.EmployeeID != 0 {
					employees[row.EmployeeID] = struct{}{}
					if projectEmployees[row.ProjectID] == nil {
						projectEmployees[row.ProjectID] = make(map[uint]struct{})
					}
					projectEmployees[row.ProjectID][row.EmployeeID] = struct{}{}
				}
				if !approvedBy(row, asOf) {
					if row.Status == domain.TimesheetStatusApproved && row.ApprovedAt != nil &&
						row.ApprovedAt.After(asOf) && approvedByPayDate(row, payDate) {
						successes++
					} else {
						// Rejected, still-pending, and late-approved rows all missed
						// this cycle's transfer and therefore count as failures.
						failures++
					}
				}
			}
			if row.CreatedAt.After(asOf) && row.Status == domain.TimesheetStatusApproved &&
				approvedByPayDate(row, payDate) {
				future += row.Amount
				projectFuture[row.ProjectID] += row.Amount
			}
		}
		// Some customers enter a completed Ky in one bulk operation after the
		// analogous forecast day. Such a cycle cannot teach pending conversion,
		// but its final approved total is still useful for estimating a target Ky
		// with no rows yet. Keep it as a per-final-employee fallback instead of
		// collapsing the forecast to zero.
		if !observedAny {
			if finalApproved == 0 {
				continue
			}
			future = finalApproved
			if len(projectFuture) == 0 {
				projectFuture = projectFinal
			}
		}
		if !observedAny && future == 0 {
			continue
		}
		observedHeadcount := len(employees)
		if observedHeadcount == 0 {
			observedHeadcount = len(finalEmployees)
			if observedHeadcount == 0 {
				// V1 aggregate rows did not carry employee identity. Treat the
				// aggregate as one normalized unit so old callers/tests remain valid.
				observedHeadcount = 1
			}
		}
		finalHeadcount := len(finalEmployees)
		if finalHeadcount == 0 {
			finalHeadcount = 1
		}
		obs := cashCycleObservation{
			key: key, payDate: payDate, observedEmployees: observedHeadcount,
			finalEmployees:       finalHeadcount,
			futurePerEmployee:    float64(future) / float64(observedHeadcount),
			finalApprovedPerHead: float64(finalApproved) / float64(finalHeadcount),
			projectFuturePerHead: make(map[uint]float64), projectObservedHeads: make(map[uint]int),
			completedOnlyFallback: !observedAny,
		}
		for projectID, amount := range projectFuture {
			heads := len(projectEmployees[projectID])
			if heads == 0 {
				heads = len(finalProjectEmployees[projectID])
			}
			if heads > 0 {
				obs.projectFuturePerHead[projectID] = float64(amount) / float64(heads)
				obs.projectObservedHeads[projectID] = heads
			}
		}
		observations = append(observations, obs)
	}
	sort.Slice(observations, func(i, j int) bool {
		if observations[i].payDate.Equal(observations[j].payDate) {
			return observations[i].key < observations[j].key
		}
		return observations[i].payDate.Before(observations[j].payDate)
	})
	return observations, successes, failures
}

func approvedByPayDate(row domain.TimesheetAccrualDailyRow, payDate time.Time) bool {
	if row.ApprovedAt == nil {
		return false
	}
	// PayDate is date-only midnight. A payroll approval at any time on that
	// calendar date is on time, so compare against the next day's midnight.
	return row.ApprovedAt.Before(payDate.AddDate(0, 0, 1))
}

func betaSmoothedRate(successes, failures int) float64 {
	// Beta(2,2) keeps sparse history away from unjustified 0%/100% conversion.
	return float64(successes+2) / float64(successes+failures+4)
}

func cashCycleKey(month string, ky int) string { return month + "/ky" + string(rune('0'+ky)) }

func medianObservedHeadcount(observations []cashCycleObservation) int {
	values := make([]int, len(observations))
	for i := range observations {
		values[i] = observations[i].observedEmployees
	}
	sort.Ints(values)
	return values[len(values)/2]
}

// recentWorkforceLevel estimates the employee participation level for the
// target Ky when it has no rows. The current observed headcount parameter is a
// defensive floor; the level is an EWMA of final participation in completed
// cycles. This preserves scale without changing the established partial-cycle
// normalization once target rows begin arriving.
func recentWorkforceLevel(observations []cashCycleObservation, currentObserved int, alpha float64) int {
	if len(observations) == 0 {
		return currentObserved
	}
	if alpha <= 0 || alpha > 1 {
		alpha = growthEWMAlphaDefault
	}
	level := float64(max(1, observations[0].finalEmployees))
	for _, observation := range observations[1:] {
		level = alpha*float64(max(1, observation.finalEmployees)) + (1-alpha)*level
	}
	return max(currentObserved, int(math.Round(level)))
}

func projectBasisSufficient(observations []cashCycleObservation, targets map[uint]map[uint]struct{}, minimum int) bool {
	if len(targets) == 0 {
		return false
	}
	for projectID := range targets {
		if len(projectRates(observations, projectID)) < minimum {
			return false
		}
	}
	return true
}

func projectRates(observations []cashCycleObservation, projectID uint) []float64 {
	result := make([]float64, 0, len(observations))
	for _, obs := range observations {
		if rate, ok := obs.projectFuturePerHead[projectID]; ok {
			result = append(result, rate)
		}
	}
	return result
}

func cashQuantile(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	return sorted[max(0, min(idx, len(sorted)-1))]
}
