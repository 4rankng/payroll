package excelkit

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// MBankStatementCols holds the 0-based column indexes of the fields the
// bank-result importers read from an MBank "Kết quả giao dịch" bulk-transfer
// statement export. Indexes are -1 when the export omits the column.
type MBankStatementCols struct {
	AccountNumber   int
	AccountName     int
	BankName        int
	Amount          int
	TransactionCode int
	Status          int
	BankRef         int
}

// maxStatementHeaderScan bounds how far into the sheet the header row is
// searched. The statement carries ~11 banner rows above the column headers.
const maxStatementHeaderScan = 40

// LocateMBankStatementHeader finds the column-header row of the MBank
// "Kết quả giao dịch" bulk-transfer statement ("THÔNG TIN CHI TIẾT GIAO DỊCH
// LÔ / Detailed Statement of Bulk Transfer"). Columns are located by header
// text, never by position, so renamed sheets or extra banner rows still
// parse. It returns the header row index, the column map, and whether the
// row carries the required columns (account, amount, transaction code,
// transaction status). Files that do not match keep their owning island's
// legacy positional parsing.
func LocateMBankStatementHeader(rows [][]string) (int, MBankStatementCols, bool) {
	emptyCols := MBankStatementCols{
		AccountNumber:   -1,
		AccountName:     -1,
		BankName:        -1,
		Amount:          -1,
		TransactionCode: -1,
		Status:          -1,
		BankRef:         -1,
	}

	limit := min(len(rows), maxStatementHeaderScan)

	for i := range limit {
		cols := mapStatementHeaderRow(rows[i])
		if cols.AccountNumber >= 0 && cols.TransactionCode >= 0 && cols.Status >= 0 && cols.Amount >= 0 {
			return i, cols, true
		}
	}

	return -1, emptyCols, false
}

// mapStatementHeaderRow maps one candidate row's cells to statement fields
// by normalized header text. More specific phrases are matched first so
// "NGÂN HÀNG THỤ HƯỞNG" can never claim the beneficiary-name column.
func mapStatementHeaderRow(row []string) MBankStatementCols {
	cols := MBankStatementCols{
		AccountNumber:   -1,
		AccountName:     -1,
		BankName:        -1,
		Amount:          -1,
		TransactionCode: -1,
		Status:          -1,
		BankRef:         -1,
	}

	for j, cell := range row {
		normalized := normalizeStatementCell(cell)
		switch {
		case cols.BankName < 0 && strings.Contains(normalized, "ngân hàng thụ hưởng"):
			cols.BankName = j
		case cols.AccountNumber < 0 && strings.Contains(normalized, "số tài khoản"):
			cols.AccountNumber = j
		case cols.TransactionCode < 0 && strings.Contains(normalized, "chi tiết thanh toán"):
			cols.TransactionCode = j
		case cols.Status < 0 && strings.Contains(normalized, "trạng thái giao dịch"):
			cols.Status = j
		case cols.Amount < 0 && strings.Contains(normalized, "số tiền"):
			cols.Amount = j
		case cols.AccountName < 0 && strings.Contains(normalized, "thụ hưởng"):
			cols.AccountName = j
		case cols.BankRef < 0 && strings.Contains(normalized, "bút toán") && !strings.Contains(normalized, "trạng thái"):
			cols.BankRef = j
		}
	}

	return cols
}

// normalizeStatementCell prepares a header cell for keyword matching: NFC
// form (Vietnamese exports may arrive NFD), lower-cased, whitespace
// collapsed.
func normalizeStatementCell(s string) string {
	return CollapseHeader(norm.NFC.String(s))
}
