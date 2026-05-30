package domain

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"api-server/internal/pkg/clock"
)

const (
	// MaxEmailRecipients defines the maximum number of recipients allowed per list
	MaxEmailRecipients = 50
	// MaxEmailSubjectLength limits the subject length to keep messages concise
	MaxEmailSubjectLength = 150
	// MaxEmailBodyLength caps the HTML or text body to 100KB
	MaxEmailBodyLength = 100 * 1024
	// MaxEmailAttachments constrains how many files can be sent at once
	MaxEmailAttachments = 5
	// MaxEmailAttachmentSize prevents oversized payloads (10 MB)
	MaxEmailAttachmentSize = 10 * 1024 * 1024
)

// EmailMessageKind distinguishes different functional emails for observability.
type EmailMessageKind string

const (
	EmailKindGeneric              EmailMessageKind = "generic"
	EmailKindPayrollReport        EmailMessageKind = "payroll_report"
	EmailKindAdvancePaymentReport EmailMessageKind = "advance_payment_report"
)

// EmailAddress represents an email address with optional display name.
type EmailAddress struct {
	Name    string
	Address string
}

// String returns the RFC 5322 compliant representation of the address.
func (a EmailAddress) String() string {
	if a.Name == "" {
		return a.Address
	}
	return fmt.Sprintf("%s <%s>", a.Name, a.Address)
}

// Mask obfuscates the local part for safe logging.
func (a EmailAddress) Mask() string {
	parts := strings.Split(a.Address, "@")
	if len(parts) != 2 {
		return "***"
	}
	local := parts[0]
	if local == "" {
		return "***@" + parts[1]
	}
	runeSlice := []rune(local)
	return fmt.Sprintf("%s***@%s", string(runeSlice[0]), parts[1])
}

// ParseEmailAddress parses and validates a raw email string.
func ParseEmailAddress(raw string) (EmailAddress, error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil {
		return EmailAddress{}, fmt.Errorf("invalid email address '%s': %w", raw, err)
	}
	return EmailAddress{Name: addr.Name, Address: strings.ToLower(addr.Address)}, nil
}

// EmailAttachment represents a binary attachment.
type EmailAttachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// Size returns the attachment size in bytes.
func (a EmailAttachment) Size() int {
	return len(a.Content)
}

// EmailMessage captures an email payload ready for delivery.
type EmailMessage struct {
	Kind        EmailMessageKind
	From        EmailAddress
	ReplyTo     *EmailAddress
	To          []EmailAddress
	CC          []EmailAddress
	BCC         []EmailAddress
	Subject     string
	HTMLBody    string
	TextBody    string
	Attachments []EmailAttachment
	Metadata    map[string]string
	CreatedAt   time.Time
}

// Validate verifies message invariants and returns a domain validation error on failure.
func (m *EmailMessage) Validate() error {
	if len(m.To) == 0 {
		return NewValidationError("email requires at least one recipient")
	}
	if len(m.To) > MaxEmailRecipients {
		return NewValidationError("recipient list exceeds allowed size")
	}
	if len(m.CC) > MaxEmailRecipients {
		return NewValidationError("cc list exceeds allowed size")
	}
	if len(m.BCC) > MaxEmailRecipients {
		return NewValidationError("bcc list exceeds allowed size")
	}

	subject := strings.TrimSpace(m.Subject)
	if subject == "" {
		return NewValidationError("email subject is required")
	}
	if utf8.RuneCountInString(subject) > MaxEmailSubjectLength {
		return NewValidationError("email subject exceeds maximum length")
	}

	htmlBodyLen := len(m.HTMLBody)
	textBodyLen := len(m.TextBody)
	if htmlBodyLen == 0 && textBodyLen == 0 {
		return NewValidationError("email body is required")
	}
	if htmlBodyLen > MaxEmailBodyLength {
		return NewValidationError("HTML body exceeds maximum size")
	}
	if textBodyLen > MaxEmailBodyLength {
		return NewValidationError("text body exceeds maximum size")
	}

	if len(m.Attachments) > MaxEmailAttachments {
		return NewValidationError("too many attachments")
	}
	for _, att := range m.Attachments {
		if att.Filename == "" {
			return NewValidationError("attachment filename is required")
		}
		if att.Size() == 0 {
			return NewValidationError(fmt.Sprintf("attachment '%s' is empty", att.Filename))
		}
		if att.Size() > MaxEmailAttachmentSize {
			return NewValidationError(fmt.Sprintf("attachment '%s' exceeds %d bytes", att.Filename, MaxEmailAttachmentSize))
		}
	}

	if m.Metadata == nil {
		m.Metadata = map[string]string{}
	}
	if m.Kind == "" {
		m.Kind = EmailKindGeneric
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = clock.Now()
	}

	return nil
}

// EmailDeliveryResult captures provider details after a successful send.
type EmailDeliveryResult struct {
	MessageID string
	Provider  string
	SentAt    time.Time
	Metadata  map[string]string
}

// EmailDeliveryPort abstracts infrastructure providers such as Resend.
type EmailDeliveryPort interface {
	Send(ctx context.Context, msg *EmailMessage) (*EmailDeliveryResult, error)
}

// EmailSentEvent is published after a message is accepted by the provider.
type EmailSentEvent struct {
	MessageID       string
	Kind            EmailMessageKind
	Subject         string
	TextBody        string
	Provider        string
	Recipients      []EmailAddress
	CC              []EmailAddress
	BCC             []EmailAddress
	Metadata        map[string]string
	PayrollMetadata *PayrollEmailMetadata // populated for payroll_report emails
	SentAt          time.Time
	InitiatedBy     *uint // User ID of who initiated the email send (nil for system-initiated)
}

// EmailEventPublisher allows application services to notify observers.
type EmailEventPublisher interface {
	PublishEmailSent(ctx context.Context, event *EmailSentEvent) error
}

// NewEmailSentEvent creates an event with sane defaults.
func NewEmailSentEvent(result *EmailDeliveryResult, msg *EmailMessage, initiatedBy *uint) *EmailSentEvent {
	return &EmailSentEvent{
		MessageID:   result.MessageID,
		Kind:        msg.Kind,
		Subject:     msg.Subject,
		TextBody:    msg.TextBody,
		Provider:    result.Provider,
		Recipients:  append([]EmailAddress{}, msg.To...),
		CC:          append([]EmailAddress{}, msg.CC...),
		BCC:         append([]EmailAddress{}, msg.BCC...),
		Metadata:    result.Metadata,
		SentAt:      result.SentAt,
		InitiatedBy: initiatedBy,
	}
}
