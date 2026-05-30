package middleware

import (
	"testing"
)

func TestIsEmployeeSpecificRoute(t *testing.T) {
	m := &AuthorizationMiddleware{}

	tests := []struct {
		path string
		want bool
		desc string
	}{
		// Aggregate routes — must NOT trigger object-level check
		{"/api/v1/employees/summary", false, "aggregate summary should be excluded"},
		{"/api/v1/employees/unassigned", false, "unassigned list should be excluded"},
		{"/api/v1/employees/export", false, "export should be excluded"},
		{"/api/v1/employees/init-users", false, "init-users should be excluded"},
		{"/api/v1/employees/cccd/123456789", false, "cccd lookup should be excluded"},
		{"/api/v1/employees/missing-bank-details", false, "missing-bank-details has no numeric ID, extractEmployeeID returns nil"},

		// Per-employee sub-routes — MUST trigger object-level check
		{"/api/v1/employees/18/summary", true, "per-employee summary must NOT be excluded (was the bug)"},
		{"/api/v1/employees/18/payroll", true, "per-employee payroll must be checked"},
		{"/api/v1/employees/18/timesheet", true, "per-employee timesheet must be checked"},
		{"/api/v1/employees/18/timesheets/summary", true, "per-employee timesheets/summary must be checked"},
		{"/api/v1/employees/18/current-projects", true, "per-employee current-projects must be checked"},
		{"/api/v1/employees/18", true, "employee detail must be checked"},

		// Excluded by their own handler-level logic
		{"/api/v1/employees/18/change-password", false, "change-password has own logic"},
		{"/api/v1/employees/18/users", false, "users has own creator check"},
		{"/api/v1/employees/18/users/5", false, "users/:id has own creator check"},

		// Non-employee routes
		{"/api/v1/projects/18", false, "project route should not match"},
		{"/api/v1/timesheets/18", false, "timesheet route should not match"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := m.isEmployeeSpecificRoute(tt.path)
			if got != tt.want {
				t.Errorf("isEmployeeSpecificRoute(%q) = %v, want %v — %s", tt.path, got, tt.want, tt.desc)
			}
		})
	}
}
