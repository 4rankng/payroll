package persistence

import (
	"context"
	"errors"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TimesheetImportJobRepository struct {
	*BaseRepository
}

func NewTimesheetImportJobRepository(db *Database) domain.TimesheetImportJobRepository {
	return &TimesheetImportJobRepository{BaseRepository: NewBaseRepository(db)}
}

func (r *TimesheetImportJobRepository) Create(ctx context.Context, job *domain.TimesheetImportJob) error {
	if err := r.db(ctx).Create(job).Error; err != nil {
		return domain.NewInternalError("failed to create timesheet import job", err)
	}
	return nil
}

func (r *TimesheetImportJobRepository) GetByAssetID(ctx context.Context, assetID uint) (*domain.TimesheetImportJob, error) {
	var job domain.TimesheetImportJob
	if err := r.db(ctx).First(&job, "asset_id = ?", assetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("timesheet import job not found")
		}
		return nil, domain.NewInternalError("failed to get timesheet import job", err)
	}
	return &job, nil
}

func (r *TimesheetImportJobRepository) GetByIdempotencyKey(ctx context.Context, uploadedBy uint, key string) (*domain.TimesheetImportJob, error) {
	var job domain.TimesheetImportJob
	err := r.db(ctx).
		Where("uploaded_by = ? AND idempotency_key = ?", uploadedBy, key).
		First(&job).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("timesheet import job not found")
		}
		return nil, domain.NewInternalError("failed to get timesheet import job by idempotency key", err)
	}
	return &job, nil
}

func (r *TimesheetImportJobRepository) Claim(ctx context.Context, assetID uint, leaseUntil time.Time) (uint, bool, error) {
	now := clock.Now()
	result := r.db(ctx).Model(&domain.TimesheetImportJob{}).
		Where(
			"asset_id = ? AND (status = ? OR (status = ? AND lease_expires_at < ?))",
			assetID,
			domain.TimesheetImportStatusPending,
			domain.TimesheetImportStatusProcessing,
			now,
		).
		Updates(map[string]any{
			"status":           domain.TimesheetImportStatusProcessing,
			"lease_expires_at": leaseUntil,
			"started_at":       gorm.Expr("COALESCE(started_at, ?)", now),
			"attempt":          gorm.Expr("attempt + 1"),
		})
	if result.Error != nil {
		return 0, false, domain.NewInternalError("failed to claim timesheet import job", result.Error)
	}
	if result.RowsAffected != 1 {
		return 0, false, nil
	}
	job, err := r.GetByAssetID(ctx, assetID)
	if err != nil {
		return 0, false, err
	}
	return job.Attempt, true, nil
}

func (r *TimesheetImportJobRepository) LockProcessingAttempt(ctx context.Context, assetID uint, attempt uint) error {
	var job domain.TimesheetImportJob
	err := r.db(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("asset_id = ? AND status = ? AND attempt = ?",
			assetID, domain.TimesheetImportStatusProcessing, attempt).
		First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.NewConflictError("timesheet import attempt is no longer active")
	}
	if err != nil {
		return domain.NewInternalError("failed to lock timesheet import attempt", err)
	}
	return nil
}

func (r *TimesheetImportJobRepository) ReleaseForRetry(ctx context.Context, assetID uint, attempt uint, reason string) error {
	result := r.db(ctx).Model(&domain.TimesheetImportJob{}).
		Where("asset_id = ? AND status = ? AND attempt = ?",
			assetID, domain.TimesheetImportStatusProcessing, attempt).
		Updates(map[string]any{
			"status":           domain.TimesheetImportStatusPending,
			"last_error":       reason,
			"lease_expires_at": nil,
		})
	if result.Error != nil {
		return domain.NewInternalError("failed to release timesheet import job for retry", result.Error)
	}
	return nil
}

func (r *TimesheetImportJobRepository) Complete(ctx context.Context, assetID uint, attempt uint, processedAt time.Time) error {
	return r.finish(ctx, assetID, attempt, domain.TimesheetImportStatusCompleted, nil, processedAt)
}

func (r *TimesheetImportJobRepository) Fail(ctx context.Context, assetID uint, attempt uint, reason string, processedAt time.Time) error {
	return r.finish(ctx, assetID, attempt, domain.TimesheetImportStatusFailed, &reason, processedAt)
}

func (r *TimesheetImportJobRepository) finish(
	ctx context.Context,
	assetID uint,
	attempt uint,
	status string,
	reason *string,
	processedAt time.Time,
) error {
	result := r.db(ctx).Model(&domain.TimesheetImportJob{}).
		Where("asset_id = ? AND status = ? AND attempt = ?",
			assetID, domain.TimesheetImportStatusProcessing, attempt).
		Updates(map[string]any{
			"status":           status,
			"last_error":       reason,
			"processed_at":     processedAt,
			"lease_expires_at": nil,
			"active_scope_key": nil,
		})
	if result.Error != nil {
		return domain.NewInternalError("failed to finish timesheet import job", result.Error)
	}
	if result.RowsAffected != 1 {
		return domain.NewConflictError("timesheet import attempt is no longer active")
	}
	return nil
}

func (r *TimesheetImportJobRepository) ListRecoverable(
	ctx context.Context,
	limit int,
	now time.Time,
) ([]*domain.TimesheetImportJob, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var jobs []*domain.TimesheetImportJob
	err := r.db(ctx).
		Where("status = ? OR (status = ? AND lease_expires_at < ?)",
			domain.TimesheetImportStatusPending,
			domain.TimesheetImportStatusProcessing,
			now,
		).
		Order("created_at ASC").
		Limit(limit).
		Find(&jobs).Error
	if err != nil {
		return nil, domain.NewInternalError("failed to list recoverable timesheet import jobs", err)
	}
	return jobs, nil
}

func (r *TimesheetImportJobRepository) MarkTerminalAuditLogged(
	ctx context.Context,
	assetID uint,
	auditedAt time.Time,
) (bool, error) {
	result := r.db(ctx).Model(&domain.TimesheetImportJob{}).
		Where("asset_id = ? AND status IN ? AND audit_logged_at IS NULL",
			assetID,
			[]string{domain.TimesheetImportStatusCompleted, domain.TimesheetImportStatusFailed},
		).
		Update("audit_logged_at", auditedAt)
	if result.Error != nil {
		return false, domain.NewInternalError("failed to mark timesheet import audit", result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (r *TimesheetImportJobRepository) db(ctx context.Context) *gorm.DB {
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.DB.WithContext(ctx)
}
