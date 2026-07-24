package onepay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"api-server/internal/domain/ports/infrastructure"
)

// ---------------------------------------------------------------------------
// Per-endpoint rate limiting at the hardcoded onepayTPS (2/sec)
// ---------------------------------------------------------------------------

func TestQueuedProvider_RateLimitsTransfers(t *testing.T) {
	var callCount atomic.Int32
	var firstCall, lastCall atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		now := time.Now().UnixMilli()
		lastCall.Store(now)
		if callCount.Add(1) == 1 {
			firstCall.Store(now)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(FundsTransferResponse{
			TransactionID:   "TXN-001",
			FundsTransferID: "FT-001",
			State:           "approved",
			ResponseCode:    "0",
		})
	}))
	defer srv.Close()

	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
		Endpoint: srv.URL, HTTPTimeout: 5 * time.Second,
	}, nil)
	p := NewProvider(c, nil)
	// The tps/bufSize args are ignored; the limiters use the onepayTPS constant.
	q := NewQueuedProvider(p, 0, 0, nil)
	defer q.Close() //nolint:errcheck

	const numRequests = 4
	results := make(chan error, numRequests)
	for i := range numRequests {
		go func(i int) {
			_, err := q.InitiateTransfer(context.Background(), infrastructure.TransferRequest{
				RequestID:   "QTEST-" + string(rune('A'+i)),
				Amount:      200000,
				BankCode:    "VCBVNVVX",
				SwiftCode:   "VCBVNVVX",
				AccountNo:   "123",
				AccountName: "TEST USER",
			})
			results <- err
		}(i)
	}

	for i := range numRequests {
		if err := <-results; err != nil {
			t.Fatalf("transfer %d: %v", i, err)
		}
	}

	if got := callCount.Load(); got != numRequests {
		t.Fatalf("expected %d calls, got %d", numRequests, got)
	}

	elapsed := time.Duration(lastCall.Load()-firstCall.Load()) * time.Millisecond
	// With TPS=2 and burst=1, numRequests-1 gaps → min (numRequests-2)*(1s/tps).
	minExpected := time.Duration(numRequests-2) * time.Second / time.Duration(onepayTPS)
	if elapsed < minExpected {
		t.Errorf("elapsed %v < expected min %v — rate limiting not working?", elapsed, minExpected)
	}
}

// TestQueuedProvider_RateLimitsCheckAccount proves the account-verification
// endpoint (used heavily by employee imports) is throttled at onepayTPS.
func TestQueuedProvider_RateLimitsCheckAccount(t *testing.T) {
	var callCount atomic.Int32
	var firstCall, lastCall atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		now := time.Now().UnixMilli()
		lastCall.Store(now)
		if callCount.Add(1) == 1 {
			firstCall.Store(now)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AccountInfoResponse{
			State: "approved", HolderName: "TEST USER", ResponseCode: "00",
		})
	}))
	defer srv.Close()

	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
		Endpoint: srv.URL, HTTPTimeout: 5 * time.Second,
	}, nil)
	p := NewProvider(c, nil)
	q := NewQueuedProvider(p, 0, 0, nil)
	defer q.Close() //nolint:errcheck

	const numRequests = 4
	results := make(chan error, numRequests)
	for i := range numRequests {
		go func(i int) {
			_, err := q.CheckAccount(context.Background(), infrastructure.AccountCheckRequest{
				RequestID: "ACCT-Q-" + string(rune('A'+i)),
				BankCode:  "VCBVNVVX", SwiftCode: "VCBVNVVX",
				AccountNo: "123", AccountName: "TEST USER",
			})
			results <- err
		}(i)
	}

	for i := range numRequests {
		if err := <-results; err != nil {
			t.Fatalf("check %d: %v", i, err)
		}
	}

	if got := callCount.Load(); got != numRequests {
		t.Fatalf("expected %d calls, got %d", numRequests, got)
	}

	elapsed := time.Duration(lastCall.Load()-firstCall.Load()) * time.Millisecond
	minExpected := time.Duration(numRequests-2) * time.Second / time.Duration(onepayTPS)
	if elapsed < minExpected {
		t.Errorf("elapsed %v < expected min %v — CheckAccount not rate limited?", elapsed, minExpected)
	}
}

// TestQueuedProvider_PerEndpointIsolation proves that the four endpoints
// draw from INDEPENDENT token buckets: firing transfers and account checks
// concurrently should NOT make each wait for the other's tokens.
func TestQueuedProvider_PerEndpointIsolation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AccountInfoResponse{State: "approved", ResponseCode: "00"})
	}))
	defer srv.Close()

	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
		Endpoint: srv.URL, HTTPTimeout: 5 * time.Second,
	}, nil)
	p := NewProvider(c, nil)
	q := NewQueuedProvider(p, 0, 0, nil)
	defer q.Close() //nolint:errcheck

	// Fire 1 transfer + 1 check simultaneously. With independent buckets
	// both should complete near-instantly (each bucket has 1 token available).
	// If they shared a bucket, the second would wait ~500ms for the next token.
	start := time.Now()
	done := make(chan error, 2)
	go func() {
		_, err := q.InitiateTransfer(context.Background(), infrastructure.TransferRequest{
			RequestID: "ISO-T", Amount: 200000, BankCode: "VCBVNVVX",
			SwiftCode: "VCBVNVVX", AccountNo: "1", AccountName: "T",
		})
		done <- err
	}()
	go func() {
		_, err := q.CheckAccount(context.Background(), infrastructure.AccountCheckRequest{
			RequestID: "ISO-A", BankCode: "VCBVNVVX", SwiftCode: "VCBVNVVX",
			AccountNo: "1", AccountName: "T",
		})
		done <- err
	}()

	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if elapsed := time.Since(start); elapsed > 300*time.Millisecond {
		t.Errorf("elapsed %v — endpoints appear to share a limiter (should be isolated)", elapsed)
	}
}

