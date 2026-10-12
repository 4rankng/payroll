package main

import (
	"fmt"
	"os"
	"time"

	"github.com/xuri/excelize/v2"
)

// uniqueMobile generates a plausible Vietnamese mobile number unique per call.
func uniqueMobile() string {
	n := time.Now().UnixNano()
	d := n % 10
	if d == 5 {
		d = 7
	}
	return fmt.Sprintf("09%d%07d", d, n%10_000_000)
}

// buildWeeklyPaymentFixture supplies valid employee identity, bank details,
// weekday headers and two different shift types for the full import pipeline.
func buildWeeklyPaymentFixture() (string, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if err := f.SetSheetName("Sheet1", "520"); err != nil {
		return "", err
	}
	for cell, value := range map[string]any{
		"A8": "STT", "B8": "Mã nhân viên", "C8": "Họ và tên", "D8": "Bộ phận", "E8": "Lương 8h",
		"F8": 29, "G8": 30, "F9": "T4", "G9": "T5", "F10": "HC", "G10": "NN",
		"A11": 1, "B11": "099260900501", "C11": "Nhân viên kiểm thử bảng lương tuần", "E11": 520,
		"F11": 8, "G11": 8,
	} {
		if err := f.SetCellValue("520", cell, value); err != nil {
			return "", err
		}
	}
	if _, err := f.NewSheet("STK"); err != nil {
		return "", err
	}
	headers := []any{"STT", "ID", "Tên", "Stk", "Tên Ngân hàng", "Ghi chú", "SĐT"}
	row := []any{1, "099260900501", "Nhân viên kiểm thử bảng lương tuần", "100000900501", "VCB", "Synthetic QA", "0909900501"}
	if err := f.SetSheetRow("STK", "A3", &headers); err != nil {
		return "", err
	}
	if err := f.SetSheetRow("STK", "A4", &row); err != nil {
		return "", err
	}
	temp, err := os.CreateTemp("", "payroll-weekly-payment-*.xlsx")
	if err != nil {
		return "", err
	}
	name := temp.Name()
	if err := temp.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	if err := f.SaveAs(name); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}
