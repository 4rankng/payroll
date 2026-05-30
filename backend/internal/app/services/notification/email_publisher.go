package notification

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"api-server/internal/constants"
	"api-server/internal/domain"
	auditctx "api-server/internal/pkg/context"
)

// EmailNotificationPublisher implements the domain.EmailEventPublisher interface
// by creating notification records when emails are sent.
type EmailNotificationPublisher struct {
	notificationRepo domain.NotificationRepository
	logger           *slog.Logger
}

// NewEmailNotificationPublisher constructs a new EmailNotificationPublisher instance.
func NewEmailNotificationPublisher(notificationRepo domain.NotificationRepository, logger *slog.Logger) *EmailNotificationPublisher {
	return &EmailNotificationPublisher{
		notificationRepo: notificationRepo,
		logger:           logger,
	}
}

// PublishEmailSent creates a notification record tracking the sent email.
// This implements the Observer pattern for email audit logging.
func (p *EmailNotificationPublisher) PublishEmailSent(ctx context.Context, event *domain.EmailSentEvent) error {
	if event == nil {
		return domain.NewValidationError(constants.MsgEmailSentEventRequiredVN)
	}

	// Serialize recipients to JSON for storage
	recipientsJSON, err := json.Marshal(event.Recipients)
	if err != nil {
		p.logger.Error("failed to serialize email recipients", "error", err)
		return domain.NewInternalError(constants.MsgFailedToSerializeRecipientsVN, err)
	}

	recipientsStr := string(recipientsJSON)
	messageID := event.MessageID

	// Serialize payroll metadata if present
	var emailMetadataStr *string
	if event.PayrollMetadata != nil {
		metaJSON, err := json.Marshal(event.PayrollMetadata)
		if err != nil {
			p.logger.Warn("failed to serialize payroll email metadata", "error", err)
		} else {
			s := string(metaJSON)
			emailMetadataStr = &s
		}
	}

	// Create notification record with email channel
	// SenderID: who initiated the email send (user or system)
	// RecipientID: nil for emails to external recipients
	senderID := constants.SystemUserID
	if ctxUserID := auditctx.GetUserID(ctx); ctxUserID != nil && *ctxUserID > 0 {
		senderID = *ctxUserID
	} else if event.InitiatedBy != nil && *event.InitiatedBy > 0 {
		senderID = *event.InitiatedBy
	}

	notification := &domain.Notification{
		SenderID:        senderID,
		RecipientID:     nil, // Emails go to external recipients
		Type:            notificationTypeFromKind(event.Kind),
		Channel:         domain.NotificationChannelEmail,
		Title:           event.Subject,
		Message:         buildEmailMessage(event),
		ContentType:     detectContentType(event.TextBody),
		EmailRecipients: &recipientsStr,
		Metadata:        emailMetadataStr,
		ResendMessageID: &messageID,
	}

	if err := p.notificationRepo.Create(ctx, notification); err != nil {
		p.logger.Error("failed to create email notification record",
			"error", err,
			"message_id", event.MessageID,
			"subject", event.Subject,
		)
		return domain.NewInternalError(constants.MsgFailedToLogEmailNotificationVN, err)
	}

	p.logger.Info("email notification recorded",
		"message_id", event.MessageID,
		"subject", event.Subject,
		"recipient_count", len(event.Recipients),
	)

	return nil
}

// buildEmailMessage returns the actual email body text for the notification record.
func buildEmailMessage(event *domain.EmailSentEvent) string {
	return event.TextBody
}

// detectContentType analyzes the message content to determine if it's HTML, Markdown, or plain text
func detectContentType(message string) domain.NotificationContentType {
	trimmed := strings.TrimSpace(message)

	// Check for HTML tags
	if strings.Contains(trimmed, "<html") ||
		strings.Contains(trimmed, "<!DOCTYPE") ||
		strings.Contains(trimmed, "<body") ||
		strings.Contains(trimmed, "<div") ||
		strings.Contains(trimmed, "<p>") ||
		strings.Contains(trimmed, "<br") ||
		strings.Contains(trimmed, "<span") {
		return domain.NotificationContentTypeHTML
	}

	// Check for common Markdown patterns
	if strings.Contains(trimmed, "##") ||
		strings.Contains(trimmed, "**") ||
		strings.Contains(trimmed, "__") ||
		strings.Contains(trimmed, "[") && strings.Contains(trimmed, "](") {
		return domain.NotificationContentTypeMarkdown
	}

	// Default to plain text
	return domain.NotificationContentTypePlainText
}

func notificationTypeFromKind(kind domain.EmailMessageKind) domain.NotificationType {
	switch kind {
	case domain.EmailKindPayrollReport:
		return domain.NotificationTypePayrollReport
	case domain.EmailKindAdvancePaymentReport:
		return domain.NotificationTypeAdvancePaymentReport
	default:
		return domain.NotificationTypeCustom
	}
}
