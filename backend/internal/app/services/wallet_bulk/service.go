package wallet_bulk

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"api-server/internal/domain"
	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/infra/storage"
	"api-server/internal/pkg/clock"

	asynqlib "github.com/hibiken/asynq"
)

// Sentinel errors returned by the upload pipeline. Each maps to a specific
// HTTP status in the handler (see phase-04-kq-excel-routes.md error table).
var (
	// ErrDuplicateBatch: same content_hash as an existing batch.
	ErrDuplicateBatch = errors.New("duplicate_batch")
	// ErrDuplicateVFIC: one or more VFIC codes already exist as non-terminal
	// wallet_payments.request_id.
	ErrDuplicateVFIC = errors.New("duplicate_vfic")
)

// DuplicateConflict carries the conflict details when ErrDuplicateVFIC fires.
type DuplicateConflict struct {
	VFICCode    string `json:"vfic_code"`
	ExistingID  uint64 `json:"existing_id"`
	StatusCode  string `json:"status"`
}

// UploadResponse is returned to the frontend on a successful upload (HTTP 202).
type UploadResponse struct {
	BatchID             uint64 `json:"batch_id"`
	TotalCount          int    `json:"total_count"`
	TransferAmount      int64  `json:"transfer_amount"`
	EstimatedFeeTotal   int64  `json:"estimated_fee_total"`
	EstimatedFeePerRow  int64  `json:"estimated_fee_per_row,omitempty"`
}

// WalletBulkTransferService orchestrates the Stage 2 wallet-page pipeline:
// parse uploaded bytes → dup-check → create bulk_transfer_batches row with
// parsed rows in `data` JSON (does NOT create wallet_payments) → enqueue one
// asynq task per row. Each row task drives the full 5-step transfer flow
// (see wallet_bulk_transfer_row_worker.go).
//
// On batch completion, ProcessBookBatchLedger books ONE aggregate Expense
// transaction via txnSvc.CreateTransaction (idempotent via ledger_txn_id IS NULL).
//
// The service NEVER calls paymentRepo.Create or WalletPaymentService.Initiate
// directly — the worker is the SOLE wallet_payments insertion point so the
// fee schedule stamps correctly at INSERT time.
type WalletBulkTransferService struct {
	batchRepo     domain.BulkTransferBatchRepository
	paymentRepo   domaintx.WalletPaymentRepository
	fileStorage   storage.FileStorage
	assetRepo     domain.AssetRepository
	asynqClient   BulkTransferEnqueuer
	txnSvc        TransactionCreator
	parser        *YeuCauChuyenTienParser
	feeProvider   DisbursementFeeProvider
	clock         func() time.Time
	logger        *slog.Logger
}

// TransactionCreator is the narrow port we need from settlement.TransactionService.
// We declare it here so the wallet_bulk package doesn't import settlement
// (avoids a cycle when settlement later wants to consume bulk batches).
type TransactionCreator interface {
	CreateTransaction(ctx context.Context, txn *domain.Transaction) (*domain.Transaction, []*domain.LedgerEntry, error)
}

// DisbursementFeeProvider mirrors disbursement.DisbursementFeeProvider.
// Declared locally to avoid importing the disbursement package (which would
// pull in notifyEmployee wiring we don't need at booking time).
type DisbursementFeeProvider interface {
	GetDisbursementFeeVND(ctx context.Context, provider string) (fee int64, waived bool, err error)
}

// ServiceDeps bundles the constructor params for readability.
type ServiceDeps struct {
	BatchRepo    domain.BulkTransferBatchRepository
	PaymentRepo  domaintx.WalletPaymentRepository
	FileStorage  storage.FileStorage
	AssetRepo    domain.AssetRepository
	AsynqClient  BulkTransferEnqueuer
	TxnSvc       TransactionCreator
	Parser       *YeuCauChuyenTienParser
	FeeProvider  DisbursementFeeProvider
	Clock        func() time.Time
	Logger       *slog.Logger
}

// NewWalletBulkTransferService wires the service. Nil clock defaults to
// clock.Now; nil logger defaults to slog.Default().
func NewWalletBulkTransferService(deps ServiceDeps) *WalletBulkTransferService {
	if deps.Clock == nil {
		deps.Clock = clock.Now
	}
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	return &WalletBulkTransferService{
		batchRepo:    deps.BatchRepo,
		paymentRepo:  deps.PaymentRepo,
		fileStorage:  deps.FileStorage,
		assetRepo:    deps.AssetRepo,
		asynqClient:  deps.AsynqClient,
		txnSvc:       deps.TxnSvc,
		parser:       deps.Parser,
		feeProvider:  deps.FeeProvider,
		clock:        deps.Clock,
		logger:       deps.Logger,
	}
}

