package persistence

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
)

// projectFinancialAggregateSQL aggregates the four denormalised project financial
// totals from the project's non-deleted timesheets, which are the authoritative
// per-project money record (transactions carry no project dimension; partner
// receipts are tracked by timesheets.revenue_paid, set when a settlement covers
// the timesheet).
//
// Placeholder order: paid status, approved status, pending status, failed status,
// revenue-paid flag true (collected), revenue-paid flag false (outstanding),
// project ID.
const projectFinancialAggregateSQL = `
	SELECT
		COALESCE(SUM(CASE WHEN payment_status = ? THEN COALESCE(paid_amount, 0) ELSE 0 END), 0) AS total_payout_vnd,
		COALESCE(SUM(CASE WHEN timesheet_status = ? AND payment_status IN (?, ?) THEN COALESCE(amount, 0) ELSE 0 END), 0) AS pending_payable_vnd,
		-- revenue the partner has already settled
		COALESCE(SUM(CASE WHEN revenue_paid = ? THEN COALESCE(revenue_receivable, 0) ELSE 0 END), 0) AS total_received_vnd,
		-- revenue billed to the partner but not settled yet
		COALESCE(SUM(CASE WHEN revenue_paid = ? THEN COALESCE(revenue_receivable, 0) ELSE 0 END), 0) AS pending_receivable_vnd
	FROM timesheets
	WHERE project_id = ? AND deleted_at IS NULL
`

// GetProjectFinancialAggregate aggregates the four denormalised project financial
// totals for one project. The meaning of each total is documented on domain.Project.
func (r *ProjectRepository) GetProjectFinancialAggregate(ctx context.Context, projectID uint) (*domain.ProjectFinancialAggregate, error) {
	var aggregate domain.ProjectFinancialAggregate

	err := r.DB.WithContext(ctx).
		Raw(projectFinancialAggregateSQL,
			domain.PaymentStatusPaid,
			domain.TimesheetStatusApproved,
			domain.PaymentStatusPending,
			domain.PaymentStatusFailed,
			true,
			false,
			projectID,
		).
		Scan(&aggregate).Error

	if err != nil {
		return nil, fmt.Errorf("aggregate timesheet financials for project %d: %w", projectID, err)
	}

	return &aggregate, nil
}

// UpdateProjectFinancialAggregate writes all four denormalised totals in a single
// UPDATE statement and stamps updated_at with the caller's business time.
//
// It is a no-op when projectID does not exist or is soft-deleted: the statement
// simply matches no row. Unknown/deleted IDs are deliberately not an error, so
// repeating the recompute is always safe.
func (r *ProjectRepository) UpdateProjectFinancialAggregate(
	ctx context.Context,
	projectID uint,
	aggregate domain.ProjectFinancialAggregate,
	updatedAt time.Time,
) error {
	result := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Where("id = ?", projectID).
		Updates(map[string]any{
			"total_payout_vnd":       aggregate.TotalPayoutVND,
			"pending_payable_vnd":    aggregate.PendingPayableVND,
			"total_received_vnd":     aggregate.TotalReceivedVND,
			"pending_receivable_vnd": aggregate.PendingReceivableVND,
			"updated_at":             updatedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("update financial aggregate for project %d: %w", projectID, result.Error)
	}

	return nil
}
