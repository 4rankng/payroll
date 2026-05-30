package domain

import (
	"testing"
	"time"
)

// TestDisbursementFeeScheduleResolvesByDate proves the core invariant: for any
// given query date, ActiveDisbursementFeeScheduleAt returns the entry with the
// latest effective_date that is <= the query date. This is the smoke test the
// user asked for: a schedule entry at fee=300 effective tomorrow should not
// override today's transfers, and an entry effective today should win
// immediately.
func TestDisbursementFeeScheduleResolvesByDate(t *testing.T) {
	parse := func(s string) time.Time {
		t.Helper()
		v, err := time.Parse(DisbursementFeeScheduleDateLayout, s)
		if err != nil {
			t.Fatalf("parse %q: %v", s, err)
		}
		return v
	}

	bootstrap := DisbursementFeeScheduleEntry{
		ID:            "bootstrap",
		EffectiveDate: "2020-01-01",
		FeeVND:        200,
	}
	tomorrowEntry := DisbursementFeeScheduleEntry{
		ID:            "tomorrow",
		EffectiveDate: "2026-05-08", // tomorrow (today is 2026-05-07 per the brief)
		FeeVND:        300,
	}
	todayEntry := DisbursementFeeScheduleEntry{
		ID:            "today",
		EffectiveDate: "2026-05-07",
		FeeVND:        250,
	}

	tests := []struct {
		name    string
		entries []DisbursementFeeScheduleEntry
		at      time.Time
		want    int64
		wantID  string
	}{
		{
			name:    "only bootstrap entry → resolves to bootstrap fee",
			entries: []DisbursementFeeScheduleEntry{bootstrap},
			at:      parse("2026-05-07"),
			want:    200,
			wantID:  "bootstrap",
		},
		{
			name:    "future entry exists, today still resolves to bootstrap",
			entries: []DisbursementFeeScheduleEntry{bootstrap, tomorrowEntry},
			at:      parse("2026-05-07"),
			want:    200,
			wantID:  "bootstrap",
		},
		{
			name:    "future entry's date arrives → resolves to future entry",
			entries: []DisbursementFeeScheduleEntry{bootstrap, tomorrowEntry},
			at:      parse("2026-05-08"),
			want:    300,
			wantID:  "tomorrow",
		},
		{
			name:    "today-effective entry beats bootstrap on same date",
			entries: []DisbursementFeeScheduleEntry{bootstrap, todayEntry},
			at:      parse("2026-05-07"),
			want:    250,
			wantID:  "today",
		},
		{
			name:    "all-future entries → no active schedule",
			entries: []DisbursementFeeScheduleEntry{tomorrowEntry},
			at:      parse("2026-05-07"),
			want:    0, // sentinel; check via wantID = "" instead
			wantID:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ActiveDisbursementFeeScheduleAt(tc.entries, "", tc.at)
			if tc.wantID == "" {
				if got != nil {
					t.Fatalf("expected nil active entry, got id=%s fee=%d", got.ID, got.FeeVND)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected entry id=%s, got nil", tc.wantID)
			}
			if got.ID != tc.wantID {
				t.Fatalf("expected id=%s, got id=%s", tc.wantID, got.ID)
			}
			if got.FeeVND != tc.want {
				t.Fatalf("expected fee=%d, got fee=%d", tc.want, got.FeeVND)
			}
		})
	}
}
