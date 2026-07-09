package workers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/domain/wallet"
	"api-server/internal/pkg/clock"
	bizconst "api-server/internal/pkg/constants"
)

// TaskWalletSettlement is the asynq task type for the EOD wallet settlement cron.
const TaskWalletSettlement = "wallet:settlement"

// WalletSettlementWorker consolidates completed wallet payments into one
// Transaction + ledger entries per completion day, following the same pattern
// as the bulk transfer result upload (ProcessBankResult). Each day's
// create-transaction → link-APRs → write-ledger sequence runs inside a single
// database transaction, so a partial failure can never leave APRs marked settled
// without their ledger entries.
type WalletSettlementWorker struct {
	walletPaymentRepo     wallet.WalletPaymentRepository
	advancePaymentReqRepo domain.AdvancePaymentRequestRepository
	transactionRepo       domain.TransactionRepository
	ledgerRepo            domain.LedgerEntryRepository
	txManager             domain.TransactionManager
	logger                *slog.Logger
}

// NewWalletSettlementWorker creates a new EOD wallet settlement worker.
func NewWalletSettlementWorker(
	walletPaymentRepo wallet.WalletPaymentRepository,
	advancePaymentReqRepo domain.AdvancePaymentRequestRepository,
	transactionRepo domain.TransactionRepository,
	ledgerRepo domain.LedgerEntryRepository,
	txManager domain.TransactionManager,
	logger *slog.Logger,
) *WalletSettlementWorker {
	return &WalletSettlementWorker{
		walletPaymentRepo:     walletPaymentRepo,
		advancePaymentReqRepo: advancePaymentReqRepo,
		transactionRepo:       transactionRepo,
		ledgerRepo:            ledgerRepo,
		txManager:             txManager,
		logger:                logger,
	}
}