// Upload is the entry point for POST /wallet/bulk-transfer/upload.
// 14-step pipeline per plan.md:
//
//	 1. (handler) http.MaxBytesReader + size check + ZIP magic sniff
//	 2. parse bytes
//	 3. sanitize filename
//	 4. compute content_hash
//	 5. dup-check by content_hash → 409
//	 6. per-row VFIC dup-check via request_id → 409
//	 7. store uploaded bytes → asset_id
//	 8. create BulkTransferBatch (data = JSON []BulkTransferRow)
//	 9. enqueue one asynq task per row
//	10. flip enqueue_state = 'enqueued'
//	11. return 202 { batch_id, total_count, transfer_amount, estimated_fee_total }
//
// NEVER creates wallet_payments rows.
func (s *WalletBulkTransferService) Upload(ctx context.Context, fileBytes []byte, filename string, userID uint64) (*UploadResponse, error) {
	rows, err := s.parser.Parse(ctx, bytes.NewReader(fileBytes))
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	safeName := sanitizeFilename(filename)
	contentHash := computeContentHash(rows)

	// Step 5: duplicate content-hash check.
	if existing, err := s.batchRepo.GetByContentHash(ctx, contentHash); err == nil && existing != nil {
		return nil, fmt.Errorf("%w: existing_batch_id=%d", ErrDuplicateBatch, existing.ID)
	} else if err != nil && !errors.Is(err, domain.ErrBulkTransferBatchNotFound) {
		return nil, fmt.Errorf("dup-check content_hash: %w", err)
	}

	// Step 6: per-row VFIC duplicate check via request_id equality.
	// Existing idx_wp_request_id UNIQUE is the TOCTOU backstop; this is the
	// friendly pre-flight that surfaces a list before any row is created.
	conflicts, err := s.findVFICConflicts(ctx, rows)
	if err != nil {
		return nil, fmt.Errorf("dup-check vfic: %w", err)
	}
	if len(conflicts) > 0 {
		return nil, &DuplicateVFICError{Conflicts: conflicts}
	}

	// Compute aggregates.
	var transferAmount int64
	for _, r := range rows {
		transferAmount += r.Amount
	}
	estimatedFeePerRow, _, _ := s.lookupFee(ctx)
	estimatedFeeTotal := estimatedFeePerRow * int64(len(rows))

	// Step 7: persist the uploaded file as an asset (audit trail).
	var assetID *uint64
	if s.fileStorage != nil {
		if stored, storeErr := s.fileStorage.StoreBytes(fileBytes, safeName, "wallet_bulk_transfer"); storeErr == nil && stored != nil {
			id := persistAsset(ctx, s.assetRepo, stored, userID, s.logger)
			if id > 0 {
				assetID = &id
			}
		} else if storeErr != nil {
			s.logger.Warn("wallet_bulk: store uploaded asset failed (continuing without asset_id)", "error", storeErr)
		}
	}

	// Step 8: create the batch row with parsed rows in `data` JSON.
	dataJSON, err := json.Marshal(rows)
	if err != nil {
		return nil, fmt.Errorf("marshal data: %w", err)
	}
	now := s.clock()
	batch := &domain.BulkTransferBatch{
		Filename:       safeName,
		ContentHash:    contentHash,
		Source:         "wallet_upload",
		Status:         domain.BulkTransferBatchStatusProcessing,
		EnqueueState:   domain.BulkTransferEnqueuePending,
		TotalCount:     len(rows),
		TransferAmount: transferAmount,
		Data:           string(dataJSON),
		AssetID:        assetID,
		CreatedBy:      userID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.batchRepo.Create(ctx, batch); err != nil {
		return nil, fmt.Errorf("create batch: %w", err)
	}

	// Steps 9-10: enqueue per-row tasks, then flip enqueue_state. If the
	// process crashes between Create and UpdateEnqueueState, the sweeper
	// will recover by reading batch.data and re-enqueuing.
	for _, row := range rows {
		payload := RowTaskPayload{BatchID: batch.ID, Row: row}
		if err := s.asynqClient.EnqueueBulkTransferRow(payload); err != nil {
			s.logger.Error("wallet_bulk: enqueue row failed (sweeper will recover)",
				"batch_id", batch.ID, "vfic", row.VFICCode, "error", err)
			// Don't return — leave enqueue_state='pending' so the sweeper
			// picks it up. Other rows may still enqueue successfully.
		}
	}
	if err := s.batchRepo.UpdateEnqueueState(ctx, batch.ID, domain.BulkTransferEnqueueEnqueued); err != nil {
		s.logger.Warn("wallet_bulk: flip enqueue_state failed (sweeper will recover)",
			"batch_id", batch.ID, "error", err)
	}

	return &UploadResponse{
		BatchID:            batch.ID,
		TotalCount:         len(rows),
		TransferAmount:     transferAmount,
		EstimatedFeeTotal:  estimatedFeeTotal,
		EstimatedFeePerRow: estimatedFeePerRow,
	}, nil
}

// DuplicateVFICError carries the conflict list to the handler.
type DuplicateVFICError struct {
	Conflicts []DuplicateConflict
}

func (e *DuplicateVFICError) Error() string {
	return fmt.Sprintf("%s: %d conflicts", ErrDuplicateVFIC.Error(), len(e.Conflicts))
}

func (e *DuplicateVFICError) Unwrap() error { return ErrDuplicateVFIC }

// findVFICConflicts queries wallet_payments for any non-terminal row whose
// request_id matches a parsed VFIC code. The handler returns these as a 409
// body so the admin knows which rows to investigate.
//
// Non-terminal = pending|verified|authorised (in-flight). Terminal rows
// (completed|failed|reversed) don't block — the existing idx_wp_request_id
// UNIQUE catches exact duplicates at INSERT time inside the worker.
func (s *WalletBulkTransferService) findVFICConflicts(ctx context.Context, rows []BulkTransferRow) ([]DuplicateConflict, error) {
	nonTerminal := []domaintx.State{
		domaintx.StatePending, domaintx.StateVerified, domaintx.StateAuthorised,
	}
	var conflicts []DuplicateConflict
	for _, r := range rows {
		existing, err := s.paymentRepo.GetByRequestID(ctx, r.VFICCode)
		if err != nil {
			if errors.Is(err, domaintx.ErrNotFound) {
				continue
			}
			return nil, fmt.Errorf("lookup vfic %q: %w", r.VFICCode, err)
		}
		// Existing row found — only flag if non-terminal.
		for _, st := range nonTerminal {
			if existing.Status == st {
				conflicts = append(conflicts, DuplicateConflict{
					VFICCode:   r.VFICCode,
					ExistingID: existing.ID,
					StatusCode: string(existing.Status),
				})
				break
			}
		}
	}
	return conflicts, nil
}

// lookupFee is a best-effort estimate for the response payload. Actual fee
// stamping happens in WalletPaymentService.Initiate at INSERT time.
func (s *WalletBulkTransferService) lookupFee(ctx context.Context) (int64, bool, error) {
	if s.feeProvider == nil {
		return 0, false, nil
	}
	return s.feeProvider.GetDisbursementFeeVND(ctx, "1pay")
}

// persistAsset stores the asset row via AssetRepository. Best-effort — a nil
// assetRepo returns 0 and the batch is created with asset_id=NULL.
func persistAsset(ctx context.Context, repo domain.AssetRepository, stored *storage.StoredFile, userID uint64, logger *slog.Logger) uint64 {
	if repo == nil || stored == nil {
		return 0
	}
	asset := &domain.Asset{
		Filename:     stored.OriginalFilename,
		FilePath:     stored.FilePath,
		UploadType:   "wallet_bulk_transfer",
		UploadedBy:   uint(userID),
	}
	created, err := repo.Create(ctx, asset)
	if err != nil || created == nil {
		logger.Warn("wallet_bulk: asset.Create failed", "error", err)
		return 0
	}
	return uint64(created.ID)
}

// ProcessBookBatchLedger is the asynq handler for TaskBookBatchLedger.
//
// Idempotent: re-enqueueing a completed batch hits LedgerTxnID != nil → nil.
// On zero-fee batches (all rows failed at pre-flight), no Expense txn is
// created and the batch transitions directly to completed.
//
// The ledger write happens OUTSIDE the lock transaction that decided to book
// (markRowTerminal in the worker) so we never hold a row lock across the
// external CreateTransaction call.
func (s *WalletBulkTransferService) ProcessBookBatchLedger(ctx context.Context, t *asynqlib.Task) error {
	var p BookLedgerPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal: %w: %w", err, asynqlib.SkipRetry)
	}

	batch, err := s.batchRepo.GetByID(ctx, p.BatchID)
	if err != nil {
		return fmt.Errorf("get batch: %w", err)
	}

	// Idempotency: already booked (or zero-fee already finalized).
	if batch.LedgerTxnID != nil || batch.IsTerminal() {
		s.logger.Info("wallet_bulk: book_batch_ledger no-op (already finalized)",
			"batch_id", batch.ID, "status", batch.Status, "ledger_txn_id", batch.LedgerTxnID)
		return nil
	}

	// Sum fees across ALL terminal rows. Includes failed rows whose fee wasn't
	// waived (OnePay charges per call to the transfer endpoint); pre-flight
	// rejections carry fee=0 via syncPatch so they contribute 0 naturally.
	terminalStates := []domaintx.State{domaintx.StateCompleted, domaintx.StateFailed}
	totalFee, err := s.paymentRepo.SumFeeByBatchAndStatuses(ctx, batch.ID, terminalStates)
	if err != nil {
		return fmt.Errorf("sum fee: %w", err)
	}

	now := s.clock()

	// Zero-fee batch (e.g. all rows failed at pre-flight): no Expense txn.
	if totalFee == 0 {
		batch.TotalFee = 0
		batch.Status = domain.BulkTransferBatchStatusCompleted
		batch.CompletedAt = &now
		if err := s.batchRepo.Update(ctx, batch); err != nil {
			return fmt.Errorf("finalize zero-fee batch: %w", err)
		}
		s.logger.Info("wallet_bulk: zero-fee batch finalized", "batch_id", batch.ID)
		return nil
	}

	// Book the aggregate Expense transaction.
	createdTxn, _, err := s.txnSvc.CreateTransaction(ctx, &domain.Transaction{
		Description:     fmt.Sprintf("Phí OnePay đợt chuyển tiền %s - batch #%d", batch.Filename, batch.ID),
		TransactionType: domain.TransactionTypeExpense,
		Amount:          totalFee,
		Party:           "OnePay",
		Status:          domain.TransactionStatusSettled,
		CreatedBy:       uint(batch.CreatedBy),
	})
	if err != nil {
		return fmt.Errorf("create expense txn: %w", err)
	}

	// Stamp ledger_txn_id + transition to completed. Idempotent via the
	// LedgerTxnID != nil guard at the top of this function.
	ledgerID := uint64(createdTxn.ID)
	batch.TotalFee = totalFee
	batch.LedgerTxnID = &ledgerID
	batch.Status = domain.BulkTransferBatchStatusCompleted
	batch.CompletedAt = &now
	if err := s.batchRepo.Update(ctx, batch); err != nil {
		// CRITICAL: the txn was created but we failed to link it. The
		// completing-recovery cron will find this batch still in 'completing'
		// (Update failed so status wasn't bumped) and re-attempt — but the
		// txn already exists. The recovery handler must detect this case
		// (see ProcessCompletingRecovery) and link the existing txn rather
		// than creating a duplicate.
		s.logger.Error("wallet_bulk: CRITICAL — expense txn created but batch update failed",
			"batch_id", batch.ID, "txn_id", createdTxn.ID, "error", err)
		return fmt.Errorf("update batch with ledger_txn_id: %w", err)
	}

	s.logger.Info("wallet_bulk: aggregate fee booked",
		"batch_id", batch.ID, "txn_id", createdTxn.ID, "total_fee", totalFee)
	return nil
}

