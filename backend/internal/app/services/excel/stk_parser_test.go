package excel

import (
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseSTKSheet(t *testing.T) {
	// Create an in-memory excel file
	f := excelize.NewFile()

	// Create STK sheet
	sheetName := "STK"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		t.Fatalf("failed to create sheet: %v", err)
	}
	f.SetActiveSheet(index)

	// Write header row (row 3)
	_ = f.SetCellValue(sheetName, "A3", "STT")
	_ = f.SetCellValue(sheetName, "B3", "ID")
	_ = f.SetCellValue(sheetName, "C3", "Tên")
	_ = f.SetCellValue(sheetName, "D3", "Stk")
	_ = f.SetCellValue(sheetName, "E3", "Tên Ngân hàng")
	_ = f.SetCellValue(sheetName, "F3", "Ghi chú")
	_ = f.SetCellValue(sheetName, "G3", "SĐT")

	// Write data rows (row 4 & 5)
	_ = f.SetCellValue(sheetName, "A4", "1")
	_ = f.SetCellValue(sheetName, "B4", "027202000029")
	_ = f.SetCellValue(sheetName, "C4", "Nguyễn Sỹ Hùng")
	_ = f.SetCellValue(sheetName, "D4", "296608866")
	_ = f.SetCellValue(sheetName, "E4", "MB")
	_ = f.SetCellValue(sheetName, "F4", "")
	_ = f.SetCellValue(sheetName, "G4", "0987654321")

	_ = f.SetCellValue(sheetName, "A5", "2")
	_ = f.SetCellValue(sheetName, "B5", "031094015626")
	_ = f.SetCellValue(sheetName, "C5", "Phạm Văn Khương")
	_ = f.SetCellValue(sheetName, "D5", "0356755184")
	_ = f.SetCellValue(sheetName, "E5", "MB")
	_ = f.SetCellValue(sheetName, "F5", "some note")
	_ = f.SetCellValue(sheetName, "G5", "0981 234 567")

	// Parse
	rows, err := ParseSTKSheet(f)
	if err != nil {
		t.Fatalf("ParseSTKSheet returned unexpected error: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 parsed rows, got %d", len(rows))
	}

	row1 := rows[0]
	if row1.CCCD != "027202000029" || row1.FullName != "Nguyễn Sỹ Hùng" || row1.BankAccount != "296608866" || row1.BankName != "MB" || row1.Note != "" || row1.Mobile != "0987654321" {
		t.Errorf("row 1 parsed incorrectly: %+v", row1)
	}

	row2 := rows[1]
	if row2.CCCD != "031094015626" || row2.FullName != "Phạm Văn Khương" || row2.BankAccount != "0356755184" || row2.BankName != "MB" || row2.Note != "some note" || row2.Mobile != "0981234567" {
		t.Errorf("row 2 parsed incorrectly: %+v", row2)
	}
}

// TestParseSTKSheet_WithBCClgdFile exercises the parser against the real
// WeeklyBCC fixture. The LGD STK sheet has a merged title on row 1, column
// headers on row 2, and a single data row on row 3 — a layout the old
// "data always starts at row 4" assumption silently dropped (returning 0 rows,
// so no bank account was ever created for the employee).
func TestParseSTKSheet_WithBCClgdFile(t *testing.T) {
	f, err := excelize.OpenFile(weeklyBCCFixture(t))
	if err != nil {
		t.Fatalf("failed to open fixture: %v", err)
	}
	defer func() { _ = f.Close() }()

	rows, err := ParseSTKSheet(f)
	if err != nil {
		t.Fatalf("ParseSTKSheet returned unexpected error: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 parsed STK row from LGD fixture, got %d", len(rows))
	}

	r := rows[0]
	if r.CCCD != "031092020742" {
		t.Errorf("CCCD: want 031092020742, got %q", r.CCCD)
	}
	if r.FullName != "Trần Đăng Đức" {
		t.Errorf("FullName: want Trần Đăng Đức, got %q", r.FullName)
	}
	if r.BankAccount != "02101010890101" {
		t.Errorf("BankAccount: want 02101010890101, got %q", r.BankAccount)
	}
	if r.BankName != "MSB" {
		t.Errorf("BankName: want MSB, got %q", r.BankName)
	}
	if r.Mobile != "" {
		t.Errorf("Mobile: want empty (no col G in fixture), got %q", r.Mobile)
	}
}

func TestParseSTKSheet_NoSheet(t *testing.T) {
	f := excelize.NewFile()
	rows, err := ParseSTKSheet(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows != nil {
		t.Errorf("expected nil rows when sheet not found, got %v", rows)
	}
}

// TestParseSTKSheet_TrailingSpaceName guards against the regression where a
// sheet exported as "STK " (trailing/leading whitespace, any case) was missed
// by the exact-name check, causing ParseSTKSheet to return zero rows and every
// STK-only employee to be reported as "not found in system" on BCC upload.
func TestParseSTKSheet_TrailingSpaceName(t *testing.T) {
	cases := []string{"STK ", " STK", " stk ", "Stk"}
	for _, sheetName := range cases {
		t.Run(sheetName, func(t *testing.T) {
			f := excelize.NewFile()
			idx, err := f.NewSheet(sheetName)
			if err != nil {
				t.Fatalf("NewSheet: %v", err)
			}
			f.SetActiveSheet(idx)
			_ = f.SetCellValue(sheetName, "B3", "ID")
			_ = f.SetCellValue(sheetName, "C3", "Tên")
			_ = f.SetCellValue(sheetName, "D3", "Stk")
			_ = f.SetCellValue(sheetName, "B4", "011207000333")
			_ = f.SetCellValue(sheetName, "C4", "Quàng Văn Hùng")
			_ = f.SetCellValue(sheetName, "D4", "102885394034")
			_ = f.SetCellValue(sheetName, "E4", "Vietinbank")

			rows, err := ParseSTKSheet(f)
			if err != nil {
				t.Fatalf("ParseSTKSheet: %v", err)
			}
			if len(rows) != 1 {
				t.Fatalf("sheet %q: expected 1 row, got %d", sheetName, len(rows))
			}
			if rows[0].CCCD != "011207000333" || rows[0].FullName != "Quàng Văn Hùng" {
				t.Errorf("sheet %q: unexpected row %+v", sheetName, rows[0])
			}
			if rows[0].Mobile != "" {
				t.Errorf("sheet %q: Mobile: want empty (no col G), got %q", sheetName, rows[0].Mobile)
			}
		})
	}
}

func TestSanitizeMobile(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"already digits", "0987654321", "0987654321"},
		{"spaces stripped", "0981 234 567", "0981234567"},
		{"plus and dashes", "+84-981-234-567", "84981234567"},
		{"parens and spaces", "(0981) 234 567", "0981234567"},
		{"over 50 chars", strings.Repeat("1", 51), ""},
		{"exactly 50", strings.Repeat("1", 50), strings.Repeat("1", 50)},
		{"leading zeros preserved", "0912345678", "0912345678"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeMobile(tc.in)
			if got != tc.want {
				t.Errorf("sanitizeMobile(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
