package onepay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	testPartnerID  = "TESTPARTNER"
	testPartnerKey = "test-partner-secret-key"
	testAccountID  = "ACC01"
)

// testClient creates a Client pointed at the given httptest.Server
// with deterministic time injection.
func testClient(t *testing.T, srvURL string, fixedTime time.Time) *Client {
	t.Helper()
	c, err := NewClient(Config{
		PartnerID:   testPartnerID,
		PartnerKey:  testPartnerKey,
		AccountID:   testAccountID,
		Endpoint:    srvURL,
		HTTPTimeout: 5 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.now = func() time.Time { return fixedTime }
	return c
}

// ---------------------------------------------------------------------------
// NewClient validation
// ---------------------------------------------------------------------------

func TestNewClient_RejectsBlankCredentials(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"no partner ID", Config{PartnerKey: "x", AccountID: "a"}},
		{"no partner key", Config{PartnerID: "x", AccountID: "a"}},
		{"no account ID", Config{PartnerID: "x", PartnerKey: "k"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewClient(tc.cfg, nil)
			if err == nil {
				t.Fatalf("expected error for incomplete config")
			}
		})
	}
}

func TestNewClient_Defaults(t *testing.T) {
	c, err := NewClient(Config{
		PartnerID:  "P",
		PartnerKey: "K",
		AccountID:  "A",
	}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.cfg.Endpoint != EndpointProduction {
		t.Errorf("default endpoint = %q, want %q", c.cfg.Endpoint, EndpointProduction)
	}
	if c.cfg.HTTPTimeout != 30*time.Second {
		t.Errorf("default timeout = %v, want 30s", c.cfg.HTTPTimeout)
	}
}

func TestNewClient_SetsSandboxEndpoint(t *testing.T) {
	c, err := NewClient(Config{
		PartnerID:  "P",
		PartnerKey: "K",
		AccountID:  "A",
		Endpoint:   "https://mtf.onepay.vn/",
	}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.cfg.Endpoint != "https://mtf.onepay.vn" {
		t.Errorf("trailing slash not stripped: %q", c.cfg.Endpoint)
	}
}

// ---------------------------------------------------------------------------
// GetAccountInfo
// ---------------------------------------------------------------------------

func TestClient_GetAccountInfo_SendsCorrectHeaders(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")
	var capturedHeaders http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()

		// Verify query params
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/onepayout/api/v1/customers" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("swift_code") != "VCBVNVVX" {
			t.Errorf("swift_code query = %q", r.URL.Query().Get("swift_code"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AccountInfoResponse{
			RequestID:     "REQ-001",
			SwiftCode:     "VCBVNVVX",
			BankName:      "Vietcombank",
			AccountNumber: "1023020330000",
			HolderName:    "NGUYEN VAN A",
			State:         "approved",
			ResponseCode:  "00",
			Message:       "Success",
		})
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	resp, err := c.GetAccountInfo(context.Background(), AccountInfoRequest{
		RequestID:     "REQ-001",
		SwiftCode:     "VCBVNVVX",
		AccountNumber: "1023020330000",
		Amount:        50000,
		AccountID:     testAccountID,
	})
	if err != nil {
		t.Fatalf("GetAccountInfo: %v", err)
	}

	// Verify required headers
	if capturedHeaders.Get("Accept") != "application/json" {
		t.Errorf("Accept = %q", capturedHeaders.Get("Accept"))
	}
	if capturedHeaders.Get("X-OP-Date") != "20260108T112907Z" {
		t.Errorf("X-OP-Date = %q", capturedHeaders.Get("X-OP-Date"))
	}
	if capturedHeaders.Get("X-OP-Expires") == "" {
		t.Error("X-OP-Expires should be sent (required by OnePay API)")
	}
	auth := capturedHeaders.Get("X-OP-Authorization")
	if !strings.HasPrefix(auth, "OWS1-HMAC-SHA256 ") {
		t.Errorf("X-OP-Authorization prefix = %q", auth[:30])
	}
	if !strings.Contains(auth, "Credential="+testPartnerID+"/") {
		t.Errorf("Authorization missing Credential with partner ID: %q", auth)
	}

	// Verify response
	if resp.State != "approved" {
		t.Errorf("State = %q", resp.State)
	}
	if resp.HolderName != "NGUYEN VAN A" {
		t.Errorf("HolderName = %q", resp.HolderName)
	}
}

// ---------------------------------------------------------------------------
// CreateFundsTransfer
// ---------------------------------------------------------------------------

func TestClient_CreateFundsTransfer_PUTWithBody(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		// Path must contain account_id and funds_transfer_id
		expectedPath := fmt.Sprintf("/onepayout/api/v1/accounts/%s/funds_transfers/FT-001", testAccountID)
		if r.URL.Path != expectedPath {
			t.Errorf("path = %q, want %q", r.URL.Path, expectedPath)
		}

		// Content-Type must be application/json for PUT
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}

		// Body should be valid JSON with our fields
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("body not valid JSON: %v", err)
		}
		if parsed["swift_code"] != "VCBVNVVX" {
			t.Errorf("swift_code = %v", parsed["swift_code"])
		}
		if parsed["amount"] != "50000" {
			t.Errorf("amount = %v", parsed["amount"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(FundsTransferResponse{
			TransactionID:   "TXN-001",
			FundsTransferID: "FT-001",
			AccountID:       testAccountID,
			AccountNumber:   "1023020330000",
			HolderName:      "NGUYEN VAN A",
			State:           "approved",
			Message:         "Success",
		})
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	resp, err := c.CreateFundsTransfer(context.Background(), FundsTransferRequest{
		AccountID:       testAccountID,
		FundsTransferID: "FT-001",
		SwiftCode:       "VCBVNVVX",
		AccountNumber:   "1023020330000",
		HolderName:      "NGUYEN VAN A",
		Amount:          "50000",
		Currency:        "VND",
	})
	if err != nil {
		t.Fatalf("CreateFundsTransfer: %v", err)
	}
	if resp.TransactionID != "TXN-001" {
		t.Errorf("TransactionID = %q", resp.TransactionID)
	}
	if resp.State != "approved" {
		t.Errorf("State = %q", resp.State)
	}
}

// ---------------------------------------------------------------------------
// InquiryFundsTransfer
// ---------------------------------------------------------------------------

func TestClient_InquiryFundsTransfer_GETParsesResponse(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(InquiryResponse{
			TransactionID:   "TXN-001",
			FundsTransferID: "FT-001",
			State:           "approved",
			BankTxnRef:      "BANK-REF-001",
			FundTime:        "20260108T112910Z",
		})
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	resp, err := c.InquiryFundsTransfer(context.Background(), "FT-001")
	if err != nil {
		t.Fatalf("InquiryFundsTransfer: %v", err)
	}
	if resp.State != "approved" {
		t.Errorf("State = %q", resp.State)
	}
	if resp.BankTxnRef != "BANK-REF-001" {
		t.Errorf("BankTxnRef = %q", resp.BankTxnRef)
	}
}

// ---------------------------------------------------------------------------
// GetBalance
// ---------------------------------------------------------------------------

func TestClient_GetBalance_ReturnsBalance(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(BalanceResponse{
			AccountID:   testAccountID,
			PartnerID:   testPartnerID,
			Balance:     json.Number("150000000"),
			PartnerName: "Test Partner",
			State:       "active",
		})
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	resp, err := c.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	bal, _ := resp.Balance.Int64()
	if bal != 150000000 {
		t.Errorf("Balance = %d, want 150000000", bal)
	}
}

// ---------------------------------------------------------------------------
// API error decoding
// ---------------------------------------------------------------------------

func TestClient_APIError_Decoding(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"response_code": "07",
			"name": "DUPLICATE_TXN",
			"message": "Duplicate transaction",
			"message_vi": "Giao dịch bị trùng",
			"state": "failed"
		}`))
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	_, err := c.CreateFundsTransfer(context.Background(), FundsTransferRequest{
		FundsTransferID: "FT-DUP",
		SwiftCode:       "VCBVNVVX",
		HolderName:      "TEST",
		Amount:          "50000",
		Currency:        "VND",
	})

	if err == nil {
		t.Fatal("expected error for 500 response")
	}

	var apiErr *APIError
	if !isAPIError(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.ResponseCode != "07" {
		t.Errorf("ResponseCode = %q", apiErr.ResponseCode)
	}
	if apiErr.Name != "DUPLICATE_TXN" {
		t.Errorf("Name = %q", apiErr.Name)
	}
	if apiErr.MessageVI != "Giao dịch bị trùng" {
		t.Errorf("MessageVI = %q", apiErr.MessageVI)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d", apiErr.StatusCode)
	}
}

func TestClient_NonJSON500(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("upstream error"))
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	_, err := c.GetBalance(context.Background())
	if err == nil {
		t.Fatal("expected error for non-JSON 500")
	}
	// Should be a plain error, not *APIError
	var apiErr *APIError
	if isAPIError(err, &apiErr) {
		t.Fatalf("should not be *APIError for non-JSON response")
	}
}

// ---------------------------------------------------------------------------
// Context cancellation / timeout
// ---------------------------------------------------------------------------

func TestClient_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until context cancelled or timeout
		time.Sleep(10 * time.Second)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c, err := NewClient(Config{
		PartnerID:   testPartnerID,
		PartnerKey:  testPartnerKey,
		AccountID:   testAccountID,
		Endpoint:    srv.URL,
		HTTPTimeout: 10 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = c.GetBalance(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("ctx.Err() = %v, want DeadlineExceeded", ctx.Err())
	}
}

// ---------------------------------------------------------------------------
// Signature determinism via time injection
// ---------------------------------------------------------------------------

func TestClient_DeterministicSignature(t *testing.T) {
	fixedTime := mustParseDate(t, "20260108T112907Z")
	var auth1, auth2 string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth1 == "" {
			auth1 = r.Header.Get("X-OP-Authorization")
		} else {
			auth2 = r.Header.Get("X-OP-Authorization")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(BalanceResponse{Balance: json.Number("0")})
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, fixedTime)
	_, _ = c.GetBalance(context.Background())
	_, _ = c.GetBalance(context.Background())

	if auth1 != auth2 {
		t.Fatalf("deterministic time must produce identical signatures:\n  %q\n  %q", auth1, auth2)
	}
}

// isAPIError checks if err wraps *APIError using errors.As pattern.
func isAPIError(err error, target **APIError) bool {
	return isAPIErrorAs(err, target)
}

// Using a separate function to avoid import of errors in test file.
func isAPIErrorAs(err error, target **APIError) bool {
	// errors.As equivalent — check if err is *APIError
	if e, ok := err.(*APIError); ok {
		*target = e
		return true
	}
	return false
}
