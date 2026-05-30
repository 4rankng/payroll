package events

import "testing"

func TestAuditEventHandler_CanHandleAllRegisteredEventTypes(t *testing.T) {
	registry := NewEventRegistry()
	handler := NewAuditEventHandler(nil)

	for _, eventType := range registry.EventTypes() {
		t.Run(eventType, func(t *testing.T) {
			if !handler.CanHandle(eventType) {
				t.Fatalf("event type %s is not handled by AuditEventHandler", eventType)
			}
		})
	}
}
