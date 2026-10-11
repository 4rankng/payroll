package approval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"api-server/internal/app/dto"

	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
)

// ---- fakes -----------------------------------------------------------------

type fakeCycle struct {
	rows []*domainServices.ProjectReportData
	err  error
}

func (f *fakeCycle) GetProjectsForPayrollReport(context.Context, time.Time) ([]*domainServices.ProjectReportData, error) {
	return f.rows, f.err
}

type fakeProjectUsers struct {
	links map[uint][]*domain.ProjectUser
}

func (f *fakeProjectUsers) GetProjectUsers(_ context.Context, projectID uint) ([]*domain.ProjectUser, error) {
	return f.links[projectID], nil
}

type fakeUsers struct {
	byID   map[uint]*domain.User
	admins []*domain.User
}

func (f *fakeUsers) GetByID(_ context.Context, id uint) (*domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.NewNotFoundError("user")
	}
	return u, nil
}

func (f *fakeUsers) GetByIDs(_ context.Context, ids []uint) (map[uint]*domain.User, error) {
	out := map[uint]*domain.User{}
	for _, id := range ids {
		if u, ok := f.byID[id]; ok {
			out[id] = u
		}
	}
	return out, nil
}

func (f *fakeUsers) ListByRole(_ context.Context, role domain.UserRole) ([]*domain.User, error) {
	if role == domain.RoleAdmin {
		return f.admins, nil
	}
	return nil, nil
}

type fakeTimesheets struct{ byID map[uint]*domain.Timesheet }

func (f *fakeTimesheets) GetByID(_ context.Context, id uint) (*domain.Timesheet, error) {
	ts, ok := f.byID[id]
	if !ok {
		return nil, domain.NewNotFoundError("timesheet")
	}
	return ts, nil
}

type fakeRepo struct {
	mu      sync.Mutex
	rows    map[uint]*domain.ApprovalRequest
	byKey   map[string]*domain.ApprovalRequest
	status  map[uint]domain.ApprovalRequestStatus
	nextID  uint
	saves   int
	lastExp []domain.ApprovalRequestStatus
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{rows: map[uint]*domain.ApprovalRequest{}, byKey: map[string]*domain.ApprovalRequest{}, status: map[uint]domain.ApprovalRequestStatus{}}
}

func (f *fakeRepo) Create(_ context.Context, r *domain.ApprovalRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, dup := f.byKey[r.IdempotencyKey]; dup {
		return errors.New("duplicate entry")
	}
	f.nextID++
	r.ID = f.nextID
	f.rows[r.ID] = r
	f.byKey[r.IdempotencyKey] = r
	f.status[r.ID] = r.Status
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, id uint) (*domain.ApprovalRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rows[id]
	if !ok {
		return nil, domain.NewNotFoundError("approval request")
	}
	return r, nil
}

func (f *fakeRepo) GetByIdempotencyKey(_ context.Context, key string) (*domain.ApprovalRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.byKey[key]; ok {
		return r, nil
	}
	return nil, domain.NewNotFoundError("approval request")
}

func (f *fakeRepo) List(_ context.Context, filters domain.ApprovalRequestFilters) ([]*domain.ApprovalRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*domain.ApprovalRequest
	for _, r := range f.rows {
		if filters.RequesterID != nil && r.RequesterID != *filters.RequesterID {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeRepo) Count(_ context.Context, filters domain.ApprovalRequestFilters) (int64, error) {
	items, err := f.List(context.Background(), filters)
	return int64(len(items)), err
}

func (f *fakeRepo) SaveTransition(_ context.Context, r *domain.ApprovalRequest, expected []domain.ApprovalRequestStatus) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	current, ok := f.status[r.ID]
	if !ok {
		return errors.New("missing row")
	}
	for _, e := range expected {
		if current == e {
			f.status[r.ID] = r.Status
			f.saves++
			f.lastExp = expected
			return nil
		}
	}
	return errors.New("conflict")
}

type fakeNotifier struct {
	calls []struct {
		userID uint
		nType  domain.NotificationType
	}
}

func (f *fakeNotifier) CreateNotification(_ context.Context, userID uint, nType domain.NotificationType, _, _ string) error {
	f.calls = append(f.calls, struct {
		userID uint
		nType  domain.NotificationType
	}{userID, nType})
	return nil
}

type fakeRetry struct {
	calls []struct {
		id      uint
		attempt int
		at      time.Time
	}
}

func (f *fakeRetry) ScheduleRetry(_ context.Context, id uint, attempt int, at time.Time) error {
	f.calls = append(f.calls, struct {
		id      uint
		attempt int
		at      time.Time
	}{id, attempt, at})
	return nil
}

type fakeExecuter struct {
	calls int
	err   error
}

func (f *fakeExecuter) Execute(context.Context, *domain.ApprovalRequest) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	return "executed-ok", nil
}

