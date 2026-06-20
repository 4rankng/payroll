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
