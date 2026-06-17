package excel

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// STKRow represents parsed bank account information for an employee from the STK sheet
type STKRow struct {
	CCCD        string
	FullName    string
	BankAccount string
	BankName    string
	Note        string
}

// ParseSTKSheet searches for and parses the STK sheet in the provided Excel file.
// Returns a slice of STKRow. If the sheet is not found, returns (nil, nil).
func ParseSTKSheet(f *excelize.File) ([]STKRow, error) {
	// Match case-insensitively after trimming: partners frequently export the
	// STK sheet with surrounding whitespace (e.g. "STK ", " stk"), which would
	// otherwise miss the exact match and silently yield zero rows — so no
	// employee gets created from STK and the matching BCC rows report
	// "nhân viên không tìm thấy trong hệ thống".
	sheetName := ""
	for _, sheet := range f.GetSheetList() {
		if strings.EqualFold(strings.TrimSpace(sheet), "STK") {
			sheetName = sheet
			break
		}
	}
	if sheetName == "" {
		return nil, nil
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows from STK sheet: %w", err)
	}

	if len(rows) < 2 {
		return nil, nil
	}

	// STK sheet layouts vary: some have a merged title on row 1 (so the column
	// header lands on row 2 and data on row 3), others place the header on
	// row 3 with data from row 4. Detect the header row by its labels rather
	// than assuming a fixed offset, otherwise a shifted layout silently yields
	// zero rows and no bank account is created.
	headerRow := -1
	scanLimit := min(len(rows), 10)
	for i := range scanLimit {
		row := rows[i]
		colB, colC, colD := "", "", ""
		if len(row) > 1 {
			colB = normHeader(row[1])
		}
		if len(row) > 2 {
			colC = normHeader(row[2])
		}
		if len(row) > 3 {
			colD = normHeader(row[3])
		}
		// Match the identity / bank-account header labels (any Vietnamese variant).
		isIDCol := colB == "id" || colB == "cccd" || strings.Contains(colB, "mã nhân viên")
		isNameCol := colC == "tên" || strings.Contains(colC, "họ tên") || strings.Contains(colC, "họ và tên")
		isSTKCol := colD == "stk" || colD == "số tk" || strings.Contains(colD, "số tài khoản") || strings.Contains(colD, "so tai khoan")
		if isIDCol || isNameCol || isSTKCol {
			headerRow = i
			break
		}
	}

	// Data starts on the row after the detected header. If no header label was
	// found, preserve the historical assumption of 3 header rows (data from
	// row 4) so previously-supported sheets keep parsing unchanged.
	dataStart := 3
	if headerRow >= 0 {
		dataStart = headerRow + 1
	}

	var stkRows []STKRow
	for i := dataStart; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 3 {
			continue
		}
		cccd := strings.TrimSpace(row[1])
		fullName := strings.TrimSpace(row[2])

		// Stop when both CCCD and Name are empty
		if cccd == "" && fullName == "" {
			break
		}
		if cccd == "" {
			continue
		}

		bankAccount := ""
		if len(row) > 3 {
			bankAccount = strings.TrimSpace(row[3])
		}
		bankName := ""
		if len(row) > 4 {
			bankName = strings.TrimSpace(row[4])
		}
		note := ""
		if len(row) > 5 {
			note = strings.TrimSpace(row[5])
		}

		stkRows = append(stkRows, STKRow{
			CCCD:        cccd,
			FullName:    fullName,
			BankAccount: bankAccount,
			BankName:    bankName,
			Note:        note,
		})
	}

	return stkRows, nil
}
