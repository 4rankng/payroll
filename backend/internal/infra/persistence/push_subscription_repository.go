package persistence

import (
	"context"

	"api-server/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type pushSubscriptionRepository struct {
	db *gorm.DB
}

func NewPushSubscriptionRepository(db *gorm.DB) domain.PushSubscriptionRepository {
	return &pushSubscriptionRepository{db: db}
}

func (r *pushSubscriptionRepository) Create(ctx context.Context, sub *domain.PushSubscription) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "endpoint"}},
		DoUpdates: clause.AssignmentColumns([]string{"p256dh", "auth", "device_type", "updated_at"}),
	}).Create(sub).Error
}

func (r *pushSubscriptionRepository) GetByUserID(ctx context.Context, userID uint) ([]*domain.PushSubscription, error) {
	var subs []*domain.PushSubscription
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&subs).Error
	return subs, err
}

func (r *pushSubscriptionRepository) DeleteByEndpoint(ctx context.Context, userID uint, endpoint string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND endpoint = ?", userID, endpoint).
		Delete(&domain.PushSubscription{}).Error
}

func (r *pushSubscriptionRepository) DeleteByUserID(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&domain.PushSubscription{}).Error
}
