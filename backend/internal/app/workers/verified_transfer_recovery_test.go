package workers

import (
	"context"
	"errors"
	"testing"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
)

type verifiedRecoveryProvider struct {
	result          *infrastructure.TransferResult
	err             error
	calls           int
	initiateResult  *infrastructure.TransferResult
	initiateErr     error
	initiateCalls   int
	initiateRequest infrastructure.TransferRequest
}

func (p *verifiedRecoveryProvider) Name() string { return "1pay" }

func (p *verifiedRecoveryProvider) InitiateTransfer(_ context.Context, request infrastructure.TransferRequest) (*infrastructure.TransferResult, error) {
	p.initiateCalls++
	p.initiateRequest = request
	return p.initiateResult, p.initiateErr
}

func (p *verifiedRecoveryProvider) VerifyAndParseWebhook(context.Context, map[string]any) (*infrastructure.WebhookEvent, error) {
	return nil, nil
}

func (p *verifiedRecoveryProvider) CheckStatus(context.Context, string) (*infrastructure.TransferResult, error) {
	p.calls++
	return p.result, p.err
}

type verifiedRecoveryRecorder struct {
	calls int
	got   disbursement.SyncResult
}

func (r *verifiedRecoveryRecorder) RecordSyncResponse(_ context.Context, _ string, result disbursement.SyncResult) (*domaintx.WalletPayment, error) {
	r.calls++
	r.got = result
	return &domaintx.WalletPayment{Status: domaintx.StateAuthorised}, nil
}

func TestRecoverVerifiedTransfer_NotFoundAllowsRetryOfSameTransferID(t *testing.T) {
	provider := &verifiedRecoveryProvider{err: infrastructure.ErrTransferNotFound}
	recorder := &verifiedRecoveryRecorder{}
	payment := &domaintx.WalletPayment{RequestID: "FT-001", Status: domaintx.StateVerified}

	result, notFound, err := recoverVerifiedTransfer(context.Background(), provider, recorder, payment)
	if err != nil {
		t.Fatalf("recoverVerifiedTransfer: %v", err)
	}
	if !notFound || result != nil {
		t.Fatalf("result=%+v notFound=%v, want explicit not-found", result, notFound)
	}
	if provider.calls != 1 || recorder.calls != 0 {
		t.Fatalf("provider calls=%d recorder calls=%d", provider.calls, recorder.calls)
	}
}

func TestRecoverVerifiedTransfer_ExistingTransferPromotesToAuthorised(t *testing.T) {
	providerResult := &infrastructure.TransferResult{
		RequestID:    "FT-001",
		ProviderRef:  "OP-001",
		Status:       infrastructure.TransferStatusPending,
		RawErrorCode: "01",
		RawMessage:   "Txn is pending",
	}
	provider := &verifiedRecoveryProvider{result: providerResult}
	recorder := &verifiedRecoveryRecorder{}
	payment := &domaintx.WalletPayment{RequestID: "FT-001", Status: domaintx.StateVerified}

	result, notFound, err := recoverVerifiedTransfer(context.Background(), provider, recorder, payment)
	if err != nil {
		t.Fatalf("recoverVerifiedTransfer: %v", err)
	}
	if notFound || result != providerResult {
		t.Fatalf("result=%+v notFound=%v", result, notFound)
	}
	if recorder.calls != 1 || !recorder.got.Accepted || recorder.got.InvoiceNo != "OP-001" {
		t.Fatalf("recorded sync result = %+v, calls=%d", recorder.got, recorder.calls)
	}
}

func TestRecoverVerifiedTransfer_InquiryFailureDoesNotRetryTransfer(t *testing.T) {
	provider := &verifiedRecoveryProvider{err: errors.New("provider unavailable")}
	recorder := &verifiedRecoveryRecorder{}
	payment := &domaintx.WalletPayment{RequestID: "FT-001", Status: domaintx.StateVerified}

	result, notFound, err := recoverVerifiedTransfer(context.Background(), provider, recorder, payment)
	if err == nil || result != nil || notFound {
		t.Fatalf("result=%+v notFound=%v err=%v", result, notFound, err)
	}
	if recorder.calls != 0 {
		t.Fatalf("recorder calls=%d, want 0", recorder.calls)
	}
}
