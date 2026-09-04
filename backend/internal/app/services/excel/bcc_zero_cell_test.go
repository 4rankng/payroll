package excel

import (
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// The zero-cell contract: an explicit 0 in a data cell is a deletion request —
// the parser emits a zero-hour entry so the import pipeline deletes the
// matching chờ duyệt timesheet. Blank cells stay skipped (no information).

func TestParseDateRowBCCFile_ExplicitZeroIsDeletionRequest(t *testing.T) {
	f := buildM1StyleWorkbook(t)
	defer func() { _ = f.Close() }()

	// Employee 1: NT 21/08 was 9.0 → explicit 0. OT 21/08 (J9=1.0) and
	// CN 23/08 (M9=8.0) keep positive hours and must survive.
	if err := f.SetCellValue("M1", "I9", 0); err != nil {
		t.Fatalf("set I9: %v", err)
	}

	data, err := ParseDateRowBCCFile(f, []string{"M1"})
	if err != nil {
		t.Fatalf("ParseDateRowBCCFile: %v", err)
	}
	first := data.Employees[0]
	want := []struct {
		day   int
		label string
		hours float64
	}{
		{21, "NT", 0},
		{21, "OT", 1},
		{23, "CN", 8},
	}
	if len(first.Entries) != len(want) {
		t.Fatalf("entries = %d, want %d (%+v)", len(first.Entries), len(want), first.Entries)
	}
	for i, w := range want {
		got := first.Entries[i]
		if got.DayNum != w.day || got.ShiftLabel != w.label || got.Hours != w.hours {
			t.Errorf("entry %d = {%d %s %.1f}, want {%d %s %.1f}",
				i, got.DayNum, got.ShiftLabel, got.Hours, w.day, w.label, w.hours)
		}
	}
}

func TestParseDateRowBCCFile_AllZeroRowKeepsEmployee(t *testing.T) {
	f := buildM1StyleWorkbook(t)
	defer func() { _ = f.Close() }()

	// Employee 2's every cell → 0: the employee must stay (bulk deletion
	// request), carrying only zero-hour entries.
	for _, cell := range []string{"I10", "J10", "K10"} {
		if err := f.SetCellValue("M1", cell, 0); err != nil {
			t.Fatalf("set %s: %v", cell, err)
		}
	}

	data, err := ParseDateRowBCCFile(f, []string{"M1"})
	if err != nil {
		t.Fatalf("ParseDateRowBCCFile: %v", err)
	}
	if len(data.Employees) != 2 {
		t.Fatalf("employees = %d, want 2", len(data.Employees))
	}
	second := data.Employees[1]
	if len(second.Entries) == 0 {
		t.Fatal("all-zero employee dropped — would lose the bulk deletion request")
	}
	for _, e := range second.Entries {
		if e.Hours != 0 {
			t.Errorf("entry {day=%d %s} = %.1f, want 0", e.DayNum, e.ShiftLabel, e.Hours)
		}
	}
}

func TestParseBCCFile_ExplicitZeroIsDeletionRequest(t *testing.T) {
	f := buildDecimalHoursBCCXLSX(t)
	defer func() { _ = f.Close() }()

	// M11 (T3=16th, CB N, was 7.5) → explicit 0 under a number format.
	mustSetNumberWithFormat(t, f, "BCC", "M11", 0, "#,##0")
	// T5=18th pair (U11 CB N 8.0, V11 OT N 4.0) → blanked: no day-18 entry.
	for _, cell := range []string{"U11", "V11"} {
		if err := f.SetCellValue("BCC", cell, ""); err != nil {
			t.Fatalf("blank %s: %v", cell, err)
		}
	}

	data, err := ParseBCCFile(f)
	if err != nil {
		t.Fatalf("ParseBCCFile: %v", err)
	}
	emp := data.Employees[0]

	var zeroEntry, day18 bool
	for _, e := range emp.Entries {
		if e.DayNum == 16 && e.ShiftLabel == "CB N" {
			if e.Hours != 0 {
				t.Errorf("day 16 CB N = %.1f, want explicit 0", e.Hours)
			}
			zeroEntry = true
		}
		if e.DayNum == 18 {
			day18 = true
		}
	}
	if !zeroEntry {
		t.Error("explicit 0 cell dropped — deletion request lost")
	}
	if day18 {
		t.Error("blank cell produced an entry — blank must stay skipped")
	}
}

func TestParseBCCFile_DraftRowZerosFilteredButNumberedRowKept(t *testing.T) {
	f := buildDecimalHoursBCCXLSX(t)
	defer func() { _ = f.Close() }()

	// Row 12: identity + blank STT + explicit zeros → placeholder, filtered.
	mustSet(t, f, "BCC", "C12", "LV000927")
	mustSet(t, f, "BCC", "D12", "040205013153")
	mustSet(t, f, "BCC", "E12", "Nguyễn Văn Nháp")
	mustSetNumberWithFormat(t, f, "BCC", "I12", 0, "#,##0")

	// Row 13: STT present (header detection picks col A — "STT" sits at both
	// A7 and B7 and the first wins) + explicit zeros → real employee with a
	// bulk deletion request, kept.
	mustSet(t, f, "BCC", "A13", "2")
	mustSet(t, f, "BCC", "C13", "LV000928")
	mustSet(t, f, "BCC", "D13", "040205013154")
	mustSet(t, f, "BCC", "E13", "Trần Văn Xóa Hết")
	mustSetNumberWithFormat(t, f, "BCC", "I13", 0, "#,##0")

	data, err := ParseBCCFile(f)
	if err != nil {
		t.Fatalf("ParseBCCFile: %v", err)
	}
	if len(data.Employees) != 2 {
		t.Fatalf("employees = %d, want 2 (original + numbered zero row; draft filtered): %+v",
			len(data.Employees), data.Employees)
	}
	last := data.Employees[1]
	if last.FullName != "Trần Văn Xóa Hết" {
		t.Fatalf("second employee = %q, want Trần Văn Xóa Hết", last.FullName)
	}
	if len(last.Entries) == 0 {
		t.Fatal("numbered all-zero row dropped its entries — deletion request lost")
	}
	for _, e := range last.Entries {
		if e.Hours != 0 {
			t.Errorf("entry {day=%d %s} = %.1f, want 0", e.DayNum, e.ShiftLabel, e.Hours)
		}
	}
}

func TestParseWeeklyBCCFile_ExplicitZeroIsDeletionRequest(t *testing.T) {
	f := newWeeklyBCCZeroWorkbook(t)
	defer func() { _ = f.Close() }()

	const sheet = "BCC-HC"
	parsed, err := ParseWeeklyBCCFile(f, []string{sheet})
	if err != nil {
		t.Fatalf("ParseWeeklyBCCFile: %v", err)
	}
	if len(parsed.Sheets) != 1 || len(parsed.Sheets[0].Employees) != 1 {
		t.Fatalf("want 1 sheet/employee, got %+v", parsed.Sheets)
	}
	entries := parsed.Sheets[0].Employees[0].Entries
	if len(entries) != 2 {
		t.Fatalf("entries = %#v, want [positive, zero]", entries)
	}
	if entries[0].Hours != 8 {
		t.Errorf("first entry hours = %.1f, want 8", entries[0].Hours)
	}
	if entries[1].Hours != 0 {
		t.Errorf("explicit-zero entry hours = %.1f, want 0 (deletion request)", entries[1].Hours)
	}
}

// newWeeklyBCCZeroWorkbook: row 1 headers (B=Mã nhân viên, C=Họ Tên, D/E date
// serials, F=Tổng stop), row 2 employee with 8h + explicit 0, row 3 a code-less
// all-zero placeholder that must be filtered.
func newWeeklyBCCZeroWorkbook(t *testing.T) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	const sheet = "BCC-HC"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		t.Fatalf("rename sheet: %v", err)
	}
	for cell, v := range map[string]any{
		"B1": "Mã nhân viên",
		"C1": "Họ Tên",
		"D1": time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
		"E1": time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC),
		"F1": "Tổng",
		"B2": "040205013152",
		"C2": "Nguyễn Zero Tuần",
		"D2": 8.0,
		"E2": 0,
		"C3": "Nguyễn Văn Nháp",
		"D3": 0,
	} {
		if err := f.SetCellValue(sheet, cell, v); err != nil {
			t.Fatalf("set %s: %v", cell, err)
		}
	}
	return f
}

