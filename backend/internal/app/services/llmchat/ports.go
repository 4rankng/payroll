// Package llmchat is the chat assistant application service: conversations,
// the prompt-injection-resistant tool layer, and the turn orchestrator around
// the phase-1 LLM port (internal/infra/llm via app/services/llm).
//
// Security model (red-team constraints): the tool manifest is the structural
// whitelist — the model can only ever reach the tools registered here, each
// re-checked against the CALLER's role and data scope at call time. Approval
// mutations go through the ApprovalFiler port only; the model can request an
// action, never perform one.
package llmchat

import (
	"context"
	"errors"
	"log/slog"

	"api-server/internal/infra/observability"
)

// Caller is the authenticated identity a chat turn executes as. Tool scope is
// derived from this — never from conversation content.
type Caller struct {
	UserID   uint
	Role     string // "admin" | "partner"
	FullName string
}

// Allowed chat roles. adv_partner is deliberately excluded (it reuses the
// admin layout and must not see the assistant surface).
const (
	RoleAdmin   = "admin"
	RolePartner = "partner"
)

// RoleAllowed reports whether a JWT role may use the chat surface at all.
// Defense in depth behind the Casbin rows — handlers AND the service check.
func RoleAllowed(role string) bool {
	return role == RoleAdmin || role == RolePartner
}

// ErrNotWired is returned by NotConfigured* port implementations when the
// backing feature (the phase-2 approval engine) has not been wired yet. It is
// a typed sentinel so the tool layer can distinguish "not connected" from a
// real failure and tell the user honestly that the action did NOT happen.
var ErrNotWired = errors.New("port chưa được kết nối")

// ErrRateLimited is returned when the caller exceeds the per-user message
// budget. The handler maps it to HTTP 429.
var ErrRateLimited = errors.New("vượt giới hạn gửi tin nhắn")

// ErrTurnTimeout is returned when a turn exceeds the hard per-turn deadline
// (model calls + tool execution end to end). The handler maps it to HTTP 504.
var ErrTurnTimeout = errors.New("hết thời gian xử lý lượt chat")

// ErrChatDisabled is returned when the admin has switched the chat surface
// off for the caller's role (settings mimo.chat_enabled_*).
var ErrChatDisabled = errors.New("trợ lý AI đã bị tắt cho vai trò của bạn")

// RoleGate reports whether the chat surface is currently enabled for a role
// (admin-controlled per-role switches). Implemented over the settings store;
// nil means "no gate wired" — the static RoleAllowed roles apply unchanged.
type RoleGate interface {
	ChatEnabled(ctx context.Context, role string) bool
}

// RoleSettings is the settings surface RoleGate needs (satisfied by
// *config.SettingsConfigService without importing it here).
type RoleSettings interface {
	GetMimoChatEnabledForRole(ctx context.Context, role string) bool
}

type settingsRoleGate struct{ settings RoleSettings }

// NewSettingsRoleGate gates the chat surface on the admin-controlled
// per-role switches.
func NewSettingsRoleGate(settings RoleSettings) RoleGate {
	return settingsRoleGate{settings: settings}
}

func (g settingsRoleGate) ChatEnabled(ctx context.Context, role string) bool {
	return g.settings.GetMimoChatEnabledForRole(ctx, role)
}

// ErrForbiddenRole is returned for roles outside the chat surface.
var ErrForbiddenRole = errors.New("vai trò không được dùng trợ lý AI")

// ApprovalFileRequest is one request to file an approval request via chat.
// Only allowlisted types may ever reach the filer — the executor enforces the
// per-role type allowlist before calling.
type ApprovalFileRequest struct {
	CallerUserID uint
	CallerRole   string
	Type         string
	Summary      string

	// Identifying params for the engine's payload builders (optional at the
	// schema level; the adapter rejects a type whose required param is absent).
	TimesheetID   *uint  `json:"timesheet_id,omitempty"`
	ReportAtDate  string `json:"report_at_date,omitempty"`
	ProjectID     *uint  `json:"project_id,omitempty"`
	RemindForDate string `json:"remind_for_date,omitempty"`
	CallerName    string `json:"-"`
}

// ApprovalFileResult is the filed request's identity.
type ApprovalFileResult struct {
	RequestID uint
	Status    string
}

