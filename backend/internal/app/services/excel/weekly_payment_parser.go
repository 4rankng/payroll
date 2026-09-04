package excel

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const (
	wpMaxColScan     = 200 // Maximum columns to scan for date/shift columns
	wpMaxRowScan     = 500 // Maximum rows to scan for employee data
	wpMaxConsecBlank = 5   // Consecutive blank rows before stopping
)

// WeeklyPaymentImportData holds parsed data from all weekly payment sheets.
type WeeklyPaymentImportData struct {
	Sheets []WeeklyPaymentSheetData
}

// WeeklyPaymentSheetData holds parsed data from a single weekly payment sheet.
type WeeklyPaymentSheetData struct {
	Position  string                // Excel sheet name, e.g. "Lương 520"
	ShiftRows WeeklyPaymentShiftRow // row 10 cells (col index → shift code)
	Employees []WeeklyPaymentEmployeeData
}

// WeeklyPaymentShiftRow maps column indices (1-based) to shift codes.
type WeeklyPaymentShiftRow struct {
	ByCol map[int]string // 1-based col → "HC" | "TCN" | "NN" | "TCNN" | ...
}

// WeeklyPaymentEmployeeData holds parsed data for one employee row.
type WeeklyPaymentEmployeeData struct {
	EmployeeCode string // col B = CCCD
	FullName     string // col C
	Department   string // col D
	SalaryTier   int    // col E = "Lương 8h" (informational; not used for rate lookup)
	Entries      []WeeklyPaymentEntryData
}

// WeeklyPaymentEntryData holds one (day, shift, hours) entry for an employee.
type WeeklyPaymentEntryData struct {
	Day      int    // 1..31, derived from column position
	ShiftKey string // e.g. "HC" or "TCN", taken directly from the row 10 column header
	Hours    float64
}

// weeklyPaymentHeaderMap stores detected column positions for a weekly payment sheet.
type weeklyPaymentHeaderMap struct {
	empCodeCol    int            // 1-based column for "Mã nhân viên"
	fullNameCol   int            // 1-based column for "Họ và tên"
	departmentCol int            // 1-based column for "Bộ phận"
	salaryTierCol int            // 1-based column for "Lương 8h"
	firstShiftCol int            // 1-based first column with shift data
	shiftCols     map[int]string // 1-based col index → shift type
	stopCol       int            // 1-based "Tổng hợp" column (exclusive stop)
	dateColumns   map[int]int    // 1-based col index → day of month (1..31)
}

// ParseWeeklyPaymentFile parses all weekly payment sheets in a weekly payment Excel file.
func ParseWeeklyPaymentFile(f *excelize.File, sheetNames []string, forMonth string) (*WeeklyPaymentImportData, error) {
	result := &WeeklyPaymentImportData{}

	// Parse forMonth to get year and month
	year, month, err := parseForMonth(forMonth)
	if err != nil {
		return nil, fmt.Errorf("lỗi phân tích tháng hiệu lực: %w", err)
	}

	for _, sheetName := range sheetNames {
		sheetData, err := parseWeeklyPaymentSheet(f, sheetName, year, month)
		if err != nil {
			return nil, fmt.Errorf("sheet %q: %w", sheetName, err)
		}
		sheetData.Position = sheetName
		result.Sheets = append(result.Sheets, *sheetData)
	}

	if len(result.Sheets) == 0 {
		return nil, fmt.Errorf("không tìm thấy sheet weekly payment hợp lệ")
	}

	return result, nil
}

// parseWeeklyPaymentSheet parses a single weekly payment sheet.
func parseWeeklyPaymentSheet(f *excelize.File, sheetName string, year int, month time.Month) (*WeeklyPaymentSheetData, error) {
	// Step 1: Build header map from rows 8, 9, 10
	hm, err := buildWeeklyPaymentHeaderMap(f, sheetName, year, month)
	if err != nil {
		return nil, err
	}

	// Step 2: Parse employees from row 11+
	employees := parseWeeklyPaymentEmployees(f, sheetName, hm)

	return &WeeklyPaymentSheetData{
		ShiftRows: WeeklyPaymentShiftRow{ByCol: hm.shiftCols},
		Employees: employees,
	}, nil
}

