package domain

import (
	"context"
	"time"
)

type CacheServiceUseCase interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeletePattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) (bool, error)
	FlushAll(ctx context.Context) error
	GenerateBankCacheKey(operation string, params ...string) string
	GenerateSettingsCacheKey(operation string, params ...string) string
	GenerateTransactionCacheKey(operation string, params ...string) string
	GenerateTimesheetCacheKey(operation string, params ...string) string
	GenerateDashboardCacheKey(operation string, params ...string) string
	GetRedisStats(ctx context.Context) (map[string]string, error)
}
