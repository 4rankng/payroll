package notification

import (
	"context"
	"fmt"
	"log/slog"

	"api-server/internal/constants"
	"api-server/internal/domain"
	pkgConstants "api-server/internal/pkg/constants"
	auditctx "api-server/internal/pkg/context"
)

// PaymentCycleNotificationPublisher implements the domain.PayCycleEventPublisher interface
// by creating notification records when payment schedules change.
// This follows the Observer pattern to decouple payment cycle logic from notification delivery.
type PaymentCycleNotificationPublisher struct {
	notificationRepo domain.NotificationRepository
	logger           *slog.Logger
}

// NewPaymentCycleNotificationPublisher constructs a new PaymentCycleNotificationPublisher instance.
func NewPaymentCycleNotificationPublisher(
	notificationRepo domain.NotificationRepository,
	logger *slog.Logger,
) *PaymentCycleNotificationPublisher {
	return &PaymentCycleNotificationPublisher{
		notificationRepo: notificationRepo,
		logger:           logger,
	}
}

// PublishPayCycleChanged creates a notification record for the payment schedule change.
// Notifications are sent to:
// - The employee (if they have an account)
// - Admins/managers who oversee the project
func (p *PaymentCycleNotificationPublisher) PublishPayCycleChanged(
	ctx context.Context,
	event *domain.PayCycleChangedEvent,
) error {
	if event == nil {
		return domain.NewValidationError(constants.MsgPaymentCycleChangedEventRequiredVN)
	}

	// Determine sender (who initiated the change)
	senderID := constants.SystemUserID
	if ctxUserID := auditctx.GetUserID(ctx); ctxUserID != nil && *ctxUserID > 0 {
		senderID = *ctxUserID
	} else if event.ChangedBy != nil && *event.ChangedBy > 0 {
		senderID = *event.ChangedBy
	}

	// Build notification message
	title, message := p.buildNotificationContent(event)

	// Create notification for the employee.
	// RecipientID is intentionally left nil here so this remains a system-level notification
	// until we introduce a reliable mapping from employee records to user accounts.
	notification := &domain.Notification{
		SenderID:    senderID,
		RecipientID: nil,
		Type:        domain.NotificationTypePaymentScheduleChange,
		Channel:     domain.NotificationChannelPush,
		Title:       title,
		Message:     message,
		ContentType: domain.NotificationContentTypePlainText,
	}

	if err := p.notificationRepo.Create(ctx, notification); err != nil {
		p.logger.Error("failed to create payment cycle change notification",
			"error", err,
			"project_id", event.ProjectID,
			"employee_id", event.EmployeeID,
			"old_cycle", event.OldCycle,
			"new_cycle", event.NewCycle,
		)
		return domain.NewInternalError(constants.MsgFailedToLogPaymentCycleChangeNotificationVN, err)
	}

	p.logger.Info("payment cycle change notification recorded",
		"project_id", event.ProjectID,
		"employee_id", event.EmployeeID,
		"old_cycle", event.OldCycle,
		"new_cycle", event.NewCycle,
		"effective_from", event.EffectiveFrom,
		"is_immediate", event.IsImmediate,
	)

	return nil
}

// buildNotificationContent creates the title and message for the notification.
func (p *PaymentCycleNotificationPublisher) buildNotificationContent(
	event *domain.PayCycleChangedEvent,
) (title, message string) {
	cycleLabel := map[string]string{
		pkgConstants.PaymentScheduleWeekly:  "tuần",
		pkgConstants.PaymentScheduleMonthly: "tháng",
	}

	oldLabel := cycleLabel[event.OldCycle]
	newLabel := cycleLabel[event.NewCycle]

	if event.IsImmediate {
		title = "Thay đổi chu kỳ trả lương"
		message = fmt.Sprintf(
			"Chu kỳ trả lương của nhân viên %s tại dự án %s đã được chuyển từ %s sang %s. "+
				"Thay đổi có hiệu lực ngay lập tức từ %s.",
			event.EmployeeName,
			event.ProjectName,
			oldLabel,
			newLabel,
			event.EffectiveFrom.Format("02/01/2006"),
		)
	} else {
		title = "Lên lịch thay đổi chu kỳ trả lương"
		message = fmt.Sprintf(
			"Chu kỳ trả lương của nhân viên %s tại dự án %s sẽ được chuyển từ %s sang %s. "+
				"Thay đổi sẽ có hiệu lực vào đầu tháng kế tiếp: %s.",
			event.EmployeeName,
			event.ProjectName,
			oldLabel,
			newLabel,
			event.EffectiveFrom.Format("02/01/2006"),
		)
	}

	return title, message
}
