package domain

import "time"

// PayrollSummary represents a summary of payroll calculations for a period
type PayrollSummary struct {
	FromDate      time.Time         `json:"fromDate"`
	ToDate        time.Time         `json:"toDate"`
	TotalAmount   int64             `json:"total_amount"`
	TotalHours    float64           `json:"total_hours"`
	EmployeeCount map[uint]struct{} `json:"-"` // Internal use, count unique employees
	ProjectCosts  map[uint]int64    `json:"project_costs"`
	GeneratedAt   time.Time         `json:"generated_at"`
}

// GetEmployeeCount returns the number of unique employees in the payroll
func (ps *PayrollSummary) GetEmployeeCount() int {
	return len(ps.EmployeeCount)
}
