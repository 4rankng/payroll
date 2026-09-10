package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// queryAdvancePaymentUploadDates returns every advance_payments.upload_date value
// via the documented payroll-mysql container (see backend/CLAUDE.md). The HTTP API
// never surfaces upload_date, so regression guards that must assert its on-disk
// format query the DB directly. Uses -N -B (tab-separated, no headers) for easy
// line splitting.
func queryAdvancePaymentUploadDates() ([]string, error) {
	out, err := exec.Command("docker", "exec", "payroll-mysql", "mysql",
		"-uroot", "-prootpassword", "payroll_db", "-N", "-B",
		"-e", "SELECT upload_date FROM advance_payments").Output()
	if err != nil {
		stderr := strings.TrimSpace(string(out))
		return nil, fmt.Errorf("query advance_payments.upload_date via payroll-mysql: %w (%s)", err, stderr)
	}
	var vals []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			vals = append(vals, line)
		}
	}
	return vals, nil
}

// isYYYYMMDD reports whether s is exactly a 10-char YYYY-MM-DD string. Kept
// dependency-free (no regexp) since the format is structurally simple.
func isYYYYMMDD(s string) bool {
	return len(s) == 10 && s[4] == '-' && s[7] == '-'
}

// execPayrollSQL runs one statement against the dev DB inside payroll-mysql.
func execPayrollSQL(sql string) error {
	out, err := exec.Command("docker", "exec", "payroll-mysql", "mysql",
		"-uroot", "-prootpassword", "payroll_db", "-N", "-B", "-e", sql).CombinedOutput()
	if err != nil {
		return fmt.Errorf("exec %q: %w (%s)", sql, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// queryPayrollScalar returns the first column of the first row produced by the
// query, or "" when the query yields no rows.
func queryPayrollScalar(sql string) (string, error) {
	out, err := exec.Command("docker", "exec", "payroll-mysql", "mysql",
		"-uroot", "-prootpassword", "payroll_db", "-N", "-B", "-e", sql).Output()
	if err != nil {
		stderr := strings.TrimSpace(string(out))
		return "", fmt.Errorf("query via payroll-mysql: %w (%s)", err, stderr)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line, nil
		}
	}
	return "", nil
}

// plantPaidTimesheet inserts an approved+paid timesheet row for the
// project/employee pair so guard behavior can be exercised end-to-end. The
// payrate/user references are sourced from an existing row to satisfy the
// foreign keys; the row itself belongs to this test run only.
func plantPaidTimesheet(projectID, employeeID uint, date string) error {
	ref, err := queryPayrollScalar("SELECT CONCAT(payrate_id, ':', created_by) FROM timesheets ORDER BY id LIMIT 1")
	if err != nil {
		return err
	}
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("no reference timesheet to source payrate_id/created_by (got %q)", ref)
	}
	return execPayrollSQL(fmt.Sprintf(
		"INSERT INTO timesheets (project_id, employee_id, payrate_id, date, hours_worked, paytype, payrate, amount, timesheet_status, payment_status, created_by, created_at, updated_at) "+
			"VALUES (%d, %d, %s, '%s', 8.00, 'Cong test', 100000, 800000, 'approved', 'paid', %s, NOW(), NOW())",
		projectID, employeeID, parts[0], date, parts[1]))
}

// deleteTimesheetsForPair removes every timesheet row for the pair (test
// fixtures planted by plantPaidTimesheet).
func deleteTimesheetsForPair(projectID, employeeID uint) error {
	return execPayrollSQL(fmt.Sprintf("DELETE FROM timesheets WHERE project_id = %d AND employee_id = %d", projectID, employeeID))
}

// getAssignmentDateColumn returns start_date/last_date for the assignment as
// YYYY-MM-DD, or "" when the column is NULL (open-ended). A missing row also
// yields "", which the caller's equality assertions turn into a failure.
func getAssignmentDateColumn(assignmentID uint, column string) (string, error) {
	return queryPayrollScalar(fmt.Sprintf(
		"SELECT IFNULL(DATE_FORMAT(%s, '%%Y-%%m-%%d'), '') FROM project_employees WHERE id = %d", column, assignmentID))
}
