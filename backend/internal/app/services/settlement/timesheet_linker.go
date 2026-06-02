package settlement

import (
	"context"
	"fmt"

	"api-server/internal/constants"
	"api-server/internal/domain"

	"gorm.io/gorm"
)

// InternalSheetRow represents a single row from the INTERNAL sheet in settlement upload
type InternalSheetRow struct {
	TimesheetID uint
	Amount      int64
}

// SettlementValidationResult contains the validated settlement allocation data
type SettlementValidationResult struct {
	Transactions           map[uint]int64 // transactionID -> allocatedAmount
	TimesheetIDs           []uint         // all valid INTERNAL timesheet IDs
	TimesheetToTransaction map[uint]uint  // timesheetID -> transactionID
	SettledInternally      bool           // true when settlement (ledger + events) already handled internally
}

// SettlementTimesheetLinker handles linking timesheets to transactions and marking them as paid
type SettlementTimesheetLinker struct {
	db              *gorm.DB
	timesheetRepo   domain.TimesheetRepository
	transactionRepo domain.TransactionRepository
}

// NewSettlementTimesheetLinker creates a new timesheet linker
func NewSettlementTimesheetLinker(
	db *gorm.DB,
	timesheetRepo domain.TimesheetRepository,
	transactionRepo domain.TransactionRepository,
) *SettlementTimesheetLinker {
	return &SettlementTimesheetLinker{
		db:              db,
		timesheetRepo:   timesheetRepo,
		transactionRepo: transactionRepo,
	}
}

