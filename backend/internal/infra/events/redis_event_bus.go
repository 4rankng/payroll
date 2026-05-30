package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"github.com/redis/go-redis/v9"
)

// RedisEventBus is a distributed event bus backed by Redis with worker pool
type RedisEventBus struct {
	redis          *redis.Client
	handlers       map[string][]domain.EventHandler
	globalHandlers []domain.EventHandler
	mu             sync.RWMutex
	logger         *slog.Logger
	eventRegistry  *EventRegistry // Event type registry for deserialization
	metrics        *EventBusMetrics

	// Worker pool configuration
	eventQueue chan *redisEventTask
	workers    int
	queueSize  int
	wg         sync.WaitGroup
	stopCh     chan struct{}

	// Redis configuration
	eventStream    string        // Redis stream name
	consumerGroup  string        // Consumer group for distributed processing
	consumerName   string        // Unique consumer ID
	eventTTL       time.Duration // How long to keep events in Redis
	processingLock time.Duration // Lock timeout for processing events
}

type redisEventTask struct {
	ctx       context.Context
	event     domain.DomainEvent
	handler   domain.EventHandler
	eventType string
	eventID   string // Redis stream message ID
}

// StoredEvent represents an event stored in Redis
type StoredEvent struct {
	EventType   string          `json:"event_type"`
	EventData   json.RawMessage `json:"event_data"`
	AggregateID uint            `json:"aggregate_id"`
	UserID      uint            `json:"user_id"`
	OccurredAt  time.Time       `json:"occurred_at"`
}

// NewRedisEventBus creates a distributed event bus with Redis backing
func NewRedisEventBus(
	redisClient *redis.Client,
	workers int,
	queueSize int,
	consumerName string,
) *RedisEventBus {
	bus := &RedisEventBus{
		redis:          redisClient,
		handlers:       make(map[string][]domain.EventHandler),
		globalHandlers: make([]domain.EventHandler, 0),
		logger:         observability.GetLogger(),
		eventRegistry:  NewEventRegistry(), // Initialize event registry
		metrics:        &EventBusMetrics{},
		eventQueue:     make(chan *redisEventTask, queueSize),
		workers:        workers,
		queueSize:      queueSize,
		stopCh:         make(chan struct{}),
		eventStream:    "payroll:events",
		consumerGroup:  "payroll-workers",
		consumerName:   consumerName,
		eventTTL:       24 * time.Hour, // Keep events for 24 hours
		processingLock: 30 * time.Second,
	}

	// Create consumer group (idempotent)
	bus.createConsumerGroup()

	// Load existing metrics from Redis
	bus.loadMetricsFromRedis()

	// Start worker pool
	for i := 0; i < workers; i++ {
		bus.wg.Add(1)
		go bus.worker(i)
	}

	// Start Redis stream consumer
	bus.wg.Add(1)
	go bus.consumeFromRedis()

	bus.logger.Info("Redis event bus initialized",
		"workers", workers,
		"queue_size", queueSize,
		"consumer_name", consumerName,
	)

	return bus
}

// createConsumerGroup creates the Redis consumer group if it doesn't exist
func (bus *RedisEventBus) createConsumerGroup() {
	ctx := context.Background()

	// Create stream if it doesn't exist by adding a dummy message
	exists, _ := bus.redis.Exists(ctx, bus.eventStream).Result()
	if exists == 0 {
		bus.redis.XAdd(ctx, &redis.XAddArgs{
			Stream: bus.eventStream,
			Values: map[string]interface{}{"_init": "true"},
		})
	}

	// Create consumer group (MKSTREAM creates stream if needed)
	err := bus.redis.XGroupCreateMkStream(ctx, bus.eventStream, bus.consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		bus.logger.Error("Failed to create consumer group", "error", err)
	}
}

// loadMetricsFromRedis loads persisted metrics from Redis on startup
func (bus *RedisEventBus) loadMetricsFromRedis() {
	ctx := context.Background()

	// Load each metric from Redis
	eventsPublished, _ := bus.redis.Get(ctx, "eventbus:metrics:published").Int64()
	eventsProcessed, _ := bus.redis.Get(ctx, "eventbus:metrics:processed").Int64()
	eventsDropped, _ := bus.redis.Get(ctx, "eventbus:metrics:dropped").Int64()
	handlerErrors, _ := bus.redis.Get(ctx, "eventbus:metrics:errors").Int64()
	handlerPanics, _ := bus.redis.Get(ctx, "eventbus:metrics:panics").Int64()
	totalProcessingNs, _ := bus.redis.Get(ctx, "eventbus:metrics:processing_ns").Int64()

	// Initialize atomic values with persisted data
	bus.metrics.EventsPublished.Store(eventsPublished)
	bus.metrics.EventsProcessed.Store(eventsProcessed)
	bus.metrics.EventsDropped.Store(eventsDropped)
	bus.metrics.HandlerErrors.Store(handlerErrors)
	bus.metrics.HandlerPanics.Store(handlerPanics)
	bus.metrics.TotalProcessingNs.Store(totalProcessingNs)

	bus.logger.Info("Loaded metrics from Redis",
		"published", eventsPublished,
		"processed", eventsProcessed,
		"dropped", eventsDropped,
		"errors", handlerErrors,
		"panics", handlerPanics,
	)
}

