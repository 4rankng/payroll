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
	if len(data.Employees) != 26 {
		t.Errorf("employees = %d, want 26", len(data.Employees))
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
	// This settlement-cutoff file's worked data ends 15/09: the Sep 16–24
	// day-number continuation columns exist structurally but hold zero hours,
	// so the entry count matches the full-date region alone.
	if total != 288 {
		t.Errorf("total entries = %d, want 288", total)
	}
	// The day map itself must span the full grid — 26 full-date days (21/08→
	// 15/09, cols 9–60) plus the 9-day day-number continuation (16–24/09,
	// cols 61–78). A file with hours in the continuation columns must not
	// silently lose them.
	_, dateCols, derr2 := findDateRowAndCols(f, "M1")
	if derr2 != nil {
		t.Fatalf("findDateRowAndCols: %v", derr2)
	}
	lastCol, lastDate := 0, time.Time{}
	for c, d := range dateCols {
		if c > lastCol {
			lastCol, lastDate = c, d
		}
	}
	if lastCol != 78 || lastDate.Format("2006-01-02") != "2026-09-24" {
		t.Errorf("day grid ends at col %d %s, want col 78 2026-09-24", lastCol, lastDate.Format("2006-01-02"))
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

// buildContinuationWorkbook replicates the wide real-template shape: three
// full-date anchor days, a day-number continuation day, a summary column
// right of the day region, a blank spacer row mid-roster, a footnote row
// after trailing blanks, and an SĐT phone column.
func buildContinuationWorkbook(t *testing.T) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	ws := "M1"
	if err := f.SetSheetName("Sheet1", ws); err != nil {
		t.Fatalf("rename sheet: %v", err)
	}
	for i, h := range []string{"STT", "Mã NV", "Họ và tên", "TK Ngân hàng", "Ngân hàng", "SĐT"} {
		_ = f.SetCellValue(ws, cellName(i+1, 5), h)
	}
	_ = f.SetCellValue(ws, cellName(9, 6), time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC))
	_ = f.SetCellValue(ws, cellName(11, 6), time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC))
	_ = f.SetCellValue(ws, cellName(13, 6), time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC))
	// Day-number continuation: 24 repeated across the day's two sub-columns.
	_ = f.SetCellValue(ws, cellName(15, 6), 24)
	_ = f.SetCellValue(ws, cellName(16, 6), 24)
	for col, code := range map[int]string{
		9: "NT", 10: "OT", 11: "NT", 12: "OT", 13: "NT", 14: "OT", 15: "NT", 16: "OT",
	} {
		_ = f.SetCellValue(ws, cellName(col, 8), code)
	}
	// Summary column beyond the day region: header + a money-like total that
	// must never be ingested as hours.
	_ = f.SetCellValue(ws, cellName(17, 5), "Tổng giờ")
	_ = f.SetCellValue(ws, cellName(17, 8), "Tổng giờ")
	_ = f.SetCellValue(ws, cellName(17, 9), 36.0)

	_ = f.SetCellValue(ws, "A9", 1)
	_ = f.SetCellValue(ws, "B9", "031082006094")
	_ = f.SetCellValue(ws, "C9", "Khổng Văn Tiến")
	_ = f.SetCellValue(ws, "F9", "0900123456")
	_ = f.SetCellValue(ws, cellName(9, 9), 9.0)  // NT 21/08
	_ = f.SetCellValue(ws, cellName(15, 9), 8.0) // NT 24/08 via continuation
	// Row 10 blank spacer, row 11 a second active employee.
	_ = f.SetCellValue(ws, "A11", 2)
	_ = f.SetCellValue(ws, "B11", "031076026147")
	_ = f.SetCellValue(ws, "C11", "Nguyễn Anh Tuấn")
	_ = f.SetCellValue(ws, cellName(10, 11), 2.0) // OT 21/08
	_ = f.SetCellValue(ws, cellName(16, 11), 1.0) // OT 24/08 via continuation
	// Rows 12 blank, 13 a footnote — the roster must end, not ingest "* Chú ý:".
	_ = f.SetCellValue(ws, "B13", "* Chú ý:")
	return f
}

