package timesheet

import (
	"context"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	"gorm.io/gorm"
)

// TransactionOrchestrator handles transactional operations with event publishing
type TransactionOrchestrator struct {
	tm     *infrastructure.TransactionManager
	events domain.EventBus
}

// NewTransactionOrchestrator creates a new transaction orchestrator
func NewTransactionOrchestrator(tm *infrastructure.TransactionManager, events domain.EventBus) *TransactionOrchestrator {
	return &TransactionOrchestrator{
		tm:     tm,
		events: events,
	}
}

// ExecuteInTransaction executes an operation within a transaction
func (to *TransactionOrchestrator) ExecuteInTransaction(ctx context.Context, operation func(tx *gorm.DB) error) error {
	return to.tm.ExecuteInTransaction(ctx, operation)
}

// ExecuteInTransactionWithEvent executes an operation in transaction with optional event publishing
func (to *TransactionOrchestrator) ExecuteInTransactionWithEvent(
	ctx context.Context,
	operation func(tx *gorm.DB) error,
	eventBuilder func() domain.DomainEvent,
) error {
	// Execute operation in transaction
	err := to.tm.ExecuteInTransaction(ctx, operation)
	if err != nil {
		return err
	}

	// Publish event after successful transaction
	if eventBuilder != nil {
		event := eventBuilder()
		if err := to.events.Publish(ctx, event); err != nil {
			// Log error but don't fail the operation
			// Event publishing failure should be retried by outbox pattern
			return err
		}
	}

	return nil
}

// ExecuteInTransactionWithEvents executes an operation in transaction with multiple events
func (to *TransactionOrchestrator) ExecuteInTransactionWithEvents(
	ctx context.Context,
	operation func(tx *gorm.DB) error,
	eventBuilders []func() domain.DomainEvent,
) error {
	// Execute operation in transaction
	err := to.tm.ExecuteInTransaction(ctx, operation)
	if err != nil {
		return err
	}

	// Publish all events after successful transaction
	for _, eventBuilder := range eventBuilders {
		event := eventBuilder()
		if err := to.events.Publish(ctx, event); err != nil {
			// Log error but don't fail the operation
			return err
		}
	}

	return nil
}
