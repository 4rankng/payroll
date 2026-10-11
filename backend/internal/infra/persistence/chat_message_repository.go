package persistence

import (
	"context"
	"errors"
	"time"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

type chatMessageRow struct {
	ID             uint      `gorm:"column:id;primaryKey"`
	ConversationID uint      `gorm:"column:conversation_id"`
	UserID         uint      `gorm:"column:user_id"`
	Role           string    `gorm:"column:role"`
	Content        string    `gorm:"column:content"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (chatMessageRow) TableName() string { return "chat_messages" }

func (r chatMessageRow) toDomain() *domain.ChatMessage {
	return &domain.ChatMessage{
		ID:             r.ID,
		ConversationID: r.ConversationID,
		UserID:         r.UserID,
		Role:           r.Role,
		Content:        r.Content,
		CreatedAt:      r.CreatedAt,
	}
}

// ChatMessageRepository implements domain.ChatMessageRepository.
type ChatMessageRepository struct {
	*BaseRepository
}

func NewChatMessageRepository(db *Database) *ChatMessageRepository {
	return &ChatMessageRepository{BaseRepository: NewBaseRepository(db)}
}

func (r *ChatMessageRepository) Create(ctx context.Context, msg *domain.ChatMessage) error {
	row := chatMessageRow{
		ConversationID: msg.ConversationID,
		UserID:         msg.UserID,
		Role:           msg.Role,
		Content:        msg.Content,
	}
	if err := r.dbForContext(ctx).Create(&row).Error; err != nil {
		return domain.NewInternalError("không thể lưu tin nhắn chat", err)
	}
	msg.ID = row.ID
	msg.CreatedAt = row.CreatedAt
	return nil
}

func (r *ChatMessageRepository) GetByID(ctx context.Context, id uint) (*domain.ChatMessage, error) {
	var row chatMessageRow
	err := r.dbForContext(ctx).First(&row, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("không tìm thấy tin nhắn")
		}
		return nil, domain.NewInternalError("không thể đọc tin nhắn chat", err)
	}
	return row.toDomain(), nil
}

// ListByConversation returns one page of messages ordered newest-first at the
// page level (page 1 = most recent turns, the polling default) and reversed to
// chronological order within the page for display. Unbounded thread fetches
// are refused by construction — callers always pass a page size.
func (r *ChatMessageRepository) ListByConversation(ctx context.Context, conversationID uint, offset, limit int) ([]*domain.ChatMessage, int64, error) {
	q := r.dbForContext(ctx).Model(&chatMessageRow{}).Where("conversation_id = ?", conversationID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, domain.NewInternalError("không thể đếm tin nhắn chat", err)
	}
	var rows []chatMessageRow
	err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, 0, domain.NewInternalError("không thể liệt kê tin nhắn chat", err)
	}
	// Reverse to chronological order within the page.
	out := make([]*domain.ChatMessage, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		out = append(out, rows[i].toDomain())
	}
	return out, total, nil
}

func (r *ChatMessageRepository) RecentTail(ctx context.Context, conversationID uint, limit int) ([]*domain.ChatMessage, error) {
	if limit < 1 {
		limit = 1
	}
	var rows []chatMessageRow
	err := r.dbForContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("id DESC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, domain.NewInternalError("không thể đọc lịch sử hội thoại", err)
	}
	out := make([]*domain.ChatMessage, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		out = append(out, rows[i].toDomain())
	}
	return out, nil
}
