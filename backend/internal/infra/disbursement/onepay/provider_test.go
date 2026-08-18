package onepay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain/ports/infrastructure"

	"github.com/gosimple/unidecode"
)

// ---------------------------------------------------------------------------
// Compile-time port assertions
// ---------------------------------------------------------------------------

func TestProvider_PortAssertions(t *testing.T) {
	c, err := NewClient(Config{
		PartnerID:  testPartnerID,
		PartnerKey: testPartnerKey,
		AccountID:  testAccountID,
	}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	p := NewProvider(c, nil)

	// Required interface
	var _ infrastructure.DisbursementProvider = p

	// Optional capabilities
	var dp infrastructure.DisbursementProvider = p
	if _, ok := dp.(infrastructure.AccountVerifier); !ok {
		t.Error("Provider must satisfy AccountVerifier")
	}
	if _, ok := dp.(infrastructure.BalanceReporter); !ok {
		t.Error("Provider must satisfy BalanceReporter")
	}
	if _, ok := dp.(infrastructure.ErrorTranslator); !ok {
		t.Error("Provider must satisfy ErrorTranslator")
	}
	if _, ok := dp.(infrastructure.StatusPoller); !ok {
		t.Error("Provider must satisfy StatusPoller")
	}
}

func TestProvider_Name(t *testing.T) {
	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
	}, nil)
	p := NewProvider(c, nil)
	if p.Name() != ProviderName {
		t.Errorf("Name() = %q, want %q", p.Name(), ProviderName)
	}
}

// ---------------------------------------------------------------------------
// InitiateTransfer
// ---------------------------------------------------------------------------

func validTransferRequest() infrastructure.TransferRequest {
	return infrastructure.TransferRequest{
		RequestID:   "FT-001",
		Amount:      200000,
		Description: "Salary May",
		BankCode:    "VCBVNVVX",
		SwiftCode:   "VCBVNVVX",
		AccountNo:   "1023020330000",
		AccountName: "NGUYEN VAN A",
	}
}

func TestProvider_InitiateTransfer_HappyPath(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(FundsTransferResponse{
			TransactionID:   "TXN-001",
			FundsTransferID: "FT-001",
			AccountID:       testAccountID,
			State:           "approved",
			ResponseCode:    "0",
			Message:         "Success",
		})
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	p := NewProvider(c, nil)

	res, err := p.InitiateTransfer(context.Background(), validTransferRequest())
	if err != nil {
		t.Fatalf("InitiateTransfer: %v", err)
	}
	if res.RequestID != "FT-001" {
		t.Errorf("RequestID = %q", res.RequestID)
	}
	if res.ProviderRef != "TXN-001" {
		t.Errorf("ProviderRef = %q", res.ProviderRef)
	}
	if res.Status != infrastructure.TransferStatusSuccess {
		t.Errorf("Status = %q, want success", res.Status)
	}
}

func TestProvider_InitiateTransfer_APIErrorBecomesFailedResult(t *testing.T) {
	// When the API returns a 500 with response_code, the provider must
	// return a TransferResult with Status=Failed, NOT a Go error.
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"response_code": "19",
			"name": "NO_BANK_PROCESS_FOUND",
			"message": "No bank found to process",
			"message_vi": "Không tìm thấy ngân hàng xử lý",
			"state": "failed"
		}`))
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	p := NewProvider(c, nil)

	res, err := p.InitiateTransfer(context.Background(), validTransferRequest())
	if err != nil {
		t.Fatalf("InitiateTransfer should not bubble API error: %v", err)
	}
	if res.Status != infrastructure.TransferStatusFailed {
		t.Errorf("Status = %q, want failed", res.Status)
	}
	if res.RawErrorCode != "19" {
		t.Errorf("RawErrorCode = %q, want 19", res.RawErrorCode)
	}
	if res.FailureReason == "" {
		t.Error("FailureReason should be populated")
	}
}

func TestProvider_InitiateTransfer_504BecomesPendingForInquiry(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = w.Write([]byte(`{
			"response_code": "01",
			"name": "TXN_PENDING",
			"message": "Txn is pending",
			"state": "pending"
		}`))
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	p := NewProvider(c, nil)

	res, err := p.InitiateTransfer(context.Background(), validTransferRequest())
	if err != nil {
		t.Fatalf("504 must become an inquiry-required pending result: %v", err)
	}
	if res.Status != infrastructure.TransferStatusPending {
		t.Fatalf("Status = %q, want pending", res.Status)
	}
	if res.RequestID != "FT-001" || res.RawErrorCode != "01" {
		t.Fatalf("pending result lost correlation: %+v", res)
	}
}

func TestProvider_InitiateTransfer_PostWriteTimeoutBecomesPendingForInquiry(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"state":"pending","response_code":"01"}`))
	}))
	defer srv.Close()

	c, err := NewClient(Config{
		PartnerID:   testPartnerID,
		PartnerKey:  testPartnerKey,
		AccountID:   testAccountID,
		Endpoint:    srv.URL,
		HTTPTimeout: 50 * time.Millisecond,
	}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.now = func() time.Time { return fixedTime }
	p := NewProvider(c, nil)

	res, err := p.InitiateTransfer(context.Background(), validTransferRequest())
	if err != nil {
		t.Fatalf("post-write timeout must become an inquiry-required pending result: %v", err)
	}
	if res.Status != infrastructure.TransferStatusPending || res.RequestID != "FT-001" {
		t.Fatalf("result = %+v, want pending FT-001", res)
	}
}

