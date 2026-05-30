package infrastructure

import (
	"context"

	"api-server/internal/app/dto"
)

// NotificationPort defines the interface for notification operations
type NotificationPort interface {
	Create(ctx context.Context, notification *dto.CustomNotificationRequest) error
	Send(ctx context.Context, notificationID uint) error
	SendEmail(ctx context.Context, req *dto.SendEmailRequest) error
}
