package project

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence"
	"api-server/internal/pkg/clock"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newAggregateDB builds the minimal schema the recompute service touches: projects
// holds the four denormalised totals, timesheets is the authoritative source, and
// the empty users table keeps the project repository's Creator preload satisfied.
func newAggregateDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/project_aggregates.db"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE projects (
			id INTEGER PRIMARY KEY,
			total_payout_vnd NUMERIC NOT NULL DEFAULT 0,
			pending_payable_vnd NUMERIC NOT NULL DEFAULT 0,
			total_received_vnd NUMERIC NOT NULL DEFAULT 0,
			pending_receivable_vnd NUMERIC NOT NULL DEFAULT 0,
			deleted_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		);
		CREATE TABLE users (id INTEGER PRIMARY KEY, deleted_at DATETIME);
		CREATE TABLE timesheets (
			id INTEGER PRIMARY KEY,
			project_id INTEGER NOT NULL,
			timesheet_status TEXT NOT NULL,
			payment_status TEXT NOT NULL,
			amount INTEGER NOT NULL DEFAULT 0,
			paid_amount INTEGER,
			revenue_receivable INTEGER,
			revenue_paid BOOLEAN,
			deleted_at DATETIME
		);
	`).Error)

	return db
}

// directAggregate is an independently written formulation of the four totals,
// used as the oracle the service result must match.
func directAggregate(t *testing.T, db *gorm.DB, projectID uint) domain.ProjectFinancialAggregate {
	t.Helper()

	var direct domain.ProjectFinancialAggregate
	require.NoError(t, db.Raw(`
		SELECT
			(SELECT COALESCE(SUM(COALESCE(paid_amount, 0)), 0) FROM timesheets
				WHERE project_id = ? AND deleted_at IS NULL AND payment_status = 'paid') AS total_payout_vnd,
			(SELECT COALESCE(SUM(amount), 0) FROM timesheets
				WHERE project_id = ? AND deleted_at IS NULL AND timesheet_status = 'approved'
					AND payment_status IN ('pending', 'failed')) AS pending_payable_vnd,
			(SELECT COALESCE(SUM(revenue_receivable), 0) FROM timesheets
				WHERE project_id = ? AND deleted_at IS NULL AND revenue_paid = 1) AS total_received_vnd,
			(SELECT COALESCE(SUM(revenue_receivable), 0) FROM timesheets
				WHERE project_id = ? AND deleted_at IS NULL AND revenue_paid = 0) AS pending_receivable_vnd
	`, projectID, projectID, projectID, projectID).Scan(&direct).Error)

	return direct
}

func storedAggregate(t *testing.T, db *gorm.DB, projectID uint) (domain.Project, domain.ProjectFinancialAggregate) {
	t.Helper()

	var project domain.Project
	// Unscoped so the assertions can also inspect soft-deleted projects.
	require.NoError(t, db.Unscoped().First(&project, projectID).Error)

	return project, domain.ProjectFinancialAggregate{
		TotalPayoutVND:       project.TotalPayoutVND,
		PendingPayableVND:    project.PendingPayableVND,
		TotalReceivedVND:     project.TotalReceivedVND,
		PendingReceivableVND: project.PendingReceivableVND,
	}
}

// The four projects.*_vnd columns are derived from timesheets. This pins what each
// total counts (paid vs approved-unpaid payouts; collected vs outstanding revenue),
// that soft-deleted rows and other projects are excluded, and that repeated runs
// leave the same numbers.
func TestRecomputeProjectWritesAggregateMatchingDirectQuery(t *testing.T) {
	db := newAggregateDB(t)
	ctx := context.Background()

	require.NoError(t, db.Exec(`INSERT INTO projects (id) VALUES (7), (8)`).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO timesheets (id, project_id, timesheet_status, payment_status, amount, paid_amount, revenue_receivable, revenue_paid, deleted_at) VALUES
			(1, 7, 'approved',         'paid',      1200000, 1000000, 1500000, 1, NULL),
			(2, 7, 'approved',         'pending',    500000, NULL,     750000, 0, NULL),
			(3, 7, 'approved',         'failed',     300000, NULL,          0, 0, NULL),
			(4, 7, 'pending_approval', 'pending',    400000, NULL,          0, 0, NULL),
			(5, 7, 'rejected',         'pending',    200000, NULL,          0, 0, NULL),
			(6, 7, 'approved',         'cancelled',  700000, NULL,          0, 0, NULL),
			(7, 7, 'approved',         'paid',       900000, 900000,   900000, 1, '2026-01-05 00:00:00'),
			(8, 8, 'approved',         'paid',       500000, 5555555, 1111111, 1, NULL),
			(9, 7, 'approved',         'paid',       800000, NULL,     200000, 1, NULL)
	`).Error)

	frozenNow := time.Date(2026, 9, 20, 9, 0, 0, 0, clock.DefaultLocation)
	service := NewAggregateRecomputeService(
		persistence.NewProjectRepository(&persistence.Database{DB: db}),
		clock.NewFake(frozenNow),
		nil,
	)

	aggregate, err := service.RecomputeProject(ctx, 7)
	require.NoError(t, err)

	// Definition pinned literally: only actually paid money counts as payout, only
	// approved work still owed counts as payable, and revenue splits on the
	// settlement flag.
	require.Equal(t, float64(1000000), aggregate.TotalPayoutVND, "1.2m gross paid timesheet counts its 1m paid_amount; NULL paid_amount counts 0")
	require.Equal(t, float64(800000), aggregate.PendingPayableVND, "approved pending + failed only")
	require.Equal(t, float64(1700000), aggregate.TotalReceivedVND, "revenue_receivable of revenue_paid rows")
	require.Equal(t, float64(750000), aggregate.PendingReceivableVND, "revenue_receivable not yet collected")

	// ...and cross-checked against the independent aggregate query.
	require.Equal(t, directAggregate(t, db, 7), *aggregate)

	project, stored := storedAggregate(t, db, 7)
	require.Equal(t, *aggregate, stored)
	require.True(t, project.UpdatedAt.Equal(frozenNow), "updated_at is stamped with the business clock, got %s", project.UpdatedAt)

	// Project 8 is untouched by project 7's recompute.
	_, other := storedAggregate(t, db, 8)
	require.Equal(t, domain.ProjectFinancialAggregate{}, other)

	otherAggregate, err := service.RecomputeProject(ctx, 8)
	require.NoError(t, err)
	require.Equal(t, domain.ProjectFinancialAggregate{
		TotalPayoutVND:   5555555,
		TotalReceivedVND: 1111111,
	}, *otherAggregate)
	require.Equal(t, directAggregate(t, db, 8), *otherAggregate)

	// Idempotent: the values are a pure function of the timesheet rows, so a second
	// run rewrites exactly the same totals.
	repeat, err := service.RecomputeProject(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, *aggregate, *repeat)
	_, storedAgain := storedAggregate(t, db, 7)
	require.Equal(t, *aggregate, storedAgain)
}

