package domain

import (
	"context"
	"time"
)

const (
	TimesheetImportStatusPending    = "pending"
	TimesheetImportStatusProcessing = "processing"
	TimesheetImportStatusCompleted  = "completed"
	TimesheetImportStatusFailed     = "failed"
)

type TimesheetImportJob struct {
	AssetID            uint       `json:"asset_id" gorm:"primaryKey;column:asset_id"`
	ProjectID          uint       `json:"project_id"`
	ForMonth           string     `json:"for_month"`
	UploadedBy         uint       `json:"uploaded_by"`
	UploaderRole       string     `json:"uploader_role"`
	Status             string     `json:"status"`
	IdempotencyKey     string     `json:"-" gorm:"column:idempotency_key"`
	RequestFingerprint string     `json:"-" gorm:"column:request_fingerprint"`
	ActiveScopeKey     *string    `json:"-" gorm:"column:active_scope_key"`
	Attempt            uint       `json:"attempt"`
	LeaseExpiresAt     *time.Time `json:"lease_expires_at,omitempty"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	ProcessedAt        *time.Time `json:"processed_at,omitempty"`
	AuditLoggedAt      *time.Time `json:"-"`
	LastError          *string    `json:"last_error,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (TimesheetImportJob) TableName() string { return "timesheet_import_jobs" }

func (j TimesheetImportJob) IsTerminal() bool {
	return j.Status == TimesheetImportStatusCompleted || j.Status == TimesheetImportStatusFailed
}

type TimesheetImportJobRepository interface {
	Create(ctx context.Context, job *TimesheetImportJob) error
	GetByAssetID(ctx context.Context, assetID uint) (*TimesheetImportJob, error)
	GetByIdempotencyKey(ctx context.Context, uploadedBy uint, key string) (*TimesheetImportJob, error)
	Claim(ctx context.Context, assetID uint, leaseUntil time.Time) (uint, bool, error)
	LockProcessingAttempt(ctx context.Context, assetID uint, attempt uint) error
	ReleaseForRetry(ctx context.Context, assetID uint, attempt uint, reason string) error
	Complete(ctx context.Context, assetID uint, attempt uint, processedAt time.Time) error
	Fail(ctx context.Context, assetID uint, attempt uint, reason string, processedAt time.Time) error
	MarkTerminalAuditLogged(ctx context.Context, assetID uint, auditedAt time.Time) (bool, error)
	ListRecoverable(ctx context.Context, limit int, now time.Time) ([]*TimesheetImportJob, error)
}

type TimesheetImportEnqueuer interface {
	EnqueueBCCImport(assetID uint) error
}