// buildWeeklyPaymentHeaderMap reads rows 8, 9, 10 to detect column layout.
// Row 8: Headers (STT, Mã nhân viên, Họ tên, Bộ phận, Lương 8h) + day columns
// Row 9: Day types (T4, T5, T6, T7, CN, T2, T3, ...)
// Row 10: Shift codes (HC, TCN, NN, TCNN, ...)
func buildWeeklyPaymentHeaderMap(f *excelize.File, sheet string, year int, month time.Month) (*weeklyPaymentHeaderMap, error) {
	hm := &weeklyPaymentHeaderMap{
		shiftCols:   make(map[int]string),
		dateColumns: make(map[int]int),
	}

	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) < 10 {
		return nil, fmt.Errorf("sheet không có đủ dữ liệu (cần ít nhất 10 hàng)")
	}

	// Detect header columns (row 8)
	// A=STT, B=Mã nhân viên, C=Họ Tên, D=Bộ phận, E=Lương 8h
	hm.empCodeCol = 2    // B
	hm.fullNameCol = 3   // C
	hm.departmentCol = 4 // D
	hm.salaryTierCol = 5 // E

	// Find first shift column (starts at column F in the template)
	// Row 8 has date values, row 9 has weekday labels, row 10 has shift codes
	// We scan row 10 to find the first shift code
	firstShiftCol := 0
	for colIdx := 5; colIdx < wpMaxColScan; colIdx++ { // Start from column F (index 5)
		cn, err := excelize.CoordinatesToCellName(colIdx+1, 10) // Row 10
		if err != nil {
			break
		}
		val, err := f.GetCellValue(sheet, cn)
		if err != nil {
			continue
		}
		val = strings.ToUpper(strings.TrimSpace(val))
		if isShiftCode(val) {
			firstShiftCol = colIdx + 1 // 1-based
			break
		}
	}

	if firstShiftCol == 0 {
		return nil, fmt.Errorf("không tìm thấy cột mã ca (HC, TCN, NN, TCNN) ở hàng 10")
	}
	hm.firstShiftCol = firstShiftCol

	// The selected month/year is authoritative. Row 8 supplies only the day of
	// month; its merged top-left values apply to following shift columns until
	// the next explicit day header. Row 9 is display-only because legacy files
	// can contain stale weekday labels.
	lastColumnDay := 0
	for colIdx := firstShiftCol - 1; colIdx < wpMaxColScan; colIdx++ {
		colName, err := excelize.ColumnNumberToName(colIdx + 1)
		if err != nil {
			break
		}

		// Summary blocks can repeat real shift codes in row 10 (HC, TCN, NN,
		// TCNN). Check their section label before visibility because the merged
		// anchor column may be hidden while later summary columns remain visible.
		isSummaryColumn := false
		for _, headerRow := range []int{8, 9} {
			headerCell, cellErr := excelize.CoordinatesToCellName(colIdx+1, headerRow)
			if cellErr != nil {
				continue
			}
			headerValue, valueErr := f.GetCellValue(sheet, headerCell)
			if valueErr == nil && isStopColumn(headerValue) {
				isSummaryColumn = true
				break
			}
		}
		if isSummaryColumn {
			hm.stopCol = colIdx + 1
			break
		}

		cn, err := excelize.CoordinatesToCellName(colIdx+1, 10)
		if err != nil {
			break
		}

		// Check for stop columns (Tổng hợp, etc.)
		val, err := f.GetCellValue(sheet, cn)
		if err != nil {
			break
		}
		val = strings.ToUpper(strings.TrimSpace(val))
		if isStopColumn(val) {
			hm.stopCol = colIdx + 1 // 1-based
			break
		}

		// Hidden attendance columns do not contribute entries, but summary
		// boundaries above must still terminate the scan.
		visible, err := f.GetColVisible(sheet, colName)
		if err == nil && !visible {
			continue
		}

		// Read shift code from row 10
		if isShiftCode(val) {
			if day, hasDay, err := weeklyPaymentHeaderDay(f, sheet, colIdx+1); err != nil {
				return nil, fmt.Errorf("không thể đọc ngày ở cột %s: %w", colName, err)
			} else if hasDay {
				lastColumnDay = day
			}
			if lastColumnDay == 0 {
				return nil, fmt.Errorf("không thể xác định ngày cho cột %s: thiếu ngày ở hàng 8", colName)
			}

			if err := validateWeeklyPaymentHeaderDay(year, month, lastColumnDay); err != nil {
				return nil, fmt.Errorf("cột %s: %w", colName, err)
			}
			hm.shiftCols[colIdx+1] = val
			hm.dateColumns[colIdx+1] = lastColumnDay
		}
	}

	if len(hm.shiftCols) == 0 {
		return nil, fmt.Errorf("không tìm thấy cột ca làm việc")
	}

	// Default stopCol if "Tổng hợp" not found
	if hm.stopCol == 0 {
		hm.stopCol = wpMaxColScan
	}

	return hm, nil
}

