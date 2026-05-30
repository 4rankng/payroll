package workers

import (
	"context"
	"fmt"
	"log/slog"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/domain/wallet"
	"api-server/internal/pkg/clock"
	bizconst "api-server/internal/pkg/constants"
)

// TaskWalletSettlement is the asynq task type for the EOD wallet settlement cron.
const TaskWalletSettlement = "wallet:settlement"

// WalletSettlementWorker consolidates completed wallet payments from the
// previous day into a single Transaction + ledger entries, following the
// same pattern as the bulk transfer result upload (ProcessBankResult).
type WalletSettlementWorker struct {
	walletPaymentRepo     wallet.WalletPaymentRepository
	advancePaymentReqRepo domain.AdvancePaymentRequestRepository
	transactionRepo       domain.TransactionRepository
	ledgerRepo            domain.LedgerEntryRepository
	logger                *slog.Logger
}

// NewWalletSettlementWorker creates a new EOD wallet settlement worker.
func NewWalletSettlementWorker(
	walletPaymentRepo wallet.WalletPaymentRepository,
	advancePaymentReqRepo domain.AdvancePaymentRequestRepository,
	transactionRepo domain.TransactionRepository,
	ledgerRepo domain.LedgerEntryRepository,
	logger *slog.Logger,
) *WalletSettlementWorker {
	return &WalletSettlementWorker{
		walletPaymentRepo:     walletPaymentRepo,
		advancePaymentReqRepo: advancePaymentReqRepo,
		transactionRepo:       transactionRepo,
		ledgerRepo:            ledgerRepo,
		logger:                logger,
	}
}

// ProcessJob runs the EOD wallet settlement for yesterday.
// It follows the same pattern as ProcessBankResult in admin_bulk_transfer.go:
// direct repo calls, 3 ledger entries, no event publishing.
func (w *WalletSettlementWorker) ProcessJob(ctx context.Context) error {
	yesterday := clock.Now().AddDate(0, 0, -1)
	description := fmt.Sprintf("Wallet disbursement %s", yesterday.Format("2006-01-02"))

	// Step 1: Date guard — skip if Transaction already exists for this date
	existing, err := w.transactionRepo.GetPendingByDescription(ctx, description)
	if err != nil {
		if !domain.IsNotFoundError(err) {
			return fmt.Errorf("wallet settlement: date guard check: %w", err)
		}
		// NotFound is expected — means no Transaction exists yet for this date
	}
	if existing != nil {
		w.logger.Info("wallet settlement: already processed",
			"date", yesterday.Format("2006-01-02"),
			"transaction_id", existing.ID,
		)
		return nil
	}

	// Step 2: Query completed wallet payments whose APRs have no settlement_transaction_id
	payments, err := w.walletPaymentRepo.GetCompletedUnsettled(ctx, yesterday)
	if err != nil {
		return fmt.Errorf("wallet settlement: query unsettled payments: %w", err)
	}
	if len(payments) == 0 {
		w.logger.Info("wallet settlement: no unsettled payments", "date", yesterday.Format("2006-01-02"))
		return nil
	}

	// Step 3: Fetch linked advance payment requests and aggregate amounts
	entityIDs := make([]uint64, 0, len(payments))
	for _, p := range payments {
		if p.EntityID != nil {
			entityIDs = append(entityIDs, *p.EntityID)
		}
	}
	if len(entityIDs) == 0 {
		w.logger.Info("wallet settlement: no payments with entity_id", "date", yesterday.Format("2006-01-02"))
		return nil
	}
	requests, err := w.advancePaymentReqRepo.GetByIDs(ctx, entityIDs)
	if err != nil {
		return fmt.Errorf("wallet settlement: fetch advance requests: %w", err)
	}
	if len(requests) != len(entityIDs) {
		w.logger.Warn("wallet settlement: some APRs not found",
			"expected", len(entityIDs),
			"found", len(requests),
		)
	}

	var grandTotalNetAmount, grandTotalFee int64
	for _, r := range requests {
		grandTotalNetAmount += int64(r.NetAmount)
		grandTotalFee += int64(r.Fee)
	}
	receivableAmount := grandTotalNetAmount + grandTotalFee

	w.logger.Info("wallet settlement: processing",
		"date", yesterday.Format("2006-01-02"),
		"payments", len(payments),
		"receivable", receivableAmount,
		"cash_out", grandTotalNetAmount,
		"fee", grandTotalFee,
	)

	// Step 4: Create Transaction (direct repo call, no events — same as ProcessBankResult)
	now := clock.Now()
	partnerCompany := bizconst.PartnerCompany

	transaction := &domain.Transaction{
		Amount:          receivableAmount,
		TransactionType: domain.TransactionTypeRevenue,
		Description:     description,
		Status:          domain.TransactionStatusPending,
		Party:           partnerCompany,
		CreatedBy:       constants.SystemUserID,
	}
	if err := w.transactionRepo.Create(ctx, transaction); err != nil {
		return fmt.Errorf("wallet settlement: create transaction: %w", err)
	}

	// Step 5: Link APRs to the transaction
	if err := w.advancePaymentReqRepo.UpdateSettlementTransactionID(ctx, entityIDs, transaction.ID); err != nil {
		return fmt.Errorf("wallet settlement: link requests to transaction: %w", err)
	}

	// Step 6: Create 3 ledger entries (same as ProcessBankResult)
	ledgerEntries := []*domain.LedgerEntry{
		{
			Date:          now,
			Account:       domain.AccountCash,
			Party:         "Nhân viên",
			Debit:         0,
			Credit:        grandTotalNetAmount,
			CreatedBy:     constants.SystemUserID,
			TransactionID: &transaction.ID,
		},
		{
			Date:          now,
			Account:       domain.AccountReceivable,
			Party:         partnerCompany,
			Debit:         receivableAmount,
			Credit:        0,
			CreatedBy:     constants.SystemUserID,
			TransactionID: &transaction.ID,
		},
		{
			Date:          now,
			Account:       domain.AccountRevenue,
			Party:         partnerCompany,
			Debit:         0,
			Credit:        grandTotalFee,
			CreatedBy:     constants.SystemUserID,
			TransactionID: &transaction.ID,
		},
	}
	if err := w.ledgerRepo.CreateTransaction(ctx, ledgerEntries); err != nil {
		return fmt.Errorf("wallet settlement: create ledger entries: %w", err)
	}

	w.logger.Info("wallet settlement: completed",
		"date", yesterday.Format("2006-01-02"),
		"transaction_id", transaction.ID,
		"payments", len(payments),
		"receivable", receivableAmount,
		"cash_out", grandTotalNetAmount,
		"fee", grandTotalFee,
	)

	return nil
}
