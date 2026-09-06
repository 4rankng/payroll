package domain

import (
	"testing"
	"time"
)

func adBannerTestWindow() (time.Time, time.Time) {
	starts := time.Date(2026, 9, 6, 0, 0, 0, 0, time.Local)
	return starts, starts.Add(30 * 24 * time.Hour)
}

// validAdBanner is the LG Display campaign shape: headline, lead paragraph,
// three bullets, phone + Zalo CTA, 30-day window.
func validAdBanner() *AdBanner {
	starts, ends := adBannerTestWindow()
	return &AdBanner{
		Title:   "TING TING SOFTWARE SOLUTIONS xin thông báo",
		Body:    "Công nhân dự án LG Display có thể CHẤM CÔNG TỰ ĐỘNG và ỨNG LƯƠNG NGAY trên điện thoại.",
		Bullets: []string{
			"Chấm công tự động, chính xác từng ca làm",
			"Ứng lương theo công đã làm, tối đa 70%, phí chỉ từ 1.3%",
			"Nhận tiền nhanh chóng, chủ động chi tiêu",
		},
		CTAs: []AdBannerCTA{
			{Label: "Gọi hotline", Type: AdBannerCTATypePhone, Value: "0914827988"},
			{Label: "Zalo", Type: AdBannerCTATypeZalo, Value: "https://zalo.me/g/ekvooqdb9hjcl3qmogof"},
		},
		Footer:   "Ting Ting Software Solutions — Đồng hành cùng người lao động.",
		StartsAt: starts,
		EndsAt:   ends,
		IsActive: true,
	}
}

func TestAdBannerValidateAcceptsCampaignShape(t *testing.T) {
	if err := validAdBanner().Validate(); err != nil {
		t.Fatalf("expected LGD-shaped campaign to validate, got %v", err)
	}
}

func TestAdBannerValidateRules(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(b *AdBanner)
	}{
		{"empty title", func(b *AdBanner) { b.Title = "   " }},
		{"title over 255", func(b *AdBanner) { b.Title = string(make([]byte, 256)) }},
		{"footer over 255", func(b *AdBanner) { b.Footer = string(make([]byte, 256)) }},
		{"seven bullets", func(b *AdBanner) {
			b.Bullets = []string{"1", "2", "3", "4", "5", "6", "7"}
		}},
		{"blank bullet", func(b *AdBanner) { b.Bullets = []string{"  "} }},
		{"bullet over 200 chars", func(b *AdBanner) { b.Bullets = []string{string(make([]byte, 201))} }},
		{"no CTAs", func(b *AdBanner) { b.CTAs = nil }},
		{"four CTAs", func(b *AdBanner) {
			b.CTAs = []AdBannerCTA{
				{Label: "A", Type: AdBannerCTATypeURL, Value: "https://a"},
				{Label: "B", Type: AdBannerCTATypeURL, Value: "https://b"},
				{Label: "C", Type: AdBannerCTATypeURL, Value: "https://c"},
				{Label: "D", Type: AdBannerCTATypeURL, Value: "https://d"},
			}
		}},
		{"empty CTA label", func(b *AdBanner) {
			b.CTAs = []AdBannerCTA{{Label: " ", Type: AdBannerCTATypeURL, Value: "https://a"}}
		}},
		{"CTA label over 40", func(b *AdBanner) {
			b.CTAs = []AdBannerCTA{{Label: string(make([]byte, 41)), Type: AdBannerCTATypeURL, Value: "https://a"}}
		}},
		{"unknown CTA type", func(b *AdBanner) {
			b.CTAs = []AdBannerCTA{{Label: "Zalo", Type: "sms", Value: "https://zalo.me"}}
		}},
		{"http zalo rejected", func(b *AdBanner) {
			b.CTAs = []AdBannerCTA{{Label: "Zalo", Type: AdBannerCTATypeZalo, Value: "http://zalo.me"}}
		}},
		{"phone with letters", func(b *AdBanner) {
			b.CTAs = []AdBannerCTA{{Label: "Gọi", Type: AdBannerCTATypePhone, Value: "0914abc988"}}
		}},
		{"phone too short", func(b *AdBanner) {
			b.CTAs = []AdBannerCTA{{Label: "Gọi", Type: AdBannerCTATypePhone, Value: "123"}}
		}},
		{"http url rejected", func(b *AdBanner) {
			b.CTAs = []AdBannerCTA{{Label: "Zalo", Type: AdBannerCTATypeURL, Value: "http://zalo.me"}}
		}},
		{"javascript url rejected", func(b *AdBanner) {
			b.CTAs = []AdBannerCTA{{Label: "Zalo", Type: AdBannerCTATypeURL, Value: "javascript:alert(1)"}}
		}},
		{"ends before starts", func(b *AdBanner) { b.StartsAt, b.EndsAt = b.EndsAt, b.StartsAt }},
		{"ends equals starts", func(b *AdBanner) { b.EndsAt = b.StartsAt }},
		{"zero window dates", func(b *AdBanner) { b.StartsAt, b.EndsAt = time.Time{}, time.Time{} }},
		{"window over 180 days", func(b *AdBanner) { b.EndsAt = b.StartsAt.Add(181 * 24 * time.Hour) }},
		{"exactly 180 days allowed", func(b *AdBanner) { b.EndsAt = b.StartsAt.Add(180 * 24 * time.Hour) }},
		{"zero target project id", func(b *AdBanner) { b.TargetProjectIDs = []uint{0} }},
		{"duplicate target project ids", func(b *AdBanner) { b.TargetProjectIDs = []uint{7, 7} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := validAdBanner()
			tc.mutate(b)
			if tc.name == "exactly 180 days allowed" {
				if err := b.Validate(); err != nil {
					t.Fatalf("expected 180-day window to validate, got %v", err)
				}
				return
			}
			if err := b.Validate(); err == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
		})
	}
}

