package services

import (
	"context"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// GetTimesheetSummaryStats calculates timesheet summary statistics with business logic
func (s *TimesheetDomainService) GetTimesheetSummaryStats(ctx context.Context, filters domain.TimesheetFilters) (*domain.TimesheetSummaryStats, error) {
	// Delegate to repository but apply business logic for interpretation
	stats, err := s.timesheetRepo.GetSummaryStats(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Apply business rules for stats calculation
	if stats.TotalEntries == 0 {
		stats.LastUpdated = clock.Now()
	}

	return stats, nil
}

// GetProjectTimesheetSummary calculates project-specific timesheet summary
func (s *TimesheetDomainService) GetProjectTimesheetSummary(ctx context.Context, projectID uint, fromDate, toDate time.Time) (*domain.TimesheetSummary, error) {
	// Validate date range
	if fromDate.After(toDate) {
		return nil, domain.NewValidationError(constants.MsgInvalidDateRangeVN)
	}

	// Validate date range is not too large (business rule: max 1 year)
	if toDate.Sub(fromDate) > 365*24*time.Hour {
		return nil, domain.NewValidationError(constants.MsgDateRangeTooLargeVN)
	}

	// Delegate to repository for data retrieval
	return s.timesheetRepo.GetSummaryByProject(ctx, projectID, fromDate, toDate)
}
