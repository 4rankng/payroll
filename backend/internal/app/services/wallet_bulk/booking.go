package wallet_bulk

import (
	"context"
	"encoding/json"
	"fmt"

	"api-server/internal/app/services/payroll/bulktransfer"
	"api-server/internal/domain"
	domaintx "api-server/internal/domain/transactions"

	asynqlib "github.com/hibiken/asynq"
)

type batchBookingData struct {
	totalFee        int64
	transferAmount  int64
	successfulCount int
	timesheetIDs    []uint
}

// ProcessBookBatchLedger records the same payroll accounting shape as the
// established bank-result upload flow. The batch lock, receivable transaction,
// four ledger entries, successful-timesheet links, provider fee, and final
// batch state all commit or roll back together.
func (s *WalletBulkTransferService) ProcessBookBatchLedger(ctx context.Context, task *asynqlib.Task) error {
	var payload BookLedgerPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal: %w: %w", err, asynqlib.SkipRetry)
	}

	batch, err := s.batchRepo.GetByID(ctx, payload.BatchID)
	if err != nil {
		return fmt.Errorf("get batch: %w", err)
	}
	if batch.LedgerTxnID != nil || batch.IsTerminal() {
		s.logger.Info("wallet_bulk: book_batch_ledger no-op (already finalized)",
			"batch_id", batch.ID, "status", batch.Status, "ledger_txn_id", batch.LedgerTxnID)
		return nil
	}

	booking, err := s.prepareBatchBooking(ctx, batch)
	if err != nil {
		return err
	}
	partner := s.partnerInfo.GetPartnerCompany(ctx)
	if partner == "" {
		return fmt.Errorf("book batch %d: partner company is empty", batch.ID)
	}
	plan := bulktransfer.BuildLedgerPlan(ctx, s.partnerInfo, float64(booking.transferAmount), partner, batch.Filename)
	now := s.clock()

	_, err = s.txRunner.WithTransactionResult(ctx, func(txCtx context.Context) (interface{}, error) {
		locked, err := s.batchRepo.GetByIDForUpdate(txCtx, batch.ID)
		if err != nil {
			return nil, fmt.Errorf("lock batch: %w", err)
		}
		if locked.LedgerTxnID != nil || locked.IsTerminal() {
			return nil, nil
		}
		if locked.Status != domain.BulkTransferBatchStatusCompleting {
			return nil, fmt.Errorf("batch %d is %q, expected completing", locked.ID, locked.Status)
		}

		var bookingTxnID *uint64
		if booking.successfulCount > 0 {
			assetID := uintPointer(locked.AssetID)
			receivableTxn, _, err := s.txnSvc.CreateTransaction(txCtx, &domain.Transaction{
				Description:     plan.Description,
				TransactionType: domain.TransactionTypeRevenue,
				Amount:          plan.ReceivableAmount,
				Party:           partner,
				Status:          domain.TransactionStatusPending,
				AssetID:         assetID,
				CreatedBy:       uint(locked.CreatedBy),
			})
			if err != nil {
				return nil, fmt.Errorf("create receivable transaction: %w", err)
			}

			manualEntries := plan.Entries(now, uint(locked.CreatedBy), partner)
			for _, entry := range manualEntries {
				entry.TransactionID = &receivableTxn.ID
			}
			if _, err := s.ledgerWriter.CreateEntries(txCtx, manualEntries, uint(locked.CreatedBy)); err != nil {
				return nil, fmt.Errorf("create salary cash/revenue entries: %w", err)
			}
			if err := s.timesheetLinker.BulkUpdateTransactionID(txCtx, receivableTxn.ID, booking.timesheetIDs); err != nil {
				return nil, fmt.Errorf("link successful timesheets: %w", err)
			}
			id := uint64(receivableTxn.ID)
			bookingTxnID = &id
		}

		if booking.totalFee > 0 {
			feeTxn, _, err := s.txnSvc.CreateTransaction(txCtx, &domain.Transaction{
				Description:     fmt.Sprintf("Phí OnePay đợt chuyển tiền %s - batch #%d", locked.Filename, locked.ID),
				TransactionType: domain.TransactionTypeExpense,
				Amount:          booking.totalFee,
				Party:           "OnePay",
				Status:          domain.TransactionStatusSettled,
				CreatedBy:       uint(locked.CreatedBy),
			})
			if err != nil {
				return nil, fmt.Errorf("create provider fee transaction: %w", err)
			}
			if bookingTxnID == nil {
				id := uint64(feeTxn.ID)
				bookingTxnID = &id
			}
		}

		if err := s.finalizeBatchColumns(txCtx, locked.ID, booking.totalFee, bookingTxnID, &now); err != nil {
			return nil, fmt.Errorf("finalize batch: %w", err)
		}
		return bookingTxnID, nil
	})
	if err != nil {
		return fmt.Errorf("book batch %d: %w", batch.ID, err)
	}

	s.logger.Info("wallet_bulk: payroll batch booked",
		"batch_id", batch.ID,
		"successful_count", booking.successfulCount,
		"transfer_amount", booking.transferAmount,
		"receivable_amount", plan.ReceivableAmount,
		"provider_fee", booking.totalFee)
	return nil
}

