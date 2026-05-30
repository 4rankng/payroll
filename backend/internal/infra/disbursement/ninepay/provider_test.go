package ninepay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain/ports/infrastructure"
)

// parseMultipart reads the multipart body 9pay clients send and returns
// each field as a single-string map, mirroring the shape url.Values.Get
// returned in the form-urlencoded era.
//
// 9pay's body is multipart/form-data per the Postman collection. The
// boundary travels in Content-Type, and Go's net/http exposes parsed
// fields via Request.MultipartForm.Value after ParseMultipartForm.
func parseMultipart(t *testing.T, r *http.Request) map[string]string {
	t.Helper()
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		t.Fatalf("body not multipart/form-data: %v", err)
	}
	out := make(map[string]string, len(r.MultipartForm.Value))
	for k, v := range r.MultipartForm.Value {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}

func validRequest() infrastructure.TransferRequest {
	return infrastructure.TransferRequest{
		RequestID:   "REQ-12345",
		Amount:      50000,
		Description: "Salary May 2026",
		BankCode:    "VCB",
		AccountNo:   "1023020330000",
		AccountName: "NGUYEN VAN A",
		AccountType: "0",
	}
}

func TestProvider_NameAndPort(t *testing.T) {
	c, _ := NewClient(Config{
		MerchantKey: "M", SecretKey: "S", SecretKeyChecksum: "C",
	}, nil)
	p := NewProvider(c, nil)
	if p.Name() != ProviderName {
		t.Errorf("Name() = %q", p.Name())
	}
	var _ infrastructure.DisbursementProvider = p // compile-time
}

func TestProvider_ValidatesRequest(t *testing.T) {
	c, _ := NewClient(Config{
		MerchantKey: "M", SecretKey: "S", SecretKeyChecksum: "C",
		Endpoint: "http://does-not-resolve.invalid",
	}, nil)
	p := NewProvider(c, nil)

	tests := []struct {
		name  string
		mut   func(*infrastructure.TransferRequest)
		wantS string
	}{
		{name: "missing RequestID", mut: func(r *infrastructure.TransferRequest) { r.RequestID = "" }, wantS: "RequestID is required"},
		{name: "RequestID too long", mut: func(r *infrastructure.TransferRequest) { r.RequestID = strings.Repeat("x", 31) }, wantS: "exceeds 30 chars"},
		{name: "amount too low", mut: func(r *infrastructure.TransferRequest) { r.Amount = 1000 }, wantS: "out of allowed range"},
		{name: "amount too high", mut: func(r *infrastructure.TransferRequest) { r.Amount = 2_000_000_001 }, wantS: "out of allowed range"},
		{name: "missing BankCode", mut: func(r *infrastructure.TransferRequest) { r.BankCode = "" }, wantS: "BankCode is required"},
		{name: "missing AccountNo", mut: func(r *infrastructure.TransferRequest) { r.AccountNo = "" }, wantS: "AccountNo is required"},
		{name: "missing AccountName", mut: func(r *infrastructure.TransferRequest) { r.AccountName = "" }, wantS: "AccountName is required"},
		{name: "missing AccountType", mut: func(r *infrastructure.TransferRequest) { r.AccountType = "" }, wantS: "AccountType is required"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validRequest()
			tc.mut(&req)
			_, err := p.InitiateTransfer(context.Background(), req)
			if err == nil || !strings.Contains(err.Error(), tc.wantS) {
				t.Fatalf("err = %v, want substring %q", err, tc.wantS)
			}
		})
	}
}

