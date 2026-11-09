package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// BankUsageSummaryRow is one precomputed bank bucket of the dashboard
// bank-usage cards. projectID 0 = the overall breakdown; >0 = a project card.
type BankUsageSummaryRow struct {
	ProjectID     uint64  `gorm:"column:project_id"`
	BankName      string  `gorm:"column:bank_name"`
	EmployeeCount int     `gorm:"column:employee_count"`
	TotalPaid     float64 `gorm:"column:total_paid"`
	TransferCount int     `gorm:"column:transfer_count"`
}

// EmployeeProjectBankUsageRow is one (employee, project, bank) aggregate from
// the single-pass timesheet scan feeding the summary refresh.
type EmployeeProjectBankUsageRow struct {
	EmployeeID    uint64  `gorm:"column:employee_id"`
	ProjectID     uint64  `gorm:"column:project_id"`
	BankName      string  `gorm:"column:bank_name"`
	TotalPaid     float64 `gorm:"column:total_paid"`
	TransferCount int64   `gorm:"column:transfer_count"`
}

// ActiveProjectPair is one active assignment (deleted_at IS NULL AND
// last_date IS NULL) — the row set both legacy bank-usage queries treat as
// "belongs on the dashboard".
type ActiveProjectPair struct {
	EmployeeID uint64 `gorm:"column:employee_id"`
	ProjectID  uint64 `gorm:"column:project_id"`
}

// ScanBankUsageByEmployeeProjectBank aggregates paid timesheets exactly once,
// grouped by (employee, project, bank).
//
// Deliberately aggregate-first: the legacy overall query joined every
// employee's paid timesheets through a LEFT JOIN whose only employee-side
// condition sits at position 4 of idx_timesheets_paid_date_cover, so MySQL
// walks the entire paid prefix once per employee (~36k entries × ~1.6k
// employees ≈ 58M row checks; measured 87-97s of pure CPU on the 1-vCPU prod
// box). One pass here; dimensions fold in
// app/services/dashboard/bank_usage_summary.go.
func (r *TimesheetAnalyticsRepository) ScanBankUsageByEmployeeProjectBank(ctx context.Context) ([]EmployeeProjectBankUsageRow, error) {
	var rows []EmployeeProjectBankUsageRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			t.employee_id,
			t.project_id,
			COALESCE(b.branch_name, 'Không có ngân hàng') AS bank_name,
			COALESCE(SUM(t.paid_amount), 0) AS total_paid,
			COUNT(*) AS transfer_count
		FROM timesheets t
		JOIN employees e ON e.id = t.employee_id AND e.deleted_at IS NULL
		LEFT JOIN banks b ON b.id = e.bank_id
		WHERE t.deleted_at IS NULL
		  AND t.payment_status = 'paid'
		GROUP BY t.employee_id, t.project_id, bank_name
	`).Scan(&rows).Error
	return rows, err
}

// ActiveProjectPairs lists every active assignment — ~1.6k rows on prod.
func (r *TimesheetAnalyticsRepository) ActiveProjectPairs(ctx context.Context) ([]ActiveProjectPair, error) {
	var rows []ActiveProjectPair
	err := r.db.WithContext(ctx).Raw(`
		SELECT employee_id, project_id
		FROM project_employees
		WHERE deleted_at IS NULL
		  AND last_date IS NULL
	`).Scan(&rows).Error
	return rows, err
}

// ActiveProjectNames maps live project ids to names (projects the legacy
// per-project join could surface).
func (r *TimesheetAnalyticsRepository) ActiveProjectNames(ctx context.Context) (map[uint64]string, error) {
	var rows []struct {
		ID   uint64 `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT id, name
		FROM projects
		WHERE deleted_at IS NULL
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	names := make(map[uint64]string, len(rows))
	for _, row := range rows {
		names[row.ID] = row.Name
	}
	return names, nil
}

// ReplaceBankUsageSummary atomically swaps the whole summary snapshot inside
// one transaction. Rows carry their scope in ProjectID (0 = overall).
func (r *TimesheetAnalyticsRepository) ReplaceBankUsageSummary(ctx context.Context, rows []BankUsageSummaryRow, computedAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM dashboard_bank_usage_summaries`).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		values := make([]map[string]interface{}, 0, len(rows))
		for _, row := range rows {
			values = append(values, map[string]interface{}{
				"project_id":         row.ProjectID,
				"bank_name":          row.BankName,
				"employee_count":     row.EmployeeCount,
				"total_paid":         row.TotalPaid,
				"transfer_count":     row.TransferCount,
				"computed_at_millis": computedAt.UnixMilli(),
			})
		}
		return tx.Table("dashboard_bank_usage_summaries").Create(values).Error
	})
}

// GetBankUsageSummary returns the buckets of one scope (projectID nil = every
// scope) plus the snapshot staleness stamp (unix millis; 0 = never computed).
func (r *TimesheetAnalyticsRepository) GetBankUsageSummary(ctx context.Context, projectID *uint) ([]BankUsageSummaryRow, int64, error) {
	var rows []BankUsageSummaryRow
	var computedMillis int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Raw(`
			SELECT project_id, bank_name, employee_count, total_paid, transfer_count
			FROM dashboard_bank_usage_summaries
		`)
		if projectID != nil {
			query = tx.Raw(`
				SELECT project_id, bank_name, employee_count, total_paid, transfer_count
				FROM dashboard_bank_usage_summaries
				WHERE project_id = ?
			`, *projectID)
		}
		if err := query.Scan(&rows).Error; err != nil {
			return err
		}
		return tx.Raw(`
			SELECT COALESCE(MAX(computed_at_millis), 0)
			FROM dashboard_bank_usage_summaries
		`).Scan(&computedMillis).Error
	})
	return rows, computedMillis, err
}
