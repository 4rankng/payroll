package project

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// AggregateRecomputeService keeps the four denormalised project financial totals
// (projects.total_payout_vnd, pending_payable_vnd, total_received_vnd,
// pending_receivable_vnd) in sync with the timesheets that summarise them. The
// meaning of each total is documented on domain.Project.
//
// The columns are derived data and have no HTTP writer — the project create/update
// DTOs deliberately omit them — so every writer goes through this service:
//   - the bulk transfer worker after a payment cycle settles timesheets;
//   - RecomputeAllProjects as a one-shot backfill or scheduled reconciliation.
type AggregateRecomputeService struct {
	projects domain.ProjectRepository
	clock    clock.Clock
	logger   *slog.Logger
}

// NewAggregateRecomputeService wires the recompute service. A nil clock falls back
// to the business clock (Asia/Ho_Chi_Minh) and a nil logger to the default logger.
func NewAggregateRecomputeService(projects domain.ProjectRepository, clk clock.Clock, logger *slog.Logger) *AggregateRecomputeService {
	if clk == nil {
		clk = clock.New()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &AggregateRecomputeService{
		projects: projects,
		clock:    clk,
		logger:   logger.With("component", "AggregateRecomputeService"),
	}
}

// RecomputeProject recomputes one project's four totals from its timesheets and
// persists them in a single statement. It returns the freshly computed totals so
// callers can log or compare them.
//
// The call is idempotent: the values are a pure function of the timesheet rows, so
// repeating it writes the same numbers. Unknown or soft-deleted projects are a
// silent no-op (the UPDATE matches no row) — projects with nothing to report
// legitimately reconcile to all zeroes.
func (s *AggregateRecomputeService) RecomputeProject(ctx context.Context, projectID uint) (*domain.ProjectFinancialAggregate, error) {
	if s.projects == nil {
		return nil, errors.New("project repository is not configured")
	}

	aggregate, err := s.projects.GetProjectFinancialAggregate(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("recompute project %d financials: %w", projectID, err)
	}

	if err := s.projects.UpdateProjectFinancialAggregate(ctx, projectID, *aggregate, s.clock.Now()); err != nil {
		return nil, fmt.Errorf("recompute project %d financials: %w", projectID, err)
	}

	s.logger.Info("Recomputed project financials",
		"project_id", projectID,
		"total_payout", aggregate.TotalPayoutVND,
		"pending_payable", aggregate.PendingPayableVND,
		"total_received", aggregate.TotalReceivedVND,
		"pending_receivable", aggregate.PendingReceivableVND,
	)

	return aggregate, nil
}

// AggregateRecomputeResult reports how a RecomputeAllProjects run went.
type AggregateRecomputeResult struct {
	// Recomputed is the number of projects whose totals were recomputed and written.
	Recomputed int
	// Failed is the number of projects that could not be recomputed.
	Failed int
}

// RecomputeAllProjects is the one-shot backfill: it recomputes every non-deleted
// project, including those with no timesheets (which reconcile to all zeroes).
//
// It is idempotent, so it doubles as the reconciliation job for a scheduler: a
// second run rewrites the same totals. Per-project failures are logged and
// accumulated instead of aborting the run, so one bad project cannot leave the
// rest of the table stale; the joined error is returned alongside the counts.
func (s *AggregateRecomputeService) RecomputeAllProjects(ctx context.Context) (*AggregateRecomputeResult, error) {
	if s.projects == nil {
		return nil, errors.New("project repository is not configured")
	}

	// Limit is 0, so List returns every non-deleted project (no pagination).
	projects, err := s.projects.List(ctx, domain.ProjectFilters{})
	if err != nil {
		return nil, fmt.Errorf("list projects for financial recompute: %w", err)
	}

	result := &AggregateRecomputeResult{}
	recomputeErrors := make([]error, 0)

	for _, project := range projects {
		if _, err := s.RecomputeProject(ctx, project.ID); err != nil {
			result.Failed++
			recomputeErrors = append(recomputeErrors, err)
			s.logger.Error("Failed to recompute project financials", "project_id", project.ID, "error", err)
			continue
		}
		result.Recomputed++
	}

	s.logger.Info("Project financial aggregate recompute completed",
		"projects", len(projects),
		"recomputed", result.Recomputed,
		"failed", result.Failed,
		"recomputed_at", s.clock.Now(),
	)

	return result, errors.Join(recomputeErrors...)
}
