package excelkit

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/shakinm/xlsReader/xls"
	"github.com/xuri/excelize/v2"
)

// The legacy BIFF reader surfaces every cell as one of a handful of record
// types. Numbers and RK-packed numbers are written back as numeric cells so
// excelize's General rendering matches the .xlsx path (parseMoney accepts
// both "345000" and "345,000.00"); everything else is stored as text.
const (
	recordTypeNumber = "*record.Number"
	recordTypeRK     = "*record.Rk"
)

// openXLS converts a legacy BIFF (.xls, OLE2 compound document) workbook into
// an in-memory xlsx workbook, so every downstream excelize-based parser keeps
// working unchanged. Cell values are copied as values (no formulas, styles or
// merges): BIFF text cells come back as strings, numeric cells as floats.
//
// BIFF8 stores plain-Latin text as 8-bit "compressed" strings — the low
// bytes of UTF-16 code units, i.e. Latin-1. The reader returns those bytes
// unconverted, which are invalid UTF-8 (e.g. "Phí" surfaces as "Ph\xed"),
// so every string passes through repairLatin1 before being written. Formulas
// and rich-text records are skipped by the reader itself; OnePay statements
// are literal data dumps, so nothing is lost for the layouts we parse.
func openXLS(r io.Reader) (*excelize.File, error) {
	data, err := io.ReadAll(io.LimitReader(r, UnzipSizeLimit()+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > UnzipSizeLimit() {
		return nil, fmt.Errorf("xls workbook exceeds %d bytes", UnzipSizeLimit())
	}

	workbook, err := xls.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("open legacy xls: %w", err)
	}

	out := excelize.NewFile()
	sheetCount := workbook.GetNumberSheets()
	for i := 0; i < sheetCount; i++ {
		sheet, err := workbook.GetSheet(i)
		if err != nil {
			return nil, fmt.Errorf("read xls sheet %d: %w", i, err)
		}
		name := sheet.GetName()
		if i == 0 {
			if err := out.SetSheetName("Sheet1", name); err != nil {
				return nil, fmt.Errorf("rename xls sheet: %w", err)
			}
		} else {
			if _, err := out.NewSheet(name); err != nil {
				return nil, fmt.Errorf("create sheet %q: %w", name, err)
			}
		}
		if err := copyXLSSheet(out, name, sheet); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func copyXLSSheet(out *excelize.File, sheetName string, sheet *xls.Sheet) error {
	for r, row := range sheet.GetRows() {
		for c, cell := range row.GetCols() {
			coord, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				return fmt.Errorf("xls cell %d,%d: %w", r+1, c+1, err)
			}
			switch cell.GetType() {
			case recordTypeNumber, recordTypeRK:
				if err := out.SetCellValue(sheetName, coord, cell.GetFloat64()); err != nil {
					return fmt.Errorf("write xls cell %s: %w", coord, err)
				}
			default:
				value := repairLatin1(cell.GetString())
				if value == "" {
					continue
				}
				if err := out.SetCellStr(sheetName, coord, value); err != nil {
					return fmt.Errorf("write xls cell %s: %w", coord, err)
				}
			}
		}
	}
	return nil
}

// repairLatin1 re-encodes raw Latin-1 bytes (BIFF8 compressed strings) as
// UTF-8. Valid UTF-8 (BIFF8 UTF-16 strings already decoded by the reader)
// passes through untouched.
func repairLatin1(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	runes := make([]rune, 0, len(s))
	for i := 0; i < len(s); i++ {
		runes = append(runes, rune(s[i]))
	}
	return string(runes)
}
