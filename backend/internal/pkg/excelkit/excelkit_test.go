package excelkit

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestNormalizeHeader(t *testing.T) {
	cases := []struct{ in, want string }{
		{"  Họ và Tên  ", "họ và tên"},                 // trim + lower
		{"họ và tên", "họ và tên"},                     // NFD → NFC
		{"Số  TK", "số  tk"},                           // internal whitespace preserved
		{"", ""},
	}
	for _, tc := range cases {
		if got := NormalizeHeader(tc.in); got != tc.want {
			t.Errorf("NormalizeHeader(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCollapseHeader(t *testing.T) {
	cases := []struct{ in, want string }{
		{"  Số   tài  khoản ", "số tài khoản"}, // collapsed to single spaces
		{"STT\n(Ord. No.)", "stt (ord. no.)"},  // newline → space
		{"", ""},
	}
	for _, tc := range cases {
		if got := CollapseHeader(tc.in); got != tc.want {
			t.Errorf("CollapseHeader(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestAliasTableResolve(t *testing.T) {
	table := AliasTable{
		"stt":              "order_no",
		"(ord. no.)":       "order_no",
		"số tài khoản":     "account_number", // diacritic key (collapsed match)
		"so tai khoan":     "account_number", // ASCII key (unidecode retry)
		"beneficiary bank": "bank_name",
	}

	cases := []struct {
		in     string
		want   string
		wantOK bool
		note   string
	}{
		{" STT ", "order_no", true, "whole cell, trimmed + lowered"},
		{"STT\n(Ord. No.)", "order_no", true, "newline segment match"},
		{"Số  tài  khoản", "account_number", true, "collapsed match on diacritic key"},
		{"Số tài khoản", "account_number", true, "unidecode retry on ASCII key"},
		{"Beneficiary Bank", "bank_name", true, "case-insensitive"},
		{"Unknown", "", false, "no match"},
	}
	for _, tc := range cases {
		got, ok := table.Resolve(tc.in)
		if ok != tc.wantOK || got != tc.want {
			t.Errorf("Resolve(%q) = (%q, %v), want (%q, %v) [%s]", tc.in, got, ok, tc.want, tc.wantOK, tc.note)
		}
	}
}

func TestRowCell(t *testing.T) {
	row := []string{"a", "b"}
	if got := RowCell(row, -1); got != "" {
		t.Errorf("RowCell(-1) = %q, want empty", got)
	}
	if got := RowCell(row, 5); got != "" {
		t.Errorf("RowCell(5) = %q, want empty", got)
	}
	if got := RowCell(row, 1); got != "b" {
		t.Errorf("RowCell(1) = %q, want b", got)
	}
}

func TestCellAndNumericCell(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if err := f.SetCellValue("Sheet1", "B2", "  Nguyễn Văn A  "); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellValue("Sheet1", "B3", 7.5); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellValue("Sheet1", "B4", 0); err != nil {
		t.Fatal(err)
	}

	if got := Cell(f, "Sheet1", 2, 2); got != "Nguyễn Văn A" {
		t.Errorf("Cell = %q, want trimmed name", got)
	}
	if got := Cell(f, "Sheet1", 99, 99); got != "" {
		t.Errorf("Cell out of range = %q, want empty", got)
	}

	if v, ok := NumericCell(f, "Sheet1", 2, 3); !ok || v != 7.5 {
		t.Errorf("NumericCell = (%v, %v), want (7.5, true)", v, ok)
	}
	if v, ok := NumericCell(f, "Sheet1", 2, 4); !ok || v != 0 {
		t.Errorf("NumericCell explicit zero = (%v, %v), want (0, true)", v, ok)
	}
	if _, ok := NumericCell(f, "Sheet1", 2, 5); ok {
		t.Error("NumericCell blank = ok, want false")
	}
}

func TestExcelDate(t *testing.T) {
	if d, ok := ExcelDate("45978"); !ok || d.Format("2006-01-02") != "2025-11-17" {
		t.Errorf("ExcelDate(45978) = (%v, %v), want 2025-11-17", d.Format("2006-01-02"), ok)
	}
	// Fractional serial floors to the date.
	if d, ok := ExcelDate("45978.75"); !ok || d.Hour() != 0 {
		t.Errorf("fractional serial not floored: %v", d)
	}
	if _, ok := ExcelDate("abc"); ok {
		t.Error("non-numeric accepted")
	}
	if _, ok := ExcelDate("0"); ok {
		t.Error("serial < 1 accepted")
	}
	if _, ok := ExcelDate("1"); ok {
		t.Error("1900 serial accepted outside 2020-2040 guard")
	}
}

func TestIsEmptyRowAndFindHeaderRow(t *testing.T) {
	if !IsEmptyRow([]string{" ", "", "\t"}) {
		t.Error("blank row not detected as empty")
	}
	if IsEmptyRow([]string{"", "x"}) {
		t.Error("non-blank row detected as empty")
	}

	rows := [][]string{
		{"junk", "junk"},
		{"", "", ""},
		{"STT", "Mã nhân viên"},
	}
	if got := FindHeaderRow(rows, 30, func(r []string) bool { return len(r) > 0 && r[0] == "STT" }); got != 2 {
		t.Errorf("FindHeaderRow = %d, want 2", got)
	}
	if got := FindHeaderRow(rows, 2, func(r []string) bool { return len(r) > 0 && r[0] == "STT" }); got != -1 {
		t.Errorf("FindHeaderRow with maxScan=2 = %d, want -1", got)
	}
}

func TestOpenReader(t *testing.T) {
	// A valid workbook opens through the capped reader.
	src := excelize.NewFile()
	if err := src.SetCellValue("Sheet1", "A1", "ok"); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := src.Write(&buf); err != nil {
		t.Fatal(err)
	}

	f, err := OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	defer func() { _ = f.Close() }()
	if got := Cell(f, "Sheet1", 1, 1); got != "ok" {
		t.Errorf("roundtrip cell = %q", got)
	}

	// Garbage is rejected with an error, not a panic.
	if _, err := OpenReader(bytes.NewReader([]byte("not a zip"))); err == nil {
		t.Error("garbage input accepted")
	}

	// Caps are the documented defaults.
	if DefaultUnzipSizeLimit != 50<<20 || DefaultUnzipXMLSizeLimit != 10<<20 {
		t.Errorf("unexpected caps: %d / %d", DefaultUnzipSizeLimit, DefaultUnzipXMLSizeLimit)
	}
}
