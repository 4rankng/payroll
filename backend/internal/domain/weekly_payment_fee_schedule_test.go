package domain

import (
	"testing"
	"time"
)

func TestWeeklyPaymentFeeScheduleValidation(t *testing.T) {
	tests := []struct {
		name    string
		entry   WeeklyPaymentFeeScheduleEntry
		wantErr bool
	}{
		{"valid 2 percent", WeeklyPaymentFeeScheduleEntry{ID: "x1", EffectiveDate: "2026-10-01", Percentage: 2}, false},
		{"valid 1.8 percent", WeeklyPaymentFeeScheduleEntry{ID: "x2", EffectiveDate: "2026-10-01", Percentage: 1.8}, false},
		{"zero percent allowed", WeeklyPaymentFeeScheduleEntry{ID: "x3", EffectiveDate: "2026-10-01", Percentage: 0}, false},
		{"negative rejected", WeeklyPaymentFeeScheduleEntry{ID: "x4", EffectiveDate: "2026-10-01", Percentage: -1}, true},
		{"over 100 rejected", WeeklyPaymentFeeScheduleEntry{ID: "x5", EffectiveDate: "2026-10-01", Percentage: 101}, true},
		{"missing id", WeeklyPaymentFeeScheduleEntry{EffectiveDate: "2026-10-01", Percentage: 2}, true},
		{"bad date", WeeklyPaymentFeeScheduleEntry{ID: "x6", EffectiveDate: "01/10/2026", Percentage: 2}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestActiveWeeklyPaymentFeeScheduleAt(t *testing.T) {
	entries := []WeeklyPaymentFeeScheduleEntry{
		{ID: "a", EffectiveDate: "2020-01-01", Percentage: 2},
		{ID: "b", EffectiveDate: "2026-10-01", Percentage: 1.8},
	}
	at := time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)
	if got := ActiveWeeklyPaymentFeeScheduleAt(entries, at); got == nil || got.ID != "a" {
		t.Fatalf("before Oct 1 expected entry a, got %+v", got)
	}
	at = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if got := ActiveWeeklyPaymentFeeScheduleAt(entries, at); got == nil || got.ID != "b" {
		t.Fatalf("on Oct 1 expected entry b, got %+v", got)
	}
	at = time.Date(2019, 12, 31, 0, 0, 0, 0, time.UTC)
	if got := ActiveWeeklyPaymentFeeScheduleAt(entries, at); got != nil {
		t.Fatalf("before all entries expected nil, got %+v", got)
	}
}

func TestFormatWeeklyPaymentFeeSummary(t *testing.T) {
	if got := FormatWeeklyPaymentFeeSummary(WeeklyPaymentFeeScheduleEntry{Percentage: 1.8}); got != "1,8%" {
		t.Fatalf("1.8 formatted as %q, want \"1,8%%\"", got)
	}
	if got := FormatWeeklyPaymentFeeSummary(WeeklyPaymentFeeScheduleEntry{Percentage: 2}); got != "2%" {
		t.Fatalf("2 formatted as %q, want \"2%%\"", got)
	}
}
