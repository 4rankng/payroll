package llmchat

import (
	"context"
	"encoding/json"

	asynqinfra "api-server/internal/infra/asynq"
	"api-server/internal/pkg/clock"
)

// asynqAuditSink lands tool-call audit entries in audit_logs through the
// existing audit:log:write pipeline (the same worker the audit event handler
// uses). Action/entity are free-text columns, so chat tool calls carry their
// own action without touching the shared audit enums.
type asynqAuditSink struct{ enq asynqinfra.AuditEnqueuer }

// NewAuditSink builds the sink. Returns nil for a nil enqueuer (tests wire a
// fake AuditSink instead).
func NewAuditSink(enq asynqinfra.AuditEnqueuer) AuditSink {
	if enq == nil {
		return nil
	}
	return asynqAuditSink{enq: enq}
}

func (a asynqAuditSink) LogChatToolCall(ctx context.Context, userID uint, toolName, outcome, message, ip, ua string) error {
	meta, err := json.Marshal(map[string]string{"tool": toolName, "outcome": outcome})
	if err != nil {
		meta = []byte(`{}`)
	}
	return a.enq.EnqueueAuditLogWrite(asynqinfra.AuditLogWritePayload{
		UserID:       userID,
		Action:       "CHAT_TOOL_CALL",
		EntityType:   "chat_tool_call",
		Message:      message,
		IPAddress:    ip,
		UserAgent:    ua,
		MetadataJSON: string(meta),
		CreatedAt:    clock.Now(),
	})
}
