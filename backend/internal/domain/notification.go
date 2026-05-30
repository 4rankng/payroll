package domain

import (
	"context"
	"time"
)

// NotificationType represents different types of notifications
type NotificationType string

const (
	NotificationTypeTimesheetReminder            NotificationType = "timesheet_reminder"
	NotificationTypeTimesheetApproval            NotificationType = "timesheet_approval"
	NotificationTypeApprovalRequest              NotificationType = "approval_request"
	NotificationTypeApprovalResult               NotificationType = "approval_result"
	NotificationTypePayrollComplete              NotificationType = "payroll_complete"
	NotificationTypePasswordChange               NotificationType = "password_change"
	NotificationTypeLoanInterestDue              NotificationType = "loan_interest_due"
	NotificationTypePayrollReport                NotificationType = "payroll_report"
	NotificationTypePaymentScheduleChange        NotificationType = "payment_schedule_change"
	NotificationTypePaymentScheduleChangeSummary NotificationType = "payment_schedule_change_summary"
	NotificationTypeTimesheetPaid                NotificationType = "timesheet_paid"
	NotificationTypeAdvancePaymentStatusChanged  NotificationType = "advance_payment_status_changed"
	NotificationTypeAdvancePaymentReport         NotificationType = "advance_payment_report"
	NotificationTypeCustom                       NotificationType = "custom"
)

// NotificationPriority represents notification priority prefixes
const (
	NotifyTypeImportant = ""
)

// NotificationChannel represents delivery channels (extensible to email)
type NotificationChannel string

const (
	NotificationChannelPush  NotificationChannel = "push"
	NotificationChannelEmail NotificationChannel = "email"
)

// NotificationContentType represents the format of notification content
type NotificationContentType string

const (
	NotificationContentTypePlainText NotificationContentType = "plain_text"
	NotificationContentTypeHTML      NotificationContentType = "html"
	NotificationContentTypeMarkdown  NotificationContentType = "markdown"
)

// Notification represents a simple notification entity
type Notification struct {
	ID              uint                    `json:"id" gorm:"primarykey;type:bigint unsigned"`
	SenderID        uint                    `json:"sender_id" gorm:"not null;index;comment:'User who initiated/sent the notification'"`
	RecipientID     *uint                   `json:"recipient_id,omitempty" gorm:"index;comment:'User who receives the notification (null for emails to external recipients)'"`
	Type            NotificationType        `json:"type" gorm:"type:varchar(50);not null"`
	Channel         NotificationChannel     `json:"channel" gorm:"type:varchar(20);not null;default:'push'"`
	Title           string                  `json:"title" gorm:"type:varchar(255);not null"`
	Message         string                  `json:"message" gorm:"type:text;not null"`
	ContentType     NotificationContentType `json:"content_type" gorm:"type:varchar(20);not null;default:'plain_text';index:idx_notifications_content_type"`
	EmailRecipients *string                 `json:"email_recipients,omitempty" gorm:"type:text;column:email_recipients;comment:'JSON array of email recipients for email notifications'"`
	Metadata        *string                 `json:"metadata,omitempty" gorm:"type:text;column:metadata;comment:'JSON object with metadata for payroll emails'"`
	ResendMessageID *string                 `json:"resend_message_id,omitempty" gorm:"type:varchar(255);comment:'Resend email message ID for tracking'"`
	ReadAt          *time.Time              `json:"read_at"`
	CreatedAt       time.Time               `json:"created_at"`

	// Relations
	Sender    *User `json:"sender,omitempty" gorm:"foreignKey:SenderID"`
	Recipient *User `json:"recipient,omitempty" gorm:"foreignKey:RecipientID"`
}

// PayrollEmailMetadata holds financial data stored alongside a payroll email notification.
type PayrollEmailMetadata struct {
	TotalAmount   int64      `json:"totalAmount"`
	FeePercentage float64    `json:"feePercentage"`
	FeeAmount     int64      `json:"feeAmount"`
	TotalWithFee  int64      `json:"totalWithFee"`
	ReportAtDate  string     `json:"reportAtDate"`
	TimesheetIDs  []uint     `json:"timesheetIds"`
	SaoKeAssetID  *uint      `json:"saoKeAssetId,omitempty"`
	SettledAt     *time.Time `json:"settledAt,omitempty"`
	CC            []string   `json:"cc,omitempty"`
	BCC           []string   `json:"bcc,omitempty"`
}

// NotificationRepository defines the interface for simple notification operations
type NotificationRepository interface {
	Create(ctx context.Context, notification *Notification) error
	GetByID(ctx context.Context, id uint) (*Notification, error)
	GetByUserID(ctx context.Context, userID uint, limit, offset int) ([]*Notification, error)
	GetUnreadByUserID(ctx context.Context, userID uint) ([]*Notification, error)
	ListByChannel(ctx context.Context, channel NotificationChannel, limit, offset int) ([]*Notification, error)
	CountUnreadByUserID(ctx context.Context, userID uint) (int64, error)
	CountByUserID(ctx context.Context, userID uint) (int64, error)
	CountByChannel(ctx context.Context, channel NotificationChannel) (int64, error)
	MarkAsRead(ctx context.Context, id uint) error
	MarkAllAsReadForUser(ctx context.Context, userID uint) error
	MarkOldNotificationsAsRead(ctx context.Context, olderThanDays int) (int64, error)
	UpdateMetadata(ctx context.Context, id uint, metadata string) error
}
