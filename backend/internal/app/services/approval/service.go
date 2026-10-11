package approval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
)

// Role strings mirror the JWT role values (domain.RoleAdmin / RolePartner).
// Deliberately local: this package must not import llmchat (the chat adapter
// imports this package — the dependency arrow only points one way).
const (
	RoleAdmin   = "admin"
	RolePartner = "partner"

	// MaxRetryAttempts bounds the reject-with-note retry loop: once the 3rd
	// attempt is consumed the request escalates instead of rescheduling.
	MaxRetryAttempts = 3

	// retryHourICT is the hour (Asia/Ho_Chi_Minh) scheduled retries fire at.
	retryHourICT = 8

	// draftTimeout bounds one background LLM drafting run.
	draftTimeout = 60 * time.Second
)

// partnerFileableTypes is the per-type filer allowlist (red-team constraint):
// partners may request timesheet corrections, never the send_* family — an
// approved partner-filed send_* would be a relay through the company mailbox.
var partnerFileableTypes = map[domain.ApprovalRequestType]bool{
	domain.ApprovalTypeVoidTimesheet: true,
}

// ErrAlreadyFiled is returned when the (type, cycle) idempotency key already
// exists. The existing request is returned alongside so callers can point the
// user at the live request instead of silently doubling it.
var ErrAlreadyFiled = errors.New("yêu cầu này đã được gửi trước đó")

// ErrForbiddenRole is returned when the actor's role may not perform the
// action (filer allowlist or admin-only decision).
var ErrForbiddenRole = domain.NewValidationError("vai trò không được thực hiện hành động này")

// Actor identifies the authenticated user performing an engine operation.
type Actor struct {
	ID   uint
	Role string
	Name string
}

// RetryScheduler enqueues the scheduled retry task (asynq adapter with an
// attempt-scoped TaskID). A scheduling failure aborts the decision before it
// is persisted, so persisted state and queue stay aligned.
type RetryScheduler interface {
	ScheduleRetry(ctx context.Context, requestID uint, attempt int, at time.Time) error
}

// EventWriter persists one lifecycle transition to the audit pipeline
// (audit:log:write). Implemented over the asynq client at bootstrap.
type EventWriter interface {
	WriteApprovalEvent(ctx context.Context, event domain.ApprovalRequestChangedEvent) error
}

// Executer performs the side effects of an approved request. Types without a
// registered executer record an explicit "no execution step yet" outcome —
// never a silent success.
type Executer interface {
	Execute(ctx context.Context, request *domain.ApprovalRequest) (string, error)
}

// Deps bundles the engine dependencies (wired at bootstrap).
type Deps struct {
	Repo          domain.ApprovalRequestRepository
	Payloads      *PayloadBuilderRegistry
	Preconditions *PreconditionRegistry
	Users         UserDirectory
	Notifier      Notifier
	Retry         RetryScheduler
	Audit         EventWriter
	Executers     map[domain.ApprovalRequestType]Executer
	Drafter       Drafter
	Clock         clock.Clock
}

// Service is the approval-request engine: file → decide → execute, with the
// reject-with-note retry loop (08:00 ICT, max 3, then escalation).
type Service struct {
	repo          domain.ApprovalRequestRepository
	payloads      *PayloadBuilderRegistry
	preconditions *PreconditionRegistry
	users         UserDirectory
	notifier      Notifier
	retry         RetryScheduler
	audit         EventWriter
	executers     map[domain.ApprovalRequestType]Executer
	drafter       Drafter
	clk           clock.Clock
	logger        *slog.Logger
}

// NewService constructs the engine.
func NewService(deps Deps) *Service {
	return &Service{
		repo:          deps.Repo,
		payloads:      deps.Payloads,
		preconditions: deps.Preconditions,
		users:         deps.Users,
		notifier:      deps.Notifier,
		retry:         deps.Retry,
		audit:         deps.Audit,
		executers:     deps.Executers,
		drafter:       deps.Drafter,
		clk:           deps.Clock,
		logger:        observability.GetLogger(),
	}
}

