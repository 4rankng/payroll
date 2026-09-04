package excel

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// buildBCCNamedWorkbook replicates the BUMHAN T09 template: employee headers
// in row 5, full-date serials in row 6 (one date per two-column day group),
// day/weekday row 7, shift codes row 8, employees from row 9. The sheet name
// and the "Vi tri" column are parameterized: the column's index drifts with
// the month's day count, so tests pin the name-based lookup at several
// indexes and header spellings.
func buildBCCNamedWorkbook(t *testing.T, sheet, positionHeader string, positionCol int) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		t.Fatalf("rename sheet: %v", err)
	}

	for i, h := range []string{"STT", "Mã NV", "Họ và tên", "TK Ngân hàng", "Ngân hàng"} {
		_ = f.SetCellValue(sheet, cellName(i+1, 5), h)
	}
	if positionHeader != "" {
		_ = f.SetCellValue(sheet, cellName(positionCol, 5), positionHeader)
		_ = f.SetCellValue(sheet, cellName(positionCol, 9), "Thợ phụ")
	}
	for col, d := range map[int]time.Time{
		9:  time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC),
		11: time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC),
		13: time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC),
	} {
		_ = f.SetCellValue(sheet, cellName(col, 6), d)
	}
	_ = f.SetCellValue(sheet, "I7", "6")
	_ = f.SetCellValue(sheet, "K7", "7")
	_ = f.SetCellValue(sheet, "M7", "CN")
	for col, code := range map[int]string{
		9: "NT", 10: "OT", 11: "T7", 12: "OT T7", 13: "CN", 14: "OT CN",
	} {
		_ = f.SetCellValue(sheet, cellName(col, 8), code)
	}
	_ = f.SetCellValue(sheet, "A9", 1)
	_ = f.SetCellValue(sheet, "B9", "031082006094")
	_ = f.SetCellValue(sheet, "C9", "Khổng Văn Tiến")
	_ = f.SetCellValue(sheet, "I9", 9.0)
	_ = f.SetCellValue(sheet, "J9", 1.0)
	_ = f.SetCellValue(sheet, "M9", 8.0)
	_ = f.SetCellValue(sheet, "A10", 2)
	_ = f.SetCellValue(sheet, "B10", "031076026147")
	_ = f.SetCellValue(sheet, "C10", "Nguyễn Anh Tuấn")
	_ = f.SetCellValue(sheet, "I10", 9.0)
	_ = f.SetCellValue(sheet, "K10", 4.5)
	return f
}

// buildLegacyWorkbook replicates the original BCC format: STT/CCCD/name
// headers in row 7, day numbers in row 8, VND rate row 9, shift-label row 10,
// employees from row 11.
func buildLegacyWorkbook(t *testing.T) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", "BCC"); err != nil {
		t.Fatalf("rename sheet: %v", err)
	}
	_ = f.SetCellValue("BCC", "A7", "STT")
	_ = f.SetCellValue("BCC", "B7", "CCCD")
	_ = f.SetCellValue("BCC", "C7", "Họ và tên")
	for col, day := range map[int]int{9: 1, 10: 2, 11: 3} {
		_ = f.SetCellValue("BCC", cellName(col, 8), day)
	}
	_ = f.SetCellValue("BCC", "I9", 36000)
	_ = f.SetCellValue("BCC", "J9", 45000)
	_ = f.SetCellValue("BCC", "I10", "CB")
	_ = f.SetCellValue("BCC", "J10", "OT")
	_ = f.SetCellValue("BCC", "A11", 1)
	_ = f.SetCellValue("BCC", "B11", "031082006094")
	_ = f.SetCellValue("BCC", "C11", "Khổng Văn Tiến")
	_ = f.SetCellValue("BCC", "I11", 8.0)
	_ = f.SetCellValue("BCC", "J11", 1.5)
	return f
}

func TestParseBCCData_LegacyNamedDateRow(t *testing.T) {
	f := buildBCCNamedWorkbook(t, "BCC", "Vị trí", 102)
	defer func() { _ = f.Close() }()

	// DetectFormat keeps its legacy tiebreak by sheet name — the strategy
	// router is what recovers the date-row layout from under the legacy name.
	res, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat: %v", err)
	}
	if res.Format != FormatLegacy {
		t.Fatalf("DetectFormat format = %v, want FormatLegacy (name tiebreak preserved)", res.Format)
	}

	data, format, err := ParseBCCData(f)
	if err != nil {
		t.Fatalf("ParseBCCData: %v", err)
	}
	if format != FormatDateRow {
		t.Fatalf("format = %v, want FormatDateRow", format)
	}
	if len(data.Employees) != 2 {
		t.Fatalf("employees = %d, want 2", len(data.Employees))
	}
	if data.Employees[0].Position != "Thợ phụ" {
		t.Errorf("Position = %q, want Thợ phụ", data.Employees[0].Position)
	}
	if len(data.Employees[0].Entries) == 0 {
		t.Errorf("first employee entries empty")
	}
}

