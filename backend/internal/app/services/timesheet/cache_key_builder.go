package timesheet

import (
	"fmt"
	"sort"
	"strings"

	"api-server/internal/domain"
)

// CacheKeyBuilder builds cache keys for timesheet queries using strategy pattern
type CacheKeyBuilder interface {
	Build(filters domain.TimesheetFilters) string
	Invalidate(key string) error
}

// FilterHandler handles individual filter components for cache key building
type FilterHandler interface {
	CanHandle(filters domain.TimesheetFilters) bool
	Apply(builder *strings.Builder, filters domain.TimesheetFilters)
}

// TimesheetCacheKeyBuilder implements cache key building with strategy pattern
type TimesheetCacheKeyBuilder struct {
	prefix   string
	handlers []FilterHandler
}

// NewTimesheetCacheKeyBuilder creates a new cache key builder
func NewTimesheetCacheKeyBuilder() *TimesheetCacheKeyBuilder {
	return &TimesheetCacheKeyBuilder{
		prefix: "timesheet",
		handlers: []FilterHandler{
			&ProjectIDFilterHandler{},
			&EmployeeIDFilterHandler{},
			&StatusFilterHandler{},
			&PaymentStatusFilterHandler{},
			&PayTypeFilterHandler{},
			&DateFilterHandler{},
		},
	}
}

// Build builds a cache key from filters
func (b *TimesheetCacheKeyBuilder) Build(filters domain.TimesheetFilters) string {
	var builder strings.Builder
	builder.WriteString(b.prefix)

	for _, handler := range b.handlers {
		handler.Apply(&builder, filters)
	}

	return builder.String()
}

// Invalidate is a placeholder for cache invalidation logic
func (b *TimesheetCacheKeyBuilder) Invalidate(key string) error {
	// Cache invalidation would be handled by cache service
	return nil
}

// ProjectIDFilterHandler handles project ID filter component
type ProjectIDFilterHandler struct{}

func (h *ProjectIDFilterHandler) CanHandle(filters domain.TimesheetFilters) bool {
	return len(filters.ProjectIDs) > 0
}

func (h *ProjectIDFilterHandler) Apply(builder *strings.Builder, filters domain.TimesheetFilters) {
	if len(filters.ProjectIDs) > 0 {
		projectIDs := make([]uint, len(filters.ProjectIDs))
		copy(projectIDs, filters.ProjectIDs)
		sort.Slice(projectIDs, func(i, j int) bool { return projectIDs[i] < projectIDs[j] })

		builder.WriteString(":projects:")
		for i, id := range projectIDs {
			if i > 0 {
				builder.WriteString(",")
			}
			fmt.Fprintf(builder, "%d", id)
		}
	}
}

// EmployeeIDFilterHandler handles employee ID filter component
type EmployeeIDFilterHandler struct{}

func (h *EmployeeIDFilterHandler) CanHandle(filters domain.TimesheetFilters) bool {
	return filters.EmployeeID != nil
}

func (h *EmployeeIDFilterHandler) Apply(builder *strings.Builder, filters domain.TimesheetFilters) {
	if filters.EmployeeID != nil {
		fmt.Fprintf(builder, ":employee:%d", *filters.EmployeeID)
	}
}

// StatusFilterHandler handles status filter component
type StatusFilterHandler struct{}

func (h *StatusFilterHandler) CanHandle(filters domain.TimesheetFilters) bool {
	return len(filters.TimesheetStatus) > 0
}

func (h *StatusFilterHandler) Apply(builder *strings.Builder, filters domain.TimesheetFilters) {
	if len(filters.TimesheetStatus) > 0 {
		builder.WriteString(":status:")
		for i, status := range filters.TimesheetStatus {
			if i > 0 {
				builder.WriteString(",")
			}
			builder.WriteString(string(status))
		}
	}
}

// PaymentStatusFilterHandler handles payment status filter component
type PaymentStatusFilterHandler struct{}

func (h *PaymentStatusFilterHandler) CanHandle(filters domain.TimesheetFilters) bool {
	return len(filters.PaymentStatus) > 0
}

func (h *PaymentStatusFilterHandler) Apply(builder *strings.Builder, filters domain.TimesheetFilters) {
	if len(filters.PaymentStatus) > 0 {
		builder.WriteString(":payment_status:")
		for i, status := range filters.PaymentStatus {
			if i > 0 {
				builder.WriteString(",")
			}
			builder.WriteString(string(status))
		}
	}
}

// PayTypeFilterHandler handles pay type filter component
type PayTypeFilterHandler struct{}

func (h *PayTypeFilterHandler) CanHandle(filters domain.TimesheetFilters) bool {
	return len(filters.PayType) > 0
}

func (h *PayTypeFilterHandler) Apply(builder *strings.Builder, filters domain.TimesheetFilters) {
	if len(filters.PayType) > 0 {
		builder.WriteString(":paytype:")
		for i, payType := range filters.PayType {
			if i > 0 {
				builder.WriteString(",")
			}
			builder.WriteString(payType)
		}
	}
}

// DateFilterHandler handles date range filter component
type DateFilterHandler struct{}

func (h *DateFilterHandler) CanHandle(filters domain.TimesheetFilters) bool {
	return filters.FromDate != nil || filters.ToDate != nil || filters.Date != nil
}

func (h *DateFilterHandler) Apply(builder *strings.Builder, filters domain.TimesheetFilters) {
	if filters.Date != nil {
		fmt.Fprintf(builder, ":date:%s", filters.Date.Format("2006-01-02"))
	} else {
		if filters.FromDate != nil {
			fmt.Fprintf(builder, ":from:%s", filters.FromDate.Format("2006-01-02"))
		}
		if filters.ToDate != nil {
			fmt.Fprintf(builder, ":to:%s", filters.ToDate.Format("2006-01-02"))
		}
	}
}
