package dto

// RejectUnpaidTimesheetsRequest scopes an Admin bulk rejection to one project
// and an inclusive ISO-8601 date range.
type RejectUnpaidTimesheetsRequest struct {
	ProjectID       uint   `json:"project_id" binding:"required"`
	FromDate        string `json:"from_date" binding:"required"`
	ToDate          string `json:"to_date" binding:"required"`
	RejectionReason string `json:"rejection_reason" binding:"required"`
}

// RejectUnpaidTimesheetsResponse reports the rows actually changed by the
// conditional database update.
type RejectUnpaidTimesheetsResponse struct {
	RejectedCount      int64    `json:"rejected_count"`
	Committed          bool     `json:"committed"`
	PostCommitComplete bool     `json:"post_commit_complete"`
	Warnings           []string `json:"warnings,omitempty"`
}
