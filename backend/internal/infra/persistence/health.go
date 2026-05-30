package persistence

import (
	"context"
	"time"
)

// DatabaseHealthChecker implements health checking for database
func (d *Database) CheckHealth(ctx context.Context) error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}

	// Set a short timeout for health check
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return sqlDB.PingContext(ctx)
}

// RedisHealthChecker implements health checking for Redis
func (r *RedisClient) CheckHealth(ctx context.Context) error {
	return r.Ping(ctx).Err()
}
