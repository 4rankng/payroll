package zalo

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
)

// SandboxSender is a dev/test Sender that captures ZNS sends in memory instead
// of hitting the real Zalo API. It implements zalo.Sender so it drops into the
// zaloreset.Service transparently. Use it in local dev (set
// ZALO_USE_SANDBOX=true) to test the full reset flow without a Zalo OA — the
// OTP code is logged and queryable via LastSend().
//
// Mirrors the email package's SandboxProvider pattern.
type SandboxSender struct {
	mu      sync.RWMutex
	sends   []*SandboxSend
	counter atomic.Uint64
	logger  *slog.Logger
}

// SandboxSend is one captured ZNS "send".
type SandboxSend struct {
	ID         uint64
	Phone      string
	TemplateID string
	TrackingID string
	Data       map[string]string
}

// NewSandboxSender constructs a sandbox sender. The logger receives the OTP
// code at Info level so a developer can read it from the terminal and complete
// the reset flow locally.
func NewSandboxSender(logger *slog.Logger) *SandboxSender {
	if logger == nil {
		logger = slog.Default()
	}
	return &SandboxSender{logger: logger}
}

func (p *SandboxSender) Send(_ context.Context, phone, templateID, trackingID string, data map[string]string) (SendResult, error) {
	id := p.counter.Add(1)
	cp := make(map[string]string, len(data))
	for k, v := range data {
		cp[k] = v
	}
	send := &SandboxSend{
		ID:         id,
		Phone:      phone,
		TemplateID: templateID,
		TrackingID: trackingID,
		Data:       cp,
	}
	p.mu.Lock()
	p.sends = append(p.sends, send)
	p.mu.Unlock()

	// Log the OTP prominently so the developer can see it in the terminal.
	otp := cp["otp"]
	p.logger.Info("📱 [ZALO SANDBOX] OTP captured (not sent to Zalo)",
		"id", id,
		"phone", phone,
		"template_id", templateID,
		"otp", otp,
	)
	fmt.Printf("\n   📱 [ZALO SANDBOX] OTP for %s: %s\n\n", phone, otp)

	return SendResult{
		MsgID:      fmt.Sprintf("sandbox-%d", id),
		ErrorCode:  0,
		ErrorMsg:   "Thành công (sandbox)",
		HTTPStatus: 200,
	}, nil
}

// LastSend returns the most recent captured send, or nil.
func (p *SandboxSender) LastSend() *SandboxSend {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.sends) == 0 {
		return nil
	}
	return p.sends[len(p.sends)-1]
}

// AllSends returns every captured send (for tests).
func (p *SandboxSender) AllSends() []*SandboxSend {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*SandboxSend, len(p.sends))
	copy(out, p.sends)
	return out
}

// Count returns the number of captured sends.
func (p *SandboxSender) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.sends)
}

// Reset clears all captured sends.
func (p *SandboxSender) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sends = nil
}
