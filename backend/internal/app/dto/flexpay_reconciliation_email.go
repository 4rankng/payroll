package dto

// SendFlexPayReconciliationEmailRequest contains parameters for sending the FlexPay reconciliation email.
type SendFlexPayReconciliationEmailRequest struct {
	ForMonth   string   `json:"forMonth" binding:"required"`
	Recipients []string `json:"recipients"`
	Cc         []string `json:"cc"`
	Bcc        []string `json:"bcc"`
}