// ValidateInternalSheet validates INTERNAL sheet rows and returns settlement allocation
// This is a pure validation method - it performs NO database writes
// totalReceivedFromClient is the E2 amount from the settlement file
func (l *SettlementTimesheetLinker) ValidateInternalSheet(
	ctx context.Context,
	rows []InternalSheetRow,
	totalReceivedFromClient int64,
) (*SettlementValidationResult, error) {
	if len(rows) == 0 {
		return nil, domain.NewValidationError(constants.MsgInternalSheetNoTimesheetIDsVN)
	}

	// Step 1: Load timesheets directly
	var timesheetIDs []uint
	timesheetIDSet := make(map[uint]bool)
	for _, row := range rows {
		if !timesheetIDSet[row.TimesheetID] {
			timesheetIDs = append(timesheetIDs, row.TimesheetID)
			timesheetIDSet[row.TimesheetID] = true
		}
	}

	timesheets, err := l.timesheetRepo.GetByIDs(ctx, timesheetIDs)
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgCannotGetTimesheetInfoVN, err)
	}

	if len(timesheets) != len(timesheetIDs) {
		return nil, domain.NewValidationError(
			fmt.Sprintf("Chỉ tìm thấy %d/%d timesheets trong hệ thống",
				len(timesheets), len(timesheetIDs)))
	}

	// Step 2: Enforce revenue_paid = 0
	var alreadyPaidIDs []uint
	for _, ts := range timesheets {
		if ts.RevenuePaid {
			alreadyPaidIDs = append(alreadyPaidIDs, ts.ID)
		}
	}
	if len(alreadyPaidIDs) > 0 {
		return nil, domain.NewValidationError(
			fmt.Sprintf("Timesheet đã được thanh toán trước đó (IDs: %v). Không thể thanh toán lại.",
				alreadyPaidIDs))
	}

	// Step 2.1: Calculate total revenue receivable from timesheets (for Validation #1)
	var totalRevenueReceivable int64
	for _, ts := range timesheets {
		totalRevenueReceivable += ts.RevenueReceivable
	}

	// Step 2.2: SUM(revenue_receivable) = totalReceivedFromClient (within tolerance)
	revenueDifference := totalRevenueReceivable - totalReceivedFromClient
	if revenueDifference < 0 {
		revenueDifference = -revenueDifference // absolute value
	}
	if revenueDifference > constants.RoundingTolerance {
		return nil, domain.NewValidationError(
			fmt.Sprintf("Tổng doanh thu phải thu từ timesheets (%d ₫) không khớp với số tiền khách hàng chuyển (%d ₫). "+
				"Chênh lệch %d ₫ vượt quá ngưỡng cho phép %d ₫.",
				totalRevenueReceivable, totalReceivedFromClient, revenueDifference, constants.RoundingTolerance))
	}

	// Step 3: Validate transaction_id presence and build allocation maps
	timesheetMap := make(map[uint]*domain.Timesheet)
	for _, ts := range timesheets {
		timesheetMap[ts.ID] = ts
	}

	transactionToTimesheets := make(map[uint][]uint)
	transactionAllocations := make(map[uint]int64)

	for _, row := range rows {
		ts := timesheetMap[row.TimesheetID]
		if ts.TransactionID == nil {
			return nil, domain.NewValidationError(
				fmt.Sprintf("Timesheet ID %d chưa được liên kết với giao dịch nào. "+
					"Vui lòng chạy endpoint /api/v1/debug/link-timesheet-transaction-id trước.",
					ts.ID))
		}

		txnID := *ts.TransactionID
		transactionToTimesheets[txnID] = append(transactionToTimesheets[txnID], ts.ID)
		transactionAllocations[txnID] += row.Amount
	}

	// Step 4: Fetch transactions and their remaining amounts
	var transactionIDs []uint
	for txnID := range transactionAllocations {
		transactionIDs = append(transactionIDs, txnID)
	}

	transactions, err := l.transactionRepo.GetByIDs(ctx, transactionIDs)
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgCannotFindTransactionsVN, err)
	}

	if len(transactions) != len(transactionIDs) {
		return nil, domain.NewValidationError(
			fmt.Sprintf("Chỉ tìm thấy %d/%d transactions trong hệ thống",
				len(transactions), len(transactionIDs)))
	}

	// Build transaction map for easy lookup
	transactionMap := make(map[uint]*domain.Transaction)
	for _, txn := range transactions {
		transactionMap[txn.ID] = txn
	}

	// Step 5: Validate per-transaction allocation, then calculate aggregate totals
	var totalAllocated int64
	var totalRemaining int64

	for txnID, allocated := range transactionAllocations {
		txn := transactionMap[txnID]
		remaining := txn.GetRemainingAmount()

		totalAllocated += allocated
		totalRemaining += remaining

		// Per-transaction validation: prevent over-settling individual transactions
		// This prevents order-dependency bugs when multiple files settle different timesheets for same transaction
		if allocated > remaining+constants.RoundingTolerance {
			return nil, domain.NewValidationError(
				fmt.Sprintf("Không thể thanh toán giao dịch #%d với số tiền %d ₫ vì chỉ còn %d ₫ chưa thanh toán (chênh lệch %d ₫ > %d ₫ cho phép)",
					txnID, allocated, remaining, allocated-remaining, constants.RoundingTolerance))
		}
	}

	// Step 6: Aggregate validation - SUM(transaction remaining amount) vs totalReceivedFromClient (within tolerance)
	// After settlement, leftover = totalReceivedFromClient - totalRemaining
	//
	// Valid cases:
	// - Client sends LESS than remaining (underpayment): Acceptable
	// - Client sends EQUAL to remaining: Full settlement
	// - Client sends SLIGHTLY MORE (within tolerance): Acceptable rounding difference
	//
	// Invalid case:
	// - Client sends TOO MUCH MORE (beyond tolerance): Excess money that cannot be explained
	leftover := totalReceivedFromClient - totalRemaining

	if leftover > constants.RoundingTolerance {
		return nil, domain.NewValidationError(
			fmt.Sprintf("Số tiền khách hàng chuyển (%d ₫) vượt quá tổng số tiền còn lại của các giao dịch (%d ₫) "+
				"với chênh lệch %d ₫ (> %d ₫ cho phép). "+
				"Có quá nhiều tiền thừa không rõ nguồn gốc. Vui lòng kiểm tra lại số tiền đối soát.",
				totalReceivedFromClient, totalRemaining, leftover, constants.RoundingTolerance))
	}

	// Note: Negative leftover (partial settlement) is allowed - no validation needed

	// Build timesheet -> transaction mapping
	tsToTxn := make(map[uint]uint, len(rows))
	for txnID, tsIDs := range transactionToTimesheets {
		for _, tsID := range tsIDs {
			tsToTxn[tsID] = txnID
		}
	}

	// Step 7: Return structured result - no DB writes
	return &SettlementValidationResult{
		Transactions:           transactionAllocations,
		TimesheetIDs:           timesheetIDs,
		TimesheetToTransaction: tsToTxn,
	}, nil
}

