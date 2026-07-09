package asynq

import (
	"context"

	asynqlib "github.com/hibiken/asynq"

	"api-server/internal/config"
	"api-server/internal/pkg/clock"
)

// Server wraps asynq.Server with lifecycle management
type Server struct {
	server    *asynqlib.Server
	mux       *asynqlib.ServeMux
	scheduler *asynqlib.Scheduler
	cfg       config.AsynqConfig
}

// NewServer creates a new Asynq server with the configured queues
func NewServer(cfg config.AsynqConfig) *Server {
	redisOpt := asynqlib.RedisClientOpt{
		Addr: cfg.RedisAddr,
		DB:   cfg.RedisDB,
	}

	server := asynqlib.NewServer(redisOpt, asynqlib.Config{
		Concurrency: cfg.Concurrency,
		Queues: map[string]int{
			QueueCritical: 6,
			QueueDefault:  3,
			QueueLow:      1,
		},
		ErrorHandler: asynqlib.ErrorHandlerFunc(func(ctx context.Context, task *asynqlib.Task, err error) {
			logger.Error("Task failed",
				"type", task.Type(),
				"error", err,
			)
		}),
	})

	// Evaluate crontab specs (e.g. wallet:settlement's "1 0 * * *") in the
	// business timezone, not asynq's UTC default. @every jobs are unaffected.
	scheduler := asynqlib.NewScheduler(redisOpt, &asynqlib.SchedulerOpts{
		Location: clock.DefaultLocation,
	})

	return &Server{
		server:    server,
		mux:       asynqlib.NewServeMux(),
		scheduler: scheduler,
		cfg:       cfg,
	}
}

// Mux returns the serve mux for handler registration
func (s *Server) Mux() *asynqlib.ServeMux {
	return s.mux
}

// Scheduler returns the scheduler for periodic task registration
func (s *Server) Scheduler() *asynqlib.Scheduler {
	return s.scheduler
}

// Start starts the asynq server and scheduler
func (s *Server) Start() error {
	go func() {
		if err := s.scheduler.Run(); err != nil {
			logger.Error("Asynq scheduler stopped", "error", err)
		}
	}()

	go func() {
		logger.Info("Starting asynq task processor", "concurrency", s.cfg.Concurrency)
		if err := s.server.Run(s.mux); err != nil {
			logger.Error("Asynq server stopped", "error", err)
		}
	}()

	return nil
}

// Shutdown gracefully stops the server and scheduler
func (s *Server) Shutdown() {
	s.server.Shutdown()
	s.scheduler.Shutdown()
	logger.Info("Asynq server and scheduler shutdown complete")
}
