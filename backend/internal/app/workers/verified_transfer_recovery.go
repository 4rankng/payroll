package workers

import (
	"context"
	"errors"
	"fmt"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
)

type syncResponseRecorder interface {
	RecordSyncResponse(ctx context.Context, requestID string, result disbursement.SyncResult) (*domaintx.WalletPayment, error)
}

type bankCodeResolver interface {
	FindByBankCode(ctx context.Context, bankCode string) (*domain.Bank, error)
}

// recoverVerifiedTransfer queries the provider before an async retry sends the
// same logical transfer again. A verified row means account validation
// completed, but the transfer response was never persisted. Only an explicit
// provider "not found" result makes a retry safe; an existing transfer is
// promoted to authorised so the normal IPN/inquiry path owns final resolution.
func recoverVerifiedTransfer(
	ctx context.Context,
	provider infrastructure.DisbursementProvider,
	recorder syncResponseRecorder,
	payment *domaintx.WalletPayment,
) (*infrastructure.TransferResult, bool, error) {
	poller, ok := provider.(infrastructure.StatusPoller)
	if !ok {
		return nil, false, errors.New("disbursement: verified transfer cannot be recovered because provider has no status inquiry")
	}
	result, err := poller.CheckStatus(ctx, payment.RequestID)
	if errors.Is(err, infrastructure.ErrTransferNotFound) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("query verified transfer %s: %w", payment.RequestID, err)
	}
	if result == nil {
		return nil, false, fmt.Errorf("query verified transfer %s: provider returned no result", payment.RequestID)
	}

	_, err = recorder.RecordSyncResponse(ctx, payment.RequestID, disbursement.SyncResult{
		Accepted:     true,
		InvoiceNo:    result.ProviderRef,
		RawErrorCode: result.RawErrorCode,
		RawMessage:   result.RawMessage,
	})
	if err != nil {
		return nil, false, fmt.Errorf("record recovered transfer acceptance: %w", err)
	}
	return result, false, nil
}

// retryVerifiedTransfer resends the original persisted transfer only after a
// status inquiry explicitly confirmed that OnePay has no record of its
// request ID. The same request ID and bank-confirmed recipient name are used.
func retryVerifiedTransfer(
	ctx context.Context,
	provider infrastructure.DisbursementProvider,
	recorder syncResponseRecorder,
	bankRepo bankCodeResolver,
	payment *domaintx.WalletPayment,
) (*infrastructure.TransferResult, *domaintx.WalletPayment, error) {
	swiftCode := payment.RecipientBank
	if !isSwiftCode(payment.RecipientBank) {
		if bankRepo == nil {
			return nil, nil, fmt.Errorf("resolve bank %s for verified transfer retry: bank repository unavailable", payment.RecipientBank)
		}
		bank, err := bankRepo.FindByBankCode(ctx, payment.RecipientBank)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve bank %s for verified transfer retry: %w", payment.RecipientBank, err)
		}
		if bank == nil || !isSwiftCode(bank.SwiftCode) {
			return nil, nil, fmt.Errorf("resolve bank %s for verified transfer retry: valid SWIFT code unavailable", payment.RecipientBank)
		}
		swiftCode = bank.SwiftCode
	}

	description := payment.RequestID
	if payment.Description != nil && *payment.Description != "" {
		description = *payment.Description
	}
	result, err := provider.InitiateTransfer(ctx, infrastructure.TransferRequest{
		RequestID:   payment.RequestID,
		Amount:      payment.RequestedAmount,
		Description: description,
		BankCode:    payment.RecipientBank,
		SwiftCode:   swiftCode,
		AccountNo:   payment.RecipientAccountNo,
		AccountName: payment.RecipientName,
		AccountType: infrastructure.AccountTypeBankAccount,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("retry verified transfer %s: %w", payment.RequestID, err)
	}
	if result == nil {
		return nil, nil, fmt.Errorf("retry verified transfer %s: provider returned no result", payment.RequestID)
	}
	recorded, err := recorder.RecordSyncResponse(ctx, payment.RequestID, disbursement.SyncResult{
		Accepted:     result.Status == infrastructure.TransferStatusPending || result.Status == infrastructure.TransferStatusSuccess,
		InvoiceNo:    result.ProviderRef,
		RawErrorCode: result.RawErrorCode,
		RawMessage:   result.RawMessage,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("record retried transfer response: %w", err)
	}
	return result, recorded, nil
}

func isSwiftCode(value string) bool {
	if len(value) < 8 || len(value) > 11 {
		return false
	}
	for _, character := range value {
		if (character < 'A' || character > 'Z') && (character < '0' || character > '9') {
			return false
		}
	}
	return true
}