// incrementMetric atomically increments both in-memory and Redis metric
func (bus *RedisEventBus) incrementMetric(ctx context.Context, metricName string, atomicCounter *atomic.Int64, delta int64) {
	// Increment in-memory counter
	atomicCounter.Add(delta)

	// Persist to Redis asynchronously (fire-and-forget to avoid blocking)
	go func() {
		redisKey := "eventbus:metrics:" + metricName
		err := bus.redis.IncrBy(ctx, redisKey, delta).Err()
		if err != nil {
			bus.logger.Warn("Failed to persist metric to Redis",
				"metric", metricName,
				"error", err,
			)
		}
	}()
}

// Publish publishes events to Redis stream
func (bus *RedisEventBus) Publish(ctx context.Context, events ...domain.DomainEvent) error {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	for _, event := range events {
		eventType := event.EventType()

		// Track published event
		bus.incrementMetric(ctx, "published", &bus.metrics.EventsPublished, 1)

		// Serialize event to JSON
		eventData, err := json.Marshal(event)
		if err != nil {
			bus.logger.Error("Failed to marshal event", "event_type", eventType, "error", err)
			continue
		}

		// Store event in Redis stream
		storedEvent := StoredEvent{
			EventType:   eventType,
			EventData:   eventData,
			AggregateID: event.AggregateID(),
			UserID:      event.UserID(),
			OccurredAt:  event.OccurredAt(),
		}

		storedData, _ := json.Marshal(storedEvent)

		// Add to Redis stream with TTL
		_, err = bus.redis.XAdd(ctx, &redis.XAddArgs{
			Stream: bus.eventStream,
			MaxLen: 10000, // Keep last 10K events to prevent unbounded growth
			Approx: true,  // Allow approximate trimming for performance
			Values: map[string]interface{}{
				"event": string(storedData),
			},
		}).Result()

		if err != nil {
			bus.logger.Error("Failed to publish event to Redis",
				"event_type", eventType,
				"error", err,
			)
			continue
		}

	}

	return nil
}

// consumeFromRedis continuously reads events from Redis stream
func (bus *RedisEventBus) consumeFromRedis() {
	defer bus.wg.Done()

	ctx := context.Background()

	for {
		select {
		case <-bus.stopCh:
			bus.logger.Info("Stopping Redis consumer")
			return
		default:
			// Read from Redis stream
			streams, err := bus.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    bus.consumerGroup,
				Consumer: bus.consumerName,
				Streams:  []string{bus.eventStream, ">"}, // ">" means new messages
				Count:    10,                             // Read up to 10 messages
				Block:    1 * time.Second,                // Block for 1 second
			}).Result()

			if err != nil {
				if err != redis.Nil {
					bus.logger.Warn("Failed to read from Redis stream", "error", err)
				}
				continue
			}

			// Process each message
			for _, stream := range streams {
				for _, message := range stream.Messages {
					bus.processRedisMessage(ctx, message)
				}
			}
		}
	}
}

// processRedisMessage processes a single message from Redis with race condition protection
func (bus *RedisEventBus) processRedisMessage(ctx context.Context, message redis.XMessage) {
	eventData, ok := message.Values["event"].(string)
	if !ok {
		bus.logger.Warn("Invalid event data in Redis message", "message_id", message.ID)
		bus.ackMessage(ctx, message.ID)
		return
	}

	// Deserialize stored event
	var storedEvent StoredEvent
	if err := json.Unmarshal([]byte(eventData), &storedEvent); err != nil {
		bus.logger.Error("Failed to unmarshal stored event",
			"message_id", message.ID,
			"error", err,
		)
		bus.ackMessage(ctx, message.ID)
		return
	}

	// RACE CONDITION PROTECTION: Use Redis lock to ensure only one consumer processes this event
	lockKey := fmt.Sprintf("lock:event:%s", message.ID)
	locked, err := bus.acquireLock(ctx, lockKey, bus.processingLock)
	if err != nil || !locked {

		return
	}

	// Ensure lock is released
	defer bus.releaseLock(ctx, lockKey)

	// Dispatch to local handlers
	bus.dispatchToHandlers(ctx, storedEvent, message.ID)

	// ACK message in Redis
	bus.ackMessage(ctx, message.ID)
}

// acquireLock acquires a distributed lock using Redis SETNX
func (bus *RedisEventBus) acquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	// Use SET with NX (not exists) and EX (expiry)
	result, err := bus.redis.SetNX(ctx, key, bus.consumerName, ttl).Result()
	if err != nil {
		return false, err
	}

	return result, nil
}

// releaseLock releases a distributed lock
func (bus *RedisEventBus) releaseLock(ctx context.Context, key string) {
	// Only delete if we own the lock
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	_, err := bus.redis.Eval(ctx, script, []string{key}, bus.consumerName).Result()
	if err != nil {
		bus.logger.Warn("Failed to release lock", "key", key, "error", err)
	}
}

