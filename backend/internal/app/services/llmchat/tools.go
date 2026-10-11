package llmchat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"api-server/internal/domain"
	llmport "api-server/internal/infra/llm"
	"api-server/internal/infra/observability"
	auditctx "api-server/internal/pkg/context"
)

// Tool kinds. Only "read" tools exist alongside exactly one "action" tool
// (file_approval_request) behind the ApprovalFiler port. NO other mutation
// tool can be added without changing this file — that is the point.
const (
	ToolKindRead   = "read"
	ToolKindAction = "action"
)

// ToolDef is one manifest entry: the model-facing declaration plus the Go
// executor that re-checks role and scope at call time.
type ToolDef struct {
	Name        string
	Description string
	Parameters  json.RawMessage // JSON Schema, byte-stable
	Kind        string
	Roles       []string // roles allowed to call; the manifest is per-role
	topic       string   // fallback deep-link topic for refusals
	run         func(ctx context.Context, tc *ToolContext, args json.RawMessage) (string, error)
}

// ToolContext is everything one tool invocation may touch. Scope is derived
// from the caller identity at turn start — tool arguments can never widen it.
type ToolContext struct {
	Caller Caller
	Scope  domain.ChatDataScope
	Store  domain.ChatToolReadStore
	Wallet WalletInfoSource
	Filer  ApprovalFiler
	Reader ApprovalReader
}

// ToolRegistry holds the manifest in fixed order (byte-stable marshalling).
type ToolRegistry struct {
	defs   []ToolDef
	byName map[string]*ToolDef
}

// NewToolRegistry builds the registry with the full manifest. Order is fixed
// and load-bearing for prompt-cache stability.
func NewToolRegistry() *ToolRegistry {
	defs := []ToolDef{
		{
			Name:        "get_timesheet_summary",
			Description: "Tổng hợp bảng công theo phạm vi của người dùng (số dòng theo trạng thái duyệt/thanh toán, tổng tiền). Tham số month dạng YYYY-MM, bỏ trống = 30 ngày gần nhất.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"month":{"type":"string","description":"Tháng YYYY-MM"}},"additionalProperties":false}`),
			Kind:        ToolKindRead,
			Roles:       []string{RoleAdmin, RolePartner},
			topic:       "timesheet",
			run:         runTimesheetSummary,
		},
		{
			Name:        "get_import_status",
			Description: "Trạng thái các lần import bảng công (file Excel) gần đây của người dùng, kèm tóm tắt lỗi của lần import thất bại mới nhất.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"limit":{"type":"integer","description":"Số lần import tối đa (mặc định 5)"}},"additionalProperties":false}`),
			Kind:        ToolKindRead,
			Roles:       []string{RoleAdmin, RolePartner},
			topic:       "import",
			run:         runImportStatus,
		},
		{
			Name:        "get_approval_requests",
			Description: "Trạng thái các yêu cầu phê duyệt của người dùng (đang chờ, đã duyệt, cần thử lại).",
			Parameters:  json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
			Kind:        ToolKindRead,
			Roles:       []string{RoleAdmin, RolePartner},
			topic:       "approval",
			run:         runApprovalRequests,
		},
		{
			Name:        "get_advance_summary",
			Description: "Tổng hợp ứng lương theo phạm vi của người dùng: quỹ lương, hạn mức, số tiền đã yêu cầu/đã chi, số yêu cầu theo trạng thái. Tham số month dạng YYYY-MM.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"month":{"type":"string","description":"Tháng YYYY-MM"}},"additionalProperties":false}`),
			Kind:        ToolKindRead,
			Roles:       []string{RoleAdmin, RolePartner},
			topic:       "advance",
			run:         runAdvanceSummary,
		},
		{
			Name:        "get_wallet_balance",
			Description: "Số dư ví chi hộ hiện tại (khả dụng, đang chờ vào/ra, chưa đối soát). Chỉ quản trị viên.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
			Kind:        ToolKindRead,
			Roles:       []string{RoleAdmin},
			topic:       "wallet",
			run:         runWalletBalance,
		},
		{
			Name:        "lookup_import_error",
			Description: "Tra cứu mã lỗi import bảng công: giải thích nguyên nhân và cách xử lý. Tham số keyword là từ khóa hoặc nội dung lỗi.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"keyword":{"type":"string","description":"Từ khóa hoặc nội dung lỗi cần tra"}},"additionalProperties":false}`),
			Kind:        ToolKindRead,
			Roles:       []string{RoleAdmin, RolePartner},
			topic:       "import",
			run:         runLookupImportError,
		},
		{
			Name:        "file_approval_request",
			Description: "Ghi nhận một yêu cầu phê duyệt mới (chỉ tạo yêu cầu chờ duyệt — không tự thực hiện thao tác). Tham số type là loại yêu cầu, summary là nội dung tóm tắt.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"type":{"type":"string","description":"Loại yêu cầu: send_sao_ke_email | send_payroll_email | send_reminder | void_timesheet | other"},"summary":{"type":"string","description":"Tóm tắt nội dung yêu cầu"},"timesheet_id":{"type":"integer","description":"ID timesheet cần void (bắt buộc khi type=void_timesheet)"},"report_at_date":{"type":"string","description":"Ngày sao kê YYYY-MM-DD (khi type=send_sao_ke_email / send_payroll_email)"},"project_id":{"type":"integer","description":"ID dự án (khi type=send_reminder)"},"remind_for_date":{"type":"string","description":"Ngày nhắc YYYY-MM-DD (khi type=send_reminder)"}},"required":["type"],"additionalProperties":false}`),
			Kind:        ToolKindAction,
			Roles:       []string{RoleAdmin, RolePartner},
			topic:       "approval",
			run:         runFileApprovalRequest,
		},
	}
	byName := make(map[string]*ToolDef, len(defs))
	for i := range defs {
		byName[defs[i].Name] = &defs[i]
	}
	return &ToolRegistry{defs: defs, byName: byName}
}

