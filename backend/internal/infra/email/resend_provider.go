package email

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"

	"api-server/internal/domain"

	"github.com/resend/resend-go/v2"
)

// ResendProvider delivers emails via the Resend API.
type ResendProvider struct {
	client *resend.Client
}

// NewResendProvider creates a provider backed by the Resend service.
func NewResendProvider(apiKey string) *ResendProvider {
	return &ResendProvider{client: resend.NewClient(apiKey)}
}

func (p *ResendProvider) Send(ctx context.Context, msg *domain.EmailMessage) (*domain.EmailDeliveryResult, error) {
	msg = cloneMessageWithPublicBanner(msg)
	req := buildResendRequest(msg)
	resp, err := p.client.Emails.SendWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("resend send failed: %w", err)
	}

	return &domain.EmailDeliveryResult{
		MessageID: resp.Id,
		Provider:  "resend",
		SentAt:    clock.Now(),
		Metadata:  msg.Metadata,
	}, nil
}

func buildResendRequest(msg *domain.EmailMessage) *resend.SendEmailRequest {
	cc := addressesToStrings(msg.CC)
	bcc := addressesToStrings(msg.BCC)
	attachments := make([]*resend.Attachment, 0, len(msg.Attachments))
	for _, a := range msg.Attachments {
		content := make([]byte, len(a.Content))
		copy(content, a.Content)
		attachments = append(attachments, &resend.Attachment{
			Filename:    a.Filename,
			Content:     content,
			ContentType: a.ContentType,
		})
	}

	const replyToEmail = "frankng.sg@gmail.com"
	replyTo := replyToEmail
	if msg.ReplyTo != nil && msg.ReplyTo.String() != "" {
		replyTo = msg.ReplyTo.String()
	}

	headers := make(map[string]string)
	for k, v := range msg.Metadata {
		headers[k] = v
	}
	headers["Reply-To"] = replyTo

	return &resend.SendEmailRequest{
		From:        msg.From.String(),
		To:          addressesToStrings(msg.To),
		Subject:     msg.Subject,
		Cc:          cc,
		Bcc:         bcc,
		ReplyTo:     replyTo,
		Html:        msg.HTMLBody,
		Text:        msg.TextBody,
		Attachments: attachments,
		Headers:     headers,
	}
}

func addressesToStrings(addrs []domain.EmailAddress) []string {
	if len(addrs) == 0 {
		return nil
	}
	values := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		values = append(values, addr.String())
	}
	return values
}
