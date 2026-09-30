package domain

import "time"

// EmployeePaidActivity is a read model for the admin data-quality export:
// one row per employee who received salary (paid timesheets) or a completed
// FlexPay advance within a window but has no mobile number on file. Payment
// recency is the activity proxy — last_activity_at is the later of the two
// last-paid dates.
type EmployeePaidActivity struct {
	EmployeeID        uint       `json:"employee_id"`
	Fullname          string     `json:"fullname"`
	CCCD              string     `json:"cccd"`
	IsWorking         bool       `json:"is_working"`
	SalaryTotal       int64      `json:"salary_total"`
	SalaryCount       int64      `json:"salary_count"`
	LastSalaryPaidAt  *time.Time `json:"last_salary_paid_at"`
	AdvanceTotal      int64      `json:"advance_total"`
	AdvanceCount      int64      `json:"advance_count"`
	LastAdvancePaidAt *time.Time `json:"last_advance_paid_at"`
	LastActivityAt    *time.Time `json:"last_activity_at"`
	Projects          string     `json:"projects"`
}
