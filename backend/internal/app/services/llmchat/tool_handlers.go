package llmchat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"api-server/internal/domain"
)

// Tool handler implementations. Every handler is read-only except
// runFileApprovalRequest, which only ever files a pending request through the
// ApprovalFiler port. All money/number text comes from deterministic queries.

// ToolRefusalError is a designed, user-facing refusal (bad tool arguments,
// disallowed approval type…). The executor records it as "forbidden"/"error"
// and surfaces the message verbatim — it never reaches the generic fallback.
type ToolRefusalError struct{ Msg string }

func (e *ToolRefusalError) Error() string { return e.Msg }

func refuse(msg string) error { return &ToolRefusalError{Msg: msg} }

type monthArgs struct {
	Month string `json:"month"`
}

type limitArgs struct {
	Limit int `json:"limit"`
}

type keywordArgs struct {
	Keyword string `json:"keyword"`
}

type approvalArgs struct {
	Type          string `json:"type"`
	Summary       string `json:"summary"`
	TimesheetID   *uint  `json:"timesheet_id,omitempty"`
	ReportAtDate  string `json:"report_at_date,omitempty"`
	ProjectID     *uint  `json:"project_id,omitempty"`
	RemindForDate string `json:"remind_for_date,omitempty"`
}

// parseArgs decodes and validates tool arguments. Model-produced JSON is
// untrusted input: malformed payloads become a designed refusal, never a
// panic or a zero-value call into the store.
func parseArgs(raw json.RawMessage, dest any) error {
	if err := json.Unmarshal(raw, dest); err != nil {
		return refuse("Tham số công cụ không hợp lệ — vui lòng thử lại với tham số đơn giản hơn.")
	}
	return nil
}

func validMonth(month string) string {
	month = strings.TrimSpace(month)
	if len(month) == 7 && month[4] == '-' {
		return month
	}
	return ""
}

// formatVND renders integer VND with thousand separators (display only; all
// arithmetic stays in the deterministic services).
func formatVND(amount int64) string {
	s := fmt.Sprintf("%d", amount)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out) + " ₫"
	}
	return string(out) + " ₫"
}

func runTimesheetSummary(ctx context.Context, tc *ToolContext, raw json.RawMessage) (string, error) {
	var args monthArgs
	if err := parseArgs(raw, &args); err != nil {
		return "", err
	}
	summary, err := tc.Store.TimesheetSummary(ctx, tc.Scope, validMonth(args.Month))
	if err != nil {
		return "", err
	}
	period := "30 ngày gần nhất"
	if validMonth(args.Month) != "" {
		period = "tháng " + summary.Month
	}
	return fmt.Sprintf(
		"%s: %d bảng công — %d chờ duyệt, %d đã duyệt, %d từ chối; %d đã thanh toán. Tổng tiền %s, đã trả %s.",
		period, summary.TotalRows, summary.PendingRows, summary.ApprovedRows, summary.RejectedRows,
		summary.PaidRows, formatVND(summary.TotalAmount), formatVND(summary.PaidAmount),
	), nil
}

