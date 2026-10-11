// Package llm is the application service around the LLM provider port. It owns
// the seam between business callers and the provider: model routing (allowlist
// + runtime settings), the shared-key guards (per-user rate limit, concurrency
// semaphore), and credential resolution from settings. Callers depend on this
// Service — never on the concrete HTTP client — so the provider stays swappable
// behind internal/infra/llm's Client port.
//
// Naming note: this module is `llm` (assistant substrate) — never "chatbot
// service": an unrelated external "TingTing chatbot" integration already lives
// under internal/app/services/integration.
package llm

import (
	"context"
	"log/slog"
	"strings"
	"time"

	llmport "api-server/internal/infra/llm"
	"api-server/internal/infra/observability"
)

// GuardWait bounds how long a call waits for a concurrency slot before giving
// up with a designed "busy" error, so a saturated shared key fails fast instead
// of queueing unboundedly. A var so tests can shorten it.
var GuardWait = 10 * time.Second

// SettingsSource is the read seam over the admin-managed MiMo settings
// (implemented by config.SettingsConfigService at bootstrap).
type SettingsSource interface {
	GetMimoAPIKey(ctx context.Context) string
	GetMimoBaseURL(ctx context.Context) string
	GetMimoDefaultModel(ctx context.Context) string
	GetMimoFallbackModel(ctx context.Context) string
}

// ClientFactory builds a provider client from runtime settings. Injectable so
// tests can substitute a fake Client without HTTP.
type ClientFactory func(cfg llmport.Config) (llmport.Client, error)

// Service orchestrates guarded LLM calls. Safe for concurrent use.
type Service struct {
	settings  SettingsSource
	guard     *llmport.Guard
	newClient ClientFactory
	logger    *slog.Logger
}

// NewService builds the LLM service. guard may be nil (no shared-key limits —
// tests only); the factory defaults to the real OpenAI-compatible client.
func NewService(settings SettingsSource, guard *llmport.Guard, factory ClientFactory) *Service {
	if factory == nil {
		factory = func(cfg llmport.Config) (llmport.Client, error) {
			return llmport.NewClient(cfg)
		}
	}
	if guard == nil {
		guard = llmport.NewGuard(llmport.NewSemaphore(1), nil)
	}
	return &Service{
		settings:  settings,
		guard:     guard,
		newClient: factory,
		logger:    observability.GetLogger(),
	}
}

// TestConnection validates the stored MiMo API key against the provider and
// reports latency. Missing keys fail fast with a designed error state — the
// admin UI renders "Chưa có khóa" instead of a network failure.
func (s *Service) TestConnection(ctx context.Context) (*llmport.TestConnectionResult, error) {
	client, err := s.clientForCall(ctx)
	if err != nil {
		return nil, err
	}

	release, err := s.acquireSlot(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	return client.TestConnection(ctx)
}

// Chat runs one guarded chat turn. req.Model is resolved through the allowlist
// (empty means the runtime default); a model-not-found failure retries once
// with the configured fallback model, which is exactly the "vendor renamed its
// model family" failure mode.
func (s *Service) Chat(ctx context.Context, userID uint, req llmport.ChatRequest) (*llmport.ChatResponse, error) {
	model, err := llmport.ResolveModel(req.Model, s.settings.GetMimoDefaultModel(ctx))
	if err != nil {
		return nil, err
	}
	req.Model = model

	client, err := s.clientForCall(ctx)
	if err != nil {
		return nil, err
	}

	if allowed, limiterErr := s.guard.AllowUser(ctx, userID); limiterErr != nil {
		// A malfunctioning guard must not take the LLM surface down: log and
		// fail open (the concurrency semaphore still applies).
		s.logger.Warn("llm: per-user rate limiter failed, failing open", "error", limiterErr)
	} else if !allowed {
		return nil, &llmport.APIError{Kind: llmport.KindRateLimited, Message: llmport.MessageFor(llmport.KindRateLimited)}
	}

	release, err := s.acquireSlot(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	resp, err := client.Chat(ctx, req)
	if err == nil {
		return resp, nil
	}

	// Fallback model: one retry when the primary model no longer exists.
	if llmport.KindOf(err) == llmport.KindModelNotFound {
		if fallback, fallbackErr := s.fallbackModel(ctx); fallbackErr == nil && fallback != "" && fallback != req.Model {
			req.Model = fallback
			return client.Chat(ctx, req)
		}
	}
	return nil, err
}

// fallbackModel resolves the configured fallback through the same allowlist as
// the primary route, so a retired family can never be configured as fallback.
func (s *Service) fallbackModel(ctx context.Context) (string, error) {
	return llmport.ResolveModel(s.settings.GetMimoFallbackModel(ctx), "")
}

// clientForCall builds a provider client from the current settings. The API key
// is read per call (never cached) so an admin key rotation takes effect on the
// very next request.
func (s *Service) clientForCall(ctx context.Context) (llmport.Client, error) {
	key := strings.TrimSpace(s.settings.GetMimoAPIKey(ctx))
	if key == "" {
		return nil, &llmport.APIError{Kind: llmport.KindMissingKey, Message: llmport.MessageFor(llmport.KindMissingKey)}
	}
	return s.newClient(llmport.Config{
		BaseURL: s.settings.GetMimoBaseURL(ctx),
		APIKey:  key,
	})
}

// acquireSlot waits (bounded) for a concurrency slot. Caller-context expiry is
// reported as-is (timeout/cancel); a full guard is a designed "busy" error.
func (s *Service) acquireSlot(ctx context.Context) (func(), error) {
	waitCtx, cancel := context.WithTimeout(ctx, GuardWait)
	release, err := s.guard.Acquire(waitCtx)
	cancel()
	if err != nil {
		if ctx.Err() != nil {
			return nil, llmport.ContextError(ctx)
		}
		return nil, &llmport.APIError{Kind: llmport.KindBusy, Message: llmport.MessageFor(llmport.KindBusy)}
	}
	return release, nil
}