// ---------------------------------------------------------------------------
// Cancelled context while waiting for a token → immediate error
// ---------------------------------------------------------------------------

func TestQueuedProvider_LimiterRespectsContextDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AccountInfoResponse{State: "approved", ResponseCode: "00"})
	}))
	defer srv.Close()

	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
		Endpoint: srv.URL, HTTPTimeout: 5 * time.Second,
	}, nil)
	p := NewProvider(c, nil)
	q := NewQueuedProvider(p, 0, 0, nil)
	defer q.Close() //nolint:errcheck

	// Consume the only token in the bucket.
	if _, err := q.CheckAccount(context.Background(), infrastructure.AccountCheckRequest{
		RequestID: "FIRST", BankCode: "VCBVNVVX", SwiftCode: "VCBVNVVX",
		AccountNo: "1", AccountName: "T",
	}); err != nil {
		t.Fatalf("first call: %v", err)
	}

	// A second call with a short deadline must fail fast rather than block
	// for the full token-regeneration window (~500ms at 2 TPS).
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := q.CheckAccount(ctx, infrastructure.AccountCheckRequest{
		RequestID: "SECOND", BankCode: "VCBVNVVX", SwiftCode: "VCBVNVVX",
		AccountNo: "1", AccountName: "T",
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error when context deadline expires while waiting for a token")
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("limiter did not honour ctx deadline: elapsed %v", elapsed)
	}
}

// ---------------------------------------------------------------------------
// Non-rate-limited methods
// ---------------------------------------------------------------------------

func TestQueuedProvider_DelegatesNonTransferMethods(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/onepayout/api/v1/customers":
			_ = json.NewEncoder(w).Encode(AccountInfoResponse{
				State: "approved", HolderName: "TEST USER", ResponseCode: "00",
			})
		case "/onepayout/api/v1/accounts/ACC01":
			_ = json.NewEncoder(w).Encode(BalanceResponse{
				Balance: json.Number("999"),
			})
		default:
			// Inquiry path
			_ = json.NewEncoder(w).Encode(InquiryResponse{
				State: "approved", ResponseCode: "0",
			})
		}
	}))
	defer srv.Close()

	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
		Endpoint: srv.URL, HTTPTimeout: 5 * time.Second,
	}, nil)
	p := NewProvider(c, nil)
	q := NewQueuedProvider(p, 0, 0, nil)
	defer q.Close() //nolint:errcheck

	// Name is trivial
	if q.Name() != ProviderName {
		t.Errorf("Name = %q", q.Name())
	}

	// CheckAccount is rate-limited but still delegates to the inner provider.
	acct, err := q.CheckAccount(context.Background(), infrastructure.AccountCheckRequest{
		RequestID: "Q-1", BankCode: "VCBVNVVX", SwiftCode: "VCBVNVVX", AccountNo: "123", AccountName: "TEST USER",
	})
	if err != nil {
		t.Fatalf("CheckAccount: %v", err)
	}
	if !acct.Valid {
		t.Error("expected valid account")
	}

	// GetBalance is rate-limited but still delegates.
	bal, err := q.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if bal.Amount != 999 {
		t.Errorf("Amount = %d", bal.Amount)
	}

	// CheckStatus is rate-limited but still delegates (OnePay has real polling).
	status, err := q.CheckStatus(context.Background(), "x")
	if err != nil {
		t.Fatalf("CheckStatus: %v", err)
	}
	if status.Status != infrastructure.TransferStatusSuccess {
		t.Errorf("Status = %q", status.Status)
	}

	// TranslateError is a pure local lookup — never hits OnePay.
	msg := q.TranslateError("07")
	if msg != "Giao dịch bị trùng" {
		t.Errorf("TranslateError = %q", msg)
	}
}

// ---------------------------------------------------------------------------
// Default constant applies regardless of constructor args (tps/bufSize ignored)
// ---------------------------------------------------------------------------

func TestQueuedProvider_IgnoresConstructorTPS(t *testing.T) {
	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
	}, nil)
	p := NewProvider(c, nil)
	// tps=0 and bufSize=0 should NOT panic and should use the onepayTPS constant.
	q := NewQueuedProvider(p, 0, 0, nil)
	defer q.Close() //nolint:errcheck
	if got := int(q.transferLimiter.Limit()); got != onepayTPS {
		t.Errorf("transfer limit = %d, want hardcoded onepayTPS = %d", got, onepayTPS)
	}
	if got := int(q.accountLimiter.Limit()); got != onepayTPS {
		t.Errorf("account limit = %d, want hardcoded onepayTPS = %d", got, onepayTPS)
	}
}

// ---------------------------------------------------------------------------
// Shutdown is a safe no-op with the limiter-based design
// ---------------------------------------------------------------------------

func TestQueuedProvider_CloseIsSafeNoOp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(FundsTransferResponse{
			State: "approved", ResponseCode: "0",
		})
	}))
	defer srv.Close()

	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
		Endpoint: srv.URL, HTTPTimeout: 5 * time.Second,
	}, nil)
	p := NewProvider(c, nil)
	q := NewQueuedProvider(p, 0, 0, nil)

	// A transfer submitted then Close should not hang.
	go func() {
		_, _ = q.InitiateTransfer(context.Background(), infrastructure.TransferRequest{
			RequestID: "CLOSE-1", Amount: 200000, BankCode: "VCBVNVVX",
			AccountNo: "1", AccountName: "T",
		})
	}()
	time.Sleep(100 * time.Millisecond)

	done := make(chan error, 1)
	go func() {
		done <- q.Close()
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close hung")
	}

	// Idempotent: a second Close must also be safe.
	if err := q.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
