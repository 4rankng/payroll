package llmchat

import (
	"context"
	"log/slog"
	"strings"

	"api-server/internal/domain"
	llmport "api-server/internal/infra/llm"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
)

// LLMPort is the model seam — satisfied by app/services/llm.Service. Kept as
// a narrow interface so tests script model responses without HTTP.
type LLMPort interface {
	Chat(ctx context.Context, userID uint, req llmport.ChatRequest) (*llmport.ChatResponse, error)
}

// Service owns the chat use cases: conversations, messages, and turns.
// Ownership is enforced HERE on every conversation/message access (Casbin is
// role×path only and cannot express per-record ownership).
type Service struct {
	llm        LLMPort
	convRepo   domain.ChatConversationRepository
	msgRepo    domain.ChatMessageRepository
	toolCalls  domain.ChatToolCallRepository
	timingRepo domain.ChatTurnTimingRepository
	registry   *ToolRegistry
	executor   *ToolExecutor
	msgLimit   Limiter
	roleGate   RoleGate
	images     AttachmentLoader
	models     ModelRouter
	logger     *slog.Logger
}

// Deps bundles the service dependencies (wired at bootstrap).
type Deps struct {
	LLM        LLMPort
	Convs      domain.ChatConversationRepository
	Messages   domain.ChatMessageRepository
	ToolCalls  domain.ChatToolCallRepository
	Timings    domain.ChatTurnTimingRepository
	Store      domain.ChatToolReadStore
	Wallet     WalletInfoSource
	Filer      ApprovalFiler
	Reader     ApprovalReader
	Scope      ScopeResolver
	Audit      AuditSink
	MsgLimiter Limiter
	RoleGate   RoleGate
	Images     AttachmentLoader
	Models     ModelRouter
}

// NewService constructs the chat service with the full tool manifest.
func NewService(deps Deps) *Service {
	registry := NewToolRegistry()
	executor := NewToolExecutor(registry, deps.Store, deps.Wallet, deps.Filer, deps.Reader, deps.Scope, deps.ToolCalls, deps.Audit)
	if deps.Filer == nil {
		deps.Filer = NewNotConfiguredFiler()
	}
	return &Service{
		llm:        deps.LLM,
		convRepo:   deps.Convs,
		msgRepo:    deps.Messages,
		toolCalls:  deps.ToolCalls,
		timingRepo: deps.Timings,
		registry:   registry,
		executor:   executor,
		msgLimit:   deps.MsgLimiter,
		roleGate:   deps.RoleGate,
		images:     deps.Images,
		models:     deps.Models,
		logger:     observability.GetLogger(),
	}
}

