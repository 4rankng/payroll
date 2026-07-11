package onepay

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type recoverer struct {
	db        *sql.DB
	cfg       Config
	ipnSched  *ipnScheduler
	processed map[string]time.Time
	mu        sync.Mutex
}

func newRecoverer(cfg Config, ipnSched *ipnScheduler) (*recoverer, error) {
	db, err := sql.Open("mysql", cfg.DBDSN)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &recoverer{
		db:        db,
		cfg:       cfg,
		ipnSched:  ipnSched,
		processed: make(map[string]time.Time),
	}, nil
}

const recoverInterval = 10 * time.Second

func (r *recoverer) run(ctx context.Context) {
	ticker := time.NewTicker(recoverInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.scan(ctx); err != nil {
				log.Printf("[onepay] recoverer: %v", err)
			}
		}
	}
}

type staleRow struct {
	RequestID          string
	InvoiceNo          string
	RecipientAccountNo string
	RecipientBank      string
	RecipientName      string
	RequestedAmount    int64
}

func (r *recoverer) scan(ctx context.Context) error {
	r.evictProcessed()

	// wallet_payments.updated_at is written by the backend in Asia/Ho_Chi_Minh
	// (TZ=Asia/Ho_Chi_Minh in docker-compose), but this mock process and the
	// MySQL server both run in UTC. Using MySQL's NOW() or Go's time.Now()
	// (both UTC) would be 7 hours behind the stored UTC+7 timestamps, making
	// every row look like it's in the future. Compare in the same timezone
	// the app used when writing the value.
	vnTz, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	cutoff := time.Now().In(vnTz).Add(-30 * time.Second).Format("2006-01-02 15:04:05")

	rows, err := r.db.QueryContext(ctx, `
		SELECT request_id, invoice_no, recipient_account_no, recipient_bank, recipient_name, requested_amount
		FROM   wallet_payments
		WHERE  provider = '1pay' AND status = 'authorised' AND updated_at < ?
		LIMIT  50`, cutoff)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var found []staleRow
	for rows.Next() {
		var s staleRow
		if err := rows.Scan(&s.RequestID, &s.InvoiceNo, &s.RecipientAccountNo, &s.RecipientBank, &s.RecipientName, &s.RequestedAmount); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		found = append(found, s)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}

	for _, s := range found {
		if r.isProcessed(s.RequestID) {
			continue
		}
		r.markProcessed(s.RequestID)

		// Use the actual invoice_no (provider-issued transaction_id) as
		// TransactionID, matching what the transfer handler assigned. The
		// backend's RecordIPN looks up wallet_payments by invoice_no, so
		// sending request_id here would never match.
		txnID := s.InvoiceNo
		if txnID == "" {
			txnID = s.RequestID
		}

		now := time.Now()
		body := IPNBody{
			TransactionID: txnID, FundsTransferID: s.RequestID,
			AccountNumber: s.RecipientAccountNo, HolderName: s.RecipientName,
			Amount: s.RequestedAmount, Currency: "VND", State: "approved",
			ResponseCode: "00", Message: "SUCCESSFUL", SwiftCode: s.RecipientBank,
			AccountID: r.cfg.AccountID, Remark: s.RequestID,
			CreateTime: FormatOnePayTime(now.Add(-time.Minute)),
			UpdateTime: FormatOnePayTime(now),
		}
		r.ipnSched.deliverWithRetry(body)
		log.Printf("[onepay] recoverer: sent IPN for request_id=%s amount=%d", s.RequestID, s.RequestedAmount)
	}
	return nil
}

func (r *recoverer) isProcessed(requestID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.processed[requestID]
	return ok
}

func (r *recoverer) markProcessed(requestID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.processed[requestID] = time.Now()
}

func (r *recoverer) evictProcessed() {
	r.mu.Lock()
	defer r.mu.Unlock()
	cutoff := time.Now().Add(-30 * time.Minute)
	for k, v := range r.processed {
		if v.Before(cutoff) {
			delete(r.processed, k)
		}
	}
}
