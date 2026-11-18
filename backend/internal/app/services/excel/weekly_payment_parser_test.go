package excel

import (
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// TestIsShiftCode tests the shift code detector.
func TestIsShiftCode(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want bool
	}{
		{"HC", "HC", true},
		{"TCN", "TCN", true},
		{"NN", "NN", true},
		{"TCNN", "TCNN", true},
		{"TC", "TC", true},
		{"compound TC NN", "TC NN", true},
		{"LE", "LE", true},
		{"compound TC LE", "TC LE", true},
		{"lowercase tc nn", "tc nn", true},
		{"lowercase", "hc", true},
		{"with space", " HC ", true},
		{"non-shift", "ABC", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isShiftCode(tt.val); got != tt.want {
				t.Errorf("isShiftCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsStopColumn tests the stop column detector.
func TestIsStopColumn(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want bool
	}{
		{"Tổng hợp", "Tổng hợp", true},
		{"TONG HOP", "TONG HOP", true},
		{"Suất ăn", "Suất ăn tăng ca", true},
		{"regular", "HC", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isStopColumn(tt.val); got != tt.want {
				t.Errorf("isStopColumn() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseForMonth tests the month string parser.
func TestParseForMonth(t *testing.T) {
	tests := []struct {
		name      string
		forMonth  string
		wantYear  int
		wantMonth time.Month
		wantErr   bool
	}{
		{"YYYY-MM format", "2026-07", 2026, time.July, false},
		{"MM/YYYY format", "07/2026", 2026, time.July, false},
		{"YYYY-MM with zero", "2026-01", 2026, time.January, false},
		{"invalid format", "invalid", 0, 0, true},
		{"empty", "", 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			year, month, err := parseForMonth(tt.forMonth)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseForMonth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if year != tt.wantYear || month != tt.wantMonth {
					t.Errorf("parseForMonth() = (%v, %v), want (%v, %v)", year, month, tt.wantYear, tt.wantMonth)
				}
			}
		})
	}
}

// TestDetectFormat_WeeklyPayment tests the format detection for weekly payment files.
func TestDetectFormat_WeeklyPayment(t *testing.T) {
	// Create a minimal weekly payment file
	f := excelize.NewFile()
	sheetName := "Lương 520"
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		t.Fatalf("rename initial sheet: %v", err)
	}

	// Row 8: Headers
	setCellValue(f, sheetName, "A8", "STT")
	setCellValue(f, sheetName, "B8", "Mã nhân viên")
	setCellValue(f, sheetName, "C8", "Họ và tên")
	setCellValue(f, sheetName, "D8", "Bộ phận")
	setCellValue(f, sheetName, "E8", "Lương 8h")

	// Row 10: Shift codes
	setCellValue(f, sheetName, "F10", "HC")
	setCellValue(f, sheetName, "G10", "TCN")

	// Set as visible (default)
	if err := f.SetSheetVisible(sheetName, true); err != nil {
		t.Fatalf("set sheet visible: %v", err)
	}

	result, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat() error = %v", err)
	}

	if result.Format != FormatWeeklyPayment {
		t.Errorf("DetectFormat() = %v, want %v", result.Format, FormatWeeklyPayment)
	}

	if len(result.WeeklyPaymentSheets) != 1 || result.WeeklyPaymentSheets[0] != sheetName {
		t.Errorf("WeeklyPaymentSheets = %v, want [%s]", result.WeeklyPaymentSheets, sheetName)
	}
}

func TestParseWeeklyPaymentFile_UsesSheetNameAsPosition(t *testing.T) {
	f := excelize.NewFile()
	const sheetName = "Lương 520"
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		t.Fatalf("rename initial sheet: %v", err)
	}

	setCellValue(f, sheetName, "F8", "1")
	setCellValue(f, sheetName, "F9", "T7")
	setCellValue(f, sheetName, "F10", "HC")
	setCellValue(f, sheetName, "G10", "TCN")
	setCellValue(f, sheetName, "B11", "012345678901")
	setCellValue(f, sheetName, "C11", "Nguyễn Văn Kiên")
	setCellValue(f, sheetName, "F11", "8")
	setCellValue(f, sheetName, "G11", "2")

	parsed, err := ParseWeeklyPaymentFile(f, []string{sheetName}, "2026-08")
	if err != nil {
		t.Fatalf("ParseWeeklyPaymentFile() error = %v", err)
	}
	if len(parsed.Sheets) != 1 {
		t.Fatalf("expected one sheet, got %d", len(parsed.Sheets))
	}
	if got := parsed.Sheets[0].Position; got != sheetName {
		t.Errorf("sheet position = %q, want %q", got, sheetName)
	}
	entries := parsed.Sheets[0].Employees[0].Entries
	if len(entries) != 2 {
		t.Fatalf("expected two shift entries, got %#v", entries)
	}
	if got := entries[0].ShiftKey; got != "HC" {
		t.Errorf("first shift code = %q, want HC", got)
	}
	if got := entries[1].ShiftKey; got != "TCN" {
		t.Errorf("second shift code = %q, want TCN", got)
	}
}

func TestParseWeeklyPaymentFileKeepsMultipleShiftColumnsOnTheSameDay(t *testing.T) {
	f := excelize.NewFile()
	const sheetName = "Lương 520"
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		t.Fatalf("rename initial sheet: %v", err)
	}

	setCellValue(f, sheetName, "A8", "STT")
	setCellValue(f, sheetName, "B8", "Mã nhân viên")
	setCellValue(f, sheetName, "C8", "Họ và tên")
	setCellValue(f, sheetName, "D8", "Bộ phận")
	setCellValue(f, sheetName, "E8", "Lương 8h")
	setCellValue(f, sheetName, "F8", "3")
	setCellValue(f, sheetName, "H8", "4")
	setCellValue(f, sheetName, "F9", "T2")
	setCellValue(f, sheetName, "H9", "T3")
	if err := f.MergeCell(sheetName, "F8", "G8"); err != nil {
		t.Fatalf("merge day header: %v", err)
	}
	if err := f.MergeCell(sheetName, "F9", "G9"); err != nil {
		t.Fatalf("merge weekday header: %v", err)
	}
	setCellValue(f, sheetName, "F10", "HC")
	setCellValue(f, sheetName, "G10", "TCN")
	setCellValue(f, sheetName, "H10", "HC")
	setCellValue(f, sheetName, "B11", "NV-01")
	setCellValue(f, sheetName, "C11", "Nguyễn Văn An")
	setCellValue(f, sheetName, "F11", "8")
	setCellValue(f, sheetName, "G11", "2")
	setCellValue(f, sheetName, "H11", "8")

	parsed, err := ParseWeeklyPaymentFile(f, []string{sheetName}, "2026-08")
	if err != nil {
		t.Fatalf("ParseWeeklyPaymentFile() error = %v", err)
	}
	entries := parsed.Sheets[0].Employees[0].Entries
	if len(entries) != 3 {
		t.Fatalf("entry count = %d, want 3", len(entries))
	}
	gotDays := []int{entries[0].Day, entries[1].Day, entries[2].Day}
	wantDays := []int{3, 3, 4}
	for i := range wantDays {
		if gotDays[i] != wantDays[i] {
			t.Fatalf("entry %d day = %d, want %d", i, gotDays[i], wantDays[i])
		}
	}
}

func TestParseWeeklyPaymentFileStopsBeforeSummaryShiftColumns(t *testing.T) {
	tests := []struct {
		name     string
		row8     string
		row9     string
		row10    string
		merge    bool
		hideColH bool
	}{
		{name: "row 8 label", row8: "Tổng hợp", row10: "HC"},
		{name: "row 9 label", row9: "Tổng hợp", row10: "HC"},
		{name: "row 10 label", row10: "Tổng hợp"},
		{name: "merged upper label", row8: "Tổng hợp", row10: "HC", merge: true},
		{name: "hidden merged anchor", row8: "Tổng hợp", row10: "HC", merge: true, hideColH: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := excelize.NewFile()
			const sheetName = "Lương 520"
			if err := f.SetSheetName("Sheet1", sheetName); err != nil {
				t.Fatalf("rename initial sheet: %v", err)
			}

			setCellValue(f, sheetName, "F8", "1")
			setCellValue(f, sheetName, "F10", "HC")
			setCellValue(f, sheetName, "G10", "TCN")
			setCellValue(f, sheetName, "H8", tt.row8)
			setCellValue(f, sheetName, "H9", tt.row9)
			setCellValue(f, sheetName, "H10", tt.row10)
			setCellValue(f, sheetName, "I10", "TCN")
			setCellValue(f, sheetName, "B11", "NV-01")
			setCellValue(f, sheetName, "C11", "Nguyễn Văn An")
			setCellValue(f, sheetName, "F11", "8")
			setCellValue(f, sheetName, "G11", "2")
			setCellValue(f, sheetName, "H11", "8")
			setCellValue(f, sheetName, "I11", "2")
			if tt.merge {
				if err := f.MergeCell(sheetName, "H8", "I8"); err != nil {
					t.Fatalf("merge summary header: %v", err)
				}
			}
			if tt.hideColH {
				if err := f.SetColVisible(sheetName, "H", false); err != nil {
					t.Fatalf("hide summary anchor: %v", err)
				}
			}

			parsed, err := ParseWeeklyPaymentFile(f, []string{sheetName}, "2026-08")
			if err != nil {
				t.Fatalf("ParseWeeklyPaymentFile() error = %v", err)
			}
			entries := parsed.Sheets[0].Employees[0].Entries
			if len(entries) != 2 {
				t.Fatalf("entry count = %d, want only 2 attendance entries before the summary block", len(entries))
			}
			if _, ok := parsed.Sheets[0].ShiftRows.ByCol[8]; ok {
				t.Fatal("summary column H was included in the shift map")
			}
			if _, ok := parsed.Sheets[0].ShiftRows.ByCol[9]; ok {
				t.Fatal("summary column I was included in the shift map")
			}
		})
	}
}

func TestParseWeeklyPaymentFileUsesSelectedMonthDespiteStaleWeekdayHeader(t *testing.T) {
	f := excelize.NewFile()
	const sheetName = "Lương 520"
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		t.Fatalf("rename initial sheet: %v", err)
	}

	// The user-selected period is authoritative; legacy templates can retain a
	// weekday label from another month.
	setCellValue(f, sheetName, "F8", "1")
	setCellValue(f, sheetName, "F9", "T4")
	setCellValue(f, sheetName, "F10", "HC")
	setCellValue(f, sheetName, "B11", "NV-01")
	setCellValue(f, sheetName, "F11", "8")

	parsed, err := ParseWeeklyPaymentFile(f, []string{sheetName}, "2026-08")
	if err != nil {
		t.Fatalf("ParseWeeklyPaymentFile() error = %v", err)
	}
	if got := parsed.Sheets[0].Employees[0].Entries[0].Day; got != 1 {
		t.Fatalf("entry day = %d, want 1", got)
	}
}

func TestParseWeeklyPaymentFileRejectsDayOutsideMonth(t *testing.T) {
	f := excelize.NewFile()
	const sheetName = "Lương 520"
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		t.Fatalf("rename initial sheet: %v", err)
	}

	setCellValue(f, sheetName, "F8", "32")
	setCellValue(f, sheetName, "F9", "T2")
	setCellValue(f, sheetName, "F10", "HC")
	setCellValue(f, sheetName, "F11", "8") // real data on an impossible day

	_, err := ParseWeeklyPaymentFile(f, []string{sheetName}, "2026-08")
	if err == nil || !strings.Contains(err.Error(), "nằm ngoài kỳ nhập") {
		t.Fatalf("expected out-of-month error, got %v", err)
	}
}