type fakeClock struct{ t time.Time }

func (f fakeClock) Now() time.Time    { return f.t }
func (f fakeClock) NowUTC() time.Time { return f.t.UTC() }
func (f fakeClock) TodayStart() time.Time {
	y, m, d := f.t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, f.t.Location())
}
func (f fakeClock) TodayEnd() time.Time {
	y, m, d := f.t.Date()
	return time.Date(y, m, d, 23, 59, 59, 999999999, f.t.Location())
}
func (f fakeClock) UnixNow() int64 { return f.t.Unix() }

// ---- fixture ---------------------------------------------------------------

func email(s string) *string { return &s }

func newTestService(t *testing.T) (*Service, *fakeCycle, *fakeTimesheets, *fakeRepo, *fakeNotifier, *fakeRetry, *fakeExecuter) {
	t.Helper()
	cycle := &fakeCycle{rows: []*domainServices.ProjectReportData{{
		Project:    &domain.Project{ID: 7, Name: "Bến Lức – Long Thành"},
		Timesheets: []*domain.Timesheet{{ID: 901, Status: domain.TimesheetStatusApproved}},
	}}}
	pu := &fakeProjectUsers{links: map[uint][]*domain.ProjectUser{7: {{ProjectID: 7, UserID: 42}}}}
	users := &fakeUsers{
		byID: map[uint]*domain.User{
			1:  {ID: 1, Role: domain.RoleAdmin, Email: email("admin@test.vn")},
			2:  {ID: 2, Role: domain.RolePartner, Email: email("partner@test.vn")},
			42: {ID: 42, Role: domain.RolePartner, Email: email("receiver@test.vn")},
		},
		admins: []*domain.User{{ID: 1, Role: domain.RoleAdmin}},
	}
	ts := &fakeTimesheets{byID: map[uint]*domain.Timesheet{55: {ID: 55, Status: domain.TimesheetStatusApproved}}}
	repo := newFakeRepo()
	notifier := &fakeNotifier{}
	retry := &fakeRetry{}
	exec := &fakeExecuter{}
	svc := NewService(Deps{
		Repo:          repo,
		Payloads:      NewPayloadBuilderRegistry(cycle, pu, users, ts, nil, nil),
		Preconditions: NewPreconditionRegistry(cycle, ts),
		Users:         users,
		Notifier:      notifier,
		Retry:         retry,
		Audit:         nil,
		Executers: map[domain.ApprovalRequestType]Executer{
			domain.ApprovalTypeSendSaoKeEmail: exec,
		},
		Clock: fakeClock{t: time.Date(2026, 10, 11, 10, 0, 0, 0, time.Local)},
	})
	return svc, cycle, ts, repo, notifier, retry, exec
}

func admin() Actor   { return Actor{ID: 1, Role: RoleAdmin, Name: "Admin"} }
func partner() Actor { return Actor{ID: 2, Role: RolePartner, Name: "Partner"} }

func saoKeParams() []byte { return []byte(`{"reportAtDate":"2026-10-05"}`) }

// ---- tests -----------------------------------------------------------------

func TestFileApproveExecute(t *testing.T) {
	svc, _, _, _, notifier, _, exec := newTestService(t)
	ctx := context.Background()

	req, err := svc.File(ctx, CreateParams{Type: domain.ApprovalTypeSendSaoKeEmail, Params: saoKeParams(), ActorID: 1, ActorRole: RoleAdmin, ActorName: "Admin"})
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	if req.Status != domain.ApprovalStatusPending {
		t.Fatalf("status = %s, want pending", req.Status)
	}
	// Recipients are server-derived, not filer-supplied.
	if req.Payload == "" || req.IdempotencyKey == "" {
		t.Fatal("payload/idempotency key must be derived")
	}

	done, err := svc.Approve(ctx, req.ID, admin())
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if done.Status != domain.ApprovalStatusExecuted {
		t.Fatalf("status = %s, want executed", done.Status)
	}
	if exec.calls != 1 {
		t.Fatalf("executer calls = %d, want 1", exec.calls)
	}
	if done.Outcome == nil || *done.Outcome != "executed-ok" {
		t.Fatalf("outcome = %v", done.Outcome)
	}
	sawResult := false
	for _, n := range notifier.calls {
		if n.nType == domain.NotificationTypeApprovalResult && n.userID == 1 {
			sawResult = true
		}
	}
	if !sawResult {
		t.Fatal("requester must be notified of the result")
	}
}

