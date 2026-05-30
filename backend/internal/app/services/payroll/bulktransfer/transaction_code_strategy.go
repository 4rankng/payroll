package bulktransfer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// TransactionCodeStrategy handles bank result files with transaction_code based matching
// Starting from row 5:
// A5=STT, B5=account no, C5=account name, D5=bank name, E5=amount
// F5=transaction code (VFIC-xxxxx), G5=skip, H5=status, I5=bank ref or error
type TransactionCodeStrategy struct {
	checksumCalculator *ChecksumCalculator
}

// NewTransactionCodeStrategy creates a new transaction code strategy instance
func NewTransactionCodeStrategy() *TransactionCodeStrategy {
	return &TransactionCodeStrategy{
		checksumCalculator: NewChecksumCalculator(),
	}
}

// Detect checks if this is a transaction_code format file
// Returns true if H2 does NOT contain date format
func (s *TransactionCodeStrategy) Detect(rows [][]string) bool {
	if len(rows) < 5 {
		return false
	}

	// If there are fewer than 2 rows, can't check H2
	if len(rows) < 2 || len(rows[1]) < 8 {
		// Default to new format if we can't determine
		return true
	}

	// Check if H2 does NOT contain date format YYYY-MM-DD
	h2Value := strings.TrimSpace(rows[1][7])

	// If H2 is empty or doesn't match date format, it's the new format
	if h2Value == "" {
		return true
	}

	// Check if it's NOT a date (simple check - not YYYY-MM-DD pattern)
	// Date format has exactly 10 characters with two dashes
	parts := strings.Split(h2Value, "-")
	if len(parts) != 3 {
		return true
	}

	// If it looks like a date, it's legacy format
	if len(parts[0]) == 4 && len(parts[1]) == 2 && len(parts[2]) == 2 {
		return false
	}

	// Default to new format
	return true
}

// ParseRows parses rows starting from row 5 (index 4)
func (s *TransactionCodeStrategy) ParseRows(ctx context.Context, rows [][]string) ([]*ParsedResultRow, error) {
	var results []*ParsedResultRow

	// Start from row 5 (index 4)
	for i := 4; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 9 { // Need columns A-I
			continue
		}

		// A=STT, B=account no, C=account name, D=bank name, E=amount
		// F=transaction code, G=skip, H=status, I=bank ref or error
		sttStr := strings.TrimSpace(row[0])          // Column A
		accountNumber := strings.TrimSpace(row[1])   // Column B
		accountName := strings.TrimSpace(row[2])     // Column C
		bankName := strings.TrimSpace(row[3])        // Column D
		amountStr := strings.TrimSpace(row[4])       // Column E
		transactionCode := strings.TrimSpace(row[5]) // Column F
		// row[6] is skipped (Column G)
		statusStr := strings.TrimSpace(row[7])      // Column H
		bankRefOrError := strings.TrimSpace(row[8]) // Column I

		// Skip empty rows
		if accountNumber == "" && transactionCode == "" {
			continue
		}

		// Parse STT
		stt, err := strconv.Atoi(sttStr)
		if err != nil {
			// If STT parsing fails, use row index
			stt = i - 3
		}

		// Parse amount
		amount := s.parseAmount(amountStr)

		// Parse status
		status := s.parseStatus(statusStr)

		// Determine bank reference or error message based on status
		bankTxnRef := ""
		errorMessage := ""
		if status == ResultStatusCompleted {
			bankTxnRef = bankRefOrError
		} else {
			errorMessage = bankRefOrError
		}

		result := &ParsedResultRow{
			STT:             stt,
			AccountNumber:   accountNumber,
			AccountName:     accountName,
			BankName:        bankName,
			Amount:          amount,
			TransactionCode: transactionCode,
			Status:          status,
			BankTxnRef:      bankTxnRef,
			ErrorMessage:    errorMessage,
		}

		results = append(results, result)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no valid transaction rows found")
	}

	return results, nil
}

// GetLookupKey calculates checksum from all transaction codes in column F
func (s *TransactionCodeStrategy) GetLookupKey(ctx context.Context, rows [][]string) (string, error) {
	var transactionCodes []string

	// Start from row 5 (index 4)
	for i := 4; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 6 {
			continue
		}

		transactionCode := strings.TrimSpace(row[5]) // Column F
		if transactionCode != "" {
			transactionCodes = append(transactionCodes, transactionCode)
		}
	}

	if len(transactionCodes) == 0 {
		return "", fmt.Errorf("no transaction codes found in file")
	}

	// Calculate checksum
	checksum, err := s.checksumCalculator.CalculateFromTransactionCodes(transactionCodes)
	if err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return checksum, nil
}

// Name returns the strategy name
func (s *TransactionCodeStrategy) Name() string {
	return "TransactionCodeStrategy"
}

// parseAmount converts amount string to float64
func (s *TransactionCodeStrategy) parseAmount(amountStr string) float64 {
	// Remove commas, spaces, and currency symbols
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

// parseStatus converts Vietnamese status text to normalized status
func (s *TransactionCodeStrategy) parseStatus(statusStr string) string {
	normalized := strings.ToLower(strings.TrimSpace(statusStr))

	// "Thành công" means "completed"
	if strings.Contains(normalized, "thành công") ||
		strings.Contains(normalized, "thanh cong") ||
		strings.Contains(normalized, "success") ||
		strings.Contains(normalized, "completed") {
		return ResultStatusCompleted
	}

	// "Thất bại" means "failed"
	if strings.Contains(normalized, "thất bại") ||
		strings.Contains(normalized, "that bai") ||
		strings.Contains(normalized, "failed") ||
		strings.Contains(normalized, "error") {
		return ResultStatusFailed
	}

	// Default to failed if status is unclear
	return ResultStatusFailed
}
