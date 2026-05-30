package infrastructure

import (
	"api-server/internal/pkg/clock"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"api-server/internal/infra/observability"
	"api-server/internal/infra/persistence"
)

// CacheService provides caching functionality using Redis
type CacheService struct {
	redis *persistence.RedisClient
}

// NewCacheService creates a new cache service instance
func NewCacheService(redis *persistence.RedisClient) *CacheService {
	return &CacheService{
		redis: redis,
	}
}

// extractCacheType extracts the cache type from a cache key (e.g., "dashboard:summary" -> "dashboard")
func (c *CacheService) extractCacheType(key string) string {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

// Get retrieves a value from cache and unmarshals it into the destination
func (c *CacheService) Get(ctx context.Context, key string, dest interface{}) error {
	start := clock.Now()
	cacheType := c.extractCacheType(key)

	val, err := c.redis.Get(ctx, key).Result()
	duration := time.Since(start).Seconds()
	observability.ObserveCacheOperation("get", cacheType, duration)

	if err != nil {
		// Cache miss
		observability.IncrementCacheMiss(cacheType)
		return err
	}

	// Cache hit
	observability.IncrementCacheHit(cacheType)
	return json.Unmarshal([]byte(val), dest)
}

// Set stores a value in cache with the specified TTL
func (c *CacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	start := clock.Now()
	cacheType := c.extractCacheType(key)

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	err = c.redis.Set(ctx, key, data, ttl).Err()
	duration := time.Since(start).Seconds()
	observability.ObserveCacheOperation("set", cacheType, duration)

	return err
}

// Delete removes a key from cache
func (c *CacheService) Delete(ctx context.Context, key string) error {
	start := clock.Now()
	cacheType := c.extractCacheType(key)

	err := c.redis.Del(ctx, key).Err()
	duration := time.Since(start).Seconds()
	observability.ObserveCacheOperation("delete", cacheType, duration)

	return err
}

// DeletePattern removes all keys matching a pattern using SCAN to avoid blocking Redis
func (c *CacheService) DeletePattern(ctx context.Context, pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := c.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if err := c.redis.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// Exists checks if a key exists in cache
func (c *CacheService) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.redis.Exists(ctx, key).Result()
	return count > 0, err
}

// FlushAll removes all keys from the current database
func (c *CacheService) FlushAll(ctx context.Context) error {
	return c.redis.FlushDB(ctx).Err()
}

// GenerateBankCacheKey generates a cache key for bank-related data
func (c *CacheService) GenerateBankCacheKey(operation string, params ...string) string {
	key := fmt.Sprintf("banks:%s", operation)
	for _, param := range params {
		key = fmt.Sprintf("%s:%s", key, param)
	}
	return key
}

// GenerateSettingsCacheKey generates a cache key for settings-related data
func (c *CacheService) GenerateSettingsCacheKey(operation string, params ...string) string {
	key := fmt.Sprintf("settings:%s", operation)
	for _, param := range params {
		key = fmt.Sprintf("%s:%s", key, param)
	}
	return key
}

// GenerateTransactionCacheKey generates a cache key for transaction-related data
func (c *CacheService) GenerateTransactionCacheKey(operation string, params ...string) string {
	key := fmt.Sprintf("transactions:%s", operation)
	for _, param := range params {
		key = fmt.Sprintf("%s:%s", key, param)
	}
	return key
}

// GenerateTimesheetCacheKey generates a cache key for timesheet-related data
func (c *CacheService) GenerateTimesheetCacheKey(operation string, params ...string) string {
	key := fmt.Sprintf("timesheets:%s", operation)
	for _, param := range params {
		key = fmt.Sprintf("%s:%s", key, param)
	}
	return key
}

// GenerateDashboardCacheKey generates a cache key for dashboard-related data
func (c *CacheService) GenerateDashboardCacheKey(operation string, params ...string) string {
	key := fmt.Sprintf("dashboard:%s", operation)
	for _, param := range params {
		key = fmt.Sprintf("%s:%s", key, param)
	}
	return key
}

// GetRedisStats retrieves Redis statistics for monitoring
func (c *CacheService) GetRedisStats(ctx context.Context) (map[string]string, error) {
	info, err := c.redis.Info(ctx, "stats", "memory").Result()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]string)
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.Contains(line, ":") && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				stats[parts[0]] = parts[1]
			}
		}
	}

	return stats, nil
}

// Bank Cache TTL constants (bank data rarely changes)
const (
	BankListCacheTTL   = 1 * time.Hour // Bank list rarely changes
	BankSearchCacheTTL = 1 * time.Hour // Search results can be cached shorter
	BankDetailCacheTTL = 1 * time.Hour // Individual bank details
)
