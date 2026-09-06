package excel

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// MultiPositionImportData holds all parsed data from a multi-position BCC file.
type MultiPositionImportData struct {
	Sheets []PositionSheetData
}

// PositionSheetData holds parsed data from one position sheet.
type PositionSheetData struct {
	Position  string // from sheet name
	Employees []PositionEmployeeData
}

// PositionEmployeeData holds one employee's parsed attendance entries from a position sheet.
type PositionEmployeeData struct {
	EmployeeCode string // Col B value (actually CCCD in the new template)
	FullName     string
	Entries      []PositionEntryData
}

// PositionEntryData represents a single non-zero hours entry with resolved rate.
type PositionEntryData struct {
	DayNum  int
	RateVND int // resolved VND rate from row 5 (int matches Flatten() return type)
	Hours   float64
}

// ParseMultiPositionFile parses all position sheets in a multi-position BCC file.
func ParseMultiPositionFile(f *excelize.File, sheetNames []string) (*MultiPositionImportData, error) {
	result := &MultiPositionImportData{}

	for _, sheetName := range sheetNames {
		sheetData, err := parsePositionSheet(f, sheetName)
		if err != nil {
			return nil, fmt.Errorf("sheet %q: %w", sheetName, err)
		}
		result.Sheets = append(result.Sheets, *sheetData)
	}

	if len(result.Sheets) == 0 {
		return nil, fmt.Errorf("no position sheets parsed")
	}

	return result, nil
}

// headerMap stores the column indices for key fields detected from row 4.
type headerMap struct {
	sttCol      int
	empCodeCol  int         // "Mã nhân viên"
	fullNameCol int         // "Họ và tên"
	dayStartCol int         // first column with a day number
	dayCols     map[int]int // colIdx → dayNum
	stopCol     int         // column where "Tổng" appears
}

// parsePositionSheet parses a single position sheet.
func parsePositionSheet(f *excelize.File, sheetName string) (*PositionSheetData, error) {
	// Step 1: Build header map from row 4
	hm, err := buildHeaderMap(f, sheetName)
	if err != nil {
		return nil, err
	}

	// Step 2: Build rate map from row 5
	rateByCol := buildRateMap(f, sheetName, hm)

	// Step 3: Parse employees from row 7+
	employees := parsePositionEmployees(f, sheetName, hm, rateByCol)

	return &PositionSheetData{
		Position:  sheetName,
		Employees: employees,
	}, nil
}

// buildHeaderMap reads row 4 to find column roles and day columns.
func buildHeaderMap(f *excelize.File, sheet string) (*headerMap, error) {
	hm := &headerMap{
		dayCols: make(map[int]int),
		stopCol: 200,
	}

	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) < 4 {
		return nil, fmt.Errorf("sheet has fewer than 4 rows")
	}

	row4 := rows[3] // Row 4 is index 3

	// Find header columns
	for colIdx, cell := range row4 {
		val := strings.TrimSpace(cell)
		if val == "STT" && hm.sttCol == 0 {
			hm.sttCol = colIdx
		}
		if headerContainsAny(val, "Mã nhân viên", "Ma nhan vien") && hm.empCodeCol == 0 {
			hm.empCodeCol = colIdx
		}
		if headerContainsAny(val, "Họ và tên", "Ho va ten", "Họ và Tên") && hm.fullNameCol == 0 {
			hm.fullNameCol = colIdx
		}
	}

	if hm.empCodeCol == 0 || hm.fullNameCol == 0 {
		return nil, fmt.Errorf("could not find required headers (Mã nhân viên, Họ và tên) in row 4")
	}

	// Scan for day numbers starting after the last header column
	lastHeaderCol := max(hm.fullNameCol, hm.sttCol)

	currentDay := 0
	for colIdx := lastHeaderCol + 1; colIdx < 200; colIdx++ {
		if colIdx >= len(row4) {
			// Try reading directly from excelize for cells beyond the row data
			cn, err := excelize.CoordinatesToCellName(colIdx+1, 4)
			if err != nil {
				break
			}
			val, err := f.GetCellValue(sheet, cn, excelize.Options{RawCellValue: true})
			if err != nil {
				break
			}
			val = strings.TrimSpace(val)

			if strings.Contains(val, "Tổng") {
				hm.stopCol = colIdx
				break
			}

			if val != "" {
				if n, err := strconv.Atoi(val); err == nil && n >= 1 && n <= 31 {
					if hm.dayStartCol == 0 {
						hm.dayStartCol = colIdx
					}
					currentDay = n
					hm.dayCols[colIdx] = n
				}
			}
			continue
		}

		val := strings.TrimSpace(row4[colIdx])
		if strings.Contains(val, "Tổng") {
			hm.stopCol = colIdx
			break
		}

		if val != "" {
			dayNum := 0
			if n, err := strconv.Atoi(val); err == nil {
				dayNum = n
			} else if serial, err := strconv.ParseFloat(val, 64); err == nil {
				dayNum = int(serial + 0.5)
			}
			if dayNum >= 1 && dayNum <= 31 {
				if hm.dayStartCol == 0 {
					hm.dayStartCol = colIdx
				}
				currentDay = dayNum
				hm.dayCols[colIdx] = dayNum
			}
		}

		// For empty columns within a day group, propagate the current day
		if currentDay > 0 && val == "" {
			hm.dayCols[colIdx] = currentDay
		}
	}

	if len(hm.dayCols) == 0 {
		return nil, fmt.Errorf("no day columns found in row 4")
	}

	return hm, nil
}

