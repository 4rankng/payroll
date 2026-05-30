package ports

import (
	"context"
	"time"
)

// CachePort defines the interface for caching operations
// This allows domains to use caching without coupling to infrastructure
type CachePort interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	InvalidatePattern(ctx context.Context, pattern string) error
}