// ApprovalFiler is the mutation seam for the file_approval_request tool. The
// real adapter talks to app/services/approval (phase 2) and is wired in the
// integration follow-up; NotConfiguredFiler is this wave's implementation.
type ApprovalFiler interface {
	FileRequest(ctx context.Context, req ApprovalFileRequest) (*ApprovalFileResult, error)
}

// ApprovalRequestStatus is one approval request as narrated by the
// get_approval_requests tool.
type ApprovalRequestStatus struct {
	RequesterName string
	TypeLabel     string
	Summary       string
	StatusLabel   string
	RetryCount    int
	ScheduledAt   string
}

// ApprovalReader is the read seam for "my approval requests status". Same
// wiring story as ApprovalFiler.
type ApprovalReader interface {
	ListForUser(ctx context.Context, caller Caller) ([]ApprovalRequestStatus, error)
}

// NotConfiguredFiler is the ApprovalFiler used until the approval engine
// adapter lands. It logs the attempt (so the gap is observable) and returns
// ErrNotWired — it NEVER fakes a filed request.
type NotConfiguredFiler struct{ logger *slog.Logger }

func NewNotConfiguredFiler() *NotConfiguredFiler {
	return &NotConfiguredFiler{logger: observability.GetLogger()}
}

func (f *NotConfiguredFiler) FileRequest(_ context.Context, req ApprovalFileRequest) (*ApprovalFileResult, error) {
	f.logger.Warn("llmchat: file_approval_request called but ApprovalFiler is not wired",
		"user_id", req.CallerUserID, "type", req.Type)
	return nil, ErrNotWired
}

// NotConfiguredReader is the ApprovalReader used until the approval engine
// adapter lands.
type NotConfiguredReader struct{ logger *slog.Logger }

func NewNotConfiguredReader() *NotConfiguredReader {
	return &NotConfiguredReader{logger: observability.GetLogger()}
}

func (r *NotConfiguredReader) ListForUser(_ context.Context, caller Caller) ([]ApprovalRequestStatus, error) {
	r.logger.Warn("llmchat: get_approval_requests called but ApprovalReader is not wired",
		"user_id", caller.UserID)
	return nil, ErrNotWired
}

// ErrAttachmentRejected is returned when an attachment fails validation
// (not an image, too large, foreign upload, missing file).
var ErrAttachmentRejected = errors.New("tệp đính kèm không hợp lệ")

// MaxImageAttachments bounds images per message.
const MaxImageAttachments = 2

// AttachmentLoader reads one uploaded image as a data URL for the vision
// path. Ownership is enforced by the implementation (uploader or admin).
type AttachmentLoader interface {
	LoadImageDataURL(ctx context.Context, caller Caller, assetID uint) (string, error)
}

// ModelRouter resolves the admin-configured model for the interactive chat
// route (settings mimo.model_chat; empty = runtime default = flash).
type ModelRouter interface {
	ChatModel(ctx context.Context) string
}

// ModelRouterFunc adapts a function onto ModelRouter.
type ModelRouterFunc func(ctx context.Context) string

// ChatModel implements ModelRouter.
func (f ModelRouterFunc) ChatModel(ctx context.Context) string { return f(ctx) }

// WalletInfoSource is the narrow read seam over the wallet service for the
// admin-only get_wallet_balance tool (adapter in wallet_source.go).
type WalletInfoSource interface {
	WalletBalance(ctx context.Context) (available, pendingIn, pendingOut, limbo int64, err error)
}

// ScopeResolver materialises the partner data scope (visible employee/project
// ID sets) for tool queries. Admin scopes resolve to empty sets.
type ScopeResolver interface {
	ResolvePartnerScope(ctx context.Context, partnerID uint) (employeeIDs, projectIDs []uint, err error)
}

// AuditSink is the seam onto the existing audit pipeline (audit:log:write →
// audit_logs). One entry per tool call — who / what / when / result.
type AuditSink interface {
	LogChatToolCall(ctx context.Context, userID uint, toolName, outcome, message, ip, ua string) error
}

// Limiter is the per-user message rate limit seam (Redis fixed window in
// production, in-memory in tests). A non-nil error means the limiter itself
// failed — callers log and fail open.
type Limiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}
