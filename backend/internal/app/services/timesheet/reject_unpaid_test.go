package timesheet

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"api-server/internal/domain"
)

type failingRejectUnpaidCache struct{}

func (failingRejectUnpaidCache) Get(context.Context, string, interface{}) error { return nil }
func (failingRejectUnpaidCache) Set(context.Context, string, interface{}, time.Duration) error {
	return nil
}
func (failingRejectUnpaidCache) Delete(context.Context, string) error { return nil }
func (failingRejectUnpaidCache) InvalidatePattern(context.Context, string) error {
	return errors.New("cache unavailable")
}
func (failingRejectUnpaidCache) Exists(context.Context, string) (bool, error) { return false, nil }

type failingRejectUnpaidEventBus struct{}

func (failingRejectUnpaidEventBus) Publish(context.Context, ...domain.DomainEvent) error {
	return errors.New("event unavailable")
}
func (failingRejectUnpaidEventBus) Subscribe(string, domain.EventHandler) {}
func (failingRejectUnpaidEventBus) SubscribeAll(domain.EventHandler)      {}

func TestDeliverRejectUnpaidPostCommitReturnsCommittedSideEffectWarnings(t *testing.T) {
	service := &TimesheetService{
		cache:  failingRejectUnpaidCache{},
		events: failingRejectUnpaidEventBus{},
		logger: slog.Default(),
	}
	projectID := uint(12)
	event := domain.NewTimesheetBulkRejectedEvent(context.Background(), 4, &projectID, 9)

	result := service.deliverRejectUnpaidPostCommit(event, 4, projectID)
	if result.RejectedCount != 4 {
		t.Fatalf("rejected count = %d, want 4", result.RejectedCount)
	}
	if result.PostCommitComplete {
		t.Fatal("post-commit outcome unexpectedly complete")
	}
	if len(result.Warnings) != 2 {
		t.Fatalf("warnings = %v, want one cache and one audit warning", result.Warnings)
	}
}
