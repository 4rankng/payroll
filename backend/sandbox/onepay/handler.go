package onepay

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"
)

type Config struct {
	ListenAddr    string
	IPNURL        string
	IPNDelay      time.Duration
	IPNRetryDelay time.Duration
	SlowDelay     time.Duration
	SkipVerify    bool
	DBDSN         string
	BalanceDB     string
	PartnerID     string
	PartnerKey    string
	AccountID     string
}

func LoadConfig() Config {
	return Config{
		ListenAddr:    env("MOCK_LISTEN_ADDR", ":9002"),
		IPNURL:        env("MOCK_ONEPAY_IPN_URL", "http://host.docker.internal:8080/api/v1/webhooks/disbursement/1pay"),
		IPNDelay:      envDuration("MOCK_ONEPAY_IPN_DELAY", 3*time.Second),
		IPNRetryDelay: envDuration("MOCK_ONEPAY_IPN_RETRY_DELAY", 5*time.Second),
		SlowDelay:     envDuration("MOCK_ONEPAY_SLOW_DELAY", 5*time.Minute),
		SkipVerify:    envBool("MOCK_ONEPAY_SKIP_VERIFY", false),
		DBDSN:         env("MOCK_ONEPAY_DB_DSN", ""),
		BalanceDB:     env("MOCK_BALANCE_DB", "onepay_mock_balance.db"),
		PartnerID:     env("ONEPAY_PARTNER_ID", ""),
		PartnerKey:    env("ONEPAY_PARTNER_KEY", ""),
		AccountID:     env("ONEPAY_ACCOUNT_ID", ""),
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envDuration(k string, def time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

type scenario string

const (
	ScenarioCompleted   scenario = "completed"
	ScenarioFailedSync  scenario = "failed_sync"
	ScenarioFailedAsync scenario = "failed_async"
	ScenarioReversed    scenario = "reversed"
	ScenarioNoIPN       scenario = "no_ipn"
	Scenario500         scenario = "http_500"
	ScenarioSlow        scenario = "slow"
	ScenarioInvalidRSA  scenario = "invalid_rsa"
)

func detectScenario(text string) scenario {
	d := strings.ToLower(text)
	switch {
	case strings.Contains(d, "mock_invalid_rsa"):
		return ScenarioInvalidRSA
	case strings.Contains(d, "mock_500"):
		return Scenario500
	case strings.Contains(d, "mock_slow"):
		return ScenarioSlow
	case strings.Contains(d, "mock_failed_sync"):
		return ScenarioFailedSync
	case strings.Contains(d, "mock_failed_async"):
		return ScenarioFailedAsync
	case strings.Contains(d, "mock_reversed"):
		return ScenarioReversed
	case strings.Contains(d, "mock_no_ipn"):
		return ScenarioNoIPN
	case strings.Contains(d, "mock_completed"):
		return ScenarioCompleted
	default:
		return ScenarioCompleted
	}
}

func hasHyphen(s string) bool { return strings.ContainsRune(s, '-') }

type transferRecord struct {
	FundsTransferID string
	AccountID       string
	Remark          string
	AccountNumber   string
	HolderName      string
	Amount          int64
	Currency        string
	State           string
	ResponseCode    string
	TransactionID   string
	SwiftCode       string
	CreateTime      string
	UpdateTime      string
}

type Server struct {
	cfg       Config
	ipnSched  *ipnScheduler
	recoverer *recoverer
	balDB     *sql.DB
	balMu     sync.Mutex
	transMu   sync.RWMutex
	transfers map[string]*transferRecord
	txCounter atomic.Int64
}

const initialBalance int64 = 100_000_000

func NewServer(cfg Config) *Server {
	s := &Server{
		cfg:       cfg,
		ipnSched:  newIPNScheduler(cfg),
		transfers: make(map[string]*transferRecord),
	}
	s.txCounter.Store(time.Now().UnixNano() % 1_000_000_000)

	db, err := sql.Open("sqlite", cfg.BalanceDB)
	if err != nil {
		log.Fatalf("[onepay] balance db: open: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("[onepay] balance db: ping: %v", err)
	}
	s.balDB = db
	s.initBalanceDB()
	return s
}

func (s *Server) InitRecoverer() {
	if s.cfg.DBDSN == "" {
		log.Printf("[onepay] recoverer: disabled (no DBDSN)")
		return
	}
	rec, err := newRecoverer(s.cfg, s.ipnSched)
	if err != nil {
		log.Fatalf("[onepay] recoverer: init: %v", err)
	}
	s.recoverer = rec
	go rec.run(context.Background())
	log.Printf("[onepay] recoverer: enabled (interval=10s, stale_after=30s)")
}

func (s *Server) initBalanceDB() {
	s.balDB.Exec(`CREATE TABLE IF NOT EXISTS balance (id INTEGER PRIMARY KEY CHECK(id=1), amount INTEGER NOT NULL)`)
	var count int
	s.balDB.QueryRow(`SELECT COUNT(*) FROM balance`).Scan(&count)
	if count == 0 {
		s.balDB.Exec(`INSERT INTO balance (id, amount) VALUES (1, ?)`, initialBalance)
		log.Printf("[onepay] balance: initialized at %d VND", initialBalance)
	} else {
		var amt int64
		s.balDB.QueryRow(`SELECT amount FROM balance WHERE id=1`).Scan(&amt)
		log.Printf("[onepay] balance: restored at %d VND", amt)
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
	log.Printf("[onepay] balance: -%d -> %d VND", amount, s.getBalanceLocked())
}

func (s *Server) addBalance(amount int64) {
	s.balMu.Lock()
	defer s.balMu.Unlock()
	s.balDB.Exec(`UPDATE balance SET amount = amount + ? WHERE id=1`, amount)
	log.Printf("[onepay] balance: +%d -> %d VND", amount, s.getBalanceLocked())
}

func (s *Server) getBalanceLocked() int64 {
	var amt int64
	s.balDB.QueryRow(`SELECT amount FROM balance WHERE id=1`).Scan(&amt)
	return amt
}

func (s *Server) storeTransfer(rec *transferRecord) {
	s.transMu.Lock()
	defer s.transMu.Unlock()
	s.transfers[rec.FundsTransferID] = rec
}

func (s *Server) getTransfer(id string) *transferRecord {
	s.transMu.RLock()
	defer s.transMu.RUnlock()
	return s.transfers[id]
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

var (
	errInvalidSignature = map[string]string{
		"response_code": "92", "name": "INVALID_AUTHORIZATION_SIGNATURE",
		"message": "Invalid authorization signature", "state": "failed",
	}
	errInvalidParams = map[string]string{
		"response_code": "21", "name": "INVALID_PARAMETERS",
		"message": "Invalid parameters", "state": "failed",
	}
	errInternal = map[string]string{
		"response_code": "100", "name": "INTERNAL_SERVER_ERROR",
		"message": "Internal server error", "state": "failed",
	}
	errInvalidRSAUserID = map[string]string{
		"response_code": "23", "name": "INVALID_RSA_USER_ID",
		"message": "Invalid RSA User ID (detected AccountID/FundsTransferID in body)", "state": "failed",
	}
)

func (s *Server) getAccountInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.cfg.SkipVerify {
		if err := VerifyOWSSignature(r, s.cfg.PartnerID, s.cfg.PartnerKey); err != nil {
			log.Printf("[onepay] getAccountInfo: signature rejected: %v", err)
			writeJSON(w, http.StatusInternalServerError, errInvalidSignature)
			return
		}
	}

	requestID := r.URL.Query().Get("request_id")
	accountNumber := r.URL.Query().Get("account_number")
	swiftCode := r.URL.Query().Get("swift_code")
	log.Printf("[onepay] getAccountInfo: request_id=%s account=%s swift=%s", requestID, accountNumber, swiftCode)

	if hasHyphen(requestID) {
		writeJSON(w, http.StatusInternalServerError, errInvalidParams)
		return
	}

	// Permissive: return an EMPTY holder_name so the backend's CheckAccount
	// skips name verification (provider.CheckAccount only matches when
	// holder_name != ""; and the execute worker only overrides the recipient
	// name when the bank-confirmed name is non-empty). A fixed literal like
	// "MOCK ACCOUNT HOLDER" mismatches every real employee name and fails
	// every transfer. OnePay's GET /customers doesn't carry the expected name
	// (unlike 9Pay's check-account, which echoes account_name), so empty is the
	// permissive choice — the backend keeps the employee's own recorded name.
	holderName := ""
	writeJSON(w, http.StatusOK, map[string]any{
		"response_code":  "00",
		"message":        "SUCCESSFUL",
		"state":          "approved",
		"account_number": accountNumber,
		"holder_name":    holderName,
		"swift_code":     swiftCode,
	})
}

func (s *Server) requestFundsTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.cfg.SkipVerify {
		if err := VerifyOWSSignature(r, s.cfg.PartnerID, s.cfg.PartnerKey); err != nil {
			log.Printf("[onepay] requestFundsTransfer: signature rejected: %v", err)
			writeJSON(w, http.StatusInternalServerError, errInvalidSignature)
			return
		}
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/onepayout/api/v1/accounts/"), "/")
	if len(parts) < 3 {
		writeJSON(w, http.StatusInternalServerError, errInvalidParams)
		return
	}
	accountID := parts[0]
	fundsTransferID := parts[2]

	if accountID != s.cfg.AccountID {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"response_code": "15", "name": "INVALID_ACCOUNT_INFO",
			"message": "Account not found", "state": "failed",
		})
		return
	}

	if hasHyphen(fundsTransferID) {
		log.Printf("[onepay] requestFundsTransfer: rejecting hyphenated funds_transfer_id=%q", fundsTransferID)
		writeJSON(w, http.StatusInternalServerError, errInvalidParams)
		return
	}

	var rawMap map[string]interface{}
	bodyBytes, _ := io.ReadAll(r.Body)
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &rawMap); err == nil {
			if _, hasAcc := rawMap["AccountID"]; hasAcc {
				log.Printf("[onepay] requestFundsTransfer: rejecting due to AccountID in body")
				writeJSON(w, http.StatusInternalServerError, errInvalidRSAUserID)
				return
			}
			if _, hasFT := rawMap["FundsTransferID"]; hasFT {
				log.Printf("[onepay] requestFundsTransfer: rejecting due to FundsTransferID in body")
				writeJSON(w, http.StatusInternalServerError, errInvalidRSAUserID)
				return
			}
		}
	}

	var reqBody struct {
		FundsTransferInfo string `json:"funds_transfer_info"`
		Remark            string `json:"remark"`
		AccountNumber     string `json:"account_number"`
		HolderName        string `json:"holder_name"`
		Amount            int64  `json:"amount"`
		Currency          string `json:"currency"`
		SwiftCode         string `json:"swift_code"`
	}
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &reqBody)
	}

	if hasHyphen(reqBody.Remark) {
		log.Printf("[onepay] requestFundsTransfer: rejecting hyphenated remark=%q", reqBody.Remark)
		writeJSON(w, http.StatusInternalServerError, errInvalidParams)
		return
	}

	scenarioText := reqBody.Remark + " " + reqBody.FundsTransferInfo
	sc := detectScenario(scenarioText)
	log.Printf("[onepay] requestFundsTransfer: ft_id=%s amount=%d scenario=%s", fundsTransferID, reqBody.Amount, sc)

	now := time.Now()
	txID := fmt.Sprintf("TX%d%06d", now.UnixMilli(), s.txCounter.Add(1)%1000000)

	switch sc {
	case Scenario500:
		writeJSON(w, http.StatusInternalServerError, errInternal)
		return
	case ScenarioFailedSync:
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"response_code": "15", "name": "INVALID_ACCOUNT_INFO",
			"message": "Invalid account information", "state": "failed",
		})
		return
	}

	syncState := "approved"
	if sc == ScenarioFailedAsync || sc == ScenarioNoIPN || sc == ScenarioSlow {
		syncState = "pending"
	}

	rec := &transferRecord{
		FundsTransferID: fundsTransferID,
		AccountID:       accountID,
		Remark:          reqBody.Remark,
		AccountNumber:   reqBody.AccountNumber,
		HolderName:      reqBody.HolderName,
		Amount:          reqBody.Amount,
		Currency:        reqBody.Currency,
		State:           syncState,
		ResponseCode:    "00",
		TransactionID:   txID,
		SwiftCode:       reqBody.SwiftCode,
		CreateTime:      FormatOnePayTime(now),
		UpdateTime:      FormatOnePayTime(now),
	}
	s.storeTransfer(rec)

	resp := map[string]any{
		"response_code": "00", "message": "SUCCESSFUL", "transaction_id": txID,
		"funds_transfer_id": fundsTransferID, "account_id": accountID,
		"remark": reqBody.Remark, "account_number": reqBody.AccountNumber,
		"holder_name": reqBody.HolderName, "amount": reqBody.Amount,
		"currency": reqBody.Currency, "state": syncState,
		"swift_code": reqBody.SwiftCode, "create_time": rec.CreateTime,
		"update_time": rec.UpdateTime,
	}
	writeJSON(w, http.StatusOK, resp)

	if s.recoverer != nil && sc != ScenarioNoIPN {
		s.recoverer.markProcessed(fundsTransferID)
	}

	switch sc {
	case ScenarioNoIPN:
		log.Printf("[onepay] requestFundsTransfer: scenario=no_ipn — skipping IPN")
	case ScenarioCompleted:
		s.deductBalance(reqBody.Amount)
		now := time.Now()
		body := IPNBody{
			TransactionID: rec.TransactionID, FundsTransferID: rec.FundsTransferID,
			AccountID: rec.AccountID, Remark: rec.Remark, AccountNumber: rec.AccountNumber,
			HolderName: rec.HolderName, Amount: rec.Amount, Currency: rec.Currency,
			State: "approved", ResponseCode: "00", Message: "SUCCESSFUL",
			SwiftCode:  rec.SwiftCode,
			CreateTime: FormatOnePayTime(now.Add(-time.Minute)),
			UpdateTime: FormatOnePayTime(now),
		}
		s.ipnSched.scheduleIPN(body, s.cfg.IPNDelay)
	case ScenarioFailedAsync:
		s.scheduleFailedAsyncIPN(rec)
	case ScenarioReversed:
		s.scheduleReversedIPN(rec)
	case ScenarioSlow:
		s.scheduleSlowIPN(rec)
	}
}

