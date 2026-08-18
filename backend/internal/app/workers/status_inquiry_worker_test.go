package workers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	disbursementservice "api-server/internal/app/services/disbursement"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
)

type statusInquiryRepo struct {
	payments               []*domaintx.WalletPayment
	finalizationCandidates []*domaintx.WalletPayment
}

func (r *statusInquiryRepo) ListStaleAuthorised(context.Context, string, time.Time, int) ([]*domaintx.WalletPayment, error) {
	return r.payments, nil
}

func (r *statusInquiryRepo) ListStaleBulkFinalizationCandidates(context.Context, string, time.Time, int) ([]*domaintx.WalletPayment, error) {
	return r.finalizationCandidates, nil
}

type statusInquiryRecorder struct {
	order  []string
	source string
}

type statusInquiryFinalizer struct {
	calls int
	err   error
}

func (f *statusInquiryFinalizer) FinalizeBulkBatchForIPN(context.Context, *domaintx.WalletPayment) error {
	f.calls++
	return f.err
}

func TestStatusInquiryPoller_NotFoundVerifiedTransferRetriesSamePersistedRequest(t *testing.T) {
	description := "LUONG THANG 8"
	provider := &verifiedRecoveryProvider{
		err: infrastructure.ErrTransferNotFound,
		initiateResult: &infrastructure.TransferResult{
			RequestID: "FT-RETRY-001",
			Status:    infrastructure.TransferStatusPending,
		},
	}
	recorder := &statusInquiryRecorder{}
	worker := NewStatusInquiryPollerWorker(
		&statusInquiryRepo{payments: []*domaintx.WalletPayment{{
			ID:                 2,
			RequestID:          "FT-RETRY-001",
			Provider:           "1pay",
			Status:             domaintx.StateVerified,
			RequestedAmount:    250_000,
			RecipientName:      "NGUYEN VAN BANK",
			RecipientAccountNo: "0123456789",
			RecipientBank:      "VCBVVNVX",
			Description:        &description,
		}}},
		recorder,
		disbursementservice.NewRegistry(provider),
		nil,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	if err := worker.ProcessJob(context.Background()); err != nil {
		t.Fatalf("ProcessJob: %v", err)
	}
	if provider.initiateCalls != 1 {
		t.Fatalf("initiate calls = %d, want 1", provider.initiateCalls)
	}
	if provider.initiateRequest.RequestID != "FT-RETRY-001" || provider.initiateRequest.AccountName != "NGUYEN VAN BANK" {
		t.Fatalf("retried request = %+v, want same ID and persisted bank-confirmed name", provider.initiateRequest)
	}
	if len(recorder.order) != 1 || recorder.order[0] != "sync:FT-RETRY-001" {
		t.Fatalf("record order = %v, want sync response for retry", recorder.order)
	}
}

func TestStatusInquiryPoller_RetriesFailedBulkFinalizationFromDurableCandidates(t *testing.T) {
	batchID := uint64(77)
	candidate := &domaintx.WalletPayment{
		ID:                  3,
		RequestID:           "FT-DONE-001",
		Provider:            "1pay",
		Status:              domaintx.StateCompleted,
		BulkTransferBatchID: &batchID,
	}
	finalizer := &statusInquiryFinalizer{err: errors.New("temporary database failure")}
	worker := NewStatusInquiryPollerWorker(
		&statusInquiryRepo{finalizationCandidates: []*domaintx.WalletPayment{candidate}},
		&statusInquiryRecorder{},
		disbursementservice.NewRegistry(&verifiedRecoveryProvider{}),
		nil,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	).WithBulkBatchFinalizer(finalizer)

	if err := worker.ProcessJob(context.Background()); err != nil {
		t.Fatalf("first ProcessJob: %v", err)
	}
	finalizer.err = nil
	if err := worker.ProcessJob(context.Background()); err != nil {
		t.Fatalf("second ProcessJob: %v", err)
	}
	if finalizer.calls != 2 {
		t.Fatalf("finalizer calls = %d, want retry on next poller run", finalizer.calls)
	}
}

func (r *statusInquiryRecorder) RecordSyncResponse(_ context.Context, requestID string, result disbursementservice.SyncResult) (*domaintx.WalletPayment, error) {
	r.order = append(r.order, "sync:"+requestID)
	return &domaintx.WalletPayment{RequestID: requestID, InvoiceNo: &result.InvoiceNo, Status: domaintx.StateAuthorised}, nil
}

func (r *statusInquiryRecorder) RecordIPN(_ context.Context, result disbursementservice.IPNResult) (*domaintx.WalletPayment, error) {
	r.order = append(r.order, "ipn:"+result.RequestID)
	r.source = result.Source
	batchID := uint64(9)
	return &domaintx.WalletPayment{RequestID: result.RequestID, Status: domaintx.StateCompleted, BulkTransferBatchID: &batchID}, nil
}

func TestStatusInquiryPoller_VerifiedTransferIsPromotedBeforeTerminalResult(t *testing.T) {
	provider := &verifiedRecoveryProvider{result: &infrastructure.TransferResult{
		RequestID:    "FT-001",
		ProviderRef:  "OP-001",
		Status:       infrastructure.TransferStatusSuccess,
		RawErrorCode: "00",
		RawMessage:   "Success",
	}}
	recorder := &statusInquiryRecorder{}
	finalizer := &statusInquiryFinalizer{}
	worker := NewStatusInquiryPollerWorker(
		&statusInquiryRepo{payments: []*domaintx.WalletPayment{{
			ID:        1,
			RequestID: "FT-001",
			Provider:  "1pay",
			Status:    domaintx.StateVerified,
		}}},
		recorder,
		disbursementservice.NewRegistry(provider),
		nil,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	).WithBulkBatchFinalizer(finalizer)

	if err := worker.ProcessJob(context.Background()); err != nil {
		t.Fatalf("ProcessJob: %v", err)
	}
	if len(recorder.order) != 2 || recorder.order[0] != "sync:FT-001" || recorder.order[1] != "ipn:FT-001" {
		t.Fatalf("transition order = %v, want sync then ipn", recorder.order)
	}
	if recorder.source != domaintx.ResolutionSourceStatusInquiry {
		t.Fatalf("resolution source = %q", recorder.source)
	}
	if finalizer.calls != 1 {
		t.Fatalf("bulk finalizer calls = %d, want 1", finalizer.calls)
	}
}
