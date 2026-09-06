package excelkit

import (
	"math"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"
)

// ExcelDate parses an Excel date serial number cell value into a time.Time.
// Fractional parts (time-of-day) are floored off; dates outside 2020–2040
// are rejected so stray numbers never masquerade as dates.
//
// Verbatim port of parseExcelDate (services/excel weekly BCC parser). The
// returned time carries no location — callers derive business dates from it
// via the clock package, never time.Local.
func ExcelDate(val string) (time.Time, bool) {
	serial, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return time.Time{}, false
	}
	if serial < 1 {
		return time.Time{}, false
	}
	serial = math.Floor(serial)
	// Excel epoch: 1900-01-01 = serial 1 (with the Lotus 123 bug where
	// 1900-02-29 = serial 60); excelize.ExcelDateToTime handles it.
	t, err := excelize.ExcelDateToTime(serial, false)
	if err != nil {
		return time.Time{}, false
	}
	if t.Year() < 2020 || t.Year() > 2040 {
		return time.Time{}, false
	}
	return t, true
}