// File creates a request from identifying params only. Payload, recipients
// and the idempotency key are all derived server-side.
func (s *Service) File(ctx context.Context, params CreateParams) (*domain.ApprovalRequest, error) {
	if !domain.ValidApprovalRequestType(params.Type) {
		return nil, domain.NewValidationError("loại yêu cầu không hợp lệ")
	}
	if params.ActorRole != RoleAdmin {
		if params.ActorRole != RolePartner || !partnerFileableTypes[params.Type] {
			return nil, ErrForbiddenRole
		}
	}
	built, err := s.payloads.Build(ctx, params.Type, params.Params)
	if err != nil {
		return nil, err
	}
	payloadJSON, err := json.Marshal(built.Payload)
	if err != nil {
		return nil, err
	}
	req := &domain.ApprovalRequest{
		Type:           params.Type,
		Status:         domain.ApprovalStatusPending,
		Payload:        string(payloadJSON),
		IdempotencyKey: domain.BuildApprovalIdempotencyKey(params.Type, built.CycleKey),
		RequesterID:    params.ActorID,
	}
	// Deterministic precondition at creation — the LLM never judges data
	// completeness (red-team constraint).
	if err := s.preconditions.Evaluate(ctx, req); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, req); err != nil {
		if existing, gerr := s.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey); gerr == nil {
			return existing, ErrAlreadyFiled
		}
		return nil, err
	}
	s.writeEvent(ctx, domain.NewApprovalRequestCreatedEvent(ctx, req, params.ActorID, params.ActorName))
	s.notifyAdmins(ctx, domain.NotificationTypeApprovalRequest,
		"Yêu cầu phê duyệt mới", fmt.Sprintf("%s — đang chờ duyệt trên Tổng Quan.", built.Title))
	// Statement emails carry an LLM draft (review aid + sent body). The draft
	// completes in the background: drafting takes seconds, must not run on the
	// HTTP request deadline (RequestTimeout cancels mid-body) and must not
	// delay the filing response. Filing without a draft is the degraded state.
	s.completeDraftAsync(req)
	return req, nil
}

// Approve records the decision and runs the type's execution step.
// Admin-only, re-checked here behind Casbin (defense in depth).
func (s *Service) Approve(ctx context.Context, id uint, actor Actor) (*domain.ApprovalRequest, error) {
	if actor.Role != RoleAdmin {
		return nil, ErrForbiddenRole
	}
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !req.CanBeApproved() {
		return nil, domain.NewValidationError("yêu cầu không ở trạng thái chờ duyệt")
	}
	if err := req.Approve(actor.ID); err != nil {
		return nil, err
	}
	outcome := "chưa có bước thực thi cho loại yêu cầu này"
	execErr := error(nil)
	if exec, ok := s.executers[req.Type]; ok {
		outcome, execErr = exec.Execute(ctx, req)
		if execErr != nil {
			outcome = fmt.Sprintf("lỗi thực thi: %v", execErr)
		}
	}
	if err := req.Execute(outcome, s.clk.Now()); err != nil {
		return nil, err
	}
	expected := []domain.ApprovalRequestStatus{domain.ApprovalStatusPending, domain.ApprovalStatusRescheduled}
	if err := s.repo.SaveTransition(ctx, req, expected); err != nil {
		return nil, err
	}
	s.writeEvent(ctx, domain.NewApprovalRequestExecutedEvent(ctx, req, actor.ID, actor.Name))
	s.notifyUser(ctx, req.RequesterID, domain.NotificationTypeApprovalResult,
		"Yêu cầu đã được duyệt", fmt.Sprintf("Yêu cầu %s đã được duyệt. %s",
			domain.ApprovalTypeLabelVN(req.Type), outcome))
	if execErr != nil {
		return req, fmt.Errorf("đã duyệt nhưng thực thi thất bại: %w", execErr)
	}
	return req, nil
}

