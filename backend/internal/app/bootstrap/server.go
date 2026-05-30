package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-server/internal/config"

	"github.com/gin-gonic/gin"
)

const defaultShutdownTimeout = 10 * time.Second

type Server struct {
	config    *config.Config
	logger    *slog.Logger
	container *Container
	httpSrv   *http.Server
}

func NewServer(cfg *config.Config, container *Container) *Server {
	return &Server{
		config:    cfg,
		logger:    container.Logger,
		container: container,
	}
}

func (s *Server) Start() error {
	// Start notification scheduler
	if err := s.container.Scheduler.Start(); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	// Start Asynq task processor
	if err := s.container.AsynqServer.Start(); err != nil {
		return fmt.Errorf("failed to start asynq server: %w", err)
	}

	router := s.setupRouter()

	s.httpSrv = &http.Server{
		Addr:         ":" + s.config.App.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	httpErr := make(chan error, 1)
	go s.startHTTPServer(httpErr)

	s.waitForShutdown(httpErr)
	return nil
}

func (s *Server) setupRouter() *gin.Engine {
	if s.config.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Configure trusted proxies so that c.ClientIP() (used by rate limiters and
	// audit logging) resolves to the real client IP rather than a proxy IP.
	// TRUSTED_PROXIES accepts a comma-separated list of CIDRs/IPs; defaults to
	// loopback only. Set to your load-balancer/nginx CIDR in production.
	if err := router.SetTrustedProxies(s.config.App.TrustedProxies); err != nil {
		s.logger.Warn("Failed to set trusted proxies, falling back to loopback", "error", err)
		_ = router.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	}

	SetupMiddleware(router, s.container)
	SetupRoutes(router, s.container)

	return router
}

func (s *Server) startHTTPServer(errCh chan<- error) {
	s.logger.Info("Server starting", "port", s.config.App.Port)

	if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		errCh <- fmt.Errorf("HTTP server failed: %w", err)
	}
}

func (s *Server) waitForShutdown(httpErr <-chan error) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		s.logger.Info("Received shutdown signal", "signal", sig)
	case err := <-httpErr:
		s.logger.Error("Server encountered fatal error", "error", err)
	}

	s.logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	if err := s.httpSrv.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server forced to shutdown", "error", err)
	}

	s.cleanup()
	s.logger.Info("Server shutdown complete")
}

func (s *Server) cleanup() {
	// Stop scheduler first
	if s.container.Scheduler != nil {
		s.container.Scheduler.Stop()
	}

	// Stop Asynq server and client
	if s.container.AsynqServer != nil {
		s.container.AsynqServer.Shutdown()
	}
	if s.container.AsynqClient != nil {
		if err := s.container.AsynqClient.Close(); err != nil {
			s.logger.Error("Failed to close Asynq client", "error", err)
		}
	}

	// Signal background goroutines to stop
	s.container.Stop()

	// Close provider resources (e.g. ninepay queued provider goroutine)
	if s.container.Services != nil && s.container.Services.NinepayCloser != nil {
		if err := s.container.Services.NinepayCloser.Close(); err != nil {
			s.logger.Error("Failed to close ninepay provider", "error", err)
		}
	}

	// Stop tenant queue workers
	if s.container.TenantQueueManager != nil {
		s.container.TenantQueueManager.ShutdownAll()
	}

	if s.container.DB != nil {
		if err := s.container.DB.Close(); err != nil {
			s.logger.Error("Failed to close database", "error", err)
		}
	}
	if s.container.Redis != nil {
		if err := s.container.Redis.Close(); err != nil {
			s.logger.Error("Failed to close Redis", "error", err)
		}
	}
}
