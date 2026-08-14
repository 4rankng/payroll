package events

import (
	"context"
	"sync"
	"testing"
	"time"

	"api-server/internal/domain"
)

// mockCacheInvalidator is a minimal test double for CacheInvalidator that
// records patterns and signals when it is called. It is safe for concurrent
// use: the handler invalidates asynchronously, so the mock is written from a
// separate goroutine while tests read it.
type mockCacheInvalidator struct {
	mu             sync.Mutex
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
	m.mu.Lock()
	m.calledPatterns = append(m.calledPatterns, pattern)
	m.mu.Unlock()
	// Non-blocking signal for tests
	select {
	case m.called <- struct{}{}:
	default:
	}
	return nil
}

// patterns returns a copy of the recorded patterns for race-free inspection.
func (m *mockCacheInvalidator) patterns() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.calledPatterns...)
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
	for _, p := range mockCache.patterns() {
		if p == "dashboard:*" {
			foundDashboardPattern = true
			break
		}
	}

	if !foundDashboardPattern {
		t.Fatalf("expected dashboard:* cache pattern to be invalidated, got %v", mockCache.patterns())
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
	for _, p := range mockCache.patterns() {
		if p == "dashboard:*" {
			foundDashboardPattern = true
			break
		}
	}

	if !foundDashboardPattern {
		t.Fatalf("expected dashboard:* cache pattern to be invalidated, got %v", mockCache.patterns())
	}
}
