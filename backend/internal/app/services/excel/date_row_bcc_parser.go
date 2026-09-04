package excel

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"github.com/xuri/excelize/v2"
)

// dateRowBCCCols scans a bounded region so a stray late numeric cell cannot
// drag the day region across the whole sheet. The real BUMHAN T09 file's day
// region ends at col 78 (26 full-date days + a 9-day day-number continuation),
// with summary/money columns from col 79 — the scan bound must cover the full
// month grid while the entry walk stays bounded by the last detected day
// column (see parseDateRowBCCSheet).
const (
	dateRowBCCMinCol  = 5  // day region starts at col E in every known template
	dateRowBCCScanMax = 96 // 2 sub-columns x 31 days from col 5 ends at col 66; headroom for wide layouts
	// The "Vị trí" (Position) column sits right of the day grid and its index
	// drifts with the month's day count (31 days → CX), so the header scan
	// must reach past every possible day-column layout.
	dateRowBCCPositionScanMax = 200
)

// dateRowEmployeeCols holds the employee-info column positions detected for a
// date-row BCC sheet (columns A-D area).
type dateRowEmployeeCols struct {
	cccdCol        int // "CCCD" or "Mã NV"/"Mã nhân viên" (the CCCD column)
	nameCol        int
	empCodeCol     int
	bankAccountCol int // "TK Ngân hàng"
	bankNameCol    int // "Ngân hàng" (without the TK prefix)
	phoneCol       int // "SĐT"/"Điện thoại" when the template carries one
	positionCol    int // "Vị trí" — located by header name, index drifts with month length
}

// ParseDateRowBCCFile parses partner BCC sheets headed by a row of full-date
// cells (BUMHAN "M1"-style): each day spans two sub-columns, the first
// carrying the (merged) date cell, with a weekday row and a shift-code row
// between the dates and the employee rows. Entries carry the exact calendar
// date in FullDate so mid-month periods survive intact.
func ParseDateRowBCCFile(f *excelize.File, sheetNames []string) (*BCCImportData, error) {
	result := &BCCImportData{}
	for _, sheet := range sheetNames {
		employees, err := parseDateRowBCCSheet(f, sheet)
		if err != nil {
			return nil, fmt.Errorf("sheet %q: %w", sheet, err)
		}
		result.Employees = append(result.Employees, employees...)
	}
	if len(result.Employees) == 0 {
		return nil, fmt.Errorf("không tìm thấy dữ liệu nhân viên trong sheet")
	}
	return result, nil
}

