package advance_payment

import (
	"strings"

	"api-server/internal/pkg/excelkit"
)

// bankResultRow is one normalized transaction row from a bank transfer
// result file, in whichever template the bank shipped.
type bankResultRow struct {
	// Row is the 1-based sheet row number, surfaced in the upload result
	Row        int
	TxCode     string
	Status     string
	PaymentRef string
}

// extractBankResultRows pulls (transaction code, status, payment reference)
// rows out of a bank transfer result file.
//
// Two layouts are supported:
//   - the MBank "Kết quả giao dịch" bulk-transfer statement: banner block,
//     content-located column header, data below (payment-details column
//     carries the transaction code). Failure statuses are matched
//     case-insensitively because the statement writes "Thất bại".
//   - the legacy flat export: data from row 3 with the transaction code in
//     column F, status in column G and the bank reference in column I.
//
// Rows without a transaction code are skipped.
func extractBankResultRows(rows [][]string) []bankResultRow {
	if headerRow, cols, ok := excelkit.LocateMBankStatementHeader(rows); ok {
		return extractStatementRows(rows, headerRow, cols)
	}
	return extractLegacyFlatRows(rows)
}

func extractStatementRows(rows [][]string, headerRow int, cols excelkit.MBankStatementCols) []bankResultRow {
	var result []bankResultRow
	for i := headerRow + 1; i < len(rows); i++ {
		row := rows[i]
		txCode := statementCellAt(row, cols.TransactionCode)
		if txCode == "" {
			continue
		}
		result = append(result, bankResultRow{
			Row:        i + 1,
			TxCode:     txCode,
			Status:     normalizeStatementStatus(statementCellAt(row, cols.Status)),
			PaymentRef: statementCellAt(row, cols.BankRef),
		})
	}
	return result
}

func extractLegacyFlatRows(rows [][]string) []bankResultRow {
	var result []bankResultRow
	for i := 2; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 9 {
			continue
		}
		txCode := strings.TrimSpace(row[5])
		if txCode == "" {
			continue
		}
		result = append(result, bankResultRow{
			Row:        i + 1,
			TxCode:     txCode,
			Status:     strings.TrimSpace(row[6]),
			PaymentRef: strings.TrimSpace(row[8]),
		})
	}
	return result
}

// normalizeStatementStatus canonicalizes the statement's status wording for
// the case-sensitive legacy check ("FAILED" / "THẤT BẠI") in ProcessBankResult.
// Failure phrases map to "THẤT BẠI"; recognized success wording passes
// through as written; any other status — including an empty cell — also maps
// to "THẤT BẠI", matching the bulk-transfer island's fail-closed default: a
// request must never be marked completed off a status the importer does not
// understand, because the consolidated transaction and ledger entries it
// triggers are not reversed afterwards.
func normalizeStatementStatus(status string) string {
	trimmed := strings.TrimSpace(status)
	lowered := strings.ToLower(trimmed)
	if strings.Contains(lowered, "thất bại") || strings.Contains(lowered, "that bai") ||
		strings.Contains(lowered, "failed") || strings.Contains(lowered, "error") {
		return "THẤT BẠI"
	}
	if strings.Contains(lowered, "thành công") || strings.Contains(lowered, "thanh cong") ||
		strings.Contains(lowered, "success") || strings.Contains(lowered, "completed") {
		return trimmed
	}
	return "THẤT BẠI"
}

func statementCellAt(row []string, col int) string {
	if col < 0 || col >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[col])
}
