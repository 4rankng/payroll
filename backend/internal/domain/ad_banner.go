package domain

import (
	"context"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"
)

// AdBannerCTAType determines how the employee portal renders and activates a
// call-to-action button: phone opens the native dialer via tel:, url opens a
// new tab, zalo hands the tap to the Zalo app through its zalo.me universal
// link (same-tab navigation on mobile, new tab on desktop). Nothing else is
// accepted — banners are typed content, never HTML.
type AdBannerCTAType string

const (
	AdBannerCTATypePhone AdBannerCTAType = "phone"
	AdBannerCTATypeURL   AdBannerCTAType = "url"
	AdBannerCTATypeZalo  AdBannerCTAType = "zalo"
)

// AdBannerMaxLifetime caps the starts_at→ends_at window so a typo'd year
// cannot recreate the forever-campaign case by hand.
const AdBannerMaxLifetime = 180 * 24 * time.Hour

const (
	adBannerMaxTitleLen    = 255
	adBannerMaxBulletLen   = 200
	adBannerMaxBullets     = 6
	adBannerMaxCTAs        = 3
	adBannerMaxCTALabelLen = 40
	adBannerMaxFooterLen   = 255
)

// AdBannerCTA is one call-to-action button on a banner.
type AdBannerCTA struct {
	Label string          `json:"label"`
	Type  AdBannerCTAType `json:"type"`
	Value string          `json:"value"`
}