func (s *Server) scheduleFailedAsyncIPN(rec *transferRecord) {
	body := IPNBody{
		TransactionID: rec.TransactionID, FundsTransferID: rec.FundsTransferID,
		AccountID: rec.AccountID, Remark: rec.Remark, AccountNumber: rec.AccountNumber,
		HolderName: rec.HolderName, Amount: rec.Amount, Currency: rec.Currency,
		State: "failed", ResponseCode: "91", Message: "Test mode async failure",
		SwiftCode: rec.SwiftCode, CreateTime: rec.CreateTime,
		UpdateTime: FormatOnePayTime(time.Now()),
	}
	s.ipnSched.scheduleIPN(body, s.cfg.IPNDelay)
	rec.State = "failed"
	rec.ResponseCode = "91"
}

func (s *Server) scheduleReversedIPN(rec *transferRecord) {
	s.deductBalance(rec.Amount)
	now := time.Now()
	approvedBody := IPNBody{
		TransactionID: rec.TransactionID, FundsTransferID: rec.FundsTransferID,
		AccountID: rec.AccountID, Remark: rec.Remark, AccountNumber: rec.AccountNumber,
		HolderName: rec.HolderName, Amount: rec.Amount, Currency: rec.Currency,
		State: "approved", ResponseCode: "00", Message: "SUCCESSFUL",
		SwiftCode: rec.SwiftCode, CreateTime: rec.CreateTime,
		UpdateTime: FormatOnePayTime(now),
	}
	s.ipnSched.scheduleIPN(approvedBody, s.cfg.IPNDelay)

	revertedBody := IPNBody{
		TransactionID: rec.TransactionID, FundsTransferID: rec.FundsTransferID,
		AccountID: rec.AccountID, Remark: rec.Remark, AccountNumber: rec.AccountNumber,
		HolderName: rec.HolderName, Amount: rec.Amount, Currency: rec.Currency,
		State: "reverted", ResponseCode: "00", Message: "Reversed",
		SwiftCode: rec.SwiftCode, CreateTime: rec.CreateTime,
		UpdateTime: FormatOnePayTime(now.Add(s.cfg.IPNDelay)),
	}
	s.ipnSched.scheduleIPN(revertedBody, s.cfg.IPNDelay*2)
	rec.State = "reverted"
}

