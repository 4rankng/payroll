package llm

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestSemaphoreBoundsConcurrency(t *testing.T) {
	sem := NewSemaphore(2)
	ctx := context.Background()

	release1, err := sem.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire 1: %v", err)
	}
	release2, err := sem.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire 2: %v", err)
	}

	// The third acquire must block until a slot frees — with a bounded wait it
	// reports the caller context as done.
	waitCtx, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancel()
	if _, err := sem.Acquire(waitCtx); err == nil {
		t.Fatal("Acquire 3 = nil error on a full semaphore with a bounded wait")
	}

	release1()
	release1() // idempotent: double release must not free an extra slot

	release3, err := sem.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire after release: %v", err)
	}
	release2()
	release3()
}

func TestMemoryLimiterFixedWindow(t *testing.T) {
	limiter := NewMemoryLimiter(50*time.Millisecond, 2)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if allowed, err := limiter.Allow(ctx, "7"); err != nil || !allowed {
			t.Fatalf("Allow %d = %v, %v; want allowed", i+1, allowed, err)
		}
	}
	if allowed, err := limiter.Allow(ctx, "7"); err != nil || allowed {
		t.Fatalf("Allow 3 = %v, %v; want denied", allowed, err)
	}
	// Budgets are per user.
	if allowed, err := limiter.Allow(ctx, "8"); err != nil || !allowed {
		t.Fatalf("other user Allow = %v, %v; want allowed", allowed, err)
	}

	time.Sleep(60 * time.Millisecond)
	if allowed, err := limiter.Allow(ctx, "7"); err != nil || !allowed {
		t.Fatalf("Allow after window = %v, %v; want allowed", allowed, err)
	}
}

func TestFixedWindowLimiterRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	limiter := NewFixedWindowLimiter(client, time.Minute, 2, "llm:rate:")
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if allowed, err := limiter.Allow(ctx, "42"); err != nil || !allowed {
			t.Fatalf("Allow %d = %v, %v; want allowed", i+1, allowed, err)
		}
	}
	if allowed, err := limiter.Allow(ctx, "42"); err != nil || allowed {
		t.Fatalf("Allow 3 = %v, %v; want denied", allowed, err)
	}
	// Keys are namespaced per user under the configured prefix.
	if got := mr.Exists("llm:rate:42"); !got {
		t.Fatal("expected the namespaced counter key to exist in Redis")
	}
}

func TestGuardAllowUser(t *testing.T) {
	ctx := context.Background()

	withoutLimiter := NewGuard(NewSemaphore(1), nil)
	if allowed, err := withoutLimiter.AllowUser(ctx, 1); err != nil || !allowed {
		t.Fatalf("nil limiter AllowUser = %v, %v; want allowed", allowed, err)
	}

	guard := NewGuard(NewSemaphore(1), NewMemoryLimiter(time.Minute, 1))
	if allowed, err := guard.AllowUser(ctx, 9); err != nil || !allowed {
		t.Fatalf("AllowUser = %v, %v; want allowed", allowed, err)
	}
	if allowed, err := guard.AllowUser(ctx, 9); err != nil || allowed {
		t.Fatalf("AllowUser over budget = %v, %v; want denied", allowed, err)
	}

	// Acquire is the concurrency seam: release frees the slot.
	release, err := guard.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		release2, acquireErr := guard.Acquire(ctx)
		if acquireErr != nil {
			t.Errorf("Acquire after release: %v", acquireErr)
			return
		}
		release2()
	}()
	time.Sleep(10 * time.Millisecond)
	release()
	wg.Wait()
}
