package events

import (
	"context"
	"log/slog"
	"math"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// RetryHandler wraps a domain.EventHandler with configurable retry and exponential backoff.
type RetryHandler struct {
	inner     domain.EventHandler
	maxTries  int
	baseDelay time.Duration
	logger    *slog.Logger
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxTries  int           // total attempts (including initial)
	BaseDelay time.Duration // initial backoff delay
}

// DefaultRetryConfig returns sensible defaults: 3 attempts, 100ms base delay.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxTries:  3,
		BaseDelay: 100 * time.Millisecond,
	}
}

// NewRetryHandler wraps a handler with retry logic.
func NewRetryHandler(inner domain.EventHandler, config RetryConfig) *RetryHandler {
	return &RetryHandler{
		inner:     inner,
		maxTries:  config.MaxTries,
		baseDelay: config.BaseDelay,
		logger:    observability.GetLogger(),
	}
}

// Handle executes the inner handler with retry and exponential backoff.
func (h *RetryHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	var lastErr error
	for attempt := 0; attempt < h.maxTries; attempt++ {
		if err := h.inner.Handle(ctx, event); err != nil {
			lastErr = err
			if attempt < h.maxTries-1 {
				delay := h.baseDelay * time.Duration(math.Pow(2, float64(attempt)))
				h.logger.Warn("event handler failed, retrying",
					"event_type", event.EventType(),
					"attempt", attempt+1,
					"max_tries", h.maxTries,
					"retry_in", delay,
					"error", err,
				)
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			continue
		}
		return nil
	}
	h.logger.Error("event handler failed after all retries",
		"event_type", event.EventType(),
		"max_tries", h.maxTries,
		"error", lastErr,
	)
	return lastErr
}

// CanHandle delegates to the inner handler.
func (h *RetryHandler) CanHandle(eventType string) bool {
	return h.inner.CanHandle(eventType)
}
