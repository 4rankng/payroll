package excel

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// buildM1StyleWorkbook replicates the BUMHAN partner layout: employee headers
// in row 5, full-date serials in row 6 (one date cell per two-column day
// group), weekday numbers in row 7, shift codes in row 8, data from row 9.
func buildM1StyleWorkbook(t *testing.T) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	ws := "M1"
	if err := f.SetSheetName("Sheet1", ws); err != nil {
		t.Fatalf("rename sheet: %v", err)
	}

	for i, h := range []string{"STT", "Mã NV", "Họ và tên", "TK Ngân hàng"} {
		_ = f.SetCellValue(ws, cellName(i+1, 5), h)
	}
	for col, d := range map[int]time.Time{
		9:  time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC),
		11: time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC),
		13: time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC),
	} {
		_ = f.SetCellValue(ws, cellName(col, 6), d)
	}
	_ = f.SetCellValue(ws, "I7", "6")
	_ = f.SetCellValue(ws, "K7", "7")
	_ = f.SetCellValue(ws, "M7", "CN")
	_ = f.SetCellValue(ws, "I8", "NT")
	_ = f.SetCellValue(ws, "J8", "OT")
	_ = f.SetCellValue(ws, "K8", "T7")
	_ = f.SetCellValue(ws, "L8", "OT T7")
	_ = f.SetCellValue(ws, "M8", "CN")
	_ = f.SetCellValue(ws, "N8", "OT CN")
	_ = f.SetCellValue(ws, "A9", 1)
	_ = f.SetCellValue(ws, "B9", "031082006094")
	_ = f.SetCellValue(ws, "C9", "Khổng Văn Tiến")
	_ = f.SetCellValue(ws, "D9", "Thợ phụ")
	_ = f.SetCellValue(ws, "I9", 9.0)
	_ = f.SetCellValue(ws, "J9", 1.0)
	_ = f.SetCellValue(ws, "M9", 8.0)
	_ = f.SetCellValue(ws, "A10", 2)
	_ = f.SetCellValue(ws, "B10", "031076026147")
	_ = f.SetCellValue(ws, "C10", "Nguyễn Anh Tuấn")
	_ = f.SetCellValue(ws, "I10", 9.0)
	_ = f.SetCellValue(ws, "J10", 1.0)
	_ = f.SetCellValue(ws, "K10", 4.5)
	return f
}

func cellName(col, row int) string {
	n, _ := excelize.CoordinatesToCellName(col, row)
	return n
}

func TestDetectFormat_DateRowBCC(t *testing.T) {
	f := buildM1StyleWorkbook(t)
	defer func() { _ = f.Close() }()

	res, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat error: %v", err)
	}
	if res.Format != FormatDateRow {
		t.Fatalf("format = %v, want FormatDateRow", res.Format)
	}
	if len(res.DateRowSheets) != 1 || res.DateRowSheets[0] != "M1" {
		t.Fatalf("DateRowSheets = %v, want [M1]", res.DateRowSheets)
	}
}

func TestParseDateRowBCCFile(t *testing.T) {
	f := buildM1StyleWorkbook(t)
	defer func() { _ = f.Close() }()

	data, err := ParseDateRowBCCFile(f, []string{"M1"})
	if err != nil {
		t.Fatalf("ParseDateRowBCCFile error: %v", err)
	}
	if len(data.Employees) != 2 {
		t.Fatalf("employees = %d, want 2", len(data.Employees))
	}
	first := data.Employees[0]
	if first.CCCD != "031082006094" || first.FullName != "Khổng Văn Tiến" {
		t.Fatalf("first employee mismatch: %+v", first)
	}
	want := []struct {
		day   int
		label string
		hours float64
		date  time.Time
	}{
		{21, "NT", 9, time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)},
		{21, "OT", 1, time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)},
		{23, "CN", 8, time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC)},
	}
	if len(first.Entries) != len(want) {
		t.Fatalf("entries = %d, want %d (%+v)", len(first.Entries), len(want), first.Entries)
	}
	for i, w := range want {
		got := first.Entries[i]
		if got.DayNum != w.day || got.ShiftLabel != w.label || got.Hours != w.hours {
			t.Errorf("entry %d = {%d %s %.1f}, want {%d %s %.1f}", i, got.DayNum, got.ShiftLabel, got.Hours, w.day, w.label, w.hours)
		}
		if got.FullDate == nil || !got.FullDate.Equal(w.date) {
			t.Errorf("entry %d FullDate = %v, want %v", i, got.FullDate, w.date)
		}
	}
}

// TestParseDateRowBCCFile_RealFile guards the real BUMHAN partner template.
// Skips when the file is absent so CI stays runnable.
func TestParseDateRowBCCFile_RealFile(t *testing.T) {
	path := "/Users/dev/Downloads/BCC BUMHAN T09.2026 thợ phụ chốt ứng lương - Copy.xlsx"
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("real file not available: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	res, derr := DetectFormat(f)
	if derr != nil {
		t.Fatalf("DetectFormat: %v", derr)
	}
	if res.Format != FormatDateRow {
		t.Fatalf("format = %v, want FormatDateRow", res.Format)
	}
	data, perr := ParseDateRowBCCFile(f, res.DateRowSheets)
	if perr != nil {
		t.Fatalf("parse: %v", perr)
	}
	if len(data.Employees) != 27 {
		t.Errorf("employees = %d, want 27", len(data.Employees))
	}
	first := data.Employees[0]
	if first.CCCD != "031082006094" || first.FullName != "Khổng Văn Tiến" {
		t.Errorf("first employee = %s / %s", first.CCCD, first.FullName)
	}
	if first.BankAccount == "" || first.BankName == "" {
		t.Errorf("bank info not captured: account=%q bank=%q", first.BankAccount, first.BankName)
	}
	total := 0
	for _, emp := range data.Employees {
		total += len(emp.Entries)
	}
	if total != 301 {
		t.Errorf("total entries = %d, want 301", total)
	}
	// Spot-check Aug 22 (Saturday): T7 9h + OT T7 1h under FullDate 2026-08-22.
	var t7h, otT7h float64
	for _, en := range first.Entries {
		if en.FullDate != nil && en.FullDate.Format("2006-01-02") == "2026-08-22" {
			switch en.ShiftLabel {
			case "T7":
				t7h = en.Hours
			case "OT T7":
				otT7h = en.Hours
			}
		}
	}
	if t7h != 9 || otT7h != 1 {
		t.Errorf("Aug 22: T7=%.1f OT T7=%.1f, want 9 / 1", t7h, otT7h)
	}
}
