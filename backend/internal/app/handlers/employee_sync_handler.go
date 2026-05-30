package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"api-server/internal/app/services/employee"
	"api-server/internal/domain"
)

// EmployeeSyncHandler handles employee name synchronization events
type EmployeeSyncHandler struct {
	employeeSyncService employee.EmployeeSyncServiceInterface
}

// NewEmployeeSyncHandler creates a new employee sync handler
func NewEmployeeSyncHandler(employeeSyncService employee.EmployeeSyncServiceInterface) *EmployeeSyncHandler {
	return &EmployeeSyncHandler{
		employeeSyncService: employeeSyncService,
	}
}

// Handle handles domain events
func (h *EmployeeSyncHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	switch e := event.(type) {
	case domain.EmployeeNameUpdatedEvent:
		return h.handleEmployeeNameUpdated(ctx, e)
	default:
		return fmt.Errorf("unsupported event type: %T", event)
	}
}

// CanHandle checks if the handler can handle the given event type
func (h *EmployeeSyncHandler) CanHandle(eventType string) bool {
	return eventType == "EmployeeNameUpdated"
}

// handleEmployeeNameUpdated handles EmployeeNameUpdatedEvent
func (h *EmployeeSyncHandler) handleEmployeeNameUpdated(ctx context.Context, event domain.EmployeeNameUpdatedEvent) error {
	// Sync the employee name to active and upcoming project assignments
	result, err := h.employeeSyncService.SyncEmployeeNameToActiveAssignments(
		ctx,
		event.AggregateID(),
		event.NewFullName,
	)

	if err != nil {
		return fmt.Errorf("failed to sync employee name for employee %d: %w", event.AggregateID(), err)
	}

	if result.UpdatedCount > 0 {
		slog.Info("synced employee name to assignments",
			"employee_id", event.AggregateID(),
			"updated_count", result.UpdatedCount,
			"old_name", event.OldFullName,
			"new_name", event.NewFullName)
	}

	return nil
}
