package zalo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	refreshLockKey       = "zalo:oa:token:refresh"
	refreshLockTTL       = 30 * time.Second
	refreshLockRetryWait = 50 * time.Millisecond
)

// RedisRefreshCoordinator provides one refresh-token authority across API
// instances. The lease TTL prevents a crashed owner from blocking renewal
// forever; owner-checked release prevents an expired owner deleting a newer
// process's lease.
type RedisRefreshCoordinator struct {
	client redis.Cmdable
	ttl    time.Duration
	retry  time.Duration
}

func NewRedisRefreshCoordinator(client redis.Cmdable) *RedisRefreshCoordinator {
	return &RedisRefreshCoordinator{client: client, ttl: refreshLockTTL, retry: refreshLockRetryWait}
}

func (c *RedisRefreshCoordinator) Acquire(ctx context.Context) (func(context.Context) error, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("zalo: redis refresh coordinator is not configured")
	}

	owner, err := refreshLockOwner()
	if err != nil {
		return nil, err
	}

	ticker := time.NewTicker(c.retry)
	defer ticker.Stop()
	for {
		acquired, setErr := c.client.SetNX(ctx, refreshLockKey, owner, c.ttl).Result()
		if setErr != nil {
			return nil, fmt.Errorf("zalo: acquire refresh lease: %w", setErr)
		}
		if acquired {
			stopRenewal := make(chan struct{})
			renewalDone := make(chan struct{})
			go c.renewLease(owner, stopRenewal, renewalDone)
			var releaseOnce sync.Once
			return func(releaseCtx context.Context) error {
				releaseOnce.Do(func() { close(stopRenewal) })
				<-renewalDone
				const releaseScript = `
					if redis.call("get", KEYS[1]) == ARGV[1] then
						return redis.call("del", KEYS[1])
					end
					return 0
				`
				if err := c.client.Eval(releaseCtx, releaseScript, []string{refreshLockKey}, owner).Err(); err != nil {
					return fmt.Errorf("zalo: release refresh lease: %w", err)
				}
				return nil
			}, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("zalo: wait for refresh lease: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func (c *RedisRefreshCoordinator) renewLease(owner string, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	interval := c.ttl / 3
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	const renewScript = `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		end
		return 0
	`
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), interval)
			result, err := c.client.Eval(ctx, renewScript, []string{refreshLockKey}, owner, c.ttl.Milliseconds()).Int()
			cancel()
			if err != nil || result != 1 {
				return
			}
		}
	}
}

func refreshLockOwner() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("zalo: generate refresh lease owner: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}
