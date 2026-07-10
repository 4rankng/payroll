package dto

import "time"

// CashReadinessResponse is the JSON payload for GET /api/v1/timesheets/cash-readiness.
// All monetary fields are VND (no decimals).
type CashReadinessResponse struct {
	ConfirmedPayable int64 `json:"confirmed_payable"`

	// Projected additional accrual between now and the next pay date.
	ProjectedP50 int64 `json:"projected_p50"`
	ProjectedP95 int64 `json:"projected_p95"`

	// Total cash-need band: [confirmed + p50, confirmed + p95].
	BandLower int64 `json:"band_lower"`
	BandUpper int64 `json:"band_upper"`

	WalletAvailable   int64 `json:"wallet_available"`
	WalletAvailableOK bool  `json:"wallet_available_ok"`

	// CashToPrepare = confirmed + p50 (expected total). Gap = max(0, CashToPrepare − wallet).
	CashToPrepare int64 `json:"cash_to_prepare"`
	Gap           int64 `json:"gap"`

	PrepareByDate string `json:"prepare_by_date"` // RFC3339 (date)
	NextPayDate   string `json:"next_pay_date"`   // RFC3339 (date)
	LeadDays      int    `json:"lead_days"`
	Ky            int    `json:"ky"`
	CycleDayToday int    `json:"cycle_day_today"`

	Method      string `json:"method"`
	Confidence  string `json:"confidence"`
	BasisCycles int    `json:"basis_cycles"`

	GeneratedAt time.Time `json:"generated_at"`
}