// buildRateMap reads row 5 rate values and resolves them per column.
// Sparse rates are propagated within day groups.
func buildRateMap(f *excelize.File, sheet string, hm *headerMap) map[int]int {
	rateByCol := make(map[int]int)

	if hm.dayStartCol == 0 {
		return rateByCol
	}

	// Read rate values from row 5
	lastKnownRate := 0
	lastKnownDay := 0

	for colIdx := hm.dayStartCol; colIdx < hm.stopCol; colIdx++ {
		dayNum, hasDay := hm.dayCols[colIdx]
		if !hasDay {
			continue
		}

		cn, err := excelize.CoordinatesToCellName(colIdx+1, 5)
		if err != nil {
			continue
		}
		val, err := f.GetCellValue(sheet, cn)
		if err != nil {
			continue
		}

		rate := 0
		if val != "" {
			clean := strings.ReplaceAll(strings.TrimSpace(val), ",", "")
			clean = strings.ReplaceAll(clean, " ", "")
			if n, err := strconv.Atoi(clean); err == nil {
				rate = n
			}
		}

		if rate > 0 {
			lastKnownRate = rate
			lastKnownDay = dayNum
			rateByCol[colIdx] = rate
		} else if dayNum == lastKnownDay && lastKnownRate > 0 {
			// Propagate rate within same day group
			rateByCol[colIdx] = lastKnownRate
		}
	}

	return rateByCol
}

// parsePositionEmployees reads employee rows starting from row 7.
func parsePositionEmployees(f *excelize.File, sheet string, hm *headerMap, rateByCol map[int]int) []PositionEmployeeData {
	var employees []PositionEmployeeData
	consecutiveBlank := 0

	for row := 7; row < 500; row++ {
		// Read employee code and full name using header-detected columns
		empCode := bccCell(f, sheet, hm.empCodeCol+1, row) // +1 for 1-based coords
		fullName := bccCell(f, sheet, hm.fullNameCol+1, row)

		// Skip blank rows but tolerate a few before stopping
		if empCode == "" && fullName == "" {
			consecutiveBlank++
			if consecutiveBlank >= 5 {
				break
			}
			continue
		}
		consecutiveBlank = 0

		// Skip summary rows
		if isSummaryRow(fullName, empCode) {
			break
		}

		emp := PositionEmployeeData{
			EmployeeCode: empCode,
			FullName:     fullName,
		}

		// Parse day entries
		for colIdx := hm.dayStartCol; colIdx < hm.stopCol; colIdx++ {
			dayNum, hasDay := hm.dayCols[colIdx]
			if !hasDay {
				continue
			}

			cn, err := excelize.CoordinatesToCellName(colIdx+1, row)
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
			rate := rateByCol[colIdx] // may be 0 if no rate found

			emp.Entries = append(emp.Entries, PositionEntryData{
				DayNum:  dayNum,
				RateVND: rate,
				Hours:   hours,
			})
		}

		// Skip rows with empty code AND no positive hours (zero-hour entries do
		// not count — an all-zero row is a bulk deletion request, not a draft).
		if empCode == "" && !multiPositionRowHasPositiveHours(emp.Entries) {
			continue
		}

		employees = append(employees, emp)
	}

	return employees
}

// multiPositionRowHasPositiveHours reports whether any entry carries positive
// hours, so a placeholder row (blank code, no positive hours) stays filtered
// while an all-zero row for a real employee survives as a deletion request.
func multiPositionRowHasPositiveHours(entries []PositionEntryData) bool {
	for _, e := range entries {
		if e.Hours > 0 {
			return true
		}
	}
	return false
}
