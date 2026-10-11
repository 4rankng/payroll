package approval

import (
	"context"
	"time"

	asynqinfra "api-server/internal/infra/asynq"
)

// AsynqRetryScheduler adapts the engine's retry port onto the asynq client.
type AsynqRetryScheduler struct {
	client *asynqinfra.Client
}

// NewAsynqRetryScheduler constructs the scheduler adapter.
func NewAsynqRetryScheduler(client *asynqinfra.Client) *AsynqRetryScheduler {
	return &AsynqRetryScheduler{client: client}
}

// ScheduleRetry enqueues the attempt-scoped approval:retry task.
func (s *AsynqRetryScheduler) ScheduleRetry(_ context.Context, requestID uint, attempt int, at time.Time) error {
	return s.client.EnqueueApprovalRetry(requestID, attempt, at)
}
