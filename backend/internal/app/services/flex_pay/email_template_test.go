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
	if !strings.Contains(htmlBody, `style="text-align:left;"`) {
		t.Fatalf("expected HTML banner wrapper to align left")
	}
	if !strings.Contains(htmlBody, `<p>Trân trọng,<br>TING TING SOFT</p>`) {
		t.Fatalf("expected HTML signature to use TING TING SOFT only")
	}
	if strings.Contains(htmlBody, "TING TING SOFT - Dịch vụ Nhận lương sớm 24/7") {
		t.Fatalf("expected HTML signature not to include service suffix")
	}
	if !strings.Contains(textBody, "Trân trọng,\nTING TING SOFT") {
		t.Fatalf("expected text signature to use TING TING SOFT only")
	}
	if strings.Contains(textBody, "TING TING SOFT - Dịch vụ Nhận lương sớm 24/7") {
		t.Fatalf("expected text signature not to include service suffix")
	}
	if !strings.Contains(htmlBody, "<p>Trân trọng cảm ơn Quý Công Ty đã hợp tác và tin tưởng sử dụng dịch vụ.</p>") {
		t.Fatalf("expected HTML thank-you sentence not to include company suffix")
	}
	if strings.Contains(htmlBody, "tin tưởng sử dụng dịch vụ của TING TING SOFT") {
		t.Fatalf("expected HTML thank-you sentence not to include TING TING SOFT")
	}
	if !strings.Contains(textBody, "Trân trọng cảm ơn Quý Công Ty đã hợp tác và tin tưởng sử dụng dịch vụ.") {
		t.Fatalf("expected text thank-you sentence not to include company suffix")
	}
	if strings.Contains(textBody, "tin tưởng sử dụng dịch vụ của TING TING SOFT") {
		t.Fatalf("expected text thank-you sentence not to include TING TING SOFT")
	}
}
