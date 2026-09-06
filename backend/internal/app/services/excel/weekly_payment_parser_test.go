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

	_, err := ParseWeeklyPaymentFile(f, []string{sheetName}, "2026-08")
	if err == nil || !strings.Contains(err.Error(), "nằm ngoài kỳ nhập") {
		t.Fatalf("expected out-of-month error, got %v", err)
	}
}

// setCellValue is a helper for test cell value setting.
func setCellValue(f *excelize.File, sheet, cell, value string) {
	if err := f.SetCellValue(sheet, cell, value); err != nil {
		panic(err)
	}
}
