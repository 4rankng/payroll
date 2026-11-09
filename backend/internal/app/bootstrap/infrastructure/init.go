package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	appConfig "api-server/internal/config"
	"api-server/internal/infra/observability"
	"api-server/internal/infra/persistence"
	"api-server/internal/pkg/retry"
	"api-server/migrations"
)

// Components holds all infrastructure components
type Components struct {
	Logger *slog.Logger
	DB     *persistence.Database
	Redis  *persistence.RedisClient
}

// Initialize initializes all infrastructure components
func Initialize(cfg *appConfig.Config) (*Components, error) {
	logger, err := initLogger(cfg)
	if err != nil {
		return nil, err
	}

	db, err := initDatabase(cfg, logger)
	if err != nil {
		return nil, err
	}

	redis, err := initRedis(cfg, logger)
	if err != nil {
		return nil, err
	}

	return &Components{
		Logger: logger,
		DB:     db,
		Redis:  redis,
	}, nil
}

func initLogger(cfg *appConfig.Config) (*slog.Logger, error) {
	return observability.NewLogger(observability.LogConfig{
		Level: cfg.Log.Level,
		Dir:   cfg.Log.Dir,
	})
}

func initDatabase(cfg *appConfig.Config, logger *slog.Logger) (*persistence.Database, error) {
	retryConfig := retry.Config{
		MaxRetries:    3,
		InitialDelay:  200 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
	}

	db, err := persistence.NewDatabase(persistence.DatabaseConfig{
		Driver:          cfg.DB.Driver,
		DSN:             cfg.DB.DSN,
		RetryConfig:     &retryConfig,
		MaxIdleConns:    cfg.DB.MaxIdleConns,
		MaxOpenConns:    cfg.DB.MaxOpenConns,
		ConnMaxLifetime: cfg.DB.ConnMaxLifetime,
	})
	if err != nil {
		return nil, err
	}

	// Apply pending SQL migrations at boot (DB_AUTO_MIGRATE=false pins the
	// schema during an incident). This is the safety net for the manual
	// flows: deploys only rebuild containers, so a new migration otherwise
	// reaches prod only if someone remembers to run it by hand.
	if cfg.DB.AutoMigrate {
		if err := persistence.RunPendingSQLMigrations(
			context.Background(),
			db.DB,
			migrations.UpSQLFiles,
			logger,
		); err != nil {
			return nil, fmt.Errorf("sql migrations: %w", err)
		}
	}

	logger.Info("Successfully connected to database")
	return db, nil
}

func initRedis(cfg *appConfig.Config, logger *slog.Logger) (*persistence.RedisClient, error) {
	redis, err := persistence.NewRedisClient(persistence.RedisConfig{
		Addr: cfg.Redis.Addr,
		DB:   cfg.Redis.DB,
	})
	if err != nil {
		return nil, err
	}

	logger.Info("Successfully connected to Redis")
	return redis, nil
}
