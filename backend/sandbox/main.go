package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"payroll-sandbox/ninepay"
	"payroll-sandbox/onepay"
)

func main() {
	ninepayCfg := ninepay.Config{
		ListenAddr:        ":9001",
		PublicURL:         strings.TrimRight(env("MOCK_PUBLIC_URL", "http://localhost:9001"), "/"),
		MerchantKey:       env("NINEPAY_MERCHANT_KEY", "mock-merchant"),
		SecretKey:         env("NINEPAY_SECRET_KEY", "mock-secret"),
		SecretKeyChecksum: env("NINEPAY_SECRET_KEY_CHECKSUM", "mock-checksum"),
		SkipVerify:        envBool("MOCK_SKIP_VERIFY", false),
		IPNURL:            env("MOCK_IPN_URL", "http://host.docker.internal:8080/api/v1/webhooks/disbursement/9pay"),
		IPNDelay:          envDuration("MOCK_IPN_DELAY", 3*time.Second),
		SlowDelay:         envDuration("MOCK_SLOW_DELAY", 5*time.Minute),
		LineWrap:          strings.ToLower(env("MOCK_BASE64_LINEWRAP", "alternate")),
		DBDSN:             env("MOCK_DB_DSN", ""),
		BalanceDB:         env("MOCK_BALANCE_DB", "9pay_mock_balance.db"),
	}

	onepayCfg := onepay.LoadConfig()

	ninepaySrv := ninepay.NewServer(ninepayCfg)
	onepaySrv := onepay.NewServer(onepayCfg)

	mux := http.NewServeMux()

	// Health check — shared
	mux.HandleFunc("/healthz", ninepaySrv.Healthz)

	// Register provider routes
	ninepaySrv.RegisterRoutes(mux)
	onepaySrv.RegisterRoutes(mux)

	// Start background workers
	ninepaySrv.InitPoller()
	onepaySrv.InitRecoverer()

	addr := env("MOCK_LISTEN_ADDR", ":9001")
	log.Printf("payroll-sandbox listening on %s", addr)
	log.Printf("  9pay routes: /disbursement/*, /api/transaction/*")
	log.Printf("  onepay routes: /onepayout/*")

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
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