func TestFileRejectReschedulesAt0800(t *testing.T) {
	svc, _, _, _, _, retry, _ := newTestService(t)
	ctx := context.Background()
	req, _ := svc.File(ctx, CreateParams{Type: domain.ApprovalTypeSendSaoKeEmail, Params: saoKeParams(), ActorID: 1, ActorRole: RoleAdmin, ActorName: "Admin"})

	out, err := svc.Reject(ctx, req.ID, admin(), "gửi mai, hôm nay dữ liệu chưa đủ")
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if out.Status != domain.ApprovalStatusRescheduled {
		t.Fatalf("status = %s, want rescheduled", out.Status)
	}
	if out.RetryCount != 1 {
		t.Fatalf("retry_count = %d, want 1", out.RetryCount)
	}
	if out.DecisionNote == nil || *out.DecisionNote != "gửi mai, hôm nay dữ liệu chưa đủ" {
		t.Fatalf("decision note = %v", out.DecisionNote)
	}
	if len(retry.calls) != 1 || retry.calls[0].attempt != 1 {
		t.Fatalf("retry scheduled = %+v", retry.calls)
	}
	want := time.Date(2026, 10, 12, 8, 0, 0, 0, retry.calls[0].at.Location())
	if !retry.calls[0].at.Equal(want) {
		t.Fatalf("retry at = %s, want %s (next day 08:00 local)", retry.calls[0].at, want)
	}
}

func TestRejectBudgetExhaustionEscalates(t *testing.T) {
	svc, _, _, _, notifier, retry, _ := newTestService(t)
	ctx := context.Background()
	req, _ := svc.File(ctx, CreateParams{Type: domain.ApprovalTypeSendSaoKeEmail, Params: saoKeParams(), ActorID: 1, ActorRole: RoleAdmin, ActorName: "Admin"})

	for i := 1; i <= 3; i++ {
		if _, err := svc.Reject(ctx, req.ID, admin(), fmt.Sprintf("note %d", i)); err != nil {
			t.Fatalf("reject %d: %v", i, err)
		}
		if err := svc.HandleScheduledRetry(ctx, req.ID); err != nil {
			t.Fatalf("retry %d: %v", i, err)
		}
		if req.Status != domain.ApprovalStatusPending {
			t.Fatalf("after retry %d status = %s, want pending", i, req.Status)
		}
	}
	out, err := svc.Reject(ctx, req.ID, admin(), "lần cuối")
	if err != nil {
		t.Fatalf("final reject: %v", err)
	}
	if out.Status != domain.ApprovalStatusEscalated {
		t.Fatalf("status = %s, want escalated", out.Status)
	}
	if len(retry.calls) != 3 {
		t.Fatalf("scheduled retries = %d, want 3 (4th must escalate, not schedule)", len(retry.calls))
	}
	escalated := false
	for _, n := range notifier.calls {
		if n.nType == domain.NotificationTypeApprovalRequest {
			escalated = true
		}
	}
	if !escalated {
		t.Fatal("admins must be notified of escalation")
	}
}

func TestScheduledRetryBlockedByPrecondition(t *testing.T) {
	svc, cycle, _, _, _, retry, _ := newTestService(t)
	ctx := context.Background()
	req, _ := svc.File(ctx, CreateParams{Type: domain.ApprovalTypeSendSaoKeEmail, Params: saoKeParams(), ActorID: 1, ActorRole: RoleAdmin, ActorName: "Admin"})
	svc.Reject(ctx, req.ID, admin(), "thử lại")

	// Data went incomplete again: a timesheet is back in pending_approval.
	cycle.rows = []*domainServices.ProjectReportData{{
		Project:    &domain.Project{ID: 7, Name: "Bến Lức – Long Thành"},
		Timesheets: []*domain.Timesheet{{ID: 901, Status: domain.TimesheetStatusPendingApproval}},
	}}
	if err := svc.HandleScheduledRetry(ctx, req.ID); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if req.Status != domain.ApprovalStatusRescheduled || req.RetryCount != 2 {
		t.Fatalf("status = %s retry = %d, want rescheduled/2", req.Status, req.RetryCount)
	}
	if len(retry.calls) != 2 {
		t.Fatalf("scheduled retries = %d, want 2", len(retry.calls))
	}
}

