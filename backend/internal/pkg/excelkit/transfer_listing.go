package excelkit

import "strings"

// TransferListingCols holds the 0-based column indexes of an OUTBOUND
// transfer listing ("chuyển lô" / bulk-payment request) workbook — the file
// an admin hands to the bank. Indexes are -1 when the listing omits the
// column.
type TransferListingCols struct {
	Order          int
	AccountNumber  int
	AccountName    int
	BankName       int
	Amount         int
	TransactionRef int
}

// maxTransferListingHeaderScan bounds how far into the sheet the header row
// is searched. The listing carries a title row above the column headers.
const maxTransferListingHeaderScan = 20

// LocateTransferListingHeader finds the column-header row of an outbound
// transfer listing. Columns are located by header text, never by position, so
// both outbound shapes parse: the MBank manual template (6 columns, payment
// detail in F) and the OnePay "eMB_BulkPayment" workbook (7 columns, SWIFT
// code in E, payment detail in G).
//
// It returns the header row index, the column map, and whether the row
// carries the columns the KQ converter cannot do without (account, name,
// amount, payment detail).
func LocateTransferListingHeader(rows [][]string) (int, TransferListingCols, bool) {
	emptyCols := TransferListingCols{
		Order:          -1,
		AccountNumber:  -1,
		AccountName:    -1,
		BankName:       -1,
		Amount:         -1,
		TransactionRef: -1,
	}

	limit := min(len(rows), maxTransferListingHeaderScan)

	for i := range limit {
		cols := mapTransferListingHeaderRow(rows[i])
		if cols.AccountNumber >= 0 && cols.AccountName >= 0 && cols.Amount >= 0 && cols.TransactionRef >= 0 {
			return i, cols, true
		}
	}

	return -1, emptyCols, false
}

// mapTransferListingHeaderRow maps one candidate row's cells to listing
// fields by normalized header text. "Ngân hàng thụ hưởng" is matched before
// the beneficiary name so it can never claim that column, and the amount
// match rejects "tổng số tiền" so a summary cell is not read as the header.
func mapTransferListingHeaderRow(row []string) TransferListingCols {
	cols := TransferListingCols{
		Order:          -1,
		AccountNumber:  -1,
		AccountName:    -1,
		BankName:       -1,
		Amount:         -1,
		TransactionRef: -1,
	}

	for j, cell := range row {
		normalized := normalizeStatementCell(cell)
		switch {
		case cols.BankName < 0 && strings.Contains(normalized, "ngân hàng thụ hưởng"):
			cols.BankName = j
		case cols.AccountNumber < 0 && strings.Contains(normalized, "số tài khoản"):
			cols.AccountNumber = j
		case cols.TransactionRef < 0 && (strings.Contains(normalized, "nội dung chuyển khoản") ||
			strings.Contains(normalized, "payment detail")):
			cols.TransactionRef = j
		case cols.AccountName < 0 && strings.Contains(normalized, "tên người thụ hưởng"):
			cols.AccountName = j
		case cols.Amount < 0 && strings.Contains(normalized, "số tiền") &&
			!strings.Contains(normalized, "tổng số tiền"):
			cols.Amount = j
		case cols.Order < 0 && strings.Contains(normalized, "stt"):
			cols.Order = j
		}
	}

	return cols
}
