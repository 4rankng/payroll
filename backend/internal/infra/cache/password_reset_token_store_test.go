package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestTokenStore(t *testing.T, ttl time.Duration) (*PasswordResetTokenStore, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewPasswordResetTokenStore(client, ttl), mr
}

func TestPasswordResetTokenStore_Create_HighEntropy(t *testing.T) {
	store, _ := newTestTokenStore(t, 30*time.Minute)
	t1, err := store.Create(context.Background(), 42)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t2, err := store.Create(context.Background(), 42)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(t1) < 40 || len(t1) > 50 {
		t.Errorf("token length = %d, want ~43 (32 bytes base64url no padding)", len(t1))
	}
	if t1 == t2 {
		t.Errorf("two Create calls produced identical tokens — entropy failure")
	}
}

func TestPasswordResetTokenStore_Consume_HappyPath(t *testing.T) {
	store, _ := newTestTokenStore(t, 30*time.Minute)
	token, err := store.Create(context.Background(), 7)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	uid, err := store.Consume(context.Background(), token)
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if uid != 7 {
		t.Errorf("uid = %d, want 7", uid)
	}
}

func TestPasswordResetTokenStore_Consume_SingleUse(t *testing.T) {
	store, _ := newTestTokenStore(t, 30*time.Minute)
	token, _ := store.Create(context.Background(), 1)
	if _, err := store.Consume(context.Background(), token); err != nil {
		t.Fatalf("first Consume: %v", err)
	}
	_, err := store.Consume(context.Background(), token)
	if !errors.Is(err, ErrPasswordResetTokenNotFound) {
		t.Errorf("second Consume err = %v, want ErrPasswordResetTokenNotFound", err)
	}
}

func TestPasswordResetTokenStore_Consume_UnknownToken(t *testing.T) {
	store, _ := newTestTokenStore(t, 30*time.Minute)
	_, err := store.Consume(context.Background(), "garbage-not-a-real-token")
	if !errors.Is(err, ErrPasswordResetTokenNotFound) {
		t.Errorf("err = %v, want ErrPasswordResetTokenNotFound", err)
	}
}

// TestPasswordResetTokenStore_Consume_Concurrent verifies the atomic GETDEL:
// exactly one of N concurrent consumers wins, the rest get not-found.
func TestPasswordResetTokenStore_Consume_Concurrent(t *testing.T) {
	store, _ := newTestTokenStore(t, 30*time.Minute)
	token, _ := store.Create(context.Background(), 99)

	const n = 10
	var wg sync.WaitGroup
	var success, failNotFound, failOther atomic.Int64
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, err := store.Consume(context.Background(), token)
			switch {
			case err == nil:
				success.Add(1)
			case errors.Is(err, ErrPasswordResetTokenNotFound):
				failNotFound.Add(1)
			default:
				failOther.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := success.Load(); got != 1 {
		t.Errorf("successes = %d, want exactly 1", got)
	}
	if got := failNotFound.Load(); got != n-1 {
		t.Errorf("not-found failures = %d, want %d", got, n-1)
	}
	if got := failOther.Load(); got != 0 {
		t.Errorf("other failures = %d, want 0", got)
	}
}

func TestPasswordResetTokenStore_Consume_Expired(t *testing.T) {
	store, mr := newTestTokenStore(t, 1*time.Second)
	token, _ := store.Create(context.Background(), 5)
	mr.FastForward(2 * time.Second) // advance miniredis clock past TTL
	_, err := store.Consume(context.Background(), token)
	if !errors.Is(err, ErrPasswordResetTokenNotFound) {
		t.Errorf("err = %v, want ErrPasswordResetTokenNotFound (expired)", err)
	}
}

// TestPasswordResetTokenStore_Consume_StoreUnavailable verifies Red Team H5:
// a Redis outage surfaces as ErrPasswordResetStoreUnavailable, not the
// not-found sentinel — so the service can map it to 500, not 401.
func TestPasswordResetTokenStore_Consume_StoreUnavailable(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewPasswordResetTokenStore(client, 30*time.Minute)

	token, _ := store.Create(context.Background(), 3)
	mr.Close() // simulate Redis going down

	_, err = store.Consume(context.Background(), token)
	if !errors.Is(err, ErrPasswordResetStoreUnavailable) {
		t.Errorf("err = %v, want ErrPasswordResetStoreUnavailable", err)
	}
}

func TestPasswordResetTokenStore_Delete(t *testing.T) {
	store, _ := newTestTokenStore(t, 30*time.Minute)
	token, _ := store.Create(context.Background(), 8)
	if err := store.Delete(context.Background(), token); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := store.Consume(context.Background(), token)
	if !errors.Is(err, ErrPasswordResetTokenNotFound) {
		t.Errorf("after Delete, Consume err = %v, want ErrPasswordResetTokenNotFound", err)
	}
}
