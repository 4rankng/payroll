package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"api-server/internal/domain"
)

// PreconditionCheck reports whether the data behind a request is complete
// enough to proceed. It MUST be deterministic — the LLM never judges data
// completeness. A non-nil error blocks the action and its message is shown to
// the reviewer / recorded on the retry.
type PreconditionCheck func(ctx context.Context, req *domain.ApprovalRequest) error

// PreconditionRegistry routes a request type to its deterministic check. The
// check runs at creation AND before every retry re-run.
type PreconditionRegistry struct {
	checks map[domain.ApprovalRequestType]PreconditionCheck
}

// NewPreconditionRegistry wires the default per-type checks over the
// deterministic data sources.
func NewPreconditionRegistry(cycle CycleReader, timesheets TimesheetReader) *PreconditionRegistry {
	r := &PreconditionRegistry{checks: make(map[domain.ApprovalRequestType]PreconditionCheck)}
	r.checks[domain.ApprovalTypeSendSaoKeEmail] = checkStatementCycle(cycle)
	r.checks[domain.ApprovalTypeSendPayrollEmail] = checkStatementCycle(cycle)
	r.checks[domain.ApprovalTypeSendReminder] = checkReminderPayload()
	r.checks[domain.ApprovalTypeVoidTimesheet] = checkVoidableTimesheet(timesheets)
	return r
}

// Evaluate runs the type's check. Types without a registered check pass — the
// engine must stay open to extension.
func (r *PreconditionRegistry) Evaluate(ctx context.Context, req *domain.ApprovalRequest) error {
	check, ok := r.checks[req.Type]
	if !ok {
		return nil
	}
	return check(ctx, req)
}

// checkStatementCycle enforces "all statements of the cycle generated": the
// cycle must have reportable rows and every timesheet feeding a statement must
// be in a final state — a pending_approval row would still change the
// statement, so the statements are not all generated yet.
func checkStatementCycle(cycle CycleReader) PreconditionCheck {
	return func(ctx context.Context, req *domain.ApprovalRequest) error {
		at, err := statementDateFromPayload(req.Payload)
		if err != nil {
			return err
		}
		rows, err := cycle.GetProjectsForPayrollReport(ctx, at)
		if err != nil {
			return domain.NewInternalError("không thể đọc dữ liệu kỳ báo cáo", err)
		}
		if len(rows) == 0 {
			return domain.NewValidationError(
				fmt.Sprintf("chưa đủ dữ liệu: kỳ báo cáo %s chưa có bảng công nào", at.Format("2006-01-02")))
		}

		pending := 0
		for _, row := range rows {
			if row == nil {
				continue
			}
			for _, ts := range row.Timesheets {
				if ts != nil && ts.Status == domain.TimesheetStatusPendingApproval {
					pending++
				}
			}
		}
		if pending > 0 {
			return domain.NewValidationError(fmt.Sprintf(
				"chưa đủ dữ liệu: còn %d bảng công chờ duyệt trong kỳ báo cáo %s — các báo cáo chưa được tạo đầy đủ",
				pending, at.Format("2006-01-02")))
		}
		return nil
	}
}

// statementDateFromPayload reads the report date out of either statement
// payload shape.
func statementDateFromPayload(payloadJSON string) (time.Time, error) {
	var payload struct {
		ReportAtDate string `json:"reportAtDate"`
	}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return time.Time{}, domain.NewValidationError("dữ liệu yêu cầu phê duyệt không hợp lệ")
	}
	return parseStatementDate(payload.ReportAtDate)
}

// checkReminderPayload verifies a reminder still has someone to reach.
func checkReminderPayload() PreconditionCheck {
	return func(_ context.Context, req *domain.ApprovalRequest) error {
		var payload ReminderPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return domain.NewValidationError("dữ liệu yêu cầu phê duyệt không hợp lệ")
		}
		if len(payload.Recipients) == 0 {
			return domain.NewValidationError("chưa đủ dữ liệu: không còn người nhận hợp lệ cho lời nhắc")
		}
		return nil
	}
}

// checkVoidableTimesheet verifies the timesheet still exists and has not been
// paid — a paid timesheet must not be voided through this flow.
func checkVoidableTimesheet(timesheets TimesheetReader) PreconditionCheck {
	return func(ctx context.Context, req *domain.ApprovalRequest) error {
		var payload VoidTimesheetPayload
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return domain.NewValidationError("dữ liệu yêu cầu phê duyệt không hợp lệ")
		}
		ts, err := timesheets.GetByID(ctx, payload.TimesheetID)
		if err != nil {
			if domain.IsNotFoundError(err) {
				return domain.NewValidationError(fmt.Sprintf(
					"chưa đủ dữ liệu: bảng công %d không còn tồn tại", payload.TimesheetID))
			}
			return err
		}
		if ts.IsPaid() {
			return domain.NewValidationError(fmt.Sprintf(
				"bảng công %d đã được thanh toán — không thể huỷ qua luồng phê duyệt", payload.TimesheetID))
		}
		return nil
	}
}