func TestProvider_InitiateTransfer_ValidationErrors(t *testing.T) {
	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
	}, nil)
	p := NewProvider(c, nil)

	tests := []struct {
		name string
		mut  func(*infrastructure.TransferRequest)
		want string
	}{
		{"empty RequestID", func(r *infrastructure.TransferRequest) { r.RequestID = "" }, "RequestID"},
		{"RequestID too long", func(r *infrastructure.TransferRequest) { r.RequestID = strings.Repeat("x", 21) }, "RequestID"},
		{"zero amount", func(r *infrastructure.TransferRequest) { r.Amount = 0 }, "Amount"},
		{"missing SwiftCode", func(r *infrastructure.TransferRequest) { r.SwiftCode = "" }, "SwiftCode"},
		{"missing AccountNo", func(r *infrastructure.TransferRequest) { r.AccountNo = "" }, "AccountNo"},
		{"missing AccountName", func(r *infrastructure.TransferRequest) { r.AccountName = "" }, "AccountName"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validTransferRequest()
			tc.mut(&req)
			_, err := p.InitiateTransfer(context.Background(), req)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestProvider_InitiateTransfer_DuplicateTxnRemainsPendingForInquiry(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"response_code": "07",
			"name": "DUPLICATE_TXN",
			"message": "Duplicate transaction",
			"state": "failed"
		}`))
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	p := NewProvider(c, nil)

	res, err := p.InitiateTransfer(context.Background(), validTransferRequest())
	if err != nil {
		t.Fatalf("should not bubble error for duplicate: %v", err)
	}
	if res.Status != infrastructure.TransferStatusPending {
		t.Errorf("Status = %q, want pending", res.Status)
	}
	if res.RawErrorCode != "07" {
		t.Errorf("RawErrorCode = %q, want 07", res.RawErrorCode)
	}
}

// ---------------------------------------------------------------------------
// CheckAccount
// ---------------------------------------------------------------------------

func validAccountCheckRequest() infrastructure.AccountCheckRequest {
	return infrastructure.AccountCheckRequest{
		RequestID:   "REQ-CHK-1",
		BankCode:    "VCBVNVVX",
		SwiftCode:   "VCBVNVVX",
		AccountNo:   "1023020330000",
		AccountName: "NGUYEN VAN A",
		Amount:      200000,
	}
}

func TestProvider_CheckAccount_HappyPath(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AccountInfoResponse{
			State:        "approved",
			HolderName:   "NGUYEN VAN A",
			ResponseCode: "00",
			Message:      "Success",
		})
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	p := NewProvider(c, nil)

	res, err := p.CheckAccount(context.Background(), validAccountCheckRequest())
	if err != nil {
		t.Fatalf("CheckAccount: %v", err)
	}
	if !res.Valid {
		t.Error("Valid = false, want true")
	}
	if res.AccountName != "NGUYEN VAN A" {
		t.Errorf("AccountName = %q, want NGUYEN VAN A", res.AccountName)
	}
}

func TestProvider_CheckAccount_RejectedAccount(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"response_code": "15",
			"name": "INVALID_ACCOUNT_INFO",
			"message": "Invalid account info",
			"state": "failed"
		}`))
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	p := NewProvider(c, nil)

	res, err := p.CheckAccount(context.Background(), validAccountCheckRequest())
	if err != nil {
		t.Fatalf("CheckAccount should not bubble error: %v", err)
	}
	if res.Valid {
		t.Error("Valid = true, want false")
	}
	if res.RawErrorCode != "15" {
		t.Errorf("RawErrorCode = %q", res.RawErrorCode)
	}
}

