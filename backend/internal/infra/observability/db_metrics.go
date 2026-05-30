package observability

import (
	"context"
	"sync/atomic"

	"gorm.io/gorm"
)

// DBRequestMetrics tracks database activity for a single HTTP request.
type DBRequestMetrics struct {
	StatementCount int64
}

type dbMetricsCtxKey struct{}

// WithDBMetrics attaches a DBRequestMetrics container to the context so that
// GORM callbacks can increment per-request statement counts.
func WithDBMetrics(ctx context.Context) (context.Context, *DBRequestMetrics) {
	metrics := &DBRequestMetrics{}
	return context.WithValue(ctx, dbMetricsCtxKey{}, metrics), metrics
}

// GetDBMetrics returns the DBRequestMetrics associated with the context, if any.
func GetDBMetrics(ctx context.Context) *DBRequestMetrics {
	if ctx == nil {
		return nil
	}

	if v := ctx.Value(dbMetricsCtxKey{}); v != nil {
		if metrics, ok := v.(*DBRequestMetrics); ok {
			return metrics
		}
	}

	return nil
}

// RegisterDBMetricsCallbacks installs GORM callbacks that increment the
// per-request statement counter for all SQL operations.
func RegisterDBMetricsCallbacks(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	callback := func(tx *gorm.DB) {
		if tx == nil || tx.Statement == nil {
			return
		}

		ctx := tx.Statement.Context
		if ctx == nil {
			return
		}

		if metrics := GetDBMetrics(ctx); metrics != nil {
			atomic.AddInt64(&metrics.StatementCount, 1)
		}
	}

	if err := db.Callback().Query().After("gorm:query").Register("observability:db_metrics_query", callback); err != nil {
		return err
	}
	if err := db.Callback().Create().After("gorm:create").Register("observability:db_metrics_create", callback); err != nil {
		return err
	}
	if err := db.Callback().Update().After("gorm:update").Register("observability:db_metrics_update", callback); err != nil {
		return err
	}
	if err := db.Callback().Delete().After("gorm:delete").Register("observability:db_metrics_delete", callback); err != nil {
		return err
	}
	if err := db.Callback().Raw().After("gorm:raw").Register("observability:db_metrics_raw", callback); err != nil {
		return err
	}

	return nil
}
