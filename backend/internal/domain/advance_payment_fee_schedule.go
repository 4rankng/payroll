package domain

import (
	"sort"
	"time"
)

// AdvancePaymentFeeScheduleSettingsKey is the Settings.key under which the JSON
// array of advance-payment fee schedules lives. The whole list is one settings
// row; per-entry CRUD is done by parsing the JSON, mutating the slice, and
// writing the row back inside a SELECT … FOR UPDATE.
const AdvancePaymentFeeScheduleSettingsKey = "advance_payment_fee_schedules"

// AdvancePaymentFeeScheduleDateLayout is the canonical wire format for
// FeeScheduleEntry.EffectiveDate. Stored and exchanged as YYYY-MM-DD so that
// string-comparison ordering matches chronological ordering.
const AdvancePaymentFeeScheduleDateLayout = "2006-01-02"

// FeeScheduleTier is one tier inside a fee schedule. Tiers are sorted ascending
// by MinAmount; the applicable tier for a request is the last tier whose
// MinAmount <= request amount.
type FeeScheduleTier struct {
	MinAmount  uint64  `json:"min_amount"` // inclusive lower bound, VND
	Percentage float64 `json:"percentage"` // e.g. 2.0 for 2%
}

// FeeScheduleEntry is one row in the fee schedule history. Entries are
// totally ordered by EffectiveDate; the active entry for a transaction is the
// one with the latest EffectiveDate <= transaction date.
//
// ID is a stable per-entry identifier (UUID) so PATCH/DELETE can target a
// single entry inside the JSON array.
type FeeScheduleEntry struct {
	ID              string            `json:"id"`
	EffectiveDate   string            `json:"effective_date"` // YYYY-MM-DD
	Tiers           []FeeScheduleTier `json:"tiers"`
	MinFeeVND       uint64            `json:"min_fee_vnd"`
	Notes           string            `json:"notes,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	CreatedByUserID *uint             `json:"created_by_user_id,omitempty"`
}

// Validate enforces the structural invariants: at least one tier, first tier
// has MinAmount == 0, tiers strictly increasing by MinAmount, percentages in
// [0, 100], MinFeeVND >= 0 (0 disables the minimum-fee floor, allowing a
// fully free advance), and a parseable effective_date.
func (e *FeeScheduleEntry) Validate() error {
	if e.ID == "" {
		return NewValidationError("fee schedule entry id is required")
	}
	if _, err := time.Parse(AdvancePaymentFeeScheduleDateLayout, e.EffectiveDate); err != nil {
		return NewValidationError("effective_date must be in YYYY-MM-DD format")
	}
	if len(e.Tiers) == 0 {
		return NewValidationError("at least one tier is required")
	}
	if e.Tiers[0].MinAmount != 0 {
		return NewValidationError("first tier must have min_amount = 0")
	}
	prev := uint64(0)
	for i, tier := range e.Tiers {
		if tier.Percentage < 0 || tier.Percentage > 100 {
			return NewValidationError("tier percentage must be between 0 and 100")
		}
		if i > 0 && tier.MinAmount <= prev {
			return NewValidationError("tiers must be strictly increasing by min_amount")
		}
		prev = tier.MinAmount
	}
	return nil
}

// EffectiveDateAsTime parses EffectiveDate into a time.Time at UTC midnight.
// Used for comparing schedules during fee resolution.
func (e *FeeScheduleEntry) EffectiveDateAsTime() time.Time {
	t, _ := time.Parse(AdvancePaymentFeeScheduleDateLayout, e.EffectiveDate)
	return t
}

// ResolveFee returns the fee in VND for the given request amount under this
// schedule entry. Picks the highest-MinAmount tier whose MinAmount <= amount,
// then applies max(amount * percentage/100, MinFeeVND).
func (e *FeeScheduleEntry) ResolveFee(requestAmount uint64) uint64 {
	tier := e.tierFor(requestAmount)
	percentageFee := uint64(float64(requestAmount) * tier.Percentage / 100.0)
	if percentageFee > e.MinFeeVND {
		return percentageFee
	}
	return e.MinFeeVND
}

// tierFor returns the applicable tier for amount. Assumes Tiers is non-empty
// and sorted (Validate enforces both).
func (e *FeeScheduleEntry) tierFor(amount uint64) FeeScheduleTier {
	applicable := e.Tiers[0]
	for _, tier := range e.Tiers {
		if tier.MinAmount <= amount {
			applicable = tier
		} else {
			break
		}
	}
	return applicable
}

// SortFeeScheduleEntries sorts entries by EffectiveDate ascending in place.
// Callers that want descending order should reverse afterwards.
func SortFeeScheduleEntries(entries []FeeScheduleEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].EffectiveDate < entries[j].EffectiveDate
	})
}

// ActiveFeeScheduleAt returns the entry that is in effect on date `at`. The
// rule is: the entry with the latest EffectiveDate that is <= at. Returns nil
// if there are no entries with EffectiveDate <= at (e.g. all entries are
// future-dated).
func ActiveFeeScheduleAt(entries []FeeScheduleEntry, at time.Time) *FeeScheduleEntry {
	atDate := at.Format(AdvancePaymentFeeScheduleDateLayout)
	var active *FeeScheduleEntry
	for i := range entries {
		if entries[i].EffectiveDate <= atDate {
			if active == nil || entries[i].EffectiveDate > active.EffectiveDate {
				active = &entries[i]
			}
		}
	}
	return active
}
