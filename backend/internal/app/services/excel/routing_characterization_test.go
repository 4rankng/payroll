package excel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// Phase 0 routing characterization (excel-parsing-refactor): locks the
// DetectFormat → ParseBCCData dispatch semantics, including the quirks the
// refactor must preserve (hidden sheets, "STK " trim, priority order,
// unknown-template failure, BCC-named date-row fallback).

func newPositionSheet(t *testing.T, f *excelize.File, name string) {
	t.Helper()
	if _, err := f.NewSheet(name); err != nil {
		t.Fatal(err)
	}
	// Row 4 fingerprint: STT + "Mã nhân viên" + "Họ và tên"
	for col, header := range map[int]string{1: "STT", 2: "Mã nhân viên", 3: "Họ và tên"} {
		cell, _ := excelize.CoordinatesToCellName(col, 4)
		if err := f.SetCellValue(name, cell, header); err != nil {
			t.Fatal(err)
		}
	}
}

// B6: an STK-only workbook (trailing-space sheet name) must not route as a
// data format — the trim rule keeps the bank sheet out of position detection.
func TestRouting_STKOnlyWorkbookFails(t *testing.T) {
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", "STK "); err != nil {
		t.Fatal(err)
	}
	if _, err := DetectFormat(f); err == nil {
		t.Fatal("DetectFormat succeeded on STK-only workbook, want error")
	} else if !strings.Contains(err.Error(), "không nhận diện") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// B7: hidden sheets are skipped — a hidden "BCC" sheet must not force
// FormatLegacy over a visible position sheet.
func TestRouting_HiddenSheetSkipped(t *testing.T) {
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", "BCC"); err != nil {
		t.Fatal(err)
	}
	newPositionSheet(t, f, "Công nhân")
	// excelize refuses to hide the active sheet — activate the visible one first.
	visibleIdx, err := f.GetSheetIndex("Công nhân")
	if err != nil {
		t.Fatal(err)
	}
	f.SetActiveSheet(visibleIdx)
	if err := f.SetSheetVisible("BCC", false); err != nil {
		t.Fatal(err)
	}
	det, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat: %v", err)
	}
	if det.Format != FormatMultiPosition {
		t.Fatalf("format = %v, want FormatMultiPosition (hidden BCC skipped)", det.Format)
	}
}

// B8/B9: unknown template must hard-fail, never silently import.
func TestRouting_UnknownTemplateFails(t *testing.T) {
	for name, mutate := range map[string]func(*excelize.File){
		"empty workbook": func(f *excelize.File) {},
		"unrecognized sheet": func(f *excelize.File) {
			_ = f.SetCellValue("Sheet1", "A1", "Foo")
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := excelize.NewFile()
			mutate(f)
			if _, err := DetectFormat(f); err == nil {
				t.Fatal("DetectFormat succeeded on unknown template, want error")
			}
			if _, _, err := ParseBCCData(f); err == nil {
				t.Fatal("ParseBCCData succeeded on unknown template, want error")
			}
		})
	}
}

// B10: priority order — a named weekly-BCC sheet outranks position sheets.
func TestRouting_WeeklyBeatsPosition(t *testing.T) {
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", "BCC-OT150"); err != nil {
		t.Fatal(err)
	}
	newPositionSheet(t, f, "Công nhân")
	det, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat: %v", err)
	}
	if det.Format != FormatWeeklyBCC {
		t.Fatalf("format = %v, want FormatWeeklyBCC (priority over MultiPosition)", det.Format)
	}
}

// C2: a numeric-only "BCC" sheet must fail legacy Accept (pure-digit shift
// labels are manufactured, not real) and not import as legacy.
func TestRouting_NumericOnlyBCCSheetFailsLegacyAccept(t *testing.T) {
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", "BCC"); err != nil {
		t.Fatal(err)
	}
	// Numeric junk that a naive legacy parse could misread as rows.
	for r := 1; r <= 12; r++ {
		for c := 1; c <= 8; c++ {
			cell, _ := excelize.CoordinatesToCellName(c, r)
			_ = f.SetCellValue("BCC", cell, r*c)
		}
	}
	if _, _, err := ParseBCCData(f); err == nil {
		t.Fatal("ParseBCCData succeeded on numeric-only BCC sheet, want error")
	}
}

// C3 (synthetic companion to the real BUMHAN test): the M1-style builder
// routes to FormatDateRow through both DetectFormat and ParseBCCData.
func TestRouting_M1StyleWorkbookIsDateRow(t *testing.T) {
	f := buildM1StyleWorkbook(t)
	det, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat: %v", err)
	}
	if det.Format != FormatDateRow {
		t.Fatalf("format = %v, want FormatDateRow", det.Format)
	}
	_, winner, err := ParseBCCData(f)
	if err != nil {
		t.Fatalf("ParseBCCData: %v", err)
	}
	if winner != FormatDateRow {
		t.Fatalf("winner = %v, want FormatDateRow", winner)
	}
}

// B1–B5 (real files): DetectFormat classifies each committed fixture.
func TestRouting_RealFixturesClassified(t *testing.T) {
	cases := []struct {
		fixture []string
		want    BCCFormat
	}{
		{[]string{"legacy", "eva_t08.xlsx"}, FormatLegacy},
		{[]string{"date_row", "bumhan_t08.xlsx"}, FormatLegacy}, // sheet name; winner is DateRow
		{[]string{"weekly_bcc", "lgd_weekly.xlsx"}, FormatWeeklyBCC},
		{[]string{"weekly_payment", "tbd_weekly_payment.xlsx"}, FormatWeeklyPayment},
		{[]string{"multi_position", "pqc_haiphong.xlsx"}, FormatMultiPosition},
	}
	for _, tc := range cases {
		t.Run(tc.fixture[len(tc.fixture)-1], func(t *testing.T) {
			f := openBCCFixture(t, tc.fixture...)
			det, err := DetectFormat(f)
			if err != nil {
				t.Fatalf("DetectFormat: %v", err)
			}
			if det.Format != tc.want {
				t.Fatalf("format = %v, want %v", det.Format, tc.want)
			}
		})
	}
}

// Env-gated real-file sweep: any *.xlsx dropped into testplan/excelfixture
// (or BCC_REAL_FILE_DIR) is routed through DetectFormat as a smoke check.
// No assertions beyond no-panic + logged outcome, so new partner files can
// land without touching this test.
func TestRouting_RealFileDirSweep(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("BCC_REAL_FILE_DIR")
	if dir == "" {
		dir = filepath.Join(cwd, "..", "..", "..", "..", "..", "testplan", "excelfixture")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("real-file dir unavailable: %v", err)
	}
	checked := 0
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".xlsx") {
			continue
		}
		f, err := excelize.OpenFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		det, detErr := DetectFormat(f)
		switch {
		case detErr != nil:
			t.Logf("%s: DetectFormat error: %v", e.Name(), detErr)
		default:
			t.Logf("%s: %v", e.Name(), det.Format)
		}
		_ = f.Close()
		checked++
	}
	if checked == 0 {
		t.Skip("no xlsx files in real-file dir")
	}
}
