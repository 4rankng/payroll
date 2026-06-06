package onepay

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// AccountInfoRequest.toQuery()
// ---------------------------------------------------------------------------

func TestAccountInfoRequest_toQuery(t *testing.T) {
	req := AccountInfoRequest{
		RequestID:     "REQ-001",
		SwiftCode:     "VCBVNVVX",
		AccountNumber: "1023020330000",
		Amount:        50000,
		AccountID:     "ACC01",
	}
	v := req.toQuery()

	if v.Get("request_id") != "REQ-001" {
		t.Errorf("request_id = %q", v.Get("request_id"))
	}
	if v.Get("swift_code") != "VCBVNVVX" {
		t.Errorf("swift_code = %q", v.Get("swift_code"))
	}
	if v.Get("account_number") != "1023020330000" {
		t.Errorf("account_number = %q", v.Get("account_number"))
	}
	if v.Get("amount") != "50000" {
		t.Errorf("amount = %q", v.Get("amount"))
	}
	if v.Get("account_id") != "ACC01" {
		t.Errorf("account_id = %q", v.Get("account_id"))
	}
}

func TestAccountInfoRequest_toQuery_WithCardNumber(t *testing.T) {
	req := AccountInfoRequest{
		RequestID:  "REQ-002",
		SwiftCode:  "VCBVNVVX",
		CardNumber: "9704000000000018",
		Amount:     10000,
		AccountID:  "ACC01",
	}
	v := req.toQuery()

	if v.Get("card_number") != "9704000000000018" {
		t.Errorf("card_number = %q", v.Get("card_number"))
	}
	// AccountNumber should be absent when CardNumber is used
	if v.Get("account_number") != "" {
		t.Errorf("account_number should be empty when card_number is set")
	}
}

func TestAccountInfoRequest_toQuery_EncodesValues(t *testing.T) {
	req := AccountInfoRequest{
		RequestID: "REQ+special",
		SwiftCode: "VCBVNVVX",
		Amount:    100,
		AccountID: "ACC01",
	}
	v := req.toQuery()
	qs := v.Encode()
	// url.Values.Encode uses + for spaces; verify the value is encoded
	if !strings.Contains(qs, "request_id=") {
		t.Errorf("expected request_id in query: %q", qs)
	}
	// The raw value must survive round-trip
	parsed, _ := url.ParseQuery(qs)
	if parsed.Get("request_id") != "REQ+special" {
		t.Errorf("round-trip request_id = %q", parsed.Get("request_id"))
	}
}

// ---------------------------------------------------------------------------
// FundsTransferRequest JSON marshalling
// ---------------------------------------------------------------------------