func parseDateRowBCCSheet(f *excelize.File, sheet string) ([]BCCEmployeeData, error) {
	dateRow, dateCols, err := findDateRowAndCols(f, sheet)
	if err != nil {
		return nil, err
	}
	minDayCol, maxDayCol := dateRowBCCScanMax, dateRowBCCMinCol
	for col := range dateCols {
		if col < minDayCol {
			minDayCol = col
		}
		if col > maxDayCol {
			maxDayCol = col
		}
	}
	shiftRow, colToShift := findShiftCodeRow(f, sheet, dateRow, minDayCol, maxDayCol)
	if shiftRow == 0 {
		return nil, fmt.Errorf("không tìm thấy dòng mã ca dưới dòng ngày")
	}
	hm := findDateRowEmployeeCols(f, sheet, dateRow)

	// The day region ends at its last detected day column; summary and money
	// columns beyond it (totals, "OT 150%", computed amounts) must never be
	// walked even when the shift-code row carries text there.
	lastDayCol := dateRowBCCMinCol
	for col := range dateCols {
		if col > lastDayCol {
			lastDayCol = col
		}
	}

	var employees []BCCEmployeeData
	for row := shiftRow + 1; row <= shiftRow+500; row++ {
		cccd := bccCell(f, sheet, hm.cccdCol, row)
		name := bccCell(f, sheet, hm.nameCol, row)

		// Stop when both CCCD and name are empty, or at a totals row. A single
		// blank row only ends the scan when no active employee row follows —
		// blank separators inside a roster are crossed, trailing blanks before
		// footnote rows ("* Chú ý:" …) are not.
		if cccd == "" && name == "" {
			if !rosterContinuesWithin(f, sheet, row, hm, lastDayCol) {
				break
			}
			continue
		}
		lowName, lowCccd := strings.ToLower(name), strings.ToLower(cccd)
		// "Tổng cộng" totals rows end the table; a bare "Cộng" label (this
		// template's summary row) is skipped so any later rows still parse.
		if strings.Contains(lowName, "tổng cộng") || strings.Contains(lowCccd, "tổng cộng") {
			break
		}
		if lowName == "cộng" || lowCccd == "cộng" {
			continue
		}

		emp := BCCEmployeeData{
			EmployeeCode: bccCell(f, sheet, hm.empCodeCol, row),
			CCCD:         cccd,
			FullName:     name,
			BankAccount:  bccCell(f, sheet, hm.bankAccountCol, row),
			BankName:     bccCell(f, sheet, hm.bankNameCol, row),
			Mobile:       bccCell(f, sheet, hm.phoneCol, row),
			Position:     bccCell(f, sheet, hm.positionCol, row),
		}

		// Walk the day region left to right. A column's date is its own anchor
		// or, for the second sub-column of a merged day pair, the anchor of the
		// column to its left — a day whose anchor is missing entirely is
		// skipped rather than inheriting the previous day's date.
		for col := dateRowBCCMinCol; col <= lastDayCol; col++ {
			currentDate := time.Time{}
			if d, ok := dateCols[col]; ok {
				currentDate = d
			} else if d, ok := dateCols[col-1]; ok {
				currentDate = d
			}
			label, hasLabel := colToShift[col]
			if !hasLabel || label == "" || currentDate.IsZero() {
				continue
			}
			hours := dateRowNumericCell(f, sheet, col, row)
			if hours == 0 {
				continue
			}
			d := currentDate
			emp.Entries = append(emp.Entries, BCCEntryData{
				DayNum:     d.Day(),
				ShiftLabel: label,
				Hours:      hours,
				FullDate:   &d,
			})
		}

		employees = append(employees, emp)
	}

	return employees, nil
}

// findDateRowAndCols scans rows 2-10 for the first row holding >= 3 full-date
// cells (Excel serials resolving to 2020-2040) in the day region AND a
// shift-code row within the 3 rows below it — the shift-row requirement keeps
// stray date cells above the real header from winning. Returns the 1-based row
// plus a map of 1-based column -> date.
func findDateRowAndCols(f *excelize.File, sheet string) (int, map[int]time.Time, error) {
	for row := 2; row <= 10; row++ {
		dates := make(map[int]time.Time)
		for col := dateRowBCCMinCol; col <= dateRowBCCScanMax; col++ {
			cn, err := excelize.CoordinatesToCellName(col, row)
			if err != nil {
				break
			}
			val, err := f.GetCellValue(sheet, cn, excelize.Options{RawCellValue: true})
			if err != nil || val == "" {
				continue
			}
			if t, ok := parseExcelDate(strings.TrimSpace(val)); ok {
				dates[col] = t
			}
		}
		if len(dates) < 3 {
			continue
		}
		minA, maxA := dateRowBCCScanMax, dateRowBCCMinCol
		for col := range dates {
			if col < minA {
				minA = col
			}
			if col > maxA {
				maxA = col
			}
		}
		if shiftRow, _ := findShiftCodeRow(f, sheet, row, minA, maxA); shiftRow == 0 {
			continue
		}
		extendDateRowWithDayNumbers(f, sheet, row, dates)
		return row, dates, nil
	}
	return 0, nil, fmt.Errorf("không tìm thấy dòng ngày trong sheet")
}

