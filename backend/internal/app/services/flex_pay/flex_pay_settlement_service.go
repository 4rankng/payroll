package flex_pay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type FlexPaySettlementService struct {
	logger            *slog.Logger
	db                *gorm.DB
	requestRepo       domain.AdvancePaymentRequestRepository
	advPayRepo        domain.AdvancePaymentRepository
	ledgerRepo        domain.LedgerEntryRepository
	transactionRepo   domain.TransactionRepository
	uploadRepo        domain.SettlementUploadRepository
	getPartnerCompany func(ctx context.Context) string
}

func NewFlexPaySettlementService(
	db *gorm.DB,
	requestRepo domain.AdvancePaymentRequestRepository,
	advPayRepo domain.AdvancePaymentRepository,
	ledgerRepo domain.LedgerEntryRepository,
	transactionRepo domain.TransactionRepository,
	uploadRepo domain.SettlementUploadRepository,
	getPartnerCompany func(ctx context.Context) string,
) *FlexPaySettlementService {
	return &FlexPaySettlementService{
		logger:            observability.GetLogger(),
		db:                db,
		requestRepo:       requestRepo,
		advPayRepo:        advPayRepo,
		ledgerRepo:        ledgerRepo,
		transactionRepo:   transactionRepo,
		uploadRepo:        uploadRepo,
		getPartnerCompany: getPartnerCompany,
	}
}

type SettlementResult struct {
	Success      bool      `json:"success"`
	Message      string    `json:"message"`
	SettledCount int64     `json:"settled_count"`
	RequestIDs   []uint64  `json:"request_ids,omitempty"`
	SettledAt    time.Time `json:"settled_at"`
}

