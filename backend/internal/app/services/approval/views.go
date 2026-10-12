package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"api-server/internal/domain"
)

// View is the human-facing projection of one request: who asked, what it is,
// and what approving changes. Raw IDs never appear in any of these strings —
// the UI and the chat tool narrate exactly this.
type View struct {
	RequesterName string
	Summary       string
	ApproveEffect string
}

type payloadForView struct {
	Ask            string   `json:"summary"`
	EmployeeName   string   `json:"employeeName"`
	ProjectName    string   `json:"projectName"`
	Position       string   `json:"position"`
	DayType        string   `json:"dayType"`
	Hours          float64  `json:"hours"`
	AmountVND      int64    `json:"amountVnd"`
	ReportAtDate   string   `json:"reportAtDate"`
	PeriodFrom     string   `json:"periodFrom"`
	PeriodTo       string   `json:"periodTo"`
	TimesheetCount int      `json:"timesheetCount"`
	Recipients     []string `json:"recipients"`
	Date           string   `json:"date"`
	Reason         string   `json:"reason"`
	ProjectID      *uint    `json:"projectId"`
	RemindForDate  string   `json:"remindForDate"`
}

// BuildView renders the projection for one request. requester may be nil
// (deleted user) — the name degrades to a stable label.
func BuildView(req *domain.ApprovalRequest, requester *domain.User) View {
	v := View{RequesterName: "người dùng đã xoá"}
	if requester != nil {
		v.RequesterName = requester.Fullname
		if strings.TrimSpace(v.RequesterName) == "" && requester.Username != "" {
			v.RequesterName = requester.Username
		}
	}
	var p payloadForView
	_ = json.Unmarshal([]byte(req.Payload), &p)

	switch req.Type {
	case domain.ApprovalTypeSendSaoKeEmail, domain.ApprovalTypeSendPayrollEmail:
		label := "email sao kê"
		if req.Type == domain.ApprovalTypeSendPayrollEmail {
			label = "email bảng lương"
		}
		v.Summary = fmt.Sprintf("Gửi %s cho kỳ báo cáo %s (%d bảng công, %d người nhận)",
			label, p.ReportAtDate, p.TimesheetCount, len(p.Recipients))
		v.ApproveEffect = fmt.Sprintf("Email với bản thảo đã duyệt sẽ được gửi đến: %s",
			strings.Join(p.Recipients, ", "))
	case domain.ApprovalTypeSendReminder:
		v.Summary = fmt.Sprintf("Gửi lời nhắc ngày %s đến %d người nhận", p.RemindForDate, len(p.Recipients))
		v.ApproveEffect = "Lời nhắc sẽ được gửi đến: " + strings.Join(p.Recipients, ", ")
	case domain.ApprovalTypeVoidTimesheet:
		reason := strings.TrimSpace(p.Reason)
		// Full human identification: who, which project, position, day type,
		// hours, amount — the reviewer must never guess which entry this is.
		details := fmt.Sprintf("%s · %s · %s · %s · %.2fg",
			p.EmployeeName, p.ProjectName, p.Position, p.DayType, p.Hours)
		if p.AmountVND > 0 {
			details += fmt.Sprintf(" · %s", formatAmountVND(p.AmountVND))
		}
		v.Summary = fmt.Sprintf("Huỷ bảng công ngày %s — %s", p.Date, details)
		if reason != "" {
			v.Summary += fmt.Sprintf(" — lý do: %s", reason)
		}
		v.ApproveEffect = "Quyết định huỷ bảng công được ghi nhận kèm lịch sử kiểm toán (chưa có bước thực thi tự động)"
	case domain.ApprovalTypeGeneralRequest:
		// The ask IS the summary — show what the person actually wants.
		ask := strings.TrimSpace(p.Ask)
		if ask == "" {
			ask = "(không có mô tả)"
		}
		v.Summary = ask
		v.ApproveEffect = "Quản trị viên xem và quyết định trực tiếp — phê duyệt ghi nhận quyết định, việc thực thi thủ công."
	default:
		v.Summary = domain.ApprovalTypeLabelVN(req.Type)
		v.ApproveEffect = "Thao tác được ghi nhận kèm lịch sử kiểm toán"
	}
	return v
}

// ChatStatus is one request as narrated by the get_approval_requests chat
// tool: labels and names only — no IDs, no enum strings.
type ChatStatus struct {
	RequesterName string
	TypeLabel     string
	Summary       string
	StatusLabel   string
	RetryCount    int
	ScheduledAt   string
}

// ChatDigest lists the caller's requests as chat-safe projections.
func (s *Service) ChatDigest(ctx context.Context, actor Actor) ([]ChatStatus, error) {
	items, _, err := s.List(ctx, actor, domain.ApprovalRequestFilters{Limit: 20})
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(items))
	for _, r := range items {
		ids = append(ids, r.RequesterID)
	}
	usersByID := map[uint]*domain.User{}
	if s.users != nil {
		usersByID, _ = s.users.GetByIDs(ctx, ids)
	}
	out := make([]ChatStatus, 0, len(items))
	for _, r := range items {
		view := BuildView(r, usersByID[r.RequesterID])
		status := ChatStatus{
			RequesterName: view.RequesterName,
			TypeLabel:     domain.ApprovalTypeLabelVN(r.Type),
			Summary:       view.Summary,
			StatusLabel:   r.Status.LabelVN(),
			RetryCount:    r.RetryCount,
		}
		if r.ScheduledAt != nil {
			status.ScheduledAt = r.ScheduledAt.Format("02/01/2006 15:04")
		}
		out = append(out, status)
	}
	return out, nil
}

// ViewsFor renders projections for a set of rows (handler merge).
func (s *Service) ViewsFor(ctx context.Context, items []*domain.ApprovalRequest) map[uint]View {
	ids := make([]uint, 0, len(items))
	for _, r := range items {
		ids = append(ids, r.RequesterID)
	}
	usersByID := map[uint]*domain.User{}
	if s.users != nil {
		usersByID, _ = s.users.GetByIDs(ctx, ids)
	}
	out := make(map[uint]View, len(items))
	for _, r := range items {
		out[r.ID] = BuildView(r, usersByID[r.RequesterID])
	}
	return out
}

// ViewFor renders the projection for a single row.
func (s *Service) ViewFor(ctx context.Context, r *domain.ApprovalRequest) View {
	return s.ViewsFor(ctx, []*domain.ApprovalRequest{r})[r.ID]
}
