package excel

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"golang.org/x/text/unicode/norm"
)

// BCCImportData holds all parsed data from a BCC Excel attendance file.
// Month (YYYY-MM) is supplied by the caller, not parsed from the file.
type BCCImportData struct {
	ShiftRates map[string]int64 // shift label → VND rate (from row 10, first occurrence wins)
	Employees  []BCCEmployeeData
}

// BCCEmployeeData holds one employee's parsed attendance entries.
type BCCEmployeeData struct {
	EmployeeCode string
	CCCD         string
	FullName     string
	Department   string
	Entries      []BCCEntryData
}

// BCCEntryData represents a single non-zero hours entry.
type BCCEntryData struct {
	DayNum     int
	ShiftLabel string
	Hours      float64
}

// bccHeaderMap holds dynamically-detected column positions from header rows 7-8.
type bccHeaderMap struct {
	sttCol     int // column with "STT" header (used for blank-row filtering)
	cccdCol    int // column with "CCCD" header
	empCodeCol int // column with "Mã nhân viên" header
	nameCol    int // column with "Họ và tên" header
	deptCol    int // column with "Bộ phận" header
}

// buildBCCHeaderMap scans rows 7 and 8 to find column positions for employee data fields.
// Headers may be split across both rows (e.g. "CCCD" appears in row 8 while "Họ và tên" in row 7).
func buildBCCHeaderMap(f *excelize.File, sheet string) *bccHeaderMap {
	hm := &bccHeaderMap{}

	for row := 7; row <= 8; row++ {
		for col := 1; col <= 10; col++ {
			val := bccCell(f, sheet, col, row)
			v := normHeader(val)

			if v == "stt" && hm.sttCol == 0 {
				hm.sttCol = col
			}
			if v == "cccd" && hm.cccdCol == 0 {
				hm.cccdCol = col
			}
			if strings.Contains(v, "mã nhân viên") && hm.empCodeCol == 0 {
				hm.empCodeCol = col
			}
			if strings.Contains(v, "họ và tên") && hm.nameCol == 0 {
				hm.nameCol = col
			}
			if strings.Contains(v, "bộ phận") && hm.deptCol == 0 {
				hm.deptCol = col
			}
		}
	}

	// Fallback to original hardcoded positions if headers not found.
	// Original layout: col A(1)=STT, col B(2)=EmployeeCode, col C(3)=CCCD, col D(4)=FullName, col G(7)=Department.
	if hm.sttCol == 0 || hm.cccdCol == 0 || hm.nameCol == 0 {
		slog.Warn("BCC parser: header row not fully detected, falling back to hardcoded layout",
			"sttCol", hm.sttCol, "cccdCol", hm.cccdCol, "nameCol", hm.nameCol,
			"empCodeCol", hm.empCodeCol, "deptCol", hm.deptCol)
	}
	if hm.sttCol == 0 {
		hm.sttCol = 1
	}
	if hm.empCodeCol == 0 {
		hm.empCodeCol = 2
	}
	if hm.cccdCol == 0 {
		// Two scenarios when "CCCD" header is absent:
		//   1) Old format: gap between empCode and name (e.g. col 2=code, col 3=CCCD, col 4=name).
		//      In this case empCodeCol+1 < nameCol → use the gap column.
		//   2) Compact format (e.g. EPE): no gap (e.g. col 2=code/CCCD, col 3=name).
		//      In this case empCodeCol+1 >= nameCol → reuse the employee-code column.
		if hm.nameCol > 0 && hm.empCodeCol+1 < hm.nameCol {
			hm.cccdCol = hm.empCodeCol + 1
		} else {
			hm.cccdCol = hm.empCodeCol
		}
	}
	if hm.nameCol == 0 {
		hm.nameCol = 4
	}
	// deptCol intentionally left as 0 when "Bộ phận" header is absent.
	// bccCell handles col=0 by returning "" — the Department field will be empty.

	return hm
}

