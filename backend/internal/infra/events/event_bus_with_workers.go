package events

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// EventBusMetrics tracks event bus performance metrics
type EventBusMetrics struct {
	EventsPublished   atomic.Int64 // Total events published
	EventsProcessed   atomic.Int64 // Total events successfully processed
	EventsDropped     atomic.Int64 // Events dropped due to full queue
	HandlerErrors     atomic.Int64 // Handler execution errors
	HandlerPanics     atomic.Int64 // Handler panics recovered
	TotalProcessingNs atomic.Int64 // Total processing time in nanoseconds
}

// WorkerPoolEventBus is an event bus with bounded worker pool and event queue
type WorkerPoolEventBus struct {
	handlers       map[string][]domain.EventHandler
	globalHandlers []domain.EventHandler
	mu             sync.RWMutex
	logger         *slog.Logger
	metrics        *EventBusMetrics

	// Worker pool configuration
	eventQueue chan *eventTask
	workers    int
	queueSize  int
	wg         sync.WaitGroup
	stopCh     chan struct{}
}

type eventTask struct {
	ctx       context.Context
	event     domain.DomainEvent
	handler   domain.EventHandler
	eventType string
}

// NewWorkerPoolEventBus creates an event bus with worker pool
// workers: number of concurrent workers (e.g., 10)
// queueSize: max buffered events (e.g., 1000)
func NewWorkerPoolEventBus(workers, queueSize int) *WorkerPoolEventBus {
	bus := &WorkerPoolEventBus{
		handlers:       make(map[string][]domain.EventHandler),
		globalHandlers: make([]domain.EventHandler, 0),
		logger:         observability.GetLogger(),
		metrics:        &EventBusMetrics{},
		eventQueue:     make(chan *eventTask, queueSize),
		workers:        workers,
		queueSize:      queueSize,
		stopCh:         make(chan struct{}),
	}

	// Start worker pool
	for i := 0; i < workers; i++ {
		bus.wg.Add(1)
		go bus.worker(i)
	}

	bus.logger.Info("Event bus initialized with worker pool",
		"workers", workers,
		"queue_size", queueSize,
	)

	return bus
}

// worker processes events from the queue
func (bus *WorkerPoolEventBus) worker(id int) {
	defer bus.wg.Done()

	for {
		select {
		case task := <-bus.eventQueue:
			bus.handleEvent(task)
		case <-bus.stopCh:

			return
		}
	}
}

// handleEvent processes a single event task with panic recovery
func (bus *WorkerPoolEventBus) handleEvent(task *eventTask) {
	startTime := time.Now()

	defer func() {
		// Track processing time
		duration := time.Since(startTime)
		bus.metrics.TotalProcessingNs.Add(duration.Nanoseconds())

		// Track panics
		if r := recover(); r != nil {
			bus.metrics.HandlerPanics.Add(1)
			bus.logger.Error("Event handler panicked",
				"event_type", task.eventType,
				"panic", r,
			)
		}
	}()

	// Detach from caller context so async handlers do not fail when HTTP request finishes
	handlerCtx := context.Background()

	// Handle event
	if err := task.handler.Handle(handlerCtx, task.event); err != nil {
		bus.metrics.HandlerErrors.Add(1)
		bus.logger.Error("Event handler failed",
			"event_type", task.eventType,
			"error", err,
		)
	} else {
		bus.metrics.EventsProcessed.Add(1)
	}
}

