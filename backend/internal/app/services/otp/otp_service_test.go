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
	defer func() { _ = redisClient.Close() }()

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

type fakeOTPUserRepo struct {
	domain.UserRepository
	user *domain.User
}

func (r *fakeOTPUserRepo) GetByID(_ context.Context, id uint) (*domain.User, error) {
	if r.user != nil && r.user.ID == id {
		return r.user, nil
	}
	return nil, domain.NewNotFoundError("user not found")
}

func (r *fakeOTPUserRepo) UpdateOTPLockout(_ context.Context, _ uint, _ int, _ *time.Time) error {
	return nil
}

func TestStartLoginRefreshesChallengeAfterCooldown(t *testing.T) {
	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer redisServer.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	defer func() { _ = redisClient.Close() }()

	ttl := 5 * time.Minute
	sender := &recordingEmailSender{sent: make(chan *domain.EmailMessage, 4)}
	fakeClock := clock.NewFake(time.Date(2026, 7, 17, 20, 0, 0, 0, clock.DefaultLocation))
	service := NewOTPService(
		cache.NewOTPPendingStore(redisClient, ttl),
		nil,
		sender,
		"noreply@example.com",
		config.OTPConfig{CodeTTL: ttl, ResendCooldown: 30 * time.Second},
		fakeClock,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	email := "owner@example.com"
	user := &domain.User{ID: 42, Fullname: "Owner", Email: &email}
	ctx := context.Background()

	sessionID1, err := service.StartLogin(ctx, user, "203.0.113.1", "test-agent")
	if err != nil {
		t.Fatalf("first StartLogin: %v", err)
	}
	select {
	case <-sender.sent:
	case <-time.After(time.Second):
		t.Fatal("initial OTP email was not sent")
	}

	// Advance clock past 30s resend cooldown.
	fakeClock.Advance(35 * time.Second)

	sessionID2, err := service.StartLogin(ctx, user, "203.0.113.2", "test-agent")
	if err != nil {
		t.Fatalf("second StartLogin: %v", err)
	}
	if sessionID2 != sessionID1 {
		t.Fatalf("expected reused session ID %q, got %q", sessionID1, sessionID2)
	}

	select {
	case msg := <-sender.sent:
		if len(msg.To) == 0 || msg.To[0].Address != email {
			t.Fatalf("unexpected recipient: %v", msg.To)
		}
	case <-time.After(time.Second):
		t.Fatal("second OTP email after cooldown was not sent")
	}
}

func TestVerifyLoginAllowsClientIPChange(t *testing.T) {
	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer redisServer.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	defer func() { _ = redisClient.Close() }()

	ttl := 5 * time.Minute
	sender := &recordingEmailSender{sent: make(chan *domain.EmailMessage, 2)}
	email := "owner@example.com"
	user := &domain.User{ID: 42, Fullname: "Owner", Email: &email}
	userRepo := &fakeOTPUserRepo{user: user}

	fakeClock := clock.NewFake(time.Date(2026, 7, 17, 20, 0, 0, 0, clock.DefaultLocation))
	service := NewOTPService(
		cache.NewOTPPendingStore(redisClient, ttl),
		userRepo,
		sender,
		"noreply@example.com",
		config.OTPConfig{CodeTTL: ttl, ResendCooldown: 30 * time.Second},
		fakeClock,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	ctx := context.Background()
	sessionID, err := service.StartLogin(ctx, user, "104.28.122.109", "iPhone-Safari")
	if err != nil {
		t.Fatalf("StartLogin: %v", err)
	}

	var sentMsg *domain.EmailMessage
	select {
	case sentMsg = <-sender.sent:
	case <-time.After(time.Second):
		t.Fatal("OTP email was not sent")
	}

	// Extract code from text body: "Mã đăng nhập của bạn là: 123456"
	var code string
	for _, line := range sentMsg.TextBody {
		_ = line
	}
	idx := len("Mã đăng nhập của bạn là: ")
	if len(sentMsg.TextBody) >= idx+6 {
		code = sentMsg.TextBody[idx : idx+6]
	} else {
		t.Fatalf("cannot parse code from text body %q", sentMsg.TextBody)
	}

	// Verify with a different IP (e.g. iCloud Private Relay rotated egress IP)
	verifiedUser, err := service.VerifyLogin(ctx, sessionID, code, "104.28.122.111", "iPhone-Safari")
	if err != nil {
		t.Fatalf("VerifyLogin with different IP should succeed, got err: %v", err)
	}
	if verifiedUser.ID != user.ID {
		t.Fatalf("expected user ID %d, got %d", user.ID, verifiedUser.ID)
	}
}

func TestVerifyLoginRejectsUserAgentMismatch(t *testing.T) {
	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer redisServer.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	defer func() { _ = redisClient.Close() }()

	ttl := 5 * time.Minute
	sender := &recordingEmailSender{sent: make(chan *domain.EmailMessage, 2)}
	email := "owner@example.com"
	user := &domain.User{ID: 42, Fullname: "Owner", Email: &email}
	userRepo := &fakeOTPUserRepo{user: user}

	fakeClock := clock.NewFake(time.Date(2026, 7, 17, 20, 0, 0, 0, clock.DefaultLocation))
	service := NewOTPService(
		cache.NewOTPPendingStore(redisClient, ttl),
		userRepo,
		sender,
		"noreply@example.com",
		config.OTPConfig{CodeTTL: ttl, ResendCooldown: 30 * time.Second},
		fakeClock,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	ctx := context.Background()
	sessionID, err := service.StartLogin(ctx, user, "104.28.122.109", "iPhone-Safari")
	if err != nil {
		t.Fatalf("StartLogin: %v", err)
	}

	var sentMsg *domain.EmailMessage
	select {
	case sentMsg = <-sender.sent:
	case <-time.After(time.Second):
		t.Fatal("OTP email was not sent")
	}

	idx := len("Mã đăng nhập của bạn là: ")
	code := sentMsg.TextBody[idx : idx+6]

	// Verify with different User-Agent should fail
	_, err = service.VerifyLogin(ctx, sessionID, code, "104.28.122.109", "Different-Browser")
	if err != ErrSessionBindingMismatch {
		t.Fatalf("expected ErrSessionBindingMismatch, got: %v", err)
	}
}
