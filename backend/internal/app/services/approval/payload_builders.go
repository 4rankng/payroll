package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// BuildFunc turns filer-supplied identifying parameters into a fully
// server-derived payload. It must never copy recipient or content fields from
// the input: everything user-visible is computed from cycle data.
type BuildFunc func(ctx context.Context, params json.RawMessage) (*BuiltPayload, error)

// PayloadBuilderRegistry routes a request type to its builder. Unknown types
// are rejected — the filer cannot invent types at the API boundary.
type PayloadBuilderRegistry struct {
	builders map[domain.ApprovalRequestType]BuildFunc
}

// NewPayloadBuilderRegistry wires the default per-type builders over the
// deterministic data sources.
func NewPayloadBuilderRegistry(cycle CycleReader, projectUsers ProjectUserDirectory, users UserDirectory, timesheets TimesheetReader) *PayloadBuilderRegistry {
	r := &PayloadBuilderRegistry{builders: make(map[domain.ApprovalRequestType]BuildFunc)}
	r.builders[domain.ApprovalTypeSendSaoKeEmail] = buildStatementEmailPayload(cycle, projectUsers, users, "sao-ke")
	r.builders[domain.ApprovalTypeSendPayrollEmail] = buildStatementEmailPayload(cycle, projectUsers, users, "bang-luong")
	r.builders[domain.ApprovalTypeSendReminder] = buildReminderPayload(projectUsers, users)
	r.builders[domain.ApprovalTypeVoidTimesheet] = buildVoidTimesheetPayload(timesheets)
	return r
}

// Build produces the server-derived payload for a request type.
func (r *PayloadBuilderRegistry) Build(ctx context.Context, requestType domain.ApprovalRequestType, params json.RawMessage) (*BuiltPayload, error) {
	builder, ok := r.builders[requestType]
	if !ok {
		return nil, domain.NewValidationError(fmt.Sprintf(
			"loại yêu cầu phê duyệt '%s' chưa được hỗ trợ", requestType))
	}
	return builder(ctx, params)
}

// deriveRecipients resolves the recipient list for a set of projects from
// cycle data: the users attached to each project (project_users), deduplicated
// and sorted so the payload is deterministic. Returns an error when no valid
// address exists — a request with nobody to reach must not be filed.
func deriveRecipients(ctx context.Context, projectUsers ProjectUserDirectory, users UserDirectory, projectIDs []uint) ([]string, error) {
	seen := map[uint]bool{}
	var userIDs []uint
	for _, projectID := range projectIDs {
		links, err := projectUsers.GetProjectUsers(ctx, projectID)
		if err != nil {
			return nil, domain.NewInternalError("không thể đọc danh sách người dùng của dự án", err)
		}
		for _, link := range links {
			if link == nil || link.UserID == 0 || seen[link.UserID] {
				continue
			}
			seen[link.UserID] = true
			userIDs = append(userIDs, link.UserID)
		}
	}
	if len(userIDs) == 0 {
		return nil, domain.NewValidationError(
			"không tìm thấy người nhận hợp lệ cho chu kỳ này (chưa gán người dùng vào dự án)")
	}

	byID, err := users.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, domain.NewInternalError("không thể đọc thông tin người nhận", err)
	}
	emails := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		u := byID[id]
		if u == nil || u.Email == nil {
			continue
		}
		email := strings.TrimSpace(*u.Email)
		if email == "" {
			continue
		}
		emails = append(emails, strings.ToLower(email))
	}
	if len(emails) == 0 {
		return nil, domain.NewValidationError(
			"người nhận của chu kỳ này chưa có địa chỉ email hợp lệ")
	}
	sort.Strings(emails)
	return emails, nil
}

// parseStatementDate validates the report date with the same rules the manual
// payroll-report endpoint uses (days 11–23 of the month are rejected).
func parseStatementDate(reportAtDate string) (time.Time, error) {
	req := dto.SendPayrollReportEmailRequest{ReportAtDate: strings.TrimSpace(reportAtDate)}
	at, err := req.ParseReportAtDate()
	if err != nil {
		return time.Time{}, domain.NewValidationError(err.Error())
	}
	return at, nil
}

