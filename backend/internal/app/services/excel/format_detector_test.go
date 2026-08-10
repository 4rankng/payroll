package excel

import (
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// setBCCHeaders writes the standard BCC header row at row 4 (STT, Ma nhan vien, Ho va ten).
func setBCCHeaders(t *testing.T, f *excelize.File, sheet string) {
	t.Helper()
	_ = f.SetCellValue(sheet, "A4", "STT")
	_ = f.SetCellValue(sheet, "B4", "Mã nhân viên")
	_ = f.SetCellValue(sheet, "C4", "Họ và tên")
}

func TestDetectFormat_Legacy_BCCSheet(t *testing.T) {
	t.Parallel()
	f := excelize.NewFile()

	idx, err := f.NewSheet("BCC")
	if err != nil {
		t.Fatalf("create BCC sheet: %v", err)
	}
	f.SetActiveSheet(idx)
	setBCCHeaders(t, f, "BCC")

	result, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat returned error: %v", err)
	}
	if result.Format != FormatLegacy {
		t.Errorf("Format = %v, want FormatLegacy", result.Format)
	}
}

func TestDetectFormat_MultiPosition(t *testing.T) {
	t.Parallel()
	f := excelize.NewFile()

	idx, err := f.NewSheet("Có tay nghề")
	if err != nil {
		t.Fatalf("create sheet: %v", err)
	}
	f.SetActiveSheet(idx)
	setBCCHeaders(t, f, "Có tay nghề")

	_, _ = f.NewSheet("Stk")

	result, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat returned error: %v", err)
	}
	if result.Format != FormatMultiPosition {
		t.Errorf("Format = %v, want FormatMultiPosition", result.Format)
	}
	if len(result.PositionSheets) != 1 || result.PositionSheets[0] != "Có tay nghề" {
		t.Errorf("PositionSheets = %v, want [\"Có tay nghề\"]", result.PositionSheets)
	}
}

func TestDetectFormat_MultiPosition_MultipleSheets(t *testing.T) {
	t.Parallel()
	f := excelize.NewFile()

	for _, name := range []string{"Có tay nghề", "Không có tay nghề"} {
		idx, err := f.NewSheet(name)
		if err != nil {
			t.Fatalf("create sheet %q: %v", name, err)
		}
		f.SetActiveSheet(idx)
		setBCCHeaders(t, f, name)
	}
	_, _ = f.NewSheet("Stk")

	result, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat returned error: %v", err)
	}
	if result.Format != FormatMultiPosition {
		t.Errorf("Format = %v, want FormatMultiPosition", result.Format)
	}
	if len(result.PositionSheets) != 2 {
		t.Fatalf("PositionSheets count = %d, want 2", len(result.PositionSheets))
	}
	want := map[string]bool{"Có tay nghề": true, "Không có tay nghề": true}
	for _, s := range result.PositionSheets {
		if !want[s] {
			t.Errorf("unexpected position sheet %q", s)
		}
	}
}

func TestDetectFormat_HiddenBCC(t *testing.T) {
	t.Parallel()
	f := excelize.NewFile()

	_, err := f.NewSheet("BCC")
	if err != nil {
		t.Fatalf("create BCC sheet: %v", err)
	}
	setBCCHeaders(t, f, "BCC")

	visIdx, err := f.NewSheet("Có tay nghề")
	if err != nil {
		t.Fatalf("create position sheet: %v", err)
	}
	// Set position sheet active BEFORE hiding BCC â excelize won't hide the active sheet.
	f.SetActiveSheet(visIdx)
	setBCCHeaders(t, f, "Có tay nghề")

	if err := f.SetSheetVisible("BCC", false); err != nil {
		t.Fatalf("hide BCC sheet: %v", err)
	}
	setBCCHeaders(t, f, "Có tay nghề")

	result, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat returned error: %v", err)
	}
	if result.Format != FormatMultiPosition {
		t.Errorf("Format = %v, want FormatMultiPosition (hidden BCC skipped)", result.Format)
	}
	if len(result.PositionSheets) != 1 || result.PositionSheets[0] != "Có tay nghề" {
		t.Errorf("PositionSheets = %v, want [\"Có tay nghề\"]", result.PositionSheets)
	}
}

func TestDetectFormat_NoBCC_NoPositionSheets(t *testing.T) {
	t.Parallel()
	f := excelize.NewFile()

	idx, err := f.NewSheet("Stk")
	if err != nil {
		t.Fatalf("create Stk sheet: %v", err)
	}
	f.SetActiveSheet(idx)

	_, err = DetectFormat(f)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "không nhận diện được") {
		t.Errorf("error = %q, want containing 'không nhận diện được'", err.Error())
	}
}

