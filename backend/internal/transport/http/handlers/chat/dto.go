package chat

import (
	"time"

	"api-server/internal/domain"
)

// Request/response shapes for the chat API. Kept local to the transport
// package — the chat surface is self-contained.

type createConversationRequest struct {
	Title string `json:"title"`
}

type sendMessageRequest struct {
	Content string `json:"content"`
}

type conversationResponse struct {
	ID           uint      `json:"id"`
	UserID       uint      `json:"user_id"`
	Role         string    `json:"role"`
	Title        string    `json:"title"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type messageResponse struct {
	ID        uint      `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type toolCallResponse struct {
	ID            uint      `json:"id"`
	MessageID     *uint     `json:"message_id"`
	ToolName      string    `json:"tool_name"`
	Arguments     string    `json:"arguments"`
	Outcome       string    `json:"outcome"`
	ResultSummary string    `json:"result_summary"`
	DurationMS    int64     `json:"duration_ms"`
	CreatedAt     time.Time `json:"created_at"`
}

type turnTimingResponse struct {
	DurationMS int64 `json:"duration_ms"`
	ModelMS    int64 `json:"model_ms"`
	ToolMS     int64 `json:"tool_ms"`
	RoundTrips int   `json:"round_trips"`
	ToolCalls  int   `json:"tool_calls"`
	TimedOut   bool  `json:"timed_out"`
}

type conversationDetailResponse struct {
	Conversation  conversationResponse `json:"conversation"`
	Messages      []messageResponse    `json:"messages"`
	ToolCalls     []toolCallResponse   `json:"tool_calls"`
	TotalMessages int64                `json:"total_messages"`
}

type turnResponse struct {
	Message   messageResponse    `json:"message"`
	ToolCalls []toolCallResponse `json:"tool_calls"`
	Timing    turnTimingResponse `json:"timing"`
}

func newConversationResponse(conv *domain.ChatConversation) conversationResponse {
	return conversationResponse{
		ID:           conv.ID,
		UserID:       conv.UserID,
		Role:         conv.Role,
		Title:        conv.Title,
		MessageCount: conv.MessageCount,
		CreatedAt:    conv.CreatedAt,
		UpdatedAt:    conv.UpdatedAt,
	}
}

func newMessageResponse(msg *domain.ChatMessage) messageResponse {
	return messageResponse{
		ID:        msg.ID,
		Role:      msg.Role,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	}
}

func newToolCallResponse(call *domain.ChatToolCall) toolCallResponse {
	return toolCallResponse{
		ID:            call.ID,
		MessageID:     call.MessageID,
		ToolName:      call.ToolName,
		Arguments:     call.Arguments,
		Outcome:       call.Outcome,
		ResultSummary: call.ResultSummary,
		DurationMS:    call.DurationMS,
		CreatedAt:     call.CreatedAt,
	}
}
