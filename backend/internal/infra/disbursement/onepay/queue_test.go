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
// Rate limiting at 3 TPS
// ---------------------------------------------------------------------------

func TestQueuedProvider_RateLimitsTransfers(t *testing.T) {
	const tps = 3
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
	q := NewQueuedProvider(p, tps, 100, nil)
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
	// With TPS=3, we expect ~333ms between calls. For numRequests-1 gaps,
	// minimum expected time is (numRequests-2) * (1s/tps) for burst=1.
	minExpected := time.Duration(numRequests-2) * time.Second / time.Duration(tps)
	if elapsed < minExpected {
		t.Errorf("elapsed %v < expected min %v — rate limiting not working?", elapsed, minExpected)
	}
}

// ---------------------------------------------------------------------------
// Queue full back-pressure
// ---------------------------------------------------------------------------

func TestQueuedProvider_QueueFullReturnsError(t *testing.T) {
	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
		Endpoint:    "http://127.0.0.1:1", // connection refused — instant failure
		HTTPTimeout: 30 * time.Second,
	}, nil)
	p := NewProvider(c, nil)
	q := NewQueuedProvider(p, 1, 100, nil) // low TPS so drain is slow
	defer func() { _ = q.Close() }()

	// Fill all 100 buffer slots directly
	for range 100 {
		ch := make(chan transferJobResult, 1)
		q.queue <- transferJob{ctx: context.Background(), result: ch}
	}

	// Queue is full. InitiateTransfer should fail immediately.
	_, err := q.InitiateTransfer(context.Background(), infrastructure.TransferRequest{
		RequestID: "OVERFLOW", Amount: 200000, BankCode: "VCBVNVVX",
		AccountNo: "1", AccountName: "T",
	})
	if err == nil {
		t.Fatal("expected error when queue is full")
	}
}

// ---------------------------------------------------------------------------
// Non-rate-limited methods bypass queue
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
	q := NewQueuedProvider(p, 1, 100, nil)
	defer q.Close() //nolint:errcheck

	// Name is trivial
	if q.Name() != ProviderName {
		t.Errorf("Name = %q", q.Name())
	}

	// CheckAccount bypasses the queue
	acct, err := q.CheckAccount(context.Background(), infrastructure.AccountCheckRequest{
		RequestID: "Q-1", BankCode: "VCBVNVVX", SwiftCode: "VCBVNVVX", AccountNo: "123", AccountName: "TEST USER",
	})
	if err != nil {
		t.Fatalf("CheckAccount: %v", err)
	}
	if !acct.Valid {
		t.Error("expected valid account")
	}

	// GetBalance bypasses the queue
	bal, err := q.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if bal.Amount != 999 {
		t.Errorf("Amount = %d", bal.Amount)
	}

	// CheckStatus bypasses the queue (OnePay has real polling)
	status, err := q.CheckStatus(context.Background(), "x")
	if err != nil {
		t.Fatalf("CheckStatus: %v", err)
	}
	if status.Status != infrastructure.TransferStatusSuccess {
		t.Errorf("Status = %q", status.Status)
	}

	// TranslateError bypasses queue
	msg := q.TranslateError("07")
	if msg != "Giao dịch bị trùng" {
		t.Errorf("TranslateError = %q", msg)
	}
}

// ---------------------------------------------------------------------------
// Default TPS=3 when 0 provided
// ---------------------------------------------------------------------------

func TestQueuedProvider_DefaultsTPS(t *testing.T) {
	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
	}, nil)
	p := NewProvider(c, nil)
	// TPS=0 should default to 3 without panicking
	q := NewQueuedProvider(p, 0, 100, nil)
	defer q.Close() //nolint:errcheck
}

// ---------------------------------------------------------------------------
// Shutdown drains pending
// ---------------------------------------------------------------------------

func TestQueuedProvider_CloseDrainsPending(t *testing.T) {
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
	q := NewQueuedProvider(p, 10, 100, nil)

	// Submit a transfer
	go func() {
		_, _ = q.InitiateTransfer(context.Background(), infrastructure.TransferRequest{
			RequestID: "DRAIN-1", Amount: 200000, BankCode: "VCBVNVVX",
			AccountNo: "1", AccountName: "T",
		})
	}()

	// Give it a moment to enqueue
	time.Sleep(100 * time.Millisecond)

	// Close should drain and not hang
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
		t.Fatal("Close hung — pending items not drained")
	}
}
