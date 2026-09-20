package handlers

import (
	"context"

	"api-server/internal/app/services/scheduler"
	"api-server/internal/infra/persistence"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type CronHandler struct {
	repo      *persistence.CronJobStatusRepository
	scheduler *scheduler.Scheduler
}

func NewCronHandler(repo *persistence.CronJobStatusRepository, sched *scheduler.Scheduler) *CronHandler {
	return &CronHandler{repo: repo, scheduler: sched}
}

type cronJobResponse struct {
	Name       string  `json:"name"`
	Cron       string  `json:"cron"`
	IsEnabled  bool    `json:"is_enabled"`
	Status     *string `json:"last_status"`
	LastRunAt  *string `json:"last_run"`
	DurationMs *int64  `json:"last_duration_ms"`
	LastError  *string `json:"last_error"`
}

func (h *CronHandler) GetCronJobs(c *gin.Context) {
	jobs, err := h.repo.GetAll(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Failed to get cron jobs")
		return
	}

	result := make([]cronJobResponse, len(jobs))
	for i, job := range jobs {
		var lastRun *string
		if job.LastRunAt != nil {
			s := job.LastRunAt.Format("2006-01-02T15:04:05Z07:00")
			lastRun = &s
		}
		var status *string
		if job.LastRunAt != nil && job.Status != "" {
			s := string(job.Status)
			status = &s
		}
		result[i] = cronJobResponse{
			Name:       job.JobName,
			Cron:       job.Cron,
			IsEnabled:  job.IsEnabled,
			Status:     status,
			LastRunAt:  lastRun,
			DurationMs: job.DurationMs,
			LastError:  job.LastError,
		}
	}

	response.Success(c, result, "Cron jobs retrieved")
}

func (h *CronHandler) ToggleCronJob(c *gin.Context) {
	jobName := c.Param("name")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	// An unknown job is caller input: ToggleJob surfaces the repository's typed
	// not-found error, which HandleDomainError answers as 404 rather than the
	// blanket 500 that hid the distinction from clients and alerting.
	if err := h.scheduler.ToggleJob(c.Request.Context(), jobName, req.Enabled); err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, gin.H{"name": jobName, "is_enabled": req.Enabled}, "Cron job toggled")
}

// RunCronJob asynchronously kicks off a registered cron job by name. The
// handler is wrapped with the same status-recording wrapper used by the cron
// tick, so progress can be observed via GetCronJobs (last_run, last_status).
// We run it in a goroutine so that the HTTP request returns immediately even
// if the handler does long DB work — the request itself only validates that
// the job exists.
func (h *CronHandler) RunCronJob(c *gin.Context) {
	jobName := c.Param("name")
	// Validate that the job exists before kicking it off.
	found := false
	for _, j := range h.scheduler.GetJobs() {
		if j.Name == jobName {
			found = true
			break
		}
	}
	if !found {
		// Same status as the toggle path: the named job is not a resource here.
		response.NotFound(c, "Job not found: "+jobName)
		return
	}
	go func() {
		// Use a fresh context — the request context is cancelled as soon as
		// the HTTP response is written, which would abort the underlying
		// transaction mid-flight.
		_ = h.scheduler.RunJobOnce(context.Background(), jobName)
	}()
	response.Success(c, gin.H{"name": jobName, "started": true}, "Cron job triggered (async)")
}
