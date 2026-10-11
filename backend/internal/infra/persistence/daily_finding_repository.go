package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"api-server/internal/domain"
)

// DailyFindingRepository persists the daily ops digest.
type DailyFindingRepository struct {
	db *Database
}

// NewDailyFindingRepository constructs the repository.
func NewDailyFindingRepository(db *Database) domain.DailyFindingRepository {
	return &DailyFindingRepository{db: db}
}

// Create inserts one finding; a duplicate (date, fingerprint) is a no-op so
// re-running the digest for a date stays idempotent.
func (r *DailyFindingRepository) Create(ctx context.Context, finding *domain.DailyFinding) error {
	err := r.db.DB.WithContext(ctx).Create(finding).Error
	if err != nil && isDuplicateEntry(err) {
		return nil
	}
	return err
}

// ListByDate returns the date's findings, newest severity first.
func (r *DailyFindingRepository) ListByDate(ctx context.Context, date string) ([]*domain.DailyFinding, error) {
	var out []*domain.DailyFinding
	err := r.db.DB.WithContext(ctx).
		Where("`date` = ?", date).
		Order("FIELD(severity, 'critical', 'warning', 'info'), created_at ASC").
		Find(&out).Error
	return out, err
}

// RecentlySeenFingerprints returns fingerprints recorded strictly before the
// date, within the window — the dedup set.
func (r *DailyFindingRepository) RecentlySeenFingerprints(ctx context.Context, date string, windowDays int) (map[string]bool, error) {
	type row struct {
		Fingerprint string `gorm:"column:fingerprint"`
	}
	var rows []row
	err := r.db.DB.WithContext(ctx).
		Model(&domain.DailyFinding{}).
		Select("fingerprint").
		Where("`date` < ? AND `date` >= DATE_SUB(?, INTERVAL ? DAY)", date, date, windowDays).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[r.Fingerprint] = true
	}
	return out, nil
}

// DismissedTypes returns finding types dismissed within the window before the
// date — dismissed families stay suppressed.
func (r *DailyFindingRepository) DismissedTypes(ctx context.Context, before string, windowDays int) (map[string]bool, error) {
	type row struct {
		Type string `gorm:"column:type"`
	}
	var rows []row
	err := r.db.DB.WithContext(ctx).
		Model(&domain.DailyFinding{}).
		Select("DISTINCT `type`").
		Where("dismissed_at IS NOT NULL AND dismissed_at < ? AND dismissed_at >= DATE_SUB(?, INTERVAL ? DAY)", before, before, windowDays).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[r.Type] = true
	}
	return out, nil
}

// GetByID loads one finding.
func (r *DailyFindingRepository) GetByID(ctx context.Context, id uint) (*domain.DailyFinding, error) {
	var finding domain.DailyFinding
	err := r.db.DB.WithContext(ctx).First(&finding, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.NewNotFoundError("không tìm thấy phát hiện")
	}
	if err != nil {
		return nil, err
	}
	return &finding, nil
}

// Dismiss marks the finding (and by extension its family) dismissed.
func (r *DailyFindingRepository) Dismiss(ctx context.Context, id uint, at time.Time) error {
	return r.db.DB.WithContext(ctx).
		Model(&domain.DailyFinding{}).
		Where("id = ?", id).
		Update("dismissed_at", at).Error
}
