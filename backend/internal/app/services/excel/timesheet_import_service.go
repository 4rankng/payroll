package excel

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// TimesheetImportData represents parsed data from timesheet template Excel
type TimesheetImportData struct {
	ProjectID uint
	Employees []EmployeeTimesheetData
}

// EmployeeTimesheetData represents timesheet data for one employee
type EmployeeTimesheetData struct {
	EmployeeID uint
	Position   string
	Entries    []TimesheetEntryData
}

// TimesheetEntryData represents a single timesheet entry
type TimesheetEntryData struct {
	Date       string             // YYYY-MM-DD format
	DayType    string             // e.g., "Ngày thường", "Ngày nghỉ"
	ShiftHours map[string]float64 // shift type -> hours, e.g., {"Ca ngày": 8, "Ca đêm": 0}
}

// ParseTimesheetTemplateFile parses an uploaded Excel file in the timesheet template format
func ParseTimesheetTemplateFile(f *excelize.File) (*TimesheetImportData, error) {
	// Get the Template sheet
	sheetName := "Template"
	sheets := f.GetSheetList()
	found := false
	for _, sheet := range sheets {
		if sheet == sheetName {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("sheet '%s' not found in Excel file", sheetName)
	}

	// Get all rows from the Template sheet
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows from sheet: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("template sheet is empty")
	}

	// Parse project ID from first row (format: "Du an: [ProjectID] ProjectName (ProjectCode)")
	projectID, err := parseProjectIDFromHeader(rows[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse project ID: %w", err)
	}

	// Parse employee sections
	employees, err := parseEmployeeSections(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to parse employee sections: %w", err)
	}

	return &TimesheetImportData{
		ProjectID: projectID,
		Employees: employees,
	}, nil
}

// parseProjectIDFromHeader extracts project ID from the header row
// Format: "Dự án: [ProjectID] ProjectName (ProjectCode)"
func parseProjectIDFromHeader(headerRow []string) (uint, error) {
	if len(headerRow) < 2 {
		return 0, fmt.Errorf("header row too short")
	}

	// The project header is in column B (index 1)
	headerText := headerRow[1]

	// Extract ID using regex: [ID]
	re := regexp.MustCompile(`\[(\d+)\]`)
	matches := re.FindStringSubmatch(headerText)
	if len(matches) < 2 {
		return 0, fmt.Errorf("project ID not found in header: %s", headerText)
	}

	projectID, err := strconv.ParseUint(matches[1], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid project ID: %w", err)
	}

	return uint(projectID), nil
}

// parseEmployeeSections parses all employee sections from the rows
func parseEmployeeSections(rows [][]string) ([]EmployeeTimesheetData, error) {
	var employees []EmployeeTimesheetData

	i := 0
	for i < len(rows) {
		// Skip until we find an employee header
		// Format: "Nhan vien: [EmployeeID] EmployeeName (CCCD)"
		if i >= len(rows) || len(rows[i]) < 2 {
			i++
			continue
		}

		cellValue := rows[i][1] // Column B
		if !strings.HasPrefix(cellValue, "Nhân viên:") {
			i++
			continue
		}

		// Parse employee ID
		employeeID, err := parseEmployeeIDFromHeader(cellValue)
		if err != nil {
			// Skip malformed employee headers
			i++
			continue
		}

		// Next row should be position (format: "Vi tri: Position")
		i++
		if i >= len(rows) || len(rows[i]) < 2 {
			break
		}
		position := parsePositionFromRow(rows[i][1])

		// Next row should be column headers (STT, Ngay thang, Loai ngay, shift types...)
		i++
		if i >= len(rows) || len(rows[i]) < 3 {
			break
		}

		shiftTypes := parseShiftTypesFromHeaders(rows[i])

		// Parse data rows for this employee
		i++
		entries := []TimesheetEntryData{}
		for i < len(rows) {
			// Stop if we hit the next employee section or empty rows
			if len(rows[i]) < 2 {
				i++
				break
			}

			// Check if we've reached the next employee section
			if len(rows[i]) >= 2 && strings.HasPrefix(rows[i][1], "Nhân viên:") {
				break
			}

			// Parse data row
			entry, err := parseTimesheetEntryRow(rows[i], shiftTypes)
			if err == nil && entry != nil {
				entries = append(entries, *entry)
			}

			i++
		}

		// Add employee data
		employees = append(employees, EmployeeTimesheetData{
			EmployeeID: employeeID,
			Position:   position,
			Entries:    entries,
		})
	}

	return employees, nil
}

// parseEmployeeIDFromHeader extracts employee ID from employee header
// Format: "Nhân viên: [EmployeeID] EmployeeName (CCCD)"
func parseEmployeeIDFromHeader(headerText string) (uint, error) {
	re := regexp.MustCompile(`\[(\d+)\]`)
	matches := re.FindStringSubmatch(headerText)
	if len(matches) < 2 {
		return 0, fmt.Errorf("employee ID not found in header: %s", headerText)
	}

	employeeID, err := strconv.ParseUint(matches[1], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid employee ID: %w", err)
	}

	return uint(employeeID), nil
}

// parsePositionFromRow extracts position from position row
// Format: "Vị trí: Position"
func parsePositionFromRow(positionText string) string {
	if strings.HasPrefix(positionText, "Vị trí:") {
		return strings.TrimSpace(strings.TrimPrefix(positionText, "Vị trí:"))
	}
	return ""
}

// parseShiftTypesFromHeaders extracts shift types from header row
// Format: STT | Ngày tháng | Loại ngày | ShiftType1 | ShiftType2 | ...
func parseShiftTypesFromHeaders(headerRow []string) []string {
	var shiftTypes []string
	// Skip first 3 columns (STT, Ngày tháng, Loại ngày)
	for i := 3; i < len(headerRow); i++ {
		shiftType := strings.TrimSpace(headerRow[i])
		if shiftType != "" {
			shiftTypes = append(shiftTypes, shiftType)
		}
	}
	return shiftTypes
}

// parseTimesheetEntryRow parses a single data row into a timesheet entry
// Returns nil if the row should be skipped (empty day type or all zero hours)
func parseTimesheetEntryRow(row []string, shiftTypes []string) (*TimesheetEntryData, error) {
	if len(row) < 3 {
		return nil, fmt.Errorf("row too short")
	}

	// Column A: STT (skip)
	// Column B: Date (YYYY-MM-DD)
	date := strings.TrimSpace(row[1])
	if date == "" {
		return nil, fmt.Errorf("empty date")
	}

	// Column C: Day Type (Loai ngay)
	dayType := strings.TrimSpace(row[2])

	// Skip rows with empty day type as per requirement
	if dayType == "" {
		return nil, nil
	}

	// Parse shift hours (columns D, E, F, ...)
	shiftHours := make(map[string]float64)
	hasNonZeroHours := false

	for i, shiftType := range shiftTypes {
		colIndex := 3 + i // Column D is index 3
		if colIndex >= len(row) {
			break
		}

		hoursStr := strings.TrimSpace(row[colIndex])
		if hoursStr == "" {
			continue
		}

		hours, err := strconv.ParseFloat(hoursStr, 64)
		if err != nil {
			// Skip invalid hour values
			continue
		}

		// Only add non-zero hours as per requirement
		if hours > 0 {
			shiftHours[shiftType] = hours
			hasNonZeroHours = true
		}
	}

	// Skip entry if all hours are zero or empty
	if !hasNonZeroHours {
		return nil, nil
	}

	return &TimesheetEntryData{
		Date:       date,
		DayType:    dayType,
		ShiftHours: shiftHours,
	}, nil
}
