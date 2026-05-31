package excel

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

const bccTestFile = "/Users/dev/Documents/projects/payroll/docs/timesheets-excel/BCC LƯƠNG DỰ ÁN EVA Sample.xlsx"

func openBCCTestData(t *testing.T) *BCCImportData {
	t.Helper()
	f, err := excelize.OpenFile(bccTestFile)
	if err != nil {
		t.Skipf("BCC test file not available: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	data, err := ParseBCCFile(f)
	if err != nil {
		t.Fatalf("ParseBCCFile: %v", err)
	}
	return data
}

func TestParseBCCFile_EmployeeCount(t *testing.T) {
	data := openBCCTestData(t)
	if len(data.Employees) != 32 {
		t.Errorf("employee count = %d, want 32", len(data.Employees))
	}
}

func TestParseBCCFile_ShiftRates(t *testing.T) {
	data := openBCCTestData(t)
	cases := []struct {
		label string
		want  int64
	}{
		{"CB N", 36000},
		{"OT N", 54000},
		{"CB Đ", 46800},
		{"OT Đ", 75600},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			got, ok := data.ShiftRates[tc.label]
			if !ok {
				t.Fatalf("ShiftRates[%q] not found", tc.label)
			}
			if got != tc.want {
				t.Errorf("ShiftRates[%q] = %d, want %d", tc.label, got, tc.want)
			}
		})
	}
}

func TestParseBCCFile_NonZeroHoursOnly(t *testing.T) {
	data := openBCCTestData(t)
	for _, emp := range data.Employees {
		for _, e := range emp.Entries {
			if e.Hours == 0 {
				t.Errorf("employee %q has zero-hours entry day=%d shift=%q", emp.FullName, e.DayNum, e.ShiftLabel)
			}
		}
	}
}

// TestParseBCCFile_DateSerialDayNumbers verifies that day numbers stored as Excel
// date serials (format "dd") are read correctly. In x.xlsx the day cells hold
// serial 22–28 formatted as "dd", which excelize's formatted-value path converts
// to "21"–"27" (off by 1 due to epoch). The parser must return 22–28.
func TestParseBCCFile_DateSerialDayNumbers(t *testing.T) {
	const xFile = "/Users/dev/Downloads/x.xlsx"
	f, err := excelize.OpenFile(xFile)
	if err != nil {
		t.Skipf("x.xlsx not available: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	data, err := ParseBCCFile(f)
	if err != nil {
		t.Fatalf("ParseBCCFile: %v", err)
	}

	// Collect all unique day numbers across all employees.
	daySet := make(map[int]bool)
	for _, emp := range data.Employees {
		for _, e := range emp.Entries {
			daySet[e.DayNum] = true
		}
	}

	// Days with entries in x.xlsx: 22, 23, 25, 26, 27, 28.
	// Day 24 (Sunday) has no shifts for any employee.
	for _, d := range []int{22, 23, 25, 26, 27, 28} {
		if !daySet[d] {
			t.Errorf("expected day %d in parsed entries, got days: %v", d, daySet)
		}
	}
	// Must NOT contain day 21 (off-by-1 from date serial epoch bug).
	if daySet[21] {
		t.Error("day 21 should not appear (off-by-1 from date serial epoch bug)")
	}
}

func TestParseBCCFile_FirstEmployee(t *testing.T) {
	data := openBCCTestData(t)
	const targetCCCD = "036203011380"
	var found *BCCEmployeeData
	for i := range data.Employees {
		if data.Employees[i].CCCD == targetCCCD {
			found = &data.Employees[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("employee CCCD %q not found", targetCCCD)
	}
	if found.FullName != "Nguyễn Trung Hải" {
		t.Errorf("FullName = %q, want %q", found.FullName, "Nguyễn Trung Hải")
	}
	if len(found.Entries) == 0 {
		t.Error("expected at least one entry but got none")
	}
	daySet := make(map[int]bool)
	for _, e := range found.Entries {
		daySet[e.DayNum] = true
	}
	for _, d := range []int{22, 23, 25} {
		if !daySet[d] {
			t.Errorf("expected entries on day %d, got days: %v", d, daySet)
		}
	}
	// Day 21 must NOT appear (off-by-1 epoch bug was fixed).
	if daySet[21] {
		t.Errorf("day 21 should not appear (off-by-1 epoch bug), got days: %v", daySet)
	}
}
