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

	// Legacy final target-Ky payout band: [p50 total, p95 total].
	BandLower     int64 `json:"band_lower"`
	BandUpper     int64 `json:"band_upper"`
	ExpectedTotal int64 `json:"expected_total"`

	PendingTargetAmount   int64   `json:"pending_target_amount"`
	ExpectedPendingAmount int64   `json:"expected_pending_amount"`
	ExpectedFutureAmount  int64   `json:"expected_future_amount"`
	ExpectedPayout        int64   `json:"expected_payout"`
	RecommendedReserve    int64   `json:"recommended_reserve"`
	IntervalLower         int64   `json:"interval_lower"`
	IntervalUpper         int64   `json:"interval_upper"`
	ModelVersion          string  `json:"model_version"`
	CalibrationSamples    int     `json:"calibration_samples"`
	ReliabilityState      string  `json:"reliability_state"`
	AccuracyWAPE          float64 `json:"accuracy_wape"`
	AccuracyBias          float64 `json:"accuracy_bias"`
	IntervalCoverage      float64 `json:"interval_coverage"`
	ReserveShortfallRate  float64 `json:"reserve_shortfall_rate"`

	WalletAvailable   int64 `json:"wallet_available"`
	WalletAvailableOK bool  `json:"wallet_available_ok"`

	// Legacy planning contract: CashToPrepare = p50 total and
	// Gap = max(0, CashToPrepare − wallet). New consumers should use
	// RecommendedReserve for the service-level reserve.
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
