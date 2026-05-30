package ninepay

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"api-server/internal/domain/ports/infrastructure"
)

// Provider implements infrastructure.DisbursementProvider on top of
// the 9pay HTTP API. Constructed by bootstrap and registered in the
// disbursement.Registry.
//
// The provider has no runtime kill switch of its own: registration
// itself is the gate, controlled by the ENABLE_NINEPAY env var at boot.
// When the flag is off, bootstrap doesn't construct the Provider, so
// it cannot be reached via the registry. This keeps the runtime path
// simple — there's nothing to check on the hot path of every transfer.
type Provider struct {
	client *Client
	logger *slog.Logger
}

// NewProvider wraps a Client into a DisbursementProvider.
func NewProvider(client *Client, logger *slog.Logger) *Provider {
	if logger == nil {
		logger = slog.Default()
	}
	return &Provider{client: client, logger: logger}
}

// Compile-time check: *Provider satisfies the port plus the optional
// account-verification, balance-reporting, error-translation, and
// reconciliation-report-export capabilities.
var (
	_ infrastructure.DisbursementProvider = (*Provider)(nil)
	_ infrastructure.AccountVerifier      = (*Provider)(nil)
	_ infrastructure.BalanceReporter      = (*Provider)(nil)
	_ infrastructure.ErrorTranslator      = (*Provider)(nil)
	_ infrastructure.ReportExporter       = (*Provider)(nil)
	_ infrastructure.StatusPoller         = (*Provider)(nil)
	_ infrastructure.TransferLimiter      = (*Provider)(nil)
)

// NinePay per-transfer amount bounds.
const (
	ninepayMinAmount int64 = 100_000
	ninepayMaxAmount int64 = 20_000_000
)

// Name returns the stable identifier "9pay".
func (p *Provider) Name() string { return ProviderName }

// InitiateTransfer fires a single disbursement. Returns a wrapped HTTP
// error on transport failure, and a populated TransferResult with
// Status=failed when 9pay accepts the call but rejects the transfer
// (e.g. error 1001 / duplicate request_id 702).
func (p *Provider) InitiateTransfer(ctx context.Context, req infrastructure.TransferRequest) (*infrastructure.TransferResult, error) {
	if err := validateTransferRequest(req); err != nil {
		return nil, err
	}

	resp, err := p.client.CreateDisbursement(ctx, createDisbursementRequest{
		RequestID:   req.RequestID,
		Amount:      req.Amount,
		Description: req.Description,
		BankCode:    req.BankCode,
		AccountNo:   req.AccountNo,
		AccountName: req.AccountName,
		AccountType: req.AccountType,
	})
	if err != nil {
		return nil, err
	}

	bankRef := ""
	if resp.BankRef != nil {
		bankRef = *resp.BankRef
	}

	return &infrastructure.TransferResult{
		RequestID:     req.RequestID,
		ProviderRef:   normalizeInvoiceNo(resp.InvoiceNo),
		Status:        translateStatus(resp.Status, resp.ErrorCode, resp.FailureReason, resp.Message),
		BankRef:       bankRef,
		FailureReason: resp.FailureReason,
		RawErrorCode:  resp.ErrorCode,
		RawMessage:    resp.Message,
	}, nil
}

// normalizeInvoiceNo coerces 9pay's invoice_no (wire-named payment_no)
// into either the real provider reference string or the empty string.
// Treat "" and "0" as "no provider ref yet"; downstream syncPatch
// already skips writing InvoiceNo when it is empty, so we'll capture
// the real one when the IPN arrives.
func normalizeInvoiceNo(n json.Number) string {
	s := n.String()
	if s == "" || s == "0" {
		return ""
	}
	return s
}

// CheckStatus returns Status=Unknown for 9pay: the API exposes no
// disbursement-status polling endpoint, and final status is delivered
// asynchronously through the IPN webhook. Callers should rely on
// VerifyAndParseWebhook for state transitions and persist the latest
// known status alongside the request_id.
func (p *Provider) CheckStatus(_ context.Context, requestID string) (*infrastructure.TransferResult, error) {
	return &infrastructure.TransferResult{
		RequestID: requestID,
		Status:    infrastructure.TransferStatusUnknown,
	}, nil
}

// VerifyAndParseWebhook validates the IPN checksum and returns the
// normalized event. See parseAndVerifyWebhook in webhook.go.
func (p *Provider) VerifyAndParseWebhook(_ context.Context, payload map[string]any) (*infrastructure.WebhookEvent, error) {
	return parseAndVerifyWebhook(payload, p.client.secretKeyChecksum())
}

