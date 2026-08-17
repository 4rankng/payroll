package events

import (
	"encoding/json"
	"fmt"
	"sort"

	"api-server/internal/domain"
)

// EventRegistry maps event type names to deserialization functions
type EventRegistry struct {
	deserializers map[string]func([]byte) (domain.DomainEvent, error)
}

// NewEventRegistry creates a new event registry with all known event types
func NewEventRegistry() *EventRegistry {
	registry := &EventRegistry{
		deserializers: make(map[string]func([]byte) (domain.DomainEvent, error)),
	}

	// Register all event types
	registry.registerAllEvents()

	return registry
}

// Deserialize deserializes event data based on event type
func (r *EventRegistry) Deserialize(eventType string, data []byte) (domain.DomainEvent, error) {
	deserializer, exists := r.deserializers[eventType]
	if !exists {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	return deserializer(data)
}

// registerAllEvents registers all domain event types
func (r *EventRegistry) registerAllEvents() {
	// User events
	r.register("UserCreated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.UserCreatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("UserUpdated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.UserUpdatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("UserDeleted", func(data []byte) (domain.DomainEvent, error) {
		var event domain.UserDeletedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("PasswordChanged", func(data []byte) (domain.DomainEvent, error) {
		var event domain.PasswordChangedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	// Employee events
	r.register("EmployeeCreated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.EmployeeCreatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("EmployeeUpdated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.EmployeeUpdatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("EmployeeDeleted", func(data []byte) (domain.DomainEvent, error) {
		var event domain.EmployeeDeletedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("EmployeeProjectAssignmentsRemoved", func(data []byte) (domain.DomainEvent, error) {
		var event domain.EmployeeProjectAssignmentsRemovedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("EmployeeProfileUpdated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.EmployeeProfileUpdatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	// Project events
	r.register("ProjectCreated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.ProjectCreatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("ProjectUpdated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.ProjectUpdatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("ProjectDeleted", func(data []byte) (domain.DomainEvent, error) {
		var event domain.ProjectDeletedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	// ProjectEmployee events
	r.register("ProjectEmployeeUpdated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.ProjectEmployeeUpdatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	// Timesheet events
	r.register("TimesheetCreated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.TimesheetCreatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("TimesheetUpdated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.TimesheetUpdatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("TimesheetDeleted", func(data []byte) (domain.DomainEvent, error) {
		var event domain.TimesheetDeletedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("TimesheetApproved", func(data []byte) (domain.DomainEvent, error) {
		var event domain.TimesheetApprovedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("TimesheetRejected", func(data []byte) (domain.DomainEvent, error) {
		var event domain.TimesheetRejectedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("TimesheetBulkApproved", func(data []byte) (domain.DomainEvent, error) {
		var event domain.TimesheetBulkApprovedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("TimesheetBulkRejected", func(data []byte) (domain.DomainEvent, error) {
		var event domain.TimesheetBulkRejectedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("TimesheetBulkReset", func(data []byte) (domain.DomainEvent, error) {
		var event domain.TimesheetBulkResetEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("TimesheetBulkCreated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.TimesheetBulkCreatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	// Bank events
	r.register("BankCreated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.BankCreatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("BankUpdated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.BankUpdatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("BankDeleted", func(data []byte) (domain.DomainEvent, error) {
		var event domain.BankDeletedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	// Payrate events
	r.register("PayrateCreated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.PayrateCreatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("PayrateUpdated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.PayrateUpdatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("PayrateDeleted", func(data []byte) (domain.DomainEvent, error) {
		var event domain.PayrateDeletedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	// Transaction events
	r.register("TransactionCreated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.TransactionCreatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("TransactionUpdated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.TransactionUpdatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("TransactionDeleted", func(data []byte) (domain.DomainEvent, error) {
		var event domain.TransactionDeletedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("TransactionSettled", func(data []byte) (domain.DomainEvent, error) {
		var event domain.TransactionSettledEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	// Loan events
	r.register("LoanCreated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.LoanCreatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("LoanUpdated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.LoanUpdatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("LoanDeleted", func(data []byte) (domain.DomainEvent, error) {
		var event domain.LoanDeletedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("LoanApproved", func(data []byte) (domain.DomainEvent, error) {
		var event domain.LoanApprovedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("LoanRejected", func(data []byte) (domain.DomainEvent, error) {
		var event domain.LoanRejectedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("LoanDisbursed", func(data []byte) (domain.DomainEvent, error) {
		var event domain.LoanDisbursedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	// Asset events
	r.register("AssetCreated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.AssetCreatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("AssetDeleted", func(data []byte) (domain.DomainEvent, error) {
		var event domain.AssetDeletedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	// Lender events
	r.register("LenderCreated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.LenderCreatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("LenderUpdated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.LenderUpdatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("LenderDeleted", func(data []byte) (domain.DomainEvent, error) {
		var event domain.LenderDeletedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	// LedgerEntry events
	r.register("LedgerEntryCreated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.LedgerEntryCreatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("LedgerEntryUpdated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.LedgerEntryUpdatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("LedgerEntryDeleted", func(data []byte) (domain.DomainEvent, error) {
		var event domain.LedgerEntryDeletedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	// Settings events
	r.register("SettingsCreated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.SettingsCreatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("SettingsUpdated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.SettingsUpdatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("SettingsDeleted", func(data []byte) (domain.DomainEvent, error) {
		var event domain.SettingsDeletedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	// Import/Export events
	r.register("DataImported", func(data []byte) (domain.DomainEvent, error) {
		var event domain.DataImportedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("DataExported", func(data []byte) (domain.DomainEvent, error) {
		var event domain.DataExportedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	// Bulk transfer events
	r.register("BulkTransferFileExported", func(data []byte) (domain.DomainEvent, error) {
		var event domain.BulkTransferFileExportedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("BulkTransferResultImported", func(data []byte) (domain.DomainEvent, error) {
		var event domain.BulkTransferResultImportedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("BulkTransferResultParsed", func(data []byte) (domain.DomainEvent, error) {
		var event domain.BulkTransferResultParsedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("BulkTransferFileDownloaded", func(data []byte) (domain.DomainEvent, error) {
		var event domain.BulkTransferFileDownloadedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("PayrollHistoriesExported", func(data []byte) (domain.DomainEvent, error) {
		var event domain.PayrollHistoriesExportedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	// Settlement events
	r.register("SettlementCreated", func(data []byte) (domain.DomainEvent, error) {
		var event domain.SettlementCreatedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("SettlementUploadProcessed", func(data []byte) (domain.DomainEvent, error) {
		var event domain.SettlementUploadProcessedEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})

	r.register("TimesheetMarking", func(data []byte) (domain.DomainEvent, error) {
		var event domain.TimesheetMarkingEvent
		err := json.Unmarshal(data, &event)
		return event, err
	})
}

// register adds a deserializer for an event type
func (r *EventRegistry) register(eventType string, deserializer func([]byte) (domain.DomainEvent, error)) {
	r.deserializers[eventType] = deserializer
}

// EventTypes returns a sorted list of registered event type names.
func (r *EventRegistry) EventTypes() []string {
	eventTypes := make([]string, 0, len(r.deserializers))
	for eventType := range r.deserializers {
		eventTypes = append(eventTypes, eventType)
	}
	sort.Strings(eventTypes)
	return eventTypes
}

// HasEventType checks if an event type is registered
func (r *EventRegistry) HasEventType(eventType string) bool {
	_, exists := r.deserializers[eventType]
	return exists
}
