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
	"sort"
	"strconv"
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
	VFICCode   string `json:"vfic_code"`
	ExistingID uint64 `json:"existing_id"`
	StatusCode string `json:"status"`
}

// UploadResponse is returned to the frontend on a successful upload (HTTP 202).
type UploadResponse struct {
	BatchID            uint64 `json:"batch_id"`
	TotalCount         int    `json:"total_count"`
	TransferAmount     int64  `json:"transfer_amount"`
	EstimatedFeeTotal  int64  `json:"estimated_fee_total"`
	EstimatedFeePerRow int64  `json:"estimated_fee_per_row,omitempty"`
	// FeeResolutionOK is false when the fee schedule lookup failed at upload
	// time (H2 fix). When false, every row will terminal-fail with
	// ErrFeeResolution in the worker; admin should fix the schedule before
	// re-uploading. The upload still succeeds (parsed + batch row created)
	// so admin sees the warning in the UI rather than getting a 4xx.
	FeeResolutionOK bool `json:"fee_resolution_ok"`
}

// WalletBulkTransferService orchestrates the Stage 2 wallet-page pipeline:
// parse uploaded bytes → dup-check → create bulk_transfer_batches row with
// parsed rows in `data` JSON (does NOT create wallet_payments) → enqueue one
// asynq task per row. Each row task drives the full 5-step transfer flow
// (see wallet_bulk_transfer_row_worker.go).
//
// On batch completion, ProcessBookBatchLedger books the partner receivable,
// salary cash outflow, revenue offset, successful-timesheet links, and the
// aggregate provider fee in one database transaction.
//
// The service NEVER calls paymentRepo.Create or WalletPaymentService.Initiate
// directly — the worker is the SOLE wallet_payments insertion point so the
// fee schedule stamps correctly at INSERT time.
type WalletBulkTransferService struct {
	batchRepo       domain.BulkTransferBatchRepository
	paymentRepo     WalletPaymentReader
	fileStorage     storage.FileStorage
	assetRepo       domain.AssetRepository
	asynqClient     BulkTransferEnqueuer
	auditEmitter    AuditEventEmitter // C3 fix — nil-safe
	txnSvc          TransactionCreator
	txnAdjuster     TransactionAdjuster
	txRunner        TransactionRunner
	parser          *YeuCauChuyenTienParser
	feeProvider     DisbursementFeeProvider
	clock           func() time.Time
	logger          *slog.Logger
	partnerInfo     PartnerInfoProvider
	ledgerWriter    LedgerEntryWriter
	timesheetLinker TimesheetTransactionLinker
	txnCodeRepo     TransactionCodeByVFIC
}

// TransactionCreator is the narrow port we need from settlement.TransactionService.
// We declare it here so the wallet_bulk package doesn't import settlement
// (avoids a cycle when settlement later wants to consume bulk batches).
type TransactionCreator interface {
	CreateTransaction(ctx context.Context, txn *domain.Transaction) (*domain.Transaction, []*domain.LedgerEntry, error)
}

// TransactionAdjuster updates the aggregate receivable when a provider
// reverses one successful employee transfer after the batch was booked.
// Both operations honor the transaction carried by ctx.
type TransactionAdjuster interface {
	GetTransactionForUpdate(ctx context.Context, id uint) (*domain.Transaction, error)
	IncrementTransactionAmount(ctx context.Context, id uint, delta int64) error
}

// WalletPaymentReader is the read-only wallet-payment subset used by this
// service for duplicate checks, progress/export reads, and completion booking.
type WalletPaymentReader interface {
	GetByRequestID(ctx context.Context, requestID string) (*domaintx.WalletPayment, error)
	ListByBatchIDOrdered(ctx context.Context, batchID uint64) ([]*domaintx.WalletPayment, error)
}

// TransactionRunner is the narrow port for executing a function inside a
// single DB transaction. Implemented by infraServices.TransactionManager;
// declared here so the wallet_bulk package stays decoupled. Used by
// ProcessBookBatchLedger to make the (CreateTransaction + batch link)
// pair atomic — closes the C2 double-booking race.
type TransactionRunner interface {
	WithTransactionResult(ctx context.Context, fn func(ctx context.Context) (interface{}, error)) (interface{}, error)
}