// Templates ship the full 1..31 day scaffolding with row-10 codes even for
// shorter months: an impossible day over an EMPTY column is scaffolding and
// must be skipped, not fail the sheet (POURIN uploads THÁNG 11/2026 with day
// 31 columns whose data cells are blank).
func TestParseWeeklyPaymentFileSkipsEmptyDayOutsideMonthColumns(t *testing.T) {
	f := excelize.NewFile()
	const sheetName = "Lương 520"
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		t.Fatalf("rename initial sheet: %v", err)
	}

	setCellValue(f, sheetName, "A8", "STT")
	setCellValue(f, sheetName, "B8", "Mã nhân viên")
	setCellValue(f, sheetName, "C8", "Họ và tên")
	setCellValue(f, sheetName, "D8", "Bộ phận")
	setCellValue(f, sheetName, "E8", "Lương 8h")
	setCellValue(f, sheetName, "F8", "30")
	setCellValue(f, sheetName, "H8", "31")
	setCellValue(f, sheetName, "F9", "T5")
	setCellValue(f, sheetName, "H9", "T6")
	setCellValue(f, sheetName, "F10", "HC")
	setCellValue(f, sheetName, "G10", "TC")
	setCellValue(f, sheetName, "H10", "HC")
	setCellValue(f, sheetName, "I10", "TC")
	setCellValue(f, sheetName, "B11", "012345678901")
	setCellValue(f, sheetName, "C11", "Nguyễn Văn Kiên")
	setCellValue(f, sheetName, "F11", "8")
	setCellValue(f, sheetName, "G11", "2")

	parsed, err := ParseWeeklyPaymentFile(f, []string{sheetName}, "2026-11")
	if err != nil {
		t.Fatalf("ParseWeeklyPaymentFile() error = %v", err)
	}
	entries := parsed.Sheets[0].Employees[0].Entries
	if len(entries) != 2 {
		t.Fatalf("entry count = %d, want 2 (day-31 columns skipped)", len(entries))
	}
	if entries[0].Day != 30 || entries[1].Day != 30 {
		t.Fatalf("days = %d,%d, want both on day 30", entries[0].Day, entries[1].Day)
	}
}