func TestParseDateRowBCCFile_ContinuationAndSummary(t *testing.T) {
	f := buildContinuationWorkbook(t)
	defer func() { _ = f.Close() }()

	data, err := ParseDateRowBCCFile(f, []string{"M1"})
	if err != nil {
		t.Fatalf("ParseDateRowBCCFile: %v", err)
	}
	if len(data.Employees) != 2 {
		t.Fatalf("employees = %d, want 2 (spacer crossed, footnote dropped): %+v", len(data.Employees), data.Employees)
	}
	first := data.Employees[0]
	if first.Mobile != "0900123456" {
		t.Errorf("Mobile = %q, want captured from SĐT column", first.Mobile)
	}
	want := []struct {
		label string
		hours float64
		date  string
	}{
		{"NT", 9, "2026-08-21"},
		{"NT", 8, "2026-08-24"}, // day-number continuation dated correctly
	}
	if len(first.Entries) != len(want) {
		t.Fatalf("first entries = %d, want %d (%+v)", len(first.Entries), len(want), first.Entries)
	}
	for i, w := range want {
		got := first.Entries[i]
		if got.ShiftLabel != w.label || got.Hours != w.hours || got.FullDate == nil || got.FullDate.Format("2006-01-02") != w.date {
			t.Errorf("entry %d = {%s %.1f %v}, want {%s %.1f %s}", i, got.ShiftLabel, got.Hours, got.FullDate, w.label, w.hours, w.date)
		}
	}
	second := data.Employees[1]
	if len(second.Entries) != 2 || second.Entries[0].ShiftLabel != "OT" || second.Entries[1].FullDate.Format("2006-01-02") != "2026-08-24" {
		t.Errorf("second entries unexpected: %+v", second.Entries)
	}
	for _, emp := range data.Employees {
		for _, en := range emp.Entries {
			if en.Hours == 36 {
				t.Errorf("summary column ingested as hours: %+v", en)
			}
		}
	}
}

func TestParseDateRowBCCFile_ContinuationStopsOnMismatch(t *testing.T) {
	f := buildContinuationWorkbook(t)
	defer func() { _ = f.Close() }()
	// Overwrite the continuation with a day number that skips a day: after
	// 23/08 the sequence expects 24, so "26" must stop the continuation and
	// col 15's hours are dropped — never attributed to a wrong date.
	_ = f.SetCellValue("M1", cellName(15, 6), 26)
	_ = f.SetCellValue("M1", cellName(16, 6), 26)

	data, err := ParseDateRowBCCFile(f, []string{"M1"})
	if err != nil {
		t.Fatalf("ParseDateRowBCCFile: %v", err)
	}
	first := data.Employees[0]
	if len(first.Entries) != 1 || first.Entries[0].FullDate.Format("2006-01-02") != "2026-08-21" {
		t.Errorf("first entries = %+v, want only the 21/08 anchor entry", first.Entries)
	}
}

func TestParseDateRowBCCFile_BlankAnchorDayNotInherited(t *testing.T) {
	f := buildContinuationWorkbook(t)
	defer func() { _ = f.Close() }()
	// Remove the continuation numbers: col 15 has a code and hours but no
	// anchor for its day — the entry must be dropped, not dated 23/08.
	_ = f.SetCellValue("M1", cellName(15, 6), nil)
	_ = f.SetCellValue("M1", cellName(16, 6), nil)

	data, err := ParseDateRowBCCFile(f, []string{"M1"})
	if err != nil {
		t.Fatalf("ParseDateRowBCCFile: %v", err)
	}
	first := data.Employees[0]
	if len(first.Entries) != 1 || first.Entries[0].FullDate.Format("2006-01-02") != "2026-08-21" {
		t.Errorf("first entries = %+v, want only the anchored day (no cross-day inheritance)", first.Entries)
	}
	second := data.Employees[1]
	if len(second.Entries) != 1 || second.Entries[0].FullDate.Format("2006-01-02") != "2026-08-21" {
		t.Errorf("second entries = %+v, want only the anchored day", second.Entries)
	}
}

