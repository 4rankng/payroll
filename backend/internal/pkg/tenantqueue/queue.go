package tenantqueue

import (
	"context"
	"sync"

	"api-server/internal/infra/observability"
)

// Task represents a job to be executed with context
type Task func(ctx context.Context)

// TenantQueue holds a worker pool for a tenant
type TenantQueue struct {
	tasks chan Task
	wg    sync.WaitGroup
}

// NewTenantQueue creates a new queue with n workers
func NewTenantQueue(workers int) *TenantQueue {
	q := &TenantQueue{tasks: make(chan Task, workers)}
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go func() {
			defer q.wg.Done()
			for t := range q.tasks {
				// Run task with background context; caller can include timeouts inside task
				t(context.Background())
			}
		}()
	}
	return q
}

// Enqueue adds a task to the queue (non-blocking if buffer available)
// tenant is the tenant identifier used to label metrics
func (q *TenantQueue) Enqueue(t Task, tenant string) bool {
	select {
	case q.tasks <- t:
		return true
	default:
		// record a queue drop metric for this tenant
		observability.IncrementTenantQueueDrop(tenant)
		return false
	}
}

// Shutdown waits for workers to finish after closing the queue
func (q *TenantQueue) Shutdown() {
	close(q.tasks)
	q.wg.Wait()
}
