package utils

import (
	"fmt"
	"math"
	"strings"
)

// RoundToTwoDecimals rounds a float64 to 2 decimal places
func RoundToTwoDecimals(value float64) float64 {
	return math.Round(value*100) / 100
}

// FormatFloatMap rounds all float values in a map to 2 decimal places
func FormatFloatMap(input map[string]float64) map[string]float64 {
	result := make(map[string]float64)
	for key, value := range input {
		result[key] = RoundToTwoDecimals(value)
	}
	return result
}

// FormatVND formats an integer VND amount with period thousand separators and ₫ suffix.
// Example: 5000000 -> "5.000.000 ₫"; negatives preserved as "-5.000.000 ₫".
func FormatVND(amount int64) string {
	if amount == 0 {
		return "0 ₫"
	}
	neg := amount < 0
	if neg {
		amount = -amount
	}
	s := fmt.Sprintf("%d", amount)
	n := len(s)
	if n <= 3 {
		if neg {
			return "-" + s + " ₫"
		}
		return s + " ₫"
	}
	var b strings.Builder
	pre := n % 3
	if pre == 0 {
		pre = 3
	}
	b.WriteString(s[:pre])
	for i := pre; i < n; i += 3 {
		b.WriteString(".")
		b.WriteString(s[i : i+3])
	}
	out := b.String() + " ₫"
	if neg {
		out = "-" + out
	}
	return out
}
