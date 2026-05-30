package onepay

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"api-server/internal/domain/ports/infrastructure"

	"golang.org/x/time/rate"
)

// queuedProvider wraps a Provider so that InitiateTransfer calls are pushed
// into a buffered channel and drained by a single background goroutine at the
// configured TPS. Callers block until the worker returns the result (or their
// context is cancelled), so the DisbursementProvider interface stays
// synchronous from the caller's perspective.
//
// All other methods (CheckAccount, GetBalance, CheckStatus,
// VerifyAndParseWebhook, TranslateError) are delegated directly to the inner
// Provider — only transfers need throttling because they are the high-volume,
// externally-visible operation that OnePay rate-limits.
type queuedProvider struct {
	inner  *Provider
	logger *slog.Logger
	queue  chan transferJob
	done   chan struct{}
	wg     sync.WaitGroup
}

type transferJob struct {
	ctx    context.Context
	req    infrastructure.TransferRequest
	result chan<- transferJobResult
}

type transferJobResult struct {
	result *infrastructure.TransferResult
	err    error
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

// NewQueuedProvider wraps inner with a channel-based queue drained at tps
// transfers per second. When the queue buffer (bufSize) is full, callers
// receive an error immediately (back-pressure). Call Close on shutdown to
// stop the worker goroutine.
func NewQueuedProvider(inner *Provider, tps int, bufSize int, logger *slog.Logger) *queuedProvider {
	if logger == nil {
		logger = slog.Default()
	}
	if tps <= 0 {
		tps = 3
	}
	if bufSize <= 0 {
		bufSize = 100
	}
	q := &queuedProvider{
		inner:  inner,
		logger: logger,
		queue:  make(chan transferJob, bufSize),
		done:   make(chan struct{}),
	}
	q.wg.Add(1)
	go q.run(tps)
	return q
}

// shutdownContext returns a child of parent that is also cancelled when the
// provider's done channel closes, so in-flight HTTP calls unblock on Close.
func (q *queuedProvider) shutdownContext(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	go func() {
		select {
		case <-q.done:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}

// run drains the queue at tps using a rate.Limiter (burst=1 for strict TPS).
// Stops when Close is called. On exit, drains any remaining pending jobs with
// a shutdown error so callers don't block forever.
func (q *queuedProvider) run(tps int) {
	defer q.wg.Done()
	limiter := rate.NewLimiter(rate.Limit(tps), 1)
	for {
		select {
		case job, ok := <-q.queue:
			if !ok {
				return
			}
			ctx, cancel := q.shutdownContext(job.ctx)
			if err := limiter.Wait(ctx); err != nil {
				cancel()
				job.result <- transferJobResult{err: fmt.Errorf("onepay: shutting down")}
				q.drainPending()
				return
			}
			res, err := q.inner.InitiateTransfer(ctx, job.req)
			cancel()
			job.result <- transferJobResult{result: res, err: err}
		case <-q.done:
			q.drainPending()
			return
		}
	}
}

// drainPending sends a shutdown error to any callers still waiting for results.
func (q *queuedProvider) drainPending() {
	for {
		select {
		case job := <-q.queue:
			job.result <- transferJobResult{err: fmt.Errorf("onepay: provider shut down")}
		default:
			return
		}
	}
}

// Close stops the worker goroutine and cancels in-flight transfers. Safe to
// call multiple times.
func (q *queuedProvider) Close() error {
	select {
	case <-q.done:
		return nil
	default:
		close(q.done)
	}
	q.wg.Wait()
	return nil
}

// ---------------------------------------------------------------------------
// DisbursementProvider
// ---------------------------------------------------------------------------

func (q *queuedProvider) Name() string { return q.inner.Name() }

func (q *queuedProvider) InitiateTransfer(ctx context.Context, req infrastructure.TransferRequest) (*infrastructure.TransferResult, error) {
	ch := make(chan transferJobResult, 1)
	select {
	case q.queue <- transferJob{ctx: ctx, req: req, result: ch}:
	default:
		return nil, fmt.Errorf("onepay: transfer queue full (%d pending); retry later", len(q.queue))
	}

	select {
	case r := <-ch:
		return r.result, r.err
	case <-ctx.Done():
		return nil, fmt.Errorf("onepay: transfer cancelled while queued: %w", ctx.Err())
	case <-q.done:
		return nil, fmt.Errorf("onepay: provider shut down")
	}
}

func (q *queuedProvider) VerifyAndParseWebhook(ctx context.Context, payload map[string]any) (*infrastructure.WebhookEvent, error) {
	return q.inner.VerifyAndParseWebhook(ctx, payload)
}

// ---------------------------------------------------------------------------
// AccountVerifier
// ---------------------------------------------------------------------------

func (q *queuedProvider) CheckAccount(ctx context.Context, req infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
	return q.inner.CheckAccount(ctx, req)
}

// ---------------------------------------------------------------------------
// BalanceReporter
// ---------------------------------------------------------------------------

func (q *queuedProvider) GetBalance(ctx context.Context) (*infrastructure.BalanceResult, error) {
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
	return q.inner.CheckStatus(ctx, providerRef)
}

func (q *queuedProvider) Limits() infrastructure.TransferLimits {
	return q.inner.Limits()
}