func TestPartnerRoleWalls(t *testing.T) {
	svc, _, _, _, _, _, _ := newTestService(t)
	ctx := context.Background()

	// Partner may file void_timesheet (unpaid timesheet exists).
	voidParams := []byte(`{"timesheetId":55,"reason":"nhầm ngày"}`)
	req, err := svc.File(ctx, CreateParams{Type: domain.ApprovalTypeVoidTimesheet, Params: voidParams, ActorID: 2, ActorRole: RolePartner, ActorName: "Partner"})
	if err != nil {
		t.Fatalf("partner void file: %v", err)
	}
	// Partner may NOT file the send_* family.
	if _, err := svc.File(ctx, CreateParams{Type: domain.ApprovalTypeSendSaoKeEmail, Params: saoKeParams(), ActorID: 2, ActorRole: RolePartner}); !errors.Is(err, ErrForbiddenRole) {
		t.Fatalf("partner send file err = %v, want ErrForbiddenRole", err)
	}
	// Partner may NOT decide.
	if _, err := svc.Approve(ctx, req.ID, partner()); !errors.Is(err, ErrForbiddenRole) {
		t.Fatalf("partner approve err = %v, want ErrForbiddenRole", err)
	}
	if _, err := svc.Reject(ctx, req.ID, partner(), "note"); !errors.Is(err, ErrForbiddenRole) {
		t.Fatalf("partner reject err = %v, want ErrForbiddenRole", err)
	}
	// Partner list is scoped to own requests.
	items, _, err := svc.List(ctx, partner(), domain.ApprovalRequestFilters{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, r := range items {
		if r.RequesterID != 2 {
			t.Fatalf("partner saw foreign request %d (requester %d)", r.ID, r.RequesterID)
		}
	}
}

func TestFileIdempotency(t *testing.T) {
	svc, _, _, _, _, _, _ := newTestService(t)
	ctx := context.Background()
	first, err := svc.File(ctx, CreateParams{Type: domain.ApprovalTypeSendSaoKeEmail, Params: saoKeParams(), ActorID: 1, ActorRole: RoleAdmin})
	if err != nil {
		t.Fatalf("first file: %v", err)
	}
	second, err := svc.File(ctx, CreateParams{Type: domain.ApprovalTypeSendSaoKeEmail, Params: saoKeParams(), ActorID: 1, ActorRole: RoleAdmin})
	if !errors.Is(err, ErrAlreadyFiled) {
		t.Fatalf("second file err = %v, want ErrAlreadyFiled", err)
	}
	if second == nil || second.ID != first.ID {
		t.Fatal("duplicate file must return the existing request")
	}
}

func TestCancelIsTerminal(t *testing.T) {
	svc, _, _, _, _, _, exec := newTestService(t)
	ctx := context.Background()
	req, _ := svc.File(ctx, CreateParams{Type: domain.ApprovalTypeSendSaoKeEmail, Params: saoKeParams(), ActorID: 1, ActorRole: RoleAdmin})

	out, err := svc.Cancel(ctx, req.ID, admin(), "huỷ, không gửi lại")
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if out.Status != domain.ApprovalStatusCancelled {
		t.Fatalf("status = %s, want cancelled", out.Status)
	}
	if _, err := svc.Approve(ctx, req.ID, admin()); err == nil {
		t.Fatal("approve after cancel must fail")
	}
	if exec.calls != 0 {
		t.Fatal("cancel must never execute")
	}
}

func TestVoidPreconditionBlocksPaidTimesheet(t *testing.T) {
	svc, _, ts, _, _, _, _ := newTestService(t)
	ctx := context.Background()
	ts.byID[55].PaymentStatus = domain.PaymentStatusPaid
	_, err := svc.File(ctx, CreateParams{Type: domain.ApprovalTypeVoidTimesheet, Params: []byte(`{"timesheetId":55}`), ActorID: 2, ActorRole: RolePartner})
	if err == nil {
		t.Fatal("filing void on a paid timesheet must be blocked by the precondition")
	}
}

// ---- phase 4: drafting + custom send ----

type fakeDrafter struct {
	mu       sync.Mutex
	calls    int
	guidance []string
	subject  string
	body     string
	err      error
}

func (f *fakeDrafter) DraftStatementEmail(_ context.Context, _ StatementDraftFacts, guidance string) (string, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.guidance = append(f.guidance, guidance)
	return f.subject, f.body, f.err
}

func newDraftingService(t *testing.T, drafter Drafter) (*Service, *fakeRepo, *fakeExecuter) {
	t.Helper()
	svc, _, _, repo, _, _, exec := newTestService(t)
	if drafter != nil {
		svc.drafter = drafter
	}
	return svc, repo, exec
}

func TestCompleteDraftAttachesAndPersists(t *testing.T) {
	drafter := &fakeDrafter{subject: "Sao kê tuần 05/10", body: "Kính gửi Quý đối tác, …"}
	svc, repo, _ := newDraftingService(t, drafter)
	req, err := svc.File(context.Background(), CreateParams{
		Type: domain.ApprovalTypeSendSaoKeEmail, Params: saoKeParams(), ActorID: 1, ActorRole: RoleAdmin,
	})
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	// File returns without a draft; the background completion writes it.
	clone := *req
	svc.attachDraft(context.Background(), &clone, "")
	if err := svc.repo.SaveTransition(context.Background(), &clone, []domain.ApprovalRequestStatus{domain.ApprovalStatusPending}); err != nil {
		t.Fatalf("persist draft: %v", err)
	}
	var payload struct {
		DraftSubject string `json:"llmDraftSubject"`
		DraftBody    string `json:"llmDraftBody"`
	}
	if err := json.Unmarshal([]byte(clone.Payload), &payload); err != nil {
		t.Fatalf("payload: %v", err)
	}
	if payload.DraftSubject != drafter.subject || payload.DraftBody != drafter.body {
		t.Fatalf("draft not merged: %+v", payload)
	}
	if repo.saves < 1 {
		t.Fatal("draft transition must persist")
	}
}

func TestFileSurvivesDraftFailure(t *testing.T) {
	drafter := &fakeDrafter{err: errors.New("provider down")}
	svc, _, _ := newDraftingService(t, drafter)
	req, err := svc.File(context.Background(), CreateParams{
		Type: domain.ApprovalTypeSendSaoKeEmail, Params: saoKeParams(), ActorID: 1, ActorRole: RoleAdmin,
	})
	if err != nil {
		t.Fatalf("filing must survive a drafting outage: %v", err)
	}
	var payload map[string]any
	_ = json.Unmarshal([]byte(req.Payload), &payload)
	if _, has := payload["llmDraftSubject"]; has {
		t.Fatal("no draft expected on provider failure")
	}
}

func TestRetryRedraftUsesRejectionNote(t *testing.T) {
	drafter := &fakeDrafter{subject: "s", body: "b"}
	svc, _, _ := newDraftingService(t, drafter)
	ctx := context.Background()
	req, _ := svc.File(ctx, CreateParams{Type: domain.ApprovalTypeSendSaoKeEmail, Params: saoKeParams(), ActorID: 1, ActorRole: RoleAdmin})
	if _, err := svc.Reject(ctx, req.ID, admin(), "gửi mai, dữ liệu chưa đủ"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if err := svc.HandleScheduledRetry(ctx, req.ID); err != nil {
		t.Fatalf("retry: %v", err)
	}
	// File drafts asynchronously; the retry redraft runs synchronously in the
	// worker — assert the rejection note reached the drafter, order-independently.
	drafter.mu.Lock()
	sawNote := false
	for _, g := range drafter.guidance {
		if g == "gửi mai, dữ liệu chưa đủ" {
			sawNote = true
		}
	}
	calls := drafter.calls
	drafter.mu.Unlock()
	if !sawNote {
		t.Fatalf("redraft guidance = %v, want the rejection note present", drafter.guidance)
	}
	if calls < 1 {
		t.Fatalf("drafter calls = %d, want >= 1", calls)
	}
}

type fakeDispatch struct {
	last dto.SendPayrollReportEmailRequest
	key  string
}

func (f *fakeDispatch) EnqueuePayrollReportEmail(req dto.SendPayrollReportEmailRequest, _ uint, requestKey string) (string, bool, error) {
	f.last = req
	f.key = requestKey
	return "task-1", false, nil
}

func TestEmailExecuterSendsReviewedDraft(t *testing.T) {
	dispatch := &fakeDispatch{}
	exec := NewEmailExecuter(dispatch)
	req := &domain.ApprovalRequest{
		Type:           domain.ApprovalTypeSendSaoKeEmail,
		IdempotencyKey: "approval:send_sao_ke_email:abc",
		RequesterID:    1,
		Payload:        `{"reportAtDate":"2026-10-05","recipients":["a@test.vn"],"llmDraftSubject":"Sao kê tuần","llmDraftBody":"Kính gửi,"}`,
	}
	outcome, err := exec.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if dispatch.last.CustomSubject != "Sao kê tuần" || dispatch.last.CustomBody != "Kính gửi," {
		t.Fatalf("custom draft not passed: %+v", dispatch.last)
	}
	if dispatch.last.Recipients[0] != "a@test.vn" {
		t.Fatalf("recipients lost: %+v", dispatch.last)
	}
	if outcome == "" {
		t.Fatal("outcome must be human-readable")
	}
}