// Manifest returns the model-facing tool declarations for one role, in fixed
// order. The returned slice is byte-stable across calls for the same role.
func (r *ToolRegistry) Manifest(role string) []llmport.Tool {
	tools := []llmport.Tool{}
	for i := range r.defs {
		def := &r.defs[i]
		if !containsString(def.Roles, role) {
			continue
		}
		tools = append(tools, llmport.Tool{
			Type:     "function",
			Function: &llmport.FunctionDef{Name: def.Name, Description: def.Description, Parameters: def.Parameters},
		})
	}
	return tools
}

// Names returns the manifest names for one role (for tests and audit copy).
func (r *ToolRegistry) Names(role string) []string {
	names := []string{}
	for i := range r.defs {
		if containsString(r.defs[i].Roles, role) {
			names = append(names, r.defs[i].Name)
		}
	}
	return names
}

// ToolExecResult is one executed (or refused) call, aligned with the model's
// ToolCall order.
type ToolExecResult struct {
	ToolCallID string
	Name       string
	Outcome    string
	Text       string // fed back to the model as the tool message
	DurationMS int64
	AuditRowID uint // chat_tool_calls row (0 when persistence failed)
}

// ToolExecutor runs tool calls with per-call role re-checks, parallel
// dispatch, and a full audit trail. Safe for concurrent use.
type ToolExecutor struct {
	registry  *ToolRegistry
	store     domain.ChatToolReadStore
	wallet    WalletInfoSource
	filer     ApprovalFiler
	reader    ApprovalReader
	scope     ScopeResolver
	toolCalls domain.ChatToolCallRepository
	audit     AuditSink
	logger    *slog.Logger
}

func NewToolExecutor(
	registry *ToolRegistry,
	store domain.ChatToolReadStore,
	wallet WalletInfoSource,
	filer ApprovalFiler,
	reader ApprovalReader,
	scope ScopeResolver,
	toolCalls domain.ChatToolCallRepository,
	audit AuditSink,
) *ToolExecutor {
	if filer == nil {
		filer = NewNotConfiguredFiler()
	}
	if reader == nil {
		reader = NewNotConfiguredReader()
	}
	return &ToolExecutor{
		registry: registry, store: store, wallet: wallet, filer: filer, reader: reader,
		scope: scope, toolCalls: toolCalls, audit: audit, logger: observability.GetLogger(),
	}
}

// Execute dispatches the model's tool calls in parallel and returns results in
// call order. Every call — including refusals — is audited (chat_tool_calls +
// audit_logs). A turn-deadline expiry aborts the whole turn with
// ErrTurnTimeout; per-call failures degrade to a Vietnamese tool result.
func (e *ToolExecutor) Execute(ctx context.Context, caller Caller, conversationID uint, calls []llmport.ToolCall, messageID *uint) ([]ToolExecResult, error) {
	scope := domain.ChatDataScope{UserID: caller.UserID, Role: caller.Role}
	if caller.Role == RolePartner && e.scope != nil {
		employees, projects, err := e.scope.ResolvePartnerScope(ctx, caller.UserID)
		if err != nil {
			return nil, fmt.Errorf("không thể xác định phạm vi dữ liệu: %w", err)
		}
		scope.EmployeeIDs = employees
		scope.ProjectIDs = projects
	}

	results := make([]ToolExecResult, len(calls))
	var wg sync.WaitGroup
	for i := range calls {
		wg.Add(1)
		go func(i int, call llmport.ToolCall) {
			defer wg.Done()
			results[i] = e.executeOne(ctx, caller, scope, conversationID, messageID, call)
		}(i, calls[i])
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
		return results, nil
	case <-ctx.Done():
		// Hard turn deadline: abort cleanly rather than ghosting a half-run
		// tool loop. In-flight calls observe ctx and finish or fail fast.
		return nil, ErrTurnTimeout
	}
}

