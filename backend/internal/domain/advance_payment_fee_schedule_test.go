package domain

import (
	"testing"
	"time"
)

func TestFeeScheduleEntry_Validate(t *testing.T) {
	cases := []struct {
		name    string
		entry   FeeScheduleEntry
		wantErr bool
	}{
		{
			name: "valid single tier",
			entry: FeeScheduleEntry{
				ID:            "abc",
				EffectiveDate: "2025-01-01",
				Tiers:         []FeeScheduleTier{{MinAmount: 0, Percentage: 2.0}},
				MinFeeVND:     10000,
			},
		},
		{
			name: "valid two tiers ascending",
			entry: FeeScheduleEntry{
				ID:            "abc",
				EffectiveDate: "2025-01-01",
				Tiers: []FeeScheduleTier{
					{MinAmount: 0, Percentage: 2.0},
					{MinAmount: 3_500_000, Percentage: 1.3},
				},
				MinFeeVND: 10000,
			},
		},
		{
			name:    "missing id",
			entry:   FeeScheduleEntry{EffectiveDate: "2025-01-01", Tiers: []FeeScheduleTier{{Percentage: 2}}, MinFeeVND: 10000},
			wantErr: true,
		},
		{
			name: "first tier non-zero",
			entry: FeeScheduleEntry{
				ID:            "abc",
				EffectiveDate: "2025-01-01",
				Tiers:         []FeeScheduleTier{{MinAmount: 100, Percentage: 2.0}},
				MinFeeVND:     10000,
			},
			wantErr: true,
		},
		{
			name: "tiers not strictly increasing",
			entry: FeeScheduleEntry{
				ID:            "abc",
				EffectiveDate: "2025-01-01",
				Tiers: []FeeScheduleTier{
					{MinAmount: 0, Percentage: 2.0},
					{MinAmount: 0, Percentage: 1.5},
				},
				MinFeeVND: 10000,
			},
			wantErr: true,
		},
		{
			name: "percentage out of range",
			entry: FeeScheduleEntry{
				ID:            "abc",
				EffectiveDate: "2025-01-01",
				Tiers:         []FeeScheduleTier{{MinAmount: 0, Percentage: 150}},
				MinFeeVND:     10000,
			},
			wantErr: true,
		},
		{
			name: "min fee zero",
			entry: FeeScheduleEntry{
				ID:            "abc",
				EffectiveDate: "2025-01-01",
				Tiers:         []FeeScheduleTier{{MinAmount: 0, Percentage: 2.0}},
				MinFeeVND:     0,
			},
			wantErr: true,
		},
		{
			name: "bad date format",
			entry: FeeScheduleEntry{
				ID:            "abc",
				EffectiveDate: "01/01/2025",
				Tiers:         []FeeScheduleTier{{MinAmount: 0, Percentage: 2.0}},
				MinFeeVND:     10000,
			},
			wantErr: true,
		},
		{
			name:    "empty tiers",
			entry:   FeeScheduleEntry{ID: "abc", EffectiveDate: "2025-01-01", MinFeeVND: 10000},
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.entry.Validate()
			if (err != nil) != c.wantErr {
				t.Fatalf("got err=%v, wantErr=%v", err, c.wantErr)
			}
		})
	}
}

func TestFeeScheduleEntry_ResolveFee_FlatRate(t *testing.T) {
	e := FeeScheduleEntry{
		ID:            "abc",
		EffectiveDate: "2020-01-01",
		Tiers:         []FeeScheduleTier{{MinAmount: 0, Percentage: 2.0}},
		MinFeeVND:     10000,
	}

	// Below min-fee floor
	if got := e.ResolveFee(100_000); got != 10000 {
		t.Errorf("100k @ 2 percent = 2000 < 10k floor, got %d", got)
	}
	// Right at floor crossing
	if got := e.ResolveFee(500_000); got != 10000 {
		t.Errorf("500k @ 2 percent = 10000 = floor, got %d", got)
	}
	// Above floor
	if got := e.ResolveFee(1_000_000); got != 20000 {
		t.Errorf("1M @ 2%% = 20000, got %d", got)
	}
}

func TestFeeScheduleEntry_ResolveFee_Tiered(t *testing.T) {
	e := FeeScheduleEntry{
		ID:            "abc",
		EffectiveDate: "2026-06-01",
		Tiers: []FeeScheduleTier{
			{MinAmount: 0, Percentage: 2.0},
			{MinAmount: 3_500_000, Percentage: 1.3},
		},
		MinFeeVND: 10000,
	}

	expected := func(amount uint64, pct float64) uint64 {
		return uint64(float64(amount) * pct / 100.0)
	}

	// Below tier-2 boundary uses tier-1
	if got, want := e.ResolveFee(3_499_999), expected(3_499_999, 2.0); got != want {
		t.Errorf("just-below-3.5M should use 2%%, got %d want %d", got, want)
	}
	// Exactly at tier-2 boundary uses tier-2
	if got, want := e.ResolveFee(3_500_000), expected(3_500_000, 1.3); got != want {
		t.Errorf("exactly-3.5M should use 1.3%%, got %d want %d", got, want)
	}
	// Above tier-2
	if got, want := e.ResolveFee(5_000_000), expected(5_000_000, 1.3); got != want {
		t.Errorf("5M should use 1.3%%, got %d want %d", got, want)
	}
}

func TestActiveFeeScheduleAt(t *testing.T) {
	entries := []FeeScheduleEntry{
		{ID: "a", EffectiveDate: "2020-01-01", Tiers: []FeeScheduleTier{{MinAmount: 0, Percentage: 2}}, MinFeeVND: 10000},
		{ID: "b", EffectiveDate: "2026-06-01", Tiers: []FeeScheduleTier{{MinAmount: 0, Percentage: 1.5}}, MinFeeVND: 10000},
		{ID: "c", EffectiveDate: "2030-01-01", Tiers: []FeeScheduleTier{{MinAmount: 0, Percentage: 1.0}}, MinFeeVND: 10000},
	}

	cases := []struct {
		at     string
		wantID string
	}{
		{"2019-12-31", ""},  // before all
		{"2020-01-01", "a"}, // exactly at first
		{"2020-06-15", "a"},
		{"2026-05-31", "a"}, // day before b
		{"2026-06-01", "b"}, // exactly at b
		{"2027-12-31", "b"},
		{"2030-01-01", "c"},
		{"2099-01-01", "c"},
	}

	for _, c := range cases {
		t.Run(c.at, func(t *testing.T) {
			at, _ := time.Parse(AdvancePaymentFeeScheduleDateLayout, c.at)
			active := ActiveFeeScheduleAt(entries, at)
			gotID := ""
			if active != nil {
				gotID = active.ID
			}
			if gotID != c.wantID {
				t.Errorf("at=%s want id=%q got id=%q", c.at, c.wantID, gotID)
			}
		})
	}
}
