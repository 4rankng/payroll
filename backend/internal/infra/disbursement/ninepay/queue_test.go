package ninepay

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

func TestQueuedProvider_RateLimitsTransfers(t *testing.T) {
	const tps = 5
	var callCount atomic.Int32
	var firstCall, lastCall atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		now := time.Now().UnixMilli()
		lastCall.Store(now)
		if callCount.Add(1) == 1 {
			firstCall.Store(now)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(createDisbursementResponse{
			Status: 5, ErrorCode: "", Message: "ok", InvoiceNo: "123",
		})
	}))
	defer srv.Close()

	c, _ := NewClient(Config{
		MerchantKey: "M", SecretKey: "S", SecretKeyChecksum: "C",
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
				Amount:      5000,
				BankCode:    "VCB",
				AccountNo:   "123",
				AccountName: "TEST",
				AccountType: "0",
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
	minExpected := time.Duration(numRequests-2) * time.Second / time.Duration(tps)
	if elapsed < minExpected {
		t.Errorf("elapsed %v < expected min %v — rate limiting not working?", elapsed, minExpected)
	}
}

func TestQueuedProvider_DelegatesNonTransferMethods(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case pathCheckAccount:
			_ = json.NewEncoder(w).Encode(checkAccountResponse{
				Status: 5, ErrorCode: "", Message: "OK",
				BankCode: "VCB", AccountNo: "123", AccountName: "TEST",
			})
		case pathBalance:
			_ = json.NewEncoder(w).Encode(balanceResponse{
				Status: 5, ErrorCode: "", Message: "OK", Data: 999,
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	c, _ := NewClient(Config{
		MerchantKey: "M", SecretKey: "S", SecretKeyChecksum: "C",
		Endpoint: srv.URL, HTTPTimeout: 5 * time.Second,
	}, nil)
	p := NewProvider(c, nil)
	q := NewQueuedProvider(p, 3, 100, nil)
	defer q.Close() //nolint:errcheck

	if q.Name() != ProviderName {
		t.Errorf("Name = %q", q.Name())
	}

	acct, err := q.CheckAccount(context.Background(), infrastructure.AccountCheckRequest{
		RequestID: "Q-1", BankCode: "VCB", AccountNo: "123", AccountType: "0",
	})
	if err != nil {
		t.Fatalf("CheckAccount: %v", err)
	}
	if !acct.Valid {
		t.Error("expected valid account")
	}

	bal, err := q.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if bal.Amount != 999 {
		t.Errorf("Amount = %d", bal.Amount)
	}

	status, err := q.CheckStatus(context.Background(), "x")
	if err != nil {
		t.Fatalf("CheckStatus: %v", err)
	}
	if status.Status != infrastructure.TransferStatusUnknown {
		t.Errorf("Status = %q", status.Status)
	}
}

func TestQueuedProvider_QueueFullReturnsError(t *testing.T) {
	c, _ := NewClient(Config{
		MerchantKey: "M", SecretKey: "S", SecretKeyChecksum: "C",
		Endpoint:    "http://127.0.0.1:1", // connection refused — instant failure
		HTTPTimeout: 30 * time.Second,
	}, nil)
	p := NewProvider(c, nil)
	q := NewQueuedProvider(p, 1, 100, nil) // low TPS so run goroutine drains slowly
	defer func() { _ = q.Close() }()

	// Fill all 100 buffer slots directly — fast enough that the run goroutine
	// has only dequeued at most 1 (TPS=1, burst=1).
	for range 100 {
		ch := make(chan transferJobResult, 1)
		q.queue <- transferJob{ctx: context.Background(), result: ch}
	}

	// Queue is full (or nearly full). InitiateTransfer should fail immediately.
	_, err := q.InitiateTransfer(context.Background(), infrastructure.TransferRequest{
		RequestID: "OVERFLOW", Amount: 5000, BankCode: "VCB",
		AccountNo: "1", AccountName: "T", AccountType: "0",
	})
	if err == nil {
		t.Fatal("expected error when queue is full")
	}
}

func TestQueuedProvider_DefaultsTPS(t *testing.T) {
	c, _ := NewClient(Config{
		MerchantKey: "M", SecretKey: "S", SecretKeyChecksum: "C",
	}, nil)
	p := NewProvider(c, nil)
	q := NewQueuedProvider(p, 0, 100, nil)
	defer q.Close() //nolint:errcheck
}
