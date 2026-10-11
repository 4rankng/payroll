package domain

import (
	"context"
	"time"
)

// Outcomes for one assistant tool call. The executor records every call —
// including refusals — so the manifest contract is auditable.
const (
	ChatToolOutcomeOK        = "ok"
	ChatToolOutcomeError     = "error"
	ChatToolOutcomeUnlisted  = "unlisted"
	ChatToolOutcomeForbidden = "forbidden"
	ChatToolOutcomeNotWired  = "not_wired"
)

// ChatToolCall is one executed (or refused) assistant tool call — the
// structured who/what/when/result audit trail. The actor-facing summary lands
// in audit_logs through the existing audit:log:write pipeline.
type ChatToolCall struct {
	ID             uint      `json:"id"`
	ConversationID uint      `json:"conversation_id"`
	MessageID      *uint     `json:"message_id"`
	UserID         uint      `json:"user_id"`
	ToolName       string    `json:"tool_name"`
	Arguments      string    `json:"arguments"`
	Outcome        string    `json:"outcome"`
	ResultSummary  string    `json:"result_summary"`
	DurationMS     int64     `json:"duration_ms"`
	CreatedAt      time.Time `json:"created_at"`
}

// ChatToolCallRepository is the persistence port for the tool-call audit trail.
type ChatToolCallRepository interface {
	Create(ctx context.Context, call *ChatToolCall) error
	// AttachMessage links executed calls to the assistant message they fed.
	AttachMessage(ctx context.Context, ids []uint, messageID uint) error
	// ListByMessage returns the tool calls that fed one assistant message.
	ListByMessage(ctx context.Context, messageID uint) ([]*ChatToolCall, error)
	// ListByConversation returns the tool calls of one conversation, oldest
	// first (bounded — the handler caps the page).
	ListByConversation(ctx context.Context, conversationID uint, limit int) ([]*ChatToolCall, error)
}
