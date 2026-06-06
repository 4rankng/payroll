package onepay

import (
	"encoding/json"
	"net/url"
	"strconv"
)

const ProviderName = "1pay"

// ---------------------------------------------------------------------------
// GET /onepayout/api/v1/customers — Account Info
// ---------------------------------------------------------------------------

// AccountInfoRequest mirrors PDF section 3.1.
type AccountInfoRequest struct {
	RequestID     string
	SwiftCode     string
	AccountNumber string
	CardNumber    string
	Amount        int64
	AccountID     string
}

func (r AccountInfoRequest) toQuery() url.Values {
	v := url.Values{}
	v.Set("request_id", r.RequestID)
	v.Set("swift_code", r.SwiftCode)
	if r.AccountNumber != "" {
		v.Set("account_number", r.AccountNumber)
	}
	if r.CardNumber != "" {
		v.Set("card_number", r.CardNumber)
	}
	v.Set("amount", strconv.FormatInt(r.Amount, 10))
	v.Set("account_id", r.AccountID)
	return v
}

// AccountInfoResponse is the response for account info lookup.
type AccountInfoResponse struct {
	RequestID     string `json:"request_id"`
	SwiftCode     string `json:"swift_code"`
	BankName      string `json:"bank_name"`
	AccountNumber string `json:"account_number"`
	CardNumber    string `json:"card_number"`
	HolderName    string `json:"holder_name"`
	State         string `json:"state"`
	ResponseCode  string `json:"response_code"`
	Message       string `json:"message"`
	MessageVI     string `json:"message_vi"`
}

// ---------------------------------------------------------------------------
// PUT /onepayout/api/v1/accounts/{account_id}/funds_transfers/{funds_transfer_id}
// ---------------------------------------------------------------------------

// FundsTransferRequest mirrors PDF section 3.2.
type FundsTransferRequest struct {
	AccountID         string     `json:"-"` // path param
	FundsTransferID   string     `json:"-"` // path param
	SwiftCode         string     `json:"swift_code"`
	BatchID           string     `json:"batch_id,omitempty"`
	AccountNumber     string     `json:"account_number,omitempty"`
	CardNumber        string     `json:"card_number,omitempty"`
	HolderName        string     `json:"holder_name"`
	Amount            string     `json:"amount"`
	Currency          string     `json:"currency"`
	FundsTransferInfo string     `json:"funds_transfer_info,omitempty"`
	Remark            string     `json:"remark,omitempty"`
	MetaData          *MetaData  `json:"meta_data,omitempty"`
	Authentication    *AuthBlock `json:"authentication,omitempty"`
}

// AuthBlock holds authentication credentials for funds transfer.
type AuthBlock struct {
	UserName string `json:"user_name"`
	Password string `json:"password"`
}

// MetaData holds creator/verifier metadata for funds transfer.
type MetaData struct {
	CreatorID    string `json:"creator_id,omitempty"`
	CreatorName  string `json:"creator_name,omitempty"`
	VerifierID   string `json:"verifier_id,omitempty"`
	VerifierName string `json:"verifier_name,omitempty"`
}

// FundsTransferResponse is the response for funds transfer creation.
type FundsTransferResponse struct {
	TransactionID     string      `json:"transaction_id"`
	FundsTransferID   string      `json:"funds_transfer_id"`
	FundsTransferInfo string      `json:"funds_transfer_info"`
	AccountID         string      `json:"account_id"`
	Remark            string      `json:"remark"`
	AccountNumber     string      `json:"account_number"`
	HolderName        string      `json:"holder_name"`
	Amount            json.Number `json:"amount"`
	Currency          string      `json:"currency"`
	State             string      `json:"state"`
	ResponseCode      string      `json:"response_code"`
	Message           string      `json:"message"`
	SwiftCode         string      `json:"swift_code"`
	CreateTime        string      `json:"create_time"`
	UpdateTime        string      `json:"update_time"`
	BatchID           string      `json:"batch_id"`
}

// ---------------------------------------------------------------------------
// IPN (PDF section 3.3)
// ---------------------------------------------------------------------------

// IPNPayload is the body of an IPN callback from OnePay.
type IPNPayload struct {
	TransactionID     string      `json:"transaction_id"`
	FundsTransferID   string      `json:"funds_transfer_id"`
	FundsTransferInfo string      `json:"funds_transfer_info"`
	AccountID         string      `json:"account_id"`
	Remark            string      `json:"remark"`
	AccountNumber     string      `json:"account_number"`
	HolderName        string      `json:"holder_name"`
	Amount            json.Number `json:"amount"`
	Currency          string      `json:"currency"`
	State             string      `json:"state"`
	ResponseCode      string      `json:"response_code"`
	Message           string      `json:"message"`
	SwiftCode         string      `json:"swift_code"`
	CreateTime        string      `json:"create_time"`
	UpdateTime        string      `json:"update_time"`
	BatchID           string      `json:"batch_id"`
}

// IPNAck is the response we POST back to acknowledge an IPN.
type IPNAck struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

// ---------------------------------------------------------------------------
// GET /onepayout/api/v1/accounts/{account_id}/funds_transfers/{funds_transfer_id}
// ---------------------------------------------------------------------------

// InquiryResponse is the response for querying a funds transfer status.
type InquiryResponse struct {
	TransactionID     string      `json:"transaction_id"`
	MerchantID        string      `json:"merchant_id"`
	FundsTransferID   string      `json:"funds_transfer_id"`
	FundsTransferInfo string      `json:"funds_transfer_info"`
	Remark            string      `json:"remark"`
	AccountID         string      `json:"account_id"`
	SwiftCode         string      `json:"swift_code"`
	ReceiptBankName   string      `json:"receipt_bank_name"`
	AccountNumber     string      `json:"account_number"`
	HolderName        string      `json:"holder_name"`
	Amount            json.Number `json:"amount"`
	Currency          string      `json:"currency"`
	State             string      `json:"state"`
	CreateTime        string      `json:"create_time"`
	ResponseCode      string      `json:"response_code"`
	Message           string      `json:"message"`
	SenderSwiftCode   string      `json:"sender_swift_code"`
	UpdateTime        string      `json:"update_time"`
	BankTxnRef        string      `json:"bank_txn_ref"`
	FundTime          string      `json:"fund_time"`
	MetaData          any         `json:"meta_data"`
	OperatorUser      string      `json:"operator_user"`
}

// ---------------------------------------------------------------------------
// GET /onepayout/api/v1/accounts/{account_id} — Balance
// ---------------------------------------------------------------------------

// BalanceResponse is the response for the balance inquiry endpoint.
type BalanceResponse struct {
	AccountID   string      `json:"account_id"`
	PartnerID   string      `json:"partner_id"`
	Balance     json.Number `json:"balance"`
	PartnerName string      `json:"partner_name"`
	State       string      `json:"state"`
	CreateTime  string      `json:"create_time"`
}

// ---------------------------------------------------------------------------
// Error envelope (5xx)
// ---------------------------------------------------------------------------

// ErrorResponse represents a 5xx error response from OnePay.
type ErrorResponse struct {
	ResponseCode string `json:"response_code"`
	Name         string `json:"name"`
	Message      string `json:"message"`
	MessageVI    string `json:"message_vi"`
	State        string `json:"state"`
}
