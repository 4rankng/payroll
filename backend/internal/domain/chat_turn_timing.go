package domain

import (
	"context"
	"time"
)

// ChatTurnTiming is one latency observation for a chat turn — the measurement
// behind the fast contract (p95 ≤2s direct / ≤5s tool turns). Failed and
// timed-out turns are recorded too, so the p95 reflects real production tails.
type ChatTurnTiming struct {
	ID             uint      `json:"id"`
	ConversationID uint      `json:"conversation_id"`
	UserID         uint      `json:"user_id"`
	Role           string    `json:"role"`
	Model          string    `json:"model"`
	RoundTrips     int       `json:"round_trips"`
	ToolCalls      int       `json:"tool_calls"`
	TimedOut       bool      `json:"timed_out"`
	DurationMS     int64     `json:"duration_ms"`
	ModelMS        int64     `json:"model_ms"`
	ToolMS         int64     `json:"tool_ms"`
	CreatedAt      time.Time `json:"created_at"`
}

// ChatTurnTimingRepository is the persistence port for turn latency rows.
type ChatTurnTimingRepository interface {
	Create(ctx context.Context, t *ChatTurnTiming) error
}
