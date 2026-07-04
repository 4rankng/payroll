package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newTestNonceStore spins up an in-process miniredis so the test is hermetic
// (no live Redis dependency).
func newTestNonceStore(t *testing.T) (*NonceStore, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cleanup := func() {
		_ = client.Close()
		mr.Close()
	}
	return NewNonceStore(client, time.Minute), cleanup
}

// TestNonceStore_ConsumeOnce verifies the first consumer wins and a second
// presentation of the same nonce is rejected (the replay defense).
func TestNonceStore_ConsumeOnce(t *testing.T) {
	store, cleanup := newTestNonceStore(t)
	defer cleanup()
	ctx := context.Background()

	if err := store.Consume(ctx, "nonce-abc"); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if err := store.Consume(ctx, "nonce-abc"); !errors.Is(err, ErrNonceAlreadyUsed) {
		t.Fatalf("second consume: got %v, want ErrNonceAlreadyUsed", err)
	}
}

// TestNonceStore_EmptyNonceRejected verifies a missing nonce claim is treated
// as a replay (the OIDC flow always includes one, so absence is suspicious).
func TestNonceStore_EmptyNonceRejected(t *testing.T) {
	store, cleanup := newTestNonceStore(t)
	defer cleanup()

	if err := store.Consume(context.Background(), ""); !errors.Is(err, ErrNonceAlreadyUsed) {
		t.Fatalf("empty nonce: got %v, want ErrNonceAlreadyUsed", err)
	}
}

// TestNonceStore_DistinctNonces verifies different nonces don't collide.
func TestNonceStore_DistinctNonces(t *testing.T) {
	store, cleanup := newTestNonceStore(t)
	defer cleanup()
	ctx := context.Background()

	if err := store.Consume(ctx, "nonce-1"); err != nil {
		t.Fatalf("consume nonce-1: %v", err)
	}
	if err := store.Consume(ctx, "nonce-2"); err != nil {
		t.Fatalf("consume nonce-2: %v", err)
	}
}