// normHeader normalizes a header cell for matching: trim, NFC unicode, lower-case.
// Vietnamese text in xlsx files from macOS sometimes arrives in NFD form, so NFC
// normalisation prevents false misses like "ho va ten" vs "họ và tên".
func normHeader(s string) string {
	return strings.ToLower(strings.TrimSpace(norm.NFC.String(s)))
}

// ParseBCCFile parses a BCC monthly attendance Excel file.
// The sheet named "BCC" is used; if absent the first sheet is used.
func ParseBCCFile(f *excelize.File) (*BCCImportData, error) {
	sheet := resolveBCCSheet(f)

	hm := buildBCCHeaderMap(f, sheet)

	startCol, colToDayNum, stopCol, err := buildDayColMap(f, sheet)
	if err != nil {
		return nil, fmt.Errorf("ParseBCCFile: %w", err)
	}

	shiftRow, rateRow := detectShiftRows(f, sheet, startCol, stopCol)
	colToShift := buildShiftColMap(f, sheet, startCol, stopCol, shiftRow)
	shiftRates := buildShiftRates(f, sheet, colToShift, startCol, stopCol, rateRow)

	startRow := max(shiftRow, rateRow) + 1

	employees := parseEmployees(f, sheet, hm, colToDayNum, colToShift, startCol, stopCol, startRow)

	return &BCCImportData{
		ShiftRates: shiftRates,
		Employees:  employees,
	}, nil
}

func resolveBCCSheet(f *excelize.File) string {
	for _, s := range f.GetSheetList() {
		if s == "BCC" || strings.TrimSpace(s) == "BCC" {
			return s
		}
	}
	sheets := f.GetSheetList()
	if len(sheets) > 0 {
		return sheets[0]
	}
	return ""
}

// buildDayColMap reads the detected day-number row from col index 4 (Column E) onward.
// Day numbers appear every 4 columns; the number is propagated to all sub-columns
// of that day group. Returns start column index, colIndex→dayNum map and the stop column index.
//
// Day-number cells store the day-of-month as an Excel date serial with format "dd".
// We read raw (unformatted) values so that serial 22 → "22" (the correct day),
// rather than letting excelize apply its epoch offset and produce "21".
//
// The day-number row itself is detected by detectDayRow: legacy monthly templates
// put day numbers in row 8 (weekdays T2..CN in row 7), while newer weekly-cycle
// templates swap the two (day numbers in row 7, weekdays in row 8).
func buildDayColMap(f *excelize.File, sheet string) (int, map[int]int, int, error) {
	dataRow := detectDayRow(f, sheet)
	colToDayNum := make(map[int]int)
	currentDay := 0
	stopCol := 200

	// Day region start: first column in dataRow holding a day-of-month value (1-31),
	// rejecting non-day numeric cells (year "2026", counts, rate amounts).
	startColIdx := firstDayColumn(f, sheet, dataRow)
	if startColIdx == -1 {
		startColIdx = 7 // fallback to original behavior (Column H)
	}

	for colIdx := startColIdx; colIdx <= 200; colIdx++ {
		cn, err := excelize.CoordinatesToCellName(colIdx+1, dataRow)
		if err != nil {
			break
		}
		// Use RawCellValue to get the underlying serial number as a string
		// instead of the date-formatted value (which is off by 1 due to excelize's epoch).
		val, err := f.GetCellValue(sheet, cn, excelize.Options{RawCellValue: true})
		if err != nil {
			continue
		}
		val = strings.TrimSpace(val)
		if strings.Contains(val, "Tổng") {
			stopCol = colIdx
			break
		}
		if val != "" {
			if d, err2 := strconv.Atoi(val); err2 == nil {
				currentDay = d
			} else if serial, err3 := strconv.ParseFloat(val, 64); err3 == nil && serial >= 1 {
				// Fractional serial: round to nearest integer (day number).
				currentDay = int(serial + 0.5)
			}
		}
		if currentDay > 0 {
			colToDayNum[colIdx] = currentDay
		}
	}
	if len(colToDayNum) == 0 {
		return 0, nil, 0, fmt.Errorf("buildDayColMap: no day-of-month columns found in rows 7-8")
	}
	return startColIdx, colToDayNum, stopCol, nil
}

