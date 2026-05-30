package onepay

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"api-server/internal/domain/ports/infrastructure"
)

// Provider implements infrastructure.DisbursementProvider on top of
// the OnePay PayOut API. Constructed by bootstrap and registered in the
// disbursement.Registry.
//
// OnePay wire states and their mapping to abstract TransferStatus:
//
//	Wire state   → TransferStatus   → FSM trigger
//	created      → Pending          → (no trigger)
//	pending      → Pending          → (no trigger)
//	approved     → Success          → ipn_completed
//	failed       → Failed           → ipn_failed
//	reverted     → Reversed         → ipn_failed + flag
type Provider struct {
	client *Client
	logger *slog.Logger
}

// Compile-time port assertions.
var (
	_ infrastructure.DisbursementProvider = (*Provider)(nil)
	_ infrastructure.AccountVerifier      = (*Provider)(nil)
	_ infrastructure.BalanceReporter      = (*Provider)(nil)
	_ infrastructure.ErrorTranslator      = (*Provider)(nil)
	_ infrastructure.StatusPoller         = (*Provider)(nil)
	_ infrastructure.TransferLimiter      = (*Provider)(nil)
)

// NewProvider wraps a Client into a DisbursementProvider.
func NewProvider(client *Client, logger *slog.Logger) *Provider {
	if logger == nil {
		logger = slog.Default()
	}
	return &Provider{client: client, logger: logger}
}

// Name returns the stable identifier "1pay".
func (p *Provider) Name() string { return ProviderName }

// ---------------------------------------------------------------------------
// DisbursementProvider
// ---------------------------------------------------------------------------

// InitiateTransfer fires a single disbursement. Returns a wrapped HTTP
// error on transport failure, and a populated TransferResult with
// Status=failed when OnePay accepts the call but rejects the transfer
// (e.g. error 07 DUPLICATE_TXN, 19 NO_BANK_PROCESS_FOUND).
func (p *Provider) InitiateTransfer(ctx context.Context, req infrastructure.TransferRequest) (*infrastructure.TransferResult, error) {
	if err := validateTransferRequest(req); err != nil {
		return nil, err
	}

	swiftCode := req.SwiftCode
	apiReq := FundsTransferRequest{
		FundsTransferID:   req.RequestID,
		SwiftCode:         swiftCode,
		AccountNumber:     req.AccountNo,
		HolderName:        req.AccountName,
		Amount:            strconv.FormatInt(req.Amount, 10),
		Currency:          "VND",
		FundsTransferInfo: req.RequestID,
		Remark:            req.Description,
	}

	resp, err := p.client.CreateFundsTransfer(ctx, apiReq)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			return &infrastructure.TransferResult{
				RequestID:     req.RequestID,
				Status:        infrastructure.TransferStatusFailed,
				FailureReason: apiErr.Message,
				RawErrorCode:  apiErr.ResponseCode,
				RawMessage:    apiErr.Message,
			}, nil
		}
		return nil, err
	}

	return &infrastructure.TransferResult{
		RequestID:    req.RequestID,
		ProviderRef:  resp.TransactionID,
		Status:       translateState(resp.State, resp.ResponseCode),
		RawErrorCode: resp.ResponseCode,
		RawMessage:   resp.Message,
	}, nil
}

// VerifyAndParseWebhook validates an inbound OnePay IPN callback and
// returns the normalized event. Full parsing is in webhook.go (TASK-031).
func (p *Provider) VerifyAndParseWebhook(ctx context.Context, payload map[string]any) (*infrastructure.WebhookEvent, error) {
	return parseAndVerifyWebhook(payload, p.client.cfg.PartnerID, p.client.cfg.PartnerKey, p.client.now)
}

// ---------------------------------------------------------------------------
// AccountVerifier
// ---------------------------------------------------------------------------

// CheckAccount verifies a beneficiary account through OnePay's
// /onepayout/api/v1/customers endpoint. The flow is:
//
//  1. Local pre-flight validation (no API call, catches garbage early).
//  2. OnePay GET /customers (free lookup — returns account validity +
//     bank-confirmed holder_name).
//  3. Name matching: compare our stored name against the bank-confirmed
//     name (normalized, no diacritics). Blocks transfers to wrong
//     recipients, avoiding the 3,850 VND fee per Điều 6.3.b.
//
// A successful HTTP exchange where OnePay rejects the account returns
// Valid=false rather than an error — the rejection is data, not a
// transport failure.
func (p *Provider) CheckAccount(ctx context.Context, req infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
	// --- Local pre-validation (free, no API call) ---
	if pf := preflightValidate(req.SwiftCode, req.AccountNo, req.AccountName, req.Amount); pf != nil {
		return &infrastructure.AccountCheckResult{
			Valid:        false,
			AccountNo:    req.AccountNo,
			RawErrorCode: pf.Reason,
			RawMessage:   pf.Message,
		}, nil
	}

	// --- API call to OnePay GET /customers ---
	amt := req.Amount
	if amt == 0 {
		amt = onepayMinAmount
	}

	apiReq := AccountInfoRequest{
		RequestID:     req.RequestID,
		SwiftCode:     req.SwiftCode,
		AccountNumber: req.AccountNo,
		AccountID:     p.client.cfg.AccountID,
		Amount:        amt,
	}

	resp, err := p.client.GetAccountInfo(ctx, apiReq)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			return &infrastructure.AccountCheckResult{
				Valid:        false,
				RawErrorCode: apiErr.ResponseCode,
				RawMessage:   apiErr.Message,
			}, nil
		}
		return nil, err
	}

	// --- Name matching (post-API, pre-transfer) ---
	// OnePay returns holder_name without Vietnamese diacritics.
	// Compare normalized forms to catch wrong-recipient accounts.
	if resp.HolderName != "" && req.AccountName != "" {
		if mismatchMsg, ok := infrastructure.MatchAccountName(req.AccountName, resp.HolderName); !ok {
			p.logger.Info("onepay: name mismatch",
				"request_id", req.RequestID,
				"raw_expected", req.AccountName, "raw_actual", resp.HolderName)
			return &infrastructure.AccountCheckResult{
				Valid:        false,
				AccountName:  resp.HolderName,
				AccountNo:    req.AccountNo,
				RawErrorCode: "name_mismatch",
				RawMessage:   mismatchMsg,
			}, nil
		}
	}

	return &infrastructure.AccountCheckResult{
		Valid:        resp.State == "approved",
		AccountName:  resp.HolderName,
		RawErrorCode: resp.ResponseCode,
		RawMessage:   resp.Message,
	}, nil
}