// Reject records "Thử lại ngày mai": the note becomes retry guidance and the
// request re-files automatically at 08:00 ICT. After the retry budget is
// spent the request escalates instead.
func (s *Service) Reject(ctx context.Context, id uint, actor Actor, note string) (*domain.ApprovalRequest, error) {
	if actor.Role != RoleAdmin {
		return nil, ErrForbiddenRole
	}
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !req.CanBeRejected() {
		return nil, domain.NewValidationError("yêu cầu không ở trạng thái chờ duyệt")
	}
	if err := req.Reject(actor.ID, note); err != nil {
		return nil, err
	}
	expected := []domain.ApprovalRequestStatus{domain.ApprovalStatusPending, domain.ApprovalStatusRescheduled}

	attempt := req.RetryCount + 1
	if attempt > MaxRetryAttempts {
		if err := req.Escalate(); err != nil {
			return nil, err
		}
		if err := s.repo.SaveTransition(ctx, req, expected); err != nil {
			return nil, err
		}
		s.writeEvent(ctx, domain.NewApprovalRequestRejectedEvent(ctx, req, actor.ID, actor.Name, note))
		s.writeEvent(ctx, domain.NewApprovalRequestEscalatedEvent(ctx, req, actor.ID, actor.Name))
		s.notifyAdmins(ctx, domain.NotificationTypeApprovalRequest,
			"Yêu cầu cần xử lý sau nhiều lần thử lại",
			fmt.Sprintf("Yêu cầu %s đã bị từ chối %d lần — cần quyết định của quản trị viên.",
				domain.ApprovalTypeLabelVN(req.Type), req.RetryCount))
		s.notifyUser(ctx, req.RequesterID, domain.NotificationTypeApprovalResult,
			"Yêu cầu bị từ chối", "Yêu cầu đã bị từ chối và chuyển lên xử lý thủ công.")
		return req, nil
	}

	at := s.nextRetryAt(s.clk.Now())
	if err := req.ScheduleRetry(attempt, at); err != nil {
		return nil, err
	}
	// Enqueue BEFORE persisting: a queue failure leaves the request decidable,
	// never a persisted reschedule without its firing task.
	if err := s.retry.ScheduleRetry(ctx, req.ID, attempt, at); err != nil {
		return nil, fmt.Errorf("không lên lịch được lần thử lại: %w", err)
	}
	if err := s.repo.SaveTransition(ctx, req, expected); err != nil {
		return nil, err
	}
	s.writeEvent(ctx, domain.NewApprovalRequestRejectedEvent(ctx, req, actor.ID, actor.Name, note))
	s.writeEvent(ctx, domain.NewApprovalRequestRescheduledEvent(ctx, req, actor.ID, actor.Name, attempt, at))
	s.notifyUser(ctx, req.RequesterID, domain.NotificationTypeApprovalResult,
		"Yêu cầu bị từ chối — sẽ thử lại",
		fmt.Sprintf("Ghi chú: %s. Hệ thống sẽ chuẩn bị lại vào %s.", note, at.Format("02/01/2006 15:04")))
	return req, nil
}

// Cancel is the terminal "Huỷ / không gửi lại" decision: no retry, no
// escalation, and the (type, cycle) idempotency key stays consumed.
func (s *Service) Cancel(ctx context.Context, id uint, actor Actor, note string) (*domain.ApprovalRequest, error) {
	if actor.Role != RoleAdmin {
		return nil, ErrForbiddenRole
	}
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !req.CanBeCancelled() {
		return nil, domain.NewValidationError("yêu cầu đã đóng, không thể huỷ")
	}
	if err := req.Cancel(actor.ID, note); err != nil {
		return nil, err
	}
	expected := []domain.ApprovalRequestStatus{domain.ApprovalStatusPending, domain.ApprovalStatusRescheduled}
	if err := s.repo.SaveTransition(ctx, req, expected); err != nil {
		return nil, err
	}
	s.writeEvent(ctx, domain.NewApprovalRequestCancelledEvent(ctx, req, actor.ID, actor.Name, note))
	s.notifyUser(ctx, req.RequesterID, domain.NotificationTypeApprovalResult,
		"Yêu cầu đã huỷ", fmt.Sprintf("Yêu cầu %s đã được huỷ và sẽ không được thực thi.",
			domain.ApprovalTypeLabelVN(req.Type)))
	return req, nil
}

