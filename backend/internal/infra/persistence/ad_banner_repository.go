package persistence

import (
	"context"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AdBannerRepository persists ad banner campaigns and their append-only CTA
// click ledger.
type AdBannerRepository struct {
	DB *Database
}

func NewAdBannerRepository(db *Database) *AdBannerRepository {
	return &AdBannerRepository{DB: db}
}

func (r *AdBannerRepository) Create(ctx context.Context, banner *domain.AdBanner) error {
	return r.DB.WithContext(ctx).Create(banner).Error
}

func (r *AdBannerRepository) GetByID(ctx context.Context, id uint) (*domain.AdBanner, error) {
	var banner domain.AdBanner
	if err := r.DB.WithContext(ctx).First(&banner, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("Không tìm thấy chiến dịch quảng cáo")
		}
		return nil, err
	}
	return &banner, nil
}

func (r *AdBannerRepository) Update(ctx context.Context, banner *domain.AdBanner) error {
	return r.DB.WithContext(ctx).Save(banner).Error
}

func (r *AdBannerRepository) Delete(ctx context.Context, id uint) error {
	return r.DB.WithContext(ctx).Delete(&domain.AdBanner{}, id).Error
}

// List returns every non-deleted campaign, newest first. Expired and paused
// campaigns stay visible: their click counts are the campaign's record.
// Filters are part of the port for future callers; the admin list always
// wants the full history, so nothing is filtered here yet.
func (r *AdBannerRepository) List(ctx context.Context, _ domain.AdBannerFilters) ([]*domain.AdBanner, error) {
	var banners []*domain.AdBanner
	err := r.DB.WithContext(ctx).
		Model(&domain.AdBanner{}).
		Order("created_at DESC").
		Find(&banners).Error
	return banners, err
}

// ListLiveAt returns campaigns live at t in resolution order: priority DESC,
// then created_at DESC. The window predicate mirrors the domain half-open
// interval exactly (start inclusive, end exclusive).
func (r *AdBannerRepository) ListLiveAt(ctx context.Context, t time.Time) ([]*domain.AdBanner, error) {
	var banners []*domain.AdBanner
	err := r.DB.WithContext(ctx).
		Where("is_active = ? AND starts_at <= ? AND ends_at > ?", true, t, t).
		Order("priority DESC, created_at DESC").
		Find(&banners).Error
	return banners, err
}

// RecordCTAClick appends a tap. The (banner_id, employee_id, cta_index)
// unique key plus DoNothing makes repeated taps idempotent — counters count
// workers, not taps-per-worker.
func (r *AdBannerRepository) RecordCTAClick(ctx context.Context, bannerID, employeeID uint, ctaIndex int) error {
	click := &domain.AdBannerCTAClick{
		BannerID:   bannerID,
		EmployeeID: employeeID,
		CTAIndex:   ctaIndex,
		ClickedAt:  clock.Now(),
	}
	return r.DB.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(click).Error
}

// CTAClickCounts aggregates clicks per banner and CTA index in SQL, consistent
// with how dashboard stats are computed elsewhere.
func (r *AdBannerRepository) CTAClickCounts(ctx context.Context, bannerIDs []uint) (map[uint]map[int]int64, error) {
	counts := make(map[uint]map[int]int64)
	if len(bannerIDs) == 0 {
		return counts, nil
	}

	var rows []struct {
		BannerID uint  `gorm:"column:banner_id"`
		CTAIndex int   `gorm:"column:cta_index"`
		Cnt      int64 `gorm:"column:cnt"`
	}

	err := r.DB.WithContext(ctx).
		Model(&domain.AdBannerCTAClick{}).
		Select("banner_id, cta_index, COUNT(*) AS cnt").
		Where("banner_id IN ?", bannerIDs).
		Group("banner_id, cta_index").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		if counts[row.BannerID] == nil {
			counts[row.BannerID] = make(map[int]int64)
		}
		counts[row.BannerID][row.CTAIndex] = row.Cnt
	}
	return counts, nil
}
