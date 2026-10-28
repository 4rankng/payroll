package domain

import (
	"context"
	"time"
)

// ZnsMessage is THE audit record for one ZNS send attempt, whatever feature
// triggered it. Rows are written at the provider layer (infra/zalo) right
// after every send call — success or failure — so all ZNS traffic is captured
// in one place.
//
// SINGLE-TABLE DOCTRINE: zns_messages is the ONLY table admin ZNS tracking
// reads and the ONLY place ZNS delivery history lives. Do NOT create per-flow
// ZNS tables in the future — tag the send with its Feature instead. The
// flexpay_salary_notifications table keeps its internal retry-queue role
// (claim/lease) and is never an admin data source.
//
// Status vocabulary: the provider layer writes "sent"/"failed"; backfilled
// rows retain the flexpay queue states ("pending"/"processing"/"suppressed").
type ZnsMessage struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Feature       string    `json:"feature"`
	TemplateID    string    `json:"template_id"`
	Phone         string    `json:"phone"`
	Status        string    `json:"status"`
	ProviderMsgID string    `json:"provider_msg_id"`
	ProviderCode  int       `json:"provider_code"`
	ErrorMsg      string    `json:"error_msg"`
	DurationMs    int64     `json:"duration_ms"`
	CreatedAt     time.Time `json:"created_at"`
}

func (ZnsMessage) TableName() string { return "zns_messages" }

const (
	ZnsStatusSent       = "sent"
	ZnsStatusFailed     = "failed"
	ZnsStatusPending    = "pending"
	ZnsStatusProcessing = "processing"
	ZnsStatusSuppressed = "suppressed"
)

// ZnsMessageFilter carries the admin-tracking query: every filter combines
// (AND); Statuses is a multi-select, Query matches phone or provider message
// id. FromDate/ToDate bound created_at (send time) inclusively at day grain.
type ZnsMessageFilter struct {
	Statuses   []string
	Feature    string
	TemplateID string
	FromDate   *time.Time
	ToDate     *time.Time
	Query      string
	Page       int
	PageSize   int
}

type ZnsMessageRepository interface {
	// Record appends one send-attempt row. Errors are the repository's only
	// failure signal — callers decide whether recording is best-effort.
	Record(ctx context.Context, msg *ZnsMessage) error
	// ListAdmin returns one page newest-first plus the total matching count.
	ListAdmin(ctx context.Context, f ZnsMessageFilter) ([]*ZnsMessage, int64, error)
}
