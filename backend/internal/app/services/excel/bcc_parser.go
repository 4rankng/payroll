package excel

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
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

// ParseBCCFile parses a BCC monthly attendance Excel file.
// The sheet named "BCC" is used; if absent the first sheet is used.
func ParseBCCFile(f *excelize.File) (*BCCImportData, error) {
	sheet := resolveBCCSheet(f)

	startCol, colToDayNum, stopCol, err := buildDayColMap(f, sheet)
	if err != nil {
		return nil, fmt.Errorf("ParseBCCFile: %w", err)
	}

	colToShift := buildShiftColMap(f, sheet, startCol, stopCol)
	shiftRates := buildShiftRates(f, sheet, colToShift, startCol, stopCol)
	employees := parseEmployees(f, sheet, colToDayNum, colToShift, startCol, stopCol)

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

	// Dynamically detect where the timesheet dates start by scanning from Column E (index 4) onwards
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
			isDay := false
			if _, err2 := strconv.Atoi(val); err2 == nil {
				isDay = true
			} else if serial, err3 := strconv.ParseFloat(val, 64); err3 == nil && serial >= 1 {
				isDay = true
			}
			if isDay {
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

// buildShiftColMap reads row 11 up to stopCol.
func buildShiftColMap(f *excelize.File, sheet string, startCol, stopCol int) map[int]string {
	colToShift := make(map[int]string)
	for colIdx := startCol; colIdx < stopCol; colIdx++ {
		cn, err := excelize.CoordinatesToCellName(colIdx+1, 11)
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

// buildShiftRates reads row 10: first occurrence of each shift label wins.
func buildShiftRates(f *excelize.File, sheet string, colToShift map[int]string, startCol, stopCol int) map[string]int64 {
	rates := make(map[string]int64)
	for colIdx := startCol; colIdx < stopCol; colIdx++ {
		label, ok := colToShift[colIdx]
		if !ok || label == "" {
			continue
		}
		if _, exists := rates[label]; exists {
			continue
		}
		cn, err := excelize.CoordinatesToCellName(colIdx+1, 10)
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
func parseEmployees(f *excelize.File, sheet string, colToDayNum map[int]int, colToShift map[int]string, startCol, stopCol int) []BCCEmployeeData {
	var employees []BCCEmployeeData
	for row := 12; ; row++ {
		stt := bccCell(f, sheet, 1, row)  // col A
		cccd := bccCell(f, sheet, 3, row) // col C
		name := bccCell(f, sheet, 4, row) // col D

		// Stop when both CCCD and Name are empty (e.g. end of active rows, or a summary row)
		if cccd == "" && name == "" {
			break
		}
		if strings.Contains(strings.ToLower(name), "tổng cộng") || strings.Contains(strings.ToLower(cccd), "tổng cộng") {
			break
		}

		emp := BCCEmployeeData{
			EmployeeCode: bccCell(f, sheet, 2, row), // col B
			CCCD:         cccd,
			FullName:     name,
			Department:   bccCell(f, sheet, 7, row), // col G
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