func TestFindDateRowAndCols_StrayDatesAboveRealHeader(t *testing.T) {
	f := buildContinuationWorkbook(t)
	defer func() { _ = f.Close() }()
	// Three stray full dates in row 3 with no shift-code row below: the
	// detector must not stop there, and neither must the parser.
	for _, col := range []int{5, 7, 9} {
		_ = f.SetCellValue("M1", cellName(col, 3), time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC))
	}

	row, _, err := findDateRowAndCols(f, "M1")
	if err != nil {
		t.Fatalf("findDateRowAndCols: %v", err)
	}
	if row != 6 {
		t.Errorf("dateRow = %d, want 6 (stray row 3 rejected)", row)
	}
	res, derr := DetectFormat(f)
	if derr != nil || res.Format != FormatDateRow {
		t.Fatalf("DetectFormat = %+v err=%v, want FormatDateRow", res, derr)
	}
	if _, perr := ParseDateRowBCCFile(f, res.DateRowSheets); perr != nil {
		t.Errorf("parse after detection: %v", perr)
	}
}

func TestFindDateRowEmployeeCols_WideHeaderWindow(t *testing.T) {
	f := excelize.NewFile()
	ws := "M1"
	if err := f.SetSheetName("Sheet1", ws); err != nil {
		t.Fatalf("rename sheet: %v", err)
	}
	// Headers six rows above a row-8 date row — deeper header stacks must
	// still resolve the bank columns.
	for i, h := range []string{"STT", "Mã NV", "Họ và tên", "TK Ngân hàng"} {
		_ = f.SetCellValue(ws, cellName(i+1, 2), h)
	}
	_ = f.SetCellValue(ws, cellName(9, 8), time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC))
	_ = f.SetCellValue(ws, cellName(11, 8), time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC))
	_ = f.SetCellValue(ws, cellName(13, 8), time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC))
	_ = f.SetCellValue(ws, cellName(9, 9), "NT")
	_ = f.SetCellValue(ws, cellName(11, 9), "OT")

	hm := findDateRowEmployeeCols(f, ws, 8)
	if hm.bankAccountCol != 4 || hm.cccdCol != 2 || hm.nameCol != 3 {
		t.Errorf("empCols = %+v, want bankAccount=4 cccd=2 name=3", hm)
	}

	// A template without bank columns is legitimate: fields stay empty.
	f2 := excelize.NewFile()
	ws2 := "M1"
	if err := f2.SetSheetName("Sheet1", ws2); err != nil {
		t.Fatalf("rename sheet: %v", err)
	}
	for i, h := range []string{"STT", "Mã NV", "Họ và tên"} {
		_ = f2.SetCellValue(ws2, cellName(i+1, 5), h)
	}
	_ = f2.SetCellValue(ws2, cellName(9, 6), time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC))
	_ = f2.SetCellValue(ws2, cellName(11, 6), time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC))
	_ = f2.SetCellValue(ws2, cellName(13, 6), time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC))
	_ = f2.SetCellValue(ws2, cellName(9, 7), "NT")
	_ = f2.SetCellValue(ws2, cellName(11, 7), "OT")
	_ = f2.SetCellValue(ws2, "B8", "031082006094")
	_ = f2.SetCellValue(ws2, "C8", "Khổng Văn Tiến")
	_ = f2.SetCellValue(ws2, cellName(9, 8), 9.0)
	data, perr := ParseDateRowBCCFile(f2, []string{ws2})
	if perr != nil {
		t.Fatalf("ParseDateRowBCCFile without bank cols: %v", perr)
	}
	if len(data.Employees) != 1 || data.Employees[0].BankAccount != "" || data.Employees[0].BankName != "" {
		t.Errorf("bankless template parse = %+v, want 1 employee with empty bank fields", data.Employees)
	}
}
