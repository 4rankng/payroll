package approval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
)

// EmailExecuter hands approved sao kê / payroll email requests to the EXISTING
// async email pipeline (email:payroll_report). The worker is already
// idempotent per key, so the approval idempotency key doubles as the send
// guard: a double-approve or a retried send can never double-send.
type EmailExecuter struct {
	dispatch EmailDispatch
}

// NewEmailExecuter constructs the executer over the asynq email enqueue.
func NewEmailExecuter(dispatch EmailDispatch) *EmailExecuter {
	return &EmailExecuter{dispatch: dispatch}
}

// Execute enqueues the email and returns the human-readable outcome.
func (e *EmailExecuter) Execute(ctx context.Context, req *domain.ApprovalRequest) (string, error) {
	switch req.Type {
	case domain.ApprovalTypeSendSaoKeEmail, domain.ApprovalTypeSendPayrollEmail:
	default:
		return "", fmt.Errorf("loại yêu cầu không phải email: %s", req.Type)
	}
	var payload struct {
		ReportAtDate string   `json:"reportAtDate"`
		Recipients   []string `json:"recipients"`
		Cc           []string `json:"cc"`
		DraftSubject string   `json:"llmDraftSubject"`
		DraftBody    string   `json:"llmDraftBody"`
	}
	if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
		return "", fmt.Errorf("payload không hợp lệ: %w", err)
	}
	if len(payload.Recipients) == 0 {
		return "", errors.New("danh sách người nhận trống")
	}
	taskID, alreadySent, err := e.dispatch.EnqueuePayrollReportEmail(
		dto.SendPayrollReportEmailRequest{
			ReportAtDate: payload.ReportAtDate,
			Recipients:   payload.Recipients,
			Cc:           payload.Cc,
			// The reviewed draft is the sent content — empty falls back to the
			// worker template (draft never existed or provider failed).
			CustomSubject: payload.DraftSubject,
			CustomBody:    payload.DraftBody,
		},
		req.RequesterID,
		req.IdempotencyKey,
	)
	if err != nil {
		return "", err
	}
	if alreadySent {
		return "email đã được gửi trước đó theo khóa idempotency — không gửi lại", nil
	}
	return fmt.Sprintf("đã bàn giao cho hàng đợi gửi email (task %s)", taskID), nil
}
