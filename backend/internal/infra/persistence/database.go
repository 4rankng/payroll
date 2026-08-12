package persistence

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/retry"
	"api-server/internal/pkg/scopes"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseConfig struct {
	Driver      string
	DSN         string
	RetryConfig *retry.Config
	// Connection pool settings (optional)
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

type Database struct {
	*gorm.DB
}

func NewDatabase(cfg DatabaseConfig) (*Database, error) {
	// Use the centralized observability logger for all logging
	appLogger := observability.GetLogger()
	appLogger.Info("Initializing database connection", "driver", cfg.Driver)

	// Use default retry config if none provided
	retryConfig := cfg.RetryConfig
	if retryConfig == nil {
		defaultCfg := retry.DefaultConfig()
		retryConfig = &defaultCfg
	}

	var db *gorm.DB

	// Configure GORM to use our centralized logger for SQL logging
	gormLogger := logger.Default.LogMode(logger.Info)
	config := &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return clock.Now()
		},
		// Enable prepared statement caching for repeated queries
		PrepareStmt: true,
	}

	// Create database connection with retries and exponential backoff
	ctx := context.Background()
	err := retry.WithExponentialBackoff(ctx, *retryConfig, func() error {
		var connErr error

		// Connect based on driver
		switch cfg.Driver {
		case "mysql":
			db, connErr = gorm.Open(mysql.Open(cfg.DSN), config)
		case "postgres", "postgresql":
			db, connErr = gorm.Open(postgres.Open(cfg.DSN), config)
		default:
			return fmt.Errorf("unsupported database driver: %s", cfg.Driver)
		}

		if connErr != nil {
			appLogger.Warn("Database connection attempt failed", "error", connErr, "driver", cfg.Driver)
			return connErr
		}

		// Test connection by pinging
		sqlDB, sqlErr := db.DB()
		if sqlErr != nil {
			appLogger.Warn("Failed to get underlying sql.DB", "error", sqlErr)
			return sqlErr
		}

		if pingErr := sqlDB.Ping(); pingErr != nil {
			appLogger.Warn("Database ping failed", "error", pingErr)
			return pingErr
		}

		return nil
	})

	if err != nil {
		appLogger.Error("Failed to connect to database after retries", "error", err, "driver", cfg.Driver)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		appLogger.Error("Failed to get underlying sql.DB", "error", err)
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool - allow overrides from config
	// Increased defaults to handle concurrent requests better
	maxIdle := 25 // Increased from 10 to handle more concurrent requests
	if cfg.MaxIdleConns > 0 {
		maxIdle = cfg.MaxIdleConns
	}
	maxOpen := 150 // Increased from 100 for better concurrency
	if cfg.MaxOpenConns > 0 {
		maxOpen = cfg.MaxOpenConns
	}
	connMaxLifetime := 30 * time.Minute // Reduced from 1 hour for better connection recycling
	if cfg.ConnMaxLifetime > 0 {
		connMaxLifetime = cfg.ConnMaxLifetime
	}

	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // Close idle connections after 10 minutes
	appLogger.Info("Database connection pool configured",
		"max_idle_conns", maxIdle,
		"max_open_conns", maxOpen,
		"conn_max_lifetime", connMaxLifetime,
		"conn_max_idle_time", "10m")

	// Attach per-request DB metrics callbacks (statement counting)
	if err := observability.RegisterDBMetricsCallbacks(db); err != nil {
		appLogger.Warn("Failed to register DB metrics callbacks", "error", err)
	}

	// Register active records plugin
	activeRecordsPlugin := scopes.ActiveRecordsPlugin{}
	if err := db.Use(&activeRecordsPlugin); err != nil {
		appLogger.Error("Failed to register active records plugin", "error", err)
		return nil, fmt.Errorf("failed to register active records plugin: %w", err)
	}
	appLogger.Info("Active records plugin registered successfully")

	appLogger.Info("Database connection established successfully", "driver", cfg.Driver)

	database := &Database{DB: db}

	database.warmUpSchemas(db)

	go database.startHealthMonitoring()

	return database, nil
}

func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (d *Database) warmUpSchemas(db *gorm.DB) {
	models := []any{
		&domain.LedgerEntry{},
		&domain.Asset{},
		&domain.Settlement{},
		&domain.User{},
		&domain.Transaction{},
		&domain.Employee{},
		&domain.Bank{},
		&domain.Project{},
		&domain.ProjectEmployee{},
		&domain.ProjectUser{},
		&domain.Payrate{},
		&domain.Timesheet{},
		&domain.TimesheetEditRequest{},
		&domain.Notification{},
		&domain.BulkTransferFile{},
		&domain.AdvancePayment{},
		&domain.AdvancePaymentRequest{},
		&domain.Loan{},
		&domain.LoanRepaymentSchedule{},
		&domain.Lender{},
		&domain.EmployeeUser{},
		&domain.APIEndpoint{},
		&domain.APIMetric{},
		&domain.Account{},
		&domain.BlacklistedToken{},
		&domain.Settings{},
		&domain.PushSubscription{},
		&domain.CronJobStatus{},
		&domain.AssetImportMetadata{},
		&domain.FlexPaySalaryNotification{},
	}

	appLogger := observability.GetLogger()
	for _, m := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			appLogger.Warn("Schema warm-up: parse failed", "model", fmt.Sprintf("%T", m), "error", err)
		}
	}
	appLogger.Info("GORM schema cache warmed up", "models", len(models))
}

