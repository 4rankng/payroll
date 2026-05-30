package infrastructure

import (
	"context"
	"time"
)

// CachePort defines the interface for cache operations
type CachePort interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	InvalidatePattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) (bool, error)
}