func (s *Server) scheduleSlowIPN(rec *transferRecord) {
	s.deductBalance(rec.Amount)
	body := IPNBody{
		TransactionID: rec.TransactionID, FundsTransferID: rec.FundsTransferID,
		AccountID: rec.AccountID, Remark: rec.Remark, AccountNumber: rec.AccountNumber,
		HolderName: rec.HolderName, Amount: rec.Amount, Currency: rec.Currency,
		State: "approved", ResponseCode: "00", Message: "SUCCESSFUL",
		SwiftCode: rec.SwiftCode, CreateTime: rec.CreateTime,
		UpdateTime: FormatOnePayTime(time.Now()),
	}
	s.ipnSched.scheduleIPN(body, s.cfg.SlowDelay)
	rec.State = "approved"
}

func (s *Server) inquiryFundsTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.cfg.SkipVerify {
		if err := VerifyOWSSignature(r, s.cfg.PartnerID, s.cfg.PartnerKey); err != nil {
			log.Printf("[onepay] inquiryFundsTransfer: signature rejected: %v", err)
			writeJSON(w, http.StatusInternalServerError, errInvalidSignature)
			return
		}
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/onepayout/api/v1/accounts/"), "/")
	if len(parts) < 3 {
		writeJSON(w, http.StatusInternalServerError, errInvalidParams)
		return
	}
	fundsTransferID := parts[2]

	rec := s.getTransfer(fundsTransferID)
	if rec == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"response_code": "14", "name": "TRANSACTION_NOT_FOUND",
			"message": "Transaction not found", "state": "failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"response_code": rec.ResponseCode, "message": "Success",
		"transaction_id": rec.TransactionID, "funds_transfer_id": rec.FundsTransferID,
		"account_id": rec.AccountID, "remark": rec.Remark,
		"account_number": rec.AccountNumber, "holder_name": rec.HolderName,
		"amount": rec.Amount, "currency": rec.Currency, "state": rec.State,
		"swift_code": rec.SwiftCode, "create_time": rec.CreateTime,
		"update_time": rec.UpdateTime,
	})
}

