package wallet_bulk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
	domaintx "api-server/internal/domain/transactions"

	"github.com/hibiken/asynq"
	"github.com/xuri/excelize/v2"
)

type bookingPaymentReader struct{ rows []*domaintx.WalletPayment }

func (r *bookingPaymentReader) GetByRequestID(context.Context, string) (*domaintx.WalletPayment, error) {
	return nil, domaintx.ErrNotFound
}
func (r *bookingPaymentReader) ListByBatchIDOrdered(context.Context, uint64) ([]*domaintx.WalletPayment, error) {
	return r.rows, nil
}

type bookingCodeRepo struct{ codes []*domain.TransactionCode }

func (r *bookingCodeRepo) FindByCodes(context.Context, []string) ([]*domain.TransactionCode, error) {
	return r.codes, nil
}

type bookingBatchRepo struct {
	domain.BulkTransferBatchRepository
	batch   *domain.BulkTransferBatch
	updates map[string]interface{}
}

func (r *bookingBatchRepo) GetByID(context.Context, uint64) (*domain.BulkTransferBatch, error) {
	return r.batch, nil
}
func (r *bookingBatchRepo) GetByIDForUpdate(context.Context, uint64) (*domain.BulkTransferBatch, error) {
	return r.batch, nil
}
func (r *bookingBatchRepo) UpdateColumns(_ context.Context, _ uint64, updates map[string]interface{}) error {
	r.updates = updates
	if value, ok := updates["status"].(string); ok {
		r.batch.Status = domain.BulkTransferBatchStatus(value)
	}
	if value, ok := updates["success_count"].(int); ok {
		r.batch.SuccessCount = value
	}
	if value, ok := updates["failed_count"].(int); ok {
		r.batch.FailedCount = value
	}
	if value, ok := updates["transfer_amount"].(int64); ok {
		r.batch.TransferAmount = value
	}
	if value, ok := updates["ledger_txn_id"].(uint64); ok {
		r.batch.LedgerTxnID = &value
	}
	return nil
}

type bookingTxnCreator struct{ created []*domain.Transaction }

func (c *bookingTxnCreator) CreateTransaction(_ context.Context, txn *domain.Transaction) (*domain.Transaction, []*domain.LedgerEntry, error) {
	txn.ID = uint(len(c.created) + 101)
	c.created = append(c.created, txn)
	return txn, nil, nil
}

type bookingRunner struct{ calls int }

func (r *bookingRunner) WithTransactionResult(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	r.calls++
	return fn(ctx)
}

type bookingSettings struct{}

func (bookingSettings) GetPartnerCompany(context.Context) string              { return "VFIC Manpower" }
func (bookingSettings) GetWeeklyPaymentFeePercentage(context.Context) float64 { return 0.02 }

type bookingLedgerWriter struct{ entries []*domain.LedgerEntry }

func (w *bookingLedgerWriter) CreateEntries(_ context.Context, entries []*domain.LedgerEntry, _ uint) ([]*domain.LedgerEntry, error) {
	w.entries = append(w.entries, entries...)
	return entries, nil
}
func (w *bookingLedgerWriter) CreateEntriesAtomic(_ context.Context, entries []*domain.LedgerEntry, _ uint) ([]*domain.LedgerEntry, error) {
	w.entries = append(w.entries, entries...)
	return entries, nil
}

type bookingTimesheetLinker struct {
	txnID       uint
	ids         []uint
	linkedTxnID *uint
	cleared     []uint
}

func (l *bookingTimesheetLinker) GetByIDsForUpdate(_ context.Context, ids []uint) ([]*domain.Timesheet, error) {
	result := make([]*domain.Timesheet, 0, len(ids))
	for _, id := range ids {
		result = append(result, &domain.Timesheet{ID: id, TransactionID: l.linkedTxnID})
	}
	return result, nil
}

func (l *bookingTimesheetLinker) ClearTransactionID(_ context.Context, transactionID uint, ids []uint) error {
	if l.linkedTxnID == nil || *l.linkedTxnID != transactionID {
		return fmt.Errorf("unexpected transaction link")
	}
	l.cleared = append([]uint(nil), ids...)
	l.linkedTxnID = nil
	return nil
}

type bookingTxnAdjuster struct{ txn *domain.Transaction }

