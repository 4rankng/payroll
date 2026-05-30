package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/pkg/utils"
)

// TransferStatus is the normalized lifecycle state of a disbursement,
// independent of any specific provider's vocabulary.
type TransferStatus string

const (
	TransferStatusPending  TransferStatus = "pending"
	TransferStatusSuccess  TransferStatus = "success"
	TransferStatusFailed   TransferStatus = "failed"
	TransferStatusReversed TransferStatus = "reversed"
	TransferStatusUnknown  TransferStatus = "unknown"
)

// AccountType values for TransferRequest.AccountType and
// AccountCheckRequest.AccountType. Standardised across providers.
const (
	AccountTypeBankAccount = "0"
	AccountTypeCard        = "1"
)

// TransferRequest is the provider-agnostic shape of a single outbound
// disbursement to a beneficiary bank account.
type TransferRequest struct {
	// RequestID is the caller's idempotency key. Providers reject duplicate
	// IDs (e.g. 9pay returns error 702). Keep it stable per logical transfer.
	RequestID string
	// Amount is in the smallest unit of the local currency (VND has no
	// minor unit, so this is the integer VND amount).
	Amount int64
	// Description is sent to the beneficiary's bank statement. Some
	// providers restrict it to alphanumeric-only.
	Description string
	BankCode    string
	SwiftCode   string
	AccountNo   string
	AccountName string
	// AccountType distinguishes bank account from card. Use
	// AccountTypeBankAccount / AccountTypeCard constants.
	AccountType string
}

// TransferResult is the normalized response from initiating or polling
// a transfer. ProviderRef is the provider's primary identifier; callers
// should persist it alongside RequestID for reconciliation.
type TransferResult struct {
	RequestID     string
	ProviderRef   string
	Status        TransferStatus
	BankRef       string
	FailureReason string
	RawErrorCode  string
	RawMessage    string
}

// WebhookEvent is the normalized shape of an inbound provider callback
// after signature verification.
type WebhookEvent struct {
	RequestID     string
	ProviderRef   string
	Status        TransferStatus
	BankRef       string
	FailureReason string
	RawErrorCode  string
	Amount        int64
	// Method is the provider's transaction-method label (e.g. 9pay sends
	// "DISBURSEMENT"); useful for callbacks that multiplex types.
	Method string
	// DecodedPayload is the human-readable decoded webhook body. For 9pay
	// this is the base64-decoded JSON (instead of opaque base64 gibberish).
	// Providers that don't encode their payload should leave this nil.
	DecodedPayload []byte
}

// DisbursementProvider abstracts a single outbound payment provider.
// Implementations live under internal/infra/disbursement/<provider>/.
//
// The registry at internal/app/services/disbursement selects the active
// implementation at request time based on Settings.disbursement_provider,
// so use cases call providers via the registry rather than a concrete type.
type DisbursementProvider interface {
	// Name returns the stable provider identifier (e.g. "9pay", "payos").
	// Must match the value stored in Settings.disbursement_provider.
	Name() string

	// InitiateTransfer starts a single disbursement. Providers must treat
	// req.RequestID as an idempotency key and surface duplicates as an
	// error rather than re-disbursing.
	InitiateTransfer(ctx context.Context, req TransferRequest) (*TransferResult, error)

	// VerifyAndParseWebhook validates an inbound provider callback and
	// returns the normalized event. payload is the parsed form/JSON body
	// from the inbound HTTP request. Returns an error if signature
	// verification fails — callers must treat that as a hard reject.
	VerifyAndParseWebhook(ctx context.Context, payload map[string]any) (*WebhookEvent, error)
}

// StatusPoller is an optional capability for providers that expose a
// disbursement-status polling endpoint. Providers without a polling
// endpoint (like 9pay, which relies on IPN webhooks) should NOT
// implement this. Callers should type-assert and degrade gracefully.
//
//	if p, ok := provider.(infrastructure.StatusPoller); ok {
//	    res, err := p.CheckStatus(ctx, requestID)
//	}
type StatusPoller interface {
	CheckStatus(ctx context.Context, requestID string) (*TransferResult, error)
}

// AccountVerifier is an optional capability some disbursement providers
// expose: a pre-flight check that a beneficiary bank account is reachable
// before any money moves. Callers should type-assert and gracefully
// degrade for providers that don't implement it.
//
//	if v, ok := provider.(infrastructure.AccountVerifier); ok {
//	    res, err := v.CheckAccount(ctx, req)
//	    ...
//	}
type AccountVerifier interface {
	CheckAccount(ctx context.Context, req AccountCheckRequest) (*AccountCheckResult, error)
}

