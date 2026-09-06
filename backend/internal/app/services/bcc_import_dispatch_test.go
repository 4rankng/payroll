package services

import (
	"os"
	"path/filepath"
	"testing"

	excelparser "api-server/internal/app/services/excel"

	"github.com/xuri/excelize/v2"
)

// Phase 3 dispatch-contract tests (excel-parsing-refactor): the named
// helpers that carry the production behavior keys, asserted by name.

func openBCCFixtureFromServices(t *testing.T, rel ...string) *excelize.File {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(append([]string{cwd, "..", "..", "..", "tests", "fixtures", "bcc"}, rel...)...)
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Skipf("fixture unavailable: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// The winner-format contract: DetectFormat sees the "BCC" sheet name
// (FormatLegacy), but parseLegacyRoute must overwrite it with the strategy
// winner (FormatDateRow for T09-style files) — downstream STK collection,
// position precedence, and label-keyed resolution all read the winner.
func TestParseLegacyRoute_WinnerFormatContract_BumhanT09(t *testing.T) {
	f := openBCCFixtureFromServices(t, "date_row", "bumhan_t08.xlsx")

	det, err := excelparser.DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat: %v", err)
	}
	if det.Format != excelparser.FormatLegacy {
		t.Fatalf("DetectFormat = %v, want FormatLegacy (sheet-name detection)", det.Format)
	}

	parsed, perr := parseLegacyRoute(f, det, "bumhan_t08.xlsx")
	if perr != nil {
		t.Fatalf("parseLegacyRoute: %v", perr)
	}
	if det.Format != excelparser.FormatDateRow {
		t.Fatalf("winner-format overwrite: det.Format = %v, want FormatDateRow", det.Format)
	}
	if parsed == nil || len(parsed.Employees) == 0 {
		t.Fatal("winner parse produced no employees")
	}
}

func TestParseLegacyRoute_LegacyStaysLegacy(t *testing.T) {
	f := openBCCFixtureFromServices(t, "legacy", "eva_t08.xlsx")

	det, err := excelparser.DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat: %v", err)
	}
	if _, perr := parseLegacyRoute(f, det, "eva_t08.xlsx"); perr != nil {
		t.Fatalf("parseLegacyRoute: %v", perr)
	}
	if det.Format != excelparser.FormatLegacy {
		t.Fatalf("det.Format = %v, want FormatLegacy", det.Format)
	}
}

// Label-keyed resolution is date-row-only: legacy rateless files must keep
// the calendar path or old templates get silently re-priced.
func TestUseLabelKeyedResolution(t *testing.T) {
	cases := []struct {
		rateless bool
		format   excelparser.BCCFormat
		want     bool
	}{
		{true, excelparser.FormatDateRow, true},
		{true, excelparser.FormatLegacy, false},
		{false, excelparser.FormatDateRow, false},
		{false, excelparser.FormatLegacy, false},
	}
	for _, tc := range cases {
		if got := useLabelKeyedResolution(tc.rateless, tc.format); got != tc.want {
			t.Errorf("useLabelKeyedResolution(rateless=%v, %v) = %v, want %v",
				tc.rateless, tc.format, got, tc.want)
		}
	}
}

// A date-row file's "Vị trí" column outranks rate deduction; blank cells and
// non-date-row formats keep the deduced default.
func TestResolveImportPosition(t *testing.T) {
	parsed := &excelparser.BCCImportData{
		Employees: []excelparser.BCCEmployeeData{
			{CCCD: "111", FullName: "A", Position: "600"},
			{CCCD: "222", FullName: "B"}, // blank position
		},
	}
	const deduced = "phổ thông"

	if got := resolveImportPosition(parsed, excelparser.FormatDateRow, "111", deduced, nil); got != "600" {
		t.Errorf("date-row file position = %q, want file value 600", got)
	}
	if got := resolveImportPosition(parsed, excelparser.FormatDateRow, "222", deduced, nil); got != deduced {
		t.Errorf("date-row blank position = %q, want deduced %q", got, deduced)
	}
	if got := resolveImportPosition(parsed, excelparser.FormatLegacy, "111", deduced, nil); got != deduced {
		t.Errorf("legacy file position = %q, want deduced %q (file column ignored)", got, deduced)
	}
	if got := resolveImportPosition(parsed, excelparser.FormatLegacy, "missing-cccd", deduced, nil); got != deduced {
		t.Errorf("unknown cccd position = %q, want deduced %q", got, deduced)
	}
}

func TestCrossCheckSTKName(t *testing.T) {
	stk := map[string]string{"111": "Nguyễn Thị A", "222": "Trần Văn B"}

	if mismatch := crossCheckSTKName(excelparser.BCCEmployeeData{CCCD: "111", FullName: "Nguyễn Thị A"}, stk); mismatch != nil {
		t.Errorf("exact name match rejected: %+v", mismatch)
	}
	// NFD vs NFC mac export difference must not mismatch.
	if mismatch := crossCheckSTKName(excelparser.BCCEmployeeData{CCCD: "111", FullName: "Nguyễn Thı̣ A"}, stk); mismatch != nil {
		t.Errorf("NFD name rejected: %+v", mismatch)
	}
	// Single diacritic typo tolerated.
	if mismatch := crossCheckSTKName(excelparser.BCCEmployeeData{CCCD: "111", FullName: "Nguyen Thi A"}, stk); mismatch != nil {
		t.Errorf("diacritic-stripped equal name rejected: %+v", mismatch)
	}
	// Genuinely different person → mismatch.
	if mismatch := crossCheckSTKName(excelparser.BCCEmployeeData{CCCD: "111", FullName: "Lê Hoàn Toàn Khác"}, stk); mismatch == nil {
		t.Error("different person accepted")
	}
	// CCCD absent from STK → no check.
	if mismatch := crossCheckSTKName(excelparser.BCCEmployeeData{CCCD: "999", FullName: "Ai Đó"}, stk); mismatch != nil {
		t.Errorf("unknown CCCD rejected: %+v", mismatch)
	}
}
