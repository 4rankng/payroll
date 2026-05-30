package ninepay

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

type Config struct {
	ListenAddr        string
	PublicURL         string
	MerchantKey       string
	SecretKey         string
	SecretKeyChecksum string
	SkipVerify        bool
	IPNURL            string
	IPNDelay          time.Duration
	SlowDelay         time.Duration
	LineWrap          string
	DBDSN             string
	BalanceDB         string
}

type Server struct {
	cfg       Config
	paymentNo atomic.Int64
	ipnCount  atomic.Uint64
	poller    *poller
	balDB     *sql.DB
	balMu     sync.Mutex
}

const initialBalance int64 = 100_000_000

func NewServer(cfg Config) *Server {
	s := &Server{cfg: cfg}
	base := time.Now().UnixNano()
	if base < 0 {
		base = -base
	}
	s.paymentNo.Store(base % 1_000_000_000_000_000)

	db, err := sql.Open("sqlite", cfg.BalanceDB)
	if err != nil {
		log.Fatalf("[9pay] balance db: open: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("[9pay] balance db: ping: %v", err)
	}
	s.balDB = db
	s.initBalanceDB()

	return s
}

func (s *Server) InitPoller() {
	if s.cfg.DBDSN == "" {
		log.Printf("[9pay] poller: disabled (no DBDSN)")
		return
	}
	p, err := newPoller(s.cfg, &s.ipnCount)
	if err != nil {
		log.Fatalf("[9pay] poller: init: %v", err)
	}
	s.poller = p
	go p.run(context.Background())
	log.Printf("[9pay] poller: enabled (interval=10s, stale_after=15s)")
}

func (s *Server) initBalanceDB() {
	s.balDB.Exec(`CREATE TABLE IF NOT EXISTS balance (id INTEGER PRIMARY KEY CHECK(id=1), amount INTEGER NOT NULL)`)
	var count int
	s.balDB.QueryRow(`SELECT COUNT(*) FROM balance`).Scan(&count)
	if count == 0 {
		s.balDB.Exec(`INSERT INTO balance (id, amount) VALUES (1, ?)`, initialBalance)
		log.Printf("[9pay] balance: initialized at %d VND", initialBalance)
	} else {
		var amt int64
		s.balDB.QueryRow(`SELECT amount FROM balance WHERE id=1`).Scan(&amt)
		log.Printf("[9pay] balance: restored at %d VND", amt)
	}
}

func (s *Server) getBalance() int64 {
	var amt int64
	s.balDB.QueryRow(`SELECT amount FROM balance WHERE id=1`).Scan(&amt)
	return amt
}

func (s *Server) deductBalance(amount int64) {
	s.balMu.Lock()
	defer s.balMu.Unlock()
	s.balDB.Exec(`UPDATE balance SET amount = amount - ? WHERE id=1`, amount)
	log.Printf("[9pay] balance: -%d -> %d VND", amount, s.getBalanceLocked())
}

func (s *Server) addBalance(amount int64) {
	s.balMu.Lock()
	defer s.balMu.Unlock()
	s.balDB.Exec(`UPDATE balance SET amount = amount + ? WHERE id=1`, amount)
	log.Printf("[9pay] balance: +%d -> %d VND", amount, s.getBalanceLocked())
}

func (s *Server) getBalanceLocked() int64 {
	var amt int64
	s.balDB.QueryRow(`SELECT amount FROM balance WHERE id=1`).Scan(&amt)
	return amt
}

type scenario string

const (
	ScenarioCompleted   scenario = "completed"
	ScenarioFailedSync  scenario = "failed_sync"
	ScenarioFailedAsync scenario = "failed_async"
	ScenarioNoIPN       scenario = "no_ipn"
	Scenario500         scenario = "http_500"
	ScenarioSlow        scenario = "slow"
)

func detectScenario(description string) scenario {
	d := strings.ToLower(description)
	switch {
	case strings.Contains(d, "mock_500"):
		return Scenario500
	case strings.Contains(d, "mock_slow"):
		return ScenarioSlow
	case strings.Contains(d, "mock_failed_sync"):
		return ScenarioFailedSync
	case strings.Contains(d, "mock_failed_async"):
		return ScenarioFailedAsync
	case strings.Contains(d, "mock_no_ipn"):
		return ScenarioNoIPN
	case strings.Contains(d, "mock_completed"):
		return ScenarioCompleted
	default:
		return ScenarioCompleted
	}
}

type createDisbursementSuccess struct {
	Status      int    `json:"status"`
	ErrorCode   string `json:"error_code"`
	Message     string `json:"message"`
	PaymentNo   int64  `json:"payment_no"`
	InvoiceNo   string `json:"invoice_no"`
	Description string `json:"description"`
	Amount      string `json:"amount"`
	BankCode    string `json:"bank_code"`
	AccountName string `json:"account_name"`
	AccountNo   string `json:"account_no"`
	CreatedAt   string `json:"created_at"`
}

type createDisbursementError struct {
	Status    int    `json:"status"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

type checkAccountResponse struct {
	Status      int    `json:"status"`
	ErrorCode   string `json:"error_code"`
	Message     string `json:"message"`
	BankCode    string `json:"bank_code,omitempty"`
	AccountNo   string `json:"account_no,omitempty"`
	AccountName string `json:"account_name,omitempty"`
	AccountType string `json:"account_type,omitempty"`
}

type balanceResponse struct {
	Status    int    `json:"status"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
	Data      int64  `json:"data"`
	Currency  string `json:"currency"`
}

type ipnPayload struct {
	AccountName     string  `json:"account_name"`
	AccountNo       string  `json:"account_no"`
	Amount          int64   `json:"amount"`
	AmountForeign   *string `json:"amount_foreign"`
	AmountOriginal  *string `json:"amount_original"`
	AmountRequest   int64   `json:"amount_request"`
	Bank            *string `json:"bank"`
	CardBrand       string  `json:"card_brand"`
	CardInfo        *string `json:"card_info"`
	CreatedAt       string  `json:"created_at"`
	Currency        string  `json:"currency"`
	Description     string  `json:"description"`
	ErrorCode       string  `json:"error_code"`
	ExcRate         *string `json:"exc_rate"`
	FailureReason   *string `json:"failure_reason"`
	ForeignCurrency *string `json:"foreign_currency"`
	InvoiceNo       string  `json:"invoice_no"`
	Lang            *string `json:"lang"`
	Method          string  `json:"method"`
	PaymentNo       int64   `json:"payment_no"`
	ProfileID       *string `json:"profile_id"`
	RequestID       string  `json:"request_id"`
	Status          int     `json:"status"`
	Tenor           *string `json:"tenor"`
}

func ReadParams(r *http.Request) ([]OrderedParam, error) {
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		if err := r.ParseForm(); err != nil {
			return nil, fmt.Errorf("parse form: %w", err)
		}
	}
	out := make([]OrderedParam, 0, len(r.PostForm))
	for k, v := range r.PostForm {
		if len(v) > 0 {
			out = append(out, OrderedParam{Key: k, Value: v[0]})
		}
	}
	return out, nil
}

