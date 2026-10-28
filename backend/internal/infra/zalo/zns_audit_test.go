package zalo

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"api-server/internal/domain"
)

// fakeZnsRecorder captures recorded rows in order. It satisfies ZnsRecorder.
type fakeZnsRecorder struct {
	rows []*domain.ZnsMessage
	err  error
}

func (f *fakeZnsRecorder) Record(_ context.Context, msg *domain.ZnsMessage) error {
	if f.err != nil {
		return f.err
	}
	f.rows = append(f.rows, msg)
	return nil
}

// TestSendRecordsOneRowPerFeature pins the PAYR-18 contract: sending from two
// different features produces exactly one zns_messages row each, tagged with
// the calling feature.
func TestSendRecordsOneRowPerFeature(t *testing.T) {
	p, _, _, _ := newProviderWithMock(t)
	rec := &fakeZnsRecorder{}
	p.SetZnsRecorder(rec)

	if _, err := p.Send(context.Background(), FeatureSalaryNotification, "0987654321", "617976", "track-1", map[string]string{"otp_code": "1"}); err != nil {
		t.Fatalf("Send salary: %v", err)
	}
	if _, err := p.Send(context.Background(), FeaturePasswordReset, "0912345678", "617976", "track-2", map[string]string{"otp_code": "2"}); err != nil {
		t.Fatalf("Send reset: %v", err)
	}

	if len(rec.rows) != 2 {
		t.Fatalf("recorded rows = %d, want 2", len(rec.rows))
	}
	if rec.rows[0].Feature != FeatureSalaryNotification {
		t.Errorf("row 0 feature = %q, want %q", rec.rows[0].Feature, FeatureSalaryNotification)
	}
	if rec.rows[1].Feature != FeaturePasswordReset {
		t.Errorf("row 1 feature = %q, want %q", rec.rows[1].Feature, FeaturePasswordReset)
	}
	for i, row := range rec.rows {
		if row.Status != domain.ZnsStatusSent {
			t.Errorf("row %d status = %q, want %q", i, row.Status, domain.ZnsStatusSent)
		}
		if row.ProviderMsgID == "" {
			t.Errorf("row %d missing provider_msg_id", i)
		}
	}
}

// TestSendFailureRecordsFailedRow pins that a transport failure lands as a
// status=failed row carrying the error message — the admin tracking page must
// show failed sends with their reason.
func TestSendFailureRecordsFailedRow(t *testing.T) {
	p, mock, _, _ := newProviderWithMock(t)
	rec := &fakeZnsRecorder{}
	p.SetZnsRecorder(rec)

	// Force a transport failure: the endpoint answers non-JSON garbage.
	mock.sendHandler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}

	_, err := p.Send(context.Background(), FeatureTestSend, "0987654321", "617976", "track-fail", map[string]string{"otp_code": "1"})
	if err == nil {
		t.Fatal("expected transport error from broken endpoint")
	}

	if len(rec.rows) != 1 {
		t.Fatalf("recorded rows = %d, want 1", len(rec.rows))
	}
	row := rec.rows[0]
	if row.Status != domain.ZnsStatusFailed {
		t.Errorf("status = %q, want %q", row.Status, domain.ZnsStatusFailed)
	}
	if row.ErrorMsg == "" {
		t.Error("failed row must carry error_msg")
	}
}

// TestSendBusinessErrorRecordsFailedRow pins that a Zalo business error
// (Go error nil, ErrorCode != 0) also lands as status=failed with error_msg.
func TestSendBusinessErrorRecordsFailedRow(t *testing.T) {
	p, mock, _, _ := newProviderWithMock(t)
	mock.setSendError(-118)

	rec := &fakeZnsRecorder{}
	p.SetZnsRecorder(rec)

	res, err := p.Send(context.Background(), FeatureSelfCheckin, "0987654321", "617976", "track-biz", map[string]string{"otp_code": "1"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if res.ErrorCode != -118 {
		t.Fatalf("ErrorCode = %d, want -118", res.ErrorCode)
	}

	if len(rec.rows) != 1 {
		t.Fatalf("recorded rows = %d, want 1", len(rec.rows))
	}
	row := rec.rows[0]
	if row.Status != domain.ZnsStatusFailed {
		t.Errorf("status = %q, want %q", row.Status, domain.ZnsStatusFailed)
	}
	if row.ErrorMsg == "" {
		t.Error("failed row must carry error_msg")
	}
	if row.ProviderCode != -118 {
		t.Errorf("provider_code = %d, want -118", row.ProviderCode)
	}
}

// TestRecordFailureDoesNotBreakSend pins best-effort behavior: a broken audit
// sink must never change the send outcome.
func TestRecordFailureDoesNotBreakSend(t *testing.T) {
	p, _, _, _ := newProviderWithMock(t)
	p.SetZnsRecorder(&fakeZnsRecorder{err: errors.New("db down")})

	res, err := p.Send(context.Background(), FeatureTestSend, "0987654321", "617976", "track-ok", map[string]string{"otp_code": "1"})
	if err != nil {
		t.Fatalf("Send must succeed despite recorder error: %v", err)
	}
	if res.ErrorCode != 0 {
		t.Errorf("ErrorCode = %d, want 0", res.ErrorCode)
	}
}

// TestAuditMessageStatusMapping pins the outcome mapping used by recordSend.
func TestAuditMessageStatusMapping(t *testing.T) {
	start := time.Now().Add(-5 * time.Millisecond)

	sent := auditMessage(FeatureTestSend, "0987654321", "617976", start, SendResult{MsgID: "m1", ErrorCode: 0}, nil)
	if sent.Status != domain.ZnsStatusSent || sent.ProviderMsgID != "m1" || sent.DurationMs < 0 {
		t.Errorf("sent mapping wrong: %+v", sent)
	}

	failed := auditMessage(FeatureTestSend, "0987654321", "617976", start, SendResult{}, errors.New("timeout"))
	if failed.Status != domain.ZnsStatusFailed || failed.ErrorMsg != "timeout" {
		t.Errorf("transport-failure mapping wrong: %+v", failed)
	}
}