// setCellValue is a helper for test cell value setting.
func setCellValue(f *excelize.File, sheet, cell, value string) {
	if err := f.SetCellValue(sheet, cell, value); err != nil {
		panic(err)
	}
}

// TestPositionFromSalaryValue pins the integer-only rendering of the raw
// "Lương 8h" cell: whole numbers become the position name, anything else
// (blank, fractional, text) yields no candidate — no float rounding ever.
func TestPositionFromSalaryValue(t *testing.T) {
	tests := []struct {
		name    string
		rawCell string
		want    string
	}{
		{"integer", "800000", "800000"},
		{"padded", " 900000 ", "900000"},
		{"empty", "", ""},
		{"fractional", "800.5", ""},
		{"text", "abc", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PositionFromSalaryValue(tt.rawCell); got != tt.want {
				t.Errorf("PositionFromSalaryValue(%q) = %q, want %q", tt.rawCell, got, tt.want)
			}
		})
	}
}

// TestParseWeeklyPaymentFileMonthlyLayout pins the POURIN template layout:
// identity columns shifted right by "Ngày đi làm"/"Ngày nghỉ việc" (D/E) with
// Bộ phận at F and the salary header spelled "Vị trí lương/8h" (slash form,
// lower-case) at G, and the TC/TC NN/LE shift vocabulary — no day sub-column
// dropped. The tier cell is stored numeric under the #,##0 display format
// ("800.000"); the raw integer 800000 must win.
func TestParseWeeklyPaymentFileMonthlyLayoutShiftedIdentityColumns(t *testing.T) {
	f := excelize.NewFile()
	const sheetName = "Chấm công"
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		t.Fatalf("rename initial sheet: %v", err)
	}

	setCellValue(f, sheetName, "A8", "STT")
	setCellValue(f, sheetName, "B8", "Mã nhân viên")
	setCellValue(f, sheetName, "C8", "Họ và tên")
	setCellValue(f, sheetName, "D8", "Ngày đi làm")
	setCellValue(f, sheetName, "E8", "Ngày nghỉ việc")
	setCellValue(f, sheetName, "F8", "Bộ phận")
	setCellValue(f, sheetName, "G8", "Vị trí lương/8h")
	setCellValue(f, sheetName, "H8", "1")
	setCellValue(f, sheetName, "I8", "2")
	setCellValue(f, sheetName, "J8", "3")
	setCellValue(f, sheetName, "K8", "4")
	setCellValue(f, sheetName, "H10", "HC")
	setCellValue(f, sheetName, "I10", "TC")
	setCellValue(f, sheetName, "J10", "TC NN")
	setCellValue(f, sheetName, "K10", "LE")

	setCellValue(f, sheetName, "B11", "002093000710")
	setCellValue(f, sheetName, "C11", "Lý Văn Thanh")
	setCellValue(f, sheetName, "F11", "Phân xưởng Trang trí")
	if err := f.SetCellValue(sheetName, "G11", 800000); err != nil {
		t.Fatalf("set tier cell: %v", err)
	}
	numFmt := "#,##0"
	styleID, err := f.NewStyle(&excelize.Style{CustomNumFmt: &numFmt})
	if err != nil {
		t.Fatalf("new style: %v", err)
	}
	if err := f.SetCellStyle(sheetName, "G11", "G11", styleID); err != nil {
		t.Fatalf("set tier format: %v", err)
	}
	setCellValue(f, sheetName, "H11", "8")
	setCellValue(f, sheetName, "I11", "2")
	setCellValue(f, sheetName, "J11", "4")
	setCellValue(f, sheetName, "K11", "6")

	parsed, err := ParseWeeklyPaymentFile(f, []string{sheetName}, "2026-11")
	if err != nil {
		t.Fatalf("ParseWeeklyPaymentFile() error = %v", err)
	}
	if len(parsed.Sheets) != 1 {
		t.Fatalf("expected one sheet, got %d", len(parsed.Sheets))
	}
	shiftCols := parsed.Sheets[0].ShiftRows.ByCol
	wantShiftCols := map[int]string{8: "HC", 9: "TC", 10: "TC NN", 11: "LE"}
	if len(shiftCols) != len(wantShiftCols) {
		t.Fatalf("shift columns = %#v, want %#v", shiftCols, wantShiftCols)
	}
	for col, code := range wantShiftCols {
		if shiftCols[col] != code {
			t.Errorf("shift column %d = %q, want %q", col, shiftCols[col], code)
		}
	}

	employees := parsed.Sheets[0].Employees
	if len(employees) != 1 {
		t.Fatalf("employee count = %d, want 1", len(employees))
	}
	emp := employees[0]
	if emp.Department != "Phân xưởng Trang trí" {
		t.Errorf("department = %q, want the F-column value", emp.Department)
	}
	if emp.SalaryTierRaw != "800000" {
		t.Errorf("salary tier raw = %q, want the raw integer", emp.SalaryTierRaw)
	}
	if emp.SalaryTier != 800000 {
		t.Errorf("salary tier = %d, want 800000", emp.SalaryTier)
	}
	wantEntries := []WeeklyPaymentEntryData{
		{Day: 1, ShiftKey: "HC", Hours: 8},
		{Day: 2, ShiftKey: "TC", Hours: 2},
		{Day: 3, ShiftKey: "TC NN", Hours: 4},
		{Day: 4, ShiftKey: "LE", Hours: 6},
	}
	if len(emp.Entries) != len(wantEntries) {
		t.Fatalf("entries = %#v, want %d", emp.Entries, len(wantEntries))
	}
	for i, want := range wantEntries {
		got := emp.Entries[i]
		if got.Day != want.Day || got.ShiftKey != want.ShiftKey || got.Hours != want.Hours {
			t.Errorf("entry %d = {d%d %s %v}, want {d%d %s %v}", i, got.Day, got.ShiftKey, got.Hours, want.Day, want.ShiftKey, want.Hours)
		}
	}
}

