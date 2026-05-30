package events

import (
	"context"
	"log/slog"
	"strings"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// CacheInvalidator defines the interface for cache invalidation
type CacheInvalidator interface {
	DeletePattern(ctx context.Context, pattern string) error
}

// CacheInvalidationHandler listens to domain events and invalidates relevant caches
type CacheInvalidationHandler struct {
	cache  CacheInvalidator
	logger *slog.Logger
}

// NewCacheInvalidationHandler creates a new cache invalidation event handler
func NewCacheInvalidationHandler(cache CacheInvalidator) *CacheInvalidationHandler {
	return &CacheInvalidationHandler{
		cache:  cache,
		logger: observability.GetLogger(),
	}
}

// Handle processes a domain event and invalidates relevant caches asynchronously
func (h *CacheInvalidationHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	// Determine which cache patterns to invalidate based on event type
	patterns := h.getCachePatternsForEvent(event.EventType())

	if len(patterns) == 0 {
		return nil
	}

	// Invalidate cache patterns asynchronously
	go func() {
		for _, pattern := range patterns {
			if err := h.cache.DeletePattern(context.Background(), pattern); err != nil {
				h.logger.Warn("Failed to invalidate cache pattern",
					"pattern", pattern,
					"event_type", event.EventType(),
					"error", err,
				)
			}
		}
	}()

	return nil
}

// CanHandle returns true if this handler can process the given event type
func (h *CacheInvalidationHandler) CanHandle(eventType string) bool {
	// Handle events that affect cached data
	return strings.Contains(eventType, "Bank") ||
		strings.Contains(eventType, "Settings") ||
		strings.Contains(eventType, "Timesheet") ||
		strings.Contains(eventType, "Payrate") ||
		strings.Contains(eventType, "Ledger") ||
		strings.Contains(eventType, "Transaction") ||
		strings.Contains(eventType, "Project") ||
		strings.Contains(eventType, "Employee") ||
		strings.Contains(eventType, "Loan") ||
		strings.Contains(eventType, "Lender") ||
		strings.Contains(eventType, "Asset") ||
		strings.Contains(eventType, "Settlement")
}

// getCachePatternsForEvent maps event types to cache patterns that need invalidation
func (h *CacheInvalidationHandler) getCachePatternsForEvent(eventType string) []string {
	var patterns []string

	switch {
	// Project-Employee events impact both project- and employee-based dashboards
	case strings.Contains(eventType, "ProjectEmployee"):
		patterns = append(patterns,
			"projects:list*",
			"employees:*",
			"dashboard:project_summary*",
			"dashboard:partner_project_summary*",
			"dashboard:employee*",
			"dashboard:*",
		)

	// Bank events invalidate all bank-related caches
	case strings.Contains(eventType, "Bank"):
		patterns = append(patterns,
			"banks:list:*",
			"banks:search:*",
			"banks:detail:*",
		)

	// Settings events invalidate settings cache
	case strings.Contains(eventType, "Settings"):
		patterns = append(patterns,
			"settings:list:*",
			"settings:active:*",
			"settings:detail:*",
		)

	// Timesheet events invalidate summary stats, list cache and dashboard cache
	case strings.Contains(eventType, "Timesheet"):
		patterns = append(patterns,
			"timesheets:summary:*",
			"timesheets:list:*",
			"dashboard:*",
		)

	// Ledger events invalidate dashboard financial cache and ledger summaries
	case strings.Contains(eventType, "Ledger"):
		patterns = append(patterns, "dashboard:*")

	// Payrate events affect payroll calculations and timesheet-derived dashboards
	case strings.Contains(eventType, "Payrate"):
		patterns = append(patterns,
			"timesheets:summary:*",
			"timesheets:list:*",
			"dashboard:*",
		)

	// Transaction events invalidate transaction caches and dashboard cache
	case strings.Contains(eventType, "Transaction"):
		patterns = append(patterns,
			"transactions:list:*",
			"transactions:detail:*",
			"transactions:pending:*",
			"dashboard:*",
		)

	// Project events invalidate project list and dashboard caches
	case strings.Contains(eventType, "Project"):
		patterns = append(patterns,
			"projects:list*",
			"dashboard:project_summary*",
			"dashboard:partner_project_summary*",
			"dashboard:*",
		)

	// Employee events invalidate employee list and dashboard caches
	case strings.Contains(eventType, "Employee"):
		patterns = append(patterns,
			// Employee list microcache
			"employees:*",
			// Employee summary caches
			"dashboard:employee*",
			"dashboard:employees_summary*",
			"dashboard:employees_summary_creator*",
			// Any other dashboard slices that depend on employees
			"dashboard:*",
		)

	// Loan events invalidate loan-related caches
	case strings.Contains(eventType, "Loan"):
		patterns = append(patterns,
			"loan:*",
			"loans:*",
		)

	// Lender events invalidate lender-related caches
	case strings.Contains(eventType, "Lender"):
		patterns = append(patterns,
			"lender:*",
			"lenders:*",
		)

	// Asset events (e.g., uploaded files) may be cached in the future
	case strings.Contains(eventType, "Asset"):
		patterns = append(patterns, "assets:*")

	// Settlement events invalidate dashboard financial summaries
	case strings.Contains(eventType, "Settlement"):
		patterns = append(patterns,
			"dashboard:*",
			"transactions:list:*",
			"transactions:pending:*",
		)
	}

	return patterns
}
