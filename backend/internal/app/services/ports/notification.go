package ports

import (
	"context"

	"api-server/internal/app/dto"
)

// NotificationPort defines the interface for notification operations
// This allows domains to depend on notification functionality without direct coupling
type NotificationPort interface {
	CreateNotification(ctx context.Context, notification *dto.NotificationRequest) error
	SendEmailNotification(ctx context.Context, notification *dto.EmailNotificationRequest) error
	BroadcastNotification(ctx context.Context, notification *dto.BroadcastNotificationRequest) error
}

// EmailPublisherPort defines the interface for email publishing operations
type EmailPublisherPort interface {
	PublishEmail(ctx context.Context, email *dto.EmailRequest) error
}
