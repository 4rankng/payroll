package persistence

import (
	"context"
	"time"

	"api-server/internal/domain"
)

type chatToolCallRow struct {
	ID             uint      `gorm:"column:id;primaryKey"`
	ConversationID uint      `gorm:"column:conversation_id"`
	MessageID      *uint     `gorm:"column:message_id"`
	UserID         uint      `gorm:"column:user_id"`
	ToolName       string    `gorm:"column:tool_name"`
	Arguments      string    `gorm:"column:arguments"`
	Outcome        string    `gorm:"column:outcome"`
	ResultSummary  string    `gorm:"column:result_summary"`
	DurationMS     int64     `gorm:"column:duration_ms"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (chatToolCallRow) TableName() string { return "chat_tool_calls" }

func (r chatToolCallRow) toDomain() *domain.ChatToolCall {
	return &domain.ChatToolCall{
		ID:             r.ID,
		ConversationID: r.ConversationID,
		MessageID:      r.MessageID,
		UserID:         r.UserID,
		ToolName:       r.ToolName,
		Arguments:      r.Arguments,
		Outcome:        r.Outcome,
		ResultSummary:  r.ResultSummary,
		DurationMS:     r.DurationMS,
		CreatedAt:      r.CreatedAt,
	}
}

// ChatToolCallRepository implements domain.ChatToolCallRepository.
type ChatToolCallRepository struct {
	*BaseRepository
}

func NewChatToolCallRepository(db *Database) *ChatToolCallRepository {
	return &ChatToolCallRepository{BaseRepository: NewBaseRepository(db)}
}

func (r *ChatToolCallRepository) Create(ctx context.Context, call *domain.ChatToolCall) error {
	row := chatToolCallRow{
		ConversationID: call.ConversationID,
		MessageID:      call.MessageID,
		UserID:         call.UserID,
		ToolName:       call.ToolName,
		Arguments:      call.Arguments,
		Outcome:        call.Outcome,
		ResultSummary:  call.ResultSummary,
		DurationMS:     call.DurationMS,
	}
	if err := r.dbForContext(ctx).Create(&row).Error; err != nil {
		return domain.NewInternalError("không thể lưu lịch sử công cụ chat", err)
	}
	call.ID = row.ID
	call.CreatedAt = row.CreatedAt
	return nil
}

func (r *ChatToolCallRepository) AttachMessage(ctx context.Context, ids []uint, messageID uint) error {
	if len(ids) == 0 {
		return nil
	}
	err := r.dbForContext(ctx).Model(&chatToolCallRow{}).
		Where("id IN ?", ids).
		Update("message_id", messageID).Error
	if err != nil {
		return domain.NewInternalError("không thể liên kết lịch sử công cụ chat", err)
	}
	return nil
}

func (r *ChatToolCallRepository) ListByMessage(ctx context.Context, messageID uint) ([]*domain.ChatToolCall, error) {
	var rows []chatToolCallRow
	err := r.dbForContext(ctx).Where("message_id = ?", messageID).Order("id ASC").Find(&rows).Error
	if err != nil {
		return nil, domain.NewInternalError("không thể đọc lịch sử công cụ chat", err)
	}
	out := make([]*domain.ChatToolCall, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].toDomain())
	}
	return out, nil
}

func (r *ChatToolCallRepository) ListByConversation(ctx context.Context, conversationID uint, limit int) ([]*domain.ChatToolCall, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	var rows []chatToolCallRow
	err := r.dbForContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("id DESC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, domain.NewInternalError("không thể đọc lịch sử công cụ chat", err)
	}
	out := make([]*domain.ChatToolCall, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		out = append(out, rows[i].toDomain())
	}
	return out, nil
}
