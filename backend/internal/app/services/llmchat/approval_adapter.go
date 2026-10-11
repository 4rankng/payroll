package llmchat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"api-server/internal/app/services/approval"
	"api-server/internal/domain"
)

// ApprovalEngine adapts the phase-2 approval engine to the chat ports
// (ApprovalFiler + ApprovalReader). The tool layer's per-role type allowlist
// still applies first; the engine re-checks role and type server-side.
type ApprovalEngine struct{ svc *approval.Service }

// NewApprovalEngine wires the chat ports to the approval engine.
func NewApprovalEngine(svc *approval.Service) *ApprovalEngine {
	return &ApprovalEngine{svc: svc}
}

func chatTypeToDomain(t string) (domain.ApprovalRequestType, bool) {
	switch t {
	case "send_sao_ke_email":
		return domain.ApprovalTypeSendSaoKeEmail, true
	case "send_payroll_email":
		return domain.ApprovalTypeSendPayrollEmail, true
	case "send_reminder":
		return domain.ApprovalTypeSendReminder, true
	case "void_timesheet":
		return domain.ApprovalTypeVoidTimesheet, true
	case "other":
		return domain.ApprovalTypeGeneralRequest, true
	}
	return "", false
}

// FileRequest maps a chat tool call onto approval.Service.File with the
// identifying params the payload builders require. "other" and unknown types
// are refused with Vietnamese copy — never silently mis-filed.
func (a *ApprovalEngine) FileRequest(ctx context.Context, req ApprovalFileRequest) (*ApprovalFileResult, error) {
	t, ok := chatTypeToDomain(req.Type)
	if !ok {
		return nil, fmt.Errorf("loại yêu cầu '%s' chưa hỗ trợ qua trợ lý — vui lòng mô tả lại hoặc liên hệ quản trị viên", req.Type)
	}
	params, err := buildFileParams(t, req)
	if err != nil {
		return nil, err
	}
	filed, err := a.svc.File(ctx, approval.CreateParams{
		Type:      t,
		Params:    params,
		ActorID:   req.CallerUserID,
		ActorRole: req.CallerRole,
		ActorName: req.CallerName,
	})
	if err != nil {
		return nil, err
	}
	return &ApprovalFileResult{RequestID: filed.ID, Status: string(filed.Status)}, nil
}

// buildFileParams derives the engine's identifying params per type. A missing
// required param is a user-facing error, not a guess.
func buildFileParams(t domain.ApprovalRequestType, req ApprovalFileRequest) (json.RawMessage, error) {
	switch t {
	case domain.ApprovalTypeGeneralRequest:
		if strings.TrimSpace(req.Summary) == "" {
			return nil, errors.New("cần mô tả nội dung yêu cầu")
		}
		return json.Marshal(map[string]any{"summary": req.Summary})
	case domain.ApprovalTypeVoidTimesheet:
		if req.TimesheetID == nil {
			return nil, errors.New("cần cung cấp ID timesheet (timesheet_id) để gửi yêu cầu void")
		}
		return json.Marshal(map[string]any{"timesheetId": *req.TimesheetID, "reason": req.Summary})
	case domain.ApprovalTypeSendSaoKeEmail, domain.ApprovalTypeSendPayrollEmail:
		if req.ReportAtDate == "" {
			return nil, errors.New("cần cung cấp ngày sao kê (report_at_date, định dạng YYYY-MM-DD)")
		}
		return json.Marshal(map[string]any{"reportAtDate": req.ReportAtDate})
	case domain.ApprovalTypeSendReminder:
		if req.ProjectID == nil || req.RemindForDate == "" {
			return nil, errors.New("cần cung cấp dự án (project_id) và ngày nhắc (remind_for_date)")
		}
		return json.Marshal(map[string]any{"projectId": *req.ProjectID, "remindForDate": req.RemindForDate, "reason": req.Summary})
	}
	return nil, fmt.Errorf("loại yêu cầu không hỗ trợ: %s", t)
}

// ListForUser narrates the caller's own requests for the status tool.
func (a *ApprovalEngine) ListForUser(ctx context.Context, caller Caller) ([]ApprovalRequestStatus, error) {
	digest, err := a.svc.ChatDigest(ctx, approval.Actor{ID: caller.UserID, Role: caller.Role, Name: caller.FullName})
	if err != nil {
		return nil, err
	}
	out := make([]ApprovalRequestStatus, 0, len(digest))
	for _, d := range digest {
		out = append(out, ApprovalRequestStatus{
			RequesterName: d.RequesterName,
			TypeLabel:     d.TypeLabel,
			Summary:       d.Summary,
			StatusLabel:   d.StatusLabel,
			RetryCount:    d.RetryCount,
			ScheduledAt:   d.ScheduledAt,
		})
	}
	return out, nil
}
