package excel

import (
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
	"github.com/xuri/excelize/v2/opt"
)

// TestIsNumericSheetName tests the numeric sheet name detector.
func TestIsNumericSheetName(t *testing.T) {
	tests := []struct {
		name     string
		sheetName string
		want     bool
	}{
		{"numeric sheet", "520", true},
		{"numeric sheet", "700", true},
		{"numeric sheet", "750", true},
		{"non-numeric", "BCC", false},
		{"non-numeric", "STK", false},
		{"non-numeric", "Phổ thông", false},
		{"empty", "", false},
		{"whitespace", "  ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNumericSheetName(tt.sheetName); got != tt.want {
				t.Errorf("isNumericSheetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

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

// TestRateKeyFor tests the rate key builder.
func TestRateKeyFor(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		shiftCode string
		want      string
	}{
		{"520HC", "520", "HC", "520HC"},
		{"700TCN", "700", "TCN", "700TCN"},
		{"750NN", "750", "NN", "750NN"},
		{"800TCNN", "800", "TCNN", "800TCNN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RateKeyFor(tt.prefix, tt.shiftCode); got != tt.want {
				t.Errorf("RateKeyFor() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDayTypeFor tests the day type resolver.
func TestDayTypeFor(t *testing.T) {
	tests := []struct {
		name      string
		shiftCode string
		want      string
	}{
		{"weekday shift", "HC", "ngày thường"},
		{"weekday OT", "TCN", "ngày thường"},
		{"weekend shift", "NN", "ngày nghỉ"},
		{"weekend OT", "TCNN", "ngày nghỉ"},
		{"unknown", "XYZ", "ngày thường"},
		{"lowercase", "hc", "ngày thường"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DayTypeFor(tt.shiftCode); got != tt.want {
				t.Errorf("DayTypeFor() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseForMonth tests the month string parser.
func TestParseForMonth(t *testing.T) {
	tests := []struct {
		name     string
		forMonth string
		wantYear int
		wantMonth time.Month
		wantErr  bool
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

// TestBuildWeekdayToDayMap tests the weekday to day mapping.
func TestBuildWeekdayToDayMap(t *testing.T) {
	// July 2026: July 1, 2026 is Wednesday (T4)
	// So T4 should map to day 1, T5 to day 2, ..., CN to day 7, T2 to day 8, etc.
	result := buildWeekdayToDayMap(2026, time.July, "T4")

	expected := map[string]int{
		"T4": 1, // Wednesday, July 1
		"T5": 2, // Thursday
		"T6": 3, // Friday
		"T7": 4, // Saturday
		"CN": 5, // Sunday
		"T2": 6, // Monday (July 6)
		"T3": 7, // Tuesday
	}

	for wd, wantDay := range expected {
		if gotDay, ok := result[wd]; !ok {
			t.Errorf("buildWeekdayToDayMap() missing weekday %s", wd)
		} else if gotDay != wantDay {
			t.Errorf("buildWeekdayToDayMap() for %s = %v, want %v", wd, gotDay, wantDay)
		}
	}
}

// TestDetectFormat_WeeklyPayment tests the format detection for weekly payment files.
func TestDetectFormat_WeeklyPayment(t *testing.T) {
	// Create a minimal weekly payment file
	f := excelize.NewFile()
	sheetName := "520"

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
	f.SetSheetVisible(sheetName, true)

	// Delete default Sheet1
	f.DeleteSheet("Sheet1")

	result, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat() error = %v", err)
	}

	if result.Format != FormatWeeklyPayment {
		t.Errorf("DetectFormat() = %v, want %v", result.Format, FormatWeeklyPayment)
	}

	if len(result.WeeklyPaymentSheets) != 1 || result.WeeklyPaymentSheets[0] != "520" {
		t.Errorf("WeeklyPaymentSheets = %v, want [520]", result.WeeklyPaymentSheets)
	}
}

// setCellValue is a helper for test cell value setting.
func setCellValue(f *excelize.File, sheet, cell, value string) {
	col, row, _ := excelize.CellNameToCoordinates(cell)
	f.SetCellValue(sheet, row, col, value, opt.NoCellTypeErr())
}
