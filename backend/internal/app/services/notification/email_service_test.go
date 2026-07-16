package notification

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"api-server/internal/app/dto"
	"api-server/internal/config"
	"api-server/internal/domain"
	emailinfra "api-server/internal/infra/email"
)

func TestEmailServiceResolveFromAddressUsesConfiguredIdentities(t *testing.T) {
	service := &EmailService{cfg: config.NotificationConfig{
		FromName:          "TingTing",
		FromEmail:         "noreply@tingting.vip",
		AllowedFromEmails: []string{"TingTing Hỗ trợ <support@tingting.vip>"},
	}}

	t.Run("uses default when omitted", func(t *testing.T) {
		address, err := service.resolveFromAddress("")
		if err != nil {
			t.Fatalf("resolve default sender: %v", err)
		}
		if address.Name != "TingTing" || address.Address != "noreply@tingting.vip" {
			t.Fatalf("unexpected default sender: %#v", address)
		}
	})

	t.Run("uses canonical configured alternate", func(t *testing.T) {
		address, err := service.resolveFromAddress("support@tingting.vip")
		if err != nil {
			t.Fatalf("resolve configured sender: %v", err)
		}
		if address.Name != "TingTing Hỗ trợ" || address.Address != "support@tingting.vip" {
			t.Fatalf("unexpected configured sender: %#v", address)
		}
	})

	t.Run("rejects an unconfigured sender", func(t *testing.T) {
		_, err := service.resolveFromAddress("spoofed@example.com")
		if err == nil || !domain.IsValidationError(err) {
			t.Fatalf("expected validation error, got %v", err)
		}
	})
}

func TestEmailServiceSendGenericEmailUsesSelectedSenderAndBranding(t *testing.T) {
	provider := emailinfra.NewSandboxProvider(slog.Default())
	service := NewEmailService(
		config.NotificationConfig{
			FromName:          "TingTing",
			FromEmail:         "noreply@tingting.vip",
			AllowedFromEmails: []string{"TingTing Hỗ trợ <support@tingting.vip>"},
		},
		provider,
		nil,
		nil,
		nil,
		nil,
		nil,
		slog.Default(),
	)

	_, err := service.SendGenericEmail(context.Background(), &dto.SendEmailRequest{
		From:       "support@tingting.vip",
		Recipients: []string{"recipient@example.com"},
		Subject:    "Thông báo",
		HTMLBody:   "<p>Nội dung email</p>",
	})
	if err != nil {
		t.Fatalf("send generic email: %v", err)
	}

	captured := provider.LastEmail()
	if captured == nil || captured.Message == nil {
		t.Fatal("expected captured email")
	}
	if captured.Message.From.Address != "support@tingting.vip" {
		t.Fatalf("unexpected sender: %#v", captured.Message.From)
	}
	if !strings.Contains(captured.Message.HTMLBody, "tingting.vip/email-banner.jpg") {
		t.Fatalf("expected TingTing banner in generic email: %q", captured.Message.HTMLBody)
	}
}

func TestEmailServiceAvailableSendersIncludesDefaultOnlyOnce(t *testing.T) {
	service := &EmailService{cfg: config.NotificationConfig{
		FromName:          "TingTing",
		FromEmail:         "noreply@tingting.vip",
		AllowedFromEmails: []string{"TingTing <NOREPLY@tingting.vip>", "support@tingting.vip"},
	}}

	senders, err := service.AvailableSenders()
	if err != nil {
		t.Fatalf("list senders: %v", err)
	}
	if len(senders) != 2 {
		t.Fatalf("expected 2 senders, got %#v", senders)
	}
	if senders[0].Address != "noreply@tingting.vip" || senders[1].Address != "support@tingting.vip" {
		t.Fatalf("unexpected sender order: %#v", senders)
	}
}
