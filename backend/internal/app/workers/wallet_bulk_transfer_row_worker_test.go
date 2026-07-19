package workers

import (
	"context"
	"encoding/json"
	"testing"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/app/services/wallet_bulk"
	"api-server/internal/domain"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"

	"github.com/hibiken/asynq"
)

type rejectedRowPaymentService struct {
	syncResult     disbursement.SyncResult
	accountOutcome disbursement.AccountCheckOutcome
}

func (s *rejectedRowPaymentService) Initiate(context.Context, disbursement.InitiateInput) (*domaintx.WalletPayment, error) {
	return &domaintx.WalletPayment{ID: 1, Status: domaintx.StatePending}, nil
}
func (s *rejectedRowPaymentService) RecordAccountCheck(_ context.Context, _ string, outcome disbursement.AccountCheckOutcome) (*domaintx.WalletPayment, error) {
	s.accountOutcome = outcome
	status := domaintx.StateVerified
	if !outcome.Verified {
		status = domaintx.StateFailed
	}
	return &domaintx.WalletPayment{ID: 1, Status: status}, nil
}
func (s *rejectedRowPaymentService) RecordSyncResponse(_ context.Context, _ string, result disbursement.SyncResult) (*domaintx.WalletPayment, error) {
	s.syncResult = result
	return &domaintx.WalletPayment{ID: 1, Status: domaintx.StateFailed}, nil
}

type rejectedProvider struct{}

func (rejectedProvider) Name() string { return "onepay" }
func (rejectedProvider) InitiateTransfer(context.Context, infrastructure.TransferRequest) (*infrastructure.TransferResult, error) {
	return &infrastructure.TransferResult{Status: infrastructure.TransferStatusFailed, ProviderRef: "OP-FAIL", RawErrorCode: "51"}, nil
}
func (rejectedProvider) VerifyAndParseWebhook(context.Context, map[string]any) (*infrastructure.WebhookEvent, error) {
	return nil, nil
}

type rejectedRegistry struct{}

func (rejectedRegistry) Active(context.Context) (infrastructure.DisbursementProvider, error) {
	return rejectedProvider{}, nil
}

type invalidAccountProvider struct{ transferCalls int }

func (*invalidAccountProvider) Name() string { return "onepay" }
func (p *invalidAccountProvider) InitiateTransfer(context.Context, infrastructure.TransferRequest) (*infrastructure.TransferResult, error) {
	p.transferCalls++
	return &infrastructure.TransferResult{Status: infrastructure.TransferStatusPending}, nil
}
func (*invalidAccountProvider) VerifyAndParseWebhook(context.Context, map[string]any) (*infrastructure.WebhookEvent, error) {
	return nil, nil
}
func (*invalidAccountProvider) CheckAccount(context.Context, infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
	return &infrastructure.AccountCheckResult{Valid: false, RawErrorCode: "14", RawMessage: "invalid account"}, nil
}

type fixedRegistry struct {
	provider infrastructure.DisbursementProvider
}

func (r fixedRegistry) Active(context.Context) (infrastructure.DisbursementProvider, error) {
	return r.provider, nil
}

type rejectedPaymentRepo struct {
	domaintx.WalletPaymentRepository
}

func (rejectedPaymentRepo) UpdateBulkBatchLink(context.Context, uint64, uint64, uint, string) error {
	return nil
}
func (rejectedPaymentRepo) CountByBatchAndStatuses(_ context.Context, _ uint64, statuses []domaintx.State) (int64, error) {
	for _, status := range statuses {
		if status == domaintx.StateFailed {
			return 1, nil
		}
	}
	return 0, nil
}

type rejectedBatchRepo struct {
	domain.BulkTransferBatchRepository
	batch *domain.BulkTransferBatch
}

func (r *rejectedBatchRepo) UpdateWithLock(_ context.Context, _ uint64, fn func(*domain.BulkTransferBatch) (bool, error)) (*domain.BulkTransferBatch, bool, error) {
	book, err := fn(r.batch)
	return r.batch, book, err
}

type rejectedEnqueuer struct{ booked int }

