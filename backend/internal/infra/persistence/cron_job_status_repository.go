package persistence

import (
	"context"
	"fmt"

	"api-server/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CronJobStatusRepository struct {
	*BaseRepository
}

func NewCronJobStatusRepository(db *Database) *CronJobStatusRepository {
	return &CronJobStatusRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *CronJobStatusRepository) Upsert(ctx context.Context, status *domain.CronJobStatus) error {
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "job_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"cron", "is_enabled"}),
	}).Create(status).Error
}

func (r *CronJobStatusRepository) GetAll(ctx context.Context) ([]domain.CronJobStatus, error) {
	var jobs []domain.CronJobStatus
	if err := r.DB.WithContext(ctx).Order("job_name ASC").Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("failed to get cron job statuses: %w", err)
	}
	return jobs, nil
}

func (r *CronJobStatusRepository) GetByJobName(ctx context.Context, jobName string) (*domain.CronJobStatus, error) {
	var job domain.CronJobStatus
	if err := r.DB.WithContext(ctx).Where("job_name = ?", jobName).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get cron job status: %w", err)
	}
	return &job, nil
}

func (r *CronJobStatusRepository) UpdateEnabled(ctx context.Context, jobName string, enabled bool) error {
	result := r.DB.WithContext(ctx).
		Model(&domain.CronJobStatus{}).
		Where("job_name = ?", jobName).
		Update("is_enabled", enabled)
	if result.Error != nil {
		return fmt.Errorf("failed to update cron job enabled status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.NewNotFoundError(fmt.Sprintf("Không tìm thấy cron job %s", jobName))
	}
	return nil
}

func (r *CronJobStatusRepository) UpdateStatus(ctx context.Context, status *domain.CronJobStatus) error {
	return r.DB.WithContext(ctx).
		Model(&domain.CronJobStatus{}).
		Where("job_name = ?", status.JobName).
		Updates(map[string]any{
			"status":      status.Status,
			"last_run_at": status.LastRunAt,
			"duration_ms": status.DurationMs,
			"last_error":  status.LastError,
			"updated_at":  status.UpdatedAt,
		}).Error
}

func (r *CronJobStatusRepository) DeleteByName(ctx context.Context, jobName string) error {
	return r.DB.WithContext(ctx).
		Where("job_name = ?", jobName).
		Delete(&domain.CronJobStatus{}).Error
}