// ---------------------------------------------------------------------------
// GetBalance
// ---------------------------------------------------------------------------

func TestProvider_GetBalance_HappyPath(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(BalanceResponse{
			AccountID: testAccountID,
			Balance:   json.Number("150000000"),
		})
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	p := NewProvider(c, nil)

	res, err := p.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if res.Amount != 150000000 {
		t.Errorf("Amount = %d, want 150000000", res.Amount)
	}
	if res.Currency != "VND" {
		t.Errorf("Currency = %q", res.Currency)
	}
}

// ---------------------------------------------------------------------------
// TranslateError
// ---------------------------------------------------------------------------

func TestProvider_TranslateError(t *testing.T) {
	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
	}, nil)
	p := NewProvider(c, nil)

	got := p.TranslateError("07")
	if got != "Giao dịch bị trùng" {
		t.Errorf("TranslateError(07) = %q", got)
	}
	got = p.TranslateError("")
	if got != "" {
		t.Errorf("TranslateError(empty) = %q, want empty", got)
	}
}

// ---------------------------------------------------------------------------
// CheckStatus (StatusPoller)
// ---------------------------------------------------------------------------

func TestProvider_CheckStatus_HappyPath(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(InquiryResponse{
			TransactionID:   "TXN-001",
			FundsTransferID: "FT-001",
			State:           "approved",
			ResponseCode:    "0",
			BankTxnRef:      "BANK-REF-001",
			Message:         "Success",
		})
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	p := NewProvider(c, nil)

	res, err := p.CheckStatus(context.Background(), "FT-001")
	if err != nil {
		t.Fatalf("CheckStatus: %v", err)
	}
	if res.Status != infrastructure.TransferStatusSuccess {
		t.Errorf("Status = %q, want success", res.Status)
	}
	if res.ProviderRef != "TXN-001" {
		t.Errorf("ProviderRef = %q", res.ProviderRef)
	}
	if res.BankRef != "BANK-REF-001" {
		t.Errorf("BankRef = %q", res.BankRef)
	}
}

func TestProvider_CheckStatus_Failed(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(InquiryResponse{
			State:        "failed",
			ResponseCode: "19",
			Message:      "No bank found",
		})
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	p := NewProvider(c, nil)

	res, err := p.CheckStatus(context.Background(), "FT-FAIL")
	if err != nil {
		t.Fatalf("CheckStatus: %v", err)
	}
	if res.Status != infrastructure.TransferStatusFailed {
		t.Errorf("Status = %q, want failed", res.Status)
	}
}

