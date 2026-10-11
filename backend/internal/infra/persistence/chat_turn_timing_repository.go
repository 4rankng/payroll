package persistence

import (
	"context"
	"time"

	"api-server/internal/domain"
)

type chatTurnTimingRow struct {
	ID             uint      `gorm:"column:id;primaryKey"`
	ConversationID uint      `gorm:"column:conversation_id"`
	UserID         uint      `gorm:"column:user_id"`
	Role           string    `gorm:"column:role"`
	Model          string    `gorm:"column:model"`
	RoundTrips     int       `gorm:"column:round_trips"`
	ToolCalls      int       `gorm:"column:tool_calls"`
	TimedOut       bool      `gorm:"column:timed_out"`
	DurationMS     int64     `gorm:"column:duration_ms"`
	ModelMS        int64     `gorm:"column:model_ms"`
	ToolMS         int64     `gorm:"column:tool_ms"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (chatTurnTimingRow) TableName() string { return "chat_turn_timings" }

// ChatTurnTimingRepository implements domain.ChatTurnTimingRepository.
type ChatTurnTimingRepository struct {
	*BaseRepository
}

func NewChatTurnTimingRepository(db *Database) *ChatTurnTimingRepository {
	return &ChatTurnTimingRepository{BaseRepository: NewBaseRepository(db)}
}

func (r *ChatTurnTimingRepository) Create(ctx context.Context, t *domain.ChatTurnTiming) error {
	row := chatTurnTimingRow{
		ConversationID: t.ConversationID,
		UserID:         t.UserID,
		Role:           t.Role,
		Model:          t.Model,
		RoundTrips:     t.RoundTrips,
		ToolCalls:      t.ToolCalls,
		TimedOut:       t.TimedOut,
		DurationMS:     t.DurationMS,
		ModelMS:        t.ModelMS,
		ToolMS:         t.ToolMS,
	}
	if err := r.dbForContext(ctx).Create(&row).Error; err != nil {
		return domain.NewInternalError("không thể lưu số liệu hiệu năng chat", err)
	}
	t.ID = row.ID
	t.CreatedAt = row.CreatedAt
	return nil
}
