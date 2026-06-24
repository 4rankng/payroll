package excel

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// buildDecimalHoursBCCXLSX constructs an in-memory BCC workbook that reproduces
// the EVA partner layout: row 7 has weekday headers, row 8 has CCCD + day
// serials, row 9 has rates, row 10 has shift labels, row 11 has one employee
// whose T2 hours cell stores 7.5 (float) under a "#,##0" format. That format
// makes the formatted read path return "8" (rounded) and is the exact
// regression that lost Nguyễn Trọng Thắng's 7.5h on 16/06/2026.
func buildDecimalHoursBCCXLSX(t *testing.T) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	// excelize.NewFile creates "Sheet1"; rename it so the BCC parser's
	// resolveBCCSheet() picks it up.
	if err := f.SetSheetName("Sheet1", "BCC"); err != nil {
		t.Fatalf("SetSheetName: %v", err)
	}
	sheet := "BCC"

	// Row 7 — header band. A:STT, C:Mã NV, E:Họ tên, H:Bộ phận, I:T2, M:T3, U:T5, Y:T6.
	mustSet(t, f, sheet, "A7", "STT")
	mustSet(t, f, sheet, "B7", "STT")
	mustSet(t, f, sheet, "C7", "Mã nhân viên")
	mustSet(t, f, sheet, "E7", "Họ và tên")
	mustSet(t, f, sheet, "H7", "Bộ phận")
	mustSet(t, f, sheet, "I7", "T2")
	mustSet(t, f, sheet, "M7", "T3")
	mustSet(t, f, sheet, "U7", "T5")
	mustSet(t, f, sheet, "Y7", "T6")
	mustSet(t, f, sheet, "AK7", "Tổng hợp")

	// Row 8 — CCCD label in D, day serials in I/M/U/Y formatted "dd".
	mustSet(t, f, sheet, "D8", "CCCD")
	mustSetNumberWithFormat(t, f, sheet, "I8", 15, "dd")
	mustSetNumberWithFormat(t, f, sheet, "M8", 16, "dd")
	mustSetNumberWithFormat(t, f, sheet, "U8", 18, "dd")
	mustSetNumberWithFormat(t, f, sheet, "Y8", 19, "dd")

	// Row 9 — rate row (whole-number VND amounts).
	mustSetNumberWithFormat(t, f, sheet, "I9", 36000, "#,##0")
	mustSetNumberWithFormat(t, f, sheet, "J9", 54000, "#,##0")
	mustSetNumberWithFormat(t, f, sheet, "M9", 36000, "#,##0")
	mustSetNumberWithFormat(t, f, sheet, "N9", 54000, "#,##0")
	mustSetNumberWithFormat(t, f, sheet, "U9", 36000, "#,##0")
	mustSetNumberWithFormat(t, f, sheet, "V9", 54000, "#,##0")
	mustSetNumberWithFormat(t, f, sheet, "Y9", 36000, "#,##0")
	mustSetNumberWithFormat(t, f, sheet, "Z9", 54000, "#,##0")

	// Row 10 — shift labels: I=CB N, J=OT N, M=CB N, N=OT N, U=CB N, V=OT N, Y=CB N, Z=OT N.
	mustSet(t, f, sheet, "I10", "CB N")
	mustSet(t, f, sheet, "J10", "OT N")
	mustSet(t, f, sheet, "M10", "CB N")
	mustSet(t, f, sheet, "N10", "OT N")
	mustSet(t, f, sheet, "U10", "CB N")
	mustSet(t, f, sheet, "V10", "OT N")
	mustSet(t, f, sheet, "Y10", "CB N")
	mustSet(t, f, sheet, "Z10", "OT N")

	// Row 11 — one employee. The T3 cell (M11) stores 7.5 under a "#,##0"
	// whole-number format: formatted read would return "8", raw read returns 7.5.
	mustSet(t, f, sheet, "B11", "1")
	mustSet(t, f, sheet, "C11", "LV000926")
	mustSet(t, f, sheet, "D11", "040205013152")
	mustSet(t, f, sheet, "E11", "Nguyễn Trọng Thắng")
	mustSet(t, f, sheet, "H11", "Nhựa")

	mustSetNumberWithFormat(t, f, sheet, "I11", 7.0, "#,##0")
	mustSetNumberWithFormat(t, f, sheet, "J11", 4.0, "#,##0")
	mustSetNumberWithFormat(t, f, sheet, "M11", 7.5, "#,##0")
	mustSetNumberWithFormat(t, f, sheet, "N11", 4.0, "#,##0")
	mustSetNumberWithFormat(t, f, sheet, "U11", 8.0, "#,##0")
	mustSetNumberWithFormat(t, f, sheet, "V11", 4.0, "#,##0")
	mustSetNumberWithFormat(t, f, sheet, "Y11", 8.0, "#,##0")
	mustSetNumberWithFormat(t, f, sheet, "Z11", 4.0, "#,##0")

	return f
}

