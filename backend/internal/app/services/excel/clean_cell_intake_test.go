package excel

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

// Regression: partner files escape cells as text with a leading apostrophe or
// double quote ("'030172019748"). Those quotes used to ride through the
// parsers into CCCD lookups, employee auto-creation (rejected by CCCD
// validation), and the STK cross-check, so rows for employees who plainly
// exist reported "không tìm thấy nhân viên". Both intake paths — bccCell
// (→ excelkit.Cell) for the BCC/position sheets and stkCell (→ excelkit.Clean)
// for the STK sheet — must strip them.
func TestParseStripsTextEscapeQuotes(t *testing.T) {
	t.Run("position sheet", func(t *testing.T) {
		f := excelize.NewFile()
		defer func() { _ = f.Close() }()
		const sheet = "Chia chọn"
		if err := f.SetSheetName("Sheet1", sheet); err != nil {
			t.Fatal(err)
		}
		// parsePositionSheet layout: row 4 headers, row 5 rates, rows 7+ data.
		rows := [][]interface{}{
			{},                                                             // row 1
			{},                                                             // row 2
			{},                                                             // row 3
			{0: "STT", 1: "Mã nhân viên", 2: "Họ và tên", 3: 8, 4: 9, 5: "Tổng"}, // row 4
			{3: 50000, 4: 50000},                                           // row 5 rates
			{},                                                             // row 6
			{0: 1, 1: "'030172019748", 2: "'Lại Thị Hải Hà", 3: 8, 4: 7.5},       // row 7
			{0: 2, 1: `"033072000021"`, 2: `"Lương Đức Phong"`, 4: 8},            // row 8
		}
		setRows(t, f, sheet, rows)

		parsed, err := parsePositionSheet(f, sheet)
		if err != nil {
			t.Fatalf("parsePositionSheet: %v", err)
		}
		if len(parsed.Employees) != 2 {
			t.Fatalf("employees = %d, want 2 (%+v)", len(parsed.Employees), parsed.Employees)
		}
		first, second := parsed.Employees[0], parsed.Employees[1]
		if first.EmployeeCode != "030172019748" || first.FullName != "Lại Thị Hải Hà" {
			t.Errorf("first = %q / %q, want clean identity", first.EmployeeCode, first.FullName)
		}
		if second.EmployeeCode != "033072000021" || second.FullName != "Lương Đức Phong" {
			t.Errorf("second = %q / %q, want clean identity", second.EmployeeCode, second.FullName)
		}
		if len(first.Entries) != 2 {
			t.Fatalf("first entries = %d, want 2", len(first.Entries))
		}
	})

	t.Run("STK sheet", func(t *testing.T) {
		f := excelize.NewFile()
		defer func() { _ = f.Close() }()
		const sheet = "STK"
		if err := f.SetSheetName("Sheet1", sheet); err != nil {
			t.Fatal(err)
		}
		// Samsung-SDS-style labelled layout.
		rows := [][]interface{}{
			{0: "TÊN", 1: "SỐ CCCD", 2: "TÊN NGÂN HÀNG", 3: "SỐ TÀI KHOẢN"},
			{0: "'Lại Thị Hải Hà", 1: "'030172019748", 2: "'Techcombank", 3: "'0031000539909"},
		}
		setRows(t, f, sheet, rows)

		stkRows, err := ParseSTKSheet(f)
		if err != nil {
			t.Fatalf("ParseSTKSheet: %v", err)
		}
		if len(stkRows) != 1 {
			t.Fatalf("stk rows = %d, want 1", len(stkRows))
		}
		row := stkRows[0]
		if row.CCCD != "030172019748" || row.FullName != "Lại Thị Hải Hà" ||
			row.BankName != "Techcombank" || row.BankAccount != "0031000539909" {
			t.Errorf("row = %+v, want clean identity and bank fields", row)
		}
	})
}

// setRows writes a sparse row/column map (0-based indexes) into the sheet.
func setRows(t *testing.T, f *excelize.File, sheet string, rows [][]interface{}) {
	t.Helper()
	for i, r := range rows {
		for j, v := range r {
			if v == nil || v == "" {
				continue
			}
			cn, err := excelize.CoordinatesToCellName(j+1, i+1)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.SetCellValue(sheet, cn, v); err != nil {
				t.Fatal(err)
			}
		}
	}
}
