package persistence

import "fmt"

// checkInEnabledScope returns an EXISTS subquery that restricts rows to
// employees with at least one active, check-in-enabled project assignment.
// tableAlias is the outer table/alias whose employee_id and project_id
// columns are correlated (e.g. "advance_payments", "attendances", "ap").
func checkInEnabledScope(tableAlias string) string {
	return fmt.Sprintf(
		"EXISTS (SELECT 1 FROM project_employees pe WHERE pe.employee_id = %s.employee_id AND pe.project_id = %s.project_id AND pe.deleted_at IS NULL AND pe.last_date IS NULL AND pe.check_in_enabled = 1)",
		tableAlias, tableAlias,
	)
}

// checkInFailedAttemptScope is the check-in scope for the attendance_failed_attempts
// table. Unlike the general checkInEnabledScope, it also matches rows with
// project_id = 0 (checkout failures recorded with no resolvable project) so
// they are not silently excluded from dashboard counts.
func checkInFailedAttemptScope() string {
	return `EXISTS (SELECT 1 FROM project_employees pe WHERE pe.employee_id = attendance_failed_attempts.employee_id AND (attendance_failed_attempts.project_id = 0 OR (pe.project_id = attendance_failed_attempts.project_id AND pe.deleted_at IS NULL AND pe.last_date IS NULL)) AND pe.check_in_enabled = 1)`
}
