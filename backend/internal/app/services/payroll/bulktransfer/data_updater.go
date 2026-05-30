package bulktransfer

import (
	"encoding/json"
	"fmt"

	"api-server/internal/app/dto"
	"api-server/internal/infra/observability"
)

// DataUpdater handles updating bulk_transfer_files.data with transaction results
type DataUpdater struct{}

// NewDataUpdater creates a new DataUpdater instance
func NewDataUpdater() *DataUpdater {
	return &DataUpdater{}
}

// UpdateTransactionData matches uploaded results with export data and updates status fields
// Returns updated JSON string and statistics
func (u *DataUpdater) UpdateTransactionData(
	originalDataJSON string,
	parsedResults []*ParsedResultRow,
) (string, *UpdateStats, error) {
	logger := observability.GetLogger()

	// Unmarshal original data
	originalData, err := dto.ParseBulkTransferFileData(originalDataJSON)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse original data: %w", err)
	}

	// Create lookup map by transaction_code for O(1) matching
	resultMap := make(map[string]*ParsedResultRow)
	for _, result := range parsedResults {
		if result.TransactionCode != "" {
			resultMap[result.TransactionCode] = result
		}
	}

	// Statistics
	stats := &UpdateStats{
		TotalTransactions: len(originalData),
	}

	// Update transactions by matching transaction_code
	for i := range originalData {
		txCode := originalData[i].TransactionCode

		// Look up matching result
		result, found := resultMap[txCode]
		if !found {
			logger.Warn("Transaction code not found in result file",
				"transaction_code", txCode,
				"employee_id", originalData[i].EmployeeID)
			stats.UnmatchedCount++
			continue
		}

		stats.MatchedCount++

		// Update status fields based on result
		originalData[i].TransferStatus = result.Status

		if result.Status == ResultStatusCompleted {
			originalData[i].BankTxnRef = result.BankTxnRef
			originalData[i].ErrorMessage = "" // Clear any previous error
			stats.CompletedCount++
		} else {
			originalData[i].ErrorMessage = result.ErrorMessage
			originalData[i].BankTxnRef = "" // Clear any previous bank ref
			stats.FailedCount++
		}

	}

	// Marshal back to JSON
	updatedJSON, err := json.Marshal(originalData)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal updated data: %w", err)
	}

	logger.Info("Data update completed",
		"total", stats.TotalTransactions,
		"matched", stats.MatchedCount,
		"completed", stats.CompletedCount,
		"failed", stats.FailedCount,
		"unmatched", stats.UnmatchedCount)

	return string(updatedJSON), stats, nil
}

// ValidateResults validates parsed results before processing
func (u *DataUpdater) ValidateResults(results []*ParsedResultRow) error {
	if len(results) == 0 {
		return fmt.Errorf("no results to process")
	}

	// Check for duplicate transaction codes
	seen := make(map[string]bool)
	for _, result := range results {
		if result.TransactionCode == "" {
			continue
		}

		if seen[result.TransactionCode] {
			return fmt.Errorf("duplicate transaction code found: %s", result.TransactionCode)
		}
		seen[result.TransactionCode] = true
	}

	return nil
}