func TestFundsTransferRequest_JSONMarshalling(t *testing.T) {
	req := FundsTransferRequest{
		AccountID:         "ACC01",
		FundsTransferID:   "FT001",
		SwiftCode:         "VCBVNVVX",
		AccountNumber:     "1023020330000",
		HolderName:        "NGUYEN VAN A",
		Amount:            "50000",
		Currency:          "VND",
		FundsTransferInfo: "REF-001",
		Remark:            "Salary May",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// Path params should NOT appear in JSON body
	if strings.Contains(string(data), `"account_id"`) {
		t.Error("AccountID (path param) must not appear in JSON body")
	}
	if strings.Contains(string(data), `"funds_transfer_id"`) {
		t.Error("FundsTransferID (path param) must not appear in JSON body")
	}

	// Body fields must be present
	if !strings.Contains(string(data), `"swift_code":"VCBVNVVX"`) {
		t.Errorf("swift_code missing from JSON: %s", data)
	}
	if !strings.Contains(string(data), `"amount":"50000"`) {
		t.Errorf("amount missing from JSON: %s", data)
	}
}

func TestFundsTransferRequest_OmitsEmptyFields(t *testing.T) {
	req := FundsTransferRequest{
		SwiftCode:  "VCBVNVVX",
		HolderName: "TEST",
		Amount:     "50000",
		Currency:   "VND",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// Optional fields with omitempty should be absent
	if strings.Contains(string(data), `"batch_id"`) {
		t.Error("batch_id should be omitted when empty")
	}
	if strings.Contains(string(data), `"funds_transfer_info"`) {
		t.Error("funds_transfer_info should be omitted when empty")
	}
	if strings.Contains(string(data), `"remark"`) {
		t.Error("remark should be omitted when empty")
	}
	if strings.Contains(string(data), `"meta_data"`) {
		t.Error("meta_data should be omitted when nil")
	}
	if strings.Contains(string(data), `"authentication"`) {
		t.Error("authentication should be omitted when nil")
	}
}

func TestFundsTransferRequest_WithMetaData(t *testing.T) {
	req := FundsTransferRequest{
		SwiftCode:  "VCBVNVVX",
		HolderName: "TEST",
		Amount:     "50000",
		Currency:   "VND",
		MetaData: &MetaData{
			CreatorID:   "user-001",
			CreatorName: "Admin",
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(data), `"creator_id":"user-001"`) {
		t.Errorf("meta_data.creator_id missing: %s", data)
	}
}

// ---------------------------------------------------------------------------
// IPNPayload JSON unmarshalling
// ---------------------------------------------------------------------------

func TestIPNPayload_Unmarshal_AllStates(t *testing.T) {
	states := []string{"created", "approved", "failed", "pending", "reverted"}
	for _, state := range states {
		t.Run("state="+state, func(t *testing.T) {
			body := `{"transaction_id":"TXN1","funds_transfer_id":"FT1","state":"` + state + `","response_code":"0","amount":50000,"currency":"VND"}`
			var ipn IPNPayload
			if err := json.Unmarshal([]byte(body), &ipn); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if ipn.State != state {
				t.Errorf("State = %q, want %q", ipn.State, state)
			}
			if ipn.TransactionID != "TXN1" {
				t.Errorf("TransactionID = %q", ipn.TransactionID)
			}
		})
	}
}

func TestIPNPayload_Unmarshal_ResponseCodeAsString(t *testing.T) {
	// response_code is always a string
	body := `{"response_code":"7","amount":100000}`
	var ipn IPNPayload
	if err := json.Unmarshal([]byte(body), &ipn); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if ipn.ResponseCode != "7" {
		t.Errorf("ResponseCode = %q, want 7", ipn.ResponseCode)
	}
}

// ---------------------------------------------------------------------------
// IPNAck serialization
// ---------------------------------------------------------------------------

func TestIPNAck_Success(t *testing.T) {
	ack := IPNAck{ErrorCode: "0", Message: "Success"}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"error_code":"0","message":"Success"}`
	if string(data) != want {
		t.Errorf("IPNAck = %q, want %q", data, want)
	}
}

// ---------------------------------------------------------------------------
// ErrorResponse unmarshalling
// ---------------------------------------------------------------------------

func TestErrorResponse_Unmarshal(t *testing.T) {
	body := `{"response_code":"07","name":"DUPLICATE_TXN","message":"Duplicate transaction","message_vi":"Giao dịch bị trùng","state":"failed"}`
	var errResp ErrorResponse
	if err := json.Unmarshal([]byte(body), &errResp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if errResp.ResponseCode != "07" {
		t.Errorf("ResponseCode = %q", errResp.ResponseCode)
	}
	if errResp.Name != "DUPLICATE_TXN" {
		t.Errorf("Name = %q", errResp.Name)
	}
	if errResp.MessageVI != "Giao dịch bị trùng" {
		t.Errorf("MessageVI = %q", errResp.MessageVI)
	}
	if errResp.State != "failed" {
		t.Errorf("State = %q", errResp.State)
	}
}

// ---------------------------------------------------------------------------
// BalanceResponse with json.Number parsing
// ---------------------------------------------------------------------------

func TestBalanceResponse_JSONNumber(t *testing.T) {
	body := `{"account_id":"ACC01","partner_id":"PARTNER01","balance":150000000,"partner_name":"Test","state":"active","create_time":"2026-01-01T00:00:00Z"}`
	var bal BalanceResponse
	if err := json.Unmarshal([]byte(body), &bal); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if bal.AccountID != "ACC01" {
		t.Errorf("AccountID = %q", bal.AccountID)
	}
	balInt, err := bal.Balance.Int64()
	if err != nil {
		t.Fatalf("Balance.Int64: %v", err)
	}
	if balInt != 150000000 {
		t.Errorf("Balance = %d, want 150000000", balInt)
	}
}

func TestBalanceResponse_LargeNumber(t *testing.T) {
	body := `{"balance":999999999999}`
	var bal BalanceResponse
	if err := json.Unmarshal([]byte(body), &bal); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	balInt, _ := bal.Balance.Int64()
	if balInt != 999999999999 {
		t.Errorf("Balance = %d, want 999999999999", balInt)
	}
}

// ---------------------------------------------------------------------------
// FundsTransferResponse
// ---------------------------------------------------------------------------

func TestFundsTransferResponse_Unmarshal(t *testing.T) {
	body := `{
		"transaction_id": "TXN-001",
		"funds_transfer_id": "FT-001",
		"account_id": "ACC01",
		"account_number": "1023020330000",
		"holder_name": "NGUYEN VAN A",
		"amount": 50000,
		"currency": "VND",
		"state": "approved",
		"response_code": "0",
		"message": "Success",
		"swift_code": "VCBVNVVX",
		"create_time": "20260108T112907Z",
		"update_time": "20260108T112910Z"
	}`
	var resp FundsTransferResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if resp.TransactionID != "TXN-001" {
		t.Errorf("TransactionID = %q", resp.TransactionID)
	}
	if resp.State != "approved" {
		t.Errorf("State = %q", resp.State)
	}
	amt, _ := resp.Amount.Int64()
	if amt != 50000 {
		t.Errorf("Amount = %d", amt)
	}
}

// ---------------------------------------------------------------------------
// InquiryResponse
// ---------------------------------------------------------------------------

func TestInquiryResponse_Unmarshal(t *testing.T) {
	body := `{
		"transaction_id": "TXN-001",
		"funds_transfer_id": "FT-001",
		"state": "approved",
		"response_code": "0",
		"bank_txn_ref": "BANK-REF-001",
		"fund_time": "20260108T112910Z"
	}`
	var resp InquiryResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if resp.BankTxnRef != "BANK-REF-001" {
		t.Errorf("BankTxnRef = %q", resp.BankTxnRef)
	}
	if resp.FundTime != "20260108T112910Z" {
		t.Errorf("FundTime = %q", resp.FundTime)
	}
}
