package bulktransfer

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// LegacyResultStrategy handles bank result files with H2/H3 metadata format
// H2 contains date (YYYY-MM-DD)
// H3 contains [project_id][timesheet_ids]
type LegacyResultStrategy struct {
	rowParser *RowParser
}

// NewLegacyResultStrategy creates a new legacy strategy instance
func NewLegacyResultStrategy(rowParser *RowParser) *LegacyResultStrategy {
	return &LegacyResultStrategy{
		rowParser: rowParser,
	}
}

// Detect checks if this is a legacy format file by looking for date in H2
func (s *LegacyResultStrategy) Detect(rows [][]string) bool {
	if len(rows) < 2 {
		return false
	}

	// Check if H2 (row 1, column 7) contains date format YYYY-MM-DD
	if len(rows[1]) < 8 {
		return false
	}

	h2Value := strings.TrimSpace(rows[1][7]) // Column H is index 7

	// Check if it matches date format YYYY-MM-DD
	datePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	return datePattern.MatchString(h2Value)
}

// ParseRows parses rows using the legacy format
// For legacy format, this method is less critical as lookup is done via H2/H3
func (s *LegacyResultStrategy) ParseRows(ctx context.Context, rows [][]string) ([]*ParsedResultRow, error) {
	var results []*ParsedResultRow

	// Start from row 3 (index 2) - skip header rows
	for i := 2; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 6 {
			continue // Skip incomplete rows
		}

		// Parse basic fields
		accountNumber := strings.TrimSpace(row[1]) // Column B
		accountName := strings.TrimSpace(row[2])   // Column C
		amountStr := strings.TrimSpace(row[4])     // Column E

		if accountNumber == "" {
			continue // Skip empty rows
		}

		// Parse amount
		amount := s.parseAmount(amountStr)

		// Parse status from column G
		statusStr := ""
		if len(row) > 6 {
			statusStr = strings.TrimSpace(row[6])
		}

		status := s.parseStatus(statusStr)

		// Parse tracking data from column H
		bankTxnRef := ""
		errorMessage := ""
		if len(row) > 7 {
			trackingData := strings.TrimSpace(row[7])
			switch status {
			case ResultStatusCompleted:
				bankTxnRef = trackingData
			case ResultStatusFailed:
				errorMessage = trackingData
			}
		}

		result := &ParsedResultRow{
			STT:           i - 1, // Row number minus header
			AccountNumber: accountNumber,
			AccountName:   accountName,
			Amount:        amount,
			Status:        status,
			BankTxnRef:    bankTxnRef,
			ErrorMessage:  errorMessage,
		}

		results = append(results, result)
	}

	return results, nil
}

// GetLookupKey returns the lookup key from H2 and H3
// For legacy format, this returns metadata from header rows
func (s *LegacyResultStrategy) GetLookupKey(ctx context.Context, rows [][]string) (string, error) {
	if len(rows) < 3 {
		return "", fmt.Errorf("insufficient rows for legacy format")
	}

	// H2 contains date
	if len(rows[1]) < 8 {
		return "", fmt.Errorf("H2 cell not found")
	}
	h2Date := strings.TrimSpace(rows[1][7])

	// H3 contains [project_id][timesheet_ids]
	if len(rows[2]) < 8 {
		return "", fmt.Errorf("H3 cell not found")
	}
	h3Metadata := strings.TrimSpace(rows[2][7])

	// Return combination of H2 and H3 as lookup key
	// This will be used by the result processor to find matching records
	return fmt.Sprintf("%s|%s", h2Date, h3Metadata), nil
}

// Name returns the strategy name
func (s *LegacyResultStrategy) Name() string {
	return "LegacyH2H3Strategy"
}

// parseAmount converts amount string to float64
func (s *LegacyResultStrategy) parseAmount(amountStr string) float64 {
	// Remove commas and spaces
	cleaned := strings.ReplaceAll(amountStr, ",", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")

	amount, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0
	}

	return amount
}

// parseStatus converts Vietnamese status text to normalized status
func (s *LegacyResultStrategy) parseStatus(statusStr string) string {
	normalized := strings.ToLower(strings.TrimSpace(statusStr))

	if strings.Contains(normalized, "thành công") || strings.Contains(normalized, "success") {
		return ResultStatusCompleted
	}

	if strings.Contains(normalized, "thất bại") || strings.Contains(normalized, "failed") {
		return ResultStatusFailed
	}

	// Default to failed if status is unclear
	return ResultStatusFailed
}
