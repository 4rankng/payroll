package email

import (
	"context"
	"strings"
	"testing"

	"api-server/internal/domain"
)

func TestWithPublicEmailBannerAddsBannerToHTMLFragment(t *testing.T) {
	body := withPublicEmailBanner("<p>Xin chào</p>", "")

	if count := strings.Count(body, publicEmailBannerURL); count != 1 {
		t.Fatalf("expected exactly one public banner URL, got %d in %q", count, body)
	}
	if strings.Contains(body, "cid:") {
		t.Fatalf("expected public URL banner, not inline CID image")
	}
	if !strings.Contains(body, `<table role="presentation" width="640"`) {
		t.Fatalf("expected HTML fragments to be wrapped in the branded email shell")
	}
	if !strings.Contains(body, "border-radius:16px") {
		t.Fatalf("expected branded email shell to use polished card styling")
	}
}

func TestWithPublicEmailBannerDoesNotDuplicateExistingBanner(t *testing.T) {
	body := withPublicEmailBanner(publicEmailBannerHTML+"\n<p>Nội dung</p>", "")

	if count := strings.Count(body, publicEmailBannerURL); count != 1 {
		t.Fatalf("expected existing public banner URL to stay single, got %d", count)
	}
}

func TestWithPublicEmailBannerReplacesDuplicateOrNonImageReferences(t *testing.T) {
	body := withPublicEmailBanner(
		`<img src="https://tingting.vip/email-banner.jpg?v=old"><img src='https://tingting.vip/email-banner.jpg?v=older'><a href="https://tingting.vip/email-banner.jpg">Tải banner</a><!-- tingting.vip/email-banner.jpg -->`,
		"",
	)

	if count := strings.Count(body, publicEmailBannerURL); count != 1 {
		t.Fatalf("expected exactly one canonical banner image, got %d in %q", count, body)
	}
	if !strings.Contains(body, `href="https://tingting.vip/email-banner.jpg"`) {
		t.Fatalf("expected non-image banner reference to remain untouched")
	}
}

func TestWithPublicEmailBannerBuildsHTMLForTextOnlyEmail(t *testing.T) {
	body := withPublicEmailBanner("", "Dòng 1\nDòng <2>")

	if !strings.Contains(body, publicEmailBannerURL) {
		t.Fatalf("expected text-only email to receive public banner HTML")
	}
	if !strings.Contains(body, "Dòng 1<br>") {
		t.Fatalf("expected text newlines to be preserved in generated HTML")
	}
	if !strings.Contains(body, "Dòng &lt;2&gt;") {
		t.Fatalf("expected generated HTML to escape text body")
	}
	if !strings.Contains(body, "Thông báo") {
		t.Fatalf("expected text-only email to use the branded notification shell")
	}
}

func TestWithPublicEmailBannerInsertsAfterBodyTag(t *testing.T) {
	body := withPublicEmailBanner(`<!doctype html><html><body style="margin:0"><p>Nội dung</p></body></html>`, "")

	bodyTagIdx := strings.Index(body, `<body style="margin:0">`)
	bannerIdx := strings.Index(body, publicEmailBannerHTML)
	contentIdx := strings.Index(body, "<p>Nội dung</p>")
	if bodyTagIdx < 0 || bannerIdx < 0 || contentIdx < 0 {
		t.Fatalf("expected body tag, banner, and content in normalized HTML: %q", body)
	}
	if bodyTagIdx >= bannerIdx || bannerIdx >= contentIdx {
		t.Fatalf("expected banner after body tag and before body content")
	}
}

func TestSandboxProviderCapturesMessageWithPublicBanner(t *testing.T) {
	to, err := domain.ParseEmailAddress("recipient@example.com")
	if err != nil {
		t.Fatalf("parse recipient: %v", err)
	}
	from, err := domain.ParseEmailAddress("TingTing <noreply@tingting.vip>")
	if err != nil {
		t.Fatalf("parse sender: %v", err)
	}
	msg := &domain.EmailMessage{
		Kind:     domain.EmailKindGeneric,
		From:     from,
		To:       []domain.EmailAddress{to},
		Subject:  "Thông báo",
		TextBody: "Nội dung thông báo",
	}
	if err := msg.Validate(); err != nil {
		t.Fatalf("validate message: %v", err)
	}

	provider := NewSandboxProvider(nil)
	if _, err := provider.Send(context.Background(), msg); err != nil {
		t.Fatalf("send sandbox email: %v", err)
	}

	captured := provider.LastEmail()
	if captured == nil || captured.Message == nil {
		t.Fatalf("expected captured email")
	}
	if !strings.Contains(captured.Message.HTMLBody, publicEmailBannerURL) {
		t.Fatalf("expected captured email HTML to include public banner URL")
	}
	if strings.Contains(captured.Message.HTMLBody, "cid:") {
		t.Fatalf("expected captured email to avoid inline CID images")
	}
	if msg.HTMLBody != "" {
		t.Fatalf("expected sandbox provider not to mutate caller's original message")
	}
}
