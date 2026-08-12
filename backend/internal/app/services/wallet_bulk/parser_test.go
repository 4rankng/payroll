package wallet_bulk

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// buildOnePayInputXlsx constructs an in-memory "Yêu cầu chuyển tiền" workbook
// in the exact layout the OnePay exporter (Phase 2) produces. Row 1 is empty
// (margin), row 2 is the header, row 3+ is data. Used to test the parser
// against every supported header variant without committing a binary fixture.
func buildOnePayInputXlsx(t *testing.T, headers []string, data [][]any, sheetName string) []byte {
	t.Helper()
	if sheetName == "" {
		sheetName = ExpectedSheet
	}
	f := excelize.NewFile()
	if idx, _ := f.GetSheetIndex("Sheet1"); idx >= 0 {
		if err := f.DeleteSheet("Sheet1"); err != nil {
			t.Fatalf("DeleteSheet: %v", err)
		}
	}
	if _, err := f.NewSheet(sheetName); err != nil {
		t.Fatalf("NewSheet: %v", err)
	}
	if idx, _ := f.GetSheetIndex(sheetName); idx >= 0 {
		f.SetActiveSheet(idx)
	}

	// Row 1: blank (matches production eMB layout — header sits on row 2).
	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, headerRowIndex)
		if err := f.SetCellValue(sheetName, cell, h); err != nil {
			t.Fatalf("set header %s: %v", cell, err)
		}
	}
	for rowOffset, row := range data {
		excelRow := firstDataRow + rowOffset
		for colIdx, v := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, excelRow)
			if err := f.SetCellValue(sheetName, cell, v); err != nil {
				t.Fatalf("set %s: %v", cell, err)
			}
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("write xlsx: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close xlsx: %v", err)
	}
	return buf.Bytes()
}

// defaultHeaders returns the production header layout (Vietnamese with
// English in parens), matching what OnePayExporter.GenerateOnePayExport writes.
func defaultHeaders() []string {
	return []string{
		"STT\n(Ord. No.)",
		"Số tài khoản\n(Account No.)",
		"Tên người thụ hưởng\n(Beneficiary)",
		"Ngân hàng thụ hưởng/Chi nhánh\n(Beneficiary Bank)",
		"Mã SWIFT\n(SWIFT Code)",
		"Số tiền\n(Amount)",
		"Nội dung chuyển khoản\n(Payment Detail)",
	}
}

// defaultHappyRows returns two canonical rows using real SWIFT codes from
// migration 059 (MB → MBBEVNVX, VCB → BFTVVNVX).
func defaultHappyRows() [][]any {
	return [][]any{
		{1, "99990001", "Nguyen Test A", "Quân đội (MB)", "MBBEVNVX", int64(1_500_000), "VFIC3ba3ec31LUONGT1"},
		{2, "99990002", "Nguyen Test B", "Ngân hàng TMCP Ngoại thương (VCB)", "BFTVVNVX", int64(2_500_000), "VFIC4cd5ef62LUONGT2"},
	}
}

func TestParser_HappyPath(t *testing.T) {
	bytes := buildOnePayInputXlsx(t, defaultHeaders(), defaultHappyRows(), "")
	p := NewYeuCauChuyenTienParser(nil)
	rows, err := p.Parse(context.Background(), bytesReader(bytes))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	assertRow(t, rows[0], 1, "99990001", "Nguyen Test A", "Quân đội (MB)", "MBBEVNVX", 1_500_000, "VFIC3ba3ec31")
	assertRow(t, rows[1], 2, "99990002", "Nguyen Test B", "Ngân hàng TMCP Ngoại thương (VCB)", "BFTVVNVX", 2_500_000, "VFIC4cd5ef62")
}

func TestParser_MissingSwiftColumn(t *testing.T) {
	// Headers WITHOUT the SWIFT column — should fail with ErrMissingSwiftColumn.
	headers := []string{
		"STT", "Số tài khoản", "Tên người thụ hưởng",
		"Ngân hàng thụ hưởng/Chi nhánh", "Số tiền", "Nội dung chuyển khoản",
	}
	data := [][]any{
		{1, "99990001", "Nguyen Test A", "Quân đội (MB)", int64(1_500_000), "VFIC3ba3ec31"},
	}
	bytes := buildOnePayInputXlsx(t, headers, data, "")
	p := NewYeuCauChuyenTienParser(nil)
	_, err := p.Parse(context.Background(), bytesReader(bytes))
	if !errors.Is(err, ErrMissingSwiftColumn) {
		t.Fatalf("expected ErrMissingSwiftColumn, got %v", err)
	}
}