func TestProvider_InitiateTransfer_HappyPath(t *testing.T) {
	const merchantKey, secretKey = "MK", "SK-secret"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != pathCreateDisbursement {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		// Verify Date header round-trips through the signature.
		date := r.Header.Get("Date")
		if _, err := strconv.ParseInt(date, 10, 64); err != nil {
			t.Errorf("Date header not unix timestamp: %q", date)
		}
		// Verify Authorization carries the merchant key and a non-empty
		// signature.
		auth := r.Header.Get("Authorization")
		if !strings.Contains(auth, "Credential="+merchantKey) {
			t.Errorf("Authorization missing Credential=%s: %q", merchantKey, auth)
		}

		// Body is multipart/form-data per the Postman collection.
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("Content-Type = %q, want multipart/form-data prefix", ct)
		}
		form := parseMultipart(t, r)

		// Rebuild the params slice in the same insertion order
		// createDisbursementRequest.toParams() emits, then recompute
		// the signature. Comparing the two proves the client's
		// canonical-form ordering matches what the server saw on
		// the wire — the most likely place a signing regression hides.
		req := validRequest()
		expectParams := createDisbursementRequest{
			RequestID:   req.RequestID,
			Amount:      req.Amount,
			Description: req.Description,
			BankCode:    req.BankCode,
			AccountNo:   req.AccountNo,
			AccountName: req.AccountName,
			AccountType: req.AccountType,
		}.toParams()
		// Cross-check that every form field on the wire round-trips:
		for _, p := range expectParams {
			if got := form[p.Key]; got != p.Value {
				t.Errorf("body field %q = %q, want %q", p.Key, got, p.Value)
			}
		}
		expectSig := Sign(http.MethodPost, "http://"+r.Host+pathCreateDisbursement, CanonicalizeParams(expectParams), date, []byte(secretKey))
		if !strings.Contains(auth, "Signature="+expectSig) {
			t.Errorf("signature mismatch:\n  got  %q\n  want signature suffix %q", auth, expectSig)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(createDisbursementResponse{
			Status: 5, ErrorCode: "", Message: "Success",
			InvoiceNo: "99887766", Amount: "50000",
			BankCode: "VCB", AccountName: "NGUYEN VAN A", AccountNo: "1023020330000",
		})
	}))
	defer srv.Close()

	c, err := NewClient(Config{
		MerchantKey:       merchantKey,
		SecretKey:         secretKey,
		SecretKeyChecksum: "ck",
		Endpoint:          srv.URL,
		HTTPTimeout:       5 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	p := NewProvider(c, nil)

	res, err := p.InitiateTransfer(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("InitiateTransfer: %v", err)
	}
	if res.Status != infrastructure.TransferStatusSuccess {
		t.Errorf("Status = %q", res.Status)
	}
	if res.ProviderRef != "99887766" {
		t.Errorf("ProviderRef = %q", res.ProviderRef)
	}
}

// TestProvider_InitiateTransfer_InvoiceNoZeroIsEmpty pins the
// normalizeInvoiceNo behavior: 9pay's synchronous /disbursement/create
// response carries the invoice_no field (wire-named payment_no) as the
// JSON number 0 before it has assigned a real provider reference. We
// must NOT persist "0" as the invoice_no — downstream syncPatch keys
// idempotency on it, and a shared-across-rows "0" would collide on
// the next debug call.
func TestProvider_InitiateTransfer_InvoiceNoZeroIsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Hand-rolled JSON so payment_no is a JSON number 0, not a
		// string — that's the wire shape 9pay actually sends.
		_, _ = w.Write([]byte(`{"status":5,"error_code":"000","message":"Success","payment_no":"0","amount":"50000","bank_code":"VCB","account_name":"NGUYEN VAN A","account_no":"1023020330000"}`))
	}))
	defer srv.Close()

	c, _ := NewClient(Config{
		MerchantKey: "M", SecretKey: "S", SecretKeyChecksum: "C",
		Endpoint: srv.URL, HTTPTimeout: 5 * time.Second,
	}, nil)
	p := NewProvider(c, nil)

	res, err := p.InitiateTransfer(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("InitiateTransfer: %v", err)
	}
	if res.ProviderRef != "" {
		t.Errorf("ProviderRef = %q, want empty (payment_no=0 must not be persisted)", res.ProviderRef)
	}
}

