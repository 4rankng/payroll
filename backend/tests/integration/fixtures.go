package main

import (
	"os"

	"github.com/xuri/excelize/v2"
)

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

// buildPourinWeeklyPaymentFixture mirrors the POURIN full-month template:
// identity columns shifted right by "Ngày đi làm"/"Ngày nghỉ việc" (D/E), Bộ
// phận at F, the salary header spelled "Vị trí lương/8h" at G with the tier
// stored numeric under the #,##0 display format, TC shift codes, and an STK
// sheet whose headers start on row 2. The tier value 800000 must equal the
// project payrate position the import should price against.
func buildPourinWeeklyPaymentFixture() (string, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if err := f.SetSheetName("Sheet1", "Chấm công"); err != nil {
		return "", err
	}
	for cell, value := range map[string]any{
		"A8": "STT", "B8": "Mã nhân viên", "C8": "Họ và tên",
		"D8": "Ngày đi làm", "E8": "Ngày nghỉ việc", "F8": "Bộ phận",
		"G8": "Vị trí lương/8h",
		"H8": 27, "I8": 28, "H9": "T1", "I9": "T2",
		"H10": "HC", "I10": "TC",
		"A11": 1, "B11": "099260900602", "C11": "Nhân viên kiểm thử POURIN",
		"F11": "Phân xưởng Trang trí", "G11": 800000,
		"H11": 8, "I11": 4,
	} {
		if err := f.SetCellValue("Chấm công", cell, value); err != nil {
			return "", err
		}
	}
	numFmt := "#,##0"
	styleID, err := f.NewStyle(&excelize.Style{CustomNumFmt: &numFmt})
	if err != nil {
		return "", err
	}
	if err := f.SetCellStyle("Chấm công", "G11", "G11", styleID); err != nil {
		return "", err
	}
	if _, err := f.NewSheet("STK"); err != nil {
		return "", err
	}
	headers := []any{"Stt", "Mã nhân viên", "Tên", "Stk", "Ngân Hàng", "sdt"}
	row := []any{1, "099260900602", "Nhân viên kiểm thử POURIN", "100000900602", "VCB", "0909900602"}
	if err := f.SetSheetRow("STK", "A2", &headers); err != nil {
		return "", err
	}
	if err := f.SetSheetRow("STK", "A3", &row); err != nil {
		return "", err
	}
	temp, err := os.CreateTemp("", "payroll-pourin-weekly-payment-*.xlsx")
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
