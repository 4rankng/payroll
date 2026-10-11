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
		// Names and summaries only — internal IDs never reach the model, so
		// replies cannot quote them back at users.
		line := fmt.Sprintf("• %s — %s (người gửi: %s)", req.Summary, req.StatusLabel, req.RequesterName)
		if req.RetryCount > 0 {
			line += fmt.Sprintf(", đã thử lại %d lần", req.RetryCount)
		}
		if req.ScheduledAt != "" {
			line += fmt.Sprintf(", thử lại lúc %s", req.ScheduledAt)
		}
		b.WriteString(line + "\n")
	}
	return strings.TrimSpace(b.String()), nil
}

func runListEmployees(ctx context.Context, tc *ToolContext, raw json.RawMessage) (string, error) {
	var args struct {
		Limit int `json:"limit"`
	}
	if err := parseArgs(raw, &args); err != nil {
		return "", err
	}
	employees, total, err := tc.Store.EmployeeDirectory(ctx, tc.Scope, args.Limit)
	if err != nil {
		return "", err
	}
	if total == 0 {
		return "Không có nhân viên nào trong phạm vi của bạn.", nil
	}
	names := make([]string, 0, len(employees))
	for _, e := range employees {
		if e == nil {
			continue
		}
		label := e.FullName
		if label == "" {
			label = e.EmployeeCode
		}
		if label != "" {
			names = append(names, label)
		}
	}
	shown := strings.Join(names, ", ")
	if total > int64(len(names)) {
		return fmt.Sprintf("Tổng cộng %d nhân viên trong phạm vi của bạn. Một số người: %s…", total, shown), nil
	}
	return fmt.Sprintf("Tổng cộng %d nhân viên trong phạm vi của bạn: %s.", total, shown), nil
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

// requireStore degrades a misconfigured bootstrap into a designed refusal
// instead of a nil-interface panic mid-turn.
func requireStore(tc *ToolContext) error {
	if tc == nil || tc.Store == nil {
		return refuse("Dữ liệu tra cứu đang tạm thời không sẵn sàng — vui lòng thử lại sau.")
	}
	return nil
}

// periodArgs is the shared shape for every range-scoped read. Empty bounds mean
// "current month" on that side — resolved server-side, never guessed by the model.
type periodArgs struct {
	FromMonth string `json:"from_month"`
	ToMonth   string `json:"to_month"`
	Limit     int    `json:"limit"`
}

// runListAdvanceRecipients answers "how many employees actually received an
// advance" from COMPLETED requests. The copy is explicit about the window that
// was scanned so the model can restate it instead of inventing "this year".
func runListAdvanceRecipients(ctx context.Context, tc *ToolContext, raw json.RawMessage) (string, error) {
	if err := requireStore(tc); err != nil {
		return "", err
	}
	var args periodArgs
	if err := parseArgs(raw, &args); err != nil {
		return "", err
	}
	res, err := tc.Store.AdvanceRecipients(ctx, tc.Scope, args.FromMonth, args.ToMonth, args.Limit)
	if err != nil {
		return "", err
	}
	if res.TotalCount == 0 {
		// Pin the zero to the scanned window. The hallucinated answer came from
		// the model widening a narrow empty result into "all of this year".
		return fmt.Sprintf(
			"Khoảng %s: KHÔNG có nhân viên nào nhận ứng lương (0 người, 0 đồng). "+
				"Số 0 này CHỈ áp dụng cho đúng khoảng %s — không suy rộng ra các khoảng khác.",
			res.Period, res.Period,
		), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Khoảng %s: %d nhân viên đã nhận ứng lương.", res.Period, res.TotalCount)
	shown := res.Rows
	if len(shown) > 10 {
		shown = shown[:10]
	}
	for i, row := range shown {
		sep := ", "
		if i == 0 {
			sep = " Gồm: "
		}
		name := row.FullName
		if name == "" {
			name = fmt.Sprintf("#%d", row.EmployeeID)
		}
		fmt.Fprintf(&b, "%s%s (%d lượt, %s)", sep, name, row.RequestCount, formatVND(row.NetPaidAmount))
	}
	if len(res.Rows) > len(shown) {
		fmt.Fprintf(&b, "… và %d người khác.", res.TotalCount-int64(len(shown)))
	}
	return b.String(), nil
}

// renderScopedList is the shared narrator for every list tool: it names the
// window, states the count, and lists rows — so no handler can drift into
// inventing a number or widening the period.
func renderScopedList(head string, list *domain.ChatScopedList, empty string) string {
	if list == nil {
		return "Không đọc được dữ liệu."
	}
	if len(list.Rows) == 0 {
		return fmt.Sprintf("%s (%s): %s", head, list.Period, empty)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s): %d mục.", head, list.Period, list.TotalCount)
	shown := list.Rows
	if len(shown) > 10 {
		shown = shown[:10]
	}
	for _, row := range shown {
		fmt.Fprintf(&b, "\n- %s", row.Label)
		if row.SubLabel != "" {
			fmt.Fprintf(&b, " (%s)", row.SubLabel)
		}
		if row.Amount != 0 {
			fmt.Fprintf(&b, ": %s", formatVND(row.Amount))
		}
		if row.Detail != "" {
			fmt.Fprintf(&b, " — %s", row.Detail)
		}
	}
	if len(list.Rows) > len(shown) {
		fmt.Fprintf(&b, "\n… và %d mục khác.", list.TotalCount-int64(len(shown)))
	}
	return b.String()
}

// runListMissingTimesheets answers "who has not submitted a timesheet".
func runListMissingTimesheets(ctx context.Context, tc *ToolContext, raw json.RawMessage) (string, error) {
	if err := requireStore(tc); err != nil {
		return "", err
	}
	var args struct {
		Month string `json:"month"`
		Limit int    `json:"limit"`
	}
	if err := parseArgs(raw, &args); err != nil {
		return "", err
	}
	list, err := tc.Store.MissingTimesheets(ctx, tc.Scope, validMonth(args.Month), args.Limit)
	if err != nil {
		return "", err
	}
	return renderScopedList("Thiếu bảng công", list,
		"không có nhân viên nào thiếu bảng công — tất cả đã gửi đủ."), nil
}

// runListAdvanceLimitBreaches answers "who exceeded their advance limit".
func runListAdvanceLimitBreaches(ctx context.Context, tc *ToolContext, raw json.RawMessage) (string, error) {
	if err := requireStore(tc); err != nil {
		return "", err
	}
	var args periodArgs
	if err := parseArgs(raw, &args); err != nil {
		return "", err
	}
	list, err := tc.Store.AdvanceLimitBreaches(ctx, tc.Scope, args.FromMonth, args.ToMonth, args.Limit)
	if err != nil {
		return "", err
	}
	return renderScopedList("Vượt hạn mức ứng lương", list,
		"không có trường hợp nào vượt hạn mức trong khoảng này."), nil
}

// runListProjectCashflow answers "which project moved the most money".
func runListProjectCashflow(ctx context.Context, tc *ToolContext, raw json.RawMessage) (string, error) {
	if err := requireStore(tc); err != nil {
		return "", err
	}
	var args periodArgs
	if err := parseArgs(raw, &args); err != nil {
		return "", err
	}
	list, err := tc.Store.ProjectCashflow(ctx, tc.Scope, args.FromMonth, args.ToMonth, args.Limit)
	if err != nil {
		return "", err
	}
	return renderScopedList("Tiền đã trả theo dự án", list,
		"không có dự án nào phát sinh thanh toán trong khoảng này."), nil
}

// runLookupDisbursement answers "why did this transfer fail".
func runLookupDisbursement(ctx context.Context, tc *ToolContext, raw json.RawMessage) (string, error) {
	if err := requireStore(tc); err != nil {
		return "", err
	}
	var args struct {
		Reference string `json:"reference"`
		Status    string `json:"status"`
		Limit     int    `json:"limit"`
	}
	if err := parseArgs(raw, &args); err != nil {
		return "", err
	}
	list, err := tc.Store.Disbursements(ctx, tc.Scope, args.Reference, args.Status, args.Limit)
	if err != nil {
		return "", err
	}
	return renderScopedList("Giao dịch chi hộ", list,
		"không tìm thấy giao dịch nào khớp — kiểm tra lại mã giao dịch hoặc bộ lọc trạng thái."), nil
}

// runListMissingBankDetails answers "who cannot be paid yet". It reports the
// employee and WHICH fields are missing — never the account values.
func runListMissingBankDetails(ctx context.Context, tc *ToolContext, raw json.RawMessage) (string, error) {
	if err := requireStore(tc); err != nil {
		return "", err
	}
	var args struct {
		Limit int `json:"limit"`
	}
	if err := parseArgs(raw, &args); err != nil {
		return "", err
	}
	list, err := tc.Store.MissingBankDetails(ctx, tc.Scope, args.Limit)
	if err != nil {
		return "", err
	}
	return renderScopedList("Nhân viên thiếu thông tin ngân hàng", list,
		"mọi nhân viên trong phạm vi đã có đủ thông tin ngân hàng."), nil
}

// runReadOnlyReport runs one allow-listed admin report. There is no SQL
// parameter — report_key is matched against the registry and refused otherwise,
// so model text can never reach the statement.
func runReadOnlyReport(ctx context.Context, tc *ToolContext, raw json.RawMessage) (string, error) {
	if err := requireStore(tc); err != nil {
		return "", err
	}
	var args struct {
		ReportKey string `json:"report_key"`
		periodArgs
	}
	if err := parseArgs(raw, &args); err != nil {
		return "", err
	}
	key := strings.TrimSpace(args.ReportKey)
	if key == "" {
		return "", refuse("Thiếu report_key — hãy chọn một báo cáo trong danh mục.")
	}

	// Exact allow-list match. No prefix, no normalisation, no interpolation —
	// anything not present here is refused before a statement is built.
	var spec *domain.ChatReportSpec
	for i := range chatReportSpecs {
		if chatReportSpecs[i].Key == key {
			spec = &chatReportSpecs[i]
			break
		}
	}
	if spec == nil {
		return "", refuse("Báo cáo \"" + key + "\" không có trong danh mục — chỉ dùng các mã được liệt kê.")
	}

	res, err := tc.Store.RunCuratedReport(ctx, tc.Scope, *spec, args.FromMonth, args.ToMonth, args.Limit)
	if err != nil {
		return "", err
	}
	if len(res.Rows) == 0 {
		return fmt.Sprintf("%s — khoảng %s: không có dữ liệu.", spec.Title, res.PeriodFrom), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s — khoảng %s (%d dòng):", spec.Title, res.PeriodFrom, len(res.Rows))
	for _, row := range res.Rows {
		label := row.Label
		if label == "" {
			label = "(không rõ)"
		}
		fmt.Fprintf(&b, "\n- %s: %d lượt, %s", label, row.Count, formatVND(row.Amount))
		if row.Detail != "" {
			fmt.Fprintf(&b, " (%s)", row.Detail)
		}
	}
	if res.Truncated {
		b.WriteString("\n(đã giới hạn số dòng — hãy thu hẹp khoảng thời gian để xem đầy đủ)")
	}
	return b.String(), nil
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
		"Đã ghi nhận yêu cầu phê duyệt: %s (trạng thái: %s) — thao tác chỉ được thực hiện sau khi quản trị viên duyệt: %s",
		domain.ApprovalTypeLabelVN(domain.ApprovalRequestType(requestType)),
		domain.ApprovalRequestStatus(result.Status).LabelVN(),
		DeepLink(tc.Caller.Role, "approval"),
	), nil
}
