package domain

import (
	"context"
	"time"
)

// ChatConversation is one assistant chat session. Ownership is per-user: the
// service layer refuses reads for any non-admin whose user_id differs from the
// conversation's, because Casbin is role×path only and cannot express
// per-record ownership (red-team constraint).
type ChatConversation struct {
	ID           uint      `json:"id"`
	UserID       uint      `json:"user_id"`
	Role         string    `json:"role"`
	Title        string    `json:"title"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ChatConversationRepository is the persistence port for chat conversations.
type ChatConversationRepository interface {
	Create(ctx context.Context, conv *ChatConversation) error
	GetByID(ctx context.Context, id uint) (*ChatConversation, error)
	// ListByUser returns one page of the user's conversations (newest first)
	// and the total count for pagination.
	ListByUser(ctx context.Context, userID uint, offset, limit int) ([]*ChatConversation, int64, error)
	// RecordMessage appends one message to the conversation: increments
	// message_count and refreshes updated_at (list ordering key).
	RecordMessage(ctx context.Context, id uint, at time.Time) error
	// UpdateTitle stores the conversation title (seeded from the first user
	// message when the caller did not provide one).
	UpdateTitle(ctx context.Context, id uint, title string) error
}