// Publish publishes events to all registered handlers via worker pool
func (bus *WorkerPoolEventBus) Publish(ctx context.Context, events ...domain.DomainEvent) error {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	var dropped int
	for _, event := range events {
		eventType := event.EventType()

		// Track published event
		bus.metrics.EventsPublished.Add(1)

		bus.logger.Info("Event bus received event", "event_type", eventType, "aggregate_id", event.AggregateID())

		// Collect all handlers for this event
		var handlersToExecute []domain.EventHandler

		// Event-specific handlers
		if handlers, exists := bus.handlers[eventType]; exists {
			handlersToExecute = append(handlersToExecute, handlers...)
			bus.logger.Info("Found event-specific handlers", "event_type", eventType, "count", len(handlers))
		} else {
			bus.logger.Info("No event-specific handlers found", "event_type", eventType)
		}

		// Global handlers
		for _, handler := range bus.globalHandlers {
			if handler.CanHandle(eventType) {
				handlersToExecute = append(handlersToExecute, handler)
			}
		}

		bus.logger.Info("Total handlers to execute", "event_type", eventType, "count", len(handlersToExecute))

		// Queue tasks for workers
		for _, handler := range handlersToExecute {
			task := &eventTask{
				ctx:       ctx,
				event:     event,
				handler:   handler,
				eventType: eventType,
			}

			select {
			case bus.eventQueue <- task:
				bus.logger.Info("Event task queued", "event_type", eventType, "handler", fmt.Sprintf("%T", handler))
				// Task queued successfully
			default:
				// Queue is full - track dropped event
				dropped++
				bus.metrics.EventsDropped.Add(1)
				bus.logger.Error("Event queue full, dropping event",
					"event_type", eventType,
					"queue_size", bus.queueSize,
					"aggregate_id", event.AggregateID(),
				)
			}
		}
	}

	if dropped > 0 {
		return fmt.Errorf("event bus: %d event(s) dropped due to full queue", dropped)
	}
	return nil
}

// Subscribe registers a handler for a specific event type
func (bus *WorkerPoolEventBus) Subscribe(eventType string, handler domain.EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.handlers[eventType] == nil {
		bus.handlers[eventType] = make([]domain.EventHandler, 0)
	}

	bus.handlers[eventType] = append(bus.handlers[eventType], handler)

	bus.logger.Info("Event handler subscribed",
		"event_type", eventType,
	)
}

// SubscribeAll registers a handler for all events
func (bus *WorkerPoolEventBus) SubscribeAll(handler domain.EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.globalHandlers = append(bus.globalHandlers, handler)

	bus.logger.Info("Global event handler subscribed")
}

// ValidateRegistry checks that all subscribed event types have a deserializer
// in the EventRegistry. Returns an error listing any missing registrations.
func (bus *WorkerPoolEventBus) ValidateRegistry(registry *EventRegistry) error {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	var missing []string
	for eventType := range bus.handlers {
		if !registry.HasEventType(eventType) {
			missing = append(missing, eventType)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("event types with subscribed handlers but no registry deserializer: %v — add them to EventRegistry.registerAllEvents()", missing)
	}
	return nil
}

// Shutdown gracefully stops the event bus and waits for workers to finish
func (bus *WorkerPoolEventBus) Shutdown(ctx context.Context) error {
	bus.logger.Info("Shutting down event bus")

	// Close stop channel to signal workers
	close(bus.stopCh)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		bus.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		bus.logger.Info("Event bus shutdown complete")
		return nil
	case <-ctx.Done():
		bus.logger.Warn("Event bus shutdown timed out")
		return ctx.Err()
	}
}

// EventBusMetricsSnapshot represents a point-in-time snapshot of metrics
type EventBusMetricsSnapshot struct {
	EventsPublished   int64
	EventsProcessed   int64
	EventsDropped     int64
	HandlerErrors     int64
	HandlerPanics     int64
	AvgProcessingTime time.Duration
	QueueDepth        int
}

// GetMetrics returns current event bus metrics snapshot
func (bus *WorkerPoolEventBus) GetMetrics() EventBusMetricsSnapshot {
	return EventBusMetricsSnapshot{
		EventsPublished:   bus.metrics.EventsPublished.Load(),
		EventsProcessed:   bus.metrics.EventsProcessed.Load(),
		EventsDropped:     bus.metrics.EventsDropped.Load(),
		HandlerErrors:     bus.metrics.HandlerErrors.Load(),
		HandlerPanics:     bus.metrics.HandlerPanics.Load(),
		AvgProcessingTime: bus.AvgProcessingTime(),
		QueueDepth:        bus.QueueDepth(),
	}
}

// QueueDepth returns current number of events in queue
func (bus *WorkerPoolEventBus) QueueDepth() int {
	return len(bus.eventQueue)
}

// AvgProcessingTime returns average event processing time
func (bus *WorkerPoolEventBus) AvgProcessingTime() time.Duration {
	processed := bus.metrics.EventsProcessed.Load()
	if processed == 0 {
		return 0
	}

	totalNs := bus.metrics.TotalProcessingNs.Load()
	return time.Duration(totalNs / processed)
}
