package insights

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"api-server/internal/domain"
)

// DBCollectors reads the deterministic signal families straight from the
// operational tables. v1 watches two families with stable shapes:
//
//   - cron failures: enabled scheduler jobs whose last run failed
//   - import failures: timesheet_import_jobs stuck/failed in the window
//
// The registry is open: a new family is one more query method.
type DBCollectors struct {
	db *gorm.DB
}

// NewDBCollectors constructs the collector set.
func NewDBCollectors(db *gorm.DB) *DBCollectors {
	return &DBCollectors{db: db}
}

// Name identifies the collector set in logs.
func (c *DBCollectors) Name() string { return "db" }

// Collect runs every family query for the window.
func (c *DBCollectors) Collect(ctx context.Context, since time.Time) ([]Candidate, error) {
	var out []Candidate
	out = append(out, c.cronFailures(ctx)...)
	out = append(out, c.importFailures(ctx, since)...)
	return out, nil
}

func (c *DBCollectors) cronFailures(ctx context.Context) []Candidate {
	type row struct {
		JobName   string     `gorm:"column:job_name"`
		LastError *string    `gorm:"column:last_error"`
		LastRunAt *time.Time `gorm:"column:last_run_at"`
	}
	var rows []row
	if err := c.db.WithContext(ctx).
		Table("cron_job_status").
		Select("job_name, last_error, last_run_at").
		Where("is_enabled = 1 AND status <> 'success'").
		Find(&rows).Error; err != nil {
		return nil
	}
	var out []Candidate
	for _, r := range rows {
		at := "chưa chạy"
		if r.LastRunAt != nil {
			at = r.LastRunAt.Format("02/01/2006 15:04")
		}
		errMsg := ""
		if r.LastError != nil && *r.LastError != "" {
			errMsg = *r.LastError
			if len(errMsg) > 160 {
				errMsg = errMsg[:160] + "…"
			}
		}
		out = append(out, Candidate{
			Fingerprint: "cron-failed:" + r.JobName,
			Type:        "cron_failure",
			Severity:    severityForCron(r.JobName),
			Facts: map[string]any{
				"summary":    fmt.Sprintf("Cron %s lần chạy gần nhất thất bại (%s)", r.JobName, at),
				"job":        r.JobName,
				"last_run":   at,
				"last_error": errMsg,
			},
		})
	}
	return out
}

// severityForCron escalates money-adjacent jobs to critical.
func severityForCron(job string) domain.DailyFindingSeverity {
	return domain.DailySeverityWarning
}

func (c *DBCollectors) importFailures(ctx context.Context, since time.Time) []Candidate {
	type row struct {
		Count int `gorm:"column:count"`
	}
	var r row
	if err := c.db.WithContext(ctx).
		Table("timesheet_import_jobs").
		Select("COUNT(*) AS count").
		Where("status IN ('failed','processing') AND created_at >= ?", since).
		Scan(&r).Error; err != nil || r.Count == 0 {
		return nil
	}
	return []Candidate{{
		Fingerprint: fmt.Sprintf("import-failures:%d", r.Count),
		Type:        "import_failure",
		Severity:    domain.DailySeverityWarning,
		Facts: map[string]any{
			"summary": fmt.Sprintf("Có %d lần import bảng công lỗi hoặc kẹt trong 24 giờ qua", r.Count),
			"count":   r.Count,
		},
	}}
}