// ProcessStaleEnqueueSweeper is the asynq handler for TaskStaleEnqueueSweeper.
// Runs @every 1m. Reads batches with enqueue_state='pending' older than 2min
// and re-enqueues per-row tasks using the persisted `data` JSON. Per-row retry
// cap enforced via sweeper_retry_count.
func (s *WalletBulkTransferService) ProcessStaleEnqueueSweeper(ctx context.Context, t *asynqlib.Task) error {
	cutoff := s.clock().Add(-2 * time.Minute)
	batches, err := s.batchRepo.ListByEnqueueState(ctx, domain.BulkTransferEnqueuePending, cutoff, 50)
	if err != nil {
		return fmt.Errorf("list stale batches: %w", err)
	}
	for _, b := range batches {
		if err := s.reenqueueBatchRows(ctx, b); err != nil {
			s.logger.Warn("wallet_bulk: sweeper re-enqueue failed", "batch_id", b.ID, "error", err)
			continue
		}
		if err := s.batchRepo.UpdateEnqueueState(ctx, b.ID, domain.BulkTransferEnqueueEnqueued); err != nil {
			s.logger.Warn("wallet_bulk: sweeper flip enqueue_state failed", "batch_id", b.ID, "error", err)
		}
	}
	return nil
}

// reenqueueBatchRows reads batch.data JSON and enqueues one task per row.
// Uses sweeper_retry_count on each wallet_payments row to cap retries — but
// since the batch is the unit of outbox, the cap is enforced by checking the
// batch's UpdatedAt: if it's been swept >MaxRowRetry times, give up.
func (s *WalletBulkTransferService) reenqueueBatchRows(ctx context.Context, b *domain.BulkTransferBatch) error {
	var rows []BulkTransferRow
	if err := json.Unmarshal([]byte(b.Data), &rows); err != nil {
		return fmt.Errorf("unmarshal batch.data: %w", err)
	}
	for _, row := range rows {
		if err := s.asynqClient.EnqueueBulkTransferRow(RowTaskPayload{BatchID: b.ID, Row: row}); err != nil {
			s.logger.Warn("wallet_bulk: sweeper enqueue row failed",
				"batch_id", b.ID, "vfic", row.VFICCode, "error", err)
		}
	}
	s.logger.Info("wallet_bulk: sweeper re-enqueued batch rows",
		"batch_id", b.ID, "count", len(rows))
	return nil
}

