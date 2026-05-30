package ninepay

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type poller struct {
	db                *sql.DB
	ipnURL            string
	secretKeyChecksum string
	lineWrap          string
	processed         map[string]time.Time
	mu                sync.Mutex
	ipnCount          *atomic.Uint64
}

type authorisedRow struct {
	ID                 uint64
	RequestID          string
	InvoiceNo          string
	RequestedAmount    int64
	RecipientName      string
	RecipientAccountNo string
	RecipientBank      string
}

func newPoller(cfg Config, ipnCount *atomic.Uint64) (*poller, error) {
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
	return &poller{
		db:                db,
		ipnURL:            cfg.IPNURL,
		secretKeyChecksum: cfg.SecretKeyChecksum,
		lineWrap:          cfg.LineWrap,
		processed:         make(map[string]time.Time),
		ipnCount:          ipnCount,
	}, nil
}

const pollInterval = 10 * time.Second

func (p *poller) run(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.scan(ctx); err != nil {
				log.Printf("[9pay] poller: %v", err)
			}
		}
	}
}

func (p *poller) scan(ctx context.Context) error {
	p.evictProcessed()

	rows, err := p.db.QueryContext(ctx, `
		SELECT id, request_id, invoice_no, requested_amount,
		       recipient_name, recipient_account_no, recipient_bank
		FROM   wallet_payments
		WHERE  status = 'authorised'
		ORDER  BY created_at ASC
		LIMIT  50`)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var found []authorisedRow
	for rows.Next() {
		var r authorisedRow
		if err := rows.Scan(&r.ID, &r.RequestID, &r.InvoiceNo, &r.RequestedAmount,
			&r.RecipientName, &r.RecipientAccountNo, &r.RecipientBank); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		found = append(found, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}

	for _, r := range found {
		if p.isProcessed(r.RequestID) {
			continue
		}
		p.markProcessed(r.RequestID)

		paymentNo, _ := strconv.ParseInt(r.InvoiceNo, 10, 64)
		sendIPN(ipnPayload{
			AccountName:   r.RecipientName,
			AccountNo:     r.RecipientAccountNo,
			Amount:        r.RequestedAmount,
			AmountRequest: r.RequestedAmount,
			CardBrand:     r.RecipientBank,
			CreatedAt:     formatRealCreatedAt(time.Now()),
			Currency:      "VND",
			InvoiceNo:     r.RequestID,
			Method:        "DISBURSEMENT",
			PaymentNo:     paymentNo,
			RequestID:     r.RequestID,
			Status:        5,
			ErrorCode:     "000",
		}, p.ipnURL, p.secretKeyChecksum, p.lineWrap, p.ipnCount)
		log.Printf("[9pay] poller: sent IPN for authorised wallet_payment id=%d request_id=%s", r.ID, r.RequestID)
	}
	return nil
}

func (p *poller) isProcessed(requestID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.processed[requestID]
	return ok
}

func (p *poller) markProcessed(requestID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processed[requestID] = time.Now()
}

func (p *poller) evictProcessed() {
	p.mu.Lock()
	defer p.mu.Unlock()
	cutoff := time.Now().Add(-30 * time.Minute)
	for k, v := range p.processed {
		if v.Before(cutoff) {
			delete(p.processed, k)
		}
	}
}
