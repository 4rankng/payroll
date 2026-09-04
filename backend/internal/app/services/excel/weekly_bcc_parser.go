package excel

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const (
	wbccMaxColScan     = 200 // Maximum columns to scan for date headers
	wbccMaxRowScan     = 500 // Maximum rows to scan for employee data
	wbccMaxConsecBlank = 5   // Consecutive blank rows before stopping
)

// WeeklyBCCImportData holds parsed data from all BCC-* sheets.
type WeeklyBCCImportData struct {
	Sheets []WeeklyBCCSheetData
}

// WeeklyBCCSheetData holds parsed data from a single BCC-<shiftType> sheet.
type WeeklyBCCSheetData struct {
	ShiftType string // e.g. "HC", "OT150", "390"
	Employees []WeeklyBCCEmployeeData
}

// WeeklyBCCEmployeeData holds parsed data for one employee row in a weekly BCC sheet.
type WeeklyBCCEmployeeData struct {
	EmployeeCode string // Col B value (CCCD)
	FullName     string // Col C value
	Project      string // Col D value (only present when col D is "Dự án")
	Entries      []WeeklyBCCEntryData
}

// WeeklyBCCEntryData holds one (date, hours) entry for an employee.
type WeeklyBCCEntryData struct {
	Date  time.Time
	Hours float64
}

// weeklyBCCHeaderMap stores detected column positions for a weekly BCC sheet.
type weeklyBCCHeaderMap struct {
	empCodeCol   int               // 1-based column for "Mã nhân viên"
	fullNameCol  int               // 1-based column for "Họ Tên"
	projectCol   int               // 1-based column for "Dự án" (0 if not present)
	dateStartCol int               // 1-based first date column
	stopCol      int               // 1-based "Tổng" column (exclusive stop)
	dateCols     map[int]time.Time // 1-based col index → date
}

// ParseWeeklyBCCFile parses all BCC-* sheets in a weekly BCC Excel file.
func ParseWeeklyBCCFile(f *excelize.File, sheetNames []string) (*WeeklyBCCImportData, error) {
	result := &WeeklyBCCImportData{}

	for _, sheetName := range sheetNames {
		shiftType := ExtractShiftType(sheetName)
		if shiftType == "" {
			continue
		}

		sheetData, err := parseWeeklyBCCSheet(f, sheetName)
		if err != nil {
			return nil, fmt.Errorf("sheet %q: %w", sheetName, err)
		}
		sheetData.ShiftType = shiftType
		result.Sheets = append(result.Sheets, *sheetData)
	}

	if len(result.Sheets) == 0 {
		return nil, fmt.Errorf("không tìm thấy sheet BCC hợp lệ")
	}

	return result, nil
}

// parseWeeklyBCCSheet parses a single BCC-<shiftType> sheet.
func parseWeeklyBCCSheet(f *excelize.File, sheetName string) (*WeeklyBCCSheetData, error) {
	// Step 1: Build header map from row 1
	hm, err := buildWeeklyBCCHeaderMap(f, sheetName)
	if err != nil {
		return nil, err
	}

	// Step 2: Parse employees from row 2+
	employees := parseWeeklyBCCEmployees(f, sheetName, hm)

	return &WeeklyBCCSheetData{
		Employees: employees,
	}, nil
}

// buildWeeklyBCCHeaderMap reads row 1 to detect column layout and date columns.
//
// Layout variants:
//   - With project: STT | Mã nhân viên | Họ Tên | Dự án | date1 | date2 | ... | Tổng
//   - Without project: STT | Mã nhân viên | Họ Tên | date1 | date2 | ... | Tổng
func buildWeeklyBCCHeaderMap(f *excelize.File, sheet string) (*weeklyBCCHeaderMap, error) {
	hm := &weeklyBCCHeaderMap{
		dateCols: make(map[int]time.Time),
	}

	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) < 1 {
		return nil, fmt.Errorf("sheet không có dữ liệu")
	}

	row1 := rows[0]

	// Detect header columns (A=STT, B=Mã nhân viên, C=Họ Tên)
	hm.empCodeCol = 2  // B
	hm.fullNameCol = 3 // C

	// Check col D (index 3): is it "Dự án" or a date?
	dateStartCol := 4 // default: dates start at col D (1-based)
	if len(row1) > 3 {
		colDVal := strings.TrimSpace(row1[3])
		if isProjectColumn(colDVal) {
			hm.projectCol = 4 // D is "Dự án"
			dateStartCol = 5  // dates start at col E
		}
	}
	hm.dateStartCol = dateStartCol

	// Scan date columns using raw cell values (GetRows returns formatted day numbers, not serials)
	for colIdx := dateStartCol - 1; colIdx < wbccMaxColScan; colIdx++ {
		// Check if column is hidden
		colName, err := excelize.ColumnNumberToName(colIdx + 1)
		if err != nil {
			break
		}
		visible, err := f.GetColVisible(sheet, colName)
		if err == nil && !visible {
			continue
		}

		cn, err := excelize.CoordinatesToCellName(colIdx+1, 1)
		if err != nil {
			break
		}

		// Check formatted value for "Tổng" (total) column
		formatted, err := f.GetCellValue(sheet, cn)
		if err != nil {
			break
		}
		formatted = strings.TrimSpace(formatted)
		if strings.Contains(formatted, "Tổng") || strings.Contains(formatted, "tổng") {
			hm.stopCol = colIdx + 1 // 1-based
			break
		}

		// Read raw value (serial number) for date parsing
		rawVal, err := f.GetCellValue(sheet, cn, excelize.Options{RawCellValue: true})
		if err != nil || rawVal == "" {
			continue
		}

		parsedDate, ok := parseExcelDate(rawVal)
		if ok {
			hm.dateCols[colIdx+1] = parsedDate // store as 1-based
		}
	}

	if len(hm.dateCols) == 0 {
		return nil, fmt.Errorf("không tìm thấy cột ngày trong sheet")
	}

	// Default stopCol if "Tổng" not found
	if hm.stopCol == 0 {
		hm.stopCol = wbccMaxColScan
	}

	return hm, nil
}