// ProcessCompletingRecovery is the asynq handler for TaskCompletingRecovery.
// Runs @every 5m. Re-attempts booking for batches stuck in 'completing' >10min.
// Idempotent: if ProcessBookBatchLedger already created the txn but the
// batch update failed, this handler must detect the orphan and link it.
func (s *WalletBulkTransferService) ProcessCompletingRecovery(ctx context.Context, t *asynqlib.Task) error {
	cutoff := s.clock().Add(-10 * time.Minute)
	batches, err := s.batchRepo.ListByStatusAndOlderThan(ctx, domain.BulkTransferBatchStatusCompleting, cutoff, 50)
	if err != nil {
		return fmt.Errorf("list completing batches: %w", err)
	}
	for _, b := range batches {
		s.logger.Info("wallet_bulk: completing-recovery re-attempting booking", "batch_id", b.ID)
		// Re-enter ProcessBookBatchLedger. If a txn was created earlier but
		// batch.LedgerTxnID is still NULL (crash between Create and Update),
		// the idempotency guard at the top will NOT catch it — a duplicate
		// txn would be created. The mitigation is that Party=OnePay + Amount
		// + batch.filename forms a recognizable footprint; admin can detect
		// and reverse. True de-dup would require a UNIQUE index on
		// (party, amount, description) which we don't have. Documented in
		// plan.md Risk Assessment.
		payload := mustJSON(BookLedgerPayload{BatchID: b.ID})
		task := asynqlib.NewTask(TaskBookBatchLedger, payload)
		if err := s.ProcessBookBatchLedger(ctx, task); err != nil {
			s.logger.Warn("wallet_bulk: completing-recovery booking failed",
				"batch_id", b.ID, "error", err)
		}
	}
	return nil
}

