package common

import (
	"fmt"
	"strings"
	"time"

	"api-server/internal/pkg/utils"

	"gorm.io/gorm"
)

// DefaultMaxResults is the maximum number of records returned by list-all queries without explicit limits.
const DefaultMaxResults = 5000

// FilterBuilder provides reusable filter application logic for repositories.
// It eliminates code duplication by providing composable filter functions
// that can be shared across different repositories.
type FilterBuilder struct {
	db *gorm.DB
}

// NewFilterBuilder creates a new FilterBuilder instance.
func NewFilterBuilder(db *gorm.DB) *FilterBuilder {
	return &FilterBuilder{db: db}
}

// StatusFilter defines the interface for filters that apply to status fields.
type StatusFilter interface {
	GetStatuses() []string
	GetStatusField() string
}

// ApplyStatus applies a status filter to the query.
// If the filter provides no statuses, the query is returned unchanged.
func (fb *FilterBuilder) ApplyStatus(query *gorm.DB, filter StatusFilter) *gorm.DB {
	if filter == nil {
		return query
	}

	statuses := filter.GetStatuses()
	if len(statuses) > 0 {
		field := filter.GetStatusField()
		query = query.Where(fmt.Sprintf("%s IN ?", field), statuses)
	}

	return query
}

// DateRangeFilter defines the interface for filters that apply to date ranges.
type DateRangeFilter interface {
	GetFromDate() *time.Time
	GetToDate() *time.Time
	GetDateField() string
}

// CalendarDateRangeFilter marks filters whose database column is SQL DATE,
// not DATETIME/TIMESTAMP. DATE values have no timezone and must be bound as
// YYYY-MM-DD strings so the MySQL driver's connection location cannot shift a
// boundary across (or into) a calendar day.
type CalendarDateRangeFilter interface {
	UsesCalendarDates() bool
}

// ApplyDateRange applies a date range filter to the query using a half-open interval
// [fromDate, toDate+1day). This avoids timezone boundary issues when comparing
// DATE columns against time.Time values that may shift across midnight due to
// Go/MySQL timezone mismatch.
func (fb *FilterBuilder) ApplyDateRange(query *gorm.DB, filter DateRangeFilter) *gorm.DB {
	if filter == nil {
		return query
	}
	calendarDates := false
	if typed, ok := filter.(CalendarDateRangeFilter); ok {
		calendarDates = typed.UsesCalendarDates()
	}

	if fromDate := filter.GetFromDate(); fromDate != nil {
		field := filter.GetDateField()
		if calendarDates {
			query = query.Where(fmt.Sprintf("%s >= ?", field), formatCalendarDate(*fromDate))
		} else {
			query = query.Where(fmt.Sprintf("%s >= ?", field), *fromDate)
		}
	}

	if toDate := filter.GetToDate(); toDate != nil {
		field := filter.GetDateField()
		if calendarDates {
			query = query.Where(fmt.Sprintf("%s < ?", field), formatNextCalendarDate(*toDate))
		} else {
			nextDay := toDate.Add(24 * time.Hour)
			query = query.Where(fmt.Sprintf("%s < ?", field), nextDay)
		}
	}

	return query
}

func formatCalendarDate(value time.Time) string {
	return fmt.Sprintf("%04d-%02d-%02d", value.Year(), value.Month(), value.Day())
}

func formatNextCalendarDate(value time.Time) string {
	dateOnly := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	return dateOnly.AddDate(0, 0, 1).Format("2006-01-02")
}

// CreatorFilter defines the interface for filters that apply to creator/user fields.
type CreatorFilter interface {
	GetCreatedBy() *uint
	GetCreatorField() string
}

// ApplyCreator applies a creator filter to the query.
// Used for filtering by the user who created a record.
func (fb *FilterBuilder) ApplyCreator(query *gorm.DB, filter CreatorFilter) *gorm.DB {
	if filter == nil {
		return query
	}

	if createdBy := filter.GetCreatedBy(); createdBy != nil {
		field := filter.GetCreatorField()
		query = query.Where(fmt.Sprintf("%s = ?", field), *createdBy)
	}

	return query
}

// SearchFilter defines the interface for filters that apply text search.
type SearchFilter interface {
	GetSearch() string
	GetSearchFields() []string
}

// ApplyVietnameseSearch applies a text search filter with Vietnamese normalization.
// It normalizes the search text and applies it to all specified search fields using OR.
// This is essential for proper Vietnamese text search functionality.
func (fb *FilterBuilder) ApplyVietnameseSearch(query *gorm.DB, filter SearchFilter) *gorm.DB {
	if filter == nil {
		return query
	}

	search := filter.GetSearch()
	if search == "" {
		return query
	}

	// Normalize Vietnamese text for search
	normalizedSearch := utils.NormalizeVietnameseForSearch(search)
	fields := filter.GetSearchFields()

	if len(fields) == 0 {
		return query
	}

	// Build combined OR conditions as a single WHERE clause to preserve AND grouping
	conditions := make([]string, len(fields))
	values := make([]interface{}, len(fields))
	for i, field := range fields {
		conditions[i] = fmt.Sprintf("%s LIKE ?", field)
		values[i] = normalizedSearch
	}

	combined := "(" + strings.Join(conditions, " OR ") + ")"
	return query.Where(combined, values...)
}

// ApplyPagination applies limit and offset to the query.
// If limit is 0 or negative, no limit is applied.
func (fb *FilterBuilder) ApplyPagination(query *gorm.DB, limit, offset int) *gorm.DB {
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	return query
}

// ApplySorting applies sorting to the query.
// If sortBy is empty or unsafe, defaultSort is used.
// If sortOrder is empty or unsafe, "DESC" is used as default.
//
// Both fields are sanitized before interpolation: sortBy is constrained to a
// safe SQL-identifier shape and sortOrder to exactly ASC/DESC. This is the
// SQL-injection boundary for ORDER BY clauses (column names cannot be
// parameterized), so all callers of this method inherit injection-safe sorting.
func (fb *FilterBuilder) ApplySorting(query *gorm.DB, sortBy, sortOrder, defaultSort string) *gorm.DB {
	sortBy = SanitizeSortColumn(sortBy, defaultSort)
	sortOrder = SanitizeSortOrder(sortOrder, "DESC")
	return query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))
}