// extendDateRowWithDayNumbers continues the date map past the last full-date
// anchor using bare day-of-month numbers: partner files switch to plain day
// numbers ("16","16","17",…) once the merged-date region ends. A number
// qualifies only as the previous day repeated (second sub-column) or the next
// day; anything else — summary header text, totals — ends the continuation.
func extendDateRowWithDayNumbers(f *excelize.File, sheet string, row int, dates map[int]time.Time) {
	lastCol := 0
	var last time.Time
	for col, d := range dates {
		if col > lastCol {
			lastCol = col
			last = d
		}
	}
	next := last.AddDate(0, 0, 1)
	for col := lastCol + 1; col <= dateRowBCCScanMax; col++ {
		cn, err := excelize.CoordinatesToCellName(col, row)
		if err != nil {
			return
		}
		val, err := f.GetCellValue(sheet, cn, excelize.Options{RawCellValue: true})
		if err != nil || strings.TrimSpace(val) == "" {
			continue // merged second sub-column may be blank
		}
		n, err := strconv.Atoi(strings.TrimSpace(val))
		if err != nil {
			return // non-number text ends the day region
		}
		switch n {
		case next.Day():
		case next.AddDate(0, 0, 1).Day():
			next = next.AddDate(0, 0, 1)
		default:
			return
		}
		dates[col] = next
	}
}

// rosterContinuesWithin reports whether an active employee row — CCCD/name
// present plus at least one day-region hour — appears within 3 rows below a
// blank row. Blank and bare-"Cộng" rows are looked through; a "Tổng cộng"
// totals row or a named row without day hours (a footnote) ends the roster.
func rosterContinuesWithin(f *excelize.File, sheet string, blankRow int, hm dateRowEmployeeCols, lastDayCol int) bool {
	for row := blankRow + 1; row <= blankRow+3; row++ {
		cccd := bccCell(f, sheet, hm.cccdCol, row)
		name := bccCell(f, sheet, hm.nameCol, row)
		if cccd == "" && name == "" {
			continue
		}
		low := strings.ToLower(cccd + " " + name)
		if strings.Contains(low, "tổng cộng") {
			return false
		}
		if low == "cộng" {
			continue
		}
		for col := dateRowBCCMinCol; col <= lastDayCol; col++ {
			if dateRowNumericCell(f, sheet, col, row) != 0 {
				return true
			}
		}
		return false
	}
	return false
}

// weekdayToken reports whether a cell is a pure weekday label (T2..T7, CN),
// tolerating dotted forms like "T.6".
func weekdayToken(s string) bool {
	v := strings.ToUpper(strings.NewReplacer(".", "", " ", "").Replace(strings.TrimSpace(s)))
	return weekdayAbbrs[v]
}

// findShiftCodeRow returns the row below the date row holding the shift
// codes, mapped 1-based col -> code. Qualifying cells are short (<= 6 runes)
// letter tokens that are not pure weekday labels and sit inside the day
// region ([minDateCol, maxDateCol]) — the span bound keeps employee-header
// tokens ("SĐT", "Ghi chú") leaking from the header block above the day
// region from masquerading as a shift-code row, weekday-excluded so the
// weekday row never qualifies. Pure weekday-named codes like T7/CN are
// captured once the row is picked.
func findShiftCodeRow(f *excelize.File, sheet string, dateRow, minDateCol, maxDateCol int) (int, map[int]string) {
	for row := dateRow + 1; row <= dateRow+3; row++ {
		codes := make(map[int]string)
		nonWeekday := 0
		for col := dateRowBCCMinCol; col <= dateRowBCCScanMax; col++ {
			val := bccCell(f, sheet, col, row)
			if val == "" {
				continue
			}
			if !strings.ContainsAny(val, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzĐđ") {
				continue
			}
			if utf8.RuneCountInString(val) <= 6 && !weekdayToken(val) &&
				col >= minDateCol && col <= maxDateCol {
				nonWeekday++
			}
			codes[col] = val
		}
		if nonWeekday >= 2 {
			return row, codes
		}
	}
	return 0, nil
}

