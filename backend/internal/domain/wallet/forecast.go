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
type WalletDemandPrediction struct {
	ActualSoFar        int64   `json:"actual_so_far"`
	ProjectedTotal     int64   `json:"projected_total"`
	ProjectedPaid      int64   `json:"projected_paid"`
	AlreadyPaid        int64   `json:"already_paid"`
	RemainingToPay     int64   `json:"remaining_to_pay"`
	RecommendedBalance int64   `json:"recommended_balance"`
	CurrentAvailable   int64   `json:"current_available"`
	Shortfall          int64   `json:"shortfall"`
	Surplus            int64   `json:"surplus"`
	CompletionRate     float64 `json:"completion_rate"`
	Method             string  `json:"method"`     // cohort-median | avg-final | no-history
	Confidence         string  `json:"confidence"` // high | medium | low
	BasisPeriods       int     `json:"basis_periods"`
}
