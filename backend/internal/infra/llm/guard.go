package llm

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Shared-key guards. Every LLM consumer (drafting, chat, vision, digest) bills
// one paid API key, so two cheap protections bound the blast radius of a
// runaway loop: a concurrency semaphore for the key as a whole, and a
// fixed-window rate limit per user. Deliberately minimal — spend accounting is
// a later concern; these two stop starvation now.

// Semaphore bounds concurrent in-flight LLM calls across the process.
type Semaphore struct {
	slots chan struct{}
}

// NewSemaphore creates a semaphore admitting at most maxConcurrent calls at a
// time. Values < 1 are raised to 1.
func NewSemaphore(maxConcurrent int) *Semaphore {
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	return &Semaphore{slots: make(chan struct{}, maxConcurrent)}
}

// Acquire blocks until a slot is free or ctx is done. The returned release
// function is idempotent and must be called exactly once per successful
// acquire (defer it).
func (s *Semaphore) Acquire(ctx context.Context) (release func(), err error) {
	select {
	case s.slots <- struct{}{}:
		var once sync.Once
		return func() {
			once.Do(func() { <-s.slots })
		}, nil
	case <-ctx.Done():
		return nil, ContextError(ctx)
	}
}

// Limiter is the per-user rate limit hook. Implementations are keyed by the
// caller's user ID so one user's loop cannot exhaust another's budget.
//
// Contract: a non-nil error means the limiter itself malfunctioned (e.g. Redis
// down), NOT that the call is denied — callers log and fail open, because an
// unavailable guard must not take the whole LLM surface down with it.
type Limiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// FixedWindowLimiter is a Redis-backed fixed-window counter
// (INCR + PEXPIRE): one round trip on the hot path, no Lua.
type FixedWindowLimiter struct {
	client redis.UniversalClient
	prefix string
	window time.Duration
	limit  int64
}

// NewFixedWindowLimiter builds the Redis-backed limiter. keyPrefix namespaces
// the counter keys (e.g. "llm:rate:").
func NewFixedWindowLimiter(client redis.UniversalClient, window time.Duration, limit int64, keyPrefix string) *FixedWindowLimiter {
	return &FixedWindowLimiter{client: client, prefix: keyPrefix, window: window, limit: limit}
}

// Allow increments the caller's window counter and reports whether the call
// fits in the configured limit.
func (l *FixedWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	fullKey := l.prefix + key
	count, err := l.client.Incr(ctx, fullKey).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		if err := l.client.PExpire(ctx, fullKey, l.window).Err(); err != nil {
			return false, err
		}
	}
	return count <= l.limit, nil
}

// MemoryLimiter is the same fixed window without Redis — used in tests and in
// environments where no Redis client is wired. One process only.
type MemoryLimiter struct {
	mu     sync.Mutex
	limit  int64
	window time.Duration
	counts map[string]memoryWindow
}

type memoryWindow struct {
	start time.Time
	count int64
}

// NewMemoryLimiter builds the in-process fixed-window limiter.
func NewMemoryLimiter(window time.Duration, limit int64) *MemoryLimiter {
	return &MemoryLimiter{limit: limit, window: window, counts: make(map[string]memoryWindow)}
}

// Allow implements Limiter.
func (l *MemoryLimiter) Allow(_ context.Context, key string) (bool, error) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	win, ok := l.counts[key]
	if !ok || now.Sub(win.start) >= l.window {
		l.counts[key] = memoryWindow{start: now, count: 1}
		return true, nil
	}
	win.count++
	l.counts[key] = win
	return win.count <= l.limit, nil
}

// Guard bundles the two shared-key protections around LLM calls. A nil Limiter
// disables the per-user limit (the semaphore still applies).
type Guard struct {
	sem     *Semaphore
	limiter Limiter
}

// NewGuard bundles a concurrency semaphore with an optional per-user limiter.
func NewGuard(sem *Semaphore, limiter Limiter) *Guard {
	return &Guard{sem: sem, limiter: limiter}
}

// Acquire blocks for a concurrency slot. Defer the returned release.
func (g *Guard) Acquire(ctx context.Context) (func(), error) {
	return g.sem.Acquire(ctx)
}

// AllowUser applies the per-user fixed window. allowed=false means the caller
// is over budget; err != nil means the limiter malfunctioned (log and fail
// open).
func (g *Guard) AllowUser(ctx context.Context, userID uint) (bool, error) {
	if g.limiter == nil {
		return true, nil
	}
	return g.limiter.Allow(ctx, strconv.FormatUint(uint64(userID), 10))
}
