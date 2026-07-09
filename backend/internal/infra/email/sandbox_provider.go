package email

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"api-server/internal/domain"
)

// SandboxProvider captures emails in memory without delivering them.
// Safe for tests and local development.
type SandboxProvider struct {
	mu      sync.RWMutex
	emails  []*CapturedEmail
	counter atomic.Uint64
	logger  *slog.Logger
}

// CapturedEmail stores a single email that was "sent" via the sandbox.
type CapturedEmail struct {
	ID         uint64
	Message    *domain.EmailMessage
	Result     *domain.EmailDeliveryResult
	CapturedAt time.Time
}

// NewSandboxProvider creates a sandbox that logs emails instead of sending.
func NewSandboxProvider(logger *slog.Logger) *SandboxProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &SandboxProvider{logger: logger}
}

func (p *SandboxProvider) Send(ctx context.Context, msg *domain.EmailMessage) (*domain.EmailDeliveryResult, error) {
	msg = cloneMessageWithPublicBanner(msg)
	id := p.counter.Add(1)
	now := clock.Now()

	result := &domain.EmailDeliveryResult{
		MessageID: fmt.Sprintf("sandbox-%d", id),
		Provider:  "sandbox",
		SentAt:    now,
		Metadata:  msg.Metadata,
	}

	p.mu.Lock()
	p.emails = append(p.emails, &CapturedEmail{
		ID:         id,
		Message:    msg,
		Result:     result,
		CapturedAt: now,
	})
	p.mu.Unlock()

	p.logger.Info("sandbox email captured",
		"message_id", result.MessageID,
		"subject", msg.Subject,
		"to", addressesTostrings(msg.To),
		"kind", string(msg.Kind),
	)

	return result, nil
}

// SentEmails returns all captured emails.
func (p *SandboxProvider) SentEmails() []*CapturedEmail {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*CapturedEmail, len(p.emails))
	copy(out, p.emails)
	return out
}

// LastEmail returns the most recently captured email, or nil.
func (p *SandboxProvider) LastEmail() *CapturedEmail {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.emails) == 0 {
		return nil
	}
	return p.emails[len(p.emails)-1]
}

// Reset clears all captured emails.
func (p *SandboxProvider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.emails = nil
}

// Count returns the number of captured emails.
func (p *SandboxProvider) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.emails)
}

func addressesTostrings(addrs []domain.EmailAddress) []string {
	if len(addrs) == 0 {
		return nil
	}
	values := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		values = append(values, addr.Address)
	}
	return values
}