func TestDetectFormat_NoValidHeaders(t *testing.T) {
	t.Parallel()
	f := excelize.NewFile()

	idx, err := f.NewSheet("Hỗ trợ khác")
	if err != nil {
		t.Fatalf("create sheet: %v", err)
	}
	f.SetActiveSheet(idx)
	_ = f.SetCellValue("Hỗ trợ khác", "A4", "Cột 1")
	_ = f.SetCellValue("Hỗ trợ khác", "B4", "Cột 2")

	_, err = DetectFormat(f)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "không nhận diện được") {
		t.Errorf("error = %q, want containing 'không nhận diện được'", err.Error())
	}
}

func TestDetectFormat_BCCTiebreaker(t *testing.T) {
	t.Parallel()
	f := excelize.NewFile()

	bccIdx, err := f.NewSheet("BCC")
	if err != nil {
		t.Fatalf("create BCC sheet: %v", err)
	}
	f.SetActiveSheet(bccIdx)
	setBCCHeaders(t, f, "BCC")

	posIdx, err := f.NewSheet("Có tay nghề")
	if err != nil {
		t.Fatalf("create position sheet: %v", err)
	}
	f.SetActiveSheet(posIdx)
	setBCCHeaders(t, f, "Có tay nghề")

	result, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat returned error: %v", err)
	}
	if result.Format != FormatLegacy {
		t.Errorf("Format = %v, want FormatLegacy (BCC tiebreaker)", result.Format)
	}
	if len(result.PositionSheets) != 0 {
		t.Errorf("PositionSheets = %v, want empty (legacy format has none)", result.PositionSheets)
	}
}

// TestDetectFormat_STKSheetTrailingSpace guards the STK-sheet skip against
// surrounding whitespace: a sheet exported as "STK " must still be skipped by
// DetectFormat (not treated as a position sheet), so a BCC+STK file is routed
// to the legacy monthly importer.
func TestDetectFormat_STKSheetTrailingSpace(t *testing.T) {
	for _, stkName := range []string{"STK ", " STK", " stk ", "Stk"} {
		t.Run(stkName, func(t *testing.T) {
			f := excelize.NewFile()
			idx, err := f.NewSheet("BCC")
			if err != nil {
				t.Fatalf("create BCC sheet: %v", err)
			}
			f.SetActiveSheet(idx)
			setBCCHeaders(t, f, "BCC")

			if _, err := f.NewSheet(stkName); err != nil {
				t.Fatalf("create STK sheet %q: %v", stkName, err)
			}
			// Put position-like headers in the STK sheet to prove it is NOT
			// misclassified as a position sheet.
			_ = f.SetCellValue(stkName, "A4", "STT")
			_ = f.SetCellValue(stkName, "B4", "Mã nhân viên")
			_ = f.SetCellValue(stkName, "C4", "Họ và tên")

			result, err := DetectFormat(f)
			if err != nil {
				t.Fatalf("DetectFormat: %v", err)
			}
			if result.Format != FormatLegacy {
				t.Errorf("sheet %q: expected FormatLegacy, got %v", stkName, result.Format)
			}
		})
	}
}

// setWeeklyPaymentHeaders writes the weekly payment header row at row 8.
func setWeeklyPaymentHeaders(t *testing.T, f *excelize.File, sheet string) {
	t.Helper()
	_ = f.SetCellValue(sheet, "A8", "STT")
	_ = f.SetCellValue(sheet, "B8", "Mã nhân viên")
	_ = f.SetCellValue(sheet, "C8", "Họ và tên")
	_ = f.SetCellValue(sheet, "D8", "Bộ phận")
	_ = f.SetCellValue(sheet, "E8", "Lương 8h")
}

// setWeeklyPaymentShiftRow writes shift codes at row 10.
func setWeeklyPaymentShiftRow(t *testing.T, f *excelize.File, sheet string) {
	t.Helper()
	_ = f.SetCellValue(sheet, "F10", "HC")
	_ = f.SetCellValue(sheet, "G10", "TCN")
	_ = f.SetCellValue(sheet, "H10", "HC")
	_ = f.SetCellValue(sheet, "I10", "TCN")
	_ = f.SetCellValue(sheet, "J10", "HC")
	_ = f.SetCellValue(sheet, "K10", "TCN")
	_ = f.SetCellValue(sheet, "L10", "HC")
	_ = f.SetCellValue(sheet, "M10", "TCN")
	_ = f.SetCellValue(sheet, "N10", "NN")
	_ = f.SetCellValue(sheet, "O10", "TCNN")
}

