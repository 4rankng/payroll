package excelkit

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestClean(t *testing.T) {
	cases := []struct{ in, want string }{
		{"'030172019748", "030172019748"},          // Excel text-escape apostrophe
		{`"030172019748"`, "030172019748"},         // double-quote escape
		{"  030172019748  ", "030172019748"},       // plain padding (existing contract)
		{"'\t 033072000021 \n\"", "033072000021"},  // mixed junk on both ends
		{"Lại Thị Hải Hà", "Lại Thị Hải Hà"},       // untouched
		{"O'Brien", "O'Brien"},                     // interior quote untouched
		{"Techcombank'", "Techcombank"},            // trailing escape
		{"   ", ""},                                // whitespace-only → blank
		{"''", ""},                                 // all quotes → blank
		{"", ""},
	}
	for _, tc := range cases {
		if got := Clean(tc.in); got != tc.want {
			t.Errorf("Clean(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// Cell must strip text-escape quotes from real worksheet cells: partner files
// ship identity columns escaped as text ('030172019748), and the BCC/STK
// parsers read every identity field through Cell.
func TestCellStripsTextEscapeQuotes(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	cells := map[string]string{
		"B2": "'030172019748",
		"C2": `" 033072000021 "`,
		"D2": "Lại Thị Hải Hà",
		"E2": "  trimmed only  ",
	}
	for cn, v := range cells {
		if err := f.SetCellValue("Sheet1", cn, v); err != nil {
			t.Fatal(err)
		}
	}

	if got := Cell(f, "Sheet1", 2, 2); got != "030172019748" {
		t.Errorf("Cell(B2) = %q, want 030172019748", got)
	}
	if got := Cell(f, "Sheet1", 3, 2); got != "033072000021" {
		t.Errorf("Cell(C2) = %q, want 033072000021", got)
	}
	if got := Cell(f, "Sheet1", 4, 2); got != "Lại Thị Hải Hà" {
		t.Errorf("Cell(D2) = %q, want untouched name", got)
	}
	if got := Cell(f, "Sheet1", 5, 2); got != "trimmed only" {
		t.Errorf("Cell(E2) = %q, want trimmed only", got)
	}
}