func TestParseWeeklyPaymentFile_ExplicitZeroIsDeletionRequest(t *testing.T) {
	f := excelize.NewFile()
	const sheetName = "Lương 520"
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		t.Fatalf("rename sheet: %v", err)
	}
	setCellValue(f, sheetName, "F8", "1")
	setCellValue(f, sheetName, "F9", "T7")
	setCellValue(f, sheetName, "F10", "HC")
	setCellValue(f, sheetName, "G10", "TCN")
	setCellValue(f, sheetName, "B11", "012345678901")
	setCellValue(f, sheetName, "C11", "Nguyễn Văn Kiên")
	setCellValue(f, sheetName, "F11", "8")
	setCellValue(f, sheetName, "G11", "0")
	// Placeholder row: blank code + explicit zero → filtered.
	setCellValue(f, sheetName, "C12", "Nguyễn Văn Nháp")
	setCellValue(f, sheetName, "F12", "0")

	parsed, err := ParseWeeklyPaymentFile(f, []string{sheetName}, "2026-08")
	if err != nil {
		t.Fatalf("ParseWeeklyPaymentFile: %v", err)
	}
	if len(parsed.Sheets[0].Employees) != 1 {
		t.Fatalf("employees = %d, want 1 (placeholder filtered)", len(parsed.Sheets[0].Employees))
	}
	entries := parsed.Sheets[0].Employees[0].Entries
	if len(entries) != 2 {
		t.Fatalf("entries = %#v, want [HC 8, TCN 0]", entries)
	}
	if entries[1].ShiftKey != "TCN" || entries[1].Hours != 0 {
		t.Errorf("explicit-zero entry = {%s %.1f}, want {TCN 0}", entries[1].ShiftKey, entries[1].Hours)
	}
}

