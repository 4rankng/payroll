package dto

import "time"

// WeeklyPaymentFeeScheduleEntryResponse is the wire shape returned by the
// admin weekly-payment fee schedule endpoints. IsCurrentlyActive is computed
// server-side against time.Now(): exactly one entry is active at any moment.
type WeeklyPaymentFeeScheduleEntryResponse struct {
	ID                string    `json:"id"`
	EffectiveDate     string    `json:"effectiveDate"` // YYYY-MM-DD
	Percentage        float64   `json:"percentage"`    // e.g. 2 for 2%
	Notes             string    `json:"notes,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	CreatedByUserID   *uint     `json:"createdByUserId,omitempty"`
	IsCurrentlyActive bool      `json:"isCurrentlyActive"`
	IsPending         bool      `json:"isPending"` // effective_date > today
	Summary           string    `json:"summary"`   // human-readable Vietnamese description
}

// WeeklyPaymentFeeScheduleListResponse is the response for
// GET /admin/weekly-payment-fees.
type WeeklyPaymentFeeScheduleListResponse struct {
	Entries []WeeklyPaymentFeeScheduleEntryResponse `json:"entries"`
}

// CreateWeeklyPaymentFeeScheduleRequest is the body for
// POST /admin/weekly-payment-fees. EffectiveDate must be today or later —
// past dates would rewrite the fee on already-completed disbursements and are
// rejected at the service layer.
type CreateWeeklyPaymentFeeScheduleRequest struct {
	EffectiveDate string `json:"effectiveDate" binding:"required"` // YYYY-MM-DD
	// Percentage has no binding guard: 0 is a valid value (a fully free
	// schedule) and float64 rules out type violations. Range validation
	// (0–100) happens in the domain Validate().
	Percentage float64 `json:"percentage"`
	Notes      string  `json:"notes,omitempty"`
}

// UpdateWeeklyPaymentFeeScheduleRequest is the body for
// PATCH /admin/weekly-payment-fees/{id}. Same shape as create — the service
// replaces the entry wholesale rather than patching individual fields.
// Updates are only allowed on entries whose effective_date is in the future.
type UpdateWeeklyPaymentFeeScheduleRequest struct {
	EffectiveDate string  `json:"effectiveDate" binding:"required"` // YYYY-MM-DD
	Percentage    float64 `json:"percentage"`
	Notes         string  `json:"notes,omitempty"`
}
