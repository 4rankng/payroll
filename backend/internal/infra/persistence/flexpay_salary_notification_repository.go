package persistence

import (
	"context"
	"time"

	"api-server/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FlexPaySalaryNotificationRepository struct{ *BaseRepository }

func NewFlexPaySalaryNotificationRepository(db *Database) domain.FlexPaySalaryNotificationRepository {
	return &FlexPaySalaryNotificationRepository{BaseRepository: NewBaseRepository(db)}
}

func (r *FlexPaySalaryNotificationRepository) CreateIfAbsent(ctx context.Context, n *domain.FlexPaySalaryNotification) (*domain.FlexPaySalaryNotification, bool, error) {
	db := r.db(ctx)
	result := db.Clauses(clause.OnConflict{DoNothing: true}).Create(n)
	if result.Error != nil {
		return nil, false, domain.NewInternalError("failed to create salary notification", result.Error)
	}
	created := result.RowsAffected == 1
	if created {
		return n, true, nil
	}
	var existing domain.FlexPaySalaryNotification
	if err := db.Where("asset_id = ? AND project_id = ? AND employee_id = ? AND template_id = ?", n.AssetID, n.ProjectID, n.EmployeeID, n.TemplateID).First(&existing).Error; err != nil {
		return nil, false, domain.NewInternalError("failed to load existing salary notification", err)
	}
	return &existing, false, nil
}

func (r *FlexPaySalaryNotificationRepository) Claim(ctx context.Context, id uint, now, leaseUntil time.Time) (*domain.FlexPaySalaryNotification, bool, error) {
	result := r.db(ctx).Model(&domain.FlexPaySalaryNotification{}).
		Where("id = ? AND (status = ? OR (status = ? AND lease_expires_at < ?))", id, domain.FlexPaySalaryNotificationPending, domain.FlexPaySalaryNotificationProcessing, now).
		Updates(map[string]any{
			"status":           domain.FlexPaySalaryNotificationProcessing,
			"lease_expires_at": leaseUntil,
			"attempt":          gorm.Expr("attempt + 1"),
		})
	if result.Error != nil {
		return nil, false, domain.NewInternalError("failed to claim salary notification", result.Error)
	}
	if result.RowsAffected != 1 {
		return nil, false, nil
	}
	var notification domain.FlexPaySalaryNotification
	if err := r.db(ctx).First(&notification, id).Error; err != nil {
		return nil, false, domain.NewInternalError("failed to load claimed salary notification", err)
	}
	return &notification, true, nil
}

func (r *FlexPaySalaryNotificationRepository) ReleaseForRetry(ctx context.Context, id uint, attempt uint, reason string) error {
	return r.finish(ctx, id, attempt, domain.FlexPaySalaryNotificationPending, nil, reason, 0, "")
}

func (r *FlexPaySalaryNotificationRepository) MarkSent(ctx context.Context, id uint, attempt uint, sentAt time.Time, providerMsgID string) error {
	return r.finish(ctx, id, attempt, domain.FlexPaySalaryNotificationSent, &sentAt, "", 0, providerMsgID)
}

func (r *FlexPaySalaryNotificationRepository) MarkFailed(ctx context.Context, id uint, attempt uint, _ time.Time, reason string, providerCode int) error {
	return r.finish(ctx, id, attempt, domain.FlexPaySalaryNotificationFailed, nil, reason, providerCode, "")
}

func (r *FlexPaySalaryNotificationRepository) MarkSuppressed(ctx context.Context, id uint, attempt uint, _ time.Time, reason string, providerCode int) error {
	return r.finish(ctx, id, attempt, domain.FlexPaySalaryNotificationSuppressed, nil, reason, providerCode, "")
}

func (r *FlexPaySalaryNotificationRepository) finish(ctx context.Context, id uint, attempt uint, status string, at *time.Time, reason string, providerCode int, providerMsgID string) error {
	result := r.db(ctx).Model(&domain.FlexPaySalaryNotification{}).
		Where("id = ? AND status = ? AND attempt = ?", id, domain.FlexPaySalaryNotificationProcessing, attempt).
		Updates(map[string]any{
			"status":           status,
			"lease_expires_at": nil,
			"sent_at":          at,
			"last_error":       reason,
			"provider_code":    providerCode,
			"provider_msg_id":  providerMsgID,
		})
	if result.Error != nil {
		return domain.NewInternalError("failed to finish salary notification", result.Error)
	}
	if result.RowsAffected != 1 {
		return domain.NewConflictError("salary notification attempt is no longer active")
	}
	return nil
}

func (r *FlexPaySalaryNotificationRepository) ListRecoverable(ctx context.Context, now time.Time, limit int) ([]*domain.FlexPaySalaryNotification, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var notifications []*domain.FlexPaySalaryNotification
	err := r.db(ctx).Where("status = ? OR (status = ? AND lease_expires_at < ?)", domain.FlexPaySalaryNotificationPending, domain.FlexPaySalaryNotificationProcessing, now).Order("created_at ASC").Limit(limit).Find(&notifications).Error
	if err != nil {
		return nil, domain.NewInternalError("failed to list recoverable salary notifications", err)
	}
	return notifications, nil
}

func (r *FlexPaySalaryNotificationRepository) db(ctx context.Context) *gorm.DB {
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.DB.WithContext(ctx)
}