// findDateRowEmployeeCols detects the employee-info columns by scanning the
// rows above the date row. "Mã NV" variants are accepted as the CCCD column
// because partner templates label the CCCD column that way.
func findDateRowEmployeeCols(f *excelize.File, sheet string, dateRow int) dateRowEmployeeCols {
	hm := dateRowEmployeeCols{}
	from := max(dateRow-6, 1)
	for row := from; row < dateRow; row++ {
		for col := 1; col <= 10; col++ {
			v := normHeader(bccCell(f, sheet, col, row))
			if v == "" {
				continue
			}

			if (strings.Contains(v, "cccd") || strings.Contains(v, "mã nv") || strings.Contains(v, "mã nhân viên")) && hm.cccdCol == 0 {
				hm.cccdCol = col
			}
			if strings.Contains(v, "họ và tên") && hm.nameCol == 0 {
				hm.nameCol = col
			}
			if strings.Contains(v, "mã nv") || strings.Contains(v, "mã nhân viên") {
				hm.empCodeCol = col
			}
			if strings.Contains(v, "tk ngân hàng") && hm.bankAccountCol == 0 {
				hm.bankAccountCol = col
			}
			if strings.Contains(v, "ngân hàng") && !strings.Contains(v, "tk") && hm.bankNameCol == 0 {
				hm.bankNameCol = col
			}
			if (strings.Contains(v, "sđt") || strings.Contains(v, "điện thoại") || strings.Contains(v, "so dt")) && hm.phoneCol == 0 {
				hm.phoneCol = col
			}
		}
		// The "Vị trí" (Position) column's index drifts with the month's day
		// count (a 31-day month pushes it to CX), so locate it by header name
		// alone, anywhere right of the employee block; diacritics optional.
		for col := 1; hm.positionCol == 0 && col <= dateRowBCCPositionScanMax; col++ {
			if isPositionHeader(normHeader(bccCell(f, sheet, col, row))) {
				hm.positionCol = col
			}
		}
		// Complete only after the whole row is scanned so trailing columns
		// (bank info) are not cut off by an early exit at the name column.
		if hm.cccdCol != 0 && hm.nameCol != 0 && hm.empCodeCol != 0 && hm.bankAccountCol != 0 {
			return hm
		}
	}
	// Fallbacks mirroring the legacy parser: CCCD/code in col B, name in col C.
	if hm.cccdCol == 0 {
		hm.cccdCol = 2
	}
	if hm.empCodeCol == 0 {
		hm.empCodeCol = hm.cccdCol
	}
	if hm.nameCol == 0 {
		hm.nameCol = 3
	}
	return hm
}

// isPositionHeader reports whether a header cell names the Position column
// ("Vị trí"). The match is diacritic-insensitive: stripping tone marks turns
// "Vị trí" and "Vi tri" into the same needle, so templates written without
// dấu still resolve the column.
func isPositionHeader(v string) bool {
	return strings.Contains(stripDiacritics(v), "vi tri")
}

// stripDiacritics drops NFD combining marks so Vietnamese header matching
// tolerates templates written without tone marks.
func stripDiacritics(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(s) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// dateRowNumericCell reads a data cell as a raw number (format-proof) and
// returns 0 for blank or non-numeric cells.
func dateRowNumericCell(f *excelize.File, sheet string, col, row int) float64 {
	cn, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return 0
	}
	val, err := f.GetCellValue(sheet, cn, excelize.Options{RawCellValue: true})
	if err != nil || val == "" || val == "0" {
		return 0
	}
	hours, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
	if err != nil || hours == 0 {
		return 0
	}
	return hours
}

// isDateRowBCCSheet reports whether a sheet matches the date-row BCC
// fingerprint. It delegates to findDateRowAndCols so detection and parsing
// share one definition of the layout — a sheet accepted here always parses.
func isDateRowBCCSheet(f *excelize.File, sheet string) bool {
	_, _, err := findDateRowAndCols(f, sheet)
	return err == nil
}