// dispatchToHandlers dispatches event to registered handlers
func (bus *RedisEventBus) dispatchToHandlers(ctx context.Context, storedEvent StoredEvent, eventID string) {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// Reconstruct domain event from stored data
	event, err := bus.deserializeEvent(storedEvent)
	if err != nil {
		bus.logger.Error("Failed to deserialize event",
			"event_type", storedEvent.EventType,
			"error", err,
		)
		return
	}

	// Collect handlers
	var handlersToExecute []domain.EventHandler

	// Event-specific handlers
	if handlers, exists := bus.handlers[storedEvent.EventType]; exists {
		handlersToExecute = append(handlersToExecute, handlers...)
	}

	// Global handlers
	for _, handler := range bus.globalHandlers {
		if handler.CanHandle(storedEvent.EventType) {
			handlersToExecute = append(handlersToExecute, handler)
		}
	}

	// Queue tasks for workers
	for _, handler := range handlersToExecute {
		task := &redisEventTask{
			ctx:       ctx,
			event:     event,
			handler:   handler,
			eventType: storedEvent.EventType,
			eventID:   eventID,
		}

		select {
		case bus.eventQueue <- task:
			// Task queued successfully
		default:
			// Queue is full - track dropped event
			bus.incrementMetric(ctx, "dropped", &bus.metrics.EventsDropped, 1)
			bus.logger.Warn("Event queue full, skipping handler",
				"event_type", storedEvent.EventType,
				"event_id", eventID,
			)
		}
	}
}

// deserializeEvent reconstructs a domain event from stored data
func (bus *RedisEventBus) deserializeEvent(storedEvent StoredEvent) (domain.DomainEvent, error) {
	return bus.eventRegistry.Deserialize(storedEvent.EventType, storedEvent.EventData)
}

// ackMessage acknowledges a message in Redis stream
func (bus *RedisEventBus) ackMessage(ctx context.Context, messageID string) {
	err := bus.redis.XAck(ctx, bus.eventStream, bus.consumerGroup, messageID).Err()
	if err != nil {
		bus.logger.Error("Failed to ACK message",
			"message_id", messageID,
			"error", err,
		)
	}
}

// worker processes events from the queue
func (bus *RedisEventBus) worker(id int) {
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
func (bus *RedisEventBus) handleEvent(task *redisEventTask) {
	startTime := time.Now()

	defer func() {
		// Track processing time
		duration := time.Since(startTime)
		bus.incrementMetric(task.ctx, "processing_ns", &bus.metrics.TotalProcessingNs, duration.Nanoseconds())

		// Track panics
		if r := recover(); r != nil {
			bus.incrementMetric(task.ctx, "panics", &bus.metrics.HandlerPanics, 1)
			bus.logger.Error("Event handler panicked",
				"event_type", task.eventType,
				"event_id", task.eventID,
				"panic", r,
			)
		}
	}()

	// Detach context so event handlers run independently from Redis connection context
	handlerCtx := context.Background()

	// Handle event
	if err := task.handler.Handle(handlerCtx, task.event); err != nil {
		bus.incrementMetric(task.ctx, "errors", &bus.metrics.HandlerErrors, 1)
		bus.logger.Error("Event handler failed",
			"event_type", task.eventType,
			"event_id", task.eventID,
			"error", err,
		)
	} else {
		bus.incrementMetric(task.ctx, "processed", &bus.metrics.EventsProcessed, 1)
	}
}

// Subscribe registers a handler for a specific event type
func (bus *RedisEventBus) Subscribe(eventType string, handler domain.EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.handlers[eventType] == nil {
		bus.handlers[eventType] = make([]domain.EventHandler, 0)
	}

	bus.handlers[eventType] = append(bus.handlers[eventType], handler)

	bus.logger.Info("Event handler subscribed", "event_type", eventType)
}

// SubscribeAll registers a handler for all events
func (bus *RedisEventBus) SubscribeAll(handler domain.EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.globalHandlers = append(bus.globalHandlers, handler)

	bus.logger.Info("Global event handler subscribed")
}

// Shutdown gracefully stops the event bus
func (bus *RedisEventBus) Shutdown(ctx context.Context) error {
	bus.logger.Info("Shutting down Redis event bus")

	// Close stop channel to signal workers and consumer
	close(bus.stopCh)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		bus.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		bus.logger.Info("Redis event bus shutdown complete")
		return nil
	case <-ctx.Done():
		bus.logger.Warn("Redis event bus shutdown timed out")
		return ctx.Err()
	}
}

// GetMetrics returns current event bus metrics snapshot
func (bus *RedisEventBus) GetMetrics() EventBusMetricsSnapshot {
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
func (bus *RedisEventBus) QueueDepth() int {
	return len(bus.eventQueue)
}

// AvgProcessingTime returns average event processing time
func (bus *RedisEventBus) AvgProcessingTime() time.Duration {
	processed := bus.metrics.EventsProcessed.Load()
	if processed == 0 {
		return 0
	}

	totalNs := bus.metrics.TotalProcessingNs.Load()
	return time.Duration(totalNs / processed)
}
