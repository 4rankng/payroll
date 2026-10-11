package domain

import (
	"context"
	"time"
)

// Chat message roles. Intermediate tool traffic is not persisted as messages;
// see ChatToolCall.
const (
	ChatMessageRoleUser      = "user"
	ChatMessageRoleAssistant = "assistant"
)

// ChatMessage is one persisted turn of a conversation.
type ChatMessage struct {
	ID             uint      `json:"id"`
	ConversationID uint      `json:"conversation_id"`
	UserID         uint      `json:"user_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

// ChatMessageRepository is the persistence port for chat messages.
type ChatMessageRepository interface {
	Create(ctx context.Context, msg *ChatMessage) error
	GetByID(ctx context.Context, id uint) (*ChatMessage, error)
	// ListByConversation returns one page of messages (oldest first within the
	// page, pages ordered newest-first so polling clients see recent turns)
	// and the total count.
	ListByConversation(ctx context.Context, conversationID uint, offset, limit int) ([]*ChatMessage, int64, error)
	// RecentTail returns the most recent messages of a conversation (newest
	// last) up to limit — the prompt context window.
	RecentTail(ctx context.Context, conversationID uint, limit int) ([]*ChatMessage, error)
}
