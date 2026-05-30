package domain

import (
	"time"

	"api-server/internal/pkg/clock"
)

type CronJobStatusType string

const (
	CronJobStatusRunning CronJobStatusType = "running"
	CronJobStatusSuccess CronJobStatusType = "success"
	CronJobStatusFailed  CronJobStatusType = "failed"
)

type CronJobStatus struct {
	ID         uint              `gorm:"primaryKey"`
	JobName    string            `gorm:"column:job_name;type:varchar(100);uniqueIndex:uk_job_name;not null"`
	Cron       string            `gorm:"column:cron;type:varchar(50);not null;default:''"`
	IsEnabled  bool              `gorm:"column:is_enabled;not null;default:true"`
	Status     CronJobStatusType `gorm:"column:status;type:varchar(20);not null;default:''"`
	LastRunAt  *time.Time        `gorm:"column:last_run_at"`
	DurationMs *int64            `gorm:"column:duration_ms"`
	LastError  *string           `gorm:"column:last_error;type:text"`
	UpdatedAt  *time.Time        `gorm:"column:updated_at"`
}

func (CronJobStatus) TableName() string {
	return "cron_job_status"
}

func NewCronJobStatus(jobName, cronExpr string, enabled bool) *CronJobStatus {
	return &CronJobStatus{
		JobName:   jobName,
		Cron:      cronExpr,
		IsEnabled: enabled,
		Status:    "",
	}
}

func (s *CronJobStatus) MarkRunning() {
	now := clock.Now()
	s.Status = CronJobStatusRunning
	s.LastRunAt = &now
	s.LastError = nil
	s.DurationMs = nil
}

func (s *CronJobStatus) MarkSuccess() {
	s.Status = CronJobStatusSuccess
	s.setDuration()
	s.LastError = nil
}

func (s *CronJobStatus) MarkFailed(err error) {
	s.Status = CronJobStatusFailed
	s.setDuration()
	if err != nil {
		errMsg := err.Error()
		s.LastError = &errMsg
	}
}

func (s *CronJobStatus) setDuration() {
	if s.LastRunAt != nil {
		ms := time.Since(*s.LastRunAt).Milliseconds()
		s.DurationMs = &ms
		now := clock.Now()
		s.UpdatedAt = &now
	}
}
