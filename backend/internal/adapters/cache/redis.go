package cache

import (
	"context"
	"time"

	"api-server/internal/app/services/infrastructure"
	infrastructureports "api-server/internal/domain/ports/infrastructure"
)

// RedisAdapter implements infrastructureports.CachePort using existing CacheService
type RedisAdapter struct {
	cacheService *infrastructure.CacheService
}

// NewRedisAdapter creates a new Redis cache adapter
func NewRedisAdapter(cacheService *infrastructure.CacheService) infrastructureports.CachePort {
	return &RedisAdapter{
		cacheService: cacheService,
	}
}

// Get retrieves a value from cache
func (a *RedisAdapter) Get(ctx context.Context, key string, dest interface{}) error {
	return a.cacheService.Get(ctx, key, dest)
}

// Set stores a value in cache with TTL
func (a *RedisAdapter) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return a.cacheService.Set(ctx, key, value, ttl)
}

// Delete removes a key from cache
func (a *RedisAdapter) Delete(ctx context.Context, key string) error {
	return a.cacheService.Delete(ctx, key)
}

// InvalidatePattern removes all keys matching a pattern
func (a *RedisAdapter) InvalidatePattern(ctx context.Context, pattern string) error {
	return a.cacheService.DeletePattern(ctx, pattern)
}

// Exists checks if a key exists in cache
func (a *RedisAdapter) Exists(ctx context.Context, key string) (bool, error) {
	return a.cacheService.Exists(ctx, key)
}