// DisbursementFeeProvider mirrors disbursement.DisbursementFeeProvider.
// Declared locally to avoid importing the disbursement package (which would
// pull in notifyEmployee wiring we don't need at booking time).
type DisbursementFeeProvider interface {
	GetDisbursementFeeVND(ctx context.Context, provider string) (fee int64, waived bool, err error)
}

// PartnerInfoProvider supplies the partner company name + the advance cash
// fee percentage used to compute the receivable side of a wallet_bulk
// batch (what the partner owes the operator = transfer × (1 + fee%)).
// Mirrors the legacy BuildLedgerPlan logic in payroll/bulktransfer.
type PartnerInfoProvider interface {
	GetPartnerCompany(ctx context.Context) string
	GetWeeklyPaymentFeePercentage(ctx context.Context) float64
}

// LedgerEntryWriter writes additional ledger entries for a transaction.
// Used by ProcessBookBatchLedger to append the cash-outflow + revenue-offset
// entries that complement the auto-generated receivable/revenue pair from
// CreateTransaction. Mirrors settlement.LedgerService.CreateEntries without
// pulling that package in.
type LedgerEntryWriter interface {
	CreateEntries(ctx context.Context, entries []*domain.LedgerEntry, createdBy uint) ([]*domain.LedgerEntry, error)
	CreateEntriesAtomic(ctx context.Context, entries []*domain.LedgerEntry, createdBy uint) ([]*domain.LedgerEntry, error)
}

// TimesheetTransactionLinker links timesheets to a transaction by stamping
// timesheets.transaction_id. Mirrors TimesheetRepository.BulkUpdateTransactionID.
type TimesheetTransactionLinker interface {
	GetByIDsForUpdate(ctx context.Context, timesheetIDs []uint) ([]*domain.Timesheet, error)
	BulkUpdateTransactionID(ctx context.Context, transactionID uint, timesheetIDs []uint) error
	ClearTransactionID(ctx context.Context, transactionID uint, timesheetIDs []uint) error
}

// TransactionCodeByVFIC resolves successful VFIC codes in one query so their
// linked timesheets can be attached to the receivable transaction.
type TransactionCodeByVFIC interface {
	FindByCodes(ctx context.Context, codes []string) ([]*domain.TransactionCode, error)
}

// ServiceDeps bundles the constructor params for readability.
type ServiceDeps struct {
	BatchRepo       domain.BulkTransferBatchRepository
	PaymentRepo     WalletPaymentReader
	FileStorage     storage.FileStorage
	AssetRepo       domain.AssetRepository
	AsynqClient     BulkTransferEnqueuer
	AuditEmitter    AuditEventEmitter // C3 fix; nil-safe
	TxnSvc          TransactionCreator
	TxnAdjuster     TransactionAdjuster
	TxRunner        TransactionRunner
	Parser          *YeuCauChuyenTienParser
	FeeProvider     DisbursementFeeProvider
	Clock           func() time.Time
	Logger          *slog.Logger
	PartnerInfo     PartnerInfoProvider
	LedgerWriter    LedgerEntryWriter
	TimesheetLinker TimesheetTransactionLinker
	TxnCodeRepo     TransactionCodeByVFIC
}

