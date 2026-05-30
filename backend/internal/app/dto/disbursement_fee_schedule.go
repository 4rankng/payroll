package dto

import "time"

// DisbursementFeeScheduleEntryResponse is the wire shape returned by the admin
// disbursement-fee schedule endpoints. IsCurrentlyActive is computed
// server-side against time.Now(): exactly one entry is active at any moment.
type DisbursementFeeScheduleEntryResponse struct {
	ID                string    `json:"id"`
	EffectiveDate     string    `json:"effectiveDate"` // YYYY-MM-DD
	FeeVND            int64     `json:"feeVnd"`
	Notes             string    `json:"notes,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	CreatedByUserID   *uint     `json:"createdByUserId,omitempty"`
	IsCurrentlyActive bool      `json:"isCurrentlyActive"`
	IsPending         bool      `json:"isPending"` // effective_date > today
	Summary           string    `json:"summary"`   // human-readable Vietnamese description
}

// DisbursementFeeScheduleListResponse is the response for
// GET /admin/disbursement-fees.
type DisbursementFeeScheduleListResponse struct {
	Entries []DisbursementFeeScheduleEntryResponse `json:"entries"`
}

// CreateDisbursementFeeScheduleRequest is the body for
// POST /admin/disbursement-fees. EffectiveDate must be today or later — past
// dates would rewrite history of already-completed transfers and are rejected
// at the service layer. FeeVND is allowed to be 0 (free-transfer promotion).
type CreateDisbursementFeeScheduleRequest struct {
	EffectiveDate string `json:"effectiveDate" binding:"required"` // YYYY-MM-DD
	FeeVND        int64  `json:"feeVnd" binding:"min=0"`
	Notes         string `json:"notes,omitempty"`
}

// UpdateDisbursementFeeScheduleRequest is the body for
// PATCH /admin/disbursement-fees/{id}. Same shape as create — the service
// replaces the entry wholesale rather than patching individual fields.
// Updates are only allowed on entries whose effective_date is in the future.
type UpdateDisbursementFeeScheduleRequest struct {
	EffectiveDate string `json:"effectiveDate" binding:"required"`
	FeeVND        int64  `json:"feeVnd" binding:"min=0"`
	Notes         string `json:"notes,omitempty"`
}
