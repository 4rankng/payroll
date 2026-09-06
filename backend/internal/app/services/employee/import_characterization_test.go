package employee

import (
	"context"
	"testing"

	"github.com/xuri/excelize/v2"
)

// Phase 0 characterization (excel-parsing-refactor): the employee import
// parser had zero coverage. These tests lock CURRENT behavior — including
// quirks (blind first sheet, fixed B..I offsets, silent row skipping) that
// later phases must preserve while adopting shared primitives.

func newEmployeeImportWorkbook(t *testing.T) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	const sheet = "Sheet1"

	// Header row (blindly skipped by the parser).
	headers := []string{"STT", "CCCD", "Dự án", "Vị trí", "SĐT", "Họ và tên", "Chủ tài khoản", "Số TK", "Ngân hàng"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			t.Fatal(err)
		}
	}

	rows := [][]any{
		// Valid full row.
		{1, "012345678901", "EVA", "Công nhân", "0901234567", "Nguyễn Văn A", "NGUYEN VAN A", "1234567890", "MB Bank"},
		// Missing CCCD → silently skipped (current behavior, locked).
		{2, "", "EVA", "Công nhân", "0912345678", "Nguyễn Văn B", "", "", ""},
		// Missing Fullname → silently skipped (current behavior, locked).
		{3, "022233344455", "EVA", "", "0923456789", "", "", "", ""},
		// Sparse row: defaults position to "phổ thông", empty optionals.
		{4, "033344455566", "", "", "", "Trần Thị C", "", "", ""},
		// 16-char mobile is truncated to 15 (current behavior, locked).
		{5, "044455566677", "", "", "0901234567890123", "Lê Văn D", "", "", ""},
		// Column A empty → row skipped before parse.
		{"", "055566677788", "EVA", "", "", "Phạm Văn E", "", "", ""},
	}
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	return f
}

func TestCharacterization_EmployeeImportParse(t *testing.T) {
	svc := NewImportService(nil, nil, nil, nil, nil) // parse-only; deps unused on this path
	f := newEmployeeImportWorkbook(t)
	defer func() { _ = f.Close() }()

	got, total, err := svc.ParseEmployeeExcel(context.Background(), f)
	if err != nil {
		t.Fatalf("ParseEmployeeExcel: %v", err)
	}

	// Rows 2 (full), 5 (sparse), 6 (long mobile) parse; rows 3, 4, 7 are skipped.
	if total != 3 || len(got) != 3 {
		t.Fatalf("parsed %d rows (total=%d), want 3: %+v", len(got), total, got)
	}

	full := got[0]
	if full.CCCD != "012345678901" || full.Fullname != "Nguyễn Văn A" ||
		full.ProjectCode != "EVA" || full.Position != "Công nhân" ||
		full.Mobile != "0901234567" || full.BankAccountName != "NGUYEN VAN A" ||
		full.BankAccount != "1234567890" || full.BankName != "MB Bank" {
		t.Fatalf("full row mismatch: %+v", full)
	}

	sparse := got[1]
	if sparse.Position != "phổ thông" {
		t.Fatalf("sparse row position = %q, want default %q", sparse.Position, "phổ thông")
	}
	if sparse.Mobile != "" || sparse.BankAccount != "" {
		t.Fatalf("sparse row should have empty optionals: %+v", sparse)
	}

	longMobile := got[2]
	if want := "090123456789012"; longMobile.Mobile != want {
		t.Fatalf("mobile = %q, want truncated %q", longMobile.Mobile, want)
	}
}

// Locked quirk: empty workbook / no data rows → error, not empty success.
func TestCharacterization_EmployeeImportEmptyWorkbook(t *testing.T) {
	svc := NewImportService(nil, nil, nil, nil, nil)
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	if _, _, err := svc.ParseEmployeeExcel(context.Background(), f); err == nil {
		t.Fatal("expected error on workbook with no data rows")
	}
}
