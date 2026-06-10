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
	sttCol    int // column with "STT" header (used for blank-row filtering)
	cccdCol   int // column with "CCCD" header
	empCodeCol int // column with "Mã nhân viên" header
	nameCol   int // column with "Họ và tên" header
	deptCol   int // column with "Bộ phận" header
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
		hm.cccdCol = 3
	}
	if hm.nameCol == 0 {
		hm.nameCol = 4
	}
	if hm.deptCol == 0 {
		hm.deptCol = 7
	}

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
	employees := parseEmployees(f, sheet, hm, colToDayNum, colToShift, startCol, stopCol)

	return &BCCImportData{
		ShiftRates: shiftRates,
		Employees:  employees,
	}, nil
}

func resolveBCCSheet(f *excelize.File) string {
	for _, s := range f.GetSheetList() {
		if s == "BCC" {
			return s
		}
	}
	sheets := f.GetSheetList()
	if len(sheets) > 0 {
		return sheets[0]
	}
	return ""
}

// buildDayColMap reads row 8 from col index 4 (Column E) onward.
// Day numbers appear every 4 columns; the number is propagated to all sub-columns
// of that day group. Returns start column index, colIndex→dayNum map and the stop column index.
//
// Row 8 cells store the day-of-month as an Excel date serial with format "dd".
// We read raw (unformatted) values so that serial 22 → "22" (the correct day),
// rather than letting excelize apply its epoch offset and produce "21".
func buildDayColMap(f *excelize.File, sheet string) (int, map[int]int, int, error) {
	const dataRow = 8
	colToDayNum := make(map[int]int)
	currentDay := 0
	stopCol := 200

	// Dynamically detect where the timesheet dates start by scanning from Column E (index 4) onwards.
	// Only accept values in the valid day-of-month range (1-31) to avoid false positives
	// from non-day numeric cells (e.g. year "2026", counts, or rate amounts).
	startColIdx := -1
	for colIdx := 4; colIdx <= 200; colIdx++ {
		cn, err := excelize.CoordinatesToCellName(colIdx+1, dataRow)
		if err != nil {
			break
		}
		val, err := f.GetCellValue(sheet, cn, excelize.Options{RawCellValue: true})
		if err != nil {
			continue
		}
		val = strings.TrimSpace(val)
		if val != "" {
			dayNum := 0
			if n, err2 := strconv.Atoi(val); err2 == nil {
				dayNum = n
			} else if serial, err3 := strconv.ParseFloat(val, 64); err3 == nil {
				dayNum = int(serial)
			}
			if dayNum >= 1 && dayNum <= 31 {
				startColIdx = colIdx
				break
			}
		}
	}

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
		return 0, nil, 0, fmt.Errorf("buildDayColMap: no day columns found in row %d", dataRow)
	}
	return startColIdx, colToDayNum, stopCol, nil
}

// detectShiftRows scans rows 9-11 to find which row contains shift labels (e.g. "CB N", "OT N")
// and which contains rate amounts. Returns (shiftLabelRow, rateRow).
// Some files have labels in row 10/rates in row 9; others have labels in row 11/rates in row 10.
// Shift labels always contain a space (e.g. "CB N") — weekday abbreviations don't (e.g. "T2", "CN").
func detectShiftRows(f *excelize.File, sheet string, startCol, stopCol int) (int, int) {
	for row := 9; row <= 11; row++ {
		textCount := 0
		numCount := 0
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
			// Shift labels contain a space (e.g. "CB N", "OT Đ", "CN N", "Tăng ca").
			// Weekday abbreviations are single tokens ("T2", "T3", "CN") — no space.
			if strings.ContainsAny(val, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzĐđ") && strings.Contains(val, " ") {
				textCount++
			} else {
				// Numeric value (rate amount)
				clean := strings.ReplaceAll(val, ",", "")
				if n, err := strconv.ParseInt(clean, 10, 64); err == nil && n > 0 {
					numCount++
				}
			}
		}
		if textCount >= 2 {
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

// parseEmployees reads rows 12+ until CCCD and name are empty.
// Column positions are determined dynamically from header rows via bccHeaderMap.
func parseEmployees(f *excelize.File, sheet string, hm *bccHeaderMap, colToDayNum map[int]int, colToShift map[int]string, startCol, stopCol int) []BCCEmployeeData {
	var employees []BCCEmployeeData
	for row := 12; ; row++ {
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
			val, err := f.GetCellValue(sheet, cn)
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