// AccountCheckRequest is the input to AccountVerifier.CheckAccount.
// RequestID is a per-call idempotency key (some providers require it).
type AccountCheckRequest struct {
	RequestID   string
	BankCode    string
	SwiftCode   string
	AccountNo   string
	AccountName string
	// Amount is the expected disbursement amount (some providers verify this).
	Amount int64
	// AccountType distinguishes bank account from card. Use
	// AccountTypeBankAccount / AccountTypeCard constants.
	AccountType string
}

// AccountCheckResult is the normalized response from an account-check
// call. Valid is true only when the provider confirmed the account
// exists and matches. AccountName is the bank-confirmed holder name
// (some providers return it; others leave it blank).
type AccountCheckResult struct {
	Valid        bool
	BankCode     string
	AccountNo    string
	AccountName  string
	AccountType  string
	RawErrorCode string
	RawMessage   string
}

// BalanceReporter is an optional capability for providers that expose a
// partner-balance endpoint. Like AccountVerifier, callers should type-
// assert and degrade gracefully when the provider doesn't implement it.
type BalanceReporter interface {
	GetBalance(ctx context.Context) (*BalanceResult, error)
}

// BalanceResult is the normalized response from BalanceReporter.GetBalance.
// Amount is in the smallest unit of the local currency (VND has no minor
// unit, so this is integer VND).
type BalanceResult struct {
	Amount       int64
	Currency     string
	RawErrorCode string
	RawMessage   string
}

// ReportExporter is an optional capability for providers that expose
// a reconciliation report — a CSV (or other tabular format) of
// disbursements within a date range, used by ops for off-ledger
// reconciliation. Callers should type-assert and degrade gracefully
// when the active provider doesn't implement it.
//
//	if e, ok := provider.(infrastructure.ReportExporter); ok {
//	    csvBytes, fileName, err := e.ExportReconciliation(ctx, from, to)
//	    ...
//	}
type ReportExporter interface {
	// ExportReconciliation fetches the reconciliation report for
	// [dateFrom, dateTo] inclusive. Returns the report bytes and the
	// provider-issued file name (without extension) — callers stream
	// the bytes to the HTTP response and use the file name in the
	// Content-Disposition header.
	ExportReconciliation(ctx context.Context, dateFrom, dateTo time.Time) (csv []byte, fileName string, err error)
}

// ErrorTranslator is an optional capability for providers that maintain
// a code → user-facing message table. Service-level callers translate
// raw provider error codes into Vietnamese strings without importing
// the concrete provider package:
//
//	if t, ok := provider.(infrastructure.ErrorTranslator); ok {
//	    msg = t.TranslateError(code)
//	}
//
// Implementations should return "" for the empty code so callers can
// short-circuit on success, and a generic fallback (e.g. "unknown
// error (code: X)") for codes outside their known set.
type ErrorTranslator interface {
	// TranslateError returns the user-facing Vietnamese message for a
	// raw provider error code. Empty input must yield "".
	TranslateError(code string) string
}

// TransferLimits describes the provider's per-transfer amount bounds.
type TransferLimits struct {
	// MinAmount is the minimum single-transfer amount in VND. Zero means no
	// provider-enforced minimum.
	MinAmount int64
	// MaxAmount is the maximum single-transfer amount in VND. Zero means no
	// provider-enforced maximum.
	MaxAmount int64
}

// TransferLimiter is an optional capability for providers that enforce
// per-transfer amount bounds. Callers should type-assert and degrade
// gracefully when the provider doesn't implement it:
//
//	if l, ok := provider.(infrastructure.TransferLimiter); ok {
//	    limits := l.Limits()
//	}
type TransferLimiter interface {
	Limits() TransferLimits
}

// ---------------------------------------------------------------------------
// Shared account-check helpers (used by all providers)
// ---------------------------------------------------------------------------

// NormalizeNameForComparison prepares a Vietnamese name for equality comparison:
// strip diacritics → uppercase → collapse whitespace.
// Both OnePay and 9Pay return holder names without Vietnamese diacritics
// (e.g. "NGUYEN VAN A"), so we normalize both sides the same way.
func NormalizeNameForComparison(name string) string {
	s := utils.NormalizeVietnamese(name) // strips diacritics, lowercases
	s = strings.ToUpper(s)
	return strings.Join(strings.Fields(s), " ")
}

// MatchAccountName compares a stored name against a bank-confirmed name,
// both normalized to uppercase no-diacritics. Returns ("", nil) on match,
// or a descriptive mismatch message on failure.
func MatchAccountName(storedName, bankConfirmedName string) (mismatchMsg string, ok bool) {
	if bankConfirmedName == "" || storedName == "" {
		return "", true // skip when either side is empty (best-effort)
	}
	expected := NormalizeNameForComparison(storedName)
	actual := NormalizeNameForComparison(bankConfirmedName)
	if expected == actual {
		return "", true
	}
	return fmt.Sprintf("Tên chủ TK không khớp: hệ thống ghi \"%s\", ngân hàng xác nhận \"%s\"", storedName, bankConfirmedName), false
}