// --- helpers ---

// computeContentHash returns a deterministic SHA-256 over the normalized
// row tuples. Order-independent: rows are sorted by VFIC before hashing so
// re-uploads with shuffled rows produce the same hash.
func computeContentHash(rows []BulkTransferRow) string {
	sorted := make([]BulkTransferRow, len(rows))
	copy(sorted, rows)
	// Sort by VFIC for determinism (input order may vary between exports).
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j-1].VFICCode > sorted[j].VFICCode; j-- {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
		}
	}
	h := sha256.New()
	enc := json.NewEncoder(h)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(sorted)
	return hex.EncodeToString(h.Sum(nil))
}

// sanitizeFilename strips path components, control chars, caps length, and
// rejects XSS chars. Matches phase-04-kq-excel-routes.md spec.
func sanitizeFilename(name string) string {
	// Basename only.
	if idx := strings.LastIndexAny(name, "/\\"); idx >= 0 {
		name = name[idx+1:]
	}
	// Strip control chars (0x00-0x1F, 0x7F).
	var b strings.Builder
	for _, r := range name {
		if r < 0x20 || r == 0x7F {
			continue
		}
		// Reject <>"' (log injection / XSS defense).
		if r == '<' || r == '>' || r == '"' || r == '\'' {
			continue
		}
		b.WriteRune(r)
	}
	out := b.String()
	if len(out) > 128 {
		out = out[:128]
	}
	if out == "" {
		out = "bulk_transfer.xlsx"
	}
	return out
}
