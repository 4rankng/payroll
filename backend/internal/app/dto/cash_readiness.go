package dto

import "time"

// CashReadinessResponse is the JSON payload for GET /api/v1/timesheets/cash-readiness.
// All monetary fields are VND (no decimals).
type CashReadinessResponse struct {
	// Approved value already observed inside the target Ky. This is not the
	// outstanding-payment amount shown by the timesheet summary.
	ObservedApproved int64 `json:"observed_approved"`

	// Projected additional target-Ky approvals between now and its pay date.
	ProjectedP50      int64 `json:"projected_p50"`
	ProjectedExpected int64 `json:"projected_expected"`
	ProjectedP95      int64 `json:"projected_p95"`

	// Final target-Ky payout band: [observed approved + p50, observed approved + p95].
	BandLower     int64 `json:"band_lower"`
	BandUpper     int64 `json:"band_upper"`
	ExpectedTotal int64 `json:"expected_total"`

	WalletAvailable   int64 `json:"wallet_available"`
	WalletAvailableOK bool  `json:"wallet_available_ok"`

	// CashToPrepare = observed approved + p50. Gap = max(0, CashToPrepare − wallet).
	CashToPrepare int64 `json:"cash_to_prepare"`
	Gap           int64 `json:"gap"`

	PrepareByDate string `json:"prepare_by_date"` // RFC3339 (date)
	NextPayDate   string `json:"next_pay_date"`   // RFC3339 (date)
	LeadDays      int    `json:"lead_days"`
	Ky            int    `json:"ky"`
	CycleDayToday int    `json:"cycle_day_today"`

	Method      string  `json:"method"`
	Confidence  string  `json:"confidence"`
	BasisCycles int     `json:"basis_cycles"`
	GrowthRate  float64 `json:"growth_rate"`

	GeneratedAt time.Time `json:"generated_at"`
}
