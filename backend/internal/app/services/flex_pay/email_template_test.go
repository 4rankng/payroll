package flex_pay

import (
	"strings"
	"testing"
)

func TestBuildSaoKeEmailBodiesUsesPublicBannerURL(t *testing.T) {
	htmlBody, textBody := BuildSaoKeEmailBodies("2026-06", "31/07/2026", "118.110.000 d")

	if !strings.Contains(htmlBody, `src="https://tingting.vip/email-banner.jpg"`) {
		t.Fatalf("expected HTML body to include public email banner URL")
	}
	if strings.Contains(htmlBody, "cid:") {
		t.Fatalf("expected HTML body to use public URL instead of inline CID image")
	}
	if strings.Contains(textBody, "email-banner.jpg") {
		t.Fatalf("expected plain text body not to include image URL")
	}
}
