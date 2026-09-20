package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"api-server/internal/app/bootstrap"
	bootstrapInfra "api-server/internal/app/bootstrap/infrastructure"
	bootstrapRepos "api-server/internal/app/bootstrap/repositories"
	"api-server/internal/app/services/project"
	"api-server/internal/config"
	"api-server/internal/infra/events"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
)

const API_SERVER_VERSION = "v1.10.0"

// @title API Server
// @version 1.0
// @description Production-grade Go API server with clean architecture
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1
func main() {
	// One-off maintenance commands run before the HTTP server boots.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "healthcheck":
			healthcheck()
			return
		case "recompute-project-aggregates":
			recomputeProjectAggregates()
			return
		default:
			fmt.Printf("unknown command %q (available: healthcheck, recompute-project-aggregates)\n", os.Args[1])
			os.Exit(2)
		}
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize singleton logger with proper configuration
	loggerConfig := observability.LogConfig{
		Level: cfg.Log.Level,
		File:  cfg.Log.File,
	}
	logger, err := observability.NewLogger(loggerConfig)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	observability.SetLogger(logger)

	// Bootstrap application
	container, err := bootstrap.NewContainer(cfg, API_SERVER_VERSION)
	if err != nil {
		observability.GetLogger().Error("Failed to initialize application", "error", err)
		os.Exit(1)
	}

	container.Logger.Info("Starting API server", "env", cfg.App.Env, "port", cfg.App.Port)

	// Start server
	server := bootstrap.NewServer(cfg, container)
	if err := server.Start(); err != nil {
		container.Logger.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}

// healthcheck is used for Docker health checks
func healthcheck() {
	resp, err := http.Get("http://localhost:8080/api/healthz")
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Health check failed with status: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	fmt.Println("Health check passed")
}

// recomputeProjectAggregates is the one-off backfill for the denormalised
// projects.*_vnd totals (Tổng đã chi / Chờ chi / Tổng đã nhận / Chờ thu).
//
// The runtime projection only runs for a project when its bulk-transfer payment
// worker settles timesheets, so projects that predate that wiring keep zeroed
// totals forever. This command recomputes every project from the authoritative
// timesheet columns and writes the four totals; it is idempotent, so running it
// twice is safe. Usage (inside the backend container):
//
//	/api-server recompute-project-aggregates
func recomputeProjectAggregates() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	infra, err := bootstrapInfra.Initialize(cfg)
	if err != nil {
		fmt.Printf("Failed to initialize infrastructure: %v\n", err)
		os.Exit(1)
	}

	// A local event bus is required by the repository factory only; this command
	// publishes no events, so a minimal pool is enough and nothing is started.
	eventBus := events.NewWorkerPoolEventBus(1, 8)
	defer func() { _ = eventBus.Shutdown(context.Background()) }()

	repos := bootstrapRepos.Initialize(infra.DB, eventBus)
	svc := project.NewAggregateRecomputeService(repos.Project, clock.New(), infra.Logger)

	result, err := svc.RecomputeAllProjects(context.Background())
	if err != nil {
		fmt.Printf("Recompute failed: %v\n", err)
		os.Exit(1)
	}

	infra.Logger.Info("project financial aggregates recomputed",
		"recomputed", result.Recomputed, "failed", result.Failed)
	if result.Failed > 0 {
		os.Exit(1)
	}
}
