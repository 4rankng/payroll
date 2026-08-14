package zalo

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisRefreshCoordinator_WaitsThenAcquires(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	coordinator := NewRedisRefreshCoordinator(client)

	releaseFirst, err := coordinator.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	acquiredSecond := make(chan func(context.Context) error, 1)
	acquireErr := make(chan error, 1)
	go func() {
		release, err := coordinator.Acquire(context.Background())
		if err != nil {
			acquireErr <- err
			return
		}
		acquiredSecond <- release
	}()

	select {
	case <-acquiredSecond:
		t.Fatal("second caller acquired while first lease was active")
	case err := <-acquireErr:
		t.Fatalf("second acquire failed: %v", err)
	case <-time.After(75 * time.Millisecond):
	}
	if err := releaseFirst(context.Background()); err != nil {
		t.Fatalf("release first: %v", err)
	}

	select {
	case releaseSecond := <-acquiredSecond:
		if err := releaseSecond(context.Background()); err != nil {
			t.Fatalf("release second: %v", err)
		}
	case err := <-acquireErr:
		t.Fatalf("second acquire failed: %v", err)
	case <-time.After(time.Second):
		t.Fatal("second caller did not acquire after release")
	}
}

func TestRedisRefreshCoordinator_OldOwnerCannotDeleteNewLease(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	coordinator := NewRedisRefreshCoordinator(client)

	release, err := coordinator.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if err := client.Set(context.Background(), refreshLockKey, "new-owner", refreshLockTTL).Err(); err != nil {
		t.Fatalf("replace lease owner: %v", err)
	}
	if err := release(context.Background()); err != nil {
		t.Fatalf("release old owner: %v", err)
	}
	if got, err := mr.Get(refreshLockKey); err != nil || got != "new-owner" {
		t.Fatalf("lease value = %q, want new-owner", got)
	}
}

func TestRedisRefreshCoordinator_RenewsOwnedLease(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	coordinator := NewRedisRefreshCoordinator(client)
	coordinator.ttl = 90 * time.Millisecond
	coordinator.retry = 5 * time.Millisecond

	release, err := coordinator.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	mr.FastForward(60 * time.Millisecond)
	time.Sleep(45 * time.Millisecond) // allow at least one ttl/3 heartbeat
	mr.FastForward(60 * time.Millisecond)
	if !mr.Exists(refreshLockKey) {
		t.Fatal("owned lease expired instead of being renewed")
	}
	if err := release(context.Background()); err != nil {
		t.Fatalf("release: %v", err)
	}
}
