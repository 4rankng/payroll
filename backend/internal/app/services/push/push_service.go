package push

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"api-server/internal/config"
	"api-server/internal/domain"

	"github.com/SherClockHolmes/webpush-go"
)

// PushService handles web push notifications
type PushService struct {
	subRepo domain.PushSubscriptionRepository
	cfg     config.NotificationConfig
	logger  *slog.Logger
}

// NewPushService creates a new push notification service
func NewPushService(
	subRepo domain.PushSubscriptionRepository,
	cfg config.NotificationConfig,
	logger *slog.Logger,
) *PushService {
	return &PushService{
		subRepo: subRepo,
		cfg:     cfg,
		logger:  logger,
	}
}

// Subscribe stores a push subscription for a user
func (s *PushService) Subscribe(ctx context.Context, userID uint, endpoint, p256dh, auth, deviceType string) error {
	sub := &domain.PushSubscription{
		UserID:     userID,
		Endpoint:   endpoint,
		P256DH:     p256dh,
		Auth:       auth,
		DeviceType: deviceType,
	}
	if err := s.subRepo.Create(ctx, sub); err != nil {
		return fmt.Errorf("failed to save push subscription: %w", err)
	}
	s.logger.Info("Push subscription saved", "user_id", userID, "device", deviceType)
	return nil
}

// Unsubscribe removes a push subscription
func (s *PushService) Unsubscribe(ctx context.Context, userID uint, endpoint string) error {
	if err := s.subRepo.DeleteByEndpoint(ctx, userID, endpoint); err != nil {
		return fmt.Errorf("failed to remove push subscription: %w", err)
	}
	s.logger.Info("Push subscription removed", "user_id", userID)
	return nil
}

// SendToUser sends a push notification to all devices of a user
func (s *PushService) SendToUser(ctx context.Context, userID uint, title, body string) error {
	subs, err := s.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get subscriptions: %w", err)
	}

	if len(subs) == 0 {
		return nil // User has no push subscriptions
	}

	payload, _ := json.Marshal(map[string]string{
		"title": title,
		"body":  body,
		"icon":  "/favicon.png",
		"tag":   "tingting-notification",
	})

	vapidPrivateKey := s.cfg.VAPIDPrivateKey
	vapidPublicKey := s.cfg.VAPIDPublicKey
	vapidSubject := s.cfg.VAPIDSubject

	for _, sub := range subs {
		webSub := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256DH,
				Auth:   sub.Auth,
			},
		}

		resp, err := webpush.SendNotificationWithContext(ctx, payload, webSub, &webpush.Options{
			VAPIDPublicKey:  vapidPublicKey,
			VAPIDPrivateKey: vapidPrivateKey,
			Subscriber:      vapidSubject,
			TTL:             86400, // 24 hours — survive device sleep / network gaps
			Urgency:         webpush.UrgencyHigh,
		})

		if err != nil {
			s.logger.Error("Failed to send push notification",
				"user_id", userID,
				"endpoint", sub.Endpoint,
				"error", err)
			continue
		}
		_ = resp.Body.Close()

		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			s.logger.Info("Push notification sent",
				"user_id", userID,
				"device", sub.DeviceType,
				"status", resp.StatusCode)
		case resp.StatusCode == 410 || resp.StatusCode == 404:
			s.logger.Info("Push subscription expired, removing",
				"user_id", userID,
				"endpoint", sub.Endpoint,
				"status", resp.StatusCode)
			_ = s.subRepo.DeleteByEndpoint(ctx, userID, sub.Endpoint)
		default:
			// 401 (VAPID mismatch), 400 (bad payload), 429 (rate-limited), etc.
			s.logger.Warn("Push notification rejected by push service",
				"user_id", userID,
				"device", sub.DeviceType,
				"endpoint", sub.Endpoint,
				"status", resp.StatusCode)
			// Remove invalid subscriptions so they get re-created on next visit
			_ = s.subRepo.DeleteByEndpoint(ctx, userID, sub.Endpoint)
		}
	}

	return nil
}

// GetVAPIDPublicKey returns the public key for frontend push subscription
func (s *PushService) GetVAPIDPublicKey() string {
	return s.cfg.VAPIDPublicKey
}
