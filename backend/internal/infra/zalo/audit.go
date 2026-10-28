package zalo

import (
	"context"
	"time"

	"api-server/internal/domain"
)

// Feature names for zns_messages rows — one per ZNS call source. Every Send
// caller passes its feature so admin tracking can group/filter by origin.
const (
	FeatureSalaryNotification = "salary_notification"
	FeaturePasswordReset      = "password_reset"
	FeatureIntegrationReset   = "integration_reset"
	FeatureSelfCheckin        = "self_checkin"
	FeatureTestSend           = "test_send"
)

// ZnsRecorder is the narrow audit seam used by Send implementations to append
// one zns_messages row per send attempt. domain.ZnsMessageRepository satisfies
// it, so bootstrap wires the repository straight in. Recording is best-effort:
// senders log recorder errors and never fail the send because of them.
type ZnsRecorder interface {
	Record(ctx context.Context, msg *domain.ZnsMessage) error
}

// recordSend persists one zns_messages row for a finished send attempt. It is
// safe with a nil recorder and never propagates errors — an audit hiccup must
// not break delivery. The write runs under a detached, short-lived context so a
// cancelled/expired send context cannot silently drop the row.
func recordSend(ctx context.Context, recorder ZnsRecorder, msg *domain.ZnsMessage) {
	if recorder == nil {
		return
	}
	recCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_ = recorder.Record(recCtx, msg)
}

// auditMessage builds the zns_messages row for one Send call. Status is "sent"
// only when Zalo accepted the message (no Go error, business code 0).
func auditMessage(feature, phone, templateID string, start time.Time, res SendResult, err error) *domain.ZnsMessage {
	msg := &domain.ZnsMessage{
		Feature:       feature,
		TemplateID:    templateID,
		Phone:         phone,
		Status:        domain.ZnsStatusSent,
		ProviderMsgID: res.MsgID,
		ProviderCode:  res.ErrorCode,
		DurationMs:    time.Since(start).Milliseconds(),
	}
	if err != nil {
		msg.Status = domain.ZnsStatusFailed
		msg.ErrorMsg = err.Error()
	} else if res.ErrorCode != 0 {
		msg.Status = domain.ZnsStatusFailed
		msg.ErrorMsg = res.ErrorMsg
	}
	return msg
}