func (s *Server) getBalanceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.cfg.SkipVerify {
		if err := VerifyOWSSignature(r, s.cfg.PartnerID, s.cfg.PartnerKey); err != nil {
			log.Printf("[onepay] getBalance: signature rejected: %v", err)
			writeJSON(w, http.StatusInternalServerError, errInvalidSignature)
			return
		}
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/onepayout/api/v1/accounts/"), "/")
	accountID := parts[0]
	if accountID != s.cfg.AccountID {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"response_code": "15", "name": "INVALID_ACCOUNT_INFO",
			"message": "Account not found", "state": "failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"response_code": "00", "message": "SUCCESSFUL",
		"account_id": accountID, "amount": s.getBalance(), "currency": "VND",
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

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// RegisterRoutes wires all onepay endpoints into the given mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/admin/topup/1pay", s.topupBalance)
	mux.HandleFunc("/onepayout/api/v1/customers", bodyCapture(s.getAccountInfo))
	mux.HandleFunc("/onepayout/api/v1/accounts/", bodyCapture(s.accountRouter))
}

func (s *Server) accountRouter(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/onepayout/api/v1/accounts/")
	segments := strings.Split(trimmed, "/")

	switch len(segments) {
	case 1:
		s.getBalanceHandler(w, r)
	case 3:
		if segments[1] == "funds_transfers" {
			switch r.Method {
			case http.MethodPut:
				s.requestFundsTransfer(w, r)
			case http.MethodGet:
				s.inquiryFundsTransfer(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		} else {
			http.NotFound(w, r)
		}
	default:
		http.NotFound(w, r)
	}
}

func bodyCapture(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut || r.Method == http.MethodPost {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
			ctx := context.WithValue(r.Context(), CtxKeyBody, bodyBytes)
			r = r.WithContext(ctx)
		}
		next(w, r)
	}
}
