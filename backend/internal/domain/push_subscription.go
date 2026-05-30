package domain

import (
	"context"
	"time"
)

// PushSubscription stores a user's web push subscription details
type PushSubscription struct {
	ID         uint      `json:"id" gorm:"primarykey;type:bigint unsigned"`
	UserID     uint      `json:"user_id" gorm:"not null;uniqueIndex:idx_push_sub_user_endpoint"`
	Endpoint   string    `json:"endpoint" gorm:"type:varchar(500);not null;uniqueIndex:idx_push_sub_user_endpoint"`
	P256DH     string    `json:"p256dh" gorm:"column:p256dh;type:varchar(200);not null"`
	Auth       string    `json:"auth" gorm:"type:varchar(100);not null"`
	DeviceType string    `json:"device_type" gorm:"type:varchar(20);not null;default:'web'"` // web, android, ios
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (PushSubscription) TableName() string {
	return "push_subscriptions"
}

// PushSubscriptionRepository manages push subscriptions
type PushSubscriptionRepository interface {
	Create(ctx context.Context, sub *PushSubscription) error
	GetByUserID(ctx context.Context, userID uint) ([]*PushSubscription, error)
	DeleteByEndpoint(ctx context.Context, userID uint, endpoint string) error
	DeleteByUserID(ctx context.Context, userID uint) error
}
