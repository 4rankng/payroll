package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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

// chatPeriodMonths normalises a (from,to) "YYYY-MM" pair into a closed range.
// Empty bounds mean "current month" on that side; from > to is swapped so a
// sloppy model call still returns a sane, self-describing window. Returns the
// period label plus the resolved bounds — never a silent fallback the model
// cannot see.
func chatPeriodMonths(from, to string, now time.Time) (label, lo, hi string, ok bool) {
	cur := now.Format("2006-01")
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	valid := func(s string) bool {
		if len(s) != 7 || s[4] != '-' {
			return false
		}
		_, err := time.ParseInLocation("2006-01", s, now.Location())
		return err == nil
	}
	switch {
	case from == "" && to == "":
		return cur, cur, cur, true
	case from == "":
		if !valid(to) {
			return "", "", "", false
		}
		return to, to, to, true
	case to == "":
		if !valid(from) {
			return "", "", "", false
		}
		return from, from, from, true
	}
	if !valid(from) || !valid(to) {
		return "", "", "", false
	}
	if from > to {
		from, to = to, from
	}
	return from + " → " + to, from, to, true
}

// AdvanceRecipients answers "which employees actually received an advance
// payment in <period>" from COMPLETED requests only. Counting the advance_payments
// roster instead would report every employee with a limit configured — the bug
// that made the assistant claim a year had zero recipients.
func (s *ChatToolReadStore) AdvanceRecipients(ctx context.Context, scope domain.ChatDataScope, fromMonth, toMonth string, limit int) (*domain.ChatAdvanceRecipients, error) {
	now := clock.Now()
	label, lo, hi, ok := chatPeriodMonths(fromMonth, toMonth, now)
	if !ok {
		return nil, domain.NewValidationError("khoang thoi gian khong hop le")
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	scopeCond, scopeArgs := chatScopeSQL("r", scope)

	var total int64
	countSQL := `
        SELECT COUNT(DISTINCT r.employee_id)
        FROM advance_payment_requests r
        JOIN advance_payments ap ON ap.id = r.adv_pay_id
        WHERE r.status = 'COMPLETED' AND ap.for_month >= ? AND ap.for_month <= ? AND ` + scopeCond
	countArgs := append([]interface{}{lo, hi}, scopeArgs...)
	if err := s.dbForContext(ctx).Raw(countSQL, countArgs...).Scan(&total).Error; err != nil {
		return nil, domain.NewInternalError("khong dem duoc nguoi da nhan ung luong", err)
	}

	rows := []*domain.ChatAdvanceRecipient{}
	listSQL := `
        SELECT r.employee_id AS employee_id, e.fullname AS full_name,
               COUNT(*) AS request_count,
               COALESCE(SUM(r.net_amount), 0) AS net_paid_amount,
               MIN(ap.for_month) AS first_paid_month,
               MAX(ap.for_month) AS last_paid_month
        FROM advance_payment_requests r
        JOIN advance_payments ap ON ap.id = r.adv_pay_id
        JOIN employees e ON e.id = r.employee_id AND e.deleted_at IS NULL
        WHERE r.status = 'COMPLETED' AND ap.for_month >= ? AND ap.for_month <= ? AND ` + scopeCond + `
        GROUP BY r.employee_id, e.fullname
        ORDER BY net_paid_amount DESC, r.employee_id ASC
        LIMIT ?`
	listArgs := append(append([]interface{}{lo, hi}, scopeArgs...), limit)
	if err := s.dbForContext(ctx).Raw(listSQL, listArgs...).Scan(&rows).Error; err != nil {
		return nil, domain.NewInternalError("khong doc duoc danh sach nguoi nhan ung luong", err)
	}

	return &domain.ChatAdvanceRecipients{
		Period:     label,
		From:       lo,
		To:         hi,
		TotalCount: total,
		Rows:       rows,
	}, nil
}

// Operational list SQL. Named constants (not inline strings) so the safety
// tests can assert on every statement the chatbot can reach. Each carries a
// /*SCOPE*/ placeholder that chatScopeSQL fills at execution time.
const (
	chatSQLMissingTimesheets = `
        SELECT e.id AS employee_id, e.fullname AS label, pe.project_id AS project_id,
               COALESCE(p.name, '') AS sub_label, 0 AS count, 0 AS amount,
               '' AS status, '' AS detail, '' AS period_at
        FROM project_employees pe
        JOIN employees e ON e.id = pe.employee_id AND e.deleted_at IS NULL
        LEFT JOIN projects p ON p.id = pe.project_id
        WHERE pe.deleted_at IS NULL
          AND (pe.last_date IS NULL OR pe.last_date >= ?)
          AND NOT EXISTS (
              SELECT 1 FROM timesheets t
              WHERE t.project_id = pe.project_id
                AND t.employee_id = pe.employee_id
                AND t.deleted_at IS NULL
                AND t.date >= ? AND t.date < ?
          )
          AND ` + chatScopeCondPlaceholder + `
        ORDER BY e.fullname ASC
        LIMIT ?`

	chatSQLAdvanceLimitBreaches = `
        SELECT ap.employee_id AS employee_id, e.fullname AS label,
               ap.project_id AS project_id, ap.for_month AS sub_label,
               COUNT(r.id) AS count,
               COALESCE(SUM(r.request_amount), 0) AS amount,
               'OVER_LIMIT' AS status,
               CONCAT('Han muc: ', ap.max_adv_amount, ' — vuot: ',
                      COALESCE(SUM(r.request_amount), 0) - ap.max_adv_amount) AS detail,
               ap.for_month AS period_at
        FROM advance_payments ap
        JOIN employees e ON e.id = ap.employee_id AND e.deleted_at IS NULL
        JOIN advance_payment_requests r ON r.adv_pay_id = ap.id
        WHERE ap.for_month >= ? AND ap.for_month <= ?
          AND r.status IN ('PENDING', 'APPROVED', 'COMPLETED')
          AND ` + chatScopeCondPlaceholder + `
        GROUP BY ap.id, ap.employee_id, e.fullname, ap.project_id, ap.for_month, ap.max_adv_amount
        HAVING COALESCE(SUM(r.request_amount), 0) > ap.max_adv_amount
        ORDER BY (COALESCE(SUM(r.request_amount), 0) - ap.max_adv_amount) DESC
        LIMIT ?`

	// MissingBankDetails reads bank fields ONLY inside CASE expressions to
	// count what is blank. The account number itself is never projected.
	chatSQLMissingBankDetails = `
        SELECT DISTINCT e.id AS employee_id, e.fullname AS label,
               0 AS project_id, '' AS sub_label,
               (CASE WHEN e.bank_id IS NULL THEN 1 ELSE 0 END
                + CASE WHEN e.bank_account_number = '' THEN 1 ELSE 0 END
                + CASE WHEN e.bank_account_name = '' THEN 1 ELSE 0 END) AS count,
               0 AS amount, 'MISSING_BANK' AS status,
               CONCAT(
                 CASE WHEN e.bank_account_status = 'invalid' THEN 'Tai khoan bi ngan hang tu choi. ' ELSE '' END,
                 CASE WHEN e.bank_id IS NULL THEN 'thieu ngan hang; ' ELSE '' END,
                 CASE WHEN e.bank_account_number = '' THEN 'thieu so tai khoan; ' ELSE '' END,
                 CASE WHEN e.bank_account_name = '' THEN 'thieu ten chu tai khoan' ELSE ''
               ) AS detail,
               '' AS period_at
        FROM employees e
        JOIN project_employees pe ON pe.employee_id = e.id AND pe.deleted_at IS NULL
        WHERE e.deleted_at IS NULL
          AND (e.bank_id IS NULL OR e.bank_account_number = '' OR e.bank_account_name = '')
          AND ` + chatScopeCondPlaceholder + `
        ORDER BY e.fullname ASC
        LIMIT ?`
)

// chatMonthStart returns the first instant of a "YYYY-MM" month, falling back to
// the current month for empty/malformed input.
func chatMonthStart(month string, now time.Time) time.Time {
	month = strings.TrimSpace(month)
	if len(month) == 7 && month[4] == '-' {
		if parsed, err := time.ParseInLocation("2006-01", month, now.Location()); err == nil {
			return parsed
		}
	}
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}

// scanChatList runs a scoped list statement and returns the rows plus the total.
// It substitutes the /*SCOPE*/ placeholder centrally, so no call site can
// forget to apply the caller's scope — an unscoped leak would otherwise be a
// silent, per-query mistake.
func (s *ChatToolReadStore) scanChatList(ctx context.Context, sqlText, scopeCond string, args []interface{}) ([]*domain.ChatListRow, int64, error) {
	if scopeCond == "" {
		return nil, 0, fmt.Errorf("chat list query requires a scope predicate")
	}
	statement := strings.ReplaceAll(sqlText, chatScopeCondPlaceholder, scopeCond)
	rows := []*domain.ChatListRow{}
	if err := s.dbForContext(ctx).Raw(statement, args...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, int64(len(rows)), nil
}

// MissingTimesheets answers "who has not submitted a timesheet this month".
// Active assignments with no timesheet row at all — the gap an admin hunts for
// before payroll runs.
func (s *ChatToolReadStore) MissingTimesheets(ctx context.Context, scope domain.ChatDataScope, month string, limit int) (*domain.ChatScopedList, error) {
	now := clock.Now()
	start := chatMonthStart(month, now)
	label := month
	if strings.TrimSpace(month) == "" {
		label = now.Format("2006-01")
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	scopeCond, scopeArgs := chatScopeSQL("pe", scope)
	nextMonth := start.AddDate(0, 1, 0)
	args := append([]interface{}{start}, append([]interface{}{start, nextMonth}, scopeArgs...)...)
	rows, total, err := s.scanChatList(ctx, chatSQLMissingTimesheets, scopeCond, append(args, limit))
	if err != nil {
		return nil, domain.NewInternalError("khong tim duoc danh sach thieu bang cong", err)
	}
	return &domain.ChatScopedList{Period: label, From: label, To: label, TotalCount: total, Rows: rows}, nil
}

// AdvanceLimitBreaches finds advance rows whose live requests exceed the
// configured limit — a compliance signal, not a summary number.
func (s *ChatToolReadStore) AdvanceLimitBreaches(ctx context.Context, scope domain.ChatDataScope, fromMonth, toMonth string, limit int) (*domain.ChatScopedList, error) {
	now := clock.Now()
	label, lo, hi, ok := chatPeriodMonths(fromMonth, toMonth, now)
	if !ok {
		return nil, domain.NewValidationError("khoang thoi gian khong hop le")
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	scopeCond, scopeArgs := chatScopeSQL("ap", scope)
	rows, total, err := s.scanChatList(ctx, chatSQLAdvanceLimitBreaches, scopeCond,
		append([]interface{}{lo, hi}, append(scopeArgs, limit)...))
	if err != nil {
		return nil, domain.NewInternalError("khong tim duoc truong hop vuot han muc", err)
	}
	return &domain.ChatScopedList{Period: label, From: lo, To: hi, TotalCount: total, Rows: rows}, nil
}

// ProjectCashflow aggregates paid money per project so an admin can ask
// "which project moved the most" without leaving chat.
func (s *ChatToolReadStore) ProjectCashflow(ctx context.Context, scope domain.ChatDataScope, fromMonth, toMonth string, limit int) (*domain.ChatScopedList, error) {
	now := clock.Now()
	label, lo, hi, ok := chatPeriodMonths(fromMonth, toMonth, now)
	if !ok {
		return nil, domain.NewValidationError("khoang thoi gian khong hop le")
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	from := chatMonthStart(lo, now)
	to := chatMonthStart(hi, now).AddDate(0, 1, 0)

	scopeCond, scopeArgs := chatScopeSQL("t", scope)
	sqlText := `
        SELECT t.project_id AS project_id,
               COALESCE(p.name, '(khong ro)') AS label,
               '' AS sub_label, 0 AS employee_id,
               COUNT(*) AS count,
               COALESCE(SUM(t.paid_amount), 0) AS amount,
               '' AS status, '' AS detail, '' AS period_at
        FROM timesheets t
        LEFT JOIN projects p ON p.id = t.project_id
        WHERE t.deleted_at IS NULL AND t.payment_status = 'paid'
          AND t.date >= ? AND t.date < ?
          AND ` + scopeCond + `
        GROUP BY t.project_id, p.name
        ORDER BY amount DESC
        LIMIT ?`
	rows, _, err := s.scanChatList(ctx, sqlText, scopeCond, append([]interface{}{from, to}, append(scopeArgs, limit)...))
	if err != nil {
		return nil, domain.NewInternalError("khong tong hop duoc luu chuyen theo du an", err)
	}
	return &domain.ChatScopedList{Period: label, From: lo, To: hi, TotalCount: int64(len(rows)), Rows: rows}, nil
}

// Disbursements answers "why did this transfer fail" by matching a wallet
// payment reference. Recipient ACCOUNT NUMBERS are never selected.
func (s *ChatToolReadStore) Disbursements(ctx context.Context, scope domain.ChatDataScope, reference, status string, limit int) (*domain.ChatScopedList, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	reference = strings.TrimSpace(reference)
	status = strings.ToUpper(strings.TrimSpace(status))

	// wallet_payments carries no project/employee column of its own; the scope
	// is enforced through the linked advance request. For a partner, require
	// the linked request to exist and match their scope.
	where := "1 = 1"
	args := []interface{}{reference, reference, reference, reference, status, status}
	if !scope.Admin() {
		where = "apr.employee_id IN (?) OR apr.project_id IN (?)"
		args = append(args, scope.EmployeeIDs, scope.ProjectIDs)
	}
	args = append(args, limit)

	sqlText := `
        SELECT COALESCE(e.id, 0) AS employee_id,
               COALESCE(e.fullname, wp.recipient_name, '') AS label,
               COALESCE(apr.project_id, 0) AS project_id, '' AS sub_label,
               1 AS count, wp.requested_amount AS amount, wp.status AS status,
               COALESCE(wp.error_message, '') AS detail,
               DATE_FORMAT(wp.created_at, '%Y-%m') AS period_at
        FROM wallet_payments wp
        LEFT JOIN advance_payment_requests apr ON apr.id = wp.entity_id
        LEFT JOIN employees e ON e.id = apr.employee_id
        WHERE (? = '' OR wp.txn_id = ? OR wp.request_id = ? OR wp.invoice_no = ?)
          AND (? = '' OR wp.status = ?)
          AND ` + where + `
        ORDER BY wp.created_at DESC
        LIMIT ?`
	rows, total, err := s.scanChatList(ctx, sqlText, where, args)
	if err != nil {
		return nil, domain.NewInternalError("khong tra cuu duoc giao dich chi ho", err)
	}
	return &domain.ChatScopedList{Period: "gần đây", TotalCount: total, Rows: rows}, nil
}

// MissingBankDetails lists employees who cannot be paid because a banking
// field is blank. It selects the employee identity ONLY — the account number
// and bank are never read, so the chatbot cannot echo them back.
func (s *ChatToolReadStore) MissingBankDetails(ctx context.Context, scope domain.ChatDataScope, limit int) (*domain.ChatScopedList, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	scopeCond, scopeArgs := chatScopeSQL("pe", scope)
	rows, total, err := s.scanChatList(ctx, chatSQLMissingBankDetails, scopeCond, append(scopeArgs, limit))
	if err != nil {
		return nil, domain.NewInternalError("khong tim duoc nhan vien thieu thong tin ngan hang", err)
	}
	return &domain.ChatScopedList{Period: "hiện tại", TotalCount: total, Rows: rows}, nil
}

// Curated reports. The model picks a KEY from this list; the SQL is a compile-
// time constant here. There is deliberately no free-form SQL path into the
// chatbot — `settings.value` holds the MiMo secret key and employees hold CCCD
// and bank numbers, so an interpolating "run this query" tool would hand an
// LLM a credential-exfiltration primitive.
var curatedChatReports = map[string]struct {
	Title   string
	Summary string
	SQL     string
}{
	"advance_paid_by_month": {
		Title:   "Số tiền ứng lương đã chi theo tháng",
		Summary: "Tổng tiền ứng lương đã chi (COMPLETED) theo từng tháng trong khoảng.",
		SQL: `
        SELECT ap.for_month AS label, COUNT(*) AS count, COALESCE(SUM(r.net_amount), 0) AS amount,
               '' AS detail, ap.for_month AS period_at
        FROM advance_payment_requests r
        JOIN advance_payments ap ON ap.id = r.adv_pay_id
        WHERE r.status = 'COMPLETED' AND ap.for_month >= ? AND ap.for_month <= ? AND ` + chatScopeCondPlaceholder + `
        GROUP BY ap.for_month ORDER BY ap.for_month`,
	},
	"timesheet_paid_by_project": {
		Title:   "Số bảng công đã thanh toán theo dự án",
		Summary: "Số bảng công đã thanh toán và tổng tiền theo từng dự án trong khoảng.",
		SQL: `
        SELECT COALESCE(p.name, '(khong ro)') AS label, COUNT(*) AS count,
               COALESCE(SUM(t.paid_amount), 0) AS amount, '' AS detail, '' AS period_at
        FROM timesheets t
        JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
        WHERE t.deleted_at IS NULL AND t.payment_status = 'paid'
          AND DATE_FORMAT(t.date, '%Y-%m') >= ? AND DATE_FORMAT(t.date, '%Y-%m') <= ?
          AND ` + chatScopeCondPlaceholder + `
        GROUP BY p.name ORDER BY count DESC`,
	},
	"employee_payment_activity": {
		Title:   "Nhân viên đã nhận lương trong khoảng",
		Summary: "Danh sách nhân viên có bảng công đã thanh toán trong khoảng, kèm số bảng công.",
		SQL: `
        SELECT e.fullname AS label, COUNT(*) AS count, COALESCE(SUM(t.paid_amount), 0) AS amount,
               CONCAT('Thanh toán gần nhất: ', DATE_FORMAT(MAX(t.paid_at), '%d/%m/%Y')) AS detail,
               DATE_FORMAT(MAX(t.paid_at), '%Y-%m') AS period_at
        FROM timesheets t
        JOIN employees e ON e.id = t.employee_id AND e.deleted_at IS NULL
        WHERE t.deleted_at IS NULL AND t.payment_status = 'paid'
          AND DATE_FORMAT(t.date, '%Y-%m') >= ? AND DATE_FORMAT(t.date, '%Y-%m') <= ?
          AND ` + chatScopeCondPlaceholder + `
        GROUP BY e.id, e.fullname ORDER BY amount DESC`,
	},
}

// chatScopeCondPlaceholder is swapped for the caller's scope predicate at
// execution time. It is a token, never SQL, so no model-supplied text can ever
// reach the statement.
const chatScopeCondPlaceholder = "/*SCOPE*/"

func (s *ChatToolReadStore) RunCuratedReport(ctx context.Context, scope domain.ChatDataScope, spec domain.ChatReportSpec, fromMonth, toMonth string, limit int) (*domain.ChatReportResult, error) {
	entry, ok := curatedChatReports[spec.Key]
	if !ok {
		return nil, domain.NewValidationError("bao cao khong ton tai")
	}
	now := clock.Now()
	label, lo, hi, ok := chatPeriodMonths(fromMonth, toMonth, now)
	if !ok {
		return nil, domain.NewValidationError("khoang thoi gian khong hop le")
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	// Each report joins a different alias, so the scope predicate is rendered
	// for the alias the statement actually exposes (r = requests, t = timesheets).
	scopeCond, scopeArgs := chatScopeSQL("r", scope)
	if spec.Key == "timesheet_paid_by_project" || spec.Key == "employee_payment_activity" {
		scopeCond, scopeArgs = chatScopeSQL("t", scope)
	}
	sqlText := strings.ReplaceAll(entry.SQL, chatScopeCondPlaceholder, scopeCond)
	// LIMIT is appended as a bound parameter — never string-concatenated.
	args := append([]interface{}{lo, hi}, scopeArgs...)
	args = append(args, limit+1)

	rows := []*domain.ChatReportRow{}
	if err := s.dbForContext(ctx).Raw(sqlText, args...).Scan(&rows).Error; err != nil {
		return nil, domain.NewInternalError("khong chay duoc bao cao", err)
	}
	truncated := false
	if len(rows) > limit {
		rows = rows[:limit]
		truncated = true
	}
	return &domain.ChatReportResult{
		ReportKey:  spec.Key,
		PeriodFrom: label,
		PeriodTo:   hi,
		Rows:       rows,
		Truncated:  truncated,
	}, nil
}

// CuratedReportSpecs exposes the allow-list to the app layer (manifest + tests).
func CuratedReportSpecs() []domain.ChatReportSpec {
	specs := make([]domain.ChatReportSpec, 0, len(curatedChatReports))
	for key, entry := range curatedChatReports {
		specs = append(specs, domain.ChatReportSpec{Key: key, Title: entry.Title, Description: entry.Summary})
	}
	// Stable order for prompt-cache discipline.
	sort.Slice(specs, func(i, j int) bool { return specs[i].Key < specs[j].Key })
	return specs
}

// Admin sees everyone; a partner sees employees attached to their projects;
// an empty partner scope yields zero rows (fail closed).
// EmployeeDirectory lists employees visible in the caller's scope with the
// total count — the read-model mirror of the partner employees endpoint.
// Admin sees everyone; a partner sees employees attached to their projects;
// an empty partner scope yields zero rows (fail closed).
func (s *ChatToolReadStore) EmployeeDirectory(ctx context.Context, scope domain.ChatDataScope, limit int) ([]*domain.ChatEmployeeEntry, int64, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	where, args := chatScopeSQL("pe", scope)
	db := s.DB.WithContext(ctx)
	var total int64
	if err := db.Raw(
		"SELECT COUNT(DISTINCT pe.employee_id) FROM project_employees pe WHERE pe.deleted_at IS NULL AND "+where,
		args...,
	).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []*domain.ChatEmployeeEntry{}
	listArgs := append(append([]interface{}{}, args...), limit)
	if err := db.Raw(
		"SELECT DISTINCT pe.employee_id AS id, e.fullname AS full_name, e.cccd AS employee_code "+
			"FROM project_employees pe JOIN employees e ON e.id = pe.employee_id AND e.deleted_at IS NULL "+
			"WHERE pe.deleted_at IS NULL AND "+where+" LIMIT ?",
		listArgs...,
	).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
