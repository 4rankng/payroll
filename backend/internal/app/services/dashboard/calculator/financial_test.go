package calculator

import (
	"testing"
)

func TestCalculatePercentageChangeFloat64(t *testing.T) {
	tests := []struct {
		name     string
		previous float64
		current  float64
		want     float64
	}{
		{
			name:     "positive change",
			previous: 100.0,
			current:  150.0,
			want:     50.0,
		},
		{
			name:     "negative change",
			previous: 100.0,
			current:  75.0,
			want:     -25.0,
		},
		{
			name:     "no change",
			previous: 100.0,
			current:  100.0,
			want:     0.0,
		},
		{
			name:     "previous is zero and current is positive",
			previous: 0.0,
			current:  100.0,
			want:     100.0,
		},
		{
			name:     "both are zero",
			previous: 0.0,
			current:  0.0,
			want:     0.0,
		},
		{
			name:     "previous is zero and current is negative",
			previous: 0.0,
			current:  -50.0,
			want:     0.0,
		},
		{
			name:     "rounding test",
			previous: 3.0,
			current:  4.0,
			want:     33.33,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePercentageChangeFloat64(tt.previous, tt.current)
			if got != tt.want {
				t.Errorf("CalculatePercentageChangeFloat64(%v, %v) = %v, want %v", tt.previous, tt.current, got, tt.want)
			}
		})
	}
}

func TestCalculateProfitMargin(t *testing.T) {
	tests := []struct {
		name     string
		revenue  int64
		expenses int64
		want     float64
	}{
		{
			name:     "positive profit",
			revenue:  1000,
			expenses: 600,
			want:     40.0,
		},
		{
			name:     "no profit",
			revenue:  1000,
			expenses: 1000,
			want:     0.0,
		},
		{
			name:     "loss",
			revenue:  1000,
			expenses: 1200,
			want:     -20.0,
		},
		{
			name:     "zero revenue",
			revenue:  0,
			expenses: 100,
			want:     0.0,
		},
		{
			name:     "zero expenses",
			revenue:  1000,
			expenses: 0,
			want:     100.0,
		},
		{
			name:     "rounding test",
			revenue:  3,
			expenses: 1,
			want:     66.67,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateProfitMargin(tt.revenue, tt.expenses)
			if got != tt.want {
				t.Errorf("CalculateProfitMargin(%v, %v) = %v, want %v", tt.revenue, tt.expenses, got, tt.want)
			}
		})
	}
}
