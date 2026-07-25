package passwordreset

import (
	"context"
	"strings"
	"testing"

	"api-server/internal/infra/email"
)

func TestBuildResetEmailMessage_DeliveredBannerStaysInsideContentTable(t *testing.T) {
	msg, err := BuildResetEmailMessage(
		"test-token",
		"user@example.com",
		"Test User",
		"no-reply@example.com",
		"https://test.example/reset-password",
	)
	if err != nil {
		t.Fatalf("BuildResetEmailMessage returned error: %v", err)
	}

	provider := email.NewSandboxProvider(nil)
	if _, err := provider.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	delivered := provider.LastEmail()
	if delivered == nil || delivered.Message == nil {
		t.Fatal("expected the sandbox provider to capture the delivered message")
	}

	htmlBody := delivered.Message.HTMLBody
	tableIdx := strings.Index(htmlBody, `width="520"`)
	bannerIdx := strings.Index(htmlBody, `src="https://tingting.vip/email-banner.jpg?v=20260709"`)
	headingIdx := strings.Index(htmlBody, "Đặt lại mật khẩu TingTing")
	if tableIdx < 0 || bannerIdx < 0 || headingIdx < 0 {
		t.Fatalf("expected content table, banner, and reset heading in delivered HTML")
	}
	if tableIdx >= bannerIdx || bannerIdx >= headingIdx {
		t.Fatalf("expected the delivered banner to stay in the reset email content table")
	}
	if strings.Contains(htmlBody, `<div style="max-width:640px;margin:0 auto 16px;">`) {
		t.Fatal("expected the provider not to detach the banner above the content table")
	}
}
