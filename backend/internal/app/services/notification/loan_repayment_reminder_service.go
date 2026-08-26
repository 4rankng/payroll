package notification

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"sort"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/pkg/utils"
)

type loanRepaymentReminderRepository interface {
	ListPendingForReminder(ctx context.Context, start, end time.Time) ([]*domain.LoanRepaymentReminder, error)
}

type loanRepaymentReminderUserRepository interface {
	ListByRole(ctx context.Context, role domain.UserRole) ([]*domain.User, error)
}

type loanRepaymentReminderNotifier interface {
	NotifyUsersByRole(ctx context.Context, role domain.UserRole, notificationType domain.NotificationType, title, message string) error
}

type loanRepaymentReminderEmailSender interface {
	SendGenericEmail(ctx context.Context, payload *dto.SendEmailRequest) (string, error)
}

// LoanRepaymentReminderService sends a consolidated due-date reminder to
// every administrator on the interest payout day itself (never in advance).
// Push and email are independent best-effort channels; this service never
// mutates loan or repayment state.
type LoanRepaymentReminderService struct {
	schedules     loanRepaymentReminderRepository
	users         loanRepaymentReminderUserRepository
	notifications loanRepaymentReminderNotifier
	email         loanRepaymentReminderEmailSender
	logger        *slog.Logger
}

func NewLoanRepaymentReminderService(
	schedules loanRepaymentReminderRepository,
	users loanRepaymentReminderUserRepository,
	notifications loanRepaymentReminderNotifier,
	email loanRepaymentReminderEmailSender,
	logger *slog.Logger,
) *LoanRepaymentReminderService {
	if logger == nil {
		logger = slog.Default()
	}
	return &LoanRepaymentReminderService{
		schedules:     schedules,
		users:         users,
		notifications: notifications,
		email:         email,
		logger:        logger,
	}
}

// Send finds schedules due today (the interest payout date itself) in now's
// location and dispatches one consolidated reminder — same day only, never
// in advance. A missed run (deploy, downtime) therefore skips that day's
// reminder; the daily 09:00 cron is monitored via cron_job_status. It
// returns the number of schedules.
func (s *LoanRepaymentReminderService) Send(ctx context.Context, now time.Time) (int, error) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start, end := today, today.AddDate(0, 0, 1)

	reminders, err := s.schedules.ListPendingForReminder(ctx, start, end)
	if err != nil {
		return 0, fmt.Errorf("list pending loan repayments for reminder: %w", err)
	}
	if len(reminders) == 0 {
		return 0, nil
	}

	sort.SliceStable(reminders, func(i, j int) bool {
		if reminders[i].LoanCode == reminders[j].LoanCode {
			if reminders[i].Period == reminders[j].Period {
				return reminders[i].ScheduleID < reminders[j].ScheduleID
			}
			return reminders[i].Period < reminders[j].Period
		}
		return reminders[i].LoanCode < reminders[j].LoanCode
	})

	title, textBody, htmlBody := buildLoanRepaymentReminderContent(today, reminders)
	var deliveryErrors []error

	if s.notifications == nil {
		deliveryErrors = append(deliveryErrors, errors.New("loan repayment reminder notifications are not configured"))
	} else if err := s.notifications.NotifyUsersByRole(
		ctx,
		domain.RoleAdmin,
		domain.NotificationTypeLoanInterestDue,
		title,
		textBody,
	); err != nil {
		s.logger.Error("Failed to send loan repayment reminder notification", "error", err)
		deliveryErrors = append(deliveryErrors, fmt.Errorf("send loan repayment reminder notification: %w", err))
	}

	if s.users == nil || s.email == nil {
		deliveryErrors = append(deliveryErrors, errors.New("loan repayment reminder email is not configured"))
		return len(reminders), errors.Join(deliveryErrors...)
	}

	admins, err := s.users.ListByRole(ctx, domain.RoleAdmin)
	if err != nil {
		s.logger.Error("Failed to list admin email recipients for loan repayment reminder", "error", err)
		deliveryErrors = append(deliveryErrors, fmt.Errorf("list admin email recipients: %w", err))
		return len(reminders), errors.Join(deliveryErrors...)
	}

	recipients := uniqueAdminEmails(admins)
	if len(recipients) == 0 {
		s.logger.Warn("No admin email recipients configured for loan repayment reminder")
		return len(reminders), errors.Join(deliveryErrors...)
	}

	if _, err := s.email.SendGenericEmail(ctx, &dto.SendEmailRequest{
		Recipients: recipients,
		Subject:    title,
		HTMLBody:   htmlBody,
		TextBody:   textBody,
	}); err != nil {
		s.logger.Error("Failed to send loan repayment reminder email", "error", err)
		deliveryErrors = append(deliveryErrors, fmt.Errorf("send loan repayment reminder email: %w", err))
	}

	return len(reminders), errors.Join(deliveryErrors...)
}

func buildLoanRepaymentReminderContent(today time.Time, reminders []*domain.LoanRepaymentReminder) (string, string, string) {
	// The query window is [today, tomorrow), so every reminder is due today.
	var textBody strings.Builder
	var htmlBody strings.Builder
	when := "hôm nay " + today.Format("02/01/2006")
	title := "Nhắc thanh toán lãi vay: " + when

	fmt.Fprintf(&textBody, "Có %d kỳ thanh toán khoản vay đến hạn %s:\n", len(reminders), when)
	fmt.Fprintf(&htmlBody, "<p>Có <strong>%d</strong> kỳ thanh toán khoản vay đến hạn %s:</p><ul>", len(reminders), when)

	var total int64
	for _, reminder := range reminders {
		total += reminder.Amount
		amount := utils.FormatVND(reminder.Amount)
		dueLabel := "đến hạn " + reminder.DueDate.In(today.Location()).Format("02/01/2006")
		fmt.Fprintf(
			&textBody,
			"- %s · Kỳ %d · %s · %s · %s\n",
			reminder.LoanCode,
			reminder.Period,
			reminder.LenderName,
			amount,
			dueLabel,
		)
		fmt.Fprintf(
			&htmlBody,
			"<li><strong>%s</strong> · Kỳ %d · %s · %s · %s</li>",
			html.EscapeString(reminder.LoanCode),
			reminder.Period,
			html.EscapeString(reminder.LenderName),
			html.EscapeString(amount),
			html.EscapeString(dueLabel),
		)
	}

	fmt.Fprintf(&textBody, "Tổng số tiền theo lịch: %s\nVui lòng chuẩn bị thanh toán đúng hạn.", utils.FormatVND(total))
	fmt.Fprintf(&htmlBody, "</ul><p><strong>Tổng số tiền theo lịch: %s</strong></p>", html.EscapeString(utils.FormatVND(total)))
	htmlBody.WriteString("<p>Vui lòng chuẩn bị thanh toán đúng hạn.</p>")

	return title, textBody.String(), htmlBody.String()
}

func uniqueAdminEmails(admins []*domain.User) []string {
	seen := make(map[string]struct{}, len(admins))
	recipients := make([]string, 0, len(admins))
	for _, admin := range admins {
		if admin == nil || admin.Email == nil {
			continue
		}
		emailAddress := strings.TrimSpace(*admin.Email)
		if emailAddress == "" {
			continue
		}
		key := strings.ToLower(emailAddress)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		recipients = append(recipients, emailAddress)
	}
	return recipients
}
