package dto

import "time"

// AdBannerCTADTO is the wire shape of one call-to-action button.
type AdBannerCTADTO struct {
	Label string `json:"label"`
	Type  string `json:"type"` // "phone" | "url"
	Value string `json:"value"`
}

// AdBannerResponse is the wire shape shared by the admin list and the
// employee resolve endpoint. UpdatedAt doubles as the campaign version the
// portal keys its dismissal state on.
type AdBannerResponse struct {
	ID               uint             `json:"id"`
	Title            string           `json:"title"`
	Body             string           `json:"body"`
	Bullets          []string         `json:"bullets"`
	CTAs             []AdBannerCTADTO `json:"ctas"`
	Footer           string           `json:"footer"`
	TargetProjectIDs []uint           `json:"targetProjectIds"`
	Priority         int              `json:"priority"`
	StartsAt         time.Time        `json:"startsAt"`
	EndsAt           time.Time        `json:"endsAt"`
	IsActive         bool             `json:"isActive"`
	CreatedAt        time.Time        `json:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt"`
	// ClickCounts maps CTA index ("0", "1", "2") to tap count. JSON object
	// keys are strings, so the int index is stringified. Omitted for the
	// employee resolve response.
	ClickCounts map[string]int64 `json:"clickCounts,omitempty"`
}

// AdBannerListResponse is the response for GET /api/v1/ad-banners.
type AdBannerListResponse struct {
	Banners []AdBannerResponse `json:"banners"`
}

// CreateAdBannerRequest is the body for POST /api/v1/ad-banners. Window and
// content rules are enforced by domain validation; starts_at/ends_at accept
// RFC3339 timestamps.
type CreateAdBannerRequest struct {
	Title            string           `json:"title" binding:"required"`
	Body             string           `json:"body"`
	Bullets          []string         `json:"bullets"`
	CTAs             []AdBannerCTADTO `json:"ctas" binding:"required,min=1"`
	Footer           string           `json:"footer"`
	TargetProjectIDs []uint           `json:"targetProjectIds"`
	Priority         int              `json:"priority"`
	StartsAt         time.Time        `json:"startsAt" binding:"required"`
	EndsAt           time.Time        `json:"endsAt" binding:"required"`
	IsActive         *bool            `json:"isActive"`
}

// UpdateAdBannerRequest replaces the campaign content and window wholesale.
type UpdateAdBannerRequest struct {
	Title            string           `json:"title" binding:"required"`
	Body             string           `json:"body"`
	Bullets          []string         `json:"bullets"`
	CTAs             []AdBannerCTADTO `json:"ctas" binding:"required,min=1"`
	Footer           string           `json:"footer"`
	TargetProjectIDs []uint           `json:"targetProjectIds"`
	Priority         int              `json:"priority"`
	StartsAt         time.Time        `json:"startsAt" binding:"required"`
	EndsAt           time.Time        `json:"endsAt" binding:"required"`
	IsActive         *bool            `json:"isActive"`
}

// AdBannerClickRequest is the body for POST /api/v1/me/ad-banner/:id/click.
type AdBannerClickRequest struct {
	CTAIndex int `json:"cta_index" binding:"min=0,max=2"`
}