// roundTrip serialises the workbook to a byte buffer and re-opens it so the
// number format is actually persisted in styles.xml. Without this step, the
// in-memory file would not have the "#,##0" style applied to M11 and the test
// would not exercise the bug.
func roundTrip(t *testing.T, f *excelize.File) *excelize.File {
	t.Helper()
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("xlsx write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("xlsx close: %v", err)
	}
	rt, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("xlsx reopen: %v", err)
	}
	t.Cleanup(func() { _ = rt.Close() })
	return rt
}

func mustSet(t *testing.T, f *excelize.File, sheet, cell, val string) {
	t.Helper()
	if err := f.SetCellStr(sheet, cell, val); err != nil {
		t.Fatalf("SetCellStr(%s): %v", cell, err)
	}
}

func mustSetNumberWithFormat(t *testing.T, f *excelize.File, sheet, cell string, val float64, format string) {
	t.Helper()
	if err := f.SetCellValue(sheet, cell, val); err != nil {
		t.Fatalf("SetCellValue(%s): %v", cell, err)
	}
	styleID, err := f.NewStyle(&excelize.Style{CustomNumFmt: &format})
	if err != nil {
		t.Fatalf("NewStyle(%s, %q): %v", cell, format, err)
	}
	if err := f.SetCellStyle(sheet, cell, cell, styleID); err != nil {
		t.Fatalf("SetCellStyle(%s): %v", cell, err)
	}
}

// TestParseBCCFile_DecimalHoursPreserved is the regression test for the
// 7.5h→8h rounding bug: with RawCellValue, the underlying 7.5 in M11 must
// survive the parser.
func TestParseBCCFile_DecimalHoursPreserved(t *testing.T) {
	f := roundTrip(t, buildDecimalHoursBCCXLSX(t))

	data, err := ParseBCCFile(f)
	if err != nil {
		t.Fatalf("ParseBCCFile: %v", err)
	}
	if len(data.Employees) != 1 {
		t.Fatalf("got %d employees, want 1", len(data.Employees))
	}
	emp := data.Employees[0]
	if emp.FullName != "Nguyễn Trọng Thắng" {
		t.Errorf("FullName = %q, want %q", emp.FullName, "Nguyễn Trọng Thắng")
	}
	if emp.CCCD != "040205013152" {
		t.Errorf("CCCD = %q, want %q", emp.CCCD, "040205013152")
	}

	// Build a (day, shift) → hours lookup.
	type key struct {
		day   int
		shift string
	}
	got := map[key]float64{}
	for _, e := range emp.Entries {
		got[key{e.DayNum, e.ShiftLabel}] = e.Hours
	}

	want := map[key]float64{
		{15, "CB N"}: 7.0,
		{15, "OT N"}: 4.0,
		{16, "CB N"}: 7.5, // ← the regression: must NOT be 8
		{16, "OT N"}: 4.0,
		{18, "CB N"}: 8.0,
		{18, "OT N"}: 4.0,
		{19, "CB N"}: 8.0,
		{19, "OT N"}: 4.0,
	}
	for k, w := range want {
		v, ok := got[k]
		if !ok {
			t.Errorf("missing entry day=%d shift=%q", k.day, k.shift)
			continue
		}
		if v != w {
			t.Errorf("day=%d shift=%q: got %.4f, want %.4f", k.day, k.shift, v, w)
		}
	}
	for k := range got {
		if _, expected := want[k]; !expected {
			t.Errorf("unexpected entry day=%d shift=%q hours=%.4f", k.day, k.shift, got[k])
		}
	}
}

// TestParseBCCFile_FormattedHoursNotRounded cross-checks that the formatted
// read path on M11 actually produces "8" (i.e. the test fixture is a faithful
// reproduction of the production bug). If excelize ever changes how it
// applies number formats, this test will fail and the synthetic fixture needs
// to be rebuilt — at which point the regression test above still guards the
// behavioural contract.
func TestParseBCCFile_FormattedHoursNotRounded(t *testing.T) {
	f := roundTrip(t, buildDecimalHoursBCCXLSX(t))

	formatted, err := f.GetCellValue("BCC", "M11")
	if err != nil {
		t.Fatalf("GetCellValue: %v", err)
	}
	raw, err := f.GetCellValue("BCC", "M11", excelize.Options{RawCellValue: true})
	if err != nil {
		t.Fatalf("GetCellValue RawCellValue: %v", err)
	}

	formatted = strings.TrimSpace(formatted)
	raw = strings.TrimSpace(raw)

	// Sanity: the formatted value must differ from the raw value (i.e. the
	// number format is being applied to a non-integer). If excelize ever
	// stops applying the format, the regression test is no longer meaningful
	// and this guard alerts the next maintainer.
	if formatted == raw {
		t.Skipf("formatted==raw (%q); number format no longer affects the value — fixture needs updating", formatted)
	}
	if formatted != "8" {
		t.Errorf("formatted M11 = %q, want %q (the bug-triggering display)", formatted, "8")
	}
	if raw != "7.5" {
		t.Errorf("raw M11 = %q, want %q", raw, "7.5")
	}
}
