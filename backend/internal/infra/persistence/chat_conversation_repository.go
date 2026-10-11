package persistence

import (
	"context"
	"errors"
	"time"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

type chatConversationRow struct {
	ID           uint      `gorm:"column:id;primaryKey"`
	UserID       uint      `gorm:"column:user_id"`
	Role         string    `gorm:"column:role"`
	Title        string    `gorm:"column:title"`
	MessageCount int       `gorm:"column:message_count"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (chatConversationRow) TableName() string { return "chat_conversations" }

func (r chatConversationRow) toDomain() *domain.ChatConversation {
	return &domain.ChatConversation{
		ID:           r.ID,
		UserID:       r.UserID,
		Role:         r.Role,
		Title:        r.Title,
		MessageCount: r.MessageCount,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

// ChatConversationRepository implements domain.ChatConversationRepository.
type ChatConversationRepository struct {
	*BaseRepository
}

func NewChatConversationRepository(db *Database) *ChatConversationRepository {
	return &ChatConversationRepository{BaseRepository: NewBaseRepository(db)}
}

func (r *ChatConversationRepository) Create(ctx context.Context, conv *domain.ChatConversation) error {
	row := chatConversationRow{
		UserID:       conv.UserID,
		Role:         conv.Role,
		Title:        conv.Title,
		MessageCount: conv.MessageCount,
	}
	if err := r.dbForContext(ctx).Create(&row).Error; err != nil {
		return domain.NewInternalError("không thể tạo hội thoại chat", err)
	}
	conv.ID = row.ID
	conv.CreatedAt = row.CreatedAt
	conv.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *ChatConversationRepository) GetByID(ctx context.Context, id uint) (*domain.ChatConversation, error) {
	var row chatConversationRow
	err := r.dbForContext(ctx).First(&row, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("không tìm thấy hội thoại")
		}
		return nil, domain.NewInternalError("không thể đọc hội thoại chat", err)
	}
	return row.toDomain(), nil
}

func (r *ChatConversationRepository) ListByUser(ctx context.Context, userID uint, offset, limit int) ([]*domain.ChatConversation, int64, error) {
	q := r.dbForContext(ctx).Model(&chatConversationRow{}).Where("user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, domain.NewInternalError("không thể đếm hội thoại chat", err)
	}
	var rows []chatConversationRow
	err := q.Order("updated_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, 0, domain.NewInternalError("không thể liệt kê hội thoại chat", err)
	}
	out := make([]*domain.ChatConversation, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].toDomain())
	}
	return out, total, nil
}

func (r *ChatConversationRepository) RecordMessage(ctx context.Context, id uint, at time.Time) error {
	err := r.dbForContext(ctx).Model(&chatConversationRow{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"message_count": gorm.Expr("message_count + 1"),
			"updated_at":    at,
		}).Error
	if err != nil {
		return domain.NewInternalError("không thể cập nhật hội thoại chat", err)
	}
	return nil
}

func (r *ChatConversationRepository) UpdateTitle(ctx context.Context, id uint, title string) error {
	err := r.dbForContext(ctx).Model(&chatConversationRow{}).
		Where("id = ?", id).
		Update("title", title).Error
	if err != nil {
		return domain.NewInternalError("không thể cập nhật tiêu đề hội thoại", err)
	}
	return nil
}