// weeklyPaymentHeaderDay returns the explicitly displayed day in row 8. A
// blank result is expected for merged header cells and is carried forward by
// the caller. We accept both day numbers and normal date cell formats.
func weeklyPaymentHeaderDay(f *excelize.File, sheet string, col int) (int, bool, error) {
	cell, err := excelize.CoordinatesToCellName(col, 8)
	if err != nil {
		return 0, false, err
	}
	value, err := f.GetCellValue(sheet, cell)
	if err != nil {
		return 0, false, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false, nil
	}

	if day, err := strconv.Atoi(value); err == nil {
		return day, true, nil
	}
	for _, layout := range []string{"02/01/2006", "2/1/2006", "2006-01-02"} {
		if date, err := time.Parse(layout, value); err == nil {
			return date.Day(), true, nil
		}
	}
	return 0, false, fmt.Errorf("giá trị %q không phải ngày", value)
}

func validateWeeklyPaymentHeaderDay(year int, month time.Month, day int) error {
	date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if day < 1 || date.Month() != month {
		return fmt.Errorf("ngày %d nằm ngoài kỳ nhập %02d/%d", day, month, year)
	}
	return nil
}

// parseWeeklyPaymentEmployees reads employee rows from row 11+.
func parseWeeklyPaymentEmployees(f *excelize.File, sheet string, hm *weeklyPaymentHeaderMap) []WeeklyPaymentEmployeeData {
	var employees []WeeklyPaymentEmployeeData
	consecutiveBlank := 0

	for row := 11; row < wpMaxRowScan; row++ {
		// Skip hidden rows
		visible, err := f.GetRowVisible(sheet, row)
		if err == nil && !visible {
			continue
		}

		// Read employee code and name
		empCode := bccCell(f, sheet, hm.empCodeCol, row)
		fullName := bccCell(f, sheet, hm.fullNameCol, row)

		// Stop on 5 consecutive blank rows
		if empCode == "" && fullName == "" {
			consecutiveBlank++
			if consecutiveBlank >= wpMaxConsecBlank {
				break
			}
			continue
		}
		consecutiveBlank = 0

		// Skip summary rows
		if strings.Contains(strings.ToLower(fullName), "tổng cộng") ||
			strings.Contains(strings.ToLower(empCode), "tổng cộng") {
			break
		}

		// Read department and salary tier
		department := bccCell(f, sheet, hm.departmentCol, row)
		salaryTierStr := bccCell(f, sheet, hm.salaryTierCol, row)
		salaryTier := 0
		if salaryTierStr != "" {
			salaryTier, _ = strconv.Atoi(strings.TrimSpace(salaryTierStr))
		}

		emp := WeeklyPaymentEmployeeData{
			EmployeeCode: empCode,
			FullName:     fullName,
			Department:   department,
			SalaryTier:   salaryTier,
		}

		// Parse hour entries from shift columns
		for col1Based := hm.firstShiftCol; col1Based < hm.stopCol; col1Based++ {
			shiftCode, hasShift := hm.shiftCols[col1Based]
			if !hasShift {
				continue
			}

			dayNum, hasDay := hm.dateColumns[col1Based]
			if !hasDay || dayNum < 1 || dayNum > 31 {
				continue
			}

			cn, err := excelize.CoordinatesToCellName(col1Based, row)
			if err != nil {
				continue
			}

			val, err := f.GetCellValue(sheet, cn, excelize.Options{RawCellValue: true})
			if err != nil || val == "" {
				continue
			}

			hours, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
			if err != nil {
				continue
			}

			// hours == 0 is an explicit zero: keep the entry so the import
			// deletes the matching chờ duyệt timesheet; blank cells above
			// stay skipped.
			emp.Entries = append(emp.Entries, WeeklyPaymentEntryData{
				Day:      dayNum,
				ShiftKey: shiftCode,
				Hours:    hours,
			})
		}

		// Skip rows with empty code AND no positive hours (zero-hour entries do
		// not count — an all-zero row is a bulk deletion request, not a draft).
		if empCode == "" && !weeklyPaymentRowHasPositiveHours(emp.Entries) {
			continue
		}

		employees = append(employees, emp)
	}

	return employees
}