func (s *FlexPaySettlementService) ProcessSettlementFile(ctx context.Context, file *excelize.File) (*SettlementResult, error) {
	s.logger.Info("Starting reconciliation file settlement processing")

	requestIDs, err := s.extractRequestIDsFromInternalSheet(file)
	if err != nil {
		s.logger.Error("Failed to extract request IDs from INTERNAL sheet", "error", err)
		return nil, fmt.Errorf("failed to extract request IDs: %w", err)
	}

	if len(requestIDs) == 0 {
		s.logger.Info("No request IDs found in INTERNAL sheet")
		return &SettlementResult{
			Success:      true,
			Message:      "No request IDs found in the reconciliation file",
			SettledCount: 0,
			SettledAt:    clock.Now(),
		}, nil
	}

	// Idempotency: hash the request-ID set so a re-upload of the same recon
	// content short-circuits to the prior result instead of re-touching the
	// books. The row is written only on full success, so a partial-failure
	// re-upload reprocesses the failed records (already-settled ones are
	// skipped by the per-iteration status precondition inside settleOneTxn).
	fileHash, err := computeSettlementHash(requestIDs)
	if err != nil {
		return nil, fmt.Errorf("compute settlement idempotency hash: %w", err)
	}
	if prior, err := s.uploadRepo.GetByFileHash(ctx, fileHash); err != nil {
		return nil, fmt.Errorf("settlement idempotency check: %w", err)
	} else if prior != nil {
		s.logger.Info("Settlement file already processed, skipping (idempotent)",
			"file_hash", fileHash, "settled_count", prior.SettledCount)
		return &SettlementResult{
			Success:      true,
			Message:      "Settlement file already processed",
			SettledCount: prior.SettledCount,
			RequestIDs:   requestIDs,
			SettledAt:    prior.UploadedAt,
		}, nil
	}

	s.logger.Info("Extracted request IDs from file", "count", len(requestIDs), "ids", requestIDs)

	requests, err := s.requestRepo.GetByIDs(ctx, requestIDs)
	if err != nil {
		s.logger.Error("Failed to get requests by IDs", "error", err)
		return nil, fmt.Errorf("failed to get requests: %w", err)
	}

	var totalSettledAmount int64
	var forMonth string
	for _, req := range requests {
		totalSettledAmount += int64(req.RequestAmount)
		if forMonth == "" && req.AdvPayID > 0 {
			advPay, err := s.advPayRepo.GetByID(ctx, uint64(req.AdvPayID))
			if err == nil && advPay.ForMonth != "" {
				forMonth = advPay.ForMonth
			}
		}
	}

	settledAt := clock.Now()
	settledCount, err := s.requestRepo.MarkReceivableSettled(ctx, requestIDs, settledAt)
	if err != nil {
		s.logger.Error("Failed to mark requests as settled", "error", err, "count", len(requestIDs))
		return nil, fmt.Errorf("failed to mark requests as settled: %w", err)
	}

	// Settle each linked transaction in its own DB transaction. A failure rolls
	// back only that record; the loop continues so the rest of the batch still
	// settles. Failed transaction IDs surface in the result (Success=false) and
	// the run is NOT recorded in settlement_uploads, so a re-upload retries the
	// failed records while already-settled ones are skipped.
	var failedTxnIDs []uint
	if totalSettledAmount > 0 && len(requests) > 0 {
		partnerCompany := s.getPartnerCompany(ctx)

		txnAmounts := make(map[uint]int64)
		txnMap := make(map[uint]*domain.Transaction)
		for _, req := range requests {
			if req.SettlementTransactionID == nil || *req.SettlementTransactionID == 0 {
				continue
			}
			txnID := *req.SettlementTransactionID
			if _, seen := txnMap[txnID]; seen {
				txnAmounts[txnID] += int64(req.RequestAmount)
				continue
			}
			tx, err := s.transactionRepo.GetByID(ctx, txnID)
			if err != nil || tx.Status != domain.TransactionStatusPending {
				continue
			}
			txnMap[txnID] = tx
			txnAmounts[txnID] += int64(req.RequestAmount)
		}

		for txnID := range txnMap {
			if err := s.settleOneTxn(ctx, txnID, txnAmounts[txnID], partnerCompany, settledAt); err != nil {
				s.logger.Error("Settlement iteration failed — record rolled back, continuing with batch",
					"transaction_id", txnID, "error", err)
				failedTxnIDs = append(failedTxnIDs, txnID)
			}
		}

		if len(txnMap) == 0 {
			s.logger.Warn("No matching pending transactions found for settlement",
				"request_count", len(requests),
				"total_amount", totalSettledAmount)
		}
	}

	if len(failedTxnIDs) > 0 {
		// Do NOT record the idempotency row — a re-upload must reprocess these.
		s.logger.Error("Settlement completed with failures",
			"failed_count", len(failedTxnIDs), "failed_txn_ids", failedTxnIDs)
		return &SettlementResult{
			Success:      false,
			Message:      fmt.Sprintf("Settlement partially failed: %d transaction(s) rolled back: %v", len(failedTxnIDs), failedTxnIDs),
			SettledCount: settledCount,
			RequestIDs:   requestIDs,
			SettledAt:    settledAt,
		}, nil
	}

	// Full success — record the idempotency row. If this insert fails the books
	// are still correct; a re-upload reprocesses (the per-record status
	// precondition makes it a no-op), so log and continue rather than failing.
	requestIDsJSON, _ := json.Marshal(requestIDs)
	if err := s.uploadRepo.Create(ctx, &domain.SettlementUpload{
		FileHash:       fileHash,
		UploadedAt:     settledAt,
		RequestIDsJSON: string(requestIDsJSON),
		SettledCount:   settledCount,
	}); err != nil {
		s.logger.Warn("Failed to record settlement idempotency row — re-upload will reprocess",
			"file_hash", fileHash, "error", err)
	}

	s.logger.Info("Successfully settled receivables",
		"settled_count", settledCount,
		"total_ids", len(requestIDs),
		"total_amount", totalSettledAmount,
		"settled_at", settledAt)

	return &SettlementResult{
		Success:      true,
		Message:      fmt.Sprintf("Successfully settled %d receivables", settledCount),
		SettledCount: settledCount,
		RequestIDs:   requestIDs,
		SettledAt:    settledAt,
	}, nil
}

