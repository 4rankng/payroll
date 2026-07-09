package otp

import (
	"strings"
	"testing"
)

func TestBuildOTPEmailMessage_BannerUsesFullContentWidth(t *testing.T) {
	msg, err := BuildOTPEmailMessage(
		"057210",
		"user@example.com",
		"Test User",
		"no-reply@example.com",
	)
	if err != nil {
		t.Fatalf("BuildOTPEmailMessage returned error: %v", err)
	}

	if !strings.Contains(msg.HTMLBody, `src="https://tingting.vip/email-banner.jpg?v=20260709"`) {
		t.Fatalf("HTMLBody missing brand banner URL")
	}
	if !strings.Contains(msg.HTMLBody, `width="520"`) {
		t.Fatalf("HTMLBody banner width should match the OTP email content width")
	}
	if !strings.Contains(msg.HTMLBody, `width:100%`) {
		t.Fatalf("HTMLBody banner should fill the available email content width")
	}
	if strings.Contains(msg.HTMLBody, `width="220"`) {
		t.Fatalf("HTMLBody still contains the old narrow banner width")
	}
}