func TestParseMultiPositionFile_ExplicitZeroIsDeletionRequest(t *testing.T) {
	f := excelize.NewFile()
	const sheet = "Thợ hàn"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		t.Fatalf("rename sheet: %v", err)
	}
	for cell, v := range map[string]any{
		"A4": "STT", "B4": "Mã nhân viên", "C4": "Họ và tên",
		"D4": "1", "E4": "2",
		"D5": 600000, "E5": 600000,
		"B7": "031085013087", "C7": "Bùi Văn Quân",
		"D7": 8.0, "E7": 0,
		// Placeholder row: blank code + explicit zeros → filtered.
		"C8": "Nguyễn Văn Nháp", "D8": 0,
	} {
		if err := f.SetCellValue(sheet, cell, v); err != nil {
			t.Fatalf("set %s: %v", cell, err)
		}
	}

	parsed, err := ParseMultiPositionFile(f, []string{sheet})
	if err != nil {
		t.Fatalf("ParseMultiPositionFile: %v", err)
	}
	emps := parsed.Sheets[0].Employees
	if len(emps) != 1 {
		t.Fatalf("employees = %d, want 1 (placeholder filtered): %+v", len(emps), emps)
	}
	entries := emps[0].Entries
	if len(entries) != 2 {
		t.Fatalf("entries = %#v, want [positive, zero]", entries)
	}
	if entries[0].Hours != 8 || entries[1].Hours != 0 {
		t.Errorf("hours = [%.1f, %.1f], want [8, 0] (explicit zero = deletion request)", entries[0].Hours, entries[1].Hours)
	}
}