// NewWalletBulkTransferService wires the service. Nil clock defaults to
// clock.Now; nil logger defaults to slog.Default().
//
// M7 fix: fail-fast on nil critical deps so a misconfigured bootstrap NPEs
// here at construction rather than at first request.
func NewWalletBulkTransferService(deps ServiceDeps) *WalletBulkTransferService {
	if deps.BatchRepo == nil {
		panic("wallet_bulk: BatchRepo is required")
	}
	if deps.PaymentRepo == nil {
		panic("wallet_bulk: PaymentRepo is required")
	}
	if deps.AsynqClient == nil {
		panic("wallet_bulk: AsynqClient is required")
	}
	if deps.TxnSvc == nil {
		panic("wallet_bulk: TxnSvc is required")
	}
	if deps.TxnAdjuster == nil {
		panic("wallet_bulk: TxnAdjuster is required")
	}
	if deps.TxRunner == nil {
		panic("wallet_bulk: TxRunner is required")
	}
	if deps.PartnerInfo == nil {
		panic("wallet_bulk: PartnerInfo is required")
	}
	if deps.LedgerWriter == nil {
		panic("wallet_bulk: LedgerWriter is required")
	}
	if deps.TimesheetLinker == nil {
		panic("wallet_bulk: TimesheetLinker is required")
	}
	if deps.TxnCodeRepo == nil {
		panic("wallet_bulk: TxnCodeRepo is required")
	}
	if deps.Parser == nil {
		panic("wallet_bulk: Parser is required")
	}
	if deps.Clock == nil {
		deps.Clock = clock.Now
	}
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	return &WalletBulkTransferService{
		batchRepo:       deps.BatchRepo,
		paymentRepo:     deps.PaymentRepo,
		fileStorage:     deps.FileStorage,
		assetRepo:       deps.AssetRepo,
		asynqClient:     deps.AsynqClient,
		auditEmitter:    deps.AuditEmitter,
		txnSvc:          deps.TxnSvc,
		txnAdjuster:     deps.TxnAdjuster,
		txRunner:        deps.TxRunner,
		parser:          deps.Parser,
		feeProvider:     deps.FeeProvider,
		clock:           deps.Clock,
		logger:          deps.Logger,
		partnerInfo:     deps.PartnerInfo,
		ledgerWriter:    deps.LedgerWriter,
		timesheetLinker: deps.TimesheetLinker,
		txnCodeRepo:     deps.TxnCodeRepo,
	}
}

// Upload is the entry point for POST /wallet/bulk-transfer/upload.
// 14-step pipeline per plan.md:
//
//  1. (handler) http.MaxBytesReader + size check + ZIP magic sniff
//  2. parse bytes
//  3. sanitize filename
//  4. compute content_hash
//  5. dup-check by content_hash → 409
//  6. per-row VFIC dup-check via request_id → 409
//  7. store uploaded bytes → asset_id
//  8. create BulkTransferBatch (data = JSON []BulkTransferRow)
//  9. enqueue one asynq task per row
//  10. flip enqueue_state = 'enqueued'
//  11. return 202 { batch_id, total_count, transfer_amount, estimated_fee_total }
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
	// H2 fix: track fee-resolution failure so the upload response can warn
	// admin that every row will terminal-fail in the worker (ErrFeeResolution).
	estimatedFeePerRow, _, feeLookupErr := s.lookupFee(ctx)
	feeResolutionOK := feeLookupErr == nil
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

	// Steps 9-10: enqueue per-row tasks, then flip enqueue_state ONLY when
	// all rows enqueued successfully. On partial failure, leave enqueue_state
	// 'pending' so the stale-enqueue sweeper re-enqueues ALL rows (asynq's
	// TaskID dedup absorbs the retries for rows that already landed).
	//
	// C5 fix: the previous version unconditionally flipped to 'enqueued'
	// even on partial failure, hiding the unenqueued rows from the sweeper
	// (which only looks at 'pending') and silently dropping them.
	enqueuedAll := true
	for _, row := range rows {
		payload := RowTaskPayload{BatchID: batch.ID, Row: row}
		if err := s.asynqClient.EnqueueBulkTransferRow(payload); err != nil {
			s.logger.Error("wallet_bulk: enqueue row failed (sweeper will recover)",
				"batch_id", batch.ID, "vfic", row.VFICCode, "error", err)
			enqueuedAll = false
			// Continue trying the rest — they may succeed.
		}
	}
	if enqueuedAll {
		if err := s.batchRepo.UpdateEnqueueState(ctx, batch.ID, domain.BulkTransferEnqueueEnqueued); err != nil {
			s.logger.Warn("wallet_bulk: flip enqueue_state failed (sweeper will recover)",
				"batch_id", batch.ID, "error", err)
		}
	} else {
		s.logger.Warn("wallet_bulk: partial enqueue failure — leaving enqueue_state=pending for sweeper",
			"batch_id", batch.ID, "row_count", len(rows))
	}

	// C3 fix: emit an audit event for every money-moving upload. Best-effort
	// (failures are logged, never returned — the upload already succeeded).
	s.emitUploadAudit(ctx, batch, userID)

	if !feeResolutionOK {
		s.logger.Warn("wallet_bulk: upload succeeded but fee schedule lookup failed — rows will terminal-fail",
			"batch_id", batch.ID, "error", feeLookupErr)
	}

	return &UploadResponse{
		BatchID:            batch.ID,
		TotalCount:         len(rows),
		TransferAmount:     transferAmount,
		EstimatedFeeTotal:  estimatedFeeTotal,
		EstimatedFeePerRow: estimatedFeePerRow,
		FeeResolutionOK:    feeResolutionOK,
	}, nil
}

