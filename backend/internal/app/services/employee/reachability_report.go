package employee

import (
	"context"
	"fmt"
	"strings"

	"api-server/internal/constants"
	"api-server/internal/domain"
)

// Pagination bounds for the reachability report. The cohort is operator-facing
// and small, but the cap stops a hand-written ?limit= from pulling the whole
// employee table into one response.
const (
	defaultReachabilityReportLimit = 50
	maxReachabilityReportLimit     = 200
)

// GetUnreachableEmployeesReport builds the contact-reachability report: which
// employees ops cannot reach on their mobile and what payroll exposure hangs off
// them.
//
// It is READ-ONLY by design. The project decision is that mobile values are not
// validated or normalised on write (a CCCD-shaped value is the one exception),
// so this report surfaces bad contact data for humans to fix instead of
// rejecting it at input.
//
// Pagination applies to the reported rows only. The Summary always describes the
// whole project-scoped cohort, so a filtered page is never mistaken for the size
// of the worklist.
func (s *EmployeeService) GetUnreachableEmployeesReport(ctx context.Context, filters domain.EmployeeReachabilityFilters) (*domain.UnreachableEmployeesReport, error) {
	if filters.State != "" && !domain.IsEmployeeReachabilityState(filters.State) {
		return nil, domain.NewValidationError(fmt.Sprintf(
			constants.MsgInvalidEmployeeReachabilityStateFilterVN,
			strings.Join(domain.EmployeeReachabilityStates(), ", "),
		))
	}

	filters.Limit, filters.Offset = normalizeReachabilityPagination(filters.Limit, filters.Offset)

	employees, err := s.EmployeeRepo.ListEmployeeReachability(ctx, filters)
	if err != nil {
		return nil, err
	}
	// Counters size the whole project-scoped worklist, so the state filter is
	// dropped for the counting call rather than left to the repository to ignore.
	cohortFilters := filters
	cohortFilters.State = ""
	summary, err := s.EmployeeRepo.CountEmployeeReachability(ctx, cohortFilters)
	if err != nil {
		return nil, err
	}

	return &domain.UnreachableEmployeesReport{
		Employees: employees,
		Summary:   *summary,
		// CountForState returns the distinct cohort size for an empty filter,
		// otherwise the counter of the requested state. Per-state counters
		// overlap, so they must never be summed to size a filtered page.
		Total:  summary.CountForState(filters.State),
		Limit:  filters.Limit,
		Offset: filters.Offset,
	}, nil
}

func normalizeReachabilityPagination(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = defaultReachabilityReportLimit
	}
	if limit > maxReachabilityReportLimit {
		limit = maxReachabilityReportLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
