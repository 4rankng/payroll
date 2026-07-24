package onepay

import (
	"context"
	"log/slog"
	"sync"

	"api-server/internal/domain/ports/infrastructure"

	"golang.org/x/time/rate"
)

// onepayTPS is the hardcoded, sustained rate (calls per second) at which
// every OnePay endpoint is called. We deliberately do NOT read this from
// the ONEPAY_TPS env var: OnePay's per-endpoint rate limit is a
// contractual constant, not a tunable, and silently drifting above it
// during a bulk import risks OnePay throttling (429) or a temporary ban.
// To change it, change this constant.
const onepayTPS = 2

// queuedProvider wraps a Provider so that EVERY call hitting the OnePay API
// is rate-limited. Each OnePay endpoint gets its own *rate.Limiter so a
// burst of account verifications (e.g. a 500-row employee import) cannot
// starve transfers or balance checks — they draw from independent token
// buckets.
//
// Why per-endpoint instead of one global limiter: OnePay's TPS limit is
// per-operation-type, not aggregate. A 2-TPS limiter shared across all
// endpoints would force a manual disbursement (transfer) to wait behind a
// 500-row import's account checks, even though they are independent
// concerns from OnePay's side.
//
// All wrappers call limiter.Wait(ctx) before delegating. When the caller's
// context deadline expires (e.g. the BankAccountValidator's 5s budget),
// Wait returns an error and the caller fails open — no token consumed,
// no OnePay call made.
type queuedProvider struct {
	inner  *Provider
	logger *slog.Logger

	// Per-endpoint limiters. Each is *rate.Limiter with burst=1 for strict
	// sustained TPS — exactly `onepayTPS` tokens/sec, no bursting.
	transferLimiter *rate.Limiter
	accountLimiter  *rate.Limiter
	balanceLimiter  *rate.Limiter
	statusLimiter   *rate.Limiter

	// mu guards the once-style Close semantics.
	mu     sync.Mutex
	closed bool
}

// Compile-time checks: queuedProvider satisfies the same ports as Provider.
var (
	_ infrastructure.DisbursementProvider = (*queuedProvider)(nil)
	_ infrastructure.AccountVerifier      = (*queuedProvider)(nil)
	_ infrastructure.BalanceReporter      = (*queuedProvider)(nil)
	_ infrastructure.ErrorTranslator      = (*queuedProvider)(nil)
	_ infrastructure.StatusPoller         = (*queuedProvider)(nil)
	_ infrastructure.TransferLimiter      = (*queuedProvider)(nil)
)

// newLimiters builds four independent *rate.Limiters, each pinned to the
// hardcoded onepayTPS with burst=1. Burst=1 enforces a strict sustained
// rate: tokens regenerate one-per-(1/onepayTPS)-seconds and only one call
// may consume a token at a time — no bursting.
func newLimiters() (transfer, account, balance, status *rate.Limiter) {
	limit := rate.Limit(onepayTPS)
	const burst = 1
	return rate.NewLimiter(limit, burst),
		rate.NewLimiter(limit, burst),
		rate.NewLimiter(limit, burst),
		rate.NewLimiter(limit, burst)
}

// NewQueuedProvider wraps inner with per-endpoint rate limiters, each
// hardcoded to onepayTPS (2/sec) with burst=1.
//
// The tps and bufSize parameters are IGNORED — kept in the signature only
// for source-compatibility with the ninepay queue and existing callers.
// The actual rate is the onepayTPS constant. Do not wire this to an env
// var; OnePay's per-endpoint limit is contractual.
func NewQueuedProvider(inner *Provider, _ int, _ int, logger *slog.Logger) *queuedProvider {
	if logger == nil {
		logger = slog.Default()
	}
	transfer, account, balance, status := newLimiters()
	return &queuedProvider{
		inner:           inner,
		logger:          logger,
		transferLimiter: transfer,
		accountLimiter:  account,
		balanceLimiter:  balance,
		statusLimiter:   status,
	}
}

// wait blocks until l grants a token or ctx is cancelled. Returns the
// ctx error on cancellation so callers can fail fast (the validator maps
// any error to BankAccountStatusUnverified — no token consumed, no call).
func wait(ctx context.Context, l *rate.Limiter) error {
	return l.Wait(ctx)
}

// Close is retained for backward-compatibility with callers that defer it
// from the previous channel-based implementation. There is no background
// goroutine to stop with the per-limiter design; this is now a no-op that
// is idempotent and safe to call multiple times.
func (q *queuedProvider) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	return nil
}

// ---------------------------------------------------------------------------
// DisbursementProvider
// ---------------------------------------------------------------------------

func (q *queuedProvider) Name() string { return q.inner.Name() }

func (q *queuedProvider) InitiateTransfer(ctx context.Context, req infrastructure.TransferRequest) (*infrastructure.TransferResult, error) {
	if err := wait(ctx, q.transferLimiter); err != nil {
		return nil, err
	}
	return q.inner.InitiateTransfer(ctx, req)
}

func (q *queuedProvider) VerifyAndParseWebhook(ctx context.Context, payload map[string]any) (*infrastructure.WebhookEvent, error) {
	// Webhook verification is an inbound callback parse — it does NOT call
	// out to OnePay, so no rate limiting applies.
	return q.inner.VerifyAndParseWebhook(ctx, payload)
}

// ---------------------------------------------------------------------------
// AccountVerifier
// ---------------------------------------------------------------------------

func (q *queuedProvider) CheckAccount(ctx context.Context, req infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
	if err := wait(ctx, q.accountLimiter); err != nil {
		return nil, err
	}
	return q.inner.CheckAccount(ctx, req)
}

// ---------------------------------------------------------------------------
// BalanceReporter
// ---------------------------------------------------------------------------

func (q *queuedProvider) GetBalance(ctx context.Context) (*infrastructure.BalanceResult, error) {
	if err := wait(ctx, q.balanceLimiter); err != nil {
		return nil, err
	}
	return q.inner.GetBalance(ctx)
}

// ---------------------------------------------------------------------------
// ErrorTranslator
// ---------------------------------------------------------------------------

func (q *queuedProvider) TranslateError(code string) string { return q.inner.TranslateError(code) }

// ---------------------------------------------------------------------------
// StatusPoller
// ---------------------------------------------------------------------------

func (q *queuedProvider) CheckStatus(ctx context.Context, providerRef string) (*infrastructure.TransferResult, error) {
	if err := wait(ctx, q.statusLimiter); err != nil {
		return nil, err
	}
	return q.inner.CheckStatus(ctx, providerRef)
}

func (q *queuedProvider) Limits() infrastructure.TransferLimits {
	return q.inner.Limits()
}
