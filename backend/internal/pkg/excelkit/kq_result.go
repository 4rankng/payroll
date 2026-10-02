package excelkit

import (
	"bytes"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

// KQ result workbook layout. This is the "Kết quả chuyển tiền" sheet the
// bank-result importer reads back through TransactionCodeStrategy: sheet
// `data`, rows 1-3 title block, row 4 column headers, data from row 5, nine
// columns A-I.
//
// The layout lives here — not in the wallet or payroll service — because two
// independent producers write it (the wallet batch KQ download and the
// "Tạo KQ CK" chuyenlo-file converter) and one importer parses it.
const (
	KQSheetName    = "data"
	kqTitleRow     = 1
	kqRefRow       = 2
	kqDateRow      = 3
	KQHeaderRow    = 4
	KQFirstDataRow = 5
	kqLastCol      = "I"
)

// KQHeaders is the row-4 header set of the KQ result workbook.
var KQHeaders = []string{
	"STT",
	"Số tài khoản",
	"Tên người thụ hưởng",
	"Ngân hàng thụ hưởng",
	"Số tiền",
	"Nội dung",
	"Phí",
	"Trạng thái",
	"FT / Ghi chú",
}

// KQRow is one transfer line of a KQ result workbook. Column G (Phí) and
// column I (FT / Ghi chú) are carried verbatim; the importer ignores G and
// reads I as the bank reference on success rows or the error message on
// failed rows.
type KQRow struct {
	Order          int
	AccountNumber  string
	AccountName    string
	BankName       string
	Amount         int64
	TransactionRef string
	Fee            int64
	Status         string
	Reference      string
}

// KQTitle carries the three title-block rows printed above the headers.
type KQTitle struct {
	Title    string
	Ref      string
	DateLine string
}

// WriteKQWorkbook renders rows into the KQ result workbook layout and returns
// the .xlsx bytes.
func WriteKQWorkbook(title KQTitle, rows []KQRow) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	if err := f.SetSheetName("Sheet1", KQSheetName); err != nil {
		return nil, fmt.Errorf("rename sheet: %w", err)
	}

	for cell, val := range map[string]any{
		fmt.Sprintf("A%d", kqTitleRow): title.Title,
		fmt.Sprintf("A%d", kqRefRow):   title.Ref,
		fmt.Sprintf("A%d", kqDateRow):  title.DateLine,
	} {
		if err := f.SetCellValue(KQSheetName, cell, val); err != nil {
			return nil, fmt.Errorf("set title cell %s: %w", cell, err)
		}
	}

	for i, h := range KQHeaders {
		cell, err := excelize.CoordinatesToCellName(i+1, KQHeaderRow)
		if err != nil {
			return nil, fmt.Errorf("header cell %d: %w", i, err)
		}
		if err := f.SetCellValue(KQSheetName, cell, h); err != nil {
			return nil, fmt.Errorf("set header %s: %w", cell, err)
		}
	}

	for idx, row := range rows {
		excelRow := KQFirstDataRow + idx
		cells := []struct {
			col string
			val any
		}{
			{"A", row.Order},
			{"B", row.AccountNumber},
			{"C", row.AccountName},
			{"D", row.BankName},
			{"E", row.Amount},
			{"F", row.TransactionRef},
			{"G", row.Fee},
			{"H", row.Status},
			{"I", row.Reference},
		}
		for _, c := range cells {
			cell := fmt.Sprintf("%s%d", c.col, excelRow)
			if err := f.SetCellValue(KQSheetName, cell, c.val); err != nil {
				return nil, fmt.Errorf("set cell %s: %w", cell, err)
			}
		}
	}

	widths := map[string]float64{
		"A": 6, "B": 22, "C": 28, "D": 36, "E": 18,
		"F": 40, "G": 12, "H": 16, "I": 28,
	}
	for col, w := range widths {
		if err := f.SetColWidth(KQSheetName, col, col, w); err != nil {
			return nil, fmt.Errorf("set col width %s: %w", col, err)
		}
	}

	lastRow := KQFirstDataRow + len(rows) - 1
	if lastRow < KQHeaderRow {
		lastRow = KQHeaderRow
	}
	if err := f.SetSheetDimension(KQSheetName, fmt.Sprintf("A1:%s%d", kqLastCol, lastRow)); err != nil {
		return nil, fmt.Errorf("set sheet dimension: %w", err)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write xlsx: %w", err)
	}
	return buf.Bytes(), nil
}

// KQDateLine renders the Vietnamese date stamp used by the title block.
func KQDateLine(now time.Time) string {
	return fmt.Sprintf("Ngày: %s", now.Format("02/01/2006 15:04:05"))
}
