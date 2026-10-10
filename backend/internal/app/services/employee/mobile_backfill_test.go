package employee

import (
	"bytes"
	"testing"

	"api-server/internal/domain"

	"github.com/xuri/excelize/v2"
)

// buildBackfillWorkbook mirrors the /employees/export-paid-without-mobile
// layout (optionally with an extra or reordered column) so parser tests run
// against the real shape the admin uploads.
func buildBackfillWorkbook(t *testing.T, headers []string, dataRows [][]interface{}) []byte {
	t.Helper()
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	for i, h := range headers {
		if err := f.SetCellValue(sheet, cell(i, 1), h); err != nil {
			t.Fatalf("set header: %v", err)
		}
	}
	for r, row := range dataRows {
		for c, v := range row {
			if err := f.SetCellValue(sheet, cell(c, r+2), v); err != nil {
				t.Fatalf("set cell: %v", err)
			}
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("write workbook: %v", err)
	}
	return buf.Bytes()
}

func cell(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col+1, row)
	return name
}

func TestParseMobileBackfillExcel(t *testing.T) {
	headers := []string{"STT", "Họ và tên", "CCCD", "Lần trả lương cuối", "Dự án", "Mobile"}
	rows := [][]interface{}{
		{1, "Nguyễn Văn A", "031098001111", "02/10/2026", "Dự án 1", "0987 654 321"},
		{2, "Trần Thị B", "031098002222", "02/10/2026", "Dự án 2", "0908-765-432"},
		{3, "Lê Văn C", "031098003333", "02/10/2026", "Dự án 3", ""},          // blank mobile: dropped
		{4, "", "", "", "", ""},                                               // fully blank row
		{5, "Phạm D", "", "02/10/2026", "Dự án 4", "0912345678"},              // missing CCCD: row error
		{6, "Hoàng E", "031098004444", "02/10/2026", "Dự án 5", " 099.887.766 "}, // separators stripped
	}

	parsed, rowErrs, err := ParseMobileBackfillExcel(buildBackfillWorkbook(t, headers, rows))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rowErrs) != 1 || rowErrs[0].RowNumber != 6 {
		t.Fatalf("expected one row error at sheet row 6 (missing CCCD), got %+v", rowErrs)
	}
	want := []MobileBackfillRow{
		{RowNumber: 2, CCCD: "031098001111", Mobile: "0987654321"},
		{RowNumber: 3, CCCD: "031098002222", Mobile: "0908765432"},
		{RowNumber: 7, CCCD: "031098004444", Mobile: "099887766"},
	}
	if len(parsed) != len(want) {
		t.Fatalf("parsed %d rows, want %d: %+v", len(parsed), len(want), parsed)
	}
	for i, w := range want {
		if parsed[i] != w {
			t.Errorf("row %d: got %+v, want %+v", i, parsed[i], w)
		}
	}
}

func TestParseMobileBackfillExcelHeaderVariants(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		ok      bool
	}{
		{"reordered", []string{"Mobile", "Họ và tên", "CCCD"}, true},
		{"sdt alias", []string{"STT", "CCCD", "SĐT"}, true},
		{"missing mobile", []string{"STT", "CCCD", "Dự án"}, false},
		{"missing cccd", []string{"STT", "Họ và tên", "Mobile"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := [][]interface{}{{1, "A", "031098001111", "x", "0987654321"}}
			_, _, err := ParseMobileBackfillExcel(buildBackfillWorkbook(t, tt.headers, rows))
			if tt.ok && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatal("expected header error, got nil")
			}
		})
	}
}

func emp(id uint, cccd, mobile string) *domain.Employee {
	return &domain.Employee{ID: id, CCCD: cccd, Mobile: mobile}
}

func TestClassifyMobileBackfill(t *testing.T) {
	byCCCD := map[string][]*domain.Employee{
		"031098001111": {emp(1, "031098001111", "")},          // empty → fill
		"031098002222": {emp(2, "031098002222", "0911111222")}, // has one → skip
		"031098008888": {emp(8, "031098008888", "")},           // candidate for an owned number
		"031098006666": {emp(6, "031098006666", "")},           // candidate for an invalid number
		"031098007777": {emp(7, "031098007777", "")},           // candidate for a CCCD-as-mobile
	}
	ownersByMobile := map[string][]*domain.Employee{
		"0933333333": {emp(9, "031098009999", "0933333333")}, // owned by someone else
	}

	rows := []MobileBackfillRow{
		{RowNumber: 2, CCCD: "031098001111", Mobile: "0987654321"}, // fill
		{RowNumber: 3, CCCD: "031098002222", Mobile: "0988777666"}, // skip existing
		{RowNumber: 4, CCCD: "031098003333", Mobile: "0987654322"}, // not found
		{RowNumber: 5, CCCD: "031098008888", Mobile: "0933333333"}, // owned by other → error
		{RowNumber: 6, CCCD: "031098006666", Mobile: "09ab"},        // invalid → error
		{RowNumber: 7, CCCD: "031098007777", Mobile: "031098001111"}, // 12-digit CCCD → error
	}

	updates, result := classifyMobileBackfill(rows, byCCCD, ownersByMobile)

	if result.TotalRows != len(rows) {
		t.Errorf("TotalRows = %d, want %d", result.TotalRows, len(rows))
	}
	if result.Updated != 1 || result.SkippedExisting != 1 || result.NotFound != 1 {
		t.Errorf("counts: updated=%d skipped=%d notFound=%d, want 1/1/1", result.Updated, result.SkippedExisting, result.NotFound)
	}
	if result.ErrorCount != 3 {
		t.Errorf("ErrorCount = %d, want 3: %+v", result.ErrorCount, result.Errors)
	}
	if len(updates) != 1 || updates[0].EmployeeID != 1 || updates[0].Mobile != "0987654321" {
		t.Errorf("updates = %+v, want one write of 0987654321 to employee 1", updates)
	}
}

