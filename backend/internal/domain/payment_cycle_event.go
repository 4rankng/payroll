package domain

import (
	"context"
	"time"

	"api-server/internal/pkg/clock"
)

// PayCycleChangedEvent is published when an employee's payment schedule changes.
// This implements the Observer pattern allowing multiple subscribers to react
// to payment cycle changes (e.g., notifications, approval workflows, audit logs).
type PayCycleChangedEvent struct {
	ProjectID     uint
	EmployeeID    uint
	OldCycle      string    // "weekly" or "monthly"
	NewCycle      string    // "weekly" or "monthly"
	EffectiveFrom time.Time // When the change takes effect
	IsImmediate   bool      // True if applied immediately, false if deferred
	ChangedBy     *uint     // User ID who initiated the change (nil for system)
	ChangedAt     time.Time
	ProjectName   string // For notification context
	EmployeeName  string // For notification context
}

// PayCycleEventPublisher allows application services to notify observers
// when payment schedule changes occur.
type PayCycleEventPublisher interface {
	PublishPayCycleChanged(ctx context.Context, event *PayCycleChangedEvent) error
}

// NewPayCycleChangedEvent creates an event with sane defaults.
func NewPayCycleChangedEvent(
	projectID, employeeID uint,
	oldCycle, newCycle string,
	effectiveFrom time.Time,
	isImmediate bool,
	changedBy *uint,
	projectName, employeeName string,
) *PayCycleChangedEvent {
	return &PayCycleChangedEvent{
		ProjectID:     projectID,
		EmployeeID:    employeeID,
		OldCycle:      oldCycle,
		NewCycle:      newCycle,
		EffectiveFrom: effectiveFrom,
		IsImmediate:   isImmediate,
		ChangedBy:     changedBy,
		ChangedAt:     clock.Now(),
		ProjectName:   projectName,
		EmployeeName:  employeeName,
	}
}