// detectDayRow returns the row (7 or 8) that holds the day-of-month numbers.
//
// Legacy monthly templates place day numbers in row 8 with weekday abbreviations
// (T2..CN) in row 7. Newer weekly-cycle templates swap these rows: day numbers in
// row 7 and weekdays in row 8. We pick the row whose first day number (1-31)
// appears at the smaller column index — the day region always begins at column I
// in every known template, so the correct row wins and any stray late numeric
// cell in the other row (e.g. a partial day range) loses. Row 8 (legacy) wins ties.
func detectDayRow(f *excelize.File, sheet string) int {
	row8 := firstDayColumn(f, sheet, 8)
	row7 := firstDayColumn(f, sheet, 7)
	switch {
	case row7 == -1:
		return 8
	case row8 == -1:
		return 7
	case row7 < row8:
		return 7
	default:
		return 8
	}
}

// firstDayColumn scans columns E..GR (col index 4..200) in row and returns the
// col-index (where col-index+1 is the 1-based column number) of the first cell
// holding a day-of-month value (1-31), or -1 if none. Reading RawCellValue
// avoids excelize's date-epoch offset on "dd"-formatted serials.
func firstDayColumn(f *excelize.File, sheet string, row int) int {
	for colIdx := 4; colIdx <= 200; colIdx++ {
		cn, err := excelize.CoordinatesToCellName(colIdx+1, row)
		if err != nil {
			break
		}
		val, err := f.GetCellValue(sheet, cn, excelize.Options{RawCellValue: true})
		if err != nil {
			continue
		}
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}
		dayNum := 0
		if n, err := strconv.Atoi(val); err == nil {
			dayNum = n
		} else if serial, err := strconv.ParseFloat(val, 64); err == nil {
			dayNum = int(serial)
		}
		if dayNum >= 1 && dayNum <= 31 {
			return colIdx
		}
	}
	return -1
}

// weekdayAbbrs is the set of Vietnamese weekday abbreviations that appear in
// the row above shift labels. Used by detectShiftRows to distinguish weekday
// names from actual shift labels.
var weekdayAbbrs = map[string]bool{
	"T2": true, "T3": true, "T4": true, "T5": true,
	"T6": true, "T7": true, "CN": true,
}

// detectShiftRows scans rows 9-11 to find which row contains shift labels
// (e.g. "CB N", "OT N", "HC", "TCN") and which contains rate amounts.
// Returns (shiftLabelRow, rateRow).
// Some files have labels in row 10/rates in row 9; others have labels in row 11/rates in row 10.
// Shift labels are distinguished from weekday abbreviations (T2..CN) by exclusion.
func detectShiftRows(f *excelize.File, sheet string, startCol, stopCol int) (int, int) {
	for row := 9; row <= 11; row++ {
		shiftCount := 0
		for colIdx := startCol; colIdx < stopCol; colIdx++ {
			cn, err := excelize.CoordinatesToCellName(colIdx+1, row)
			if err != nil {
				continue
			}
			val, err := f.GetCellValue(sheet, cn)
			if err != nil || val == "" {
				continue
			}
			val = strings.TrimSpace(val)
			// Treat as a shift label if it contains letters AND is not a weekday abbreviation.
			// Handles both space-containing labels ("CB N", "OT Đ") and compact ones ("HC", "TCN", "NN").
			if strings.ContainsAny(val, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzĐđ") {
				if !weekdayAbbrs[strings.ToUpper(val)] {
					shiftCount++
				}
			}
		}
		if shiftCount >= 2 {
			// This row has shift labels. Rate row is the row above (if it has numbers).
			rateRow := row - 1
			if rateRow < 9 {
				rateRow = 10
			}
			return row, rateRow
		}
	}
	// Fallback to original hardcoded positions
	slog.Warn("BCC parser: shift-label row not detected, falling back to hardcoded rows 11/10",
		"startCol", startCol, "stopCol", stopCol)
	return 11, 10
}