// Stats returns the underlying sql.DB statistics for monitoring
func (d *Database) Stats() map[string]any {
	statsMap := make(map[string]any)
	if sqlDB, err := d.DB.DB(); err == nil {
		s := sqlDB.Stats()
		statsMap["MaxOpenConnections"] = s.MaxOpenConnections
		statsMap["OpenConnections"] = s.OpenConnections
		statsMap["InUse"] = s.InUse
		statsMap["Idle"] = s.Idle
		statsMap["WaitCount"] = s.WaitCount
		statsMap["WaitDuration"] = s.WaitDuration.String()
		statsMap["MaxIdleClosed"] = s.MaxIdleClosed
		statsMap["MaxIdleTimeClosed"] = s.MaxIdleTimeClosed
		statsMap["MaxLifetimeClosed"] = s.MaxLifetimeClosed

		// Calculate connection pool health metrics
		if s.MaxOpenConnections > 0 {
			utilizationPct := float64(s.OpenConnections) / float64(s.MaxOpenConnections) * 100
			statsMap["UtilizationPct"] = utilizationPct

			// Warning thresholds
			statsMap["HighUtilization"] = utilizationPct > 80.0
			statsMap["CriticalUtilization"] = utilizationPct > 95.0
		}
	}
	return statsMap
}

// LogConnectionPoolStats logs current connection pool statistics
func (d *Database) LogConnectionPoolStats() {
	appLogger := observability.GetLogger()
	stats := d.Stats()

	// Convert stats to structured logging fields
	appLogger.Info("Database connection pool statistics",
		"max_open_connections", stats["MaxOpenConnections"],
		"open_connections", stats["OpenConnections"],
		"in_use", stats["InUse"],
		"idle", stats["Idle"],
		"wait_count", stats["WaitCount"],
		"wait_duration", stats["WaitDuration"],
		"utilization_pct", stats["UtilizationPct"],
		"high_utilization", stats["HighUtilization"],
		"critical_utilization", stats["CriticalUtilization"])

	// Log warnings for high utilization
	if highUtil, ok := stats["HighUtilization"].(bool); ok && highUtil {
		appLogger.Warn("Database connection pool utilization is high",
			"utilization_pct", stats["UtilizationPct"],
			"recommendation", "Consider increasing MaxOpenConns or optimizing query patterns")
	}

	if criticalUtil, ok := stats["CriticalUtilization"].(bool); ok && criticalUtil {
		appLogger.Error("Database connection pool utilization is critical",
			"utilization_pct", stats["UtilizationPct"],
			"recommendation", "Immediate action required: increase MaxOpenConns or investigate connection leaks")
	}
}

// startHealthMonitoring starts a goroutine that periodically monitors database health
func (d *Database) startHealthMonitoring() {
	appLogger := observability.GetLogger()
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for range ticker.C {
		if err := d.healthCheck(); err != nil {
			appLogger.Error("Database health check failed", "error", err)
		}
	}
}

// healthCheck performs a comprehensive database health check
func (d *Database) healthCheck() error {
	appLogger := observability.GetLogger()

	// Check basic connectivity
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Log connection pool statistics
	d.LogConnectionPoolStats()

	// Check for high connection utilization and log warnings
	stats := d.Stats()
	if highUtil, ok := stats["HighUtilization"].(bool); ok && highUtil {
		appLogger.Warn("High database connection utilization detected",
			"utilization_pct", stats["UtilizationPct"],
			"open_connections", stats["OpenConnections"],
			"max_open_connections", stats["MaxOpenConnections"],
			"recommendation", "Consider optimizing long-running queries or increasing connection pool size")
	}

	if criticalUtil, ok := stats["CriticalUtilization"].(bool); ok && criticalUtil {
		appLogger.Error("Critical database connection utilization detected",
			"utilization_pct", stats["UtilizationPct"],
			"open_connections", stats["OpenConnections"],
			"max_open_connections", stats["MaxOpenConnections"],
			"recommendation", "Immediate action required: investigate connection leaks or scale database resources")
	}

	return nil
}

// IsHealthy checks if the database connection is healthy
func (d *Database) IsHealthy() bool {
	return d.healthCheck() == nil
}

// GetHealthStatus returns detailed health status information
func (d *Database) GetHealthStatus() map[string]any {
	status := make(map[string]any)

	err := d.healthCheck()
	status["healthy"] = err == nil
	if err != nil {
		status["error"] = err.Error()
	}

	status["connection_stats"] = d.Stats()
	status["last_check"] = clock.Now().Format(time.RFC3339)

	return status
}
