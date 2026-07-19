package wallet_bulk

import (
	"context"
	"encoding/json"
	"fmt"

	"api-server/internal/domain"
	domaintx "api-server/internal/domain/transactions"
)

// HandleReversedPayment compensates one employee transfer that changed from
// completed to reversed after the aggregate batch had already been booked.
// It never re-books the batch: the original receivable, ledger, batch totals,
// and only this employee's timesheets are adjusted atomically.
func (s *WalletBulkTransferService) HandleReversedPayment(ctx context.Context, payment *domaintx.WalletPayment) error {
	if payment == nil || payment.BulkTransferBatchID == nil || payment.Status != domaintx.StateReversed {
		return nil
	}

	_, err := s.txRunner.WithTransactionResult(ctx, func(txCtx context.Context) (interface{}, error) {
		batch, err := s.batchRepo.GetByIDForUpdate(txCtx, *payment.BulkTransferBatchID)
		if err != nil {
			return nil, fmt.Errorf("lock reversed batch: %w", err)
		}
		// A reversal received before initial booking is simply a failed terminal
		// row. The normal all-rows-terminal finalizer will exclude it.
		if batch.Status != domain.BulkTransferBatchStatusCompleted || batch.LedgerTxnID == nil {
			return nil, nil
		}

		payments, err := s.paymentRepo.ListByBatchIDOrdered(txCtx, batch.ID)
		if err != nil {
			return nil, fmt.Errorf("lock batch payments for reversal: %w", err)
		}
		completedCount := 0
		reversedRows := make([]*domaintx.WalletPayment, 0)
		for _, candidate := range payments {
			if candidate.Status == domaintx.StateCompleted {
				completedCount++
			}
			if candidate.Status == domaintx.StateReversed {
				reversedRows = append(reversedRows, candidate)
			}
		}
		foundTrigger := false
		for _, reversed := range reversedRows {
			if reversed.ID == payment.ID {
				foundTrigger = true
				break
			}
		}
		if !foundTrigger {
			return nil, fmt.Errorf("payment %d is not a reversed row in batch %d", payment.ID, batch.ID)
		}
		// More than one reversal callback can commit before either reconciliation
		// acquires the batch lock. Reconcile every outstanding reversed row while
		// holding the lock instead of assuming only the triggering row changed.
		outstandingCount := batch.SuccessCount - completedCount
		if outstandingCount == 0 {
			return nil, nil
		}
		if outstandingCount < 0 || outstandingCount > len(reversedRows) {
			return nil, fmt.Errorf("batch %d reversal count mismatch: batch_success=%d completed_rows=%d reversed_rows=%d", batch.ID, batch.SuccessCount, completedCount, len(reversedRows))
		}
		if batch.TransferAmount <= 0 {
			return nil, fmt.Errorf("batch %d has invalid remaining transfer total %d", batch.ID, batch.TransferAmount)
		}

		requestIDs := make([]string, 0, len(reversedRows))
		for _, reversed := range reversedRows {
			requestIDs = append(requestIDs, reversed.RequestID)
		}
		codes, err := s.txnCodeRepo.FindByCodes(txCtx, requestIDs)
		if err != nil {
			return nil, fmt.Errorf("resolve reversed VFIC codes: %w", err)
		}
		if len(codes) != len(requestIDs) {
			return nil, fmt.Errorf("batch %d reversed VFIC code mismatch: expected=%d found=%d", batch.ID, len(requestIDs), len(codes))
		}
		codesByID := make(map[string]*domain.TransactionCode, len(codes))
		for _, code := range codes {
			codesByID[code.Code] = code
		}

		type reversalCandidate struct {
			payment      *domaintx.WalletPayment
			timesheetIDs []uint
		}
		candidates := make([]reversalCandidate, 0, len(reversedRows))
		allTimesheetIDs := make([]uint, 0)
		seenTimesheets := make(map[uint]struct{})
		for _, reversed := range reversedRows {
			if reversed.RequestedAmount <= 0 || reversed.RequestedAmount > batch.TransferAmount {
				return nil, fmt.Errorf("batch %d invalid reversal amount: payment=%d transfer_total=%d", batch.ID, reversed.RequestedAmount, batch.TransferAmount)
			}
			code := codesByID[reversed.RequestID]
			if code == nil {
				return nil, fmt.Errorf("reversed VFIC code %s not found", reversed.RequestID)
			}
			var codeData domain.TransactionCodeData
			if err := json.Unmarshal(code.Data, &codeData); err != nil {
				return nil, fmt.Errorf("decode reversed VFIC code %s: %w", reversed.RequestID, err)
			}
			if codeData.GetAmount() != reversed.RequestedAmount {
				return nil, fmt.Errorf("reversed VFIC code %s amount mismatch: code=%d payment=%d", reversed.RequestID, codeData.GetAmount(), reversed.RequestedAmount)
			}
			ids := codeData.GetTimesheetIDs()
			if len(ids) == 0 {
				return nil, fmt.Errorf("reversed VFIC code %s has no timesheet IDs", reversed.RequestID)
			}
			for _, id := range ids {
				if _, exists := seenTimesheets[id]; exists {
					return nil, fmt.Errorf("timesheet %d is referenced by multiple reversed transaction codes", id)
				}
				seenTimesheets[id] = struct{}{}
				allTimesheetIDs = append(allTimesheetIDs, id)
			}
			candidates = append(candidates, reversalCandidate{payment: reversed, timesheetIDs: ids})
		}

		txnID := uint(*batch.LedgerTxnID)
		receivableTxn, err := s.txnAdjuster.GetTransactionForUpdate(txCtx, txnID)
		if err != nil {
			return nil, fmt.Errorf("lock batch receivable transaction: %w", err)
		}
		if receivableTxn.TransactionType != domain.TransactionTypeRevenue {
			return nil, fmt.Errorf("batch %d ledger transaction %d is not revenue", batch.ID, txnID)
		}
		timesheets, err := s.timesheetLinker.GetByIDsForUpdate(txCtx, allTimesheetIDs)
		if err != nil {
			return nil, fmt.Errorf("lock reversed timesheets: %w", err)
		}
		if len(timesheets) != len(allTimesheetIDs) {
			return nil, fmt.Errorf("reversed timesheet count mismatch: expected=%d found=%d", len(allTimesheetIDs), len(timesheets))
		}
		timesheetsByID := make(map[uint]*domain.Timesheet, len(timesheets))
		for _, timesheet := range timesheets {
			timesheetsByID[timesheet.ID] = timesheet
		}

		pending := make([]reversalCandidate, 0, outstandingCount)
		for _, candidate := range candidates {
			linked := 0
			unlinked := 0
			for _, id := range candidate.timesheetIDs {
				timesheet := timesheetsByID[id]
				if timesheet == nil {
					return nil, fmt.Errorf("reversed timesheet %d was not locked", id)
				}
				switch {
				case timesheet.TransactionID == nil:
					unlinked++
				case *timesheet.TransactionID == txnID:
					linked++
				default:
					return nil, fmt.Errorf("timesheet %d is linked to unexpected transaction %d", id, *timesheet.TransactionID)
				}
			}
			if linked > 0 && unlinked > 0 {
				return nil, fmt.Errorf("reversed VFIC code %s has partially unlinked timesheets", candidate.payment.RequestID)
			}
			if linked == len(candidate.timesheetIDs) {
				pending = append(pending, candidate)
			}
		}
		if len(pending) != outstandingCount {
			return nil, fmt.Errorf("batch %d outstanding reversal mismatch: count_delta=%d linked_rows=%d", batch.ID, outstandingCount, len(pending))
		}

		now := s.clock()
		entries := make([]*domain.LedgerEntry, 0, len(pending)*4)
		clearIDs := make([]uint, 0)
		remainingReceivable := receivableTxn.Amount
		remainingTransfer := batch.TransferAmount
		var totalReceivableShare int64
		var totalReversedAmount int64
		for _, candidate := range pending {
			amount := candidate.payment.RequestedAmount
			receivableShare := remainingReceivable
			if amount != remainingTransfer {
				receivableShare = (amount*remainingReceivable + remainingTransfer/2) / remainingTransfer
			}
			if receivableShare <= 0 || receivableShare > remainingReceivable {
				return nil, fmt.Errorf("batch %d invalid receivable reversal share %d of %d", batch.ID, receivableShare, remainingReceivable)
			}
			entries = append(entries,
				&domain.LedgerEntry{Date: now, Account: domain.AccountReceivable, Party: receivableTxn.Party, Credit: receivableShare, TransactionID: &txnID},
				&domain.LedgerEntry{Date: now, Account: domain.AccountRevenue, Party: receivableTxn.Party, Debit: receivableShare, TransactionID: &txnID},
				&domain.LedgerEntry{Date: now, Account: domain.AccountCash, Party: "Nhân viên", Debit: amount, TransactionID: &txnID},
				&domain.LedgerEntry{Date: now, Account: domain.AccountRevenue, Party: receivableTxn.Party, Credit: amount, TransactionID: &txnID},
			)
			clearIDs = append(clearIDs, candidate.timesheetIDs...)
			totalReceivableShare += receivableShare
			totalReversedAmount += amount
			remainingReceivable -= receivableShare
			remainingTransfer -= amount
		}
		if err := s.txnAdjuster.IncrementTransactionAmount(txCtx, txnID, -totalReceivableShare); err != nil {
			return nil, fmt.Errorf("reduce batch receivable: %w", err)
		}
		if _, err := s.ledgerWriter.CreateEntriesAtomic(txCtx, entries, uint(batch.CreatedBy)); err != nil {
			return nil, fmt.Errorf("create reversed-transfer ledger adjustment: %w", err)
		}
		if err := s.timesheetLinker.ClearTransactionID(txCtx, txnID, clearIDs); err != nil {
			return nil, err
		}
		if err := s.batchRepo.UpdateColumns(txCtx, batch.ID, map[string]interface{}{
			"success_count":   batch.SuccessCount - len(pending),
			"failed_count":    batch.FailedCount + len(pending),
			"transfer_amount": batch.TransferAmount - totalReversedAmount,
		}); err != nil {
			return nil, fmt.Errorf("update reversed batch totals: %w", err)
		}
		return nil, nil
	})
	return err
}
