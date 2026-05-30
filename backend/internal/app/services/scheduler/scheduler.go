package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/infra/persistence"

	"github.com/robfig/cron/v3"
)

type Job struct {
	Name    string
	Cron    string
	Handler func()
	Enabled bool
}

type Scheduler struct {
	cron     *cron.Cron
	logger   *slog.Logger
	timezone string
	enabled  bool
	jobs     []Job
	entryMap map[string]cron.EntryID
	repo     *persistence.CronJobStatusRepository
}

func NewScheduler(
	logger *slog.Logger,
	timezone string,
	enabled bool,
) *Scheduler {
	if logger == nil {
		logger = observability.GetLogger()
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		logger.Warn("Invalid timezone, using UTC", "timezone", timezone, "error", err)
		loc = time.UTC
	}

	return &Scheduler{
		cron:     cron.New(cron.WithLocation(loc)),
		logger:   logger,
		timezone: timezone,
		enabled:  enabled,
		jobs:     make([]Job, 0),
		entryMap: make(map[string]cron.EntryID),
	}
}

func (s *Scheduler) SetRepo(repo *persistence.CronJobStatusRepository) {
	s.repo = repo
}

func (s *Scheduler) AddJob(job Job) {
	s.jobs = append(s.jobs, job)
}

func (s *Scheduler) GetJobs() []Job {
	return s.jobs
}

func (s *Scheduler) Start() error {
	if !s.enabled {
		s.logger.Info("Scheduler is disabled")
		return nil
	}

	if s.repo != nil {
		s.seedJobStatuses()
	}

	for _, job := range s.jobs {
		if !job.Enabled {
			s.logger.Info("Job disabled, skipping", "name", job.Name)
			continue
		}

		if s.repo != nil && !s.isJobEnabledInDB(job.Name) {
			s.logger.Info("Job disabled in DB, skipping", "name", job.Name)
			continue
		}

		handler := s.wrapHandler(job.Name, job.Handler)
		entryID, err := s.cron.AddFunc(job.Cron, handler)
		if err != nil {
			return fmt.Errorf("failed to schedule %s: %w", job.Name, err)
		}

		s.entryMap[job.Name] = entryID
		s.logger.Info("Job registered", "name", job.Name, "cron", job.Cron)
	}

	s.cron.Start()
	s.logger.Info("Scheduler started")
	return nil
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
	s.logger.Info("Scheduler stopped")
}

// RunJobOnce manually triggers a registered job by name and runs its handler
// synchronously (using the same wrapper that records status to DB). Used by
// the admin manual-trigger endpoint for on-demand runs and QA verification.
func (s *Scheduler) RunJobOnce(ctx context.Context, jobName string) error {
	for _, job := range s.jobs {
		if job.Name == jobName {
			handler := s.wrapHandler(job.Name, job.Handler)
			handler()
			return nil
		}
	}
	return fmt.Errorf("job not found: %s", jobName)
}

func (s *Scheduler) ToggleJob(ctx context.Context, jobName string, enabled bool) error {
	if s.repo == nil {
		return fmt.Errorf("scheduler repository not configured")
	}

	if err := s.repo.UpdateEnabled(ctx, jobName, enabled); err != nil {
		return fmt.Errorf("failed to toggle job %s: %w", jobName, err)
	}

	if entryID, exists := s.entryMap[jobName]; exists {
		s.cron.Remove(entryID)
		delete(s.entryMap, jobName)
		s.logger.Info("Job removed from scheduler", "name", jobName)
	}

	if enabled {
		for _, job := range s.jobs {
			if job.Name == jobName {
				handler := s.wrapHandler(job.Name, job.Handler)
				entryID, err := s.cron.AddFunc(job.Cron, handler)
				if err != nil {
					return fmt.Errorf("failed to re-schedule %s: %w", job.Name, err)
				}
				s.entryMap[jobName] = entryID
				s.logger.Info("Job re-enabled in scheduler", "name", job.Name)
				break
			}
		}
	}

	return nil
}

func (s *Scheduler) wrapHandler(jobName string, handler func()) func() {
	return func() {
		if s.repo == nil {
			handler()
			return
		}

		ctx := context.Background()
		status := &domain.CronJobStatus{JobName: jobName}
		status.MarkRunning()
		if err := s.repo.UpdateStatus(ctx, status); err != nil {
			s.logger.Error("Failed to mark job as running", "name", jobName, "error", err)
		}

		s.runJob(ctx, jobName, handler, status)
	}
}

func (s *Scheduler) runJob(ctx context.Context, jobName string, handler func(), status *domain.CronJobStatus) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic: %v", r)
			status.MarkFailed(err)
			if updateErr := s.repo.UpdateStatus(ctx, status); updateErr != nil {
				s.logger.Error("Failed to update job status after panic", "name", jobName, "error", updateErr)
			}
			s.logger.Error("Job panicked", "name", jobName, "error", err)
			panic(r)
		}
	}()

	handler()
	status.MarkSuccess()
	if err := s.repo.UpdateStatus(ctx, status); err != nil {
		s.logger.Error("Failed to update job status after success", "name", jobName, "error", err)
	}
}

func (s *Scheduler) seedJobStatuses() {
	ctx := context.Background()
	registered := make(map[string]struct{}, len(s.jobs))
	for _, job := range s.jobs {
		registered[job.Name] = struct{}{}
		status := domain.NewCronJobStatus(job.Name, job.Cron, job.Enabled)
		if err := s.repo.Upsert(ctx, status); err != nil {
			s.logger.Error("Failed to seed job status", "name", job.Name, "error", err)
		}
	}

	existing, err := s.repo.GetAll(ctx)
	if err != nil {
		s.logger.Error("Failed to fetch existing job statuses for cleanup", "error", err)
		return
	}
	for _, job := range existing {
		if _, ok := registered[job.JobName]; !ok {
			if err := s.repo.DeleteByName(ctx, job.JobName); err != nil {
				s.logger.Error("Failed to delete orphaned job status", "name", job.JobName, "error", err)
			} else {
				s.logger.Info("Removed orphaned job status", "name", job.JobName)
			}
		}
	}
}

func (s *Scheduler) isJobEnabledInDB(jobName string) bool {
	ctx := context.Background()
	status, err := s.repo.GetByJobName(ctx, jobName)
	if err != nil || status == nil {
		return true
	}
	return status.IsEnabled
}

func IsLastDayOfMonth(date time.Time) bool {
	nextMonth := time.Date(date.Year(), date.Month()+1, 1, 0, 0, 0, 0, date.Location())
	lastDayOfMonth := nextMonth.AddDate(0, 0, -1)
	return date.Day() == lastDayOfMonth.Day()
}
