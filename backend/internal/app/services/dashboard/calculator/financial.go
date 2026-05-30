package calculator

import (
	"math"
)

// CalculatePercentageChangeFloat64 calculates the percentage change between two float64 values
// Returns value rounded to 2 decimal places
func CalculatePercentageChangeFloat64(previous, current float64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100.0
		}
		return 0.0
	}
	result := (current - previous) / previous * 100
	return math.Round(result*100) / 100
}

// CalculateProfitMargin calculates the profit margin percentage given revenue and expenses
// Returns value rounded to 2 decimal places
func CalculateProfitMargin(revenue, expenses int64) float64 {
	if revenue == 0 {
		return 0
	}
	profit := revenue - expenses
	result := float64(profit) / float64(revenue) * 100
	return math.Round(result*100) / 100
}
