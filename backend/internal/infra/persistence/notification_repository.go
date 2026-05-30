package persistence

import (
	"api-server/internal/pkg/clock"
	"context"

	"api-server/internal/domain"
)

type NotificationRepository struct {
	*BaseRepository
}

func NewNotificationRepository(db *Database) domain.NotificationRepository {
	return &NotificationRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create creates a new notification
func (r *NotificationRepository) Create(ctx context.Context, notification *domain.Notification) error {
	return r.DB.WithContext(ctx).Create(notification).Error
}

// GetByID retrieves a notification by its primary key
func (r *NotificationRepository) GetByID(ctx context.Context, id uint) (*domain.Notification, error) {
	var notification domain.Notification
	err := r.DB.WithContext(ctx).
		Preload("Sender").
		First(&notification, id).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

// GetByUserID retrieves notifications for a user with pagination
func (r *NotificationRepository) GetByUserID(ctx context.Context, userID uint, limit, offset int) ([]*domain.Notification, error) {
	var notifications []*domain.Notification
	err := r.DB.WithContext(ctx).
		Where("recipient_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error
	return notifications, err
}

// GetUnreadByUserID retrieves unread notifications for a user
func (r *NotificationRepository) GetUnreadByUserID(ctx context.Context, userID uint) ([]*domain.Notification, error) {
	var notifications []*domain.Notification
	err := r.DB.WithContext(ctx).
		Where("recipient_id = ? AND read_at IS NULL", userID).
		Order("created_at DESC").
		Find(&notifications).Error
	return notifications, err
}

// ListByChannel retrieves notifications by delivery channel with pagination
func (r *NotificationRepository) ListByChannel(ctx context.Context, channel domain.NotificationChannel, limit, offset int) ([]*domain.Notification, error) {
	var notifications []*domain.Notification
	err := r.DB.WithContext(ctx).
		Preload("Sender").
		Where("channel = ?", channel).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error
	return notifications, err
}

// CountUnreadByUserID counts unread notifications for a user
func (r *NotificationRepository) CountUnreadByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Notification{}).
		Where("recipient_id = ? AND read_at IS NULL", userID).
		Count(&count).Error
	return count, err
}

// CountByUserID counts all notifications for a user
func (r *NotificationRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Notification{}).
		Where("recipient_id = ?", userID).
		Count(&count).Error
	return count, err
}

// CountByChannel counts notifications for a given delivery channel
func (r *NotificationRepository) CountByChannel(ctx context.Context, channel domain.NotificationChannel) (int64, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Notification{}).
		Where("channel = ?", channel).
		Count(&count).Error
	return count, err
}

// MarkAsRead marks a notification as read
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id uint) error {
	return r.DB.WithContext(ctx).
		Model(&domain.Notification{}).
		Where("id = ?", id).
		Update("read_at", clock.Now()).Error
}

// MarkAllAsReadForUser marks all notifications as read for a user
func (r *NotificationRepository) MarkAllAsReadForUser(ctx context.Context, userID uint) error {
	return r.DB.WithContext(ctx).
		Model(&domain.Notification{}).
		Where("recipient_id = ? AND read_at IS NULL", userID).
		Update("read_at", clock.Now()).Error
}

// MarkOldNotificationsAsRead marks notifications older than specified days as read for all users
func (r *NotificationRepository) MarkOldNotificationsAsRead(ctx context.Context, olderThanDays int) (int64, error) {
	result := r.DB.WithContext(ctx).
		Model(&domain.Notification{}).
		Where("read_at IS NULL AND created_at <= DATE_SUB(NOW(), INTERVAL ? DAY)", olderThanDays).
		Update("read_at", clock.Now())

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// UpdateMetadata updates the metadata JSON column for a notification.
func (r *NotificationRepository) UpdateMetadata(ctx context.Context, id uint, metadata string) error {
	return r.DB.WithContext(ctx).
		Model(&domain.Notification{}).
		Where("id = ?", id).
		Update("metadata", metadata).Error
}