func (e *ToolExecutor) executeOne(ctx context.Context, caller Caller, scope domain.ChatDataScope, conversationID uint, messageID *uint, call llmport.ToolCall) ToolExecResult {
	start := time.Now()
	name := call.Function.Name
	args := json.RawMessage(strings.TrimSpace(call.Function.Arguments))
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}

	outcome := domain.ChatToolOutcomeOK
	text := ""

	def := e.registry.byName[name]
	switch {
	case def == nil:
		// Unlisted tool: the model asked for a capability outside the
		// manifest. Nothing runs — graceful Vietnamese fallback only.
		outcome = domain.ChatToolOutcomeUnlisted
		text = FallbackUnsupportedAction(caller.Role, FallbackTopic(name))
	case !containsString(def.Roles, caller.Role):
		outcome = domain.ChatToolOutcomeForbidden
		text = FallbackUnsupportedAction(caller.Role, def.topic)
	default:
		result, err := def.run(ctx, &ToolContext{
			Caller: caller, Scope: scope,
			Store: e.store, Wallet: e.wallet, Filer: e.filer, Reader: e.reader,
		}, args)
		var refusal *ToolRefusalError
		switch {
		case err == nil:
			text = result
		case errors.Is(err, ErrNotWired):
			outcome = domain.ChatToolOutcomeNotWired
			text = NotWiredActionMessage(caller.Role, def.topic)
		case errors.As(err, &refusal):
			// Designed refusal (bad arguments, disallowed approval type): the
			// user-facing message is exact, the audit outcome shows the refusal.
			outcome = domain.ChatToolOutcomeForbidden
			text = refusal.Msg
		case errors.Is(err, ErrTurnTimeout):
			// Propagate: the turn is over.
			text = "Hết thời gian xử lý."
			rowID := e.persistCall(ctx, conversationID, messageID, caller, name, args, domain.ChatToolOutcomeError, text, start)
			return ToolExecResult{ToolCallID: call.ID, Name: name, Outcome: domain.ChatToolOutcomeError, Text: text, DurationMS: time.Since(start).Milliseconds(), AuditRowID: rowID}
		default:
			outcome = domain.ChatToolOutcomeError
			e.logger.Warn("llmchat: tool failed", "tool", name, "user_id", caller.UserID, "error", err)
			text = "Không lấy được dữ liệu lúc này — vui lòng thử lại sau."
		}
	}

	rowID := e.persistCall(ctx, conversationID, messageID, caller, name, args, outcome, text, start)
	return ToolExecResult{
		ToolCallID: call.ID,
		Name:       name,
		Outcome:    outcome,
		Text:       text,
		DurationMS: time.Since(start).Milliseconds(),
		AuditRowID: rowID,
	}
}

// persistCall records the structured audit row and the audit_logs entry and
// returns the audit row ID (0 when persistence failed). Failures are logged,
// never fatal to the turn.
func (e *ToolExecutor) persistCall(ctx context.Context, conversationID uint, messageID *uint, caller Caller, name string, args json.RawMessage, outcome, text string, start time.Time) uint {
	duration := time.Since(start).Milliseconds()
	var rowID uint
	if e.toolCalls != nil {
		row := &domain.ChatToolCall{
			ConversationID: conversationID,
			MessageID:      messageID,
			UserID:         caller.UserID,
			ToolName:       name,
			Arguments:      truncateRunes(string(args), 2000),
			Outcome:        outcome,
			ResultSummary:  truncateRunes(text, 480),
			DurationMS:     duration,
		}
		if err := e.toolCalls.Create(ctx, row); err != nil {
			e.logger.Warn("llmchat: failed to persist tool call audit row", "error", err)
		} else {
			rowID = row.ID
		}
	}
	if e.audit != nil {
		actor := caller.FullName
		if actor == "" {
			actor = fmt.Sprintf("người dùng #%d", caller.UserID)
		}
		message := fmt.Sprintf("%s đã dùng công cụ trợ lý %s (kết quả: %s)", actor, name, outcome)
		if err := e.audit.LogChatToolCall(ctx, caller.UserID, name, outcome, message,
			auditctx.GetIPAddress(ctx), auditctx.GetUserAgent(ctx)); err != nil {
			e.logger.Warn("llmchat: failed to enqueue audit log", "error", err)
		}
	}
	return rowID
}