// parseWeeklyBCCEmployees reads employee rows from row 2 downward.
func parseWeeklyBCCEmployees(f *excelize.File, sheet string, hm *weeklyBCCHeaderMap) []WeeklyBCCEmployeeData {
	var employees []WeeklyBCCEmployeeData
	consecutiveBlank := 0

	for row := 2; row < wbccMaxRowScan; row++ {
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
			if consecutiveBlank >= wbccMaxConsecBlank {
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

		emp := WeeklyBCCEmployeeData{
			EmployeeCode: empCode,
			FullName:     fullName,
		}

		// Read project column if present
		if hm.projectCol > 0 {
			emp.Project = bccCell(f, sheet, hm.projectCol, row)
		}

		// Parse date entries
		for col1Based := hm.dateStartCol; col1Based < hm.stopCol; col1Based++ {
			date, hasDate := hm.dateCols[col1Based]
			if !hasDate {
				continue
			}

			cn, err := excelize.CoordinatesToCellName(col1Based, row)
			if err != nil {
				continue
			}
			// RawCellValue bypasses the cell's number format so 7.5h with a
			// "0" format is read as 7.5, not rounded to 8.
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
			emp.Entries = append(emp.Entries, WeeklyBCCEntryData{
				Date:  date,
				Hours: hours,
			})
		}

		// Skip rows with empty code AND no positive hours (zero-hour entries do
		// not count — an all-zero row is a bulk deletion request, not a draft).
		if empCode == "" && !weeklyBCCRowHasPositiveHours(emp.Entries) {
			continue
		}

		employees = append(employees, emp)
	}

	return employees
}

// isProjectColumn checks if a cell value indicates a "Dự án" (project) column.
func isProjectColumn(val string) bool {
	lower := strings.ToLower(strings.TrimSpace(val))
	return strings.Contains(lower, "dự án") ||
		strings.Contains(lower, "du an") ||
		strings.Contains(lower, "dự àn") ||
		lower == "project" ||
		lower == "cty" ||
		lower == "nhà máy"
}

// parseExcelDate attempts to parse a cell value as an Excel date serial number.
// Returns the date and true if successful, or zero value and false.
func parseExcelDate(val string) (time.Time, bool) {
	// Try as float serial number
	serial, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return time.Time{}, false
	}

	// Excel date serial must be positive and reasonable (> 1 to skip small numbers)
	if serial < 1 {
		return time.Time{}, false
	}

	// Fractional part represents time — we only want the date
	serial = math.Floor(serial)

	// Convert Excel serial to time.Time
	// Excel epoch: 1900-01-01 = serial 1 (with the Lotus 123 bug: 1900-02-29 = serial 60)
	// excelize.ExcelDateToTime handles this correctly
	t, err := excelize.ExcelDateToTime(serial, false)
	if err != nil {
		return time.Time{}, false
	}

	// Only accept dates in a reasonable range (year 2020-2040)
	if t.Year() < 2020 || t.Year() > 2040 {
		return time.Time{}, false
	}

	return t, true
}

// weeklyBCCRowHasPositiveHours reports whether any entry carries positive
// hours, so a placeholder row (blank code, no positive hours) stays filtered
// while an all-zero row for a real employee survives as a deletion request.
func weeklyBCCRowHasPositiveHours(entries []WeeklyBCCEntryData) bool {
	for _, e := range entries {
		if e.Hours > 0 {
			return true
		}
	}
	return false
}
