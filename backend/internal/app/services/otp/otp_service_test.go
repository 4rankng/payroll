package otp

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"api-server/internal/config"
	"api-server/internal/domain"
	"api-server/internal/infra/cache"
	"api-server/internal/pkg/clock"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type recordingEmailSender struct {
	sent chan *domain.EmailMessage
}

func (s *recordingEmailSender) Send(_ context.Context, msg *domain.EmailMessage) (*domain.EmailDeliveryResult, error) {
	s.sent <- msg
	return &domain.EmailDeliveryResult{MessageID: "test-message", Provider: "test"}, nil
}

func TestStartLoginReusesPendingChallengeWithoutSendingAgain(t *testing.T) {
	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer redisServer.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	defer redisClient.Close()

	ttl := 5 * time.Minute
	sender := &recordingEmailSender{sent: make(chan *domain.EmailMessage, 2)}
	service := NewOTPService(
		cache.NewOTPPendingStore(redisClient, ttl),
		nil,
		sender,
		"noreply@example.com",
		config.OTPConfig{CodeTTL: ttl, ResendCooldown: 30 * time.Second},
		clock.NewFake(time.Date(2026, 7, 17, 20, 0, 0, 0, clock.DefaultLocation)),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	email := "owner@example.com"
	user := &domain.User{ID: 42, Fullname: "Owner", Email: &email}
	ctx := context.Background()

	firstSessionID, err := service.StartLogin(ctx, user, "203.0.113.1", "test-agent")
	if err != nil {
		t.Fatalf("first StartLogin: %v", err)
	}
	secondSessionID, err := service.StartLogin(ctx, user, "203.0.113.1", "test-agent")
	if err != nil {
		t.Fatalf("second StartLogin: %v", err)
	}
	if secondSessionID != firstSessionID {
		t.Fatalf("second StartLogin session = %q, want reused %q", secondSessionID, firstSessionID)
	}

	select {
	case <-sender.sent:
	case <-time.After(time.Second):
		t.Fatal("initial OTP email was not sent")
	}
	select {
	case <-sender.sent:
		t.Fatal("duplicate StartLogin sent a second OTP email")
	case <-time.After(100 * time.Millisecond):
	}
}
