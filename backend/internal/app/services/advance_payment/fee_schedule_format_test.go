package advance_payment

import (
	"testing"

	"api-server/internal/domain"
)

func TestFormatScheduleSummary(t *testing.T) {
	cases := []struct {
		name string
		in   domain.FeeScheduleEntry
		want string
	}{
		{
			name: "single tier integer percent",
			in: domain.FeeScheduleEntry{
				Tiers:     []domain.FeeScheduleTier{{MinAmount: 0, Percentage: 2.0}},
				MinFeeVND: 10000,
			},
			want: "2% (tối thiểu 10.000 VND)",
		},
		{
			name: "single tier decimal percent",
			in: domain.FeeScheduleEntry{
				Tiers:     []domain.FeeScheduleTier{{MinAmount: 0, Percentage: 1.3}},
				MinFeeVND: 10000,
			},
			want: "1,3% (tối thiểu 10.000 VND)",
		},
		{
			name: "two tiers",
			in: domain.FeeScheduleEntry{
				Tiers: []domain.FeeScheduleTier{
					{MinAmount: 0, Percentage: 2.0},
					{MinAmount: 3_500_000, Percentage: 1.3},
				},
				MinFeeVND: 10000,
			},
			want: "2% / 1,3% trên 3.500.000 VND (tối thiểu 10.000 VND)",
		},
		{
			name: "three tiers",
			in: domain.FeeScheduleEntry{
				Tiers: []domain.FeeScheduleTier{
					{MinAmount: 0, Percentage: 2.0},
					{MinAmount: 1_000_000, Percentage: 1.5},
					{MinAmount: 5_000_000, Percentage: 1.0},
				},
				MinFeeVND: 10000,
			},
			want: "2% / 1,5% trên 1.000.000 VND / 1% trên 5.000.000 VND (tối thiểu 10.000 VND)",
		},
		{
			name: "empty tiers returns empty",
			in:   domain.FeeScheduleEntry{MinFeeVND: 10000},
			want: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := FormatScheduleSummary(c.in)
			if got != c.want {
				t.Errorf("got=%q want=%q", got, c.want)
			}
		})
	}
}

func TestFormatVNDInt(t *testing.T) {
	cases := []struct {
		in   uint64
		want string
	}{
		{0, "0"},
		{500, "500"},
		{1_000, "1.000"},
		{10_000, "10.000"},
		{100_000, "100.000"},
		{1_000_000, "1.000.000"},
		{3_500_000, "3.500.000"},
		{12_345_678, "12.345.678"},
	}
	for _, c := range cases {
		if got := formatVNDInt(c.in); got != c.want {
			t.Errorf("formatVNDInt(%d)=%q want=%q", c.in, got, c.want)
		}
	}
}
