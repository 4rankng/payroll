package events

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

type CashForecastAccuracyHandler struct {
	resolver domain.CashForecastOutcomeResolver
}

func NewCashForecastAccuracyHandler(resolver domain.CashForecastOutcomeResolver) *CashForecastAccuracyHandler {
	return &CashForecastAccuracyHandler{resolver: resolver}
}

func (h *CashForecastAccuracyHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	exported, ok := event.(domain.BulkTransferFileExportedEvent)
	if !ok {
		if ptr, ptrOK := event.(*domain.BulkTransferFileExportedEvent); ptrOK && ptr != nil {
			exported = *ptr
		} else {
			return nil
		}
	}
	if exported.Cycle != "weekly" || !exported.CompanyWide || len(exported.ForecastOutcomeItems) == 0 {
		return nil
	}
	if h.resolver == nil {
		return fmt.Errorf("cash forecast outcome resolver is nil")
	}

	fromDate, err := time.ParseInLocation("2006-01-02", exported.FromDate, clock.DefaultLocation)
	if err != nil {
		return fmt.Errorf("parse weekly export from date %q: %w", exported.FromDate, err)
	}
	toDate, err := time.ParseInLocation("2006-01-02", exported.ToDate, clock.DefaultLocation)
	if err != nil {
		return fmt.Errorf("parse weekly export to date %q: %w", exported.ToDate, err)
	}
	if fromDate.After(toDate) {
		return fmt.Errorf("weekly export date range is inverted: %s to %s", exported.FromDate, exported.ToDate)
	}

	_, err = h.resolver.ResolveWeeklyExport(
		ctx,
		fromDate,
		toDate,
		exported.ForecastOutcomeItems,
		domain.CashForecastOutcomeWeeklyExportDistinctTimesheets,
	)
	if err != nil {
		return fmt.Errorf("resolve cash forecast from weekly export: %w", err)
	}
	return nil
}

func (h *CashForecastAccuracyHandler) CanHandle(eventType string) bool {
	return eventType == "BulkTransferFileExported"
}

var _ domain.EventHandler = (*CashForecastAccuracyHandler)(nil)