func runImportStatus(ctx context.Context, tc *ToolContext, raw json.RawMessage) (string, error) {
	var args limitArgs
	if err := parseArgs(raw, &args); err != nil {
		return "", err
	}
	jobs, err := tc.Store.RecentImportJobs(ctx, tc.Scope, args.Limit)
	if err != nil {
		return "", err
	}
	if len(jobs) == 0 {
		return "Chưa có lần import bảng công nào.", nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%d lần import gần nhất:\n", len(jobs)))
	for _, job := range jobs {
		name := job.OriginalName
		if name == "" {
			name = fmt.Sprintf("file #%d", job.AssetID)
		}
		b.WriteString(fmt.Sprintf("• %s — tháng %s: %s (%d dòng, %d tạo mới, %d bỏ qua, %d lỗi)\n",
			name, job.ForMonth, job.Status, job.TotalRows, job.CreatedCount, job.SkippedCount, job.ErrorCount))
	}

	// Latest failed import summary — normalised through the error catalog so
	// no raw reason (PII, technical identifiers) reaches the prompt.
	for _, job := range jobs {
		if job.Status != domain.TimesheetImportStatusFailed {
			continue
		}
		items := ParseImportErrors(job.ErrorDetailJSON)
		if len(items) == 0 && strings.TrimSpace(job.LastError) != "" {
			items = []ImportErrorItem{{Reason: SafeImportErrorReason(job.LastError)}}
		}
		if len(items) == 0 {
			b.WriteString("Lần import thất bại gần nhất không có chi tiết lỗi lưu lại.")
			break
		}
		b.WriteString("Lỗi của lần import thất bại gần nhất:\n")
		for _, group := range GroupImportErrors(items) {
			employee := group.Employee
			if employee == "" {
				employee = "không rõ nhân viên"
			}
			b.WriteString(fmt.Sprintf("• %s: %s%s\n", employee, group.Reason, DescribeGroupedError(group)))
		}
		break
	}
	return strings.TrimSpace(b.String()), nil
}

func runApprovalRequests(ctx context.Context, tc *ToolContext, _ json.RawMessage) (string, error) {
	requests, err := tc.Reader.ListForUser(ctx, tc.Caller)
	if err != nil {
		return "", err
	}
	if len(requests) == 0 {
		return "Chưa có yêu cầu phê duyệt nào.", nil
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%d yêu cầu phê duyệt gần nhất:\n", len(requests)))
	for _, req := range requests {
		line := fmt.Sprintf("• #%d %s — %s", req.ID, req.Type, req.Status)
		if req.RetryCount > 0 {
			line += fmt.Sprintf(" (đã thử lại %d lần)", req.RetryCount)
		}
		b.WriteString(line + "\n")
	}
	return strings.TrimSpace(b.String()), nil
}

func runAdvanceSummary(ctx context.Context, tc *ToolContext, raw json.RawMessage) (string, error) {
	var args monthArgs
	if err := parseArgs(raw, &args); err != nil {
		return "", err
	}
	summary, err := tc.Store.AdvanceSummary(ctx, tc.Scope, validMonth(args.Month))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"Tháng %s: %d nhân viên — quỹ lương %s, hạn mức ứng %s; đã yêu cầu %s, đã chi %s. Yêu cầu: %d chờ duyệt, %d hoàn thành, %d thất bại.",
		summary.Month, summary.EmployeeCount, formatVND(summary.TotalSalary), formatVND(summary.MaxAdvance),
		formatVND(summary.RequestedAmount), formatVND(summary.NetPaidAmount),
		summary.PendingRequests, summary.CompletedRequests, summary.FailedRequests,
	), nil
}

func runWalletBalance(ctx context.Context, tc *ToolContext, _ json.RawMessage) (string, error) {
	available, pendingIn, pendingOut, limbo, err := tc.Wallet.WalletBalance(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"Ví chi hộ: khả dụng %s, chờ vào %s, chờ ra %s, chưa đối soát %s.",
		formatVND(available), formatVND(pendingIn), formatVND(pendingOut), formatVND(limbo),
	), nil
}

func runLookupImportError(_ context.Context, _ *ToolContext, raw json.RawMessage) (string, error) {
	var args keywordArgs
	if err := parseArgs(raw, &args); err != nil {
		return "", err
	}
	entries := LookupImportCatalog(args.Keyword)
	if len(entries) == 0 {
		// The keyword may be a raw error message — normalise it through the
		// shared catalog semantics and serve that category.
		normalized := SafeImportErrorReason(args.Keyword)
		for _, entry := range ImportCatalog {
			if entry.Explanation == normalized {
				entries = append(entries, entry)
				break
			}
		}
	}
	if len(entries) == 0 {
		return "Chưa có mục nào khớp từ khóa — thử từ khóa khác (ví dụ: lương, trùng lặp, ngày chấm công).", nil
	}
	var b strings.Builder
	for _, entry := range entries {
		b.WriteString(fmt.Sprintf("• [%s] %s — cách xử lý: %s\n", entry.Code, entry.Explanation, entry.FixHint))
	}
	return strings.TrimSpace(b.String()), nil
}

// approvalTypesByRole is the per-role allowlist of fileable request types. The
// model chooses from this list ONLY — a partner can never file the send_*
// (outbound email) types at MVP (red-team confused-deputy constraint).
var approvalTypesByRole = map[string][]string{
	RoleAdmin:   {"send_sao_ke_email", "send_payroll_email", "send_reminder", "void_timesheet", "other"},
	RolePartner: {"void_timesheet", "other"},
}

func runFileApprovalRequest(ctx context.Context, tc *ToolContext, raw json.RawMessage) (string, error) {
	var args approvalArgs
	if err := parseArgs(raw, &args); err != nil {
		return "", err
	}
	requestType := strings.TrimSpace(args.Type)
	if requestType == "" {
		return "", refuse("Thiếu loại yêu cầu phê duyệt — loại hợp lệ: " + strings.Join(approvalTypesByRole[tc.Caller.Role], ", ") + ".")
	}
	if !containsString(approvalTypesByRole[tc.Caller.Role], requestType) {
		return "", refuse(FallbackUnsupportedAction(tc.Caller.Role, "approval") +
			" Loại yêu cầu " + requestType + " không được phép gửi qua chat.")
	}

	result, err := tc.Filer.FileRequest(ctx, ApprovalFileRequest{
		CallerUserID:  tc.Caller.UserID,
		CallerRole:    tc.Caller.Role,
		Type:          requestType,
		Summary:       truncateRunes(strings.TrimSpace(args.Summary), 500),
		TimesheetID:   args.TimesheetID,
		ReportAtDate:  strings.TrimSpace(args.ReportAtDate),
		ProjectID:     args.ProjectID,
		RemindForDate: strings.TrimSpace(args.RemindForDate),
		CallerName:    tc.Caller.FullName,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"Đã ghi nhận yêu cầu phê duyệt #%d (loại %s, trạng thái %s) — thao tác chỉ được thực hiện sau khi được duyệt: %s",
		result.RequestID, requestType, result.Status, DeepLink(tc.Caller.Role, "approval"),
	), nil
}
