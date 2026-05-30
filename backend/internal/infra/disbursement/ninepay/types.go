package ninepay

import "encoding/json"

// isSuccessCode reports whether a 9pay error_code indicates success.
// 9pay uses both empty string and "000" interchangeably across endpoints
// and response paths to mean "no error".
func isSuccessCode(s string) bool { return s == "" || s == "000" }

// Wire-level shapes for 9pay's disbursement API. Field tags use form
// names because the API is form-urlencoded, not JSON.
//
// The provider layer translates between these and the
// infrastructure.TransferRequest / TransferResult / WebhookEvent types
// so callers never see the 9pay-specific vocabulary.

// createDisbursementRequest is the body of POST /disbursement/create.
//
// Field constraints (from the docs):
//   - request_id: max 30 chars, must be unique per logical transfer
//   - amount: VND, range 2,000–2,000,000,000
//   - description: max 150 chars, alphanumeric only
//   - account_no: max 20 chars
//   - account_name: max 50 chars
//   - account_type: "0" (account) or "1" (card)
type createDisbursementRequest struct {
	RequestID   string
	Amount      int64
	Description string
	BankCode    string
	AccountNo   string
	AccountName string
	AccountType string
}

// toParams returns the body fields in the order the 9pay Postman
// collection declares them. Order is significant: it is the order the
// HMAC signing string consumes, and 9pay's pre-request script uses
// JS Object.keys (insertion order), not lexicographic sorting.
func (r createDisbursementRequest) toParams() []OrderedParam {
	return []OrderedParam{
		{Key: "request_id", Value: r.RequestID},
		{Key: "amount", Value: formatInt(r.Amount)},
		{Key: "description", Value: r.Description},
		{Key: "bank_code", Value: r.BankCode},
		{Key: "account_no", Value: r.AccountNo},
		{Key: "account_name", Value: r.AccountName},
		{Key: "account_type", Value: r.AccountType},
	}
}

// createDisbursementResponse is the JSON body 9pay returns on
// POST /disbursement/create. status, error_code, message are present
// on every response; the rest only on success.
type createDisbursementResponse struct {
	Status    int    `json:"status"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
	// InvoiceNo is what 9pay's wire format calls `payment_no`. The
	// JSON tag stays `payment_no` because that is what 9pay actually
	// sends; the Go field name follows our internal convention
	// (invoice_no). See domain/transactions/provider_transaction.go
	// for the full vocabulary explanation.
	InvoiceNo     json.Number `json:"payment_no"`
	Amount        string      `json:"amount"`
	Description   string      `json:"description"`
	BankCode      string      `json:"bank_code"`
	AccountName   string      `json:"account_name"`
	AccountNo     string      `json:"account_no"`
	CreatedAt     string      `json:"created_at"`
	FailureReason string      `json:"failure_reason,omitempty"`
	BankRef       *string     `json:"bank_ref,omitempty"`
}

// checkAccountRequest is the body of POST /disbursement/check-account
// ("Kiểm tra tài khoản nhận tiền"). Fields mirror the disbursement
// create payload but without amount/description.
type checkAccountRequest struct {
	RequestID   string
	BankCode    string
	AccountNo   string
	AccountName string
	AccountType string
}

// toParams returns the body fields in Postman-collection order. See
// createDisbursementRequest.toParams for why insertion order matters.
func (r checkAccountRequest) toParams() []OrderedParam {
	return []OrderedParam{
		{Key: "request_id", Value: r.RequestID},
		{Key: "bank_code", Value: r.BankCode},
		{Key: "account_no", Value: r.AccountNo},
		{Key: "account_name", Value: r.AccountName},
		{Key: "account_type", Value: r.AccountType},
	}
}

// checkAccountResponse mirrors the JSON returned by /disbursement/
// check-account. account_name is the bank-confirmed holder name and
// is only populated on a successful lookup.
type checkAccountResponse struct {
	Status      int    `json:"status"`
	ErrorCode   string `json:"error_code"`
	Message     string `json:"message"`
	RequestID   string `json:"request_id"`
	BankCode    string `json:"bank_code"`
	AccountNo   string `json:"account_no"`
	AccountName string `json:"account_name"`
	AccountType string `json:"account_type"`
}

// balanceResponse mirrors the JSON returned by GET /disbursement/balance
// ("Kiểm tra số dư của đối tác"). Data is the balance amount in VND.
type balanceResponse struct {
	Status    int    `json:"status"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
	Data      int64  `json:"data"`
}

// webhookResultBody is the decoded base64 payload that 9pay places in
// the IPN's `result` form field. It mirrors createDisbursementResponse
// plus a `method` discriminator the IPN uses to multiplex transaction
// types ("DISBURSEMENT" for our case).
type webhookResultBody struct {
	Status        int         `json:"status"`
	ErrorCode     string      `json:"error_code"`
	FailureReason string      `json:"failure_reason,omitempty"`
	Message       string      `json:"message"`
	Method        string      `json:"method"`
	InvoiceNo     json.Number `json:"payment_no"`
	Amount        int64       `json:"amount"`
	Description   string      `json:"description"`
	BankCode      string      `json:"bank_code"`
	AccountName   string      `json:"account_name"`
	AccountNo     string      `json:"account_no"`
	CreatedAt     string      `json:"created_at"`
	BankRef       *string     `json:"bank_ref,omitempty"`
	RequestID     string      `json:"request_id,omitempty"`
}