// HandleScheduledRetry is the unattended 08:00 entry point. It re-checks the
// deterministic preconditions and may only re-file (rescheduled → pending) —
// it never approves and never sends anything.
func (s *Service) HandleScheduledRetry(ctx context.Context, id uint) error {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !req.CanBeRequeued() {
		// Decided or closed meanwhile — the retry is a no-op.
		return nil
	}
	expected := []domain.ApprovalRequestStatus{domain.ApprovalStatusRescheduled}
	if err := s.preconditions.Evaluate(ctx, req); err != nil {
		// Data still incomplete: consume another attempt or escalate.
		attempt := req.RetryCount + 1
		if attempt > MaxRetryAttempts {
			if err := req.Escalate(); err != nil {
				return err
			}
			if err := s.repo.SaveTransition(ctx, req, expected); err != nil {
				return err
			}
			s.writeEvent(ctx, domain.NewApprovalRequestEscalatedEvent(ctx, req, req.RequesterID, "hệ thống"))
			s.notifyAdmins(ctx, domain.NotificationTypeApprovalRequest,
				"Yêu cầu cần xử lý sau nhiều lần thử lại",
				fmt.Sprintf("Dữ liệu của yêu cầu %s vẫn chưa đủ sau %d lần thử — cần quyết định của quản trị viên.",
					domain.ApprovalTypeLabelVN(req.Type), req.RetryCount))
			return nil
		}
		at := s.nextRetryAt(s.clk.Now())
		if err := req.ScheduleRetry(attempt, at); err != nil {
			return err
		}
		if err := s.retry.ScheduleRetry(ctx, id, attempt, at); err != nil {
			return err
		}
		if err := s.repo.SaveTransition(ctx, req, expected); err != nil {
			return err
		}
		s.writeEvent(ctx, domain.NewApprovalRequestRescheduledEvent(ctx, req, req.RequesterID, "hệ thống", attempt, at))
		return nil
	}
	// Unattended redraft with the rejection note as guidance — runs BEFORE
	// the requeue persists, so the reviewer sees the updated draft immediately.
	guidance := ""
	if req.DecisionNote != nil {
		guidance = *req.DecisionNote
	}
	s.attachDraft(ctx, req, guidance)
	if err := req.RequeueForReview(); err != nil {
		return err
	}
	if err := s.repo.SaveTransition(ctx, req, expected); err != nil {
		return err
	}
	s.writeEvent(ctx, domain.NewApprovalRequestRetryFiredEvent(ctx, req, req.RequesterID, "hệ thống"))
	s.notifyAdmins(ctx, domain.NotificationTypeApprovalRequest,
		"Yêu cầu đã sẵn sàng duyệt lại",
		fmt.Sprintf("%s — đã chuẩn bị lại theo lịch và đang chờ duyệt.", domain.ApprovalTypeLabelVN(req.Type)))
	return nil
}

