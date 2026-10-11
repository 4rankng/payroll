package approval

import (
	"context"
	"time"

	"api-server/internal/domain"
	asynqinfra "api-server/internal/infra/asynq"
)

// AsynqEventWriter maps approval lifecycle events onto the existing
// audit:log:write pipeline. One audit row per transition.
type AsynqEventWriter struct {
	client asynqinfra.AuditEnqueuer
}

// NewAsynqEventWriter constructs the writer over the asynq client.
func NewAsynqEventWriter(client asynqinfra.AuditEnqueuer) *AsynqEventWriter {
	return &AsynqEventWriter{client: client}
}

// WriteApprovalEvent enqueues the audit row for one lifecycle transition.
func (w *AsynqEventWriter) WriteApprovalEvent(ctx context.Context, e domain.ApprovalRequestChangedEvent) error {
	entityID := e.EntityID
	createdAt := e.Timestamp
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	return w.client.EnqueueAuditLogWrite(asynqinfra.AuditLogWritePayload{
		UserID:     e.ActorUserID,
		Action:     string(e.Action),
		EntityType: string(e.EntityType),
		EntityID:   &entityID,
		Message:    e.AuditMessage,
		IPAddress:  e.IPAddress,
		UserAgent:  e.UserAgent,
		CreatedAt:  createdAt,
	})
}
