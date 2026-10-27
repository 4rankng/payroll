// Package approval is the HTTP transport for the approval-request engine:
// filing (identifying params only — payloads are server-derived), listing,
// and the three decisions (approve / reject-with-note / terminal cancel).
package approval

import (
	"encoding/json"

	"api-server/internal/app/services/approval"
	"time"

	"api-server/internal/domain"
)

// fileRequest is the ONLY client-supplied input for filing: a request type
// plus identifying params. Recipients and message content are never accepted.
type fileRequest struct {
	Type   string         `json:"type" binding:"required"`
	Params map[string]any `json:"params"`
}

// decisionRequest carries the optional decision note. On reject it is the
// retry guidance fed back into the next preparation run.
type decisionRequest struct {
	Note string `json:"note"`
}

// requestResponse renders one request for list/detail views.
type requestResponse struct {
	ID           uint           `json:"id"`
	Type         string         `json:"type"`
	TypeLabel    string         `json:"type_label"`
	Status       string         `json:"status"`
	StatusLabel  string         `json:"status_label"`
	Payload      map[string]any `json:"payload,omitempty"`
	DecisionNote *string        `json:"decision_note,omitempty"`
	Outcome      *string        `json:"outcome,omitempty"`
	RetryCount   int            `json:"retry_count"`
	ScheduledAt  *string        `json:"scheduled_at,omitempty"`
	RequesterID  uint           `json:"requester_id"`
	// Human-facing projection: requestor name, what the request is, and what
	// approving changes. The UI renders THESE, never the ids above.
	RequesterName string `json:"requester_name"`
	Summary       string `json:"summary"`
	ApproveEffect string `json:"approve_effect"`
	CreatedAt     string `json:"created_at"`
}

func newRequestResponseWithView(r *domain.ApprovalRequest, view approval.View) requestResponse {
	out := requestResponse{
		ID:            r.ID,
		Type:          string(r.Type),
		TypeLabel:     domain.ApprovalTypeLabelVN(r.Type),
		Status:        string(r.Status),
		StatusLabel:   r.Status.LabelVN(),
		DecisionNote:  r.DecisionNote,
		Outcome:       r.Outcome,
		RetryCount:    r.RetryCount,
		ScheduledAt:   nil,
		RequesterID:   r.RequesterID,
		RequesterName: view.RequesterName,
		Summary:       view.Summary,
		ApproveEffect: view.ApproveEffect,
		CreatedAt:     r.CreatedAt.Format(time.RFC3339),
	}
	if r.ScheduledAt != nil {
		s := r.ScheduledAt.Format(time.RFC3339)
		out.ScheduledAt = &s
	}
	var payload map[string]any
	if json.Unmarshal([]byte(r.Payload), &payload) == nil {
		out.Payload = payload
	}
	return out
}
