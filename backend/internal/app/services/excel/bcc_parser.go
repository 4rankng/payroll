package excel

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// BCCImportData holds all parsed data from a BCC Excel attendance file.
// ForMonth is NOT parsed from the file — it comes from the frontend upload request.
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

	colToDayNum, stopCol, err := buildDayColMap(f, sheet)
	if err != nil {
		return nil, fmt.Errorf("ParseBCCFile: %w", err)
	}

	colToShift := buildShiftColMap(f, sheet, stopCol)
	shiftRates := buildShiftRates(f, sheet, colToShift, stopCol)
	employees := parseEmployees(f, sheet, colToDayNum, colToShift, stopCol)

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

// buildDayColMap reads row 8 from col index 7 onward.
// Day numbers appear every 4 columns; the number is propagated to all sub-columns
// of that day group. Returns colIndex→dayNum map and the stop column index.
//
// Row 8 cells store the day-of-month as an Excel date serial with format "dd".
// We read raw (unformatted) values so that serial 22 → "22" (the correct day),
// rather than letting excelize apply its epoch offset and produce "21".
func buildDayColMap(f *excelize.File, sheet string) (map[int]int, int, error) {
	const dataRow = 8
	colToDayNum := make(map[int]int)
	currentDay := 0
	stopCol := 200

	for colIdx := 7; colIdx <= 200; colIdx++ {
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
		return nil, 0, fmt.Errorf("buildDayColMap: no day columns found in row %d", dataRow)
	}
	return colToDayNum, stopCol, nil
}

// buildShiftColMap reads row 11 up to stopCol.
func buildShiftColMap(f *excelize.File, sheet string, stopCol int) map[int]string {
	colToShift := make(map[int]string)
	for colIdx := 7; colIdx < stopCol; colIdx++ {
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
func buildShiftRates(f *excelize.File, sheet string, colToShift map[int]string, stopCol int) map[string]int64 {
	rates := make(map[string]int64)
	for colIdx := 7; colIdx < stopCol; colIdx++ {
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

// parseEmployees reads rows 12+ until col A is non-numeric.
func parseEmployees(f *excelize.File, sheet string, colToDayNum map[int]int, colToShift map[int]string, stopCol int) []BCCEmployeeData {
	var employees []BCCEmployeeData
	for row := 12; ; row++ {
		sttCN, _ := excelize.CoordinatesToCellName(1, row)
		sttVal, _ := f.GetCellValue(sheet, sttCN)
		sttVal = strings.TrimSpace(sttVal)
		if sttVal == "" {
			break
		}
		if _, err := strconv.ParseFloat(sttVal, 64); err != nil {
			break
		}

		emp := BCCEmployeeData{
			EmployeeCode: bccCell(f, sheet, 2, row), // col B
			CCCD:         bccCell(f, sheet, 3, row), // col C
			FullName:     bccCell(f, sheet, 4, row), // col D
			Department:   bccCell(f, sheet, 7, row), // col G
		}

		for colIdx := 7; colIdx < stopCol; colIdx++ {
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