// The POURIN template spells the salary header "Vị trí lương/8h" (slash form,
// lower-case) — the workbook must still classify as the weekly-payment
// family instead of falling out of the registry as an unknown template.
func TestDetectFormatWeeklyPaymentPourinSlashSalaryHeader(t *testing.T) {
	f := excelize.NewFile()
	const sheetName = "Chấm công"
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		t.Fatalf("rename initial sheet: %v", err)
	}

	setCellValue(f, sheetName, "A8", "STT")
	setCellValue(f, sheetName, "B8", "Mã nhân viên")
	setCellValue(f, sheetName, "C8", "Họ và tên")
	setCellValue(f, sheetName, "D8", "Ngày đi làm")
	setCellValue(f, sheetName, "E8", "Ngày nghỉ việc")
	setCellValue(f, sheetName, "F8", "Bộ phận")
	setCellValue(f, sheetName, "G8", "Vị trí lương/8h")
	setCellValue(f, sheetName, "H8", "1")
	setCellValue(f, sheetName, "I8", "2")
	setCellValue(f, sheetName, "H10", "HC")
	setCellValue(f, sheetName, "I10", "TC")

	result, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat() error = %v", err)
	}
	if result.Format != FormatWeeklyPayment {
		t.Fatalf("DetectFormat() = %v, want %v", result.Format, FormatWeeklyPayment)
	}
	if len(result.WeeklyPaymentSheets) != 1 || result.WeeklyPaymentSheets[0] != sheetName {
		t.Errorf("WeeklyPaymentSheets = %v, want [%s]", result.WeeklyPaymentSheets, sheetName)
	}
}
