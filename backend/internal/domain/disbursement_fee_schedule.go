package domain

import (
	"sort"
	"time"
)

// DisbursementFeeScheduleSettingsKey is the Settings.key under which the JSON
// array of disbursement-provider fee schedules lives. Mirrors the pattern of
// AdvancePaymentFeeScheduleSettingsKey: the whole list is one settings row;
// per-entry CRUD is done by parsing the JSON, mutating the slice, and writing
// the row back inside a SELECT … FOR UPDATE.
const DisbursementFeeScheduleSettingsKey = "disbursement_fee_schedules"

// DisbursementFeeScheduleDateLayout is the canonical wire format for
// DisbursementFeeScheduleEntry.EffectiveDate. YYYY-MM-DD so string ordering
// matches chronological ordering.
const DisbursementFeeScheduleDateLayout = "2006-01-02"

// DisbursementFeeScheduleEntry is one row in the disbursement-fee history.
// Unlike the advance-payment schedule there are no tiers and no minimum: the
// 9pay (or any future provider) charges a flat per-transfer fee in VND, so a
// single FeeVND value is all we need.
//
// Entries are totally ordered by EffectiveDate; the active entry for a transfer
// is the one with the latest EffectiveDate <= transfer date.
type DisbursementFeeScheduleEntry struct {
	ID              string    `json:"id"`
	Provider        string    `json:"provider"`       // "9pay" | "1pay"
	EffectiveDate   string    `json:"effective_date"` // YYYY-MM-DD
	FeeVND          int64     `json:"fee_vnd"`
	Notes           string    `json:"notes,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedByUserID *uint     `json:"created_by_user_id,omitempty"`
}

// Validate enforces the structural invariants: parseable effective_date,
// non-negative fee. FeeVND == 0 is allowed (the rare case where the provider
// runs a free-transfer promotion).
func (e *DisbursementFeeScheduleEntry) Validate() error {
	if e.ID == "" {
		return NewValidationError("disbursement fee schedule entry id is required")
	}
	if _, err := time.Parse(DisbursementFeeScheduleDateLayout, e.EffectiveDate); err != nil {
		return NewValidationError("effective_date must be in YYYY-MM-DD format")
	}
	if e.FeeVND < 0 {
		return NewValidationError("fee_vnd must be non-negative")
	}
	return nil
}

// EffectiveDateAsTime parses EffectiveDate into a time.Time at UTC midnight.
func (e *DisbursementFeeScheduleEntry) EffectiveDateAsTime() time.Time {
	t, _ := time.Parse(DisbursementFeeScheduleDateLayout, e.EffectiveDate)
	return t
}

// SortDisbursementFeeScheduleEntries sorts entries by EffectiveDate ascending
// in place. Callers that want descending order should reverse afterwards.
func SortDisbursementFeeScheduleEntries(entries []DisbursementFeeScheduleEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].EffectiveDate < entries[j].EffectiveDate
	})
}

// ActiveDisbursementFeeScheduleAt returns the entry for the given provider
// that is in effect on date `at`. Same rule as the advance-payment counterpart:
// the entry with the latest EffectiveDate that is <= at. Returns nil if no
// matching entry is found.
func ActiveDisbursementFeeScheduleAt(entries []DisbursementFeeScheduleEntry, provider string, at time.Time) *DisbursementFeeScheduleEntry {
	atDate := at.Format(DisbursementFeeScheduleDateLayout)
	var active *DisbursementFeeScheduleEntry
	for i := range entries {
		if entries[i].Provider != provider {
			continue
		}
		if entries[i].EffectiveDate <= atDate {
			if active == nil || entries[i].EffectiveDate > active.EffectiveDate {
				active = &entries[i]
			}
		}
	}
	return active
}