func TestProvider_CheckStatus_NotFoundReturnsPortableSentinel(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{
			"response_code":"38",
			"name":"TXN_NOT_FOUND",
			"message":"Transaction not found",
			"state":"failed"
		}`))
	}))
	defer srv.Close()

	p := NewProvider(testClient(t, srv.URL, fixedTime), nil)
	_, err := p.CheckStatus(context.Background(), "FT-MISSING")
	if !errors.Is(err, infrastructure.ErrTransferNotFound) {
		t.Fatalf("CheckStatus error = %v, want ErrTransferNotFound", err)
	}
}

// ---------------------------------------------------------------------------
// translateState
// ---------------------------------------------------------------------------

func TestTranslateState(t *testing.T) {
	cases := []struct {
		state string
		want  infrastructure.TransferStatus
	}{
		{"created", infrastructure.TransferStatusPending},
		{"pending", infrastructure.TransferStatusPending},
		{"approved", infrastructure.TransferStatusSuccess},
		{"failed", infrastructure.TransferStatusFailed},
		{"reverted", infrastructure.TransferStatusReversed},
		{"unknown_state", infrastructure.TransferStatusUnknown},
	}
	for _, tc := range cases {
		t.Run("state="+tc.state, func(t *testing.T) {
			got := translateState(tc.state, "0")
			if got != tc.want {
				t.Errorf("translateState(%q) = %q, want %q", tc.state, got, tc.want)
			}
		})
	}
}

func TestTranslateState_AllWireStates(t *testing.T) {
	// Exhaustive check of every OnePay wire state from the spec.
	expected := map[string]infrastructure.TransferStatus{
		"created":  infrastructure.TransferStatusPending,
		"approved": infrastructure.TransferStatusSuccess,
		"failed":   infrastructure.TransferStatusFailed,
		"pending":  infrastructure.TransferStatusPending,
		"reverted": infrastructure.TransferStatusReversed,
	}
	for state, want := range expected {
		t.Run("wire/"+state, func(t *testing.T) {
			got := translateState(state, "0")
			if got != want {
				t.Errorf("translateState(%q) = %q, want %q", state, got, want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Provider name constant
// ---------------------------------------------------------------------------

func TestProviderName(t *testing.T) {
	if ProviderName != "1pay" {
		t.Errorf("ProviderName = %q, want 1pay", ProviderName)
	}
}

// ---------------------------------------------------------------------------
// VerifyAndParseWebhook delegation
// ---------------------------------------------------------------------------

func TestProvider_VerifyAndParseWebhook_Delegates(t *testing.T) {
	// This test validates that Provider.VerifyAndParseWebhook delegates
	// to parseAndVerifyWebhook. Full webhook tests are in webhook_test.go.
	// Here we just verify the delegation path doesn't panic on malformed input.
	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
	}, nil)
	p := NewProvider(c, nil)

	_, err := p.VerifyAndParseWebhook(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("expected error for empty payload")
	}
}

// ---------------------------------------------------------------------------
// InitiateTransfer maps request fields correctly
// ---------------------------------------------------------------------------

func TestProvider_InitiateTransfer_MapsRequestFields(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode body
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		_ = json.Unmarshal(body, &capturedBody)

		// Verify path contains account_id and funds_transfer_id
		expectedPath := fmt.Sprintf("/onepayout/api/v1/accounts/%s/funds_transfers/REQ-MAP-001", testAccountID)
		if r.URL.Path != expectedPath {
			t.Errorf("path = %q, want %q", r.URL.Path, expectedPath)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(FundsTransferResponse{
			TransactionID: "TXN-MAP",
			State:         "created",
			ResponseCode:  "0",
		})
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	p := NewProvider(c, nil)

	res, err := p.InitiateTransfer(context.Background(), infrastructure.TransferRequest{
		RequestID:   "REQ-MAP-001",
		Amount:      100000,
		BankCode:    "BIDVVNVX",
		SwiftCode:   "BIDVVNVX",
		AccountNo:   "1234567890",
		AccountName: "TRAN VAN B",
		Description: "Bonus Q1",
	})
	if err != nil {
		t.Fatalf("InitiateTransfer: %v", err)
	}
	if res.Status != infrastructure.TransferStatusPending {
		t.Errorf("Status = %q, want pending (created maps to pending)", res.Status)
	}

	// Verify the JSON body had correct field mapping
	if capturedBody["swift_code"] != "BIDVVNVX" {
		t.Errorf("swift_code = %v", capturedBody["swift_code"])
	}
	if capturedBody["account_number"] != "1234567890" {
		t.Errorf("account_number = %v", capturedBody["account_number"])
	}
	if capturedBody["holder_name"] != "TRAN VAN B" {
		t.Errorf("holder_name = %v", capturedBody["holder_name"])
	}
	if capturedBody["amount"] != "100000" {
		t.Errorf("amount = %v", capturedBody["amount"])
	}
	if capturedBody["currency"] != "VND" {
		t.Errorf("currency = %v", capturedBody["currency"])
	}
}

// ---------------------------------------------------------------------------
// preflightValidate
// ---------------------------------------------------------------------------

func TestPreflightValidate(t *testing.T) {
	tests := []struct {
		name        string
		swiftCode   string
		accountNo   string
		accountName string
		amount      int64
		wantNil     bool
		wantReason  string
	}{
		{"valid request", "VCBVNVVX", "1023020330000", "NGUYEN VAN A", 200000, true, ""},
		{"zero amount is OK", "VCBVNVVX", "1023020330000", "NGUYEN VAN A", 0, true, ""},
		{"empty swift code", "", "1023020330000", "NGUYEN VAN A", 200000, false, "missing_swift_code"},
		{"empty account no", "VCBVNVVX", "", "NGUYEN VAN A", 200000, false, "missing_account_no"},
		{"empty account name", "VCBVNVVX", "1023020330000", "", 200000, false, "missing_holder_name"},
		{"amount below minimum", "VCBVNVVX", "1023020330000", "NGUYEN VAN A", 99999, false, "amount_below_min"},
		{"amount at minimum", "VCBVNVVX", "1023020330000", "NGUYEN VAN A", 100000, true, ""},
		{"amount above maximum", "VCBVNVVX", "1023020330000", "NGUYEN VAN A", 20_000_001, false, "amount_above_max"},
		{"amount at maximum", "VCBVNVVX", "1023020330000", "NGUYEN VAN A", 20_000_000, true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := preflightValidate(tc.swiftCode, tc.accountNo, tc.accountName, tc.amount)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got reason=%q message=%q", got.Reason, got.Message)
				}
			} else {
				if got == nil {
					t.Fatalf("expected non-nil result with reason %q", tc.wantReason)
				}
				if got.Reason != tc.wantReason {
					t.Errorf("Reason = %q, want %q", got.Reason, tc.wantReason)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NormalizeNameForComparison (shared in infrastructure package)
// ---------------------------------------------------------------------------

func TestNormalizeNameForComparison(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Nguyễn Văn A", "NGUYEN VAN A"},
		{"NGUYEN VAN A", "NGUYEN VAN A"},
		{"nguyen van a", "NGUYEN VAN A"},
		{"  TRAN   THI  B  ", "TRAN THI B"},
		{"Trần Thị Hồng", "TRAN THI HONG"},
		{"Đoàn Minh Đức", "DOAN MINH DUC"},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := infrastructure.NormalizeNameForComparison(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeNameForComparison(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CheckAccount: name matching
// ---------------------------------------------------------------------------

func TestProvider_CheckAccount_NameMismatch(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AccountInfoResponse{
			State:        "approved",
			HolderName:   "TRAN THI B",
			ResponseCode: "00",
			Message:      "Success",
		})
	}))
	defer srv.Close()

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	c := testClient(t, srv.URL, fixedTime)
	p := NewProvider(c, logger)

	req := validAccountCheckRequest()
	req.AccountName = "NGUYEN VAN A"

	res, err := p.CheckAccount(context.Background(), req)
	if err != nil {
		t.Fatalf("CheckAccount: %v", err)
	}
	if res.Valid {
		t.Error("Valid = true, want false (name mismatch should block)")
	}
	if res.RawErrorCode != "name_mismatch" {
		t.Errorf("RawErrorCode = %q, want name_mismatch", res.RawErrorCode)
	}
	if res.AccountName != "TRAN THI B" {
		t.Errorf("AccountName = %q, want bank-confirmed name", res.AccountName)
	}
	if !strings.Contains(res.RawMessage, "NGUYEN VAN A") || !strings.Contains(res.RawMessage, "TRAN THI B") {
		t.Errorf("RawMessage should contain both names, got %q", res.RawMessage)
	}
	if strings.Contains(logs.String(), "NGUYEN VAN A") || strings.Contains(logs.String(), "TRAN THI B") {
		t.Fatalf("name mismatch logs leaked holder names: %s", logs.String())
	}
}

func TestProvider_CheckAccount_NameMatchWithDiacritics(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AccountInfoResponse{
			State:        "approved",
			HolderName:   "Tran Van Truong",
			ResponseCode: "00",
			Message:      "Success",
		})
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	p := NewProvider(c, nil)

	req := validAccountCheckRequest()
	req.AccountName = "Trần Văn Trường" // Stored Vietnamese name must match OnePay's ASCII holder name.

	res, err := p.CheckAccount(context.Background(), req)
	if err != nil {
		t.Fatalf("CheckAccount: %v", err)
	}
	if !res.Valid {
		t.Errorf("Valid = false, want true (diacritics should be normalized)")
	}
}

func TestProvider_CheckAccount_PreflightBlocksBeforeAPI(t *testing.T) {
	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
	}, nil)
	p := NewProvider(c, nil)

	tests := []struct {
		name string
		mut  func(*infrastructure.AccountCheckRequest)
		want string
	}{
		{"empty SwiftCode", func(r *infrastructure.AccountCheckRequest) { r.SwiftCode = "" }, "missing_swift_code"},
		{"empty AccountNo", func(r *infrastructure.AccountCheckRequest) { r.AccountNo = "" }, "missing_account_no"},
		{"empty AccountName", func(r *infrastructure.AccountCheckRequest) { r.AccountName = "" }, "missing_holder_name"},
		{"amount below min", func(r *infrastructure.AccountCheckRequest) { r.Amount = 50000 }, "amount_below_min"},
		{"amount above max", func(r *infrastructure.AccountCheckRequest) { r.Amount = 400_000_000 }, "amount_above_max"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validAccountCheckRequest()
			tc.mut(&req)
			res, err := p.CheckAccount(context.Background(), req)
			if err != nil {
				t.Fatalf("should not return error for preflight: %v", err)
			}
			if res.Valid {
				t.Error("Valid = true, want false")
			}
			if res.RawErrorCode != tc.want {
				t.Errorf("RawErrorCode = %q, want %q", res.RawErrorCode, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// validateTransferRequest: amount bounds
// ---------------------------------------------------------------------------

func TestValidateTransferRequest_AmountBounds(t *testing.T) {
	tests := []struct {
		name    string
		amount  int64
		wantErr bool
	}{
		{"below minimum", 99999, true},
		{"at minimum", 100000, false},
		{"at maximum", 20_000_000, false},
		{"above maximum", 20_000_001, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validTransferRequest()
			req.Amount = tc.amount
			err := validateTransferRequest(req)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestToASCII(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain ascii", "Nguyen Van A", "NGUYEN VAN A"},
		{"combining diacritics stripped", "Phàn Phủ Nghị", "PHAN PHU NGHI"},
		// Regression: Đ/đ (U+0110/U+0111) are precomposed — NFD + Mn-removal
		// leaves them intact. They MUST be converted to D/d, otherwise OnePay's
		// holder-name match fails with response_code 15 "Invalid account info".
		{"capital D-stroke", "Đỗ Duy Tuyên", "DO DUY TUYEN"},
		{"lowercase d-stroke", "đỗ duy tuyển", "DO DUY TUYEN"},
		{"d-stroke only", "Đ", "D"},
		{"leading d-stroke in surname", "Đặng Thị Huệ", "DANG THI HUE"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := toASCII(tc.in); got != tc.want {
				t.Errorf("toASCII(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestToASCII_MatchesUnidecode proves toASCII folds EVERY Vietnamese letter
// correctly, not just the hand-picked cases above. unidecode is a complete
// Unicode→ASCII table and is what the CheckAccount path already relies on
// (utils.NormalizeVietnamese). If toASCII agrees with it across the full
// Vietnamese alphabet + real names, toASCII is correct by construction — no
// hand-rolled NFD/Mn edge case can survive.
func TestToASCII_MatchesUnidecode(t *testing.T) {
	corpus := []string{
		// full Vietnamese tonal vowel inventory, lower + upper
		"á à ả ã ạ ă ắ ằ ẳ ẵ ặ â ấ ầ ẩ ẫ ậ",
		"Á À Ả Ã Ạ Ă Ắ Ằ Ẳ Ẵ Ặ Â Ấ Ầ Ẩ Ẫ Ậ",
		"é è ẻ ẽ ẹ ê ế ề ể ễ ệ", "É È Ẻ Ẽ Ẹ Ê Ế Ề Ể Ễ Ệ",
		"í ì ỉ ĩ ị", "Í Ì Ỉ Ĩ Ị",
		"ó ò ỏ õ ọ ô ố ồ ổ ỗ ộ ơ ớ ờ ở ỡ ợ",
		"Ó Ò Ỏ Õ Ọ Ô Ố Ồ Ổ Ỗ Ộ Ơ Ớ Ờ Ở Ỡ Ợ",
		"ú ù ủ ũ ụ ư ứ ừ ử ữ ự", "Ú Ù Ủ Ũ Ụ Ư Ứ Ừ Ử Ữ Ự",
		"ý ỳ ỷ ỹ ỵ", "Ý Ỳ Ỷ Ỹ Ỵ",
		// the precomposed D-stroke that NFD+unicode.Mn misses
		"Đ đ ĐHong đê",
		// real recipient names incl. Đ/đ surnames
		"Đỗ Duy Tuyên", "Đặng Thị Huệ", "Đào Văn Hùng", "Đinh Quốc Bảo",
		"Phạm Thị Mai", "Nguyễn Thị Thu Huyền", "Phàn Phủ Nghị",
		"Lê Hoàng Phúc", "Bùi Tường Lan", "Hồ Ngọc Hà", "Lò Văn Thủy",
	}
	for _, name := range corpus {
		want := strings.ToUpper(unidecode.Unidecode(name))
		got := toASCII(name)
		if got != want {
			t.Errorf("toASCII(%q) = %q; unidecode (known-good) = %q", name, got, want)
		}
	}
}
