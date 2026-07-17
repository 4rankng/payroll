package events

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

type recordingCashForecastResolver struct {
	calls  int
	from   time.Time
	to     time.Time
	items  []domain.CashForecastOutcomeItem
	source string
	err    error
}

func (r *recordingCashForecastResolver) ResolveWeeklyExport(_ context.Context, fromDate, toDate time.Time, items []domain.CashForecastOutcomeItem, outcomeSource string) (int64, error) {
	r.calls++
	r.from = fromDate
	r.to = toDate
	r.items = items
	r.source = outcomeSource
	return 1, r.err
}

func TestCashForecastAccuracyHandlerResolvesWeeklyExportUsingLocalDates(t *testing.T) {
	resolver := &recordingCashForecastResolver{}
	handler := NewCashForecastAccuracyHandler(resolver)
	event := domain.NewBulkTransferFileExportedEvent(
		context.Background(), 1, "weekly.xlsx", "weekly",
		"2026-07-01", "2026-07-07", "", 4, 12_345_000,
		true,
		[]domain.CashForecastOutcomeItem{{TimesheetID: 7, Amount: 12_345_000}},
	)

	if err := handler.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if resolver.calls != 1 {
		t.Fatalf("resolver calls = %d, want 1", resolver.calls)
	}
	if resolver.from.Location() != clock.DefaultLocation || resolver.to.Location() != clock.DefaultLocation {
		t.Fatalf("dates must use business location: from=%v to=%v", resolver.from.Location(), resolver.to.Location())
	}
	if got := resolver.from.Format("2006-01-02"); got != "2026-07-01" {
		t.Fatalf("from date = %s", got)
	}
	if got := resolver.to.Format("2006-01-02"); got != "2026-07-07" {
		t.Fatalf("to date = %s", got)
	}
	if len(resolver.items) != 1 || resolver.items[0].Amount != 12_345_000 {
		t.Fatalf("items = %#v", resolver.items)
	}
	if resolver.source != domain.CashForecastOutcomeWeeklyExportDistinctTimesheets {
		t.Fatalf("source = %q", resolver.source)
	}
}

func TestCashForecastAccuracyHandlerFiltersNonWeeklyAndInvalidDates(t *testing.T) {
	resolver := &recordingCashForecastResolver{}
	handler := NewCashForecastAccuracyHandler(resolver)

	monthly := domain.NewBulkTransferFileExportedEvent(
		context.Background(), 1, "monthly.xlsx", "monthly",
		"not-a-date", "also-not-a-date", "2026-07", 1, 10,
		true,
		nil,
	)
	if err := handler.Handle(context.Background(), monthly); err != nil {
		t.Fatalf("monthly event should be ignored, got %v", err)
	}
	if resolver.calls != 0 {
		t.Fatalf("resolver called for monthly event")
	}

	weekly := domain.NewBulkTransferFileExportedEvent(
		context.Background(), 1, "weekly.xlsx", "weekly",
		"2026-02-30", "2026-03-07", "", 1, 10,
		true,
		[]domain.CashForecastOutcomeItem{{TimesheetID: 1, Amount: 10}},
	)
	if err := handler.Handle(context.Background(), &weekly); err == nil {
		t.Fatal("invalid weekly local date should fail")
	}
	if resolver.calls != 0 {
		t.Fatalf("resolver called for invalid weekly event")
	}
}

func TestCashForecastAccuracyHandlerIgnoresScopedWeeklyExport(t *testing.T) {
	resolver := &recordingCashForecastResolver{}
	handler := NewCashForecastAccuracyHandler(resolver)
	event := domain.NewBulkTransferFileExportedEvent(
		context.Background(), 1, "filtered.xlsx", "weekly",
		"2026-07-01", "2026-07-07", "", 1, 10, false,
		[]domain.CashForecastOutcomeItem{{TimesheetID: 1, Amount: 10}},
	)
	if err := handler.Handle(context.Background(), event); err != nil {
		t.Fatalf("scoped weekly event should be ignored: %v", err)
	}
	if resolver.calls != 0 {
		t.Fatalf("resolver called %d times for scoped export", resolver.calls)
	}
}

func TestCashForecastAccuracyHandlerCanHandleOnlyBulkTransferExport(t *testing.T) {
	handler := NewCashForecastAccuracyHandler(&recordingCashForecastResolver{})
	if !handler.CanHandle("BulkTransferFileExported") {
		t.Fatal("expected weekly export event type to be supported")
	}
	if handler.CanHandle("BulkTransferResultImported") {
		t.Fatal("unexpected non-export event support")
	}
}

var _ domain.CashForecastOutcomeResolver = (*recordingCashForecastResolver)(nil)
