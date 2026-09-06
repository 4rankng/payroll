package excelkit

import (
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Cell reads one cell's formatted string value by 1-based coordinates and
// trims it. Coordinate or read errors return "" — callers treat that as a
// blank cell, matching the BCC parsers' long-standing contract.
//
// Verbatim port of bccCell (services/excel).
func Cell(f *excelize.File, sheet string, col, row int) string {
	cn, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return ""
	}
	val, err := f.GetCellValue(sheet, cn)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(val)
}

// RowCell indexes into an already-materialized row slice with bounds
// guarding; out-of-range indexes return "".
//
// Verbatim port of settlement's cell helper.
func RowCell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return row[idx]
}

// NumericCell reads one cell as a raw number, bypassing the cell's display
// format (so 7.5h shown as "8" still reads 7.5), and reports whether the
// cell carried an explicit numeric value. Blank and non-numeric cells return
// (0, false); a cell storing an explicit 0 returns (0, true) — imports read
// that as a deletion request for the day.
//
// Verbatim port of dateRowNumericCell (services/excel).
func NumericCell(f *excelize.File, sheet string, col, row int) (float64, bool) {
	cn, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return 0, false
	}
	val, err := f.GetCellValue(sheet, cn, excelize.Options{RawCellValue: true})
	if err != nil || val == "" {
		return 0, false
	}
	hours, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
	if err != nil {
		return 0, false
	}
	return hours, true
}
