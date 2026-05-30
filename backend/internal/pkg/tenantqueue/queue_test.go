package tenantqueue

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTenantQueue(t *testing.T) {
	q := NewTenantQueue(2)
	assert.NotNil(t, q)
	assert.NotNil(t, q.tasks)

	// Cleanup
	q.Shutdown()
}

func TestTenantQueue_Enqueue_Success(t *testing.T) {
	q := NewTenantQueue(2)
	defer q.Shutdown()

	var executed atomic.Bool
	task := func(ctx context.Context) {
		executed.Store(true)
	}

	success := q.Enqueue(task, "test-tenant")
	assert.True(t, success)

	// Wait for task to execute
	time.Sleep(100 * time.Millisecond)
	assert.True(t, executed.Load())
}

func TestTenantQueue_Enqueue_MultipleTasks(t *testing.T) {
	q := NewTenantQueue(5)
	defer q.Shutdown()

	var counter atomic.Int32
	var wg sync.WaitGroup

	// Enqueue multiple tasks
	for i := 0; i < 3; i++ {
		wg.Add(1)
		task := func(ctx context.Context) {
			counter.Add(1)
			wg.Done()
		}
		success := q.Enqueue(task, "test-tenant")
		assert.True(t, success)
	}

	// Wait for all tasks to complete
	wg.Wait()
	assert.Equal(t, int32(3), counter.Load())
}

func TestTenantQueue_Enqueue_QueueFull(t *testing.T) {
	// Create a queue with 0 workers to prevent task consumption
	q := &TenantQueue{tasks: make(chan Task, 2)}

	// Fill the buffer completely
	for i := 0; i < 2; i++ {
		task := func(ctx context.Context) {}
		success := q.Enqueue(task, "test-tenant")
		assert.True(t, success)
	}

	// Try to enqueue when queue is full - should fail
	task := func(ctx context.Context) {}
	success := q.Enqueue(task, "test-tenant")
	assert.False(t, success, "Should return false when queue is full")
}

func TestTenantQueue_Shutdown(t *testing.T) {
	q := NewTenantQueue(5)

	var counter atomic.Int32
	var wg sync.WaitGroup

	// Enqueue some tasks
	for i := 0; i < 3; i++ {
		wg.Add(1)
		task := func(ctx context.Context) {
			counter.Add(1)
			wg.Done()
		}
		success := q.Enqueue(task, "test-tenant")
		if !success {
			wg.Done() // Avoid hanging if enqueue fails
		}
	}

	// Wait for tasks to complete
	wg.Wait()

	// Shutdown should wait for all tasks to complete
	q.Shutdown()

	assert.GreaterOrEqual(t, counter.Load(), int32(1))
}

func TestTenantQueue_TaskWithContext(t *testing.T) {
	q := NewTenantQueue(1)
	defer q.Shutdown()

	var receivedContext context.Context
	var wg sync.WaitGroup
	wg.Add(1)

	task := func(ctx context.Context) {
		receivedContext = ctx
		wg.Done()
	}

	success := q.Enqueue(task, "test-tenant")
	assert.True(t, success)

	wg.Wait()
	assert.NotNil(t, receivedContext)
}
