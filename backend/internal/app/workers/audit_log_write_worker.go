package workers

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/ipgeo"
	"api-server/internal/pkg/ua"
)

// AuditLogWriteJob is the domain-level job passed from the asynq handler to the worker.
type AuditLogWriteJob struct {
	UserID       uint
	Action       string
	EntityType   string
	EntityID     *uint
	Message      string
	IPAddress    string
	UserAgent    string
	MetadataJSON string
	CreatedAt    time.Time
}

// AuditLogWriteWorker persists audit log entries dispatched via the audit:log:write asynq task.
type AuditLogWriteWorker struct {
	auditRepo domain.AuditLogRepository
	logger    *slog.Logger
}

// NewAuditLogWriteWorker creates a new AuditLogWriteWorker.
func NewAuditLogWriteWorker(auditRepo domain.AuditLogRepository) *AuditLogWriteWorker {
	return &AuditLogWriteWorker{
		auditRepo: auditRepo,
		logger:    slog.Default().With("component", "AuditLogWriteWorker"),
	}
}

// ProcessJob writes an audit log row to the database.
// Asynq retries on non-nil return, so transient DB errors are automatically recovered.
func (w *AuditLogWriteWorker) ProcessJob(ctx context.Context, job AuditLogWriteJob) error {
	var browser, platform *string
	if job.UserAgent != "" {
		b, p := ua.Parse(job.UserAgent)
		if b != "" && b != "Unknown" {
			browser = &b
		}
		if p != "" && p != "Unknown" {
			platform = &p
		}
	}

	var meta map[string]interface{}
	if job.MetadataJSON != "" {
		if err := json.Unmarshal([]byte(job.MetadataJSON), &meta); err != nil {
			w.logger.Error("audit: failed to unmarshal metadata", "error", err)
			meta = map[string]interface{}{}
		}
	}

	// IP geolocation — runs here (not in the event-bus goroutine) so the 3-second
	// lookup benefits from asynq retries and doesn't stall event-bus workers.
	if job.IPAddress != "" {
		if _, hasLocation := meta["location"]; !hasLocation {
			geoCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if loc, err := ipgeo.Lookup(geoCtx, job.IPAddress); err == nil && loc != nil {
				if meta == nil {
					meta = make(map[string]interface{})
				}
				meta["location"] = map[string]string{
					"country": loc.Country,
					"city":    loc.City,
					"region":  loc.Region,
				}
			}
		}
	}

	createdAt := job.CreatedAt
	if createdAt.IsZero() {
		createdAt = clock.Now()
	}

	var ipPtr *string
	if job.IPAddress != "" {
		ipPtr = &job.IPAddress
	}

	auditLog := &domain.AuditLog{
		UserID:     job.UserID,
		Action:     domain.AuditAction(job.Action),
		EntityType: domain.EntityType(job.EntityType),
		EntityID:   job.EntityID,
		Message:    job.Message,
		IPAddress:  ipPtr,
		Browser:    browser,
		Platform:   platform,
		CreatedAt:  createdAt,
	}

	if len(meta) > 0 {
		if err := auditLog.SetMetadata(meta); err != nil {
			w.logger.Error("audit: failed to set metadata", "error", err)
		}
	}

	return w.auditRepo.Create(ctx, auditLog)
}