// ProcessJob ledger-settles completed wallet payments. It is self-healing:
// rather than only ever looking at yesterday, it settles every stranded
// completed payment (settlement_transaction_id IS NULL), grouped by completion
// day. Any day whose single pickup run was missed — process down at 00:01,
// partial failure, or a payment completed after the day's window closed — is
// backfilled on the next run. Each day settles in its own transaction; a failing
// day is logged and skipped rather than aborting the remaining days, and the run
// returns an aggregated error so asynq retries. The TaskID on the enqueue path
// keeps two runs from racing.
func (w *WalletSettlementWorker) ProcessJob(ctx context.Context) error {
	// Query every stranded completed-unsettled payment (any completion day),
	// oldest first. A zero date means "all dates" in the repository.
	payments, err := w.walletPaymentRepo.GetCompletedUnsettled(ctx, time.Time{})
	if err != nil {
		return fmt.Errorf("wallet settlement: query unsettled payments: %w", err)
	}
	if len(payments) == 0 {
		w.logger.Info("wallet settlement: no unsettled payments")
		return nil
	}

	// Group by completion day so each day collapses into one
	// "Wallet disbursement YYYY-MM-DD" transaction.
	groups := make(map[string][]*wallet.WalletPayment)
	for _, p := range payments {
		day := w.completionDay(p)
		groups[day] = append(groups[day], p)
	}
	dates := make([]string, 0, len(groups))
	for day := range groups {
		dates = append(dates, day)
	}
	sort.Strings(dates)

	var errs []error
	for _, day := range dates {
		if err := w.settleDay(ctx, day, groups[day]); err != nil {
			w.logger.Error("wallet settlement: day failed, continuing with remaining days",
				"date", day, "error", err)
			errs = append(errs, fmt.Errorf("%s: %w", day, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("wallet settlement: %d day(s) failed: %w", len(errs), errors.Join(errs...))
	}
	return nil
}

// settleDay ledger-settles one completion day's stranded payments into a single
// transaction. If a pending transaction already exists for that day (a prior run
// created it but failed before linking the APRs / writing ledger entries), the
// payments are linked to it and the ledger entries are created; otherwise a new
// transaction is created. The whole create/resolve → link → ledger sequence runs
// in one DB transaction, so a failure between steps rolls back — APRs are never
// marked settled without their ledger entries.
func (w *WalletSettlementWorker) settleDay(ctx context.Context, day string, payments []*wallet.WalletPayment) error {
	description := fmt.Sprintf("Wallet disbursement %s", day)

	entityIDs := make([]uint64, 0, len(payments))
	for _, p := range payments {
		if p.EntityID != nil {
			entityIDs = append(entityIDs, *p.EntityID)
		}
	}
	if len(entityIDs) == 0 {
		w.logger.Warn("wallet settlement: no payments with entity_id", "date", day, "payments", len(payments))
		return nil
	}

	// Lean fetch: the worker only needs ID + NetAmount + Fee, so skip the
	// relation preloads GetByIDs would eagerly load.
	requests, err := w.advancePaymentReqRepo.GetByIDsLean(ctx, entityIDs)
	if err != nil {
		return fmt.Errorf("wallet settlement: fetch advance requests: %w", err)
	}

	// Settle only the APRs we could fetch: link and sum the found subset so the
	// ledger never understates the receivable. Payments whose APR is unresolved
	// stay stranded (settlement_transaction_id IS NULL) and retry next run.
	foundIDs := make([]uint64, 0, len(requests))
	var totalNet, totalFee int64
	for _, r := range requests {
		foundIDs = append(foundIDs, uint64(r.ID))
		totalNet += int64(r.NetAmount)
		totalFee += int64(r.Fee)
	}
	if len(foundIDs) == 0 {
		w.logger.Warn("wallet settlement: no resolvable APRs for day", "date", day, "payments", len(entityIDs))
		return nil
	}
	if missing := len(entityIDs) - len(foundIDs); missing > 0 {
		w.logger.Warn("wallet settlement: some APRs unresolved, settling found subset",
			"date", day, "expected", len(entityIDs), "found", len(foundIDs))
	}
	receivable := totalNet + totalFee

	// Resolve the day's transaction outside the write tx (read), then perform the
	// create/resolve → link → ledger writes atomically. The shared TaskID on the
	// enqueue path keeps two runs from racing this check-then-act.
	existing, err := w.transactionRepo.GetPendingByDescription(ctx, description)
	if err != nil && !domain.IsNotFoundError(err) {
		return fmt.Errorf("wallet settlement: date guard check: %w", err)
	}

	return w.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		entries := domain.BuildDisbursementSettlementEntries(
			0, totalNet, totalFee, bizconst.PartnerCompany, constants.SystemUserID, clock.Now())

		if existing != nil {
			// Recover the dangling transaction: link the day's APRs and write the
			// ledger rows. Grow its Amount only when appending to an already-ledgered
			// day (a late payment). If the transaction has no ledger entries yet — a
			// prior run created it but never wrote the ledger — its Amount already
			// reflects these payments, so incrementing would double-count.
			w.logger.Warn("wallet settlement: transaction already exists for date, recovering",
				"date", day, "transaction_id", existing.ID, "payments", len(foundIDs))
			if err := w.advancePaymentReqRepo.UpdateSettlementTransactionID(txCtx, foundIDs, existing.ID); err != nil {
				return fmt.Errorf("wallet settlement: link requests to existing transaction: %w", err)
			}
			if len(existing.LedgerEntries) > 0 {
				if err := w.transactionRepo.IncrementAmount(txCtx, existing.ID, receivable); err != nil {
					return fmt.Errorf("wallet settlement: increment existing transaction amount: %w", err)
				}
			}
			for i := range entries {
				entries[i].TransactionID = &existing.ID
			}
			if err := w.ledgerRepo.CreateTransaction(txCtx, entries); err != nil {
				return fmt.Errorf("wallet settlement: create ledger entries: %w", err)
			}
			w.logger.Info("wallet settlement: recovered existing transaction",
				"date", day, "transaction_id", existing.ID,
				"payments", len(foundIDs), "receivable", receivable, "cash_out", totalNet, "fee", totalFee)
			return nil
		}

		w.logger.Info("wallet settlement: processing",
			"date", day, "payments", len(foundIDs),
			"receivable", receivable, "cash_out", totalNet, "fee", totalFee)
		transaction := &domain.Transaction{
			Amount:          receivable,
			TransactionType: domain.TransactionTypeRevenue,
			Description:     description,
			Status:          domain.TransactionStatusPending,
			Party:           bizconst.PartnerCompany,
			CreatedBy:       constants.SystemUserID,
		}
		if err := w.transactionRepo.Create(txCtx, transaction); err != nil {
			return fmt.Errorf("wallet settlement: create transaction: %w", err)
		}
		if err := w.advancePaymentReqRepo.UpdateSettlementTransactionID(txCtx, foundIDs, transaction.ID); err != nil {
			return fmt.Errorf("wallet settlement: link requests to transaction: %w", err)
		}
		for i := range entries {
			entries[i].TransactionID = &transaction.ID
		}
		if err := w.ledgerRepo.CreateTransaction(txCtx, entries); err != nil {
			return fmt.Errorf("wallet settlement: create ledger entries: %w", err)
		}
		w.logger.Info("wallet settlement: completed",
			"date", day, "transaction_id", transaction.ID,
			"payments", len(foundIDs), "receivable", receivable, "cash_out", totalNet, "fee", totalFee)
		return nil
	})
}

// completionDay returns the YYYY-MM-DD completion day of a payment, preferring
// settled_at and falling back to created_at. The timestamps are read through the
// loc=Local DSN, so they carry Asia/Ho_Chi_Minh and Format yields the local day.
// A completed payment missing settled_at is a data anomaly; warn so it surfaces.
func (w *WalletSettlementWorker) completionDay(p *wallet.WalletPayment) string {
	t := p.CreatedAt
	if p.SettledAt != nil {
		t = *p.SettledAt
	} else {
		w.logger.Warn("wallet settlement: completed payment missing settled_at, falling back to created_at",
			"wallet_payment_id", p.ID, "created_at", t.Format(time.RFC3339))
	}
	return t.Format("2006-01-02")
}
