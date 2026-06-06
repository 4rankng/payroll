package dto

import "time"

// DisbursementFeeScheduleEntryResponse is the wire shape returned by the admin
// disbursement-fee schedule endpoints. IsCurrentlyActive is computed
// server-side against time.Now(): exactly one entry is active at any moment.
type DisbursementFeeScheduleEntryResponse struct {
	ID                string    `json:"id"`
	Provider          string    `json:"provider"`          // "9pay" | "1pay"
	EffectiveDate     string    `json:"effectiveDate"`     // YYYY-MM-DD
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
	Provider      string `json:"provider" binding:"required,oneof=9pay 1pay"`
	EffectiveDate string `json:"effectiveDate" binding:"required"` // YYYY-MM-DD
	FeeVND        int64  `json:"feeVnd" binding:"min=0"`
	Notes         string `json:"notes,omitempty"`
}

// UpdateDisbursementFeeScheduleRequest is the body for
// PATCH /admin/disbursement-fees/{id}. Provider is immutable after creation
// but accepted in the request body so the frontend can echo it back.
// The service ignores provider on update — only date/fee/notes are applied.
// Updates are only allowed on entries whose effective_date is in the future.
type UpdateDisbursementFeeScheduleRequest struct {
	Provider      string `json:"provider" binding:"required,oneof=9pay 1pay"`
	EffectiveDate string `json:"effectiveDate" binding:"required"`
	FeeVND        int64  `json:"feeVnd" binding:"min=0"`
	Notes         string `json:"notes,omitempty"`
}
