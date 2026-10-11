package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// ChatToolReadStore implements domain.ChatToolReadStore — the deterministic
// data seam behind the assistant tools. Every query applies the caller-derived
// scope; an empty partner scope yields zero rows (fail closed), never an
// unscoped scan.
type ChatToolReadStore struct {
	*BaseRepository
}

func NewChatToolReadStore(db *Database) *ChatToolReadStore {
	return &ChatToolReadStore{BaseRepository: NewBaseRepository(db)}
}

// chatScopeSQL renders the scope predicate for a table alias. The partner
// branch mirrors partnerAccessCondition() semantics materialised as ID sets
// (see infra/persistence/repositories/partner_scope.go); admin is unscoped.
func chatScopeSQL(alias string, scope domain.ChatDataScope) (string, []interface{}) {
	if scope.Admin() {
		return "1 = 1", nil
	}
	return fmt.Sprintf("(%s.employee_id IN (?) OR %s.project_id IN (?))", alias, alias),
		[]interface{}{scope.EmployeeIDs, scope.ProjectIDs}
}

// chatMonthRange converts a "YYYY-MM" month into a half-open [from, to) date
// range. Empty month means the last 30 days; malformed input falls back to the
// same window (tool arguments arrive from the model and are never trusted).
func chatMonthRange(month string) (from, to time.Time) {
	now := clock.Now()
	month = strings.TrimSpace(month)
	if len(month) == 7 && month[4] == '-' {
		if parsed, err := time.ParseInLocation("2006-01", month, now.Location()); err == nil {
			return parsed, parsed.AddDate(0, 1, 0)
		}
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return start.AddDate(0, 0, -29), start.AddDate(0, 0, 1)
}

func (s *ChatToolReadStore) TimesheetSummary(ctx context.Context, scope domain.ChatDataScope, month string) (*domain.ChatTimesheetSummary, error) {
	from, to := chatMonthRange(month)
	scopeCond, scopeArgs := chatScopeSQL("t", scope)

	var out domain.ChatTimesheetSummary
	err := s.dbForContext(ctx).Raw(`
        SELECT
            COUNT(*) AS total_rows,
            COALESCE(SUM(t.timesheet_status = 'pending_approval'), 0) AS pending_rows,
            COALESCE(SUM(t.timesheet_status = 'approved'), 0) AS approved_rows,
            COALESCE(SUM(t.timesheet_status = 'rejected'), 0) AS rejected_rows,
            COALESCE(SUM(t.payment_status = 'paid'), 0) AS paid_rows,
            COALESCE(SUM(t.timesheet_status = 'approved' AND t.payment_status <> 'paid'), 0) AS approved_unpaid_rows,
            COALESCE(SUM(t.amount), 0) AS total_amount,
            COALESCE(SUM(t.paid_amount), 0) AS paid_amount
        FROM timesheets t
        WHERE t.deleted_at IS NULL
          AND t.date >= ? AND t.date < ?
          AND `+scopeCond,
		append([]interface{}{from, to}, scopeArgs...)...,
	).Scan(&out).Error
	if err != nil {
		return nil, domain.NewInternalError("không thể tổng hợp bảng công", err)
	}
	out.Month = month
	return &out, nil
}

type chatImportJobRow struct {
	AssetID      uint       `gorm:"column:asset_id"`
	ProjectID    uint       `gorm:"column:project_id"`
	OriginalName string     `gorm:"column:original_name"`
	ForMonth     string     `gorm:"column:for_month"`
	Status       string     `gorm:"column:status"`
	LastError    string     `gorm:"column:last_error"`
	MetadataJSON string     `gorm:"column:metadata_json"`
	ProcessedAt  *time.Time `gorm:"column:processed_at"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
}

// chatImportStatsSubset mirrors the BCCImportStats JSON stored in
// assets.metadata (app/services/bcc_import_types.go) — only the fields the
// chat tool narrates.
type chatImportStatsSubset struct {
	TotalRows    int    `json:"total_rows"`
	CreatedCount int    `json:"created_count"`
	SkippedCount int    `json:"skipped_count"`
	ErrorCount   int    `json:"error_count"`
	ErrorDetail  string `json:"error_detail"`
}

func (s *ChatToolReadStore) RecentImportJobs(ctx context.Context, scope domain.ChatDataScope, limit int) ([]*domain.ChatImportJobStatus, error) {
	if limit < 1 || limit > 20 {
		limit = 5
	}
	args := []interface{}{}
	uploaderCond := ""
	if !scope.Admin() {
		// "My import status": a partner sees only the jobs they uploaded.
		uploaderCond = "AND j.uploaded_by = ?"
		args = append(args, scope.UserID)
	}
	args = append(args, limit)

	var rows []chatImportJobRow
	err := s.dbForContext(ctx).Raw(`
        SELECT j.asset_id, j.project_id, a.filename AS original_name, j.for_month,
               j.status, j.last_error, a.metadata AS metadata_json,
               j.processed_at, j.created_at
        FROM timesheet_import_jobs j
        JOIN assets a ON a.id = j.asset_id
        WHERE 1 = 1 `+uploaderCond+`
        ORDER BY j.created_at DESC, j.asset_id DESC
        LIMIT ?`,
		args...,
	).Scan(&rows).Error
	if err != nil {
		return nil, domain.NewInternalError("không thể đọc trạng thái import", err)
	}

	out := make([]*domain.ChatImportJobStatus, 0, len(rows))
	for _, row := range rows {
		item := &domain.ChatImportJobStatus{
			AssetID:         row.AssetID,
			ProjectID:       row.ProjectID,
			OriginalName:    row.OriginalName,
			ForMonth:        row.ForMonth,
			Status:          row.Status,
			LastError:       row.LastError,
			ErrorDetailJSON: "",
			ProcessedAt:     row.ProcessedAt,
			CreatedAt:       row.CreatedAt,
		}
		if strings.TrimSpace(row.MetadataJSON) != "" {
			var stats chatImportStatsSubset
			if err := json.Unmarshal([]byte(row.MetadataJSON), &stats); err == nil {
				item.TotalRows = stats.TotalRows
				item.CreatedCount = stats.CreatedCount
				item.SkippedCount = stats.SkippedCount
				item.ErrorCount = stats.ErrorCount
				item.ErrorDetailJSON = stats.ErrorDetail
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *ChatToolReadStore) AdvanceSummary(ctx context.Context, scope domain.ChatDataScope, month string) (*domain.ChatAdvanceSummary, error) {
	month = strings.TrimSpace(month)
	if len(month) != 7 || month[4] != '-' {
		now := clock.Now()
		month = now.Format("2006-01")
	}
	scopeCond, scopeArgs := chatScopeSQL("ap", scope)

	var out domain.ChatAdvanceSummary
	err := s.dbForContext(ctx).Raw(`
        SELECT
            COUNT(DISTINCT ap.employee_id) AS employee_count,
            COALESCE(SUM(ap.salary), 0) AS total_salary,
            COALESCE(SUM(ap.max_adv_amount), 0) AS max_advance
        FROM advance_payments ap
        WHERE ap.for_month = ? AND `+scopeCond,
		append([]interface{}{month}, scopeArgs...)...,
	).Scan(&out).Error
	if err != nil {
		return nil, domain.NewInternalError("không thể tổng hợp dữ liệu ứng lương", err)
	}
	out.Month = month

	requestScopeCond, requestScopeArgs := chatScopeSQL("ap", scope)
	var req struct {
		RequestedAmount   int64 `gorm:"column:requested_amount"`
		NetPaidAmount     int64 `gorm:"column:net_paid_amount"`
		PendingRequests   int64 `gorm:"column:pending_requests"`
		CompletedRequests int64 `gorm:"column:completed_requests"`
		FailedRequests    int64 `gorm:"column:failed_requests"`
	}
	err = s.dbForContext(ctx).Raw(`
        SELECT
            COALESCE(SUM(r.request_amount), 0) AS requested_amount,
            COALESCE(SUM(CASE WHEN r.status = 'COMPLETED' THEN r.net_amount ELSE 0 END), 0) AS net_paid_amount,
            COALESCE(SUM(r.status = 'PENDING'), 0) AS pending_requests,
            COALESCE(SUM(r.status = 'COMPLETED'), 0) AS completed_requests,
            COALESCE(SUM(r.status = 'FAILED'), 0) AS failed_requests
        FROM advance_payment_requests r
        JOIN advance_payments ap ON ap.id = r.adv_pay_id
        WHERE ap.for_month = ? AND `+requestScopeCond,
		append([]interface{}{month}, requestScopeArgs...)...,
	).Scan(&req).Error
	if err != nil {
		return nil, domain.NewInternalError("không thể tổng hợp yêu cầu ứng lương", err)
	}
	out.RequestedAmount = req.RequestedAmount
	out.NetPaidAmount = req.NetPaidAmount
	out.PendingRequests = req.PendingRequests
	out.CompletedRequests = req.CompletedRequests
	out.FailedRequests = req.FailedRequests
	return &out, nil
}