// buildShiftColMap reads the detected shift label row up to stopCol.
func buildShiftColMap(f *excelize.File, sheet string, startCol, stopCol, shiftRow int) map[int]string {
	colToShift := make(map[int]string)
	for colIdx := startCol; colIdx < stopCol; colIdx++ {
		cn, err := excelize.CoordinatesToCellName(colIdx+1, shiftRow)
		if err != nil {
			continue
		}
		val, err := f.GetCellValue(sheet, cn)
		if err != nil || val == "" {
			continue
		}
		colToShift[colIdx] = strings.TrimSpace(val)
	}
	return colToShift
}

// buildShiftRates reads the detected rate row: first occurrence of each shift label wins.
func buildShiftRates(f *excelize.File, sheet string, colToShift map[int]string, startCol, stopCol, rateRow int) map[string]int64 {
	rates := make(map[string]int64)
	for colIdx := startCol; colIdx < stopCol; colIdx++ {
		label, ok := colToShift[colIdx]
		if !ok || label == "" {
			continue
		}
		if _, exists := rates[label]; exists {
			continue
		}
		cn, err := excelize.CoordinatesToCellName(colIdx+1, rateRow)
		if err != nil {
			continue
		}
		val, err := f.GetCellValue(sheet, cn)
		if err != nil || val == "" {
			continue
		}
		// Rates may be formatted with commas and spaces (e.g. " 36,000 ").
		clean := strings.ReplaceAll(strings.TrimSpace(val), ",", "")
		r, err := strconv.ParseInt(clean, 10, 64)
		if err != nil || r == 0 {
			continue
		}
		rates[label] = r
	}
	return rates
}

// parseEmployees reads rows from startRow until CCCD and name are empty.
// Column positions are determined dynamically from header rows via bccHeaderMap.
func parseEmployees(f *excelize.File, sheet string, hm *bccHeaderMap, colToDayNum map[int]int, colToShift map[int]string, startCol, stopCol, startRow int) []BCCEmployeeData {
	var employees []BCCEmployeeData
	for row := startRow; ; row++ {
		cccd := bccCell(f, sheet, hm.cccdCol, row)
		name := bccCell(f, sheet, hm.nameCol, row)

		// Stop when both CCCD and Name are empty (e.g. end of active rows, or a summary row)
		if cccd == "" && name == "" {
			break
		}
		if strings.Contains(strings.ToLower(name), "tổng cộng") || strings.Contains(strings.ToLower(cccd), "tổng cộng") {
			break
		}

		emp := BCCEmployeeData{
			EmployeeCode: bccCell(f, sheet, hm.empCodeCol, row),
			CCCD:         cccd,
			FullName:     name,
			Department:   bccCell(f, sheet, hm.deptCol, row),
		}

		for colIdx := startCol; colIdx < stopCol; colIdx++ {
			dayNum, hasDayNum := colToDayNum[colIdx]
			if !hasDayNum {
				continue
			}
			label, hasLabel := colToShift[colIdx]
			if !hasLabel || label == "" {
				continue
			}
			cn, err := excelize.CoordinatesToCellName(colIdx+1, row)
			if err != nil {
				continue
			}
			// RawCellValue bypasses the cell's number format. A cell storing 7.5
			// with format "0" would otherwise be returned as "8" (rounded) and
			// silently lost — see the Nguyễn Trọng Thắng 7.5h→8h bug.
			val, err := f.GetCellValue(sheet, cn, excelize.Options{RawCellValue: true})
			if err != nil || val == "" || val == "0" {
				continue
			}
			hours, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
			if err != nil || hours == 0 {
				continue
			}
			emp.Entries = append(emp.Entries, BCCEntryData{
				DayNum:     dayNum,
				ShiftLabel: label,
				Hours:      hours,
			})
		}

		// Skip rows that have blank STT AND have zero entries (filters out draft rows while keeping active employees with blank STT)
		stt := bccCell(f, sheet, hm.sttCol, row)
		if stt == "" && len(emp.Entries) == 0 {
			continue
		}

		employees = append(employees, emp)
	}
	return employees
}

func bccCell(f *excelize.File, sheet string, col, row int) string {
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
