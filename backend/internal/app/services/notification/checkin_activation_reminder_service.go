package notification

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/domain"
)

type checkinActivationReminderRepository interface {
	GetEmployeesWithPendingCheckInEnableEffective(ctx context.Context, effectiveDate time.Time) ([]*domain.ProjectEmployee, error)
}

type checkinActivationReminderNotificationRepo interface {
	HasNotificationForUserOn(ctx context.Context, userID uint, notificationType domain.NotificationType, day time.Time) (bool, error)
}

type checkinActivationReminderNotifier interface {
	CreateNotification(ctx context.Context, userID uint, notificationType domain.NotificationType, title, message string) error
}

// CheckInActivationReminderService notifies employees one day before their
// pending self check-in activation takes effect (activations are always
// month-start, effective day 1). Delivery is in-app + web push via
// NotificationService.CreateNotification; the service never mutates
// assignment state.
type CheckInActivationReminderService struct {
	assignments      checkinActivationReminderRepository
	notifications    checkinActivationReminderNotifier
	notificationRepo checkinActivationReminderNotificationRepo
	logger           *slog.Logger
}

func NewCheckInActivationReminderService(
	assignments checkinActivationReminderRepository,
	notifications checkinActivationReminderNotifier,
	notificationRepo checkinActivationReminderNotificationRepo,
	logger *slog.Logger,
) *CheckInActivationReminderService {
	if logger == nil {
		logger = slog.Default()
	}
	return &CheckInActivationReminderService{
		assignments:      assignments,
		notifications:    notifications,
		notificationRepo: notificationRepo,
		logger:           logger,
	}
}

// Send notifies every employee whose pending check-in enable takes effect
// tomorrow (day 1 of the next month) and returns the number of employees
// notified. The month-end guard exits on any other day: the cron fires on
// days 28-31 so every month length gets exactly one acting run.
func (s *CheckInActivationReminderService) Send(ctx context.Context, now time.Time) (int, error) {
	tomorrow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
	if tomorrow.Day() != 1 {
		return 0, nil
	}

	assignments, err := s.assignments.GetEmployeesWithPendingCheckInEnableEffective(ctx, tomorrow)
	if err != nil {
		return 0, fmt.Errorf("list pending check-in activations for reminder: %w", err)
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	notified := 0
	for _, assignment := range assignments {
		userID := assignment.Employee.UserID
		if userID == nil {
			s.logger.Warn("check-in activation reminder skipped: no linked user",
				"assignment_id", assignment.ID, "employee_id", assignment.EmployeeID)
			continue
		}

		// Same-day dedup (PAYR-17 AC-06): a manual job re-run must not spam
		// the employee. On a dedup-check error, skip — a missed reminder is
		// softer than a duplicate.
		exists, err := s.notificationRepo.HasNotificationForUserOn(ctx, *userID, domain.NotificationTypeCheckInActivationReminder, today)
		if err != nil {
			s.logger.Error("check-in activation reminder dedup check failed", "user_id", *userID, "error", err)
			continue
		}
		if exists {
			continue
		}

		title := "Chấm công tự động sắp kích hoạt"
		message := fmt.Sprintf(
			"Tính năng tự chấm công sẽ được kích hoạt từ ngày %s. Hãy mở ứng dụng để điểm danh nhé!",
			tomorrow.Format("02/01/2006"),
		)
		if err := s.notifications.CreateNotification(ctx, *userID, domain.NotificationTypeCheckInActivationReminder, title, message); err != nil {
			s.logger.Error("failed to send check-in activation reminder", "user_id", *userID, "error", err)
			continue
		}
		notified++
	}

	return notified, nil
}
