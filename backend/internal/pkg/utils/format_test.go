package utils

import (
	"testing"
)

func TestRoundToTwoDecimals(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  float64
	}{
		{name: "already rounded", value: 1.23, want: 1.23},
		{name: "round up", value: 1.235, want: 1.24},
		{name: "round down", value: 1.234, want: 1.23},
		{name: "zero", value: 0.0, want: 0.0},
		{name: "negative round up", value: -1.235, want: -1.24},
		{name: "negative round down", value: -1.234, want: -1.23},
		{name: "large number", value: 123456.789, want: 123456.79},
		{name: "very small", value: 0.001, want: 0.0},
		{name: "one decimal", value: 1.5, want: 1.5},
		{name: "three decimals round up", value: 9.999, want: 10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoundToTwoDecimals(tt.value)
			if got != tt.want {
				t.Errorf("RoundToTwoDecimals(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestFormatFloatMap(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]float64
		want  map[string]float64
	}{
		{
			name:  "empty map",
			input: map[string]float64{},
			want:  map[string]float64{},
		},
		{
			name: "single value",
			input: map[string]float64{
				"price": 12.345,
			},
			want: map[string]float64{
				"price": 12.35,
			},
		},
		{
			name: "multiple values",
			input: map[string]float64{
				"price":    12.345,
				"tax":      1.567,
				"discount": 0.999,
			},
			want: map[string]float64{
				"price":    12.35,
				"tax":      1.57,
				"discount": 1.0,
			},
		},
		{
			name: "already rounded values",
			input: map[string]float64{
				"a": 1.00,
				"b": 2.50,
			},
			want: map[string]float64{
				"a": 1.00,
				"b": 2.50,
			},
		},
		{
			name: "negative values",
			input: map[string]float64{
				"debit":  -100.456,
				"credit": 50.123,
			},
			want: map[string]float64{
				"debit":  -100.46,
				"credit": 50.12,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFloatMap(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("FormatFloatMap() returned map of length %v, want %v", len(got), len(tt.want))
				return
			}
			for key, wantValue := range tt.want {
				gotValue, exists := got[key]
				if !exists {
					t.Errorf("FormatFloatMap() missing key %v", key)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("FormatFloatMap()[%v] = %v, want %v", key, gotValue, wantValue)
				}
			}
		})
	}
}

func TestFormatVND(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
		want   string
	}{
		{name: "zero", amount: 0, want: "0 ₫"},
		{name: "small positive", amount: 100, want: "100 ₫"},
		{name: "thousand", amount: 1000, want: "1.000 ₫"},
		{name: "million", amount: 1000000, want: "1.000.000 ₫"},
		{name: "five million", amount: 5000000, want: "5.000.000 ₫"},
		{name: "irregular", amount: 12345678, want: "12.345.678 ₫"},
		{name: "small negative", amount: -100, want: "-100 ₫"},
		{name: "negative thousand", amount: -1000, want: "-1.000 ₫"},
		{name: "negative million", amount: -5000000, want: "-5.000.000 ₫"},
		{name: "one", amount: 1, want: "1 ₫"},
		{name: "99", amount: 99, want: "99 ₫"},
		{name: "999", amount: 999, want: "999 ₫"},
		{name: "1001", amount: 1001, want: "1.001 ₫"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatVND(tt.amount)
			if got != tt.want {
				t.Errorf("FormatVND(%v) = %v, want %v", tt.amount, got, tt.want)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
		want   string
	}{
		{name: "zero", amount: 0, want: "0"},
		{name: "small positive", amount: 100, want: "100"},
		{name: "thousand", amount: 1000, want: "1.000"},
		{name: "million", amount: 1000000, want: "1.000.000"},
		{name: "five million", amount: 5000000, want: "5.000.000"},
		{name: "irregular", amount: 12345678, want: "12.345.678"},
		{name: "negative million", amount: -5000000, want: "-5.000.000"},
		{name: "one", amount: 1, want: "1"},
		{name: "999", amount: 999, want: "999"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatNumber(tt.amount)
			if got != tt.want {
				t.Errorf("FormatNumber(%v) = %v, want %v", tt.amount, got, tt.want)
			}
		})
	}
}