func TestParser_InvalidSwiftFormat(t *testing.T) {
	data := [][]any{
		{1, "99990001", "Nguyen Test A", "Quân đội (MB)", "MB", int64(1_500_000), "VFIC3ba3ec31"},
	}
	bytes := buildOnePayInputXlsx(t, defaultHeaders(), data, "")
	p := NewYeuCauChuyenTienParser(nil)
	_, err := p.Parse(context.Background(), bytesReader(bytes))
	if !errors.Is(err, ErrInvalidSwift) {
		t.Fatalf("expected ErrInvalidSwift, got %v", err)
	}
}

func TestParser_MissingVFICCode(t *testing.T) {
	data := [][]any{
		{1, "99990001", "Nguyen Test A", "Quân đội (MB)", "MBBEVNVX", int64(1_500_000), "LUONG THANG 1 KHONG CO VFIC"},
	}
	bytes := buildOnePayInputXlsx(t, defaultHeaders(), data, "")
	p := NewYeuCauChuyenTienParser(nil)
	_, err := p.Parse(context.Background(), bytesReader(bytes))
	if !errors.Is(err, ErrMissingVFICCode) {
		t.Fatalf("expected ErrMissingVFICCode, got %v", err)
	}
}

func TestParser_UnknownHeader(t *testing.T) {
	headers := []string{
		"STT", "Số tài khoản", "Tên người thụ hưởng",
		"Ngân hàng thụ hưởng/Chi nhánh", "Mã SWIFT", "Số tiền",
		"Nội dung chuyển khoản", "Cột lạ không hợp lệ",
	}
	data := [][]any{
		{1, "99990001", "Nguyen Test A", "Quân đội (MB)", "MBBEVNVX", int64(1_500_000), "VFIC3ba3ec31", "junk"},
	}
	bytes := buildOnePayInputXlsx(t, headers, data, "")
	p := NewYeuCauChuyenTienParser(nil)
	_, err := p.Parse(context.Background(), bytesReader(bytes))
	if !errors.Is(err, ErrUnknownExcelHeader) {
		t.Fatalf("expected ErrUnknownExcelHeader, got %v", err)
	}
}

func TestParser_ColumnReordering(t *testing.T) {
	// Same data, columns shuffled — parser must still map by header name.
	headers := []string{
		"Mã SWIFT",                      // moved to col A
		"Số tài khoản",                  // col B
		"Tên người thụ hưởng",           // col C
		"Ngân hàng thụ hưởng/Chi nhánh", // col D
		"STT",                           // col E (was A)
		"Số tiền",                       // col F
		"Nội dung chuyển khoản",         // col G
	}
	// Data must follow the SAME shuffled order.
	data := [][]any{
		{"MBBEVNVX", "99990001", "Nguyen Test A", "Quân đội (MB)", 1, int64(1_500_000), "VFIC3ba3ec31"},
		{"BFTVVNVX", "99990002", "Nguyen Test B", "VCB", 2, int64(2_500_000), "VFIC4cd5ef62"},
	}
	bytes := buildOnePayInputXlsx(t, headers, data, "")
	p := NewYeuCauChuyenTienParser(nil)
	rows, err := p.Parse(context.Background(), bytesReader(bytes))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	assertRow(t, rows[0], 1, "99990001", "Nguyen Test A", "Quân đội (MB)", "MBBEVNVX", 1_500_000, "VFIC3ba3ec31")
}

func TestParser_EmptyFile(t *testing.T) {
	// Headers only, no data rows.
	bytes := buildOnePayInputXlsx(t, defaultHeaders(), nil, "")
	p := NewYeuCauChuyenTienParser(nil)
	_, err := p.Parse(context.Background(), bytesReader(bytes))
	if !errors.Is(err, ErrEmptyFile) {
		t.Fatalf("expected ErrEmptyFile, got %v", err)
	}
}

