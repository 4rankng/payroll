package dto

import (
	"fmt"
	"strings"
	"time"
)

// AttachmentData represents an email attachment in the DTO layer.
type AttachmentData struct {
	Filename    string
	ContentType string
	Content     []byte
}

// SendEmailRequest represents the payload for generic email sending.
type SendEmailRequest struct {
	From        string           `json:"from"`
	Recipients  []string         `json:"recipients"`
	Cc          []string         `json:"cc"`
	Bcc         []string         `json:"bcc"`
	Subject     string           `json:"subject"`
	HTMLBody    string           `json:"htmlBody"`
	TextBody    string           `json:"textBody"`
	ReplyTo     string           `json:"replyTo"` // default reply to frankng.sg@gmail.com
	Attachments []AttachmentData `json:"attachments"`
}

// Validate ensures the minimal requirements are satisfied.
func (r *SendEmailRequest) Validate() error {
	if len(r.Recipients) == 0 {
		return fmt.Errorf("recipients must not be empty")
	}
	if r.Subject == "" {
		return fmt.Errorf("subject must not be empty")
	}
	if r.HTMLBody == "" && r.TextBody == "" {
		return fmt.Errorf("either htmlBody or textBody must be provided")
	}
	return nil
}

// EmailRequest is an alias for SendEmailRequest to maintain compatibility with ports interface
type EmailRequest = SendEmailRequest

// SendPayrollReportEmailRequest contains parameters for sending the payroll report email.
type SendPayrollReportEmailRequest struct {
	ReportAtDate string   `json:"reportAtDate"`
	Recipients   []string `json:"recipients"`
	Cc           []string `json:"cc"`
	Bcc          []string `json:"bcc"`
}

// ParseReportAtDate validates and parses the requested report date using the same rules as the v3 report endpoint.
func (r *SendPayrollReportEmailRequest) ParseReportAtDate() (time.Time, error) {
	trimmed := strings.TrimSpace(r.ReportAtDate)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("reportAtDate is required")
	}

	req := PayrollReportByProjectRequest{AtDate: trimmed}
	if err := req.ValidateDate(); err != nil {
		return time.Time{}, err
	}

	return req.GetParsedDate()
}

// EmailRecipient represents an email recipient record in history responses.
type EmailRecipient struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address"`
}

// EmailHistoryEntry captures a single email delivery log entry.
type EmailHistoryEntry struct {
	ID             uint                 `json:"id"`
	Subject        string               `json:"subject"`
	Body           string               `json:"body"`
	MessageID      *string              `json:"messageId,omitempty"`
	Recipients     []EmailRecipient     `json:"recipients"`
	SenderID       uint                 `json:"senderId"`
	SenderName     string               `json:"senderName"`
	SenderUsername string               `json:"senderUsername"`
	Type           string               `json:"type"`
	Channel        string               `json:"channel"`
	SentAt         time.Time            `json:"sentAt"`
	SettledAt      *time.Time           `json:"settledAt,omitempty"`
	PayrollMeta    *PayrollEmailMetaDTO `json:"payrollMeta,omitempty"`
}

// PayrollEmailMetaDTO is the DTO representation of financial metadata for a payroll email.
type PayrollEmailMetaDTO struct {
	TotalAmount   int64    `json:"totalAmount"`
	FeePercentage float64  `json:"feePercentage"`
	FeeAmount     int64    `json:"feeAmount"`
	TotalWithFee  int64    `json:"totalWithFee"`
	ReportAtDate  string   `json:"reportAtDate"`
	TimesheetIDs  []uint   `json:"timesheetIds"`
	SaoKeAssetID  *uint    `json:"saoKeAssetId,omitempty"`
	CC            []string `json:"cc,omitempty"`
	BCC           []string `json:"bcc,omitempty"`
}

// SendEmailResponse is returned after a generic email is dispatched.
type SendEmailResponse struct {
	MessageID string `json:"message_id"`
}

// UploadSettlementResponse is returned after uploading a settlement file to an email history record.
type UploadSettlementResponse struct {
	ProcessedTimesheets int   `json:"processedTimesheets"`
	SkippedTimesheets   int   `json:"skippedTimesheets"`
	SettlementsCreated  int   `json:"settlementsCreated"`
	SettlementAmount    int64 `json:"settlementAmount"`
}