// ---------------------------------------------------------------------------
// BalanceReporter
// ---------------------------------------------------------------------------

// GetBalance returns the partner's current OnePay balance.
func (p *Provider) GetBalance(ctx context.Context) (*infrastructure.BalanceResult, error) {
	resp, err := p.client.GetBalance(ctx)
	if err != nil {
		return nil, err
	}
	bal, _ := resp.Balance.Int64()
	return &infrastructure.BalanceResult{
		Amount:   bal,
		Currency: "VND",
	}, nil
}

// ---------------------------------------------------------------------------
// ErrorTranslator
// ---------------------------------------------------------------------------

// TranslateError returns the Vietnamese translation for a OnePay error code.
func (p *Provider) TranslateError(code string) string { return MessageVI(code) }

// ---------------------------------------------------------------------------
// StatusPoller
// ---------------------------------------------------------------------------

// CheckStatus polls the status of a previously-created transfer via
// OnePay's Inquiry Funds Transfer endpoint.
func (p *Provider) CheckStatus(ctx context.Context, requestID string) (*infrastructure.TransferResult, error) {
	resp, err := p.client.InquiryFundsTransfer(ctx, requestID)
	if err != nil {
		return nil, err
	}
	return &infrastructure.TransferResult{
		RequestID:    resp.FundsTransferID,
		ProviderRef:  resp.TransactionID,
		Status:       translateState(resp.State, resp.ResponseCode),
		BankRef:      resp.BankTxnRef,
		RawErrorCode: resp.ResponseCode,
		RawMessage:   resp.Message,
	}, nil
}

// Limits returns the OnePay per-transfer amount bounds.
func (p *Provider) Limits() infrastructure.TransferLimits {
	return infrastructure.TransferLimits{
		MinAmount: onepayMinAmount,
		MaxAmount: onepayMaxAmount,
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// translateState maps the OnePay wire state to the abstract TransferStatus.
//
// Wire states: created | pending | approved | failed | reverted
func translateState(state, responseCode string) infrastructure.TransferStatus {
	switch state {
	case "approved":
		return infrastructure.TransferStatusSuccess
	case "failed":
		return infrastructure.TransferStatusFailed
	case "reverted":
		return infrastructure.TransferStatusReversed
	case "created", "pending":
		return infrastructure.TransferStatusPending
	default:
		return infrastructure.TransferStatusUnknown
	}
}

// OnePay spec amount limits (shared by preflightValidate and validateTransferRequest).
const (
	onepayMinAmount int64 = 100_000
	onepayMaxAmount int64 = 20_000_000
)

// validateTransferRequest checks request fields before calling OnePay's
// funds transfer endpoint. Amount bounds match the OnePay spec:
// minimum 100,000 VND, maximum 20,000,000 VND.
func validateTransferRequest(req infrastructure.TransferRequest) error {
	if req.RequestID == "" || len(req.RequestID) > 20 {
		return fmt.Errorf("onepay: RequestID must be 1..20 chars")
	}
	if req.Amount < onepayMinAmount {
		return fmt.Errorf("onepay: Amount must be at least %d VND", onepayMinAmount)
	}
	if req.Amount > onepayMaxAmount {
		return fmt.Errorf("onepay: Amount must not exceed %d VND", onepayMaxAmount)
	}
	if req.SwiftCode == "" {
		return fmt.Errorf("onepay: SwiftCode is required")
	}
	if req.AccountNo == "" {
		return fmt.Errorf("onepay: AccountNo is required")
	}
	if req.AccountName == "" {
		return fmt.Errorf("onepay: AccountName is required")
	}
	return nil
}

// preflightResult is the outcome of local pre-validation before any
// API call is made to OnePay.
type preflightResult struct {
	Reason  string // machine key: missing_swift_code, amount_below_min, etc.
	Message string // Vietnamese user message
}

// preflightValidate checks request fields that can be validated locally
// before any API call. Returns nil if all checks pass.
func preflightValidate(swiftCode, accountNo, accountName string, amount int64) *preflightResult {
	if swiftCode == "" {
		return &preflightResult{"missing_swift_code", "Mã ngân hàng (SWIFT) không được để trống"}
	}
	if accountNo == "" {
		return &preflightResult{"missing_account_no", "Số tài khoản không được để trống"}
	}
	if accountName == "" {
		return &preflightResult{"missing_holder_name", "Tên chủ tài khoản không được để trống"}
	}
	if amount > 0 && amount < onepayMinAmount {
		return &preflightResult{"amount_below_min", fmt.Sprintf("Số tiền chuyển tối thiểu %d VND", onepayMinAmount)}
	}
	if amount > onepayMaxAmount {
		return &preflightResult{"amount_above_max", fmt.Sprintf("Số tiền chuyển tối đa %d VND", onepayMaxAmount)}
	}
	return nil
}
