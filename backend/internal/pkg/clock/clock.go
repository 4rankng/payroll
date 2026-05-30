package clock

import (
	"sync"
	"time"
)

// DefaultLocation is the application's canonical timezone.
// All Now() calls return times in this location.
var DefaultLocation = mustLoadLocation("Asia/Ho_Chi_Minh")

// ─── Clock interface ────────────────────────────────────────────────────────

// Clock is the interface every service should depend on for time.
type Clock interface {
	// Now returns the current time in the application timezone (Asia/Ho_Chi_Minh).
	Now() time.Time

	// NowUTC returns the current time in UTC.
	NowUTC() time.Time

	// TodayStart returns 00:00:00 of today in the application timezone.
	TodayStart() time.Time

	// TodayEnd returns 23:59:59.999999999 of today in the application timezone.
	TodayEnd() time.Time

	// UnixNow returns the current Unix timestamp in seconds.
	UnixNow() int64
}

// ─── Real clock (production) ────────────────────────────────────────────────

type realClock struct{}

// New returns the production Clock backed by the system clock.
func New() Clock {
	return &realClock{}
}

func (realClock) Now() time.Time {
	return time.Now().In(DefaultLocation)
}

func (realClock) NowUTC() time.Time {
	return time.Now().UTC()
}

func (realClock) TodayStart() time.Time {
	now := time.Now().In(DefaultLocation)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, DefaultLocation)
}

func (realClock) TodayEnd() time.Time {
	now := time.Now().In(DefaultLocation)
	return time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, DefaultLocation)
}

func (realClock) UnixNow() int64 {
	return time.Now().Unix()
}

// ─── Fake clock (tests) ─────────────────────────────────────────────────────

// FakeClock is a deterministic Clock for testing.
// By default it auto-advances from real time (useful for non-prod environments
// that need admin clock manipulation without freezing background workers).
// When Set() or Advance() is called, it freezes at the specified time.
// Reset() restores auto-advance behavior.
type FakeClock struct {
	mu     sync.Mutex
	now    time.Time // zero value means "auto-advance from real time"
	frozen bool
}

// NewFake creates a FakeClock frozen at the given time.
func NewFake(t time.Time) *FakeClock {
	return &FakeClock{now: t.In(DefaultLocation), frozen: true}
}

// NewAutoFake creates a FakeClock that auto-advances from real time.
// It behaves like a RealClock until Set() or Advance() is called.
func NewAutoFake() *FakeClock {
	return &FakeClock{frozen: false}
}

func (f *FakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.frozen {
		return f.now
	}
	return time.Now().In(DefaultLocation)
}

func (f *FakeClock) NowUTC() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.frozen {
		return f.now.UTC()
	}
	return time.Now().UTC()
}

func (f *FakeClock) UnixNow() int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.frozen {
		return f.now.Unix()
	}
	return time.Now().Unix()
}

func (f *FakeClock) TodayStart() time.Time {
	now := f.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, DefaultLocation)
}

func (f *FakeClock) TodayEnd() time.Time {
	now := f.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, DefaultLocation)
}

// Advance moves the fake clock forward by d and freezes it.
func (f *FakeClock) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.frozen {
		f.now = time.Now().In(DefaultLocation)
	}
	f.now = f.now.Add(d)
	f.frozen = true
}

// Set freezes the fake clock at the given time.
func (f *FakeClock) Set(t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = t.In(DefaultLocation)
	f.frozen = true
}

// Reset restores auto-advance behavior (unfreezes the clock).
func (f *FakeClock) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = time.Time{}
	f.frozen = false
}

// IsFrozen returns whether the clock is frozen (manually set).
func (f *FakeClock) IsFrozen() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.frozen
}

// ─── Global clock (for package-level functions) ─────────────────────────────

var (
	globalMu  sync.RWMutex
	globalClk Clock = &realClock{}
)

// SetGlobal replaces the global clock. Call this once at test setup.
// Production code should NOT call this — the default realClock is fine.
func SetGlobal(c Clock) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalClk = c
}

// Global returns the current global clock.
func Global() Clock {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalClk
}

// Now returns the global clock's current time.
// Use this when you can't inject Clock via DI.
func Now() time.Time { return Global().Now() }

// NowUTC returns the global clock's current time in UTC.
func NowUTC() time.Time { return Global().NowUTC() }

// ─── Helpers ────────────────────────────────────────────────────────────────

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic("clock: failed to load timezone " + name + ": " + err.Error())
	}
	return loc
}
