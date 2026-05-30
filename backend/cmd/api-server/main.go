package main

import (
	"fmt"
	"net/http"
	"os"

	"api-server/internal/app/bootstrap"
	"api-server/internal/config"
	"api-server/internal/infra/observability"
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
	// Handle healthcheck command
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		healthcheck()
		return
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
