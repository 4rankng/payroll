package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	asynqlib "github.com/hibiken/asynq"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/domain/wallet"
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
	walletSvc            wallet.WalletService
	logger               *slog.Logger
}

func NewDisbursementExecuteWorker(
	walletPaymentService *disbursement.WalletPaymentService,
	registry *disbursement.Registry,
	bankRepo domain.BankRepository,
	walletSvc wallet.WalletService,
	logger *slog.Logger,
) *DisbursementExecuteWorker {
	return &DisbursementExecuteWorker{
		walletPaymentService: walletPaymentService,
		registry:             registry,
		bankRepo:             bankRepo,
		walletSvc:            walletSvc,
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

	// Step 0: Pre-flight balance check (safety net for races between poller and execute)
	if w.walletSvc != nil {
		balance, balErr := w.walletSvc.GetBalance(ctx)
		if balErr != nil {
			logger.Warn("disbursement execute: balance check failed, proceeding", "error", balErr)
		} else if balance.Available < p.RequestedAmount {
			logger.Info("disbursement execute: skipping — insufficient balance",
				"available", balance.Available, "requested", p.RequestedAmount)
			// Do NOT create wallet_payment row. Return nil so asynq doesn't retry.
			// The orphan recovery in the poller will re-enqueue after balance is topped up.
			return nil
		}
	}

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
		if errors.Is(err, disbursement.ErrDuplicatePaymentInProgress) {
			logger.Info("disbursement execute: skipping — duplicate payment already in progress for this recipient",
				"account_no", p.RecipientAccountNo, "bank", p.RecipientBank)
			return nil // terminal — don't retry; admin or another worker is handling it
		}
		if errors.Is(err, disbursement.ErrFeeResolution) {
			// Missing fee schedule in fail-closed mode — a config gap, not a
			// transient fault. Retrying won't help (the schedule/allowlist entry
			// must be added). Surface as a terminal failure so we don't stamp a
			// guessed fee and asynq doesn't pile on retries.
			logger.Error("disbursement execute: fee resolution failed — terminal (config gap), not retrying",
				"error", err, "advance_request_id", p.AdvanceRequestID, "request_id", p.RequestID)
			return fmt.Errorf("initiate wallet_payment: fee resolution: %w", asynqlib.SkipRetry)
		}
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

	// transferAccountName is the holder name sent to the provider on the
	// transfer. It defaults to our stored recipient name; if the account
	// check returns a bank-confirmed name, we prefer that (authoritative) so
	// we don't re-normalize and risk a holder-name mismatch at the provider —
	// OnePay compares the transfer's holder_name against the registered name
	// and rejects any divergence with response_code 15 "Invalid account info".
	transferAccountName := p.RecipientName

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
			FeeWaived:    !checkResult.Valid,
		})
		if err != nil {
			return fmt.Errorf("record account check: %w", err)
		}

		if !checkResult.Valid {
			logger.Info("disbursement execute: account check failed",
				"error_code", checkResult.RawErrorCode)
			return nil // terminal — don't retry
		}

		// Prefer the bank-confirmed holder name for the transfer.
		if checkResult.AccountName != "" {
			transferAccountName = checkResult.AccountName
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
		AccountName: transferAccountName,
		Description: p.RequestID,
		AccountType: infrastructure.AccountTypeBankAccount,
	})
	if err != nil {
		// Pre-flight validation rejection: the transfer endpoint was never
		// called, so no provider fee is charged and retrying won't help (the
		// request is malformed). Record the failure with the fee waived and
		// stop. Any other error is a transport failure → retry via asynq.
		if errors.Is(err, infrastructure.ErrPreflightValidation) {
			_, recordErr := w.walletPaymentService.RecordSyncResponse(ctx, p.RequestID, disbursement.SyncResult{
				Accepted:     false,
				RawErrorCode: "preflight_validation",
				RawMessage:   err.Error(),
				FeeWaived:    true,
			})
			if recordErr != nil {
				return fmt.Errorf("record pre-flight rejection: %w", recordErr)
			}
			logger.Info("disbursement execute: rejected at pre-flight (fee waived)", "error", err)
			return nil // terminal — don't retry a malformed request
		}
		return fmt.Errorf("initiate transfer: %w", err)
	}

	// Step 5: Record sync response (transfer endpoint was called → fee applies)
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
