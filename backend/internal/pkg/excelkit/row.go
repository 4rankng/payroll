package excelkit

import "strings"

// IsEmptyRow reports whether every cell of a materialized row is blank after
// trimming. Shared shape of the wallet-bulk rowIsEmpty and bulk-transfer
// IsEmptyRow helpers.
func IsEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// FindHeaderRow scans at most maxScan rows (0-based) from the top and
// returns the index of the first row for which match returns true, or -1.
// Used by importers whose header row drifts between template versions.
func FindHeaderRow(rows [][]string, maxScan int, match func([]string) bool) int {
	limit := len(rows)
	if maxScan > 0 && maxScan < limit {
		limit = maxScan
	}
	for i := 0; i < limit; i++ {
		if match(rows[i]) {
			return i
		}
	}
	return -1
}
