package zaloconnect

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/zalo"
	"api-server/internal/pkg/clock"
)

const (
	salaryNotificationLease       = 2 * time.Minute
	SalaryNotificationMaxAttempts = 5
)

// SalaryNotificationRecipient is the immutable notification payload selected
// while the FlexPay import is still processing its source workbook.
type SalaryNotificationRecipient struct {
	ProjectID    uint
	EmployeeID   uint
	EmployeeName string
	Mobile       string
	Amount       int64
	ExpiryDate   time.Time
}

// SalaryNotificationEnqueuer is intentionally narrow so scheduling stays in
// the application service and delivery remains an infrastructure concern.
type SalaryNotificationEnqueuer interface {
	EnqueueFlexPaySalaryNotification(notificationID uint) error
}

// SalaryNotificationDeliveryService persists, retries, and records ZNS salary
// notification delivery. It provides durable at-least-once delivery: Zalo does
// not expose a lookup or idempotency contract for an ambiguous response.
type SalaryNotificationDeliveryService struct {
	repo    domain.FlexPaySalaryNotificationRepository
	sender  zalo.Sender
	enqueue SalaryNotificationEnqueuer
	enabled func(context.Context) (bool, error)
	clock   clock.Clock
	logger  *slog.Logger
}

func NewSalaryNotificationDeliveryService(
	repo domain.FlexPaySalaryNotificationRepository,
	sender zalo.Sender,
	enqueue SalaryNotificationEnqueuer,
	enabled func(context.Context) (bool, error),
	clk clock.Clock,
	logger *slog.Logger,
) *SalaryNotificationDeliveryService {
	if logger == nil {
		logger = slog.Default()
	}
	if clk == nil {
		clk = clock.New()
	}
	return &SalaryNotificationDeliveryService{repo: repo, sender: sender, enqueue: enqueue, enabled: enabled, clock: clk, logger: logger}
}

// ScheduleImportRecipients creates one record per eligible import recipient.
// Queue outages never discard a persisted record: the periodic recovery sweep
// will enqueue it later.
func (s *SalaryNotificationDeliveryService) ScheduleImportRecipients(ctx context.Context, assetID uint, recipients []SalaryNotificationRecipient) error {
	if s == nil || s.repo == nil || len(recipients) == 0 {
		return nil
	}
	enabled, err := s.isEnabled(ctx)
	if err != nil {
		return fmt.Errorf("check FlexPay ZNS setting: %w", err)
	}
	if !enabled {
		return nil
	}

	now := s.clock.Now()
	for _, recipient := range recipients {
		phone := zalo.NormalizePhone(recipient.Mobile)
		if recipient.ProjectID == 0 || recipient.EmployeeID == 0 || recipient.Amount <= 0 || phone == "" || recipient.ExpiryDate.IsZero() {
			s.logger.Warn("flexpay salary notification skipped invalid recipient", "asset_id", assetID, "project_id", recipient.ProjectID, "employee_id", recipient.EmployeeID)
			continue
		}
		n, _, err := s.repo.CreateIfAbsent(ctx, &domain.FlexPaySalaryNotification{
			AssetID:      assetID,
			ProjectID:    recipient.ProjectID,
			EmployeeID:   recipient.EmployeeID,
			TemplateID:   FlexPaySalaryTemplateID,
			Phone:        phone,
			CustomerName: truncateString(recipient.EmployeeName, 30),
			MaxAmount:    recipient.Amount,
			ExpiryDate:   recipient.ExpiryDate,
			Status:       domain.FlexPaySalaryNotificationPending,
			EnqueuedAt:   &now,
		})
		if err != nil {
			return err
		}
		if err := s.enqueueNotification(n.ID); err != nil {
			s.logger.Warn("salary notification persisted but enqueue failed; recovery will retry", "notification_id", n.ID, "error", err)
		}
	}
	return nil
}

func (s *SalaryNotificationDeliveryService) Deliver(ctx context.Context, notificationID uint) error {
	if s == nil || s.repo == nil {
		return nil
	}
	now := s.clock.Now()
	n, claimed, err := s.repo.Claim(ctx, notificationID, now, now.Add(salaryNotificationLease))
	if err != nil || !claimed {
		return err
	}

	enabled, err := s.isEnabled(ctx)
	if err != nil {
		return s.retryOrFail(ctx, n, fmt.Sprintf("không kiểm tra được cài đặt ZNS: %v", err), 0)
	}
	if !enabled {
		return s.repo.MarkSuppressed(ctx, n.ID, n.Attempt, now, "Thông báo lương linh hoạt đang tắt", 0)
	}
	if s.sender == nil {
		return s.retryOrFail(ctx, n, "Zalo provider chưa được cấu hình", 0)
	}

	result, err := s.sender.Send(ctx, n.Phone, n.TemplateID, fmt.Sprintf("flexpay-salary-zns:%d", n.ID), map[string]string{
		"customer_name": n.CustomerName,
		"max_amount":    formatAmount(n.MaxAmount),
		"expiry_date":   n.ExpiryDate.Format("02/01/2006"),
	})
	if err != nil {
		return s.retryOrFail(ctx, n, fmt.Sprintf("lỗi gửi Zalo: %v", err), 0)
	}
	if result.ErrorCode == zalo.ErrOK {
		return s.repo.MarkSent(ctx, n.ID, n.Attempt, s.clock.Now(), result.MsgID)
	}
	if isRetryableSalaryNotificationError(result.ErrorCode) {
		return s.retryOrFail(ctx, n, result.ErrorMsg, result.ErrorCode)
	}
	reason := result.ErrorMsg
	if reason == "" {
		reason = zalo.ErrorMessage(result.ErrorCode)
	}
	return s.repo.MarkSuppressed(ctx, n.ID, n.Attempt, s.clock.Now(), reason, result.ErrorCode)
}

func (s *SalaryNotificationDeliveryService) Recover(ctx context.Context) error {
	if s == nil || s.repo == nil {
		return nil
	}
	notifications, err := s.repo.ListRecoverable(ctx, s.clock.Now(), 100)
	if err != nil {
		return err
	}
	for _, n := range notifications {
		if err := s.enqueueNotification(n.ID); err != nil {
			s.logger.Warn("failed to enqueue recoverable salary notification", "notification_id", n.ID, "error", err)
		}
	}
	return nil
}

func (s *SalaryNotificationDeliveryService) retryOrFail(ctx context.Context, n *domain.FlexPaySalaryNotification, reason string, providerCode int) error {
	if n.Attempt >= SalaryNotificationMaxAttempts {
		return s.repo.MarkFailed(ctx, n.ID, n.Attempt, s.clock.Now(), reason, providerCode)
	}
	if err := s.repo.ReleaseForRetry(ctx, n.ID, n.Attempt, reason); err != nil {
		return err
	}
	return fmt.Errorf("salary notification delivery retry: %s", reason)
}

func (s *SalaryNotificationDeliveryService) enqueueNotification(notificationID uint) error {
	if s.enqueue == nil {
		return fmt.Errorf("salary notification queue is not configured")
	}
	return s.enqueue.EnqueueFlexPaySalaryNotification(notificationID)
}

func (s *SalaryNotificationDeliveryService) isEnabled(ctx context.Context) (bool, error) {
	if s.enabled == nil {
		return false, nil
	}
	return s.enabled(ctx)
}

func isRetryableSalaryNotificationError(code int) bool {
	switch code {
	case zalo.ErrInsufficientBal, zalo.ErrDailyQuotaPhone, zalo.ErrDailyQuotaTpl, zalo.ErrZBSChargeFailed:
		return true
	default:
		return false
	}
}