func TestClassifyMobileBackfillInFileConflicts(t *testing.T) {
	byCCCD := map[string][]*domain.Employee{
		"031098001111": {emp(1, "031098001111", "")},
		"031098002222": {emp(2, "031098002222", "")},
	}

	t.Run("same cccd same mobile applied once", func(t *testing.T) {
		rows := []MobileBackfillRow{
			{RowNumber: 2, CCCD: "031098001111", Mobile: "0987654321"},
			{RowNumber: 3, CCCD: "031098001111", Mobile: "0987654321"},
		}
		updates, result := classifyMobileBackfill(rows, byCCCD, nil)
		if result.Updated != 1 || result.ErrorCount != 0 || len(updates) != 1 {
			t.Errorf("updated=%d errors=%d updates=%d, want 1/0/1", result.Updated, result.ErrorCount, len(updates))
		}
	})

	t.Run("same cccd different mobiles rejected", func(t *testing.T) {
		rows := []MobileBackfillRow{
			{RowNumber: 2, CCCD: "031098002222", Mobile: "0987654321"},
			{RowNumber: 3, CCCD: "031098002222", Mobile: "0988777666"},
		}
		updates, result := classifyMobileBackfill(rows, byCCCD, nil)
		if result.Updated != 1 || result.ErrorCount != 1 || len(updates) != 1 {
			t.Errorf("updated=%d errors=%d updates=%d, want 1/1/1", result.Updated, result.ErrorCount, len(updates))
		}
	})

	// One typo'd number must not be handed to two different people within a
	// single upload — the DB-owner snapshot cannot catch this on its own.
	t.Run("same mobile different cccds rejected", func(t *testing.T) {
		rows := []MobileBackfillRow{
			{RowNumber: 2, CCCD: "031098001111", Mobile: "0987654321"},
			{RowNumber: 3, CCCD: "031098002222", Mobile: "0987654321"},
		}
		updates, result := classifyMobileBackfill(rows, byCCCD, nil)
		if result.Updated != 1 || result.ErrorCount != 1 || len(updates) != 1 {
			t.Errorf("updated=%d errors=%d updates=%d, want 1/1/1", result.Updated, result.ErrorCount, len(updates))
		}
	})
}

func TestClassifyMobileBackfillDuplicateCCCDFillsAll(t *testing.T) {
	// Soft-delete edge: two active rows sharing a CCCD both get filled.
	byCCCD := map[string][]*domain.Employee{
		"031098001111": {emp(1, "031098001111", ""), emp(2, "031098001111", "")},
	}
	rows := []MobileBackfillRow{{RowNumber: 2, CCCD: "031098001111", Mobile: "0987654321"}}
	updates, result := classifyMobileBackfill(rows, byCCCD, nil)
	if result.Updated != 2 || len(updates) != 2 {
		t.Errorf("updated=%d updates=%d, want 2/2", result.Updated, len(updates))
	}
}

// The October 2026 manual pass left exactly these three employees unfilled
// because their numbers were already owned by other employees — the import
// must reach the same verdict.
func TestClassifyMobileBackfillOctoberHoldouts(t *testing.T) {
	byCCCD := map[string][]*domain.Employee{
		"031192003914": {emp(10, "031192003914", "")},
	}
	ownersByMobile := map[string][]*domain.Employee{
		"0383199366": {emp(11, "031098008888", "0383199366")},
	}
	rows := []MobileBackfillRow{{RowNumber: 2, CCCD: "031192003914", Mobile: "0383199366"}}

	updates, result := classifyMobileBackfill(rows, byCCCD, ownersByMobile)
	if len(updates) != 0 || result.ErrorCount != 1 || result.Updated != 0 {
		t.Errorf("want no write and one error, got updates=%d result=%+v", len(updates), result)
	}
	if result.Errors[0].CCCD != "031192003914" {
		t.Errorf("error should name the row CCCD, got %+v", result.Errors[0])
	}
}

// The owner confirmed keeping a 9-digit number in the data — validation must
// accept it (digits-only rule), unlike a VN-prefix normalizer would.
func TestClassifyMobileBackfillAcceptsOwnerConfirmedShortNumber(t *testing.T) {
	byCCCD := map[string][]*domain.Employee{
		"031098001111": {emp(1, "031098001111", "")},
	}
	rows := []MobileBackfillRow{{RowNumber: 2, CCCD: "031098001111", Mobile: "031991725"}}
	updates, result := classifyMobileBackfill(rows, byCCCD, nil)
	if result.Updated != 1 || len(updates) != 1 || updates[0].Mobile != "031991725" {
		t.Errorf("9-digit owner-confirmed number must fill: %+v", result)
	}
}
