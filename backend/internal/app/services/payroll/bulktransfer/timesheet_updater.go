package bulktransfer

import (
	"fmt"

	"api-server/internal/app/dto"
	"api-server/internal/infra/observability"
)

// TimesheetUpdater extracts timesheet update data from bulk transfer results
type TimesheetUpdater struct{}

// NewTimesheetUpdater creates a new TimesheetUpdater instance
func NewTimesheetUpdater() *TimesheetUpdater {
	return &TimesheetUpdater{}
}

// TimesheetUpdateData contains updates to be applied to timesheets
type TimesheetUpdateData struct {
	BankRefs  map[uint]string // map[timesheet_id]bank_transfer_ref
	FailedIDs []uint          // timesheet_ids with failed status
}

// BuildTimesheetUpdates extracts timesheet_ids from bulkFile.data,
// matches with parsedResults by transaction_code, and builds update map
func (u *TimesheetUpdater) BuildTimesheetUpdates(
	originalDataJSON string,
	parsedResults []*ParsedResultRow,
) (*TimesheetUpdateData, error) {
	logger := observability.GetLogger()

	// Parse original data
	originalData, err := dto.ParseBulkTransferFileData(originalDataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse original data: %w", err)
	}

	// Create map by transaction_code for O(1) lookup
	resultMap := make(map[string]*ParsedResultRow)
	for _, result := range parsedResults {
		if result.TransactionCode != "" {
			resultMap[result.TransactionCode] = result
		}
	}

	// Initialize update data
	updateData := &TimesheetUpdateData{
		BankRefs:  make(map[uint]string),
		FailedIDs: make([]uint, 0),
	}

	// Match and build updates
	for _, txn := range originalData {
		result, found := resultMap[txn.TransactionCode]
		if !found {
			logger.Warn("Transaction code not found in result file",
				"transaction_code", txn.TransactionCode,
				"employee_id", txn.EmployeeID)
			continue
		}

		// Process each timesheet ID
		for _, timesheetID := range txn.TimesheetIDs {
			if result.Status == ResultStatusCompleted {
				// Successful transaction: store bank reference
				updateData.BankRefs[timesheetID] = result.BankTxnRef

			} else {
				// Failed transaction: mark for failure and store error message
				updateData.FailedIDs = append(updateData.FailedIDs, timesheetID)
				updateData.BankRefs[timesheetID] = result.ErrorMessage

			}
		}
	}

	logger.Info("Built timesheet updates",
		"bank_ref_count", len(updateData.BankRefs),
		"failed_count", len(updateData.FailedIDs))

	return updateData, nil
}
