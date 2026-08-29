package flex_pay

import (
	"strings"
	"testing"

	"api-server/internal/app/services/config"
)

func TestBuildSaoKeEmailBodiesUsesPublicBannerURL(t *testing.T) {
	htmlBody, textBody := BuildSaoKeEmailBodies("2026-06", "31/07/2026", "118.110.000 d", config.DefaultTransferBankInfo())

	if !strings.Contains(htmlBody, `src="https://tingting.vip/email-banner.jpg?v=20260709"`) {
		t.Fatalf("expected HTML body to include public email banner URL")
	}
	if strings.Contains(htmlBody, "cid:") {
		t.Fatalf("expected HTML body to use public URL instead of inline CID image")
	}
	if strings.Contains(textBody, "email-banner.jpg") {
		t.Fatalf("expected plain text body not to include image URL")
	}
	if !strings.Contains(htmlBody, `<table role="presentation" width="640"`) {
		t.Fatalf("expected HTML body to use a centered email layout table")
	}
	if !strings.Contains(htmlBody, "Tổng tiền thanh toán") {
		t.Fatalf("expected HTML body to include a payment summary section")
	}
	if !strings.Contains(htmlBody, "border-radius:16px") {
		t.Fatalf("expected HTML body to use polished card styling")
	}
	if !strings.Contains(htmlBody, `font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Arial,sans-serif`) {
		t.Fatalf("expected HTML body to use a modern email-safe font stack")
	}
	if !strings.Contains(htmlBody, `<p style="margin:20px 0 0;color:#111827;font-size:15px;line-height:24px;font-weight:700;">TING TING SOFT</p>`) {
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
	if !strings.Contains(htmlBody, "Trân trọng cảm ơn Quý Công Ty đã hợp tác và tin tưởng sử dụng dịch vụ.") {
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

func TestBuildSaoKeEmailBodiesUsesMBTransferDetails(t *testing.T) {
	htmlBody, textBody := BuildSaoKeEmailBodies("2026-06", "31/07/2026", "118.110.000 đ", config.DefaultTransferBankInfo())

	for _, body := range []string{htmlBody, textBody} {
		if !strings.Contains(body, "271866699") {
			t.Fatalf("expected transfer details to include MB account number")
		}
		if !strings.Contains(body, "Ngân hàng Quân đội (MB)") {
			t.Fatalf("expected transfer details to include MB bank name")
		}
		if strings.Contains(body, "283866888") || strings.Contains(body, "TECHCOMBANK") {
			t.Fatalf("expected legacy Techcombank transfer details to be absent")
		}
	}
}

func TestBuildSaoKeEmailBodiesHidesBankWhenHidden(t *testing.T) {
	bank := config.DefaultTransferBankInfo()
	bank.Hidden = true

	htmlBody, textBody := BuildSaoKeEmailBodies("2026-06", "31/07/2026", "118.110.000 đ", bank)

	for _, body := range []string{htmlBody, textBody} {
		if strings.Contains(body, "Thông tin chuyển khoản") {
			t.Fatalf("expected transfer instructions heading to be absent when hidden")
		}
		if strings.Contains(body, "271866699") || strings.Contains(body, config.DefaultTransferBankHolder) {
			t.Fatalf("expected beneficiary account details to be absent when hidden")
		}
	}
	if !strings.Contains(htmlBody, "Đây là sao kê dịch vụ") {
		t.Fatalf("expected informational statement note in HTML body when hidden")
	}
	// The hidden branch contributes an empty bank block; the template must not
	// leave a double blank line where the transfer instructions used to be.
	if strings.Contains(textBody, "\n\n\n") {
		t.Fatalf("expected no stacked blank lines in text body when hidden")
	}
}
