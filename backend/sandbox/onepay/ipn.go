package onepay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type IPNBody struct {
	TransactionID     string `json:"transaction_id"`
	FundsTransferID   string `json:"funds_transfer_id"`
	FundsTransferInfo string `json:"funds_transfer_info"`
	AccountID         string `json:"account_id"`
	Remark            string `json:"remark"`
	AccountNumber     string `json:"account_number"`
	HolderName        string `json:"holder_name"`
	Amount            int64  `json:"amount"`
	Currency          string `json:"currency"`
	State             string `json:"state"`
	ResponseCode      string `json:"response_code"`
	Message           string `json:"message"`
	SwiftCode         string `json:"swift_code"`
	CreateTime        string `json:"create_time"`
	UpdateTime        string `json:"update_time"`
	BatchID           string `json:"batch_id,omitempty"`
}

type ipnScheduler struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.Mutex
	pending    map[string]bool
}

func newIPNScheduler(cfg Config) *ipnScheduler {
	return &ipnScheduler{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		pending: make(map[string]bool),
	}
}

func (s *ipnScheduler) scheduleIPN(body IPNBody, delay time.Duration) {
	go func() {
		time.Sleep(delay)
		s.deliverWithRetry(body)
	}()
}

func (s *ipnScheduler) deliverWithRetry(body IPNBody) {
	const maxRetries = 3
	retryDelay := s.cfg.IPNRetryDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[onepay] ipn: retry %d/%d for funds_transfer_id=%s after %s",
				attempt, maxRetries, body.FundsTransferID, retryDelay)
			time.Sleep(retryDelay)
		}
		if s.deliver(body) {
			return
		}
	}
	log.Printf("[onepay] ipn: exhausted retries for funds_transfer_id=%s", body.FundsTransferID)
}

func (s *ipnScheduler) deliver(body IPNBody) bool {
	payload, err := json.Marshal(body)
	if err != nil {
		log.Printf("[onepay] ipn: marshal: %v", err)
		return false
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPut,
		s.cfg.IPNURL,
		bytes.NewReader(payload),
	)
	if err != nil {
		log.Printf("[onepay] ipn: build request: %v", err)
		return false
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(payload)))

	now := time.Now().UTC()
	// OnePay's production IPN signer uses their internal gateway URL as
	// the canonical URI, not the public callback URL.
	const onepayIPNInternalURL = "http://localhost/payout-merchants/https/tingting.vip/443/api/v1/webhooks/disbursement/1pay"
	opDate, opExpires, opAuth := signOutboundOWS(
		http.MethodPut, onepayIPNInternalURL, payload,
		s.cfg.PartnerID, s.cfg.PartnerKey, now,
	)
	req.Header.Set("X-OP-Date", opDate)
	req.Header.Set("X-OP-Expires", opExpires)
	req.Header.Set("X-OP-Authorization", opAuth)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[onepay] ipn: post %s: %v", s.cfg.IPNURL, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var ack struct {
			ErrorCode string `json:"error_code"`
			Message   string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&ack); err == nil {
			if ack.ErrorCode == "0" {
				log.Printf("[onepay] ipn: delivered funds_transfer_id=%s state=%s -> ACK", body.FundsTransferID, body.State)
				return true
			}
			log.Printf("[onepay] ipn: ACK error_code=%s message=%s", ack.ErrorCode, ack.Message)
		}
		return true
	}

	log.Printf("[onepay] ipn: funds_transfer_id=%s -> HTTP %d", body.FundsTransferID, resp.StatusCode)
	return false
}

func FormatOnePayTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}