// settleOneTxn marks one transaction settled and writes its double-entry
// settlement ledger pair inside a single DB transaction with deadlock retry.
// Idempotent: a transaction no longer pending is skipped (its ledger pair was
// already written on a prior run). Mirrors SettlementEventHandler.ApplySettlement.
func (s *FlexPaySettlementService) settleOneTxn(ctx context.Context, txnID uint, amount int64, partnerCompany string, settledAt time.Time) error {
	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			txCtx := domain.WithTransactionContext(ctx, &domain.TransactionContext{
				TX:              tx,
				IsTransactional: true,
			})

			txn, err := s.transactionRepo.GetByIDForUpdate(txCtx, txnID)
			if err != nil {
				return fmt.Errorf("load transaction for update: %w", err)
			}
			// Per-record idempotency: a transaction past pending was settled on
			// a prior (possibly partial) run — skip without a duplicate ledger.
			if txn.Status != domain.TransactionStatusPending {
				s.logger.Info("Settlement: transaction not pending, skipping (idempotent)",
					"transaction_id", txnID, "status", txn.Status)
				return nil
			}

			txn.Status = domain.TransactionStatusSettled
			txn.SettledAmount = txn.Amount
			if err := s.transactionRepo.Update(txCtx, txn); err != nil {
				return fmt.Errorf("update transaction to settled: %w", err)
			}

			txIDCopy := txnID
			ledgerEntries := []*domain.LedgerEntry{
				{
					Date:          settledAt,
					Account:       domain.AccountCash,
					Party:         partnerCompany,
					Debit:         amount,
					Credit:        0,
					CreatedBy:     constants.SystemUserID,
					TransactionID: &txIDCopy,
				},
				{
					Date:          settledAt,
					Account:       domain.AccountReceivable,
					Party:         partnerCompany,
					Debit:         0,
					Credit:        amount,
					CreatedBy:     constants.SystemUserID,
					TransactionID: &txIDCopy,
				},
			}
			if err := s.ledgerRepo.CreateTransaction(txCtx, ledgerEntries); err != nil {
				return fmt.Errorf("create settlement ledger entries: %w", err)
			}
			return nil
		})
		if err == nil {
			return nil
		}
		if isDeadlockErr(err) {
			backoff := time.Duration(50*(1<<attempt)) * time.Millisecond
			jitter := time.Duration(rand.Intn(50)) * time.Millisecond
			s.logger.Warn("Settlement deadlock, retrying",
				"transaction_id", txnID,
				"attempt", attempt+1,
				"max_retries", maxRetries,
				"backoff_ms", (backoff + jitter).Milliseconds())
			time.Sleep(backoff + jitter)
			continue
		}
		return err
	}
	return fmt.Errorf("persistent deadlock for transaction %d after %d retries", txnID, maxRetries)
}

// computeSettlementHash returns the SHA-256 of the JSON-encoded, sorted
// request-ID set — a stable idempotency key for a recon upload's content.
func computeSettlementHash(requestIDs []uint64) (string, error) {
	sorted := append([]uint64(nil), requestIDs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	b, err := json.Marshal(sorted)
	if err != nil {
		return "", fmt.Errorf("marshal request ids: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// isDeadlockErr reports whether err is a MySQL deadlock / lock-wait-too-deep
// error worth retrying.
func isDeadlockErr(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "Deadlock") || strings.Contains(msg, "1213")
}

func (s *FlexPaySettlementService) extractRequestIDsFromInternalSheet(f *excelize.File) ([]uint64, error) {
	const internalSheetName = "INTERNAL"

	sheetIndex, err := f.GetSheetIndex(internalSheetName)
	if err != nil || sheetIndex < 0 {
		return nil, fmt.Errorf("INTERNAL sheet not found in the uploaded file. Please upload the correct reconciliation file")
	}

	rows, err := f.GetRows(internalSheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read INTERNAL sheet: %w", err)
	}

	var requestIDs []uint64
	for i, row := range rows {
		if len(row) == 0 || row[0] == "" {
			continue
		}

		if i == 0 && isHeaderRow(row[0]) {
			continue
		}

		id, err := strconv.ParseUint(row[0], 10, 64)
		if err != nil {
			s.logger.Warn("Failed to parse request ID", "row", i+1, "value", row[0], "error", err)
			continue
		}

		requestIDs = append(requestIDs, id)
	}

	return requestIDs, nil
}

func isHeaderRow(value string) bool {
	if strings.HasPrefix(value, "type:") {
		return true
	}
	headerPatterns := []string{"request_id", "id", "ID", "Request ID", "REQUEST_ID"}
	for _, pattern := range headerPatterns {
		if value == pattern {
			return true
		}
	}
	return false
}