func TestNormalizeInvoiceNo(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"0", ""},
		{"485493625774881", "485493625774881"},
		{"99887766", "99887766"},
	}
	for _, c := range cases {
		if got := normalizeInvoiceNo(json.Number(c.in)); got != c.want {
			t.Errorf("normalizeInvoiceNo(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestProvider_InitiateTransfer_DuplicateRequestIDIsFailedNotError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(createDisbursementResponse{
			Status:        2,
			ErrorCode:     ErrorCodeDuplicateRequestID,
			Message:       "Duplicate request_id",
			FailureReason: "request_id already exists",
		})
	}))
	defer srv.Close()

	c, _ := NewClient(Config{
		MerchantKey: "M", SecretKey: "S", SecretKeyChecksum: "C",
		Endpoint: srv.URL, HTTPTimeout: 5 * time.Second,
	}, nil)
	p := NewProvider(c, nil)

	res, err := p.InitiateTransfer(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("InitiateTransfer should not return transport error for duplicate: %v", err)
	}
	if res.Status != infrastructure.TransferStatusFailed {
		t.Errorf("Status = %q, want failed", res.Status)
	}
	if res.RawErrorCode != ErrorCodeDuplicateRequestID {
		t.Errorf("RawErrorCode = %q, want %q", res.RawErrorCode, ErrorCodeDuplicateRequestID)
	}
}

func TestProvider_InitiateTransfer_NonOKHTTPReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("upstream error"))
	}))
	defer srv.Close()

	c, _ := NewClient(Config{
		MerchantKey: "M", SecretKey: "S", SecretKeyChecksum: "C",
		Endpoint: srv.URL, HTTPTimeout: 5 * time.Second,
	}, nil)
	p := NewProvider(c, nil)

	if _, err := p.InitiateTransfer(context.Background(), validRequest()); err == nil {
		t.Fatalf("expected error for HTTP 500, got nil")
	}
}

func TestProvider_CheckStatusReturnsUnknown(t *testing.T) {
	c, _ := NewClient(Config{
		MerchantKey: "M", SecretKey: "S", SecretKeyChecksum: "C",
	}, nil)
	p := NewProvider(c, nil)
	res, err := p.CheckStatus(context.Background(), "anything")
	if err != nil {
		t.Fatalf("CheckStatus: %v", err)
	}
	if res.Status != infrastructure.TransferStatusUnknown {
		t.Fatalf("Status = %q, want unknown — 9pay has no polling endpoint", res.Status)
	}
}

func validAccountCheckRequest() infrastructure.AccountCheckRequest {
	return infrastructure.AccountCheckRequest{
		RequestID:   "REQ-CHK-1",
		BankCode:    "9PAY",
		AccountNo:   "0888523111",
		AccountType: "0",
	}
}

func TestProvider_CheckAccount_HappyPath(t *testing.T) {
	const merchantKey, secretKey = "MK", "SK-secret"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != pathCheckAccount {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		date := r.Header.Get("Date")
		if _, err := strconv.ParseInt(date, 10, 64); err != nil {
			t.Errorf("Date header not unix timestamp: %q", date)
		}
		auth := r.Header.Get("Authorization")
		if !strings.Contains(auth, "Credential="+merchantKey) {
			t.Errorf("Authorization missing Credential=%s: %q", merchantKey, auth)
		}
		// Per Postman the SignedHeaders value is empty.
		if !strings.Contains(auth, "SignedHeaders=,Signature=") {
			t.Errorf("Authorization SignedHeaders should be empty (no spaces around commas): %q", auth)
		}

		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("Content-Type = %q, want multipart/form-data prefix", ct)
		}
		form := parseMultipart(t, r)
		expectParams := checkAccountRequest{
			RequestID:   "REQ-CHK-1",
			BankCode:    "9PAY",
			AccountNo:   "0888523111",
			AccountType: "0",
		}.toParams()
		for _, p := range expectParams {
			if got := form[p.Key]; got != p.Value {
				t.Errorf("body field %q = %q, want %q", p.Key, got, p.Value)
			}
		}
		expectSig := Sign(http.MethodPost, "http://"+r.Host+pathCheckAccount, CanonicalizeParams(expectParams), date, []byte(secretKey))
		if !strings.Contains(auth, "Signature="+expectSig) {
			t.Errorf("signature mismatch:\n  got  %q\n  want signature suffix %q", auth, expectSig)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(checkAccountResponse{
			Status: 5, ErrorCode: "", Message: "OK",
			BankCode: "9PAY", AccountNo: "0888523111",
			AccountName: "TEST USER", AccountType: "0",
		})
	}))
	defer srv.Close()

	c, _ := NewClient(Config{
		MerchantKey: merchantKey, SecretKey: secretKey, SecretKeyChecksum: "ck",
		Endpoint: srv.URL, HTTPTimeout: 5 * time.Second,
	}, nil)
	p := NewProvider(c, nil)

	res, err := p.CheckAccount(context.Background(), validAccountCheckRequest())
	if err != nil {
		t.Fatalf("CheckAccount: %v", err)
	}
	if !res.Valid {
		t.Errorf("Valid = false, want true")
	}
	if res.AccountName != "TEST USER" {
		t.Errorf("AccountName = %q", res.AccountName)
	}
}

