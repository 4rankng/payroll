package llmchat

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
)

// stubStore records which method the handler reached and returns canned data.
type stubStore struct {
	domain.ChatToolReadStore

	called        string
	recipients    *domain.ChatAdvanceRecipients
	report        *domain.ChatReportResult
	list          *domain.ChatScopedList
	reportsCalled []string
}

func (s *stubStore) AdvanceRecipients(_ context.Context, _ domain.ChatDataScope, _, _ string, _ int) (*domain.ChatAdvanceRecipients, error) {
	s.called = "AdvanceRecipients"
	return s.recipients, nil
}

func (s *stubStore) RunCuratedReport(_ context.Context, _ domain.ChatDataScope, spec domain.ChatReportSpec, _, _ string, _ int) (*domain.ChatReportResult, error) {
	s.called = "RunCuratedReport"
	s.reportsCalled = append(s.reportsCalled, spec.Key)
	return s.report, nil
}

func (s *stubStore) MissingTimesheets(_ context.Context, _ domain.ChatDataScope, _ string, _ int) (*domain.ChatScopedList, error) {
	s.called = "MissingTimesheets"
	return s.list, nil
}

func (s *stubStore) AdvanceLimitBreaches(_ context.Context, _ domain.ChatDataScope, _, _ string, _ int) (*domain.ChatScopedList, error) {
	s.called = "AdvanceLimitBreaches"
	return s.list, nil
}

func (s *stubStore) ProjectCashflow(_ context.Context, _ domain.ChatDataScope, _, _ string, _ int) (*domain.ChatScopedList, error) {
	s.called = "ProjectCashflow"
	return s.list, nil
}

func (s *stubStore) Disbursements(_ context.Context, _ domain.ChatDataScope, _, _ string, _ int) (*domain.ChatScopedList, error) {
	s.called = "Disbursements"
	return s.list, nil
}

func (s *stubStore) MissingBankDetails(_ context.Context, _ domain.ChatDataScope, _ int) (*domain.ChatScopedList, error) {
	s.called = "MissingBankDetails"
	return s.list, nil
}

func adminScope() domain.ChatDataScope {
	return domain.ChatDataScope{UserID: 1, Role: RoleAdmin}
}

func TestManifestExposesNewToolsPerRole(t *testing.T) {
	reports := []domain.ChatReportSpec{{Key: "advance_paid_by_month", Title: "T"}}
	reg := NewToolRegistry(reports)

	adminNames := strings.Join(reg.Names(RoleAdmin), ",")
	if !strings.Contains(adminNames, "list_advance_recipients") {
		t.Fatalf("admin manifest missing list_advance_recipients: %s", adminNames)
	}
	if !strings.Contains(adminNames, "run_readonly_report") {
		t.Fatalf("admin manifest missing run_readonly_report: %s", adminNames)
	}

	// The per-role gate is the security boundary for curated reporting.
	partnerNames := strings.Join(reg.Names(RolePartner), ",")
	if strings.Contains(partnerNames, "run_readonly_report") {
		t.Fatalf("run_readonly_report must not reach the partner manifest: %s", partnerNames)
	}
	if !strings.Contains(partnerNames, "list_advance_recipients") {
		t.Fatalf("partner should keep list_advance_recipients (scope-filtered): %s", partnerNames)
	}
}

func TestManifestIsByteStableForSameRole(t *testing.T) {
	reports := []domain.ChatReportSpec{{Key: "advance_paid_by_month"}}
	a := NewToolRegistry(reports).Names(RoleAdmin)
	b := NewToolRegistry(reports).Names(RoleAdmin)
	if strings.Join(a, ",") != strings.Join(b, ",") {
		t.Fatalf("manifest not byte-stable across calls")
	}
}

func TestRunReadOnlyReportRefusesUnknownKey(t *testing.T) {
	reports := []domain.ChatReportSpec{{Key: "advance_paid_by_month", Title: "T"}}
	_ = NewToolRegistry(reports)
	store := &stubStore{report: &domain.ChatReportResult{}}

	tc := &ToolContext{Scope: adminScope(), Store: store}
	args, _ := json.Marshal(map[string]string{
		"report_key": "SELECT * FROM settings", // injection attempt via the key
	})
	_, err := runReadOnlyReport(context.Background(), tc, args)

	if err == nil {
		t.Fatal("expected a refusal for an unknown report_key")
	}
	var refusal *ToolRefusalError
	if !asRefusal(err, &refusal) {
		t.Fatalf("expected ToolRefusalError, got %T", err)
	}
	if store.called != "" {
		t.Fatal("store must not be reached for a refused report_key")
	}
}