// The backfill must reconcile every non-deleted project, including projects that
// only ever carried stale hand-written numbers (they belong at 0) and leave
// soft-deleted projects alone.
func TestRecomputeAllProjectsBackfillsEveryLiveProject(t *testing.T) {
	db := newAggregateDB(t)
	ctx := context.Background()

	require.NoError(t, db.Exec(`
		INSERT INTO projects (id, total_payout_vnd, pending_payable_vnd, total_received_vnd, pending_receivable_vnd) VALUES
			(1, 999, 999, 999, 999),
			(2, 0, 0, 0, 0),
			(3, 500, 500, 500, 500)
	`).Error)
	require.NoError(t, db.Exec(`UPDATE projects SET deleted_at = '2026-02-01 00:00:00' WHERE id = 3`).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO timesheets (id, project_id, timesheet_status, payment_status, amount, paid_amount, revenue_receivable, revenue_paid, deleted_at) VALUES
			(1, 2, 'approved', 'paid',    400000, 400000, 600000, 1, NULL),
			(2, 2, 'approved', 'pending', 100000, NULL,   150000, 0, NULL),
			(3, 3, 'approved', 'paid',    700000, 700000, 700000, 1, NULL)
	`).Error)

	service := NewAggregateRecomputeService(
		persistence.NewProjectRepository(&persistence.Database{DB: db}),
		clock.NewFake(time.Date(2026, 9, 20, 9, 0, 0, 0, clock.DefaultLocation)),
		nil,
	)

	result, err := service.RecomputeAllProjects(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, result.Recomputed)
	require.Equal(t, 0, result.Failed)

	_, stale := storedAggregate(t, db, 1)
	require.Equal(t, domain.ProjectFinancialAggregate{}, stale, "a project with no timesheets reconciles to zero")

	_, backfilled := storedAggregate(t, db, 2)
	require.Equal(t, directAggregate(t, db, 2), backfilled)
	require.Equal(t, float64(400000), backfilled.TotalPayoutVND)
	require.Equal(t, float64(100000), backfilled.PendingPayableVND)
	require.Equal(t, float64(600000), backfilled.TotalReceivedVND)
	require.Equal(t, float64(150000), backfilled.PendingReceivableVND)

	_, deleted := storedAggregate(t, db, 3)
	require.Equal(t, float64(500), deleted.TotalPayoutVND, "soft-deleted projects are not part of the backfill")
}

// A failing project must not abort the run: the remaining projects are still
// reconciled and the failure is reported.
type failingAggregateRepo struct {
	domain.ProjectRepository
	projects []*domain.Project
	failFor  uint
	written  map[uint]domain.ProjectFinancialAggregate
}

func (r *failingAggregateRepo) List(context.Context, domain.ProjectFilters) ([]*domain.Project, error) {
	return r.projects, nil
}

func (r *failingAggregateRepo) GetProjectFinancialAggregate(_ context.Context, projectID uint) (*domain.ProjectFinancialAggregate, error) {
	if projectID == r.failFor {
		return nil, errors.New("timesheets table is unreachable")
	}
	return &domain.ProjectFinancialAggregate{TotalPayoutVND: float64(projectID) * 100}, nil
}

func (r *failingAggregateRepo) UpdateProjectFinancialAggregate(_ context.Context, projectID uint, aggregate domain.ProjectFinancialAggregate, _ time.Time) error {
	r.written[projectID] = aggregate
	return nil
}

func TestRecomputeAllProjectsContinuesPastFailures(t *testing.T) {
	repo := &failingAggregateRepo{
		projects: []*domain.Project{{ID: 1}, {ID: 2}, {ID: 3}},
		failFor:  2,
		written:  map[uint]domain.ProjectFinancialAggregate{},
	}

	service := NewAggregateRecomputeService(repo, clock.NewFake(time.Date(2026, 9, 20, 9, 0, 0, 0, clock.DefaultLocation)), nil)

	result, err := service.RecomputeAllProjects(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "recompute project 2 financials")
	require.True(t, strings.Contains(err.Error(), "timesheets table is unreachable"))
	require.Equal(t, 2, result.Recomputed)
	require.Equal(t, 1, result.Failed)

	require.Equal(t, domain.ProjectFinancialAggregate{TotalPayoutVND: 100}, repo.written[1])
	require.Equal(t, domain.ProjectFinancialAggregate{TotalPayoutVND: 300}, repo.written[3])
	require.NotContains(t, repo.written, uint(2))
}
