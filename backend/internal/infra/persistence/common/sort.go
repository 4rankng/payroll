package common

import (
	"regexp"
	"strings"
)

// sortColumnPattern matches a single SQL identifier, optionally qualified with
// one table prefix (e.g. "created_at" or "timesheets.date").
//
// SQL column names used in ORDER BY cannot be bound as parameterized values,
// so user-controlled sort fields must be validated before interpolation. Any
// value that does not match this pattern — including whitespace, parentheses,
// commas, quotes, semicolons, comment markers or arithmetic operators — cannot
// form a SQL injection inside an ORDER BY clause and is rejected. This is the
// SQL-injection boundary for sort fields.
var sortColumnPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

// SanitizeSortColumn validates a user-supplied sort column against a safe
// identifier shape, returning defaultColumn when the input is empty or unsafe.
//
// defaultColumn is treated as trusted (it is developer-controlled, not user
// input) and is returned verbatim, so legitimate multi-token defaults are
// preserved. Only the caller-supplied sortBy is constrained.
func SanitizeSortColumn(sortBy, defaultColumn string) string {
	sortBy = strings.TrimSpace(sortBy)
	if sortBy == "" || !sortColumnPattern.MatchString(sortBy) {
		return defaultColumn
	}
	return sortBy
}

// SanitizeSortOrder returns a safe sort direction: exactly "ASC" or "DESC".
// Any value other than ASC/DESC (case-insensitive) yields defaultDir, which
// itself falls back to "DESC" when unset or invalid.
func SanitizeSortOrder(sortOrder, defaultDir string) string {
	up := strings.ToUpper(strings.TrimSpace(sortOrder))
	if up != "ASC" && up != "DESC" {
		up = strings.ToUpper(strings.TrimSpace(defaultDir))
		if up != "ASC" && up != "DESC" {
			up = "DESC"
		}
	}
	return up
}
