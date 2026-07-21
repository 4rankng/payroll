package wallet

// WalletDemandForecastResponse is the payload for GET /wallet/demand-forecast.
// It bundles the per-period cohort series (chart) and the balance prediction
// (card) since both derive from one underlying cohort query.
type WalletDemandForecastResponse struct {
	CurrentForMonth string                 `json:"current_for_month"`
	CurrentCycleDay int                    `json:"current_cycle_day"`
	MaxCycleDay     int                    `json:"max_cycle_day"`
	Periods         []WalletDemandPeriod   `json:"periods"`
	Prediction      WalletDemandPrediction `json:"prediction"`
	GeneratedAt     string                 `json:"generated_at"`
}

// WalletDemandPeriod is one cohort line: cumulative request volume for a single
// advance-payment period, sampled at every cycle day.
type WalletDemandPeriod struct {
	ForMonth  string              `json:"for_month"`
	Label     string              `json:"label"` // "Kỳ 06/2026"
	IsCurrent bool                `json:"is_current"`
	Series    []WalletDemandPoint `json:"series"`
}

// WalletDemandPoint is one X-axis sample of a period's cumulative curve.
type WalletDemandPoint struct {
	CycleDay    int    `json:"cycle_day"`
	DayLabel    string `json:"day_label"` // "20/6" … "9/7"
	Amount      int64  `json:"amount"`    // cumulative request amount through this day
	DailyAmount int64  `json:"daily_amount"`
}

// WalletDemandPrediction is the advisory balance forecast for the current
// period. DISPLAY ONLY — must not feed SyncBalance or any auto top-up.
//
// RecommendedBalance and P50/P90/P99Reference describe the remaining-cycle
// forecast through the cutoff day. LeadDays and HorizonCycleDay remain in the
// response for backward compatibility but do not limit the balance target.
type WalletDemandPrediction struct {
	ActualSoFar        int64   `json:"actual_so_far"`
	ProjectedTotal     int64   `json:"projected_total"` // p50 projection of full-period net demand
	ProjectedPaid      int64   `json:"projected_paid"`
	AlreadyPaid        int64   `json:"already_paid"`
	RemainingToPay     int64   `json:"remaining_to_pay"` // p50-based, kept for display continuity
	RecommendedBalance int64   `json:"recommended_balance"`
	CurrentAvailable   int64   `json:"current_available"`
	Shortfall          int64   `json:"shortfall"` // vs RecommendedBalance
	Surplus            int64   `json:"surplus"`
	CompletionRate     float64 `json:"completion_rate"`
	Method             string  `json:"method"`     // monte-carlo | gamma-fit | no-history | cohort-median (legacy)
	Confidence         string  `json:"confidence"` // high | medium | low (derived from n × MC/gamma agreement)
	BasisPeriods       int     `json:"basis_periods"`
	LeadDays           int     `json:"lead_days"`
	HorizonCycleDay    int     `json:"horizon_cycle_day"`

	// Newsvendor / tail-risk fields.
	P50Reference        int64                      `json:"p50_reference"`        // median remaining-cycle cash-out
	P90Reference        int64                      `json:"p90_reference"`        // p90 remaining-cycle cash-out
	P99Reference        int64                      `json:"p99_reference"`        // p99 remaining-cycle cash-out (tail)
	CoverageProbability float64                    `json:"coverage_probability"` // p* actually used (e.g. 0.95)
	NHistory            int                        `json:"n_history"`            // usable historical periods
	ConfidenceInterval  ForecastConfidenceInterval `json:"confidence_interval"`
	ServiceLevel        ForecastServiceLevel       `json:"service_level"`
}

// ForecastConfidenceInterval is the small-n band on RecommendedBalance,
// expressed as the [p50, p99] range of the predicted remaining cash-out.
type ForecastConfidenceInterval struct {
	Lower int64 `json:"lower"`
	Upper int64 `json:"upper"`
}

// ForecastServiceLevel reports the newsvendor p* used and the (optional) Cu/Co
// costs it was derived from.
type ForecastServiceLevel struct {
	Quantile  float64 `json:"quantile"`
	CostUnder float64 `json:"cost_under"`
	CostOver  float64 `json:"cost_over"`
}
