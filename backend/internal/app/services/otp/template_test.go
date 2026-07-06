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

	if !strings.Contains(msg.HTMLBody, `src="cid:brand-banner"`) {
		t.Fatalf("HTMLBody missing inline brand banner CID")
	}
	if !strings.Contains(msg.HTMLBody, `width="600"`) {
		t.Fatalf("HTMLBody banner width should match the 600px banner asset")
	}
	if !strings.Contains(msg.HTMLBody, `width:100%`) {
		t.Fatalf("HTMLBody banner should fill the available email content width")
	}
	if strings.Contains(msg.HTMLBody, `width="220"`) {
		t.Fatalf("HTMLBody still contains the old narrow banner width")
	}
}