func TestParser_InvalidSheet(t *testing.T) {
	// Workbook with the wrong sheet name.
	bytes := buildOnePayInputXlsx(t, defaultHeaders(), defaultHappyRows(), "WrongSheet")
	p := NewYeuCauChuyenTienParser(nil)
	_, err := p.Parse(context.Background(), bytesReader(bytes))
	if !errors.Is(err, ErrInvalidSheet) {
		t.Fatalf("expected ErrInvalidSheet, got %v", err)
	}
}

func TestParser_TooManyRows(t *testing.T) {
	// Build a workbook with MaxRows + 1 data rows.
	data := make([][]any, 0, MaxRows+1)
	for i := 0; i < MaxRows+1; i++ {
		data = append(data, []any{
			i + 1,
			"9999000" + strings.Repeat("1", 0),
			"Test",
			"Quân đội (MB)",
			"MBBEVNVX",
			int64(1_500_000),
			"VFIC3ba3ec31",
		})
	}
	// Each row needs a distinct VFIC for realism; the parser doesn't enforce
	// VFIC uniqueness so duplicates are fine here (we're testing the row cap).
	bytes := buildOnePayInputXlsx(t, defaultHeaders(), data, "")
	p := NewYeuCauChuyenTienParser(nil)
	_, err := p.Parse(context.Background(), bytesReader(bytes))
	if !errors.Is(err, ErrTooManyRows) {
		t.Fatalf("expected ErrTooManyRows, got %v", err)
	}
}

func TestParser_InvalidAmount(t *testing.T) {
	// Amount below MinAmountVND (100_000).
	data := [][]any{
		{1, "99990001", "Nguyen Test A", "Quân đội (MB)", "MBBEVNVX", int64(50_000), "VFIC3ba3ec31"},
	}
	bytes := buildOnePayInputXlsx(t, defaultHeaders(), data, "")
	p := NewYeuCauChuyenTienParser(nil)
	_, err := p.Parse(context.Background(), bytesReader(bytes))
	if !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestParser_AmountAsFormattedString(t *testing.T) {
	// Some producers emit "1.500.000" — parser must strip thousands separators.
	data := [][]any{
		{1, "99990001", "Nguyen Test A", "Quân đội (MB)", "MBBEVNVX", "1.500.000", "VFIC3ba3ec31"},
	}
	bytes := buildOnePayInputXlsx(t, defaultHeaders(), data, "")
	p := NewYeuCauChuyenTienParser(nil)
	rows, err := p.Parse(context.Background(), bytesReader(bytes))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if rows[0].Amount != 1_500_000 {
		t.Fatalf("expected 1_500_000, got %d", rows[0].Amount)
	}
}

func TestParser_LowercaseSwiftNormalized(t *testing.T) {
	// SWIFT entered as "mbbevnvx" — parser uppercases before validating.
	data := [][]any{
		{1, "99990001", "Nguyen Test A", "Quân đội (MB)", "mbbevnvx", int64(1_500_000), "VFIC3ba3ec31"},
	}
	bytes := buildOnePayInputXlsx(t, defaultHeaders(), data, "")
	p := NewYeuCauChuyenTienParser(nil)
	rows, err := p.Parse(context.Background(), bytesReader(bytes))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if rows[0].SwiftCode != "MBBEVNVX" {
		t.Fatalf("expected MBBEVNVX, got %q", rows[0].SwiftCode)
	}
}

func assertRow(
	t *testing.T,
	got BulkTransferRow,
	orderNo int,
	accountNo, accountName, bank, swift string,
	amount int64,
	vfic string,
) {
	t.Helper()
	if got.OrderNo != orderNo {
		t.Errorf("OrderNo: got %d, want %d", got.OrderNo, orderNo)
	}
	if got.AccountNo != accountNo {
		t.Errorf("AccountNo: got %q, want %q", got.AccountNo, accountNo)
	}
	if got.AccountName != accountName {
		t.Errorf("AccountName: got %q, want %q", got.AccountName, accountName)
	}
	if got.Bank != bank {
		t.Errorf("Bank: got %q, want %q", got.Bank, bank)
	}
	if got.SwiftCode != swift {
		t.Errorf("SwiftCode: got %q, want %q", got.SwiftCode, swift)
	}
	if got.Amount != amount {
		t.Errorf("Amount: got %d, want %d", got.Amount, amount)
	}
	if got.VFICCode != vfic {
		t.Errorf("VFICCode: got %q, want %q", got.VFICCode, vfic)
	}
}

func bytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
