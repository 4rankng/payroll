package events

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"
)

// mockCacheInvalidator is a minimal test double for CacheInvalidator that
// records patterns and signals when it is called.
type mockCacheInvalidator struct {
	calledPatterns []string
	called         chan struct{}
}

func newMockCacheInvalidator() *mockCacheInvalidator {
	return &mockCacheInvalidator{
		calledPatterns: make([]string, 0),
		called:         make(chan struct{}, 10),
	}
}

func (m *mockCacheInvalidator) DeletePattern(_ context.Context, pattern string) error {
	m.calledPatterns = append(m.calledPatterns, pattern)
	// Non-blocking signal for tests
	select {
	case m.called <- struct{}{}:
	default:
	}
	return nil
}

func TestCacheInvalidationHandler_HandlesLedgerEvents(t *testing.T) {
	mockCache := newMockCacheInvalidator()
	handler := NewCacheInvalidationHandler(mockCache)

	event := domain.LedgerEntryCreatedEvent{
		BaseEvent: domain.BaseEvent{
			EventName: "LedgerEntryCreated",
		},
	}

	if !handler.CanHandle(event.EventType()) {
		t.Fatalf("expected handler to be able to handle Ledger events")
	}

	if err := handler.Handle(context.Background(), event); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Handler invalidates cache asynchronously; wait briefly for it to run.
	select {
	case <-mockCache.called:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected cache invalidation to be triggered for Ledger events")
	}

	foundDashboardPattern := false
	for _, p := range mockCache.calledPatterns {
		if p == "dashboard:*" {
			foundDashboardPattern = true
			break
		}
	}

	if !foundDashboardPattern {
		t.Fatalf("expected dashboard:* cache pattern to be invalidated, got %v", mockCache.calledPatterns)
	}
}

func TestCacheInvalidationHandler_HandlesTimesheetEvents(t *testing.T) {
	mockCache := newMockCacheInvalidator()
	handler := NewCacheInvalidationHandler(mockCache)

	event := domain.TimesheetUpdatedEvent{
		BaseEvent: domain.BaseEvent{
			EventName: "TimesheetUpdated",
		},
	}

	if !handler.CanHandle(event.EventType()) {
		t.Fatalf("expected handler to be able to handle Timesheet events")
	}

	if err := handler.Handle(context.Background(), event); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Handler invalidates cache asynchronously; wait briefly for it to run.
	select {
	case <-mockCache.called:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected cache invalidation to be triggered for Timesheet events")
	}

	foundDashboardPattern := false
	for _, p := range mockCache.calledPatterns {
		if p == "dashboard:*" {
			foundDashboardPattern = true
			break
		}
	}

	if !foundDashboardPattern {
		t.Fatalf("expected dashboard:* cache pattern to be invalidated, got %v", mockCache.calledPatterns)
	}
}