// buildStatementEmailPayload builds payloads for the two statement-email
// types. The cycle key is identical for both types (the idempotency key still
// differs — the type is part of the key).
func buildStatementEmailPayload(cycle CycleReader, projectUsers ProjectUserDirectory, users UserDirectory, _ string) BuildFunc {
	return func(ctx context.Context, raw json.RawMessage) (*BuiltPayload, error) {
		var params struct {
			ReportAtDate string `json:"reportAtDate"`
		}
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		at, err := parseStatementDate(params.ReportAtDate)
		if err != nil {
			return nil, err
		}

		rows, err := cycle.GetProjectsForPayrollReport(ctx, at)
		if err != nil {
			return nil, domain.NewInternalError("không thể đọc dữ liệu kỳ báo cáo", err)
		}
		if len(rows) == 0 {
			return nil, domain.NewValidationError(
				"không có dữ liệu bảng công cho kỳ báo cáo này — không thể chuẩn bị yêu cầu")
		}

		projectIDs := make([]uint, 0, len(rows))
		timesheetCount := 0
		periodFrom, periodTo := rows[0].SalaryPeriodFrom, rows[0].SalaryPeriodTo
		for _, row := range rows {
			if row == nil || row.Project == nil {
				continue
			}
			projectIDs = append(projectIDs, row.Project.ID)
			timesheetCount += len(row.Timesheets)
			if row.SalaryPeriodFrom.Before(periodFrom) {
				periodFrom = row.SalaryPeriodFrom
			}
			if row.SalaryPeriodTo.After(periodTo) {
				periodTo = row.SalaryPeriodTo
			}
		}
		if len(projectIDs) == 0 {
			return nil, domain.NewValidationError(
				"không có dự án nào trong kỳ báo cáo này — không thể chuẩn bị yêu cầu")
		}

		recipients, err := deriveRecipients(ctx, projectUsers, users, projectIDs)
		if err != nil {
			return nil, err
		}

		dateStr := at.Format("2006-01-02")
		payload := SaoKeEmailPayload{
			ReportAtDate:   dateStr,
			PeriodFrom:     periodFrom.Format("2006-01-02"),
			PeriodTo:       periodTo.Format("2006-01-02"),
			ProjectIDs:     projectIDs,
			TimesheetCount: timesheetCount,
			Recipients:     recipients,
			Cc:             []string{},
		}
		return &BuiltPayload{
			Payload:  payload,
			CycleKey: "report_at_date=" + dateStr,
			Title:    "Gửi email cho kỳ báo cáo " + dateStr,
		}, nil
	}
}

// buildReminderPayload builds the send_reminder payload. Recipients are the
// project's attached users — the filer cannot address the reminder.
func buildReminderPayload(projectUsers ProjectUserDirectory, users UserDirectory) BuildFunc {
	return func(ctx context.Context, raw json.RawMessage) (*BuiltPayload, error) {
		var params ReminderParams
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		if params.ProjectID == 0 {
			return nil, domain.NewValidationError("mã dự án là bắt buộc")
		}
		if strings.TrimSpace(params.RemindForDate) == "" {
			return nil, domain.NewValidationError("ngày nhắc nhở là bắt buộc")
		}
		if _, err := time.ParseInLocation("2006-01-02", params.RemindForDate, clock.DefaultLocation); err != nil {
			return nil, domain.NewValidationError("định dạng ngày nhắc nhở không hợp lệ (YYYY-MM-DD)")
		}

		recipients, err := deriveRecipients(ctx, projectUsers, users, []uint{params.ProjectID})
		if err != nil {
			return nil, err
		}
		payload := ReminderPayload{
			ProjectID:     params.ProjectID,
			RemindForDate: params.RemindForDate,
			Recipients:    recipients,
		}
		return &BuiltPayload{
			Payload:  payload,
			CycleKey: fmt.Sprintf("project=%d|remind_for=%s", params.ProjectID, params.RemindForDate),
			Title:    "Gửi nhắc nhở cho dự án " + fmt.Sprint(params.ProjectID) + " ngày " + params.RemindForDate,
		}, nil
	}
}

// buildVoidTimesheetPayload snapshots the timesheet facts at filing time.
func buildVoidTimesheetPayload(timesheets TimesheetReader) BuildFunc {
	return func(ctx context.Context, raw json.RawMessage) (*BuiltPayload, error) {
		var params VoidTimesheetParams
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		if params.TimesheetID == 0 {
			return nil, domain.NewValidationError("mã bảng công là bắt buộc")
		}
		ts, err := timesheets.GetByID(ctx, params.TimesheetID)
		if err != nil {
			if domain.IsNotFoundError(err) {
				return nil, domain.NewNotFoundError("không tìm thấy bảng công")
			}
			return nil, err
		}
		payload := VoidTimesheetPayload{
			TimesheetID:    ts.ID,
			ProjectID:      ts.ProjectID,
			EmployeeID:     ts.EmployeeID,
			Date:           ts.Date.Format("2006-01-02"),
			StatusAtFiling: string(ts.Status),
			Reason:         strings.TrimSpace(params.Reason),
		}
		return &BuiltPayload{
			Payload:  payload,
			CycleKey: fmt.Sprintf("timesheet=%d", ts.ID),
			Title:    "Đề xuất huỷ bảng công " + fmt.Sprint(ts.ID),
		}, nil
	}
}

// decodeParams strictly decodes the per-type params and rejects malformed
// JSON with a Vietnamese validation error.
func decodeParams(raw json.RawMessage, dest any) error {
	if len(raw) == 0 {
		return domain.NewValidationError("tham số yêu cầu là bắt buộc")
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return domain.NewValidationError("tham số yêu cầu không hợp lệ")
	}
	return nil
}