func ParamValue(params []OrderedParam, key string) string {
	for _, p := range params {
		if p.Key == key {
			return p.Value
		}
	}
	return ""
}

func (s *Server) verifySignature(r *http.Request, params []OrderedParam) error {
	if s.cfg.SkipVerify {
		return nil
	}
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return fmt.Errorf("missing Authorization header")
	}
	got := extractSignature(auth)
	if got == "" {
		return fmt.Errorf("malformed Authorization header")
	}
	timestamp := r.Header.Get("Date")
	if timestamp == "" {
		return fmt.Errorf("missing Date header")
	}
	uri := s.cfg.PublicURL + r.URL.Path
	canonical := CanonicalizeParams(params)
	want := sign(r.Method, uri, canonical, timestamp, []byte(s.cfg.SecretKey))
	if want != got {
		return fmt.Errorf("signature mismatch (uri=%q canonical=%q): want %q got %q", uri, canonical, want, got)
	}
	return nil
}

func extractSignature(header string) string {
	for part := range strings.SplitSeq(header, ",") {
		part = strings.TrimSpace(part)
		if rest, ok := strings.CutPrefix(part, "Signature="); ok {
			return rest
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func hasHyphen(s string) bool { return strings.ContainsRune(s, '-') }

func formatRealCreatedAt(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000Z")
}

func (s *Server) createDisbursement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	params, err := ReadParams(r)
	if err != nil {
		log.Printf("[9pay] create: parse: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.verifySignature(r, params); err != nil {
		log.Printf("[9pay] create: signature rejected: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	requestID := ParamValue(params, "request_id")
	amount := ParamValue(params, "amount")
	description := ParamValue(params, "description")
	bankCode := ParamValue(params, "bank_code")
	accountNo := ParamValue(params, "account_no")
	accountName := ParamValue(params, "account_name")

	if hasHyphen(requestID) || hasHyphen(description) {
		log.Printf("[9pay] create: rejecting hyphenated input request_id=%q description=%q", requestID, description)
		writeJSON(w, http.StatusOK, createDisbursementError{
			Status:    1,
			ErrorCode: "318",
			Message:   "Data invalid",
		})
		return
	}

	sc := detectScenario(description)
	log.Printf("[9pay] create: request_id=%s amount=%s scenario=%s", requestID, amount, sc)

	switch sc {
	case Scenario500:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"Server Error"}`))
		return
	case ScenarioFailedSync:
		writeJSON(w, http.StatusOK, createDisbursementError{
			Status:    1,
			ErrorCode: "1004",
			Message:   "Bank account information is invalid",
		})
		return
	}

	paymentNo := s.paymentNo.Add(1)
	resp := createDisbursementSuccess{
		Status:      2,
		ErrorCode:   "000",
		Message:     "success",
		PaymentNo:   paymentNo,
		InvoiceNo:   requestID,
		Description: description,
		Amount:      amount,
		BankCode:    bankCode,
		AccountName: accountName,
		AccountNo:   accountNo,
		CreatedAt:   formatRealCreatedAt(time.Now()),
	}
	writeJSON(w, http.StatusOK, resp)

	if s.poller != nil && sc != ScenarioNoIPN {
		s.poller.markProcessed(requestID)
	}
	if sc == ScenarioNoIPN {
		log.Printf("[9pay] create: scenario=no_ipn — skipping IPN")
		return
	}

	amt, _ := strconv.ParseInt(amount, 10, 64)
	go s.scheduleIPNs(sc, requestID, paymentNo, amt, description, bankCode, accountName, accountNo)
}

func (s *Server) scheduleIPNs(
	sc scenario,
	requestID string,
	paymentNo, amount int64,
	description, bankCode, accountName, accountNo string,
) {
	delay := s.cfg.IPNDelay
	if sc == ScenarioSlow {
		delay = s.cfg.SlowDelay
	}
	time.Sleep(delay)

	base := ipnPayload{
		AccountName:   accountName,
		AccountNo:     accountNo,
		Amount:        amount,
		AmountRequest: amount,
		CardBrand:     bankCode,
		CreatedAt:     formatRealCreatedAt(time.Now()),
		Currency:      "VND",
		Description:   description,
		InvoiceNo:     requestID,
		Method:        "DISBURSEMENT",
		PaymentNo:     paymentNo,
		RequestID:     requestID,
	}

	switch sc {
	case ScenarioFailedAsync:
		failed := base
		failed.Status = 6
		failed.ErrorCode = "222"
		reason := "Test mode async failure"
		failed.FailureReason = &reason
		s.postIPN(failed)
	default:
		s.deductBalance(amount)
		success := base
		success.Status = 5
		success.ErrorCode = "000"
		s.postIPN(success)
	}
}

func sendIPN(body ipnPayload, ipnURL, secretKeyChecksum, lineWrap string, ipnCount *atomic.Uint64) {
	raw, err := json.Marshal(body)
	if err != nil {
		log.Printf("[9pay] ipn: marshal: %v", err)
		return
	}
	resultB64 := base64.StdEncoding.EncodeToString(raw)
	if shouldLineWrap(lineWrap, ipnCount) {
		resultB64 = ChunkBase64(resultB64, 64)
	}
	checksum := IpnChecksum(resultB64, secretKeyChecksum)

	form := url.Values{}
	form.Set("result", resultB64)
	form.Set("checksum", checksum)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ipnURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		log.Printf("[9pay] ipn: build request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[9pay] ipn: post %s: %v", ipnURL, err)
		return
	}
	defer resp.Body.Close()
	log.Printf("[9pay] ipn: posted invoice_no=%s payment_no=%d status=%d -> %d",
		body.InvoiceNo, body.PaymentNo, body.Status, resp.StatusCode)
}

func (s *Server) postIPN(body ipnPayload) {
	sendIPN(body, s.cfg.IPNURL, s.cfg.SecretKeyChecksum, s.cfg.LineWrap, &s.ipnCount)
}

func shouldLineWrap(mode string, ipnCount *atomic.Uint64) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	default:
		return ipnCount.Add(1)%2 == 1
	}
}

func (s *Server) checkAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	params, err := ReadParams(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.verifySignature(r, params); err != nil {
		log.Printf("[9pay] check-account: signature rejected: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	requestID := ParamValue(params, "request_id")
	bankCode := ParamValue(params, "bank_code")
	accountNo := ParamValue(params, "account_no")
	accountType := ParamValue(params, "account_type")
	log.Printf("[9pay] check-account: request_id=%s bank=%s account=%s", requestID, bankCode, accountNo)

	if hasHyphen(requestID) {
		writeJSON(w, http.StatusOK, checkAccountResponse{
			Status:    1,
			ErrorCode: "318",
			Message:   "Data invalid",
		})
		return
	}

	accountName := ParamValue(params, "account_name")
	if accountName == "" {
		accountName = "MOCK ACCOUNT HOLDER"
	}

	writeJSON(w, http.StatusOK, checkAccountResponse{
		Status:      5,
		ErrorCode:   "",
		Message:     "OK",
		BankCode:    bankCode,
		AccountNo:   accountNo,
		AccountName: accountName,
		AccountType: accountType,
	})
}

func (s *Server) balance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := s.verifySignature(r, nil); err != nil {
		log.Printf("[9pay] balance: signature rejected: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, balanceResponse{
		Status:    1,
		ErrorCode: "000",
		Message:   "OK",
		Data:      s.getBalance(),
		Currency:  "VND",
	})
}

func (s *Server) topupBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	amountStr := r.URL.Query().Get("amount")
	amt, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amt <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "amount must be a positive integer"})
		return
	}
	s.addBalance(amt)
	writeJSON(w, http.StatusOK, map[string]any{"balance": s.getBalance(), "topped_up": amt})
}

func (s *Server) Healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// RegisterRoutes wires all 9pay endpoints into the given mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/disbursement/create", s.createDisbursement)
	mux.HandleFunc("/disbursement/check-account", s.checkAccount)
	mux.HandleFunc("/disbursement/balance", s.balance)
	mux.HandleFunc("/admin/topup/9pay", s.topupBalance)
	mux.HandleFunc("/api/transaction/disbursement/index", s.exportIndex)
	mux.HandleFunc("/api/transaction/disbursement/export", s.requestExport)
	mux.HandleFunc("/api/transaction/download", s.downloadExport)
}
