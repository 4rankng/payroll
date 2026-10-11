package domain

import (
	"context"
	"time"
)

// DailyFindingSeverity orders the digest: critical findings may push, the
// rest live quietly on the Tong Quan card.
type DailyFindingSeverity string

const (
	DailySeverityInfo     DailyFindingSeverity = "info"
	DailySeverityWarning  DailyFindingSeverity = "warning"
	DailySeverityCritical DailyFindingSeverity = "critical"
)

// EntityTypeDailyFinding identifies digest findings in audit_logs.
const EntityTypeDailyFinding EntityType = "daily_finding"

// DailyFinding is one deterministic collector signal surfaced on the digest.
// Facts are computed in Go (JSON); Narrative is LLM-written prose that may
// drop items but never invent or alter numbers.
type DailyFinding struct {
	ID          uint                 `json:"id" gorm:"primarykey;type:bigint unsigned"`
	Date        string               `json:"date" gorm:"type:varchar(10);not null;uniqueIndex:idx_daily_findings_date_fingerprint;comment:'Digest date YYYY-MM-DD'"`
	Fingerprint string               `json:"fingerprint" gorm:"type:varchar(191);not null;uniqueIndex:idx_daily_findings_date_fingerprint;comment:'type + entity key for dedup'"`
	Type        string               `json:"type" gorm:"type:varchar(64);not null;index;comment:'Collector type (suppression family)'"`
	Severity    DailyFindingSeverity `json:"severity" gorm:"type:varchar(16);not null;default:'info'"`
	Facts       string               `json:"facts" gorm:"type:json;not null;comment:'Deterministic facts (quoted verbatim in narrative)'"`
	Narrative   string               `json:"narrative" gorm:"type:text;comment:'LLM Vietnamese narrative; empty when the provider failed'"`
	DismissedAt *time.Time           `json:"dismissed_at" gorm:"type:datetime(3);comment:'Admin marked this family not useful'"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// Dismissed reports whether an admin suppressed this finding.
func (f *DailyFinding) Dismissed() bool { return f.DismissedAt != nil }

// DailyFindingRepository is the persistence port for digest findings.
type DailyFindingRepository interface {
	Create(ctx context.Context, finding *DailyFinding) error
	ListByDate(ctx context.Context, date string) ([]*DailyFinding, error)
	// RecentlySeenFingerprints returns fingerprints seen within the given
	// window before `date` — the 7-day dedup set.
	RecentlySeenFingerprints(ctx context.Context, date string, windowDays int) (map[string]bool, error)
	// DismissedByType returns finding types dismissed within the window —
	// dismissed families stay suppressed.
	DismissedTypes(ctx context.Context, before string, windowDays int) (map[string]bool, error)
	GetByID(ctx context.Context, id uint) (*DailyFinding, error)
	Dismiss(ctx context.Context, id uint, at time.Time) error
}