func TestRunReadOnlyReportRefusesMissingKey(t *testing.T) {
	_ = NewToolRegistry(nil)
	store := &stubStore{}
	tc := &ToolContext{Scope: adminScope(), Store: store}
	if _, err := runReadOnlyReport(context.Background(), tc, json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected refusal for missing report_key")
	}
	if store.called != "" {
		t.Fatal("store must not be reached")
	}
}

func TestRunReadOnlyReportExecutesAllowListedKey(t *testing.T) {
	reports := []domain.ChatReportSpec{{Key: "advance_paid_by_month", Title: "Tiền ứng theo tháng"}}
	_ = NewToolRegistry(reports)
	store := &stubStore{report: &domain.ChatReportResult{
		ReportKey:  "advance_paid_by_month",
		PeriodFrom: "2026-01 → 2026-12",
		Rows: []*domain.ChatReportRow{
			{Label: "2026-01", Count: 3, Amount: 1_500_000},
		},
	}}
	tc := &ToolContext{Scope: adminScope(), Store: store}

	out, err := runReadOnlyReport(context.Background(), tc, json.RawMessage(`{"report_key":"advance_paid_by_month"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "2026-01") {
		t.Fatalf("output missing period row: %s", out)
	}
	if len(store.reportsCalled) != 1 || store.reportsCalled[0] != "advance_paid_by_month" {
		t.Fatalf("unexpected store calls: %v", store.reportsCalled)
	}
}

func TestListAdvanceRecipientsStatesWindowOnZero(t *testing.T) {
	store := &stubStore{recipients: &domain.ChatAdvanceRecipients{
		Period: "2026-10", From: "2026-10", To: "2026-10", TotalCount: 0,
	}}
	tc := &ToolContext{Scope: adminScope(), Store: store}

	out, err := runListAdvanceRecipients(context.Background(), tc, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "2026-10") {
		t.Fatalf("zero answer must name the scanned window: %s", out)
	}
	if !strings.Contains(out, "CHỈ áp dụng") {
		t.Fatalf("zero answer must forbid widening the empty result: %s", out)
	}
}

func TestListAdvanceRecipientsReportsRealCount(t *testing.T) {
	store := &stubStore{recipients: &domain.ChatAdvanceRecipients{
		Period: "2026-01 → 2026-12", From: "2026-01", To: "2026-12", TotalCount: 2,
		Rows: []*domain.ChatAdvanceRecipient{
			{EmployeeID: 1, FullName: "Nguyễn A", RequestCount: 3, NetPaidAmount: 2_000_000},
			{EmployeeID: 2, FullName: "Trần B", RequestCount: 1, NetPaidAmount: 500_000},
		},
	}}
	tc := &ToolContext{Scope: adminScope(), Store: store}

	out, err := runListAdvanceRecipients(context.Background(), tc, json.RawMessage(`{"from_month":"2026-01","to_month":"2026-12"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"2 nhân viên", "Nguyễn A", "Trần B", "2026-01 → 2026-12"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestToolsRefuseWhenStoreMissing(t *testing.T) {
	tc := &ToolContext{Scope: adminScope()}

	if _, err := runListAdvanceRecipients(context.Background(), tc, json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected refusal with nil store (recipients)")
	}
	_ = NewToolRegistry([]domain.ChatReportSpec{{Key: "k", Title: "t"}})
	if _, err := runReadOnlyReport(context.Background(), tc, json.RawMessage(`{"report_key":"k"}`)); err == nil {
		t.Fatal("expected refusal with nil store (report)")
	}
}

func TestSystemPromptCarriesGroundingRules(t *testing.T) {
	p := BuildSystemPrompt(RoleAdmin)
	if !strings.Contains(p, "QUY TẮC SỐ LIỆU") {
		t.Fatal("system prompt is missing the data-grounding rules block")
	}
	if !strings.Contains(p, "list_advance_recipients") {
		t.Fatal("grounding rules must name the recipient tool")
	}
	// The frozen template must stay byte-stable: no runtime date in it. The
	// date anchor is injected separately via RuntimeDateContext.
	if strings.Contains(p, RuntimeDateContext()) {
		t.Fatal("date context leaked into the frozen system prompt")
	}
	// Same input → identical bytes (prompt-cache discipline).
	if p != BuildSystemPrompt(RoleAdmin) {
		t.Fatal("system prompt is not byte-stable for the same role")
	}
}

func TestRuntimeDateContextIsClockDerived(t *testing.T) {
	out := RuntimeDateContext()
	if !strings.Contains(out, "MỐC THỜI GIAN HIỆN TẠI") {
		t.Fatalf("missing anchor header: %s", out)
	}
	if !strings.Contains(out, time.Now().Format("2006")) {
		t.Fatalf("anchor should name the current year: %s", out)
	}
}

// asRefusal is a local errors.As shim to avoid importing errors just for one call.
func asRefusal(err error, target **ToolRefusalError) bool {
	r, ok := err.(*ToolRefusalError)
	if ok {
		*target = r
	}
	return ok
}

// The five operational list tools must all be present for both roles (or admin
// only where noted), routed to the right store method, and must render an empty
// result honestly instead of inventing rows.
func TestListToolsAreWiredAndRoleGated(t *testing.T) {
	_ = NewToolRegistry(nil)

	admin := strings.Join(NewToolRegistry(nil).Names(RoleAdmin), ",")
	partner := strings.Join(NewToolRegistry(nil).Names(RolePartner), ",")

	for _, name := range []string{
		"list_missing_timesheets", "list_advance_limit_breaches",
		"list_missing_bank_details", "list_project_cashflow", "lookup_disbursement",
	} {
		if !strings.Contains(admin, name) {
			t.Errorf("admin manifest missing %s", name)
		}
	}

	// Disbursement lookup exposes provider errors across the company: admin only.
	if strings.Contains(partner, "lookup_disbursement") {
		t.Error("lookup_disbursement must not be in the partner manifest")
	}
	// The rest are project-scoped and safe for a partner.
	for _, name := range []string{
		"list_missing_timesheets", "list_advance_limit_breaches",
		"list_missing_bank_details", "list_project_cashflow",
	} {
		if !strings.Contains(partner, name) {
			t.Errorf("partner manifest missing %s", name)
		}
	}
}

func TestListToolsRouteToStoreAndRenderRows(t *testing.T) {
	_ = NewToolRegistry(nil)
	list := &domain.ChatScopedList{
		Period: "2026-10", TotalCount: 1,
		Rows: []*domain.ChatListRow{{Label: "Nguyễn A", SubLabel: "Dự án A", Amount: 1_000_000, Detail: "thiếu ngân hàng"}},
	}

	cases := []struct {
		name string
		run  func(context.Context, *ToolContext, json.RawMessage) (string, error)
		want string
	}{
		{"missing_timesheets", runListMissingTimesheets, "MissingTimesheets"},
		{"advance_limit_breaches", runListAdvanceLimitBreaches, "AdvanceLimitBreaches"},
		{"project_cashflow", runListProjectCashflow, "ProjectCashflow"},
		{"lookup_disbursement", runLookupDisbursement, "Disbursements"},
		{"missing_bank_details", runListMissingBankDetails, "MissingBankDetails"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &stubStore{list: list}
			tcCtx := &ToolContext{Scope: adminScope(), Store: store}
			out, err := tc.run(context.Background(), tcCtx, json.RawMessage(`{}`))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if store.called != tc.want {
				t.Fatalf("routed to %q, want %q", store.called, tc.want)
			}
			if !strings.Contains(out, "Nguyễn A") {
				t.Fatalf("missing row label: %s", out)
			}
		})
	}
}

func TestListToolsReportEmptyResultHonestly(t *testing.T) {
	_ = NewToolRegistry(nil)
	store := &stubStore{list: &domain.ChatScopedList{Period: "2026-10", TotalCount: 0}}
	tc := &ToolContext{Scope: adminScope(), Store: store}

	out, err := runListMissingTimesheets(context.Background(), tc, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "2026-10") {
		t.Fatalf("empty result must still name the period: %s", out)
	}
	if strings.Contains(out, "Nguyễn") {
		t.Fatalf("empty result must not invent rows: %s", out)
	}
}