// CreateConversation opens a new conversation for the caller.
func (s *Service) CreateConversation(ctx context.Context, caller Caller, title string) (*domain.ChatConversation, error) {

	// Creation backstop: the same per-user fixed window as messages. A
	// runaway client loop must not be able to mint thousands of rows.
	if s.msgLimit != nil {
		allowed, err := s.msgLimit.Allow(ctx, callerKey(caller))
		if err == nil && !allowed {
			return nil, ErrRateLimited
		}
	}
	if !RoleAllowed(caller.Role) {
		return nil, ErrForbiddenRole
	}

	if s.roleGate != nil && !s.roleGate.ChatEnabled(ctx, caller.Role) {
		return nil, ErrChatDisabled
	}
	conv := &domain.ChatConversation{
		UserID: caller.UserID,
		Role:   caller.Role,
		Title:  truncateRunes(strings.TrimSpace(title), 255),
	}
	if err := s.convRepo.Create(ctx, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

// ListConversations returns the caller's own conversations (paginated).
func (s *Service) ListConversations(ctx context.Context, caller Caller, offset, limit int) ([]*domain.ChatConversation, int64, error) {
	if !RoleAllowed(caller.Role) {
		return nil, 0, ErrForbiddenRole
	}

	if s.roleGate != nil && !s.roleGate.ChatEnabled(ctx, caller.Role) {
		return nil, 0, ErrChatDisabled
	}
	return s.convRepo.ListByUser(ctx, caller.UserID, offset, limit)
}

// ConversationView is one conversation plus a page of its messages and the
// recent tool-call trail.
type ConversationView struct {
	Conversation  *domain.ChatConversation
	Messages      []*domain.ChatMessage
	TotalMessages int64
	ToolCalls     []*domain.ChatToolCall
}

// GetConversation returns a conversation with one page of messages. Every
// access passes the ownership check: a non-admin can only ever see their own
// conversations — a foreign ID reads exactly like a missing one (no existence
// leak).
func (s *Service) GetConversation(ctx context.Context, caller Caller, id uint, msgOffset, msgLimit int) (*ConversationView, error) {
	if !RoleAllowed(caller.Role) {
		return nil, ErrForbiddenRole
	}

	if s.roleGate != nil && !s.roleGate.ChatEnabled(ctx, caller.Role) {
		return nil, ErrChatDisabled
	}
	conv, err := s.ownedConversation(ctx, caller, id)
	if err != nil {
		return nil, err
	}
	messages, total, err := s.msgRepo.ListByConversation(ctx, id, msgOffset, msgLimit)
	if err != nil {
		return nil, err
	}
	view := &ConversationView{Conversation: conv, Messages: messages, TotalMessages: total}
	view.ToolCalls, err = s.toolCalls.ListByConversation(ctx, id, 50)
	if err != nil {
		// Tool trail is auxiliary — degrade, don't fail the poll.
		s.logger.Warn("llmchat: failed to read tool call trail", "error", err)
	}
	return view, nil
}

// SendMessage runs one assistant turn: rate limit → ownership → persist the
// user message → orchestrate model + tools → persist the assistant answer.
func (s *Service) SendMessage(ctx context.Context, caller Caller, conversationID uint, content string, attachmentIDs []uint) (*TurnResult, error) {
	if !RoleAllowed(caller.Role) {
		return nil, ErrForbiddenRole
	}

	if s.roleGate != nil && !s.roleGate.ChatEnabled(ctx, caller.Role) {
		return nil, ErrChatDisabled
	}

	// Per-user fixed window on message POST (shared Redis limiter infra).
	if s.msgLimit != nil {
		allowed, err := s.msgLimit.Allow(ctx, callerKey(caller))
		if err != nil {
			// Limiter malfunction must not take chat down — log and fail open.
			s.logger.Warn("llmchat: message limiter failed, failing open", "error", err)
		} else if !allowed {
			return nil, ErrRateLimited
		}
	}

	conv, err := s.ownedConversation(ctx, caller, conversationID)
	if err != nil {
		return nil, err
	}

	text := strings.TrimSpace(content)
	if text == "" {
		return nil, domain.NewValidationError("Tin nhắn không được để trống")
	}
	if len([]rune(text)) > MaxMessageContent {
		return nil, domain.NewValidationError("Tin nhắn quá dài — tối đa 4000 ký tự")
	}

	userMsg := &domain.ChatMessage{
		ConversationID: conv.ID,
		UserID:         caller.UserID,
		Role:           domain.ChatMessageRoleUser,
		Content:        text,
	}
	if err := s.msgRepo.Create(ctx, userMsg); err != nil {
		return nil, err
	}
	if err := s.convRepo.RecordMessage(ctx, conv.ID, clock.Now()); err != nil {
		s.logger.Warn("llmchat: failed to refresh conversation counters", "error", err)
	}
	if conv.Title == "" {
		// First message seeds the conversation title (list readability).
		conv.Title = truncateRunes(text, 255)
		if err := s.convRepo.UpdateTitle(ctx, conv.ID, conv.Title); err != nil {
			s.logger.Warn("llmchat: failed to seed conversation title", "error", err)
		}
	}

	// Screenshot vision: <=2 validated images ride the CURRENT turn only —
	// history replays as text, so storage and prompts stay bounded.
	var images []llmport.ContentPart
	if len(attachmentIDs) > 0 {
		if len(attachmentIDs) > MaxImageAttachments {
			return nil, domain.NewValidationError("tối đa 2 ảnh mỗi tin nhắn")
		}
		if s.images == nil {
			return nil, ErrNotWired
		}
		for _, id := range attachmentIDs {
			dataURL, err := s.images.LoadImageDataURL(ctx, caller, id)
			if err != nil {
				return nil, err
			}
			images = append(images, llmport.ImagePart(dataURL))
		}
	}

	return s.runTurn(ctx, caller, conv, text, images)
}

// ownedConversation loads a conversation and enforces per-record ownership.
// Non-admin callers get NOT_FOUND for foreign IDs — identical to a missing
// conversation, so sequential IDs cannot be probed for existence.
func (s *Service) ownedConversation(ctx context.Context, caller Caller, id uint) (*domain.ChatConversation, error) {
	conv, err := s.convRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if caller.Role != RoleAdmin && conv.UserID != caller.UserID {
		return nil, domain.NewNotFoundError("không tìm thấy hội thoại")
	}
	return conv, nil
}

func callerKey(caller Caller) string {
	return itoa(int(caller.UserID))
}
