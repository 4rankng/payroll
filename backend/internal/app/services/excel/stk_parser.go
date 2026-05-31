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
	sheetName := ""
	for _, sheet := range f.GetSheetList() {
		if strings.ToUpper(sheet) == "STK" {
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

	if len(rows) < 4 {
		return nil, nil
	}

	var stkRows []STKRow
	for i := 3; i < len(rows); i++ {
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
