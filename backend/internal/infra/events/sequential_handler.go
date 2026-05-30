package events

import (
	"context"

	"api-server/internal/domain"
)

// SequentialEventHandler executes a list of handlers sequentially for each event.
// It is useful when we want to avoid concurrent handler execution for a single event type
// but still keep each handler's own behavior (including coordination and tracking).
type SequentialEventHandler struct {
	handlers []domain.EventHandler
}

// NewSequentialEventHandler creates a new sequential event handler wrapper.
func NewSequentialEventHandler(handlers []domain.EventHandler) *SequentialEventHandler {
	return &SequentialEventHandler{
		handlers: handlers,
	}
}

// Handle runs all inner handlers sequentially. If any handler returns an error,
// it stops and returns that error.
func (h *SequentialEventHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// CanHandle returns true if any inner handler can handle the given eventType.
func (h *SequentialEventHandler) CanHandle(eventType string) bool {
	for _, handler := range h.handlers {
		if handler.CanHandle(eventType) {
			return true
		}
	}
	return false
}
