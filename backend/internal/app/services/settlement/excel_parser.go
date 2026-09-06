package settlement

import (
	"api-server/internal/constants"
	"fmt"
	"mime/multipart"
	"slices"
	"strconv"
	"strings"

	"api-server/internal/domain"
	"api-server/internal/pkg/excelkit"

	"github.com/xuri/excelize/v2"
)

// Excel sheet and cell configuration
const (
	SheetPayrollReport = "Payroll Report"
	SheetSummary       = "Summary"
	SheetInternal      = "INTERNAL"

	LabelSettlementAmount = "Tiền CTy Phải Trả"
	ColumnLabel           = "D"
	ColumnAmount          = "E"
	SearchMaxRow          = 10

	ColumnTimesheetID = 0 // Column A

	// InternalTypePrefix is half of a writer/reader contract: payroll's
	// report_internal_sheet.go writes "type:<kind>" into the INTERNAL sheet
	// A1 cell; this parser routes on the prefix when re-importing sao-kê
	// exports. Change both together.
	InternalTypePrefix = "type:"
)

// SettlementExcelParser handles parsing of settlement Excel files
type SettlementExcelParser struct{}

// NewSettlementExcelParser creates a new Excel parser
func NewSettlementExcelParser() *SettlementExcelParser {
	return &SettlementExcelParser{}
}

// ParseSettlementFile parses the settlement Excel file and extracts key information
func (p *SettlementExcelParser) ParseSettlementFile(
	fileHeader *multipart.FileHeader,
) (*SettlementFileData, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, domain.NewValidationError(constants.MsgCannotOpenFileVN)
	}
	defer func() {
		_ = file.Close()
	}()

	excelFile, err := excelkit.OpenReader(file)
	if err != nil {
		return nil, domain.NewValidationError(constants.MsgCannotReadExcelFileVN)
	}
	defer func() {
		_ = excelFile.Close()
	}()

	data := &SettlementFileData{}

	// Extract settlement amount
	data.SettlementAmount, err = p.getSettlementAmount(excelFile)
	if err != nil {
		return nil, err
	}

	// Check for INTERNAL sheet and extract type + IDs if present
	if p.hasSheet(excelFile, SheetInternal) {
		data.HasInternalSheet = true
		data.FileType, data.TimesheetIDs, err = p.getInternalSheetData(excelFile)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

// SettlementFileData holds parsed data from settlement Excel file
type SettlementFileData struct {
	SettlementAmount int64
	HasInternalSheet bool
	TimesheetIDs     []uint
	FileType         string // "timesheet", "advance_payment", or "" (legacy files)
}

// IsAdvancePayment returns true if the file is an advance payment sao ke
func (d *SettlementFileData) IsAdvancePayment() bool {
	return d.FileType == "advance_payment"
}

// hasSheet checks if the Excel file contains a specific sheet
func (p *SettlementExcelParser) hasSheet(f *excelize.File, sheetName string) bool {
	return slices.Contains(f.GetSheetList(), sheetName)
}

// getSettlementAmount scans column D for the settlement label and reads the amount from column E.
func (p *SettlementExcelParser) getSettlementAmount(f *excelize.File) (int64, error) {
	for _, sheetName := range []string{SheetPayrollReport, SheetSummary} {
		if !p.hasSheet(f, sheetName) {
			continue
		}
		amount, err := p.findAmountByLabel(f, sheetName)
		if err != nil {
			return 0, err
		}
		if amount > 0 {
			return amount, nil
		}
	}

	return 0, domain.NewValidationError(
		fmt.Sprintf("không tìm thấy nhãn '%s' trong cột D của sheet '%s' hoặc '%s'",
			LabelSettlementAmount, SheetPayrollReport, SheetSummary))
}

// findAmountByLabel scans rows 1–SearchMaxRow in column D for the settlement label,
// then reads the corresponding column E value.
func (p *SettlementExcelParser) findAmountByLabel(f *excelize.File, sheetName string) (int64, error) {
	for row := 1; row <= SearchMaxRow; row++ {
		labelCell := fmt.Sprintf("%s%d", ColumnLabel, row)
		label, err := f.GetCellValue(sheetName, labelCell)
		if err != nil {
			continue
		}
		if !strings.Contains(label, LabelSettlementAmount) {
			continue
		}

		amountCell := fmt.Sprintf("%s%d", ColumnAmount, row)
		value, err := f.GetCellValue(sheetName, amountCell)
		if err != nil {
			return 0, domain.NewValidationError(
				fmt.Sprintf("không thể đọc cell %s từ sheet '%s': %v", amountCell, sheetName, err))
		}
		return parseCurrency(value)
	}
	return 0, nil
}

// getInternalSheetData reads the INTERNAL sheet, extracting the type header and IDs.
func (p *SettlementExcelParser) getInternalSheetData(f *excelize.File) (string, []uint, error) {
	if !p.hasSheet(f, SheetInternal) {
		return "", nil, domain.NewValidationError(
			fmt.Sprintf("không tìm thấy sheet '%s' trong file", SheetInternal))
	}

	rows, err := f.GetRows(SheetInternal)
	if err != nil {
		return "", nil, domain.NewValidationError(
			fmt.Sprintf("không thể đọc sheet '%s': %v", SheetInternal, err))
	}

	var fileType string
	var ids []uint
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}

		cellValue := strings.TrimSpace(row[ColumnTimesheetID])
		if cellValue == "" {
			continue
		}

		// Extract type from header row (e.g., "type:advance_payment" → "advance_payment")
		if typ, ok := strings.CutPrefix(cellValue, InternalTypePrefix); ok {
			fileType = typ
			continue
		}

		id, err := strconv.ParseUint(cellValue, 10, 32)
		if err != nil {
			return "", nil, domain.NewValidationError(
				fmt.Sprintf("cell A%d chứa giá trị không hợp lệ '%s': phải là số nguyên",
					i+1, cellValue))
		}

		ids = append(ids, uint(id))
	}

	return fileType, ids, nil
}

// parseCurrency parses a currency string (e.g., "1,234,567" or "1234567") to int64
func parseCurrency(value string) (int64, error) {
	// Extract only numeric digits from the value
	var numericStr strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			numericStr.WriteRune(r)
		}
	}

	value = numericStr.String()

	if value == "" {
		return 0, domain.NewValidationError(constants.MsgEmptyCurrencyValueVN)
	}

	amount, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, domain.NewValidationError(
			fmt.Sprintf("không thể phân tích giá trị tiền tệ '%s': %v", value, err))
	}

	if amount <= 0 {
		return 0, domain.NewValidationError(
			fmt.Sprintf("số tiền phải lớn hơn 0, nhận được: %d", amount))
	}

	return amount, nil
}
