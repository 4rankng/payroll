package events

import (
	"context"
	"log/slog"
	"sync"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// InMemoryEventBus is a simple in-memory event bus implementation
type InMemoryEventBus struct {
	handlers       map[string][]domain.EventHandler
	globalHandlers []domain.EventHandler
	mu             sync.RWMutex
	logger         *slog.Logger
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers:       make(map[string][]domain.EventHandler),
		globalHandlers: make([]domain.EventHandler, 0),
		logger:         observability.GetLogger(),
	}
}

// Publish publishes events to all registered handlers asynchronously
func (bus *InMemoryEventBus) Publish(ctx context.Context, events ...domain.DomainEvent) error {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	for _, event := range events {
		eventType := event.EventType()

		// Process event-specific handlers asynchronously
		if handlers, exists := bus.handlers[eventType]; exists {
			for _, handler := range handlers {
				go bus.handleEventAsync(ctx, event, handler, eventType)
			}
		}

		// Process global handlers asynchronously
		for _, handler := range bus.globalHandlers {
			if handler.CanHandle(eventType) {
				go bus.handleEventAsync(ctx, event, handler, eventType)
			}
		}
	}

	return nil
}

// handleEventAsync processes an event handler in a goroutine with panic recovery
func (bus *InMemoryEventBus) handleEventAsync(ctx context.Context, event domain.DomainEvent, handler domain.EventHandler, eventType string) {
	defer func() {
		if r := recover(); r != nil {
			bus.logger.Error("Event handler panicked",
				"event_type", eventType,
				"handler", handler,
				"panic", r,
			)
		}
	}()

	handlerCtx := context.Background()

	if err := handler.Handle(handlerCtx, event); err != nil {
		bus.logger.Error("Event handler failed",
			"event_type", eventType,
			"handler", handler,
			"error", err,
		)
	}
}

// Subscribe registers a handler for a specific event type
func (bus *InMemoryEventBus) Subscribe(eventType string, handler domain.EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.handlers[eventType] == nil {
		bus.handlers[eventType] = make([]domain.EventHandler, 0)
	}

	bus.handlers[eventType] = append(bus.handlers[eventType], handler)

	bus.logger.Info("Event handler subscribed",
		"event_type", eventType,
		"handler", handler,
	)
}

// SubscribeAll registers a handler for all events
func (bus *InMemoryEventBus) SubscribeAll(handler domain.EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.globalHandlers = append(bus.globalHandlers, handler)

	bus.logger.Info("Global event handler subscribed",
		"handler", handler,
	)
}
