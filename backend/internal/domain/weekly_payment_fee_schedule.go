package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// WeeklyPaymentFeeScheduleSettingsKey is the Settings.key under which the JSON
// array of weekly-payment fee schedules lives. The whole list is one settings
// row; per-entry CRUD is done by parsing the JSON, mutating the slice, and
// writing the row back inside a SELECT … FOR UPDATE.
//
// This schedule drives the service fee charged on weekly salary disbursements
// (lương tuần): partner receivable = transfer × (1 + fee%), statement fee
// line, and revenue_receivable updates. It is deliberately separate from
// AdvancePaymentFeeScheduleSettingsKey (FlexPay advance fees).
const WeeklyPaymentFeeScheduleSettingsKey = "weekly_payment_fee_schedules"

// WeeklyPaymentFeeScheduleDateLayout is the canonical wire format for
// WeeklyPaymentFeeScheduleEntry.EffectiveDate. Stored and exchanged as
// YYYY-MM-DD so string-comparison ordering matches chronological ordering.
const WeeklyPaymentFeeScheduleDateLayout = "2006-01-02"

// WeeklyPaymentFeeScheduleEntry is one row in the weekly-payment fee schedule
// history. Entries are totally ordered by EffectiveDate; the active entry for
// a disbursement is the one with the latest EffectiveDate <= disbursement
// date.
type WeeklyPaymentFeeScheduleEntry struct {
	ID              string    `json:"id"`
	EffectiveDate   string    `json:"effective_date"` // YYYY-MM-DD
	Percentage      float64   `json:"percentage"`     // e.g. 2 for 2%
	Notes           string    `json:"notes,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedByUserID *uint     `json:"created_by_user_id,omitempty"`
}

// Validate enforces structural invariants: a parseable effective_date and a
// percentage within [0, 100].
func (e *WeeklyPaymentFeeScheduleEntry) Validate() error {
	if e.ID == "" {
		return NewValidationError("weekly payment fee schedule entry id is required")
	}
	if _, err := time.Parse(WeeklyPaymentFeeScheduleDateLayout, e.EffectiveDate); err != nil {
		return NewValidationError("effective_date must be in YYYY-MM-DD format")
	}
	if e.Percentage < 0 || e.Percentage > 100 {
		return NewValidationError("percentage must be between 0 and 100")
	}
	return nil
}

// SortWeeklyPaymentFeeScheduleEntries sorts entries by EffectiveDate ascending
// in place. Callers that want descending order should reverse afterwards.
func SortWeeklyPaymentFeeScheduleEntries(entries []WeeklyPaymentFeeScheduleEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].EffectiveDate < entries[j].EffectiveDate
	})
}

// ActiveWeeklyPaymentFeeScheduleAt returns the entry that is in effect on
// date `at`: the entry with the latest EffectiveDate <= at. Returns nil if no
// entry qualifies (e.g. all entries are future-dated).
func ActiveWeeklyPaymentFeeScheduleAt(entries []WeeklyPaymentFeeScheduleEntry, at time.Time) *WeeklyPaymentFeeScheduleEntry {
	atDate := at.Format(WeeklyPaymentFeeScheduleDateLayout)
	var active *WeeklyPaymentFeeScheduleEntry
	for i := range entries {
		if entries[i].EffectiveDate <= atDate {
			if active == nil || entries[i].EffectiveDate > active.EffectiveDate {
				active = &entries[i]
			}
		}
	}
	return active
}

// FormatWeeklyPaymentFeeSummary builds a Vietnamese one-line description of an
// entry for audit messages and admin UI summaries. Example: "2%" or "1,8%".
func FormatWeeklyPaymentFeeSummary(entry WeeklyPaymentFeeScheduleEntry) string {
	s := fmt.Sprintf("%g", entry.Percentage)
	s = strings.ReplaceAll(s, ".", ",")
	return s + "%"
}
