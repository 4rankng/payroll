package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	asynqlib "github.com/hibiken/asynq"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
)

// DisbursementExecutePayload is the wire format for disbursement:execute tasks.
type DisbursementExecutePayload struct {
	RequestID          string `json:"request_id"`
	AdvanceRequestID   uint64 `json:"advance_request_id"`
	RequestedAmount    int64  `json:"requested_amount"`
	RecipientName      string `json:"recipient_name"`
	RecipientAccountNo string `json:"recipient_account_no"`
	RecipientBank      string `json:"recipient_bank"`
}

// DisbursementExecuteWorker processes individual disbursement tasks enqueued
// by the poller. It makes the HTTP calls to 9Pay (check-account + create-transfer)
// and drives the wallet_payment FSM through each transition.
type DisbursementExecuteWorker struct {
	walletPaymentService *disbursement.WalletPaymentService
	registry             *disbursement.Registry
	bankRepo             domain.BankRepository
	logger               *slog.Logger
}

func NewDisbursementExecuteWorker(
	walletPaymentService *disbursement.WalletPaymentService,
	registry *disbursement.Registry,
	bankRepo domain.BankRepository,
	logger *slog.Logger,
) *DisbursementExecuteWorker {
	return &DisbursementExecuteWorker{
		walletPaymentService: walletPaymentService,
		registry:             registry,
		bankRepo:             bankRepo,
		logger:               logger,
	}
}

// resolveSwiftCode looks up the SwiftCode for a bank via the bank repository.
// Returns a non-nil error when the repo itself fails (DB timeout, connection
// error) so that the caller can propagate a retryable error to asynq.
// Returns ("", nil) when the bank exists but has no SwiftCode, or when
// bankRepo is nil (graceful degradation).
func (w *DisbursementExecuteWorker) resolveSwiftCode(ctx context.Context, bankCode string) (string, error) {
	if w.bankRepo == nil {
		return "", nil
	}
	bank, err := w.bankRepo.FindByBankCode(ctx, bankCode)
	if err != nil {
		return "", fmt.Errorf("resolve swift code for bank %q: %w", bankCode, err)
	}
	return bank.SwiftCode, nil
}

// ProcessJob handles a single disbursement:execute task.
func (w *DisbursementExecuteWorker) ProcessJob(ctx context.Context, t *asynqlib.Task) error {
	var p DisbursementExecutePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		w.logger.Error("disbursement execute: malformed payload", "error", err)
		return fmt.Errorf("unmarshal payload: %w", asynqlib.SkipRetry)
	}

	logger := w.logger.With(
		"advance_request_id", p.AdvanceRequestID,
		"request_id", p.RequestID,
	)

	// Step 1: Initiate wallet_payment (idempotent — returns existing row if retry)
	row, err := w.walletPaymentService.Initiate(ctx, disbursement.InitiateInput{
		RequestID:          p.RequestID,
		RequestedAmount:    p.RequestedAmount,
		RecipientName:      p.RecipientName,
		RecipientAccountNo: p.RecipientAccountNo,
		RecipientBank:      p.RecipientBank,
		EntityID:           &p.AdvanceRequestID,
	})
	if err != nil {
		return fmt.Errorf("initiate wallet_payment: %w", err)
	}

	// If the row is past pending, another worker already processed it
	if row.Status != domaintx.StatePending {
		logger.Info("disbursement execute: skipping — already processed",
			"status", row.Status)
		return nil
	}

	// Step 2: Get active provider
	provider, err := w.registry.Active(ctx)
	if err != nil {
		return fmt.Errorf("no active provider: %w", err)
	}

	// Step 3: Check account (if supported)
	if verifier, ok := provider.(infrastructure.AccountVerifier); ok {
		swiftCode, err := w.resolveSwiftCode(ctx, p.RecipientBank)
		if err != nil {
			return fmt.Errorf("check account: %w", err)
		}
		checkResult, err := verifier.CheckAccount(ctx, infrastructure.AccountCheckRequest{
			RequestID:   p.RequestID,
			BankCode:    p.RecipientBank,
			SwiftCode:   swiftCode,
			AccountNo:   p.RecipientAccountNo,
			AccountName: p.RecipientName,
			AccountType: infrastructure.AccountTypeBankAccount,
			Amount:      p.RequestedAmount,
		})
		if err != nil {
			return fmt.Errorf("check account: %w", err)
		}

		_, err = w.walletPaymentService.RecordAccountCheck(ctx, p.RequestID, disbursement.AccountCheckOutcome{
			Verified:     checkResult.Valid,
			RawErrorCode: checkResult.RawErrorCode,
			RawMessage:   checkResult.RawMessage,
		})
		if err != nil {
			return fmt.Errorf("record account check: %w", err)
		}

		if !checkResult.Valid {
			logger.Info("disbursement execute: account check failed",
				"error_code", checkResult.RawErrorCode)
			return nil // terminal — don't retry
		}
	}

	// Step 4: Initiate transfer
	swiftCode, err := w.resolveSwiftCode(ctx, p.RecipientBank)
	if err != nil {
		return fmt.Errorf("initiate transfer: %w", err)
	}
	result, err := provider.InitiateTransfer(ctx, infrastructure.TransferRequest{
		RequestID:   p.RequestID,
		Amount:      p.RequestedAmount,
		BankCode:    p.RecipientBank,
		SwiftCode:   swiftCode,
		AccountNo:   p.RecipientAccountNo,
		AccountName: p.RecipientName,
		Description: fmt.Sprintf("Advance payment #%d", p.AdvanceRequestID),
		AccountType: infrastructure.AccountTypeBankAccount,
	})
	if err != nil {
		return fmt.Errorf("initiate transfer: %w", err)
	}

	// Step 5: Record sync response
	_, err = w.walletPaymentService.RecordSyncResponse(ctx, p.RequestID, disbursement.SyncResult{
		Accepted:     result.Status == infrastructure.TransferStatusPending || result.Status == infrastructure.TransferStatusSuccess,
		InvoiceNo:    result.ProviderRef,
		RawErrorCode: result.RawErrorCode,
		RawMessage:   result.RawMessage,
	})
	if err != nil {
		return fmt.Errorf("record sync response: %w", err)
	}

	logger.Info("disbursement execute: transfer initiated",
		"provider_ref", result.ProviderRef, "status", string(result.Status))

	return nil
}
