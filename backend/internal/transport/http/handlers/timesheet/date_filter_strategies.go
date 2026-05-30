package timesheet

import (
	"context"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/timeutil"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// DateFilterStrategy interface for date filtering strategies
type DateFilterStrategy interface {
	Apply(ctx context.Context, c *gin.Context, filters *domain.TimesheetFilters) error
	Name() string
}

// SpecificDateFilter handles specific date filtering
type SpecificDateFilter struct{}

func (f *SpecificDateFilter) Apply(ctx context.Context, c *gin.Context, filters *domain.TimesheetFilters) error {
	dateStr := c.Query("date")
	if dateStr == "" {
		return nil // Not applicable, try next strategy
	}

	date, err := time.Parse(timeutil.DateFormat, dateStr)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidDateFormatVN)
		return err
	}

	date = date.UTC().Truncate(24 * time.Hour)
	filters.Date = &date
	return nil
}

func (f *SpecificDateFilter) Name() string {
	return "specific_date"
}

// MonthFilter handles month-based filtering
type MonthFilter struct{}

func (f *MonthFilter) Apply(ctx context.Context, c *gin.Context, filters *domain.TimesheetFilters) error {
	monthStr := c.Query("month")
	if monthStr == "" {
		return nil
	}

	monthDate, err := time.Parse("2006-01", monthStr)
	if err != nil {
		response.BadRequest(c, "Invalid month format. Use YYYY-MM format")
		return err
	}

	fromDate := time.Date(monthDate.Year(), monthDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	toDate := fromDate.AddDate(0, 1, -1)
	filters.FromDate = &fromDate
	filters.ToDate = &toDate
	return nil
}

func (f *MonthFilter) Name() string {
	return "month"
}

// DateRangeFilter handles date range filtering
type DateRangeFilter struct{}

func (f *DateRangeFilter) Apply(ctx context.Context, c *gin.Context, filters *domain.TimesheetFilters) error {
	// Check if already handled by other strategies
	if filters.Date != nil || (filters.FromDate != nil && filters.ToDate != nil) {
		return nil
	}

	// Parse fromDate
	if fromDateStr := c.Query("fromDate"); fromDateStr != "" {
		fromDate, err := time.Parse(timeutil.DateFormat, fromDateStr)
		if err != nil {
			response.BadRequest(c, constants.MsgInvalidFromDateFormatVN)
			return err
		}
		fromDate = fromDate.UTC().Truncate(24 * time.Hour)
		filters.FromDate = &fromDate
	}

	// Parse toDate
	if toDateStr := c.Query("toDate"); toDateStr != "" {
		toDate, err := time.Parse(timeutil.DateFormat, toDateStr)
		if err != nil {
			response.BadRequest(c, constants.MsgInvalidToDateFormatVN)
			return err
		}
		toDate = toDate.UTC().Truncate(24 * time.Hour)
		filters.ToDate = &toDate
	}

	return nil
}

func (f *DateRangeFilter) Name() string {
	return "date_range"
}

// DateFilterProcessor chains date filter strategies
type DateFilterProcessor struct {
	strategies []DateFilterStrategy
}

// NewDateFilterProcessor creates a new date filter processor
func NewDateFilterProcessor() *DateFilterProcessor {
	return &DateFilterProcessor{
		strategies: []DateFilterStrategy{
			&SpecificDateFilter{},
			&MonthFilter{},
			&DateRangeFilter{},
		},
	}
}

// ApplyFilters applies all filter strategies in order
func (p *DateFilterProcessor) ApplyFilters(ctx context.Context, c *gin.Context, filters *domain.TimesheetFilters) error {
	for _, strategy := range p.strategies {
		if err := strategy.Apply(ctx, c, filters); err != nil {
			return err // First strategy that matches and fails
		}
	}
	return nil
}