// AdBanner is a project-targeted advertising campaign rendered in the employee
// portal. Content is typed fields (title/body/bullets/CTAs) — admins never
// author markup on a surface every worker sees.
type AdBanner struct {
	ID               uint           `json:"id" gorm:"primarykey;type:bigint unsigned"`
	Title            string         `json:"title" gorm:"type:varchar(255);not null"`
	Body             string         `json:"body" gorm:"type:text"`
	Bullets          []string       `json:"bullets" gorm:"type:json;serializer:json"`
	// column is pinned: GORM's naming strategy would otherwise mangle the
	// acronym into "ct_as" and every INSERT/UPDATE would fail.
	CTAs             []AdBannerCTA  `json:"ctas" gorm:"type:json;serializer:json;column:ctas"`
	Footer           string         `json:"footer" gorm:"type:varchar(255)"`
	TargetProjectIDs []uint         `json:"target_project_ids" gorm:"type:json;serializer:json;comment:'JSON uint array; NULL or empty = every project'"`
	Priority         int            `json:"priority" gorm:"not null;default:0"`
	StartsAt         time.Time      `json:"starts_at" gorm:"type:datetime(3);not null"`
	EndsAt           time.Time      `json:"ends_at" gorm:"type:datetime(3);not null;comment:'Mandatory campaign end; bounded lifetime'"`
	IsActive         bool           `json:"is_active" gorm:"type:tinyint(1);not null;default:true"`
	CreatedBy        uint           `json:"created_by" gorm:"not null;type:bigint unsigned"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AdBanner) TableName() string { return "ad_banners" }

// IsLiveAt reports whether the campaign is displayable at t. The window is
// half-open: live from starts_at (inclusive) until ends_at (exclusive), so a
// campaign stops resolving the instant its ends_at timestamp is reached with
// no off-by-one on the final day.
func (b *AdBanner) IsLiveAt(t time.Time) bool {
	return b.IsActive && !t.Before(b.StartsAt) && t.Before(b.EndsAt)
}

// TargetsProject reports whether the campaign addresses projectID. An empty
// target list is a broadcast: it targets every project.
func (b *AdBanner) TargetsProject(projectID uint) bool {
	if len(b.TargetProjectIDs) == 0 {
		return true
	}
	return slices.Contains(b.TargetProjectIDs, projectID)
}

// TargetsAnyProject reports whether any of the given project ids is addressed.
// An empty target list broadcasts to every project.
func (b *AdBanner) TargetsAnyProject(projectIDs []uint) bool {
	if len(b.TargetProjectIDs) == 0 {
		return true
	}
	for _, id := range projectIDs {
		if slices.Contains(b.TargetProjectIDs, id) {
			return true
		}
	}
	return false
}

// isValidCTAPhone accepts Vietnamese-shaped numbers with an optional leading
// "+": digits only, 6–15 of them.
func isValidCTAPhone(value string) bool {
	if value == "" {
		return false
	}
	digits := strings.TrimPrefix(value, "+")
	if len(digits) < 6 || len(digits) > 15 || len(digits) != len(value) && !strings.HasPrefix(value, "+") {
		return false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(digits) == len(value) || len(digits)+1 == len(value)
}

// Validate enforces the campaign content and window rules. Mirrors the spec:
// mandatory bounded lifetime, typed CTA shapes (https-only URLs, digit-only
// phone numbers), legible mobile content sizes.
func (b *AdBanner) Validate() error {
	b.Title = strings.TrimSpace(b.Title)
	if b.Title == "" {
		return NewValidationError("Tiêu đề quảng cáo là bắt buộc")
	}
	if len(b.Title) > adBannerMaxTitleLen {
		return NewValidationError("Tiêu đề quảng cáo tối đa 255 ký tự")
	}

	b.Footer = strings.TrimSpace(b.Footer)
	if len(b.Footer) > adBannerMaxFooterLen {
		return NewValidationError("Câu chào kết tối đa 255 ký tự")
	}

	if len(b.Bullets) > adBannerMaxBullets {
		return NewValidationError("Tối đa 6 gạch đầu dòng")
	}
	for _, bullet := range b.Bullets {
		bullet = strings.TrimSpace(bullet)
		if bullet == "" {
			return NewValidationError("Gạch đầu dòng không được để trống")
		}
		if len(bullet) > adBannerMaxBulletLen {
			return NewValidationError("Mỗi gạch đầu dòng tối đa 200 ký tự")
		}
	}

	if len(b.CTAs) == 0 {
		return NewValidationError("Quảng cáo cần ít nhất một nút hành động")
	}
	if len(b.CTAs) > adBannerMaxCTAs {
		return NewValidationError("Tối đa 3 nút hành động")
	}
	for _, cta := range b.CTAs {
		label := strings.TrimSpace(cta.Label)
		if label == "" {
			return NewValidationError("Nhãn nút hành động là bắt buộc")
		}
		if len(label) > adBannerMaxCTALabelLen {
			return NewValidationError("Nhãn nút hành động tối đa 40 ký tự")
		}
		switch cta.Type {
		case AdBannerCTATypePhone:
			if !isValidCTAPhone(strings.TrimSpace(cta.Value)) {
				return NewValidationError("Số điện thoại không hợp lệ")
			}
		case AdBannerCTATypeURL, AdBannerCTATypeZalo:
			if !strings.HasPrefix(strings.TrimSpace(cta.Value), "https://") {
				return NewValidationError("Đường dẫn phải bắt đầu bằng https://")
			}
		default:
			return NewValidationError("Loại nút hành động phải là 'phone', 'url' hoặc 'zalo'")
		}
	}

	if b.StartsAt.IsZero() || b.EndsAt.IsZero() {
		return NewValidationError("Thời gian bắt đầu và kết thúc là bắt buộc")
	}
	if !b.EndsAt.After(b.StartsAt) {
		return NewValidationError("Thời gian kết thúc phải sau thời gian bắt đầu")
	}
	if b.EndsAt.Sub(b.StartsAt) > AdBannerMaxLifetime {
		return NewValidationError("Thời gian hiển thị tối đa 180 ngày")
	}

	seen := make(map[uint]struct{}, len(b.TargetProjectIDs))
	for _, id := range b.TargetProjectIDs {
		if id == 0 {
			return NewValidationError("Dự án mục tiêu không hợp lệ")
		}
		if _, dup := seen[id]; dup {
			return NewValidationError("Dự án mục tiêu không được trùng lặp")
		}
		seen[id] = struct{}{}
	}

	return nil
}

// AdBannerCTAClick is an append-only CTA tap record. The (banner, employee,
// cta_index) unique key in the schema makes repeated taps idempotent.
type AdBannerCTAClick struct {
	ID         uint      `json:"id" gorm:"primarykey;type:bigint unsigned"`
	BannerID   uint      `json:"banner_id" gorm:"not null;type:bigint unsigned"`
	EmployeeID uint      `json:"employee_id" gorm:"not null;type:bigint unsigned"`
	CTAIndex   int       `json:"cta_index" gorm:"not null;type:tinyint unsigned;column:cta_index"`
	ClickedAt  time.Time `json:"clicked_at" gorm:"type:datetime(3);not null"`
}

func (AdBannerCTAClick) TableName() string { return "ad_banner_cta_clicks" }

// AdBannerFilters controls the admin list query. Expired campaigns stay
// visible (greyed) because their click counts are the campaign's record.
type AdBannerFilters struct {
	IncludeInactive bool
	Limit           int
	Offset          int
}

// AdBannerRepository defines persistence for ad banner campaigns and their
// append-only click ledger.
type AdBannerRepository interface {
	Create(ctx context.Context, banner *AdBanner) error
	GetByID(ctx context.Context, id uint) (*AdBanner, error)
	Update(ctx context.Context, banner *AdBanner) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filters AdBannerFilters) ([]*AdBanner, error)
	// ListLiveAt returns campaigns live at t ordered by priority DESC then
	// created_at DESC — the resolution order used for the single ad slot.
	ListLiveAt(ctx context.Context, t time.Time) ([]*AdBanner, error)
	// RecordCTAClick appends a tap; the schema's unique key makes it
	// idempotent per (banner, employee, cta_index).
	RecordCTAClick(ctx context.Context, bannerID, employeeID uint, ctaIndex int) error
	// CTAClickCounts aggregates the click ledger per banner and CTA index.
	CTAClickCounts(ctx context.Context, bannerIDs []uint) (map[uint]map[int]int64, error)
}