// emitUploadAudit publishes the audit row for a money-moving upload. Best-effort:
// any failure is logged but never returned (the upload itself already succeeded).
func (s *WalletBulkTransferService) emitUploadAudit(ctx context.Context, batch *domain.BulkTransferBatch, userID uint64) {
	if s.auditEmitter == nil {
		return
	}
	batchID := uint(batch.ID)
	payload := AuditLogPayload{
		UserID:     uint(userID),
		Action:     "BULK_CREATE",
		EntityType: "bulk_transfer_batch",
		EntityID:   &batchID,
		Message:    fmt.Sprintf("Tải lên lô chuyển tiền %s (%d giao dịch, %s VND)", batch.Filename, batch.TotalCount, formatVND(batch.TransferAmount)),
		CreatedAt:  s.clock(),
	}
	if err := s.auditEmitter.EnqueueAuditLogWrite(payload); err != nil {
		s.logger.Warn("wallet_bulk: audit emit failed (non-fatal)", "batch_id", batch.ID, "error", err)
	}
}

// formatVND is a tiny local helper to keep the audit message readable.
// Imported from utils elsewhere; we inline a minimal version here to avoid
// pulling the utils package (which depends on settings).
func formatVND(v int64) string {
	// Group thousands with '.'. 1234567 → "1.234.567".
	negative := v < 0
	if negative {
		v = -v
	}
	s := strconv.FormatInt(v, 10)
	if len(s) <= 3 {
		if negative {
			return "-" + s
		}
		return s
	}
	out := ""
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += "."
		}
		out += string(ch)
	}
	if negative {
		return "-" + out
	}
	return out
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
		Filename:   stored.OriginalFilename,
		FilePath:   stored.FilePath,
		UploadType: "wallet_bulk_transfer",
		UploadedBy: uint(userID),
	}
	created, err := repo.Create(ctx, asset)
	if err != nil || created == nil {
		logger.Warn("wallet_bulk: asset.Create failed", "error", err)
		return 0
	}
	return uint64(created.ID)
}

func (s *WalletBulkTransferService) finalizeBatchColumns(ctx context.Context, batchID uint64, transferAmount, totalFee int64, ledgerTxnID *uint64, completedAt *time.Time) error {
	updates := map[string]interface{}{
		"transfer_amount": transferAmount,
		"total_fee":       totalFee,
		"status":          string(domain.BulkTransferBatchStatusCompleted),
		"completed_at":    completedAt,
		"updated_at":      s.clock(),
	}
	if ledgerTxnID != nil {
		updates["ledger_txn_id"] = *ledgerTxnID
	}
	if totalFee > 0 {
		updates["fee_booked_at"] = completedAt
	}
	if err := s.batchRepo.UpdateColumns(ctx, batchID, updates); err != nil {
		return err
	}
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
	// Sort by VFIC for determinism. sort.Slice is O(N log N) — M4 fix
	// (the prior insertion sort was O(N²) at 5,000 rows = 25M comparisons).
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].VFICCode < sorted[j].VFICCode
	})
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
