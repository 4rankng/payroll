package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newTestZaloStore spins up an in-process miniredis so the store tests are
// hermetic (no external Redis dependency). Mirrors nonce_store_test.go.
func newTestZaloStore(t *testing.T) (*ZaloResetStore, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewZaloResetStore(client, 10*time.Minute), mr
}

func hashHexOf(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestZaloResetStore_CreateConsumeHappy(t *testing.T) {
	store, _ := newTestZaloStore(t)
	ctx := context.Background()

	sid, err := store.Create(ctx, 42, "codehashA")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sid == "" {
		t.Fatal("empty session id")
	}

	uid, err := store.Consume(ctx, sid, "codehashA")
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if uid != 42 {
		t.Errorf("uid = %d, want 42", uid)
	}
}

func TestZaloResetStore_WrongCodeDoesNotConsume(t *testing.T) {
	store, _ := newTestZaloStore(t)
	ctx := context.Background()

	sid, _ := store.Create(ctx, 42, "codehashA")

	// Wrong code: session survives.
	_, err := store.Consume(ctx, sid, "wronghash")
	if err != ErrZaloResetInvalidCode {
		t.Fatalf("wrong code: err = %v, want ErrZaloResetInvalidCode", err)
	}

	// Right code still works after a wrong attempt.
	uid, err := store.Consume(ctx, sid, "codehashA")
	if err != nil || uid != 42 {
		t.Errorf("right code after wrong: uid=%d err=%v", uid, err)
	}
}

func TestZaloResetStore_ConsumeUnknownSession(t *testing.T) {
	store, _ := newTestZaloStore(t)
	_, err := store.Consume(context.Background(), "nonexistent-sid", "anyhash")
	if err != ErrZaloResetSessionNotFound {
		t.Errorf("unknown session: err = %v, want ErrZaloResetSessionNotFound", err)
	}
}

func TestZaloResetStore_ConsumeAfterConsume(t *testing.T) {
	store, _ := newTestZaloStore(t)
	ctx := context.Background()
	sid, _ := store.Create(ctx, 42, "codehashA")

	if _, err := store.Consume(ctx, sid, "codehashA"); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	// Second consume → session already deleted → not found.
	_, err := store.Consume(ctx, sid, "codehashA")
	if err != ErrZaloResetSessionNotFound {
		t.Errorf("second consume: err = %v, want ErrZaloResetSessionNotFound", err)
	}
}

func TestZaloResetStore_DummySessionNeverConsumes(t *testing.T) {
	store, _ := newTestZaloStore(t)
	ctx := context.Background()
	sid, _ := store.CreateDummy(ctx)
	if sid == "" {
		t.Fatal("empty dummy session id")
	}
	// The dummy's code hash is cryptographically random and never disclosed, so
	// any submitted code will mismatch → ErrZaloResetInvalidCode (session
	// survives). A subsequent consume with the dummy hash would return uid=0
	// which the store maps to ErrZaloResetSessionNotFound. Either way: never
	// a successful uid return.
	_, err := store.Consume(ctx, sid, "some-guess")
	if err != ErrZaloResetInvalidCode && err != ErrZaloResetSessionNotFound {
		t.Errorf("dummy consume: err = %v, want invalid-code or not-found", err)
	}
}

func TestZaloResetStore_ConcurrentConsumeOnlyOneWins(t *testing.T) {
	store, _ := newTestZaloStore(t)
	ctx := context.Background()
	sid, _ := store.Create(ctx, 42, "codehashA")

	var wg sync.WaitGroup
	var successes, failures int
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			uid, err := store.Consume(ctx, sid, "codehashA")
			mu.Lock()
			defer mu.Unlock()
			if err == nil && uid == 42 {
				successes++
			} else {
				failures++
			}
		}()
	}
	wg.Wait()

	if successes != 1 {
		t.Errorf("concurrent consume: successes = %d, want exactly 1 (atomic GETDEL-via-Lua)", successes)
	}
	if failures != 9 {
		t.Errorf("concurrent consume: failures = %d, want 9", failures)
	}
}

func TestZaloResetStore_TTLExpiry(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	store := NewZaloResetStore(client, 1*time.Second)
	ctx := context.Background()
	sid, _ := store.Create(ctx, 42, "codehashA")

	// Fast-forward miniredis past the TTL.
	mr.FastForward(2 * time.Second)

	_, err = store.Consume(ctx, sid, "codehashA")
	if err != ErrZaloResetSessionNotFound {
		t.Errorf("after TTL expiry: err = %v, want ErrZaloResetSessionNotFound", err)
	}
}

func TestZaloResetStore_DummyAndRealProduceSameShapeID(t *testing.T) {
	// Anti-enumeration: a dummy session id must be structurally
	// indistinguishable from a real one (same length, same charset).
	store, _ := newTestZaloStore(t)
	ctx := context.Background()

	realSID, _ := store.Create(ctx, 1, "h")
	dummySID, _ := store.CreateDummy(ctx)

	if len(realSID) != len(dummySID) {
		t.Errorf("id length mismatch: real=%d dummy=%d (anti-enumeration leak)", len(realSID), len(dummySID))
	}
	// Both are base64url (~43 chars from 32 bytes).
	if len(realSID) < 40 {
		t.Errorf("real id too short: %q", realSID)
	}
	fmt.Sprintf("real=%q dummy=%q", realSID, dummySID) // keep fmt import used
}
