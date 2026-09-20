package employee

import (
	"context"
	"testing"

	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
)

// The report is an operator worklist, so a mistyped state must fail loudly
// instead of returning the whole employee table, the page size must be bounded
// by the service rather than trusted from the request, and the total must
// describe the filtered cohort rather than the page or the sum of the
// (deliberately overlapping) per-state counters.
type reachabilityStubRepo struct {
	domain.EmployeeRepository
	listCalls    int
	listFilters  domain.EmployeeReachabilityFilters
	countFilters domain.EmployeeReachabilityFilters
	rows         []*domain.EmployeeReachability
	summary      *domain.EmployeeReachabilitySummary
}

func (r *reachabilityStubRepo) ListEmployeeReachability(_ context.Context, filters domain.EmployeeReachabilityFilters) ([]*domain.EmployeeReachability, error) {
	r.listCalls++
	r.listFilters = filters
	return r.rows, nil
}

func (r *reachabilityStubRepo) CountEmployeeReachability(_ context.Context, filters domain.EmployeeReachabilityFilters) (*domain.EmployeeReachabilitySummary, error) {
	r.countFilters = filters
	return r.summary, nil
}

func TestGetUnreachableEmployeesReport(t *testing.T) {
	summary := &domain.EmployeeReachabilitySummary{
		NoPhone: 588, NotLinked: 23, InvalidMobile: 20, DuplicateMobile: 43, Employees: 672,
	}
	newRepo := func() *reachabilityStubRepo {
		return &reachabilityStubRepo{
			summary: summary,
			rows: []*domain.EmployeeReachability{{
				EmployeeID: 1,
				States:     []string{domain.EmployeeReachabilityStateNoPhone},
			}},
		}
	}

	t.Run("unknown state is rejected before the repository runs", func(t *testing.T) {
		repo := newRepo()
		service := &EmployeeService{EmployeeRepo: repo}

		report, err := service.GetUnreachableEmployeesReport(context.Background(), domain.EmployeeReachabilityFilters{
			State: "no_pone",
		})

		require.Error(t, err)
		require.True(t, domain.IsValidationError(err), "want validation error, got %v", err)
		require.Nil(t, report)
		require.Zero(t, repo.listCalls)
	})

	t.Run("pagination defaults and caps are enforced", func(t *testing.T) {
		repo := newRepo()
		service := &EmployeeService{EmployeeRepo: repo}

		report, err := service.GetUnreachableEmployeesReport(context.Background(), domain.EmployeeReachabilityFilters{Offset: -3})
		require.NoError(t, err)
		require.Equal(t, 50, report.Limit)
		require.Equal(t, 0, report.Offset)
		require.Equal(t, 50, repo.listFilters.Limit, "the repository must receive the effective page size")
		require.Zero(t, repo.listFilters.Offset)

		report, err = service.GetUnreachableEmployeesReport(context.Background(), domain.EmployeeReachabilityFilters{Limit: 5000, Offset: 10})
		require.NoError(t, err)
		require.Equal(t, 200, report.Limit, "a hand-written limit must not pull the whole employee table")
		require.Equal(t, 10, report.Offset)
		require.Equal(t, 200, repo.listFilters.Limit)
	})

	t.Run("total follows the filter while counters size the cohort", func(t *testing.T) {
		repo := newRepo()
		service := &EmployeeService{EmployeeRepo: repo}

		report, err := service.GetUnreachableEmployeesReport(context.Background(), domain.EmployeeReachabilityFilters{})
		require.NoError(t, err)
		require.Equal(t, int64(672), report.Total)
		require.Equal(t, summary, &report.Summary)
		require.Len(t, report.Employees, 1)

		report, err = service.GetUnreachableEmployeesReport(context.Background(), domain.EmployeeReachabilityFilters{
			State: domain.EmployeeReachabilityStateNotLinked,
		})
		require.NoError(t, err)
		require.Equal(t, int64(23), report.Total, "total is the size of the filtered cohort, not the page")
		require.Equal(t, domain.EmployeeReachabilityStateNotLinked, repo.listFilters.State)
		require.Empty(t, repo.countFilters.State, "counters are not narrowed by the state filter")
		require.Equal(t, summary, &report.Summary, "counters keep covering every state")
	})
}
