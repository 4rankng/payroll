package dto

import "time"

// FeeScheduleTierDTO is the wire shape of one tier in a schedule entry.
type FeeScheduleTierDTO struct {
	MinAmount  uint64  `json:"minAmount"`
	Percentage float64 `json:"percentage"`
}

// FeeScheduleEntryResponse is the wire shape returned by the admin fee
// schedule endpoints. IsCurrentlyActive is computed server-side against
// time.Now(): exactly one entry is active at any moment.
type FeeScheduleEntryResponse struct {
	ID                string               `json:"id"`
	EffectiveDate     string               `json:"effectiveDate"` // YYYY-MM-DD
	Tiers             []FeeScheduleTierDTO `json:"tiers"`
	MinFeeVND         uint64               `json:"minFeeVnd"`
	Notes             string               `json:"notes,omitempty"`
	CreatedAt         time.Time            `json:"createdAt"`
	CreatedByUserID   *uint                `json:"createdByUserId,omitempty"`
	IsCurrentlyActive bool                 `json:"isCurrentlyActive"`
	IsPending         bool                 `json:"isPending"` // effective_date > today
	Summary           string               `json:"summary"`   // human-readable Vietnamese description
}

// FeeScheduleListResponse is the response for GET /admin/advance-payment-fees.
type FeeScheduleListResponse struct {
	Entries []FeeScheduleEntryResponse `json:"entries"`
}

// CreateFeeScheduleRequest is the body for POST /admin/advance-payment-fees.
// EffectiveDate must be today or later — past dates would rewrite history of
// already-paid transactions and are rejected at the service layer.
type CreateFeeScheduleRequest struct {
	EffectiveDate string               `json:"effectiveDate" binding:"required"` // YYYY-MM-DD
	Tiers         []FeeScheduleTierDTO `json:"tiers" binding:"required,min=1"`
	// MinFeeVND has no binding guard: 0 is a valid value (no minimum-fee
	// floor, e.g. a fully free schedule) and uint64 rules out negatives.
	MinFeeVND uint64 `json:"minFeeVnd"`
	Notes     string `json:"notes,omitempty"`
}

// UpdateFeeScheduleRequest is the body for PATCH /admin/advance-payment-fees/{id}.
// Same shape as create — all fields are required because the service replaces
// the entry wholesale rather than patching individual fields. Updates are only
// allowed on entries whose effective_date is in the future.
type UpdateFeeScheduleRequest struct {
	EffectiveDate string               `json:"effectiveDate" binding:"required"`
	Tiers         []FeeScheduleTierDTO `json:"tiers" binding:"required,min=1"`
	// MinFeeVND has no binding guard: 0 is a valid value (no minimum-fee
	// floor, e.g. a fully free schedule) and uint64 rules out negatives.
	MinFeeVND uint64 `json:"minFeeVnd"`
	Notes     string `json:"notes,omitempty"`
}
