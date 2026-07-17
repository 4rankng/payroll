package domain

import "time"

// CashReadiness is the advisory forecast for the next timesheet payout. It
// combines the target Ky's current observed value and a short-horizon
// projection of approvals still expected before its pay date.
//
// ADVISORY ONLY: this type never feeds SyncBalance/CreateTopup or any
// disbursement — see CashReadinessForecastService's read-only ports.
type CashReadiness struct {
	// ObservedApproved is approved value already recorded inside the target Ky.
	ObservedApproved int64

	// ProjectedP50 / ProjectedExpected / ProjectedP95 are the median, arithmetic
	// mean, and tail additional target-Ky approvals expected between now and the
	// target Ky's pay date.
	ProjectedP50      int64
	ProjectedExpected int64
	ProjectedP95      int64

	// BandLower / BandUpper bound the final target-Ky payout:
	// [observed approved + p50, observed approved + p95].
	BandLower int64
	BandUpper int64

	// ExpectedTotal is the mathematical point estimate for the target-Ky payout.
	ExpectedTotal int64

	// V2 decomposition and decision outputs. RecommendedReserve is the
	// operational service-level quantile; ExpectedPayout remains the arithmetic
	// expectation. Both are floored by ObservedApproved.
	PendingTargetAmount   int64
	ExpectedPendingAmount int64
	ExpectedFutureAmount  int64
	ExpectedPayout        int64
	RecommendedReserve    int64
	IntervalLower         int64
	IntervalUpper         int64
	ModelVersion          string

	// Calibration is based only on resolved, company-wide snapshots from the
	// same horizon bucket. Rates are decimal fractions (0.10 = 10%).
	CalibrationSamples   int
	ReliabilityState     string
	AccuracyWAPE         float64
	AccuracyBias         float64
	IntervalCoverage     float64
	ReserveShortfallRate float64

	// WalletAvailable is the live wallet Available balance. WalletAvailableOK is
	// false when the wallet read failed (the figure is 0 and should be flagged).
	WalletAvailable   int64
	WalletAvailableOK bool

	// CashToPrepare is the target-Ky observed approved value + projected p50. The
	// Gap is how much of that exceeds the current wallet balance (>= 0).
	CashToPrepare int64
	Gap           int64

	// Pay-cycle framing.
	PrepareByDate time.Time
	NextPayDate   time.Time
	LeadDays      int
	Ky            int
	CycleDayToday int

	// Forecast quality signal: "monte-carlo" | "gamma-fit" | "no-history" |
	// "growth-adjusted"; "high" | "medium" | "low"; and the number of historical
	// cycles used. GrowthRate is the EWMA growth factor applied (>1 = upward
	// trend; 1.0 when stationary).
	Method      string
	Confidence  string
	BasisCycles int
	GrowthRate  float64

	GeneratedAt time.Time
}
