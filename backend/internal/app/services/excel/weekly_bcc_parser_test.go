package excel

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// weeklyBCCFixture returns the path to the WeeklyBCC test fixture file.
// Uses os.Getwd() like the existing fixtureDir() pattern in eva06_test.go.
func weeklyBCCFixture(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// This test lives at backend/internal/app/services/excel/ → go up 4 levels to backend/, 5 to repo root
	return filepath.Clean(filepath.Join(cwd, "..", "..", "..", "..", "..", "docs", "WeeklyBCC", "BCC LGD.xlsx"))
}

func TestExtractShiftType(t *testing.T) {
	tests := []struct {
		sheetName string
		expected  string
	}{
		{"BCC-HC", "HC"},
		{"BCC-OT150", "OT150"},
		{"BCC-OT30", "OT30"},
		{"BCC-OT390", "OT390"},
		{"BCC-", ""},
		{"BCC", ""},
		{"HC", ""},
		{"Something", ""},
		{"bcc-HC", "HC"},       // lowercase prefix — case-insensitive
		{"Bcc-OT150", "OT150"}, // mixed case
	}
	for _, tt := range tests {
		t.Run(tt.sheetName, func(t *testing.T) {
			got := ExtractShiftType(tt.sheetName)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestParseWeeklyBCCFile_WithBCClgdFile(t *testing.T) {
	// Use the actual WeeklyBCC sample file
	f, err := excelize.OpenFile(weeklyBCCFixture(t))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	// Expected visible BCC-* sheets
	expectedSheets := []string{"BCC-HC", "BCC-OT150", "BCC-OT30", "BCC-OT200", "BCC-OT270", "BCC-OT300", "BCC-OT390"}

	result, err := ParseWeeklyBCCFile(f, expectedSheets)
	require.NoError(t, err)
	require.Len(t, result.Sheets, 7)

	// Verify sheet names match shift types
	shiftTypes := make(map[string]bool)
	for _, s := range result.Sheets {
		shiftTypes[s.ShiftType] = true
	}
	assert.True(t, shiftTypes["HC"])
	assert.True(t, shiftTypes["OT150"])
	assert.True(t, shiftTypes["OT30"])
	assert.True(t, shiftTypes["OT200"])
	assert.True(t, shiftTypes["OT270"])
	assert.True(t, shiftTypes["OT300"])
	assert.True(t, shiftTypes["OT390"])

	// Verify BCC-HC sheet (new fixture has no "Dự án" column — dates start at col D)
	hcSheet := findSheetByShiftType(result.Sheets, "HC")
	require.NotNil(t, hcSheet)
	assert.NotEmpty(t, hcSheet.Employees)

	// First (and only) employee in BCC-HC: Trần Đăng Đức
	firstEmp := hcSheet.Employees[0]
	assert.Equal(t, "031092020742", firstEmp.EmployeeCode)
	assert.Contains(t, firstEmp.FullName, "Đức")
	assert.Empty(t, firstEmp.Project) // new fixture has no "Dự án" column
	assert.NotEmpty(t, firstEmp.Entries)

	// Verify date entries have reasonable values. Hours may be 0: an explicit
	// 0 cell is a deletion request for that day's chờ duyệt timesheet.
	for _, entry := range firstEmp.Entries {
		assert.True(t, entry.Date.Year() >= 2020 && entry.Date.Year() <= 2040, "date year should be reasonable")
		assert.GreaterOrEqual(t, entry.Hours, float64(0))
	}

	// Verify BCC-OT150 sheet (no "Dự án" column — dates start at col D)
	otSheet := findSheetByShiftType(result.Sheets, "OT150")
	require.NotNil(t, otSheet)
	assert.NotEmpty(t, otSheet.Employees)
	// OT sheets should not have project populated
	for _, emp := range otSheet.Employees {
		assert.Empty(t, emp.Project, "OT sheets should not have project column")
	}

	// Same employee should appear in both sheets
	otFirstEmp := findEmployeeByCode(otSheet.Employees, "031092020742")
	require.NotNil(t, otFirstEmp, "Trần Đăng Đức should appear in OT150 sheet")
	assert.Contains(t, otFirstEmp.FullName, "Đức")
}

func TestParseWeeklyBCCFile_DateColumns(t *testing.T) {
	f, err := excelize.OpenFile(weeklyBCCFixture(t))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	result, err := ParseWeeklyBCCFile(f, []string{"BCC-HC"})
	require.NoError(t, err)
	require.Len(t, result.Sheets, 1)

	hcSheet := result.Sheets[0]
	require.NotEmpty(t, hcSheet.Employees)

	// Check that we have entries with dates in May 2026
	foundMayDate := false
	for _, emp := range hcSheet.Employees {
		for _, entry := range emp.Entries {
			if entry.Date.Year() == 2026 && entry.Date.Month() == time.May {
				foundMayDate = true
				break
			}
		}
		if foundMayDate {
			break
		}
	}
	assert.True(t, foundMayDate, "should find entries with May 2026 dates")
}

func TestDetectFormat_WeeklyBCC(t *testing.T) {
	f, err := excelize.OpenFile(weeklyBCCFixture(t))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	result, err := DetectFormat(f)
	require.NoError(t, err)
	assert.Equal(t, FormatWeeklyBCC, result.Format)
	assert.NotEmpty(t, result.WeeklyBCCSheets)

	// Should contain BCC-HC, BCC-OT150, etc.
	sheetSet := make(map[string]bool)
	for _, s := range result.WeeklyBCCSheets {
		sheetSet[s] = true
	}
	assert.True(t, sheetSet["BCC-HC"])
	assert.True(t, sheetSet["BCC-OT150"])
	assert.True(t, sheetSet["BCC-OT390"])
}

func TestIsProjectColumn(t *testing.T) {
	assert.True(t, isProjectColumn("Dự án"))
	assert.True(t, isProjectColumn("Dự Án"))
	assert.True(t, isProjectColumn("du an"))
	assert.False(t, isProjectColumn(""))
	assert.False(t, isProjectColumn("HC"))
	assert.False(t, isProjectColumn("01"))
}

func TestParseExcelDate(t *testing.T) {
	// Excel serial for 2026-05-01
	d, ok := parseExcelDate("46143")
	assert.True(t, ok)
	assert.Equal(t, 2026, d.Year())

	// Invalid values
	_, ok = parseExcelDate("not-a-number")
	assert.False(t, ok)

	_, ok = parseExcelDate("0")
	assert.False(t, ok)

	_, ok = parseExcelDate("-1")
	assert.False(t, ok)
}

// Helpers

func findSheetByShiftType(sheets []WeeklyBCCSheetData, shiftType string) *WeeklyBCCSheetData {
	for i := range sheets {
		if sheets[i].ShiftType == shiftType {
			return &sheets[i]
		}
	}
	return nil
}

func findEmployeeByCode(employees []WeeklyBCCEmployeeData, code string) *WeeklyBCCEmployeeData {
	for i := range employees {
		if employees[i].EmployeeCode == code {
			return &employees[i]
		}
	}
	return nil
}