func TestDetectFormat_WeeklyPayment_SingleSheet(t *testing.T) {
	t.Parallel()
	f := excelize.NewFile()

	sheet := "520"
	idx, err := f.NewSheet(sheet)
	if err != nil {
		t.Fatalf("create sheet %q: %v", sheet, err)
	}
	f.SetActiveSheet(idx)
	setWeeklyPaymentHeaders(t, f, sheet)
	setWeeklyPaymentShiftRow(t, f, sheet)

	result, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat returned error: %v", err)
	}
	if result.Format != FormatWeeklyPayment {
		t.Errorf("Format = %v, want FormatWeeklyPayment", result.Format)
	}
	if len(result.WeeklyPaymentSheets) != 1 || result.WeeklyPaymentSheets[0] != sheet {
		t.Errorf("WeeklyPaymentSheets = %v, want [%q]", result.WeeklyPaymentSheets, sheet)
	}
}

func TestDetectFormat_WeeklyPayment_MultipleSheets(t *testing.T) {
	t.Parallel()
	f := excelize.NewFile()

	sheets := []string{"520", "700", "750", "800", "900"}
	for _, sheet := range sheets {
		idx, err := f.NewSheet(sheet)
		if err != nil {
			t.Fatalf("create sheet %q: %v", sheet, err)
		}
		f.SetActiveSheet(idx)
		setWeeklyPaymentHeaders(t, f, sheet)
		setWeeklyPaymentShiftRow(t, f, sheet)
	}

	result, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat returned error: %v", err)
	}
	if result.Format != FormatWeeklyPayment {
		t.Errorf("Format = %v, want FormatWeeklyPayment", result.Format)
	}
	if len(result.WeeklyPaymentSheets) != len(sheets) {
		t.Errorf("WeeklyPaymentSheets count = %d, want %d", len(result.WeeklyPaymentSheets), len(sheets))
	}
}

func TestDetectFormat_WeeklyPayment_Row4HeadersNotRow8(t *testing.T) {
	t.Parallel()
	f := excelize.NewFile()

	// Row 4 headers instead of row 8 (wrong layout for weekly payment)
	sheet := "520"
	idx, err := f.NewSheet(sheet)
	if err != nil {
		t.Fatalf("create sheet %q: %v", sheet, err)
	}
	f.SetActiveSheet(idx)
	setBCCHeaders(t, f, sheet) // Row 4 headers, not row 8

	_, _ = f.NewSheet("Stk")

	result, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat returned error: %v", err)
	}
	// Should fall through to FormatMultiPosition, not FormatWeeklyPayment
	if result.Format != FormatMultiPosition {
		t.Errorf("Format = %v, want FormatMultiPosition (wrong layout for weekly payment)", result.Format)
	}
	if len(result.PositionSheets) != 1 || result.PositionSheets[0] != sheet {
		t.Errorf("PositionSheets = %v, want [%q]", result.PositionSheets, sheet)
	}
	if len(result.WeeklyPaymentSheets) != 0 {
		t.Errorf("WeeklyPaymentSheets = %v, want empty (wrong layout)", result.WeeklyPaymentSheets)
	}
}

func TestDetectFormat_WeeklyPayment_AnySheetName(t *testing.T) {
	t.Parallel()
	f := excelize.NewFile()

	// Non-numeric sheet name but with row 8/10 fingerprint should still work
	sheet := "Phổ thông"
	idx, err := f.NewSheet(sheet)
	if err != nil {
		t.Fatalf("create sheet %q: %v", sheet, err)
	}
	f.SetActiveSheet(idx)
	setWeeklyPaymentHeaders(t, f, sheet)
	setWeeklyPaymentShiftRow(t, f, sheet)

	_, _ = f.NewSheet("Stk")

	result, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat returned error: %v", err)
	}
	// Should detect as FormatWeeklyPayment despite non-numeric name
	if result.Format != FormatWeeklyPayment {
		t.Errorf("Format = %v, want FormatWeeklyPayment (any sheet name with row 8/10 fingerprint)", result.Format)
	}
	if len(result.WeeklyPaymentSheets) != 1 || result.WeeklyPaymentSheets[0] != sheet {
		t.Errorf("WeeklyPaymentSheets = %v, want [%q]", result.WeeklyPaymentSheets, sheet)
	}
}

func TestDetectFormat_WeeklyPayment_NoShiftRow(t *testing.T) {
	t.Parallel()
	f := excelize.NewFile()

	sheet := "520"
	idx, err := f.NewSheet(sheet)
	if err != nil {
		t.Fatalf("create sheet %q: %v", sheet, err)
	}
	f.SetActiveSheet(idx)
	setWeeklyPaymentHeaders(t, f, sheet)
	// No shift row set

	_, _ = f.NewSheet("Stk")

	result, err := DetectFormat(f)
	if err == nil {
		t.Fatalf("DetectFormat() result = %#v, want error for a weekly-payment header without shift codes", result)
	}
}
