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
	RecipientAccountNo string
	RecipientBank      string
	RecipientName      string
	RequestedAmount    int64
}

func (r *recoverer) scan(ctx context.Context) error {
	r.evictProcessed()

	rows, err := r.db.QueryContext(ctx, `
		SELECT request_id, recipient_account_no, recipient_bank, recipient_name, requested_amount
		FROM   wallet_payments
		WHERE  provider = '1pay' AND status = 'authorised' AND updated_at < NOW() - INTERVAL 30 SECOND
		LIMIT  50`)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var found []staleRow
	for rows.Next() {
		var s staleRow
		if err := rows.Scan(&s.RequestID, &s.RecipientAccountNo, &s.RecipientBank, &s.RecipientName, &s.RequestedAmount); err != nil {
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

		now := time.Now()
		body := IPNBody{
			TransactionID: s.RequestID, FundsTransferID: s.RequestID,
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