// CheckAccount verifies a beneficiary account through 9pay's
// /disbursement/check-account endpoint. A successful HTTP exchange
// where 9pay rejects the account (non-empty error_code) returns
// Valid=false rather than an error — the rejection is data, not a
// transport failure.
func (p *Provider) CheckAccount(ctx context.Context, req infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
	if err := validateAccountCheckRequest(req); err != nil {
		return nil, err
	}

	resp, err := p.client.CheckAccount(ctx, checkAccountRequest{
		RequestID:   req.RequestID,
		BankCode:    req.BankCode,
		AccountNo:   req.AccountNo,
		AccountName: req.AccountName,
		AccountType: req.AccountType,
	})
	if err != nil {
		return nil, err
	}

	result := &infrastructure.AccountCheckResult{
		Valid:        isSuccessCode(resp.ErrorCode) && resp.Status == 5,
		BankCode:     resp.BankCode,
		AccountNo:    resp.AccountNo,
		AccountName:  resp.AccountName,
		AccountType:  resp.AccountType,
		RawErrorCode: resp.ErrorCode,
		RawMessage:   resp.Message,
	}

	// Name matching: compare stored name against bank-confirmed name.
	// Blocks wrong-recipient transfers, avoiding provider fee on revert.
	if result.Valid && resp.AccountName != "" && req.AccountName != "" {
		if mismatchMsg, ok := infrastructure.MatchAccountName(req.AccountName, resp.AccountName); !ok {
			p.logger.Info("ninepay: name mismatch",
				"request_id", req.RequestID,
				"raw_expected", req.AccountName, "raw_actual", resp.AccountName)
			result.Valid = false
			result.RawErrorCode = "name_mismatch"
			result.RawMessage = mismatchMsg
		}
	}

	return result, nil
}

// GetBalance returns the partner's current 9pay balance via
// /disbursement/balance.
func (p *Provider) GetBalance(ctx context.Context) (*infrastructure.BalanceResult, error) {
	resp, err := p.client.GetBalance(ctx)
	if err != nil {
		return nil, err
	}
	return &infrastructure.BalanceResult{
		Amount:       resp.Data,
		RawErrorCode: resp.ErrorCode,
		RawMessage:   resp.Message,
	}, nil
}

// Limits returns the NinePay per-transfer amount bounds.
func (p *Provider) Limits() infrastructure.TransferLimits {
	return infrastructure.TransferLimits{
		MinAmount: ninepayMinAmount,
		MaxAmount: ninepayMaxAmount,
	}
}

func validateAccountCheckRequest(req infrastructure.AccountCheckRequest) error {
	switch {
	case req.RequestID == "":
		return fmt.Errorf("ninepay: AccountCheckRequest.RequestID is required")
	case len(req.RequestID) > 30:
		return fmt.Errorf("ninepay: AccountCheckRequest.RequestID exceeds 30 chars")
	case req.BankCode == "":
		return fmt.Errorf("ninepay: AccountCheckRequest.BankCode is required")
	case req.AccountNo == "":
		return fmt.Errorf("ninepay: AccountCheckRequest.AccountNo is required")
	case req.AccountType == "":
		return fmt.Errorf("ninepay: AccountCheckRequest.AccountType is required (\"0\" for account, \"1\" for card)")
	}
	return nil
}

func validateTransferRequest(req infrastructure.TransferRequest) error {
	switch {
	case req.RequestID == "":
		return fmt.Errorf("ninepay: TransferRequest.RequestID is required")
	case len(req.RequestID) > 30:
		return fmt.Errorf("ninepay: TransferRequest.RequestID exceeds 30 chars")
	case req.Amount < ninepayMinAmount || req.Amount > ninepayMaxAmount:
		return fmt.Errorf("ninepay: TransferRequest.Amount %d out of allowed range %d–%d VND", req.Amount, ninepayMinAmount, ninepayMaxAmount)
	case req.BankCode == "":
		return fmt.Errorf("ninepay: TransferRequest.BankCode is required")
	case req.AccountNo == "":
		return fmt.Errorf("ninepay: TransferRequest.AccountNo is required")
	case req.AccountName == "":
		return fmt.Errorf("ninepay: TransferRequest.AccountName is required")
	case req.AccountType == "":
		return fmt.Errorf("ninepay: TransferRequest.AccountType is required (\"0\" for account, \"1\" for card)")
	}
	return nil
}
