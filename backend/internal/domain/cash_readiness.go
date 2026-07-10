package domain

import "time"

// CashReadiness is the advisory cash-prep forecast for the next timesheet bulk
// transfer. It composes the deterministic confirmed-payable floor with a
// short-horizon projected-accrual band and subtracts the live wallet balance to
// surface a single "cash to prepare" gap.
//
// ADVISORY ONLY: this type never feeds SyncBalance/CreateTopup or any
// disbursement — see CashReadinessForecastService's read-only ports.
type CashReadiness struct {
	// ConfirmedPayable is the approved + pending-approval, not-yet-paid total
	// (the deterministic floor, identical to GetSummaryStats.PendingPaymentAmount).
	ConfirmedPayable int64

	// ProjectedP50 / ProjectedP95 are the expected / tail additional approved pay
	// forecast to accrue between now and the next pay date.
	ProjectedP50 int64
	ProjectedP95 int64

	// BandLower / BandUpper bound the total cash need: [confirmed + p50,
	// confirmed + p95]. They are always monotonic (Lower <= Upper).
	BandLower int64
	BandUpper int64

	// WalletAvailable is the live wallet Available balance. WalletAvailableOK is
	// false when the wallet read failed (the figure is 0 and should be flagged).
	WalletAvailable   int64
	WalletAvailableOK bool

	// CashToPrepare is the expected total to prepare (confirmed + p50). The Gap
	// is how much of that exceeds the current wallet balance (>= 0).
	CashToPrepare int64
	Gap           int64

	// Pay-cycle framing.
	PrepareByDate time.Time
	NextPayDate   time.Time
	LeadDays      int
	Ky            int
	CycleDayToday int

	// Forecast quality signal: "monte-carlo" | "gamma-fit" | "no-history";
	// "high" | "medium" | "low"; and the number of historical cycles used.
	Method      string
	Confidence  string
	BasisCycles int

	GeneratedAt time.Time
}
