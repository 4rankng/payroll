package bulktransfer

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"api-server/internal/pkg/excelkit"
)

// MBankStatementStrategy handles the MBank "Kết quả giao dịch" bulk-transfer
// statement: a banner block (~11 rows) followed by a column-header row and
// data rows. Columns are located by header text (payment details column
// carries the VFIC transaction code), so banner changes or extra rows do
// not break parsing.
type MBankStatementStrategy struct {
	checksumCalculator *ChecksumCalculator
}

// NewMBankStatementStrategy creates a new MBank statement strategy instance
func NewMBankStatementStrategy() *MBankStatementStrategy {
	return &MBankStatementStrategy{
		checksumCalculator: NewChecksumCalculator(),
	}
}

// Detect checks whether the rows carry the MBank statement column header
func (s *MBankStatementStrategy) Detect(rows [][]string) bool {
	_, _, ok := excelkit.LocateMBankStatementHeader(rows)
	return ok
}

// ParseRows parses the data rows below the located column header
func (s *MBankStatementStrategy) ParseRows(ctx context.Context, rows [][]string) ([]*ParsedResultRow, error) {
	headerRow, cols, ok := excelkit.LocateMBankStatementHeader(rows)
	if !ok {
		return nil, fmt.Errorf("no MBank statement header row found")
	}

	var results []*ParsedResultRow

	for i := headerRow + 1; i < len(rows); i++ {
		row := rows[i]

		accountNumber := statementCell(row, cols.AccountNumber)
		accountName := statementCell(row, cols.AccountName)
		bankName := statementCell(row, cols.BankName)
		transactionCode := statementCell(row, cols.TransactionCode)
		statusStr := statementCell(row, cols.Status)
		bankRefOrError := statementCell(row, cols.BankRef)

		// Skip empty rows (statement exports pad short rows heavily)
		if accountNumber == "" && transactionCode == "" {
			continue
		}

		stt, err := strconv.Atoi(statementCell(row, 0))
		if err != nil {
			// If STT parsing fails, fall back to the row number
			stt = i - headerRow
		}

		amount := s.parseAmount(statementCell(row, cols.Amount))
		status := s.parseStatus(statusStr)

		bankTxnRef := ""
		errorMessage := ""
		if status == ResultStatusCompleted {
			bankTxnRef = bankRefOrError
		} else {
			errorMessage = bankRefOrError
		}

		results = append(results, &ParsedResultRow{
			STT:             stt,
			AccountNumber:   accountNumber,
			AccountName:     accountName,
			BankName:        bankName,
			Amount:          amount,
			TransactionCode: transactionCode,
			Status:          status,
			BankTxnRef:      bankTxnRef,
			ErrorMessage:    errorMessage,
		})
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no valid transaction rows found")
	}

	return results, nil
}

// GetLookupKey calculates the checksum from all transaction codes in the
// payment-details column
func (s *MBankStatementStrategy) GetLookupKey(ctx context.Context, rows [][]string) (string, error) {
	headerRow, cols, ok := excelkit.LocateMBankStatementHeader(rows)
	if !ok {
		return "", fmt.Errorf("no MBank statement header row found")
	}

	var transactionCodes []string
	for i := headerRow + 1; i < len(rows); i++ {
		if code := statementCell(rows[i], cols.TransactionCode); code != "" {
			transactionCodes = append(transactionCodes, code)
		}
	}

	if len(transactionCodes) == 0 {
		return "", fmt.Errorf("no transaction codes found in file")
	}

	checksum, err := s.checksumCalculator.CalculateFromTransactionCodes(transactionCodes)
	if err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return checksum, nil
}

// Name returns the strategy name
func (s *MBankStatementStrategy) Name() string {
	return "MBankStatementStrategy"
}

// statementCell reads one mapped column from a row, guarding short rows
// (excelize trims trailing empty cells) and absent columns
func statementCell(row []string, col int) string {
	if col < 0 || col >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[col])
}

// parseAmount converts the statement amount string ("930000.0") to float64
func (s *MBankStatementStrategy) parseAmount(amountStr string) float64 {
	cleaned := strings.ReplaceAll(amountStr, ",", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "đ", "")
	cleaned = strings.ReplaceAll(cleaned, "VND", "")

	amount, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0
	}

	return amount
}

// parseStatus converts the Vietnamese transaction-status text to a
// normalized status
func (s *MBankStatementStrategy) parseStatus(statusStr string) string {
	normalized := strings.ToLower(strings.TrimSpace(statusStr))

	if strings.Contains(normalized, "thành công") ||
		strings.Contains(normalized, "thanh cong") ||
		strings.Contains(normalized, "success") ||
		strings.Contains(normalized, "completed") {
		return ResultStatusCompleted
	}

	if strings.Contains(normalized, "thất bại") ||
		strings.Contains(normalized, "that bai") ||
		strings.Contains(normalized, "failed") ||
		strings.Contains(normalized, "error") {
		return ResultStatusFailed
	}

	// Default to failed if status is unclear
	return ResultStatusFailed
}