// isShiftCode checks if a value is a known shift code.
func isShiftCode(val string) bool {
	val = strings.ToUpper(strings.TrimSpace(val))
	shiftCodes := []string{"HC", "TCN", "NN", "TCNN"}
	for _, code := range shiftCodes {
		if strings.Contains(val, code) {
			return true
		}
	}
	return false
}

// isStopColumn checks if a column value indicates a stop column (summary, etc.)
func isStopColumn(val string) bool {
	stopIndicators := []string{"TỔNG HỢP", "TONG HOP", "TỔNG CỘNG", "TONG CONG", "SUẤT ĂN"}
	valUpper := strings.ToUpper(strings.TrimSpace(val))
	for _, indicator := range stopIndicators {
		if strings.Contains(valUpper, indicator) {
			return true
		}
	}
	return false
}

// parseForMonth parses a month string in "YYYY-MM" or "MM/YYYY" format.
func parseForMonth(forMonth string) (year int, month time.Month, err error) {
	forMonth = strings.TrimSpace(forMonth)

	// Try YYYY-MM format first
	parts := strings.Split(forMonth, "-")
	if len(parts) == 2 {
		year, err1 := strconv.Atoi(parts[0])
		monthVal, err2 := strconv.Atoi(parts[1])
		if err1 == nil && err2 == nil && monthVal >= 1 && monthVal <= 12 {
			return year, time.Month(monthVal), nil
		}
	}

	// Try MM/YYYY format
	parts = strings.Split(forMonth, "/")
	if len(parts) == 2 {
		monthVal, err1 := strconv.Atoi(parts[0])
		year, err2 := strconv.Atoi(parts[1])
		if err1 == nil && err2 == nil && monthVal >= 1 && monthVal <= 12 {
			return year, time.Month(monthVal), nil
		}
	}

	// Try parsing as full date
	t, err := time.Parse("2006-01", forMonth)
	if err == nil {
		return t.Year(), t.Month(), nil
	}

	return 0, 0, fmt.Errorf("định dạng tháng không hợp lệ: %s (expected YYYY-MM or MM/YYYY)", forMonth)
}

// buildWeekdayToDayMap builds a mapping from weekday labels to day numbers.
// For example, if 2026-07-01 is Wednesday (T4), then T4=1, T5=2, ..., CN=7, T2=8, etc.
func buildWeekdayToDayMap(year int, month time.Month, firstWeekday string) map[string]int {
	// Vietnamese weekday labels
	weekdayMap := map[string]string{
		"T2": "Monday",
		"T3": "Tuesday",
		"T4": "Wednesday",
		"T5": "Thursday",
		"T6": "Friday",
		"T7": "Saturday",
		"CN": "Sunday",
	}

	// Find the day of month for the first occurrence of each weekday
	result := make(map[string]int)

	// Find which weekday corresponds to the first weekday in the file
	firstWeekdayEN, ok := weekdayMap[strings.ToUpper(firstWeekday)]
	if !ok {
		// Default to treating unknown weekdays as Monday
		firstWeekdayEN = "Monday"
	}

	// Find the first occurrence of this weekday in the month
	for d := 1; d <= 7; d++ {
		date := time.Date(year, month, d, 0, 0, 0, 0, time.UTC)
		if date.Weekday().String() == firstWeekdayEN {
			result[strings.ToUpper(firstWeekday)] = d
			break
		}
	}

	// Fill in the rest of the weekdays
	weekdayOrder := []string{"T2", "T3", "T4", "T5", "T6", "T7", "CN"}
	firstDay := result[strings.ToUpper(firstWeekday)]

	for i, wd := range weekdayOrder {
		if _, exists := result[wd]; !exists {
			// Calculate day offset from first weekday
			weekdayIndex := i
			firstWeekdayIndex := -1
			for j, w := range weekdayOrder {
				if strings.EqualFold(w, firstWeekday) {
					firstWeekdayIndex = j
					break
				}
			}

			if firstWeekdayIndex >= 0 {
				offset := (weekdayIndex - firstWeekdayIndex + 7) % 7
				result[wd] = firstDay + offset
			}
		}
	}

	return result
}

// weeklyPaymentRowHasPositiveHours reports whether any entry carries positive
// hours, so a placeholder row (blank code, no positive hours) stays filtered
// while an all-zero row for a real employee survives as a deletion request.
func weeklyPaymentRowHasPositiveHours(entries []WeeklyPaymentEntryData) bool {
	for _, e := range entries {
		if e.Hours > 0 {
			return true
		}
	}
	return false
}