func TestProvider_CheckAccount_RejectedAccountIsNotError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(checkAccountResponse{
			Status:    2,
			ErrorCode: "1004",
			Message:   "Account not found",
			BankCode:  "9PAY",
			AccountNo: "0000000000",
		})
	}))
	defer srv.Close()

	c, _ := NewClient(Config{
		MerchantKey: "M", SecretKey: "S", SecretKeyChecksum: "C",
		Endpoint: srv.URL, HTTPTimeout: 5 * time.Second,
	}, nil)
	p := NewProvider(c, nil)

	res, err := p.CheckAccount(context.Background(), validAccountCheckRequest())
	if err != nil {
		t.Fatalf("CheckAccount should not return transport error for rejected account: %v", err)
	}
	if res.Valid {
		t.Errorf("Valid = true, want false (rejected)")
	}
	if res.RawErrorCode != "1004" {
		t.Errorf("RawErrorCode = %q", res.RawErrorCode)
	}
}

func TestProvider_GetBalance_HappyPath(t *testing.T) {
	const merchantKey, secretKey = "MK", "SK-secret"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != pathBalance {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		// GET requests carry no body and no signed query — HTTPQUERY
		// in the signing string is empty.
		date := r.Header.Get("Date")
		auth := r.Header.Get("Authorization")
		expectSig := Sign(http.MethodGet, "http://"+r.Host+pathBalance, "", date, []byte(secretKey))
		if !strings.Contains(auth, "Signature="+expectSig) {
			t.Errorf("signature mismatch:\n  got  %q\n  want signature suffix %q", auth, expectSig)
		}
		if !strings.Contains(auth, "Credential="+merchantKey) {
			t.Errorf("Authorization missing Credential=%s: %q", merchantKey, auth)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(balanceResponse{
			Status: 5, ErrorCode: "", Message: "OK",
			Data: 1500000,
		})
	}))
	defer srv.Close()

	c, _ := NewClient(Config{
		MerchantKey: merchantKey, SecretKey: secretKey, SecretKeyChecksum: "ck",
		Endpoint: srv.URL, HTTPTimeout: 5 * time.Second,
	}, nil)
	p := NewProvider(c, nil)

	res, err := p.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if res.Amount != 1500000 {
		t.Errorf("Amount = %d, want 1500000", res.Amount)
	}
}

func TestProvider_ImplementsOptionalCapabilities(t *testing.T) {
	c, _ := NewClient(Config{MerchantKey: "M", SecretKey: "S", SecretKeyChecksum: "C"}, nil)
	p := NewProvider(c, nil)

	var dp infrastructure.DisbursementProvider = p
	if _, ok := dp.(infrastructure.AccountVerifier); !ok {
		t.Errorf("9pay Provider must satisfy AccountVerifier")
	}
	if _, ok := dp.(infrastructure.BalanceReporter); !ok {
		t.Errorf("9pay Provider must satisfy BalanceReporter")
	}
}

func TestNewClient_RequiresCredentials(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"no merchant key", Config{SecretKey: "x", SecretKeyChecksum: "y"}},
		{"no secret key", Config{MerchantKey: "x", SecretKeyChecksum: "y"}},
		{"no checksum key", Config{MerchantKey: "x", SecretKey: "y"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := NewClient(c.cfg, nil); err == nil {
				t.Fatalf("expected error for incomplete config %+v", c.cfg)
			}
		})
	}
}
