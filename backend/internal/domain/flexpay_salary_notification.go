package domain

import (
	"context"
	"time"
)

const (
	FlexPaySalaryNotificationPending    = "pending"
	FlexPaySalaryNotificationProcessing = "processing"
	FlexPaySalaryNotificationSent       = "sent"
	FlexPaySalaryNotificationFailed     = "failed"
	FlexPaySalaryNotificationSuppressed = "suppressed"
)

// FlexPaySalaryNotification is the durable delivery record for one salary
// notification. Its unique upload/project/employee/template key makes an
// import retry safe while retaining a complete delivery audit.
type FlexPaySalaryNotification struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	AssetID        uint       `json:"asset_id"`
	ProjectID      uint       `json:"project_id"`
	EmployeeID     uint       `json:"employee_id"`
	TemplateID     string     `json:"template_id"`
	Phone          string     `json:"-"`
	CustomerName   string     `json:"-"`
	MaxAmount      int64      `json:"-"`
	ExpiryDate     time.Time  `json:"-"`
	Status         string     `json:"status"`
	Attempt        uint       `json:"attempt"`
	LeaseExpiresAt *time.Time `json:"lease_expires_at,omitempty"`
	EnqueuedAt     *time.Time `json:"enqueued_at,omitempty"`
	SentAt         *time.Time `json:"sent_at,omitempty"`
	ProviderMsgID  string     `json:"provider_msg_id,omitempty"`
	ProviderCode   int        `json:"provider_code,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (FlexPaySalaryNotification) TableName() string { return "flexpay_salary_notifications" }

func (n FlexPaySalaryNotification) IsTerminal() bool {
	return n.Status == FlexPaySalaryNotificationSent ||
		n.Status == FlexPaySalaryNotificationFailed ||
		n.Status == FlexPaySalaryNotificationSuppressed
}

type FlexPaySalaryNotificationRepository interface {
	CreateIfAbsent(ctx context.Context, notification *FlexPaySalaryNotification) (*FlexPaySalaryNotification, bool, error)
	Claim(ctx context.Context, id uint, now, leaseUntil time.Time) (*FlexPaySalaryNotification, bool, error)
	ReleaseForRetry(ctx context.Context, id uint, attempt uint, reason string) error
	MarkSent(ctx context.Context, id uint, attempt uint, sentAt time.Time, providerMsgID string) error
	MarkFailed(ctx context.Context, id uint, attempt uint, failedAt time.Time, reason string, providerCode int) error
	MarkSuppressed(ctx context.Context, id uint, attempt uint, at time.Time, reason string, providerCode int) error
	ListRecoverable(ctx context.Context, now time.Time, limit int) ([]*FlexPaySalaryNotification, error)
}
