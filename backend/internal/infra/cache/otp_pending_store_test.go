package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestOTPPendingStore(t *testing.T, ttl time.Duration) (*OTPPendingStore, *redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})
	return NewOTPPendingStore(client, ttl), client, mr
}

func TestOTPPendingStore_CreateSessionIfAbsentSequentialDuplicate(t *testing.T) {
	store, _, _ := newTestOTPPendingStore(t, time.Minute)
	ctx := context.Background()

	firstID, created, err := store.CreateSessionIfAbsent(ctx, 42, []byte("first"), "127.0.0.1", "first-agent")
	if err != nil {
		t.Fatalf("create first session: %v", err)
	}
	if !created {
		t.Fatal("first session was not marked created")
	}

	secondID, created, err := store.CreateSessionIfAbsent(ctx, 42, []byte("second"), "127.0.0.2", "second-agent")
	if err != nil {
		t.Fatalf("create duplicate session: %v", err)
	}
	if created {
		t.Fatal("duplicate session was marked created")
	}
	if secondID != firstID {
		t.Fatalf("duplicate session id = %q, want %q", secondID, firstID)
	}

	session, err := store.GetSession(ctx, firstID)
	if err != nil {
		t.Fatalf("get original session: %v", err)
	}
	if string(session.CodeHash) != "first" {
		t.Fatalf("stored code hash = %q, want first caller's payload", session.CodeHash)
	}
}

func TestOTPPendingStore_CreateSessionIfAbsentConcurrentDuplicate(t *testing.T) {
	store, _, _ := newTestOTPPendingStore(t, time.Minute)
	ctx := context.Background()

	const callers = 24
	type result struct {
		id      string
		created bool
		err     error
	}
	results := make(chan result, callers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			id, created, err := store.CreateSessionIfAbsent(ctx, 77, []byte("hash"), "127.0.0.1", "agent")
			results <- result{id: id, created: created, err: err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	createdCount := 0
	var sessionID string
	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent create: %v", result.err)
		}
		if sessionID == "" {
			sessionID = result.id
		} else if result.id != sessionID {
			t.Fatalf("session id = %q, want all callers to receive %q", result.id, sessionID)
		}
		if result.created {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Fatalf("created callers = %d, want 1", createdCount)
	}
}

func TestOTPPendingStore_CreateSessionIfAbsentRepairsStaleUserIndex(t *testing.T) {
	store, client, _ := newTestOTPPendingStore(t, time.Minute)
	ctx := context.Background()
	const userID = uint(91)
	if err := client.Set(ctx, userSessionKey(userID), "expired-primary", time.Minute).Err(); err != nil {
		t.Fatalf("seed stale user index: %v", err)
	}

	sessionID, created, err := store.CreateSessionIfAbsent(ctx, userID, []byte("hash"), "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if !created {
		t.Fatal("replacement session was not marked created")
	}
	indexedID, err := client.Get(ctx, userSessionKey(userID)).Result()
	if err != nil {
		t.Fatalf("read repaired user index: %v", err)
	}
	if indexedID != sessionID {
		t.Fatalf("repaired user index = %q, want %q", indexedID, sessionID)
	}
	if _, err := store.GetSession(ctx, sessionID); err != nil {
		t.Fatalf("get replacement session: %v", err)
	}
}

func TestOTPPendingStore_GetSessionRejectsSupersededSession(t *testing.T) {
	store, client, _ := newTestOTPPendingStore(t, time.Minute)
	ctx := context.Background()
	const userID = uint(15)

	sessionID, created, err := store.CreateSessionIfAbsent(ctx, userID, []byte("hash"), "127.0.0.1", "agent")
	if err != nil || !created {
		t.Fatalf("create session: created=%v err=%v", created, err)
	}
	if err := client.Set(ctx, userSessionKey(userID), "newer-session", time.Minute).Err(); err != nil {
		t.Fatalf("supersede user index: %v", err)
	}

	if _, err := store.GetSession(ctx, sessionID); !errors.Is(err, ErrOTPSessionNotFound) {
		t.Fatalf("get superseded session: got %v, want ErrOTPSessionNotFound", err)
	}
}

func TestOTPPendingStore_SaveSessionPreservesAndAlignsTTL(t *testing.T) {
	const sessionTTL = 5 * time.Minute
	store, client, mr := newTestOTPPendingStore(t, sessionTTL)
	ctx := context.Background()
	const userID = uint(23)

	sessionID, created, err := store.CreateSessionIfAbsent(ctx, userID, []byte("hash"), "127.0.0.1", "agent")
	if err != nil || !created {
		t.Fatalf("create session: created=%v err=%v", created, err)
	}
	mr.FastForward(2 * time.Minute)
	if err := client.Expire(ctx, userSessionKey(userID), 30*time.Second).Err(); err != nil {
		t.Fatalf("shorten user index ttl: %v", err)
	}

	session, err := store.GetSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	session.Attempts++
	if err := store.SaveSession(ctx, sessionID, session); err != nil {
		t.Fatalf("save session: %v", err)
	}

	primaryTTL, err := client.TTL(ctx, sessionKey(sessionID)).Result()
	if err != nil {
		t.Fatalf("read primary ttl: %v", err)
	}
	indexTTL, err := client.TTL(ctx, userSessionKey(userID)).Result()
	if err != nil {
		t.Fatalf("read user index ttl: %v", err)
	}
	if primaryTTL != 3*time.Minute {
		t.Fatalf("primary ttl = %s, want remaining 3m", primaryTTL)
	}
	if indexTTL != primaryTTL {
		t.Fatalf("user index ttl = %s, want primary ttl %s", indexTTL, primaryTTL)
	}
}
