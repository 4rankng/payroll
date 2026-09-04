package excel

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

// dateRowBCCCols scans a bounded region so a stray late numeric cell cannot
// drag the day region across the whole sheet.
const (
	dateRowBCCMinCol = 5  // day region starts at col E in every known template
	dateRowBCCMaxCol = 60 // 2 sub-columns x 31 days fits well below this
)

// dateRowEmployeeCols holds the employee-info column positions detected for a
// date-row BCC sheet (columns A-D area).
type dateRowEmployeeCols struct {
	sttCol         int
	cccdCol        int // "CCCD" or "Mã NV"/"Mã nhân viên" (the CCCD column)
	nameCol        int
	empCodeCol     int
	bankAccountCol int // "TK Ngân hàng"
	bankNameCol    int // "Ngân hàng" (without the TK prefix)
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
	shiftRow, colToShift := findShiftCodeRow(f, sheet, dateRow)
	if shiftRow == 0 {
		return nil, fmt.Errorf("không tìm thấy dòng mã ca dưới dòng ngày")
	}
	hm := findDateRowEmployeeCols(f, sheet, dateRow)

	var employees []BCCEmployeeData
	for row := shiftRow + 1; row <= shiftRow+500; row++ {
		cccd := bccCell(f, sheet, hm.cccdCol, row)
		name := bccCell(f, sheet, hm.nameCol, row)

		// Stop when both CCCD and name are empty, or at a totals row.
		if cccd == "" && name == "" {
			break
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
		}

		// Walk the day region left to right; the date forward-fills across a
		// day's two sub-columns (only the anchor column holds the date cell).
		currentDate := time.Time{}
		for col := dateRowBCCMinCol; col <= dateRowBCCMaxCol; col++ {
			if d, ok := dateCols[col]; ok {
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
// cells (Excel serials resolving to 2020-2040) in the day region, and returns
// the 1-based row plus a map of 1-based column -> date.
func findDateRowAndCols(f *excelize.File, sheet string) (int, map[int]time.Time, error) {
	for row := 2; row <= 10; row++ {
		dates := make(map[int]time.Time)
		for col := dateRowBCCMinCol; col <= dateRowBCCMaxCol; col++ {
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
		if len(dates) >= 3 {
			return row, dates, nil
		}
	}
	return 0, nil, fmt.Errorf("không tìm thấy dòng ngày trong sheet")
}

// weekdayToken reports whether a cell is a pure weekday label (T2..T7, CN),
// tolerating dotted forms like "T.6".
func weekdayToken(s string) bool {
	v := strings.ToUpper(strings.NewReplacer(".", "", " ", "").Replace(strings.TrimSpace(s)))
	return weekdayAbbrs[v]
}

// findShiftCodeRow returns the row below the date row holding the shift
// codes, mapped 1-based col -> code. Qualifying cells are short (<= 6 runes)
// letter tokens that are not pure weekday labels — short so partner header
// words ("Ngân hàng", "Mức lương /9h") leaking from the merged header block
// never qualify, weekday-excluded so the weekday row never qualifies. Pure
// weekday-named codes like T7/CN are captured once the row is picked.
func findShiftCodeRow(f *excelize.File, sheet string, dateRow int) (int, map[int]string) {
	for row := dateRow + 1; row <= dateRow+3; row++ {
		codes := make(map[int]string)
		nonWeekday := 0
		for col := dateRowBCCMinCol; col <= dateRowBCCMaxCol; col++ {
			val := bccCell(f, sheet, col, row)
			if val == "" {
				continue
			}
			if !strings.ContainsAny(val, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzĐđ") {
				continue
			}
			if utf8.RuneCountInString(val) <= 6 && !weekdayToken(val) {
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
// few rows directly above the date row. "Mã NV" variants are accepted as the
// CCCD column because partner templates label the CCCD column that way.
func findDateRowEmployeeCols(f *excelize.File, sheet string, dateRow int) dateRowEmployeeCols {
	hm := dateRowEmployeeCols{}
	from := max(dateRow-4, 1)
	for row := from; row < dateRow; row++ {
		for col := 1; col <= 10; col++ {
			v := normHeader(bccCell(f, sheet, col, row))
			if v == "" {
				continue
			}
			if v == "stt" && hm.sttCol == 0 {
				hm.sttCol = col
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
// fingerprint: a row (scanned in rows 2-10) holding >= 3 full-date cells in
// the day region, with a shift-code row (>= 2 non-weekday letter cells)
// within the 3 rows below it.
func isDateRowBCCSheet(f *excelize.File, sheet string) bool {
	for row := 2; row <= 10; row++ {
		dateCount := 0
		for col := dateRowBCCMinCol; col <= dateRowBCCMaxCol; col++ {
			cn, err := excelize.CoordinatesToCellName(col, row)
			if err != nil {
				break
			}
			val, err := f.GetCellValue(sheet, cn, excelize.Options{RawCellValue: true})
			if err != nil || val == "" {
				continue
			}
			if _, ok := parseExcelDate(strings.TrimSpace(val)); ok {
				dateCount++
			}
		}
		if dateCount < 3 {
			continue
		}
		if shiftRow, codes := findShiftCodeRow(f, sheet, row); shiftRow != 0 && len(codes) >= 2 {
			return true
		}
	}
	return false
}