func (a *bookingTxnAdjuster) GetTransactionForUpdate(context.Context, uint) (*domain.Transaction, error) {
	return a.txn, nil
}
func (a *bookingTxnAdjuster) IncrementTransactionAmount(_ context.Context, _ uint, delta int64) error {
	a.txn.Amount += delta
	return nil
}

func (l *bookingTimesheetLinker) BulkUpdateTransactionID(_ context.Context, transactionID uint, ids []uint) error {
	l.txnID = transactionID
	l.ids = append([]uint(nil), ids...)
	return nil
}

func transactionCode(t *testing.T, code string, amount int64, ids ...uint) *domain.TransactionCode {
	t.Helper()
	data, err := json.Marshal(domain.TransactionCodeData{
		WeeklyPay: &domain.CyclePayData{Amount: amount, TimesheetIDs: ids},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &domain.TransactionCode{Code: code, Data: data}
}

func TestPrepareBatchBooking_UsesOnlySuccessfulTransfersAndLinksTheirTimesheets(t *testing.T) {
	service := &WalletBulkTransferService{
		paymentRepo: &bookingPaymentReader{rows: []*domaintx.WalletPayment{
			{ID: 1, RequestID: "VFICa", RequestedAmount: 1_000_000, Fee: 3_850, Status: domaintx.StateCompleted},
			{ID: 2, RequestID: "VFICb", RequestedAmount: 500_000, Fee: 3_850, Status: domaintx.StateFailed},
		}},
		txnCodeRepo: &bookingCodeRepo{codes: []*domain.TransactionCode{
			transactionCode(t, "VFICa", 1_000_000, 10, 11),
		}},
	}
	batch := &domain.BulkTransferBatch{ID: 7, TotalCount: 2, SuccessCount: 1, FailedCount: 1}

	got, err := service.prepareBatchBooking(context.Background(), batch)
	if err != nil {
		t.Fatalf("prepareBatchBooking: %v", err)
	}
	if got.transferAmount != 1_000_000 {
		t.Fatalf("transfer amount = %d, want 1000000", got.transferAmount)
	}
	if got.totalFee != 7_700 {
		t.Fatalf("total fee = %d, want 7700", got.totalFee)
	}
	if len(got.timesheetIDs) != 2 || got.timesheetIDs[0] != 10 || got.timesheetIDs[1] != 11 {
		t.Fatalf("timesheet IDs = %v, want [10 11]", got.timesheetIDs)
	}
}

func TestProcessBookBatchLedger_ZeroProviderFeeStillBooksReceivable(t *testing.T) {
	assetID := uint64(330)
	batch := &domain.BulkTransferBatch{
		ID: 1, Filename: "Yeu_cau.xlsx", Status: domain.BulkTransferBatchStatusCompleting,
		TotalCount: 1, SuccessCount: 1, AssetID: &assetID, CreatedBy: 9,
	}
	batchRepo := &bookingBatchRepo{batch: batch}
	txnCreator := &bookingTxnCreator{}
	runner := &bookingRunner{}
	ledger := &bookingLedgerWriter{}
	linker := &bookingTimesheetLinker{}
	service := &WalletBulkTransferService{
		batchRepo: batchRepo,
		paymentRepo: &bookingPaymentReader{rows: []*domaintx.WalletPayment{
			{ID: 1, RequestID: "VFICa", RequestedAmount: 1_192_500, Status: domaintx.StateCompleted},
		}},
		txnSvc: txnCreator, txRunner: runner, partnerInfo: bookingSettings{},
		ledgerWriter: ledger, timesheetLinker: linker,
		txnCodeRepo: &bookingCodeRepo{codes: []*domain.TransactionCode{
			transactionCode(t, "VFICa", 1_192_500, 31, 32),
		}},
		clock:  func() time.Time { return time.Date(2026, 7, 19, 14, 9, 4, 0, time.Local) },
		logger: slog.Default(),
	}
	payload, _ := json.Marshal(BookLedgerPayload{BatchID: 1})

	if err := service.ProcessBookBatchLedger(context.Background(), asynq.NewTask(TaskBookBatchLedger, payload)); err != nil {
		t.Fatalf("ProcessBookBatchLedger: %v", err)
	}
	if len(txnCreator.created) != 1 || txnCreator.created[0].TransactionType != domain.TransactionTypeRevenue {
		t.Fatalf("transactions = %#v, want one revenue transaction", txnCreator.created)
	}
	if txnCreator.created[0].Amount != 1_216_350 {
		t.Fatalf("receivable = %d, want 1216350", txnCreator.created[0].Amount)
	}
	if txnCreator.created[0].AssetID == nil || *txnCreator.created[0].AssetID != 330 {
		t.Fatalf("asset ID = %v, want 330", txnCreator.created[0].AssetID)
	}
	if len(ledger.entries) != 2 || ledger.entries[0].Credit != 1_192_500 || ledger.entries[1].Debit != 1_192_500 {
		t.Fatalf("manual entries = %#v", ledger.entries)
	}
	if ledger.entries[0].TransactionID == nil || *ledger.entries[0].TransactionID != 101 ||
		ledger.entries[1].TransactionID == nil || *ledger.entries[1].TransactionID != 101 {
		t.Fatalf("manual entries are not linked to receivable transaction: %#v", ledger.entries)
	}
	if linker.txnID != 101 || len(linker.ids) != 2 {
		t.Fatalf("timesheet link txn=%d ids=%v", linker.txnID, linker.ids)
	}
	if batchRepo.updates["ledger_txn_id"] != uint64(101) || batchRepo.updates["status"] != "completed" {
		t.Fatalf("batch updates = %#v", batchRepo.updates)
	}

	if err := service.ProcessBookBatchLedger(context.Background(), asynq.NewTask(TaskBookBatchLedger, payload)); err != nil {
		t.Fatalf("idempotent rerun: %v", err)
	}
	if runner.calls != 1 || len(txnCreator.created) != 1 {
		t.Fatalf("rerun booked again: runner=%d txns=%d", runner.calls, len(txnCreator.created))
	}
}

func TestProcessBookBatchLedger_PartialSuccessExcludesFailedEmployeeFromReceivableAndTimesheets(t *testing.T) {
	batch := &domain.BulkTransferBatch{
		ID: 2, Filename: "mixed.xlsx", Status: domain.BulkTransferBatchStatusCompleting,
		TotalCount: 2, SuccessCount: 1, FailedCount: 1, CreatedBy: 9,
	}
	batchRepo := &bookingBatchRepo{batch: batch}
	txnCreator := &bookingTxnCreator{}
	ledger := &bookingLedgerWriter{}
	linker := &bookingTimesheetLinker{}
	service := &WalletBulkTransferService{
		batchRepo: batchRepo,
		paymentRepo: &bookingPaymentReader{rows: []*domaintx.WalletPayment{
			{ID: 1, RequestID: "VFICpaid", RequestedAmount: 1_000_000, Fee: 3_850, Status: domaintx.StateCompleted},
			{ID: 2, RequestID: "VFICfailed", RequestedAmount: 500_000, Fee: 3_850, Status: domaintx.StateFailed},
		}},
		txnSvc: txnCreator, txRunner: &bookingRunner{}, partnerInfo: bookingSettings{},
		ledgerWriter: ledger, timesheetLinker: linker,
		txnCodeRepo: &bookingCodeRepo{codes: []*domain.TransactionCode{
			transactionCode(t, "VFICpaid", 1_000_000, 41, 42),
		}},
		clock:  func() time.Time { return time.Date(2026, 7, 19, 15, 0, 0, 0, time.Local) },
		logger: slog.Default(),
	}
	payload, _ := json.Marshal(BookLedgerPayload{BatchID: 2})

	if err := service.ProcessBookBatchLedger(context.Background(), asynq.NewTask(TaskBookBatchLedger, payload)); err != nil {
		t.Fatal(err)
	}
	if len(txnCreator.created) != 2 {
		t.Fatalf("transactions = %d, want receivable + provider fee", len(txnCreator.created))
	}
	if txnCreator.created[0].Amount != 1_020_000 {
		t.Fatalf("receivable = %d, want successful 1000000 + 2%% only", txnCreator.created[0].Amount)
	}
	if txnCreator.created[1].TransactionType != domain.TransactionTypeExpense || txnCreator.created[1].Amount != 7_700 {
		t.Fatalf("provider fee transaction = %#v", txnCreator.created[1])
	}
	if len(ledger.entries) != 2 || ledger.entries[0].Credit != 1_000_000 || ledger.entries[1].Debit != 1_000_000 {
		t.Fatalf("manual entries included failed employee: %#v", ledger.entries)
	}
	if len(linker.ids) != 2 || linker.ids[0] != 41 || linker.ids[1] != 42 {
		t.Fatalf("linked timesheets = %v, want only paid employee [41 42]", linker.ids)
	}
}

func TestHandleReversedPayment_AdjustsOnlyReversedEmployeeAfterBooking(t *testing.T) {
	batchID := uint64(8)
	ledgerTxnID := uint64(501)
	reversed := &domaintx.WalletPayment{
		ID: 2, RequestID: "VFICreturned", RequestedAmount: 500_000,
		Status: domaintx.StateReversed, BulkTransferBatchID: &batchID,
	}
	batch := &domain.BulkTransferBatch{
		ID: batchID, Status: domain.BulkTransferBatchStatusCompleted,
		TotalCount: 2, SuccessCount: 2, TransferAmount: 1_500_000,
		LedgerTxnID: &ledgerTxnID, CreatedBy: 9,
	}
	linkedTxnID := uint(ledgerTxnID)
	linker := &bookingTimesheetLinker{linkedTxnID: &linkedTxnID}
	ledger := &bookingLedgerWriter{}
	adjuster := &bookingTxnAdjuster{txn: &domain.Transaction{
		ID: uint(ledgerTxnID), TransactionType: domain.TransactionTypeRevenue,
		Amount: 1_530_000, Party: "VFIC Manpower",
	}}
	service := &WalletBulkTransferService{
		batchRepo: &bookingBatchRepo{batch: batch},
		paymentRepo: &bookingPaymentReader{rows: []*domaintx.WalletPayment{
			{ID: 1, RequestID: "VFICpaid", RequestedAmount: 1_000_000, Status: domaintx.StateCompleted, BulkTransferBatchID: &batchID},
			reversed,
		}},
		txnAdjuster: adjuster, txRunner: &bookingRunner{}, ledgerWriter: ledger,
		timesheetLinker: linker,
		txnCodeRepo: &bookingCodeRepo{codes: []*domain.TransactionCode{
			transactionCode(t, "VFICreturned", 500_000, 71, 72),
		}},
		clock: func() time.Time { return time.Date(2026, 7, 19, 18, 0, 0, 0, time.Local) },
	}

	if err := service.HandleReversedPayment(context.Background(), reversed); err != nil {
		t.Fatal(err)
	}
	if adjuster.txn.Amount != 1_020_000 {
		t.Fatalf("remaining receivable = %d, want 1020000", adjuster.txn.Amount)
	}
	if batch.SuccessCount != 1 || batch.FailedCount != 1 || batch.TransferAmount != 1_000_000 {
		t.Fatalf("batch totals success=%d failed=%d transfer=%d", batch.SuccessCount, batch.FailedCount, batch.TransferAmount)
	}
	if len(linker.cleared) != 2 || linker.cleared[0] != 71 || linker.cleared[1] != 72 {
		t.Fatalf("cleared timesheets = %v, want [71 72]", linker.cleared)
	}
	if len(ledger.entries) != 4 || ledger.entries[0].Credit != 510_000 ||
		ledger.entries[1].Debit != 510_000 || ledger.entries[2].Debit != 500_000 ||
		ledger.entries[3].Credit != 500_000 {
		t.Fatalf("reversal entries = %#v", ledger.entries)
	}

	if err := service.HandleReversedPayment(context.Background(), reversed); err != nil {
		t.Fatalf("idempotent reversal retry: %v", err)
	}
	if len(ledger.entries) != 4 {
		t.Fatalf("reversal retry wrote entries again: %d", len(ledger.entries))
	}
}

func TestHandleReversedPayment_ReconcilesConcurrentReversalsTogether(t *testing.T) {
	batchID := uint64(9)
	ledgerTxnID := uint64(601)
	reversedA := &domaintx.WalletPayment{ID: 2, RequestID: "VFICreturnA", RequestedAmount: 500_000, Status: domaintx.StateReversed, BulkTransferBatchID: &batchID}
	reversedB := &domaintx.WalletPayment{ID: 3, RequestID: "VFICreturnB", RequestedAmount: 500_000, Status: domaintx.StateReversed, BulkTransferBatchID: &batchID}
	batch := &domain.BulkTransferBatch{
		ID: batchID, Status: domain.BulkTransferBatchStatusCompleted,
		TotalCount: 3, SuccessCount: 3, TransferAmount: 2_000_000,
		LedgerTxnID: &ledgerTxnID, CreatedBy: 9,
	}
	linkedTxnID := uint(ledgerTxnID)
	linker := &bookingTimesheetLinker{linkedTxnID: &linkedTxnID}
	ledger := &bookingLedgerWriter{}
	adjuster := &bookingTxnAdjuster{txn: &domain.Transaction{
		ID: uint(ledgerTxnID), TransactionType: domain.TransactionTypeRevenue,
		Amount: 2_040_000, Party: "VFIC Manpower",
	}}
	service := &WalletBulkTransferService{
		batchRepo: &bookingBatchRepo{batch: batch},
		paymentRepo: &bookingPaymentReader{rows: []*domaintx.WalletPayment{
			{ID: 1, RequestID: "VFICpaid", RequestedAmount: 1_000_000, Status: domaintx.StateCompleted, BulkTransferBatchID: &batchID},
			reversedA, reversedB,
		}},
		txnAdjuster: adjuster, txRunner: &bookingRunner{}, ledgerWriter: ledger,
		timesheetLinker: linker,
		txnCodeRepo: &bookingCodeRepo{codes: []*domain.TransactionCode{
			transactionCode(t, "VFICreturnA", 500_000, 81),
			transactionCode(t, "VFICreturnB", 500_000, 82),
		}},
		clock: func() time.Time { return time.Date(2026, 7, 19, 18, 30, 0, 0, time.Local) },
	}

	if err := service.HandleReversedPayment(context.Background(), reversedA); err != nil {
		t.Fatal(err)
	}
	if adjuster.txn.Amount != 1_020_000 {
		t.Fatalf("remaining receivable = %d, want 1020000", adjuster.txn.Amount)
	}
	if batch.SuccessCount != 1 || batch.FailedCount != 2 || batch.TransferAmount != 1_000_000 {
		t.Fatalf("batch totals success=%d failed=%d transfer=%d", batch.SuccessCount, batch.FailedCount, batch.TransferAmount)
	}
	if len(ledger.entries) != 8 {
		t.Fatalf("reversal entries = %d, want 8", len(ledger.entries))
	}
	if len(linker.cleared) != 2 || linker.cleared[0] != 81 || linker.cleared[1] != 82 {
		t.Fatalf("cleared timesheets = %v, want [81 82]", linker.cleared)
	}
}

func TestGetBatch_UsesStableSnakeCasePaymentDTO(t *testing.T) {
	service := &WalletBulkTransferService{
		batchRepo: &bookingBatchRepo{batch: &domain.BulkTransferBatch{ID: 4}},
		paymentRepo: &bookingPaymentReader{rows: []*domaintx.WalletPayment{
			{ID: 12, RequestID: "VFICabc", RecipientName: "Nguyen A", Status: domaintx.StateCompleted},
		}},
	}
	detail, err := service.GetBatch(context.Background(), 4)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(detail)
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(payload)
	if !strings.Contains(serialized, `"request_id":"VFICabc"`) || strings.Contains(serialized, `"RequestID"`) {
		t.Fatalf("unexpected detail JSON: %s", serialized)
	}
}

func TestDownloadKQScoped_SuccessfulOnly(t *testing.T) {
	order1, order2 := uint(1), uint(2)
	data, _ := json.Marshal([]BulkTransferRow{
		{OrderNo: 1, Bank: "MB"}, {OrderNo: 2, Bank: "VCB"},
	})
	service := &WalletBulkTransferService{
		batchRepo: &bookingBatchRepo{batch: &domain.BulkTransferBatch{ID: 5, Filename: "batch.xlsx", Data: string(data)}},
		paymentRepo: &bookingPaymentReader{rows: []*domaintx.WalletPayment{
			{ID: 1, RequestID: "VFICok", RecipientAccountNo: "111", BulkTransferOrder: &order1, Status: domaintx.StateCompleted},
			{ID: 2, RequestID: "VFICfail", RecipientAccountNo: "222", BulkTransferOrder: &order2, Status: domaintx.StateFailed},
		}},
		clock: func() time.Time { return time.Date(2026, 7, 19, 12, 0, 0, 0, time.Local) },
	}

	content, filename, err := service.DownloadKQScoped(context.Background(), 5, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(filename, "KQ_Thanh_Cong_") {
		t.Fatalf("filename = %q", filename)
	}
	workbook, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = workbook.Close() }()
	if got, _ := workbook.GetCellValue(kqSheetName, "B5"); got != "111" {
		t.Fatalf("successful account = %q, want 111", got)
	}
	if got, _ := workbook.GetCellValue(kqSheetName, "B6"); got != "" {
		t.Fatalf("failed row leaked into successful workbook: B6=%q", got)
	}
}