func (s *WalletBulkTransferService) prepareBatchBooking(ctx context.Context, batch *domain.BulkTransferBatch) (*batchBookingData, error) {
	payments, err := s.paymentRepo.ListByBatchIDOrdered(ctx, batch.ID)
	if err != nil {
		return nil, fmt.Errorf("list batch payments: %w", err)
	}

	booking := &batchBookingData{}
	successAmounts := make(map[string]int64)
	for _, payment := range payments {
		switch payment.Status {
		case domaintx.StateCompleted:
			booking.successfulCount++
			booking.transferAmount += payment.RequestedAmount
			booking.totalFee += payment.Fee
			successAmounts[payment.RequestID] = payment.RequestedAmount
		case domaintx.StateFailed:
			booking.totalFee += payment.Fee
		default:
			return nil, fmt.Errorf("batch %d payment %d is not terminal: %s", batch.ID, payment.ID, payment.Status)
		}
	}
	if booking.successfulCount != batch.SuccessCount {
		return nil, fmt.Errorf("batch %d success count mismatch: batch=%d payments=%d", batch.ID, batch.SuccessCount, booking.successfulCount)
	}
	if len(payments) != batch.SuccessCount+batch.FailedCount {
		return nil, fmt.Errorf("batch %d terminal count mismatch: batch=%d payments=%d", batch.ID, batch.SuccessCount+batch.FailedCount, len(payments))
	}
	if len(payments) != batch.TotalCount {
		return nil, fmt.Errorf("batch %d row count mismatch: total_count=%d payments=%d", batch.ID, batch.TotalCount, len(payments))
	}
	if booking.successfulCount == 0 {
		return booking, nil
	}

	codes := make([]string, 0, len(successAmounts))
	for code := range successAmounts {
		codes = append(codes, code)
	}
	transactionCodes, err := s.txnCodeRepo.FindByCodes(ctx, codes)
	if err != nil {
		return nil, fmt.Errorf("resolve successful VFIC codes: %w", err)
	}
	if len(transactionCodes) != len(codes) {
		return nil, fmt.Errorf("batch %d transaction-code mismatch: expected=%d found=%d", batch.ID, len(codes), len(transactionCodes))
	}

	seenTimesheets := make(map[uint]struct{})
	seenCodes := make(map[string]struct{}, len(transactionCodes))
	for _, transactionCode := range transactionCodes {
		expectedAmount, ok := successAmounts[transactionCode.Code]
		if !ok {
			return nil, fmt.Errorf("batch %d resolved unexpected transaction code %q", batch.ID, transactionCode.Code)
		}
		seenCodes[transactionCode.Code] = struct{}{}
		var data domain.TransactionCodeData
		if err := json.Unmarshal(transactionCode.Data, &data); err != nil {
			return nil, fmt.Errorf("decode transaction code %s: %w", transactionCode.Code, err)
		}
		if data.GetAmount() != expectedAmount {
			return nil, fmt.Errorf("transaction code %s amount mismatch: code=%d payment=%d", transactionCode.Code, data.GetAmount(), expectedAmount)
		}
		ids := data.GetTimesheetIDs()
		if len(ids) == 0 {
			return nil, fmt.Errorf("transaction code %s has no timesheet IDs", transactionCode.Code)
		}
		for _, id := range ids {
			if _, exists := seenTimesheets[id]; exists {
				continue
			}
			seenTimesheets[id] = struct{}{}
			booking.timesheetIDs = append(booking.timesheetIDs, id)
		}
	}
	if len(seenCodes) != len(codes) {
		return nil, fmt.Errorf("batch %d did not resolve every successful transaction code", batch.ID)
	}
	return booking, nil
}

func uintPointer(value *uint64) *uint {
	if value == nil {
		return nil
	}
	converted := uint(*value)
	return &converted
}