// ValidateInternalSheetWithDedup is like ValidateInternalSheet but skips already-paid
// timesheets instead of erroring. It returns the filtered result and the IDs that were skipped.
// The totalReceivedFromClient is the settlement amount from the Excel file.
func (l *SettlementTimesheetLinker) ValidateInternalSheetWithDedup(
	ctx context.Context,
	rows []InternalSheetRow,
	totalReceivedFromClient int64,
) (*SettlementValidationResult, []uint, error) {
	if len(rows) == 0 {
		return nil, nil, domain.NewValidationError(constants.MsgInternalSheetNoTimesheetIDsVN)
	}

	// Step 1: Deduplicate input rows
	var timesheetIDs []uint
	timesheetIDSet := make(map[uint]bool)
	for _, row := range rows {
		if !timesheetIDSet[row.TimesheetID] {
			timesheetIDs = append(timesheetIDs, row.TimesheetID)
			timesheetIDSet[row.TimesheetID] = true
		}
	}

	timesheets, err := l.timesheetRepo.GetByIDs(ctx, timesheetIDs)
	if err != nil {
		return nil, nil, domain.NewInternalError(constants.MsgCannotGetTimesheetInfoVN, err)
	}

	if len(timesheets) != len(timesheetIDs) {
		return nil, nil, domain.NewValidationError(
			fmt.Sprintf("Chỉ tìm thấy %d/%d timesheets trong hệ thống",
				len(timesheets), len(timesheetIDs)))
	}

	// Step 2: Split into paid and unpaid
	// Smarter dedup: only skip if revenue_paid AND transaction has actual settlements.
	// If revenue_paid but transaction has no settlements (orphaned), reprocess it.
	timesheetMap := make(map[uint]*domain.Timesheet)
	for _, ts := range timesheets {
		timesheetMap[ts.ID] = ts
	}

	var skippedIDs []uint
	var unpaidRows []InternalSheetRow
	var needsTxnCheck []InternalSheetRow

	for _, row := range rows {
		ts := timesheetMap[row.TimesheetID]
		if ts.RevenuePaid {
			needsTxnCheck = append(needsTxnCheck, row)
			continue
		}
		unpaidRows = append(unpaidRows, row)
	}

	// Step 2.1: For revenue_paid timesheets, check if their transactions actually have settlements.
	// Orphaned timesheets (revenue_paid but transaction has settled_amount=0) are reprocessed.
	if len(needsTxnCheck) > 0 {
		txnIDSet := make(map[uint]bool)
		for _, row := range needsTxnCheck {
			ts := timesheetMap[row.TimesheetID]
			if ts.TransactionID != nil {
				txnIDSet[*ts.TransactionID] = true
			}
		}

		var txnIDs []uint
		for id := range txnIDSet {
			txnIDs = append(txnIDs, id)
		}

		transactions, err := l.transactionRepo.GetByIDs(ctx, txnIDs)
		if err != nil {
			return nil, nil, domain.NewInternalError(constants.MsgCannotFindTransactionsVN, err)
		}

		settledTxnIDs := make(map[uint]bool)
		for _, txn := range transactions {
			if txn.SettledAmount > 0 {
				settledTxnIDs[txn.ID] = true
			}
		}

		for _, row := range needsTxnCheck {
			ts := timesheetMap[row.TimesheetID]
			if ts.TransactionID != nil && settledTxnIDs[*ts.TransactionID] {
				skippedIDs = append(skippedIDs, ts.ID)
			} else {
				unpaidRows = append(unpaidRows, row)
			}
		}
	}

	// All already paid — nothing to do
	if len(unpaidRows) == 0 {
		return &SettlementValidationResult{
			Transactions: make(map[uint]int64),
			TimesheetIDs: []uint{},
		}, skippedIDs, nil
	}

	// Step 3: Build allocations from unpaid timesheets only
	transactionToTimesheets := make(map[uint][]uint)
	transactionAllocations := make(map[uint]int64)
	var unpaidTimesheetIDs []uint
	unpaidIDSet := make(map[uint]bool)

	for _, row := range unpaidRows {
		ts := timesheetMap[row.TimesheetID]
		if unpaidIDSet[ts.ID] {
			continue
		}

		if ts.TransactionID == nil {
			// Timesheet was paid but never linked to a revenue transaction
			// (e.g. excluded from the original bulk-transfer export batch).
			// Skip it so the rest of the file can be settled; it will remain
			// revenue_paid=false and appear in subsequent sao ke exports until
			// an admin links it to a transaction.
			skippedIDs = append(skippedIDs, ts.ID)
			continue
		}

		unpaidIDSet[ts.ID] = true
		unpaidTimesheetIDs = append(unpaidTimesheetIDs, ts.ID)

		txnID := *ts.TransactionID
		transactionToTimesheets[txnID] = append(transactionToTimesheets[txnID], ts.ID)
		transactionAllocations[txnID] += row.Amount
	}

	// Step 4: Fetch transactions and validate per-transaction allocation
	var transactionIDs []uint
	for txnID := range transactionAllocations {
		transactionIDs = append(transactionIDs, txnID)
	}

	transactions, err := l.transactionRepo.GetByIDs(ctx, transactionIDs)
	if err != nil {
		return nil, nil, domain.NewInternalError(constants.MsgCannotFindTransactionsVN, err)
	}

	if len(transactions) != len(transactionIDs) {
		return nil, nil, domain.NewValidationError(
			fmt.Sprintf("Chỉ tìm thấy %d/%d transactions trong hệ thống",
				len(transactions), len(transactionIDs)))
	}

	transactionMap := make(map[uint]*domain.Transaction)
	for _, txn := range transactions {
		transactionMap[txn.ID] = txn
	}

	for txnID, allocated := range transactionAllocations {
		txn := transactionMap[txnID]
		remaining := txn.GetRemainingAmount()

		if allocated > remaining+constants.RoundingTolerance {
			return nil, nil, domain.NewValidationError(
				fmt.Sprintf("Không thể thanh toán giao dịch #%d với số tiền %d ₫ vì chỉ còn %d ₫ chưa thanh toán",
					txnID, allocated, remaining))
		}
	}

	// Build timesheet -> transaction mapping
	tsToTxn := make(map[uint]uint, len(unpaidRows))
	for txnID, tsIDs := range transactionToTimesheets {
		for _, tsID := range tsIDs {
			tsToTxn[tsID] = txnID
		}
	}

	return &SettlementValidationResult{
		Transactions:           transactionAllocations,
		TimesheetIDs:           unpaidTimesheetIDs,
		TimesheetToTransaction: tsToTxn,
	}, skippedIDs, nil
}

// GetTimesheetsForTransaction retrieves all timesheets linked to a transaction
func (l *SettlementTimesheetLinker) GetTimesheetsForTransaction(
	ctx context.Context,
	transactionID uint,
) ([]*domain.Timesheet, error) {
	var timesheets []*domain.Timesheet
	if err := l.db.WithContext(ctx).
		Where("transaction_id = ?", transactionID).
		Find(&timesheets).Error; err != nil {
		return nil, fmt.Errorf("failed to get timesheets for transaction %d: %w",
			transactionID, err)
	}
	return timesheets, nil
}
