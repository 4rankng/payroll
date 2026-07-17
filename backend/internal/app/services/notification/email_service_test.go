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

func TestEmailServiceResolveFromAddressUsesApprovedIdentities(t *testing.T) {
	service := &EmailService{}

	t.Run("uses default when omitted", func(t *testing.T) {
		address, err := service.resolveFromAddress("")
		if err != nil {
			t.Fatalf("resolve default sender: %v", err)
		}
		if address.Name != "Ting Ting" || address.Address != "noreply@tingting.vip" {
			t.Fatalf("unexpected default sender: %#v", address)
		}
	})

	t.Run("uses canonical approved marketing sender", func(t *testing.T) {
		address, err := service.resolveFromAddress("marketing@tingting.vip")
		if err != nil {
			t.Fatalf("resolve marketing sender: %v", err)
		}
		if address.Name != "Ting Ting Software Solution" || address.Address != "marketing@tingting.vip" {
			t.Fatalf("unexpected marketing sender: %#v", address)
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
		config.NotificationConfig{},
		provider,
		nil,
		nil,
		nil,
		nil,
		nil,
		slog.Default(),
	)

	_, err := service.SendGenericEmail(context.Background(), &dto.SendEmailRequest{
		From:       "marketing@tingting.vip",
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
	if captured.Message.From.Address != "marketing@tingting.vip" {
		t.Fatalf("unexpected sender: %#v", captured.Message.From)
	}
	if !strings.Contains(captured.Message.HTMLBody, "tingting.vip/email-banner.jpg") {
		t.Fatalf("expected TingTing banner in generic email: %q", captured.Message.HTMLBody)
	}
}

func TestEmailServiceAvailableSendersReturnsApprovedIdentities(t *testing.T) {
	service := &EmailService{}

	senders, err := service.AvailableSenders()
	if err != nil {
		t.Fatalf("list senders: %v", err)
	}
	if len(senders) != 2 {
		t.Fatalf("expected 2 senders, got %#v", senders)
	}
	if senders[0].Name != "Ting Ting" || senders[0].Address != "noreply@tingting.vip" ||
		senders[1].Name != "Ting Ting Software Solution" || senders[1].Address != "marketing@tingting.vip" {
		t.Fatalf("unexpected sender order: %#v", senders)
	}
}