func (*rejectedEnqueuer) EnqueueBulkTransferRow(wallet_bulk.RowTaskPayload) error { return nil }
func (e *rejectedEnqueuer) EnqueueBookBatchLedger(wallet_bulk.BookLedgerPayload) error {
	e.booked++
	return nil
}

func TestWalletBulkTransferRowWorker_SynchronousProviderRejectionFinalizesRow(t *testing.T) {
	paymentService := &rejectedRowPaymentService{}
	batchRepo := &rejectedBatchRepo{batch: &domain.BulkTransferBatch{
		ID: 44, Status: domain.BulkTransferBatchStatusProcessing, TotalCount: 1,
	}}
	enqueuer := &rejectedEnqueuer{}
	worker := NewWalletBulkTransferRowWorker(
		paymentService, nil, rejectedRegistry{}, rejectedPaymentRepo{}, batchRepo, enqueuer, nil,
	)
	payload, err := json.Marshal(wallet_bulk.RowTaskPayload{BatchID: 44, Row: wallet_bulk.BulkTransferRow{
		OrderNo: 1, VFICCode: "VFICabc", Amount: 500_000, SwiftCode: "MSCBVNVX",
		AccountNo: "0123456789", AccountName: "NGUYEN VAN A",
	}})
	if err != nil {
		t.Fatal(err)
	}

	if err := worker.ProcessJob(context.Background(), asynq.NewTask(wallet_bulk.TaskBulkTransferRow, payload)); err != nil {
		t.Fatal(err)
	}
	if paymentService.syncResult.Accepted {
		t.Fatal("synchronous provider rejection was recorded as accepted")
	}
	if paymentService.syncResult.FeeWaived {
		t.Fatal("provider fee was waived even though transfer endpoint was called")
	}
	if batchRepo.batch.SuccessCount != 0 || batchRepo.batch.FailedCount != 1 ||
		batchRepo.batch.Status != domain.BulkTransferBatchStatusCompleting {
		t.Fatalf("batch after rejection: success=%d failed=%d status=%s",
			batchRepo.batch.SuccessCount, batchRepo.batch.FailedCount, batchRepo.batch.Status)
	}
	if enqueuer.booked != 1 {
		t.Fatalf("ledger booking tasks = %d, want 1", enqueuer.booked)
	}
}

func TestWalletBulkTransferRowWorker_InvalidAccountWaivesFeeAndSkipsTransfer(t *testing.T) {
	paymentService := &rejectedRowPaymentService{}
	provider := &invalidAccountProvider{}
	batchRepo := &rejectedBatchRepo{batch: &domain.BulkTransferBatch{
		ID: 45, Status: domain.BulkTransferBatchStatusProcessing, TotalCount: 1,
	}}
	enqueuer := &rejectedEnqueuer{}
	worker := NewWalletBulkTransferRowWorker(
		paymentService, nil, fixedRegistry{provider: provider}, rejectedPaymentRepo{}, batchRepo, enqueuer, nil,
	)
	payload, err := json.Marshal(wallet_bulk.RowTaskPayload{BatchID: 45, Row: wallet_bulk.BulkTransferRow{
		OrderNo: 1, VFICCode: "VFICbad", Amount: 500_000, SwiftCode: "MSCBVNVX",
		AccountNo: "0000000000", AccountName: "NGUYEN VAN B",
	}})
	if err != nil {
		t.Fatal(err)
	}

	if err := worker.ProcessJob(context.Background(), asynq.NewTask(wallet_bulk.TaskBulkTransferRow, payload)); err != nil {
		t.Fatal(err)
	}
	if paymentService.accountOutcome.Verified || !paymentService.accountOutcome.FeeWaived {
		t.Fatalf("account outcome = %+v, want rejected with fee waived", paymentService.accountOutcome)
	}
	if provider.transferCalls != 0 {
		t.Fatalf("transfer endpoint calls = %d, want 0", provider.transferCalls)
	}
	if batchRepo.batch.FailedCount != 1 || batchRepo.batch.Status != domain.BulkTransferBatchStatusCompleting {
		t.Fatalf("batch after account rejection: failed=%d status=%s", batchRepo.batch.FailedCount, batchRepo.batch.Status)
	}
	if enqueuer.booked != 1 {
		t.Fatalf("ledger booking tasks = %d, want 1", enqueuer.booked)
	}
}