func TestParseBCCData_PositionColumnDriftsByMonthLength(t *testing.T) {
	f := buildBCCNamedWorkbook(t, "M1", "Vị trí", 100)
	defer func() { _ = f.Close() }()

	data, format, err := ParseBCCData(f)
	if err != nil {
		t.Fatalf("ParseBCCData: %v", err)
	}
	if format != FormatDateRow {
		t.Fatalf("format = %v, want FormatDateRow", format)
	}
	// A 30-day month shifts the column; only the header name may matter.
	if data.Employees[0].Position != "Thợ phụ" {
		t.Errorf("Position = %q, want Thợ phụ", data.Employees[0].Position)
	}
}

func TestParseBCCData_PositionHeaderWithoutDiacritics(t *testing.T) {
	f := buildBCCNamedWorkbook(t, "M1", "Vi tri", 98)
	defer func() { _ = f.Close() }()

	data, _, err := ParseBCCData(f)
	if err != nil {
		t.Fatalf("ParseBCCData: %v", err)
	}
	if data.Employees[0].Position != "Thợ phụ" {
		t.Errorf("Position = %q, want Thợ phụ (no-diacritics header)", data.Employees[0].Position)
	}
}

func TestParseBCCData_NoPositionColumn(t *testing.T) {
	f := buildBCCNamedWorkbook(t, "M1", "", 0)
	defer func() { _ = f.Close() }()

	data, _, err := ParseBCCData(f)
	if err != nil {
		t.Fatalf("ParseBCCData: %v", err)
	}
	if data.Employees[0].Position != "" {
		t.Errorf("Position = %q, want empty without the column", data.Employees[0].Position)
	}
}

func TestParseBCCData_LegacyStaysLegacy(t *testing.T) {
	f := buildLegacyWorkbook(t)
	defer func() { _ = f.Close() }()

	data, format, err := ParseBCCData(f)
	if err != nil {
		t.Fatalf("ParseBCCData: %v", err)
	}
	if format != FormatLegacy {
		t.Fatalf("format = %v, want FormatLegacy", format)
	}
	if len(data.ShiftRates) == 0 {
		t.Fatalf("rates empty, want legacy rates")
	}
	if len(data.Employees) == 0 || len(data.Employees[0].Entries) == 0 {
		t.Fatalf("employees/entries empty: %+v", data.Employees)
	}
}

func TestParseBCCData_Unrecognized(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	_, _, err := ParseBCCData(f)
	if err == nil {
		t.Fatalf("ParseBCCData on empty workbook = nil error, want failure")
	}
}

// TestParseBCCData_RealFileT09 guards the real T09 partner template: a
// date-row layout under the legacy "BCC" sheet name plus a drifting "Vi tri"
// column. Skips when the file is absent so CI stays runnable.
func TestParseBCCData_RealFileT09(t *testing.T) {
	path := "/Users/dev/Downloads/BCC BUMHAN T09.2026 Mẫu mức lương.xlsx"
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
	if res.Format != FormatLegacy {
		t.Fatalf("DetectFormat format = %v, want FormatLegacy", res.Format)
	}

	data, format, err := ParseBCCData(f)
	if err != nil {
		t.Fatalf("ParseBCCData: %v", err)
	}
	if format != FormatDateRow {
		t.Fatalf("format = %v, want FormatDateRow", format)
	}
	if len(data.Employees) != 17 {
		t.Fatalf("employees = %d, want 17", len(data.Employees))
	}
	hoc := data.Employees[2]
	// The partner encodes position as the rate tier in the "Vị trí" column
	// (600/700, a thousands-scaled display of 600k/700k per 9h). Faithfully
	// transport the displayed value; pricing stays keyed to payrate config.
	if hoc.FullName != "Phạm Hữu Học" || hoc.Position != "600" {
		t.Errorf("employee 3 = %s / %q, want Phạm Hữu Học / 600", hoc.FullName, hoc.Position)
	}
	if data.Employees[3].Position != "700" {
		t.Errorf("employee 4 Position = %q, want 700", data.Employees[3].Position)
	}
	total := 0
	labels := map[string]bool{}
	for _, emp := range data.Employees {
		for _, en := range emp.Entries {
			total++
			labels[en.ShiftLabel] = true
		}
	}
	if total == 0 {
		t.Fatalf("no entries parsed")
	}
	// Sep 1-2 are LE-coded holiday columns, but nobody worked them — the LE
	// labels must NOT appear among entries (zero-hour cells are skipped).
	if labels["LE"] || labels["OT LE"] {
		t.Errorf("unexpected holiday entries: %v", labels)
	}
}