func TestAdBannerValidateAcceptsPlusPrefixedPhone(t *testing.T) {
	b := validAdBanner()
	b.CTAs = []AdBannerCTA{{Label: "Gọi hotline", Type: AdBannerCTATypePhone, Value: "+84914827988"}}
	if err := b.Validate(); err != nil {
		t.Fatalf("expected +84 phone to validate, got %v", err)
	}
}

func TestAdBannerIsLiveAtHalfOpenWindow(t *testing.T) {
	b := validAdBanner()

	if b.IsLiveAt(b.StartsAt.Add(-time.Second)) {
		t.Error("before starts_at must not be live")
	}
	if !b.IsLiveAt(b.StartsAt) {
		t.Error("at starts_at must be live (closed start)")
	}
	if !b.IsLiveAt(b.EndsAt.Add(-time.Second)) {
		t.Error("just before ends_at must be live")
	}
	if b.IsLiveAt(b.EndsAt) {
		t.Error("at ends_at must NOT be live (open end)")
	}

	b.IsActive = false
	if b.IsLiveAt(b.StartsAt.Add(time.Hour)) {
		t.Error("paused campaign must not be live")
	}
}

func TestAdBannerTargetsProject(t *testing.T) {
	broadcast := validAdBanner()
	if !broadcast.TargetsProject(999) {
		t.Error("empty target list broadcasts to every project")
	}

	targeted := validAdBanner()
	targeted.TargetProjectIDs = []uint{12}
	if !targeted.TargetsProject(12) {
		t.Error("targeted project must match")
	}
	if targeted.TargetsProject(13) {
		t.Error("untargeted project must not match")
	}
	if !targeted.TargetsAnyProject([]uint{13, 12}) {
		t.Error("TargetsAnyProject must match when one of the employee's projects is targeted")
	}
	if targeted.TargetsAnyProject([]uint{13}) {
		t.Error("TargetsAnyProject must not match disjoint sets")
	}
}
