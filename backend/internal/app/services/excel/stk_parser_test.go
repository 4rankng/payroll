package excel

import (
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

	// Write data rows (row 4 & 5)
	_ = f.SetCellValue(sheetName, "A4", "1")
	_ = f.SetCellValue(sheetName, "B4", "027202000029")
	_ = f.SetCellValue(sheetName, "C4", "Nguyễn Sỹ Hùng")
	_ = f.SetCellValue(sheetName, "D4", "296608866")
	_ = f.SetCellValue(sheetName, "E4", "MB")
	_ = f.SetCellValue(sheetName, "F4", "")

	_ = f.SetCellValue(sheetName, "A5", "2")
	_ = f.SetCellValue(sheetName, "B5", "031094015626")
	_ = f.SetCellValue(sheetName, "C5", "Phạm Văn Khương")
	_ = f.SetCellValue(sheetName, "D5", "0356755184")
	_ = f.SetCellValue(sheetName, "E5", "MB")
	_ = f.SetCellValue(sheetName, "F5", "some note")

	// Parse
	rows, err := ParseSTKSheet(f)
	if err != nil {
		t.Fatalf("ParseSTKSheet returned unexpected error: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 parsed rows, got %d", len(rows))
	}

	row1 := rows[0]
	if row1.CCCD != "027202000029" || row1.FullName != "Nguyễn Sỹ Hùng" || row1.BankAccount != "296608866" || row1.BankName != "MB" || row1.Note != "" {
		t.Errorf("row 1 parsed incorrectly: %+v", row1)
	}

	row2 := rows[1]
	if row2.CCCD != "031094015626" || row2.FullName != "Phạm Văn Khương" || row2.BankAccount != "0356755184" || row2.BankName != "MB" || row2.Note != "some note" {
		t.Errorf("row 2 parsed incorrectly: %+v", row2)
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
