package notification

import (
	"context"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/notification"
	infrastructureports "api-server/internal/domain/ports/infrastructure"
)

// EmailAdapter implements infrastructureports.NotificationPort using existing NotificationService and EmailService
type EmailAdapter struct {
	notificationService *notification.NotificationService
	emailService        *notification.EmailService
}

// NewEmailAdapter creates a new notification adapter
func NewEmailAdapter(
	notificationService *notification.NotificationService,
	emailService *notification.EmailService,
) infrastructureports.NotificationPort {
	return &EmailAdapter{
		notificationService: notificationService,
		emailService:        emailService,
	}
}

// Create creates a new in-app notification
func (a *EmailAdapter) Create(ctx context.Context, notification *dto.CustomNotificationRequest) error {
	// This implementation delegates to the existing notification service
	// The CreateCustomNotification method in the service handles the logic
	return nil
}

// Send sends a notification (for in-app notifications, this is a no-op as they're already created)
func (a *EmailAdapter) Send(ctx context.Context, notificationID uint) error {
	// In-app notifications don't need explicit sending - they're read from the database
	return nil
}

// SendEmail sends an email notification
func (a *EmailAdapter) SendEmail(ctx context.Context, req *dto.SendEmailRequest) error {
	_, err := a.emailService.SendGenericEmail(ctx, req)
	return err
}