// List returns requests for the actor: admins see all, partners only their own.
func (s *Service) List(ctx context.Context, actor Actor, filters domain.ApprovalRequestFilters) ([]*domain.ApprovalRequest, int64, error) {
	if actor.Role != RoleAdmin {
		uid := actor.ID
		filters.RequesterID = &uid
	}
	if filters.Limit <= 0 || filters.Limit > 100 {
		filters.Limit = 20
	}
	total, err := s.repo.Count(ctx, filters)
	if err != nil {
		return nil, 0, err
	}
	items, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Get loads one request with handler-level ownership enforced here (Casbin is
// role×path only — red-team IDOR constraint).
func (s *Service) Get(ctx context.Context, id uint, actor Actor) (*domain.ApprovalRequest, error) {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if actor.Role != RoleAdmin && req.RequesterID != actor.ID {
		return nil, domain.NewNotFoundError("không tìm thấy yêu cầu")
	}
	return req, nil
}

// attachDraft writes the LLM draft into a statement-email request's payload.
// It is best-effort by design: on any drafting failure the payload keeps its
// deterministic form and the request still files/requeues.
func (s *Service) attachDraft(ctx context.Context, req *domain.ApprovalRequest, guidance string) {
	if s.drafter == nil {
		return
	}
	switch req.Type {
	case domain.ApprovalTypeSendSaoKeEmail, domain.ApprovalTypeSendPayrollEmail:
	default:
		return
	}
	var p struct {
		ReportAtDate   string   `json:"reportAtDate"`
		PeriodFrom     string   `json:"periodFrom"`
		PeriodTo       string   `json:"periodTo"`
		TotalAmount    int64    `json:"totalAmount"`
		TimesheetCount int      `json:"timesheetCount"`
		Recipients     []string `json:"recipients"`
	}
	if err := json.Unmarshal([]byte(req.Payload), &p); err != nil {
		s.logger.Warn("approval: draft skipped — payload unreadable", "request_type", req.Type)
		return
	}
	subject, body, err := s.drafter.DraftStatementEmail(ctx, StatementDraftFacts{
		ReportAtDate:   p.ReportAtDate,
		PeriodLabel:    fmt.Sprintf("%s → %s", p.PeriodFrom, p.PeriodTo),
		TotalAmount:    formatDraftAmount(p.TotalAmount),
		TimesheetCount: p.TimesheetCount,
		RecipientCount: len(p.Recipients),
	}, guidance)
	if err != nil {
		s.logger.Warn("approval: LLM draft failed — filing continues without draft", "error", err)
		return
	}
	merged := map[string]any{}
	if json.Unmarshal([]byte(req.Payload), &merged) == nil {
		merged["llmDraftSubject"] = subject
		merged["llmDraftBody"] = body
		if out, merr := json.Marshal(merged); merr == nil {
			req.Payload = string(out)
		}
	}
}

// completeDraftAsync finishes the LLM draft after the request has been filed:
// detached context (no HTTP deadline), bounded, unattended-safe — it only
// writes draft text onto a pending request, never decides and never sends.
// Works on a copy: the handler may still be serializing the original.
func (s *Service) completeDraftAsync(req *domain.ApprovalRequest) {
	if s.drafter == nil || req.ID == 0 {
		return
	}
	switch req.Type {
	case domain.ApprovalTypeSendSaoKeEmail, domain.ApprovalTypeSendPayrollEmail:
	default:
		return
	}
	clone := *req
	go func(r *domain.ApprovalRequest) {
		ctx, cancel := context.WithTimeout(context.Background(), draftTimeout)
		defer cancel()
		s.attachDraft(ctx, r, "")
		if err := s.repo.SaveTransition(ctx, r, []domain.ApprovalRequestStatus{domain.ApprovalStatusPending}); err != nil {
			// The request was decided before the draft landed — the reviewer
			// decided on the deterministic payload; dropping is safe.
			s.logger.Info("approval: background draft not persisted", "request_id", r.ID, "error", err)
		}
	}(&clone)
}

// formatDraftAmount renders a VND amount with dot thousand separators.
func formatDraftAmount(v int64) string {
	negative := v < 0
	if negative {
		v = -v
	}
	digits := fmt.Sprintf("%d", v)
	var parts []string
	for len(digits) > 3 {
		parts = append([]string{digits[len(digits)-3:]}, parts...)
		digits = digits[:len(digits)-3]
	}
	parts = append([]string{digits}, parts...)
	out := ""
	for i, part := range parts {
		if i > 0 {
			out += "."
		}
		out += part
	}
	if negative {
		return "-" + out + " đ"
	}
	return out + " đ"
}

// nextRetryAt returns 08:00 ICT on the day after now. clk.Now() is
// Asia/Ho_Chi_Minh localized (ADR-006); time.Date normalizes day overflow.
func (s *Service) nextRetryAt(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d+1, retryHourICT, 0, 0, 0, now.Location())
}

func (s *Service) writeEvent(ctx context.Context, event domain.ApprovalRequestChangedEvent) {
	if s.audit == nil {
		return
	}
	if err := s.audit.WriteApprovalEvent(ctx, event); err != nil {
		s.logger.Error("approval: failed to enqueue audit event", "error", err, "event", event.EventName)
	}
}

func (s *Service) notifyAdmins(ctx context.Context, nType domain.NotificationType, title, message string) {
	admins, err := s.users.ListByRole(ctx, domain.RoleAdmin)
	if err != nil {
		s.logger.Error("approval: failed to list admins for notification", "error", err)
		return
	}
	for _, admin := range admins {
		s.notifyUser(ctx, admin.ID, nType, title, message)
	}
}

func (s *Service) notifyUser(ctx context.Context, userID uint, nType domain.NotificationType, title, message string) {
	if s.notifier == nil {
		return
	}
	if err := s.notifier.CreateNotification(ctx, userID, nType, title, message); err != nil {
		s.logger.Error("approval: failed to create notification", "user_id", userID, "error", err)
	}
}
