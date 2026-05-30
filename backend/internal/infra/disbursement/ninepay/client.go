// Package ninepay implements the 9pay disbursement provider per the
// integration spec at https://developers.9pay.vn/chi-ho/chi-ho-tung-giao-dich.
//
// The package boundary is the DisbursementProvider port at
// internal/domain/ports/infrastructure. Callers should import this
// package only from bootstrap; everywhere else, depend on the port.
package ninepay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ProviderName is the stable identifier 9pay uses in the registry and
// in Settings.disbursement_provider.
const ProviderName = "9pay"

// Endpoints (override via Config.Endpoint for sandbox or staging).
const (
	EndpointProduction = "https://payment.9pay.vn"
	EndpointSandbox    = "https://sand-payment.9pay.vn"
)

// Disbursement API paths, relative to the configured Endpoint.
const (
	pathCreateDisbursement = "/disbursement/create"
	pathCheckAccount       = "/disbursement/check-account"
	pathBalance            = "/disbursement/balance"
)

// Common 9pay error codes worth distinguishing. The full list lives in
// the docs and is exposed verbatim through TransferResult.RawErrorCode
// for callers that need to react to specific values.
const (
	ErrorCodeDuplicateRequestID = "702"
)

// Config holds the credentials and endpoints required to talk to 9pay.
// All fields are required except Endpoint, IPNUrl, and HTTPTimeout
// (which fall back to sensible defaults).
type Config struct {
	// MerchantKey identifies the partner; sent in every Authorization header.
	MerchantKey string
	// SecretKey signs outbound requests. Treat as a secret; never log.
	SecretKey string
	// SecretKeyChecksum verifies inbound IPN callbacks. 9pay issues this
	// separately from SecretKey.
	SecretKeyChecksum string
	// Endpoint is the API base URL. Defaults to EndpointProduction.
	Endpoint string
	// WebEndpoint is the host serving the 9pay merchant-web (back-office)
	// API — separate from the disbursement payment API on Endpoint.
	// Real 9pay segregates these onto distinct hostnames (be.9pay.vn /
	// sand-be.9pay.vn). Defaults to Endpoint when blank — fine for the
	// local 9pay-mock which serves both surfaces from the same listener.
	WebEndpoint string
	// HTTPTimeout caps total request duration. Defaults to 30s.
	HTTPTimeout time.Duration
	// TPS is the max outbound requests per second to 9pay. Defaults to 3.
	TPS int
	// Portal carries the OAuth credentials used by the merchant-web export
	// endpoints. Empty when not configured — calls to RequestExport /
	// DownloadExport will fail closed in that case.
	Portal PortalAuthConfig
	// WebMerchantID is the numeric merchant_id sent on the reconciliation
	// export request. It is NOT the same identifier as MerchantKey
	// (MerchantKey is alphanumeric and used for HMAC signing on the
	// disbursement payment API). Empty means "no merchant_id filter" — 9pay
	// returns all transactions visible to the authenticated portal user,
	// which is what the payroll backend needs in the common case. Set when
	// the portal account has access to multiple merchants and exports must
	// be scoped to one.
	WebMerchantID string
}

// Client is the low-level HTTP wrapper for 9pay. The provider adapter
// in provider.go translates between port types and this package's
// wire-level shapes.
type Client struct {
	cfg    Config
	http   *http.Client
	logger *slog.Logger
	// portal carries the merchant-portal OAuth state used by the
	// reconciliation export endpoints. Nil when OAuth is not configured.
	portal *PortalAuth
	// now is injected for tests; defaults to time.Now.
	now func() time.Time
}

// NewClient builds a Client from the given Config. Returns an error if
// required credentials are blank — surfacing misconfiguration at boot
// rather than at first transfer.
func NewClient(cfg Config, logger *slog.Logger) (*Client, error) {
	if cfg.MerchantKey == "" {
		return nil, fmt.Errorf("ninepay: MerchantKey is required")
	}
	if cfg.SecretKey == "" {
		return nil, fmt.Errorf("ninepay: SecretKey is required")
	}
	if cfg.SecretKeyChecksum == "" {
		return nil, fmt.Errorf("ninepay: SecretKeyChecksum is required")
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = EndpointProduction
	}
	if cfg.WebEndpoint == "" {
		cfg.WebEndpoint = cfg.Endpoint
	}
	cfg.WebEndpoint = strings.TrimRight(cfg.WebEndpoint, "/")
	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = 30 * time.Second
	}
	cfg.Endpoint = strings.TrimRight(cfg.Endpoint, "/")
	if logger == nil {
		logger = slog.Default()
	}
	httpClient := &http.Client{Timeout: cfg.HTTPTimeout}
	return &Client{
		cfg:    cfg,
		http:   httpClient,
		logger: logger,
		portal: NewPortalAuth(cfg.Portal, httpClient, logger),
		now:    time.Now,
	}, nil
}

// CreateDisbursement initiates a single transfer. It signs and sends
// POST /disbursement/create with form-encoded params, then unmarshals
// the JSON response. Returns a structured response on any HTTP 200 —
// the response's Status / ErrorCode tell the caller whether 9pay
// accepted or rejected the request. Returns an error only for
// transport-level failures or non-200 HTTP responses.
func (c *Client) CreateDisbursement(ctx context.Context, in createDisbursementRequest) (*createDisbursementResponse, error) {
	c.logger.Info("ninepay: create disbursement",
		"request_id", in.RequestID, "amount", in.Amount, "bank_code", in.BankCode)
	var out createDisbursementResponse
	if err := c.doSigned(ctx, http.MethodPost, pathCreateDisbursement, in.toParams(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CheckAccount verifies that a beneficiary bank account exists and is
// reachable through 9pay's network ("Kiểm tra tài khoản nhận tiền").
// Sends POST /disbursement/check-account; the response's Status and
// ErrorCode indicate whether the account is valid.
func (c *Client) CheckAccount(ctx context.Context, in checkAccountRequest) (*checkAccountResponse, error) {
	c.logger.Info("ninepay: check account",
		"request_id", in.RequestID, "bank_code", in.BankCode)
	var out checkAccountResponse
	if err := c.doSigned(ctx, http.MethodPost, pathCheckAccount, in.toParams(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetBalance returns the partner's current 9pay balance ("Kiểm tra số
// dư của đối tác"). Sends GET /disbursement/balance — no body. The
// signing string's HTTPQUERY slot is empty for GETs with no params.
func (c *Client) GetBalance(ctx context.Context) (*balanceResponse, error) {
	c.logger.Info("ninepay: get balance")
	var out balanceResponse
	if err := c.doSigned(ctx, http.MethodGet, pathBalance, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// doSigned signs and dispatches a request, then JSON-decodes the
// response into out. Centralizes the common path so each public method
// only declares its endpoint, method, and params.
//
// Body encoding is multipart/form-data per the 9pay Postman collection;
// the signature is still computed from the canonical "k=v&k=v" form
// (CanonicalizeParams) regardless of body encoding. GET requests carry
// no body, and HTTPQUERY in the signing string is empty.
func (c *Client) doSigned(ctx context.Context, method, path string, params []OrderedParam, out any) error {
	var body io.Reader
	contentType := ""
	if method != http.MethodGet && len(params) > 0 {
		buf, ct, err := encodeMultipart(params)
		if err != nil {
			return fmt.Errorf("ninepay: encode multipart: %w", err)
		}
		body = buf
		contentType = ct
	}

	req, err := http.NewRequestWithContext(ctx, method, c.cfg.Endpoint+path, body)
	if err != nil {
		return fmt.Errorf("ninepay: build request: %w", err)
	}

	timestamp := strconv.FormatInt(c.now().Unix(), 10)
	// Signing URI is the FULL URL including scheme. The 9pay Postman
	// collection's pre-request script does:
	//   var message = "POST" + "\n" + END_POINT + "/disbursement/create" + ...
	// where END_POINT = portal_domain. The docs example at
	// https://developers.9pay.vn/danh-sach-api/quy-tac-tich-hop literally
	// shows "https://sand-payment.9pay.vn/payments/create" in the URI slot,
	// so portal_domain holds the full base URL with scheme.
	signingURI := c.cfg.Endpoint + path
	canonical := CanonicalizeParams(params)
	signature := Sign(method, signingURI, canonical, timestamp, []byte(c.cfg.SecretKey))
	// Log the exact pre-image and signature so any future "Invalid_Signature"
	// (9pay error 313 / 1002) can be reverse-engineered without redeploying.
	// Secret is never logged. canonical and signature are not secret on their
	// own — the secret is what makes the MAC unforgeable.
	c.logger.Info("ninepay: signing",
		"method", method, "uri", signingURI)
	req.Header.Set("Authorization", AuthorizationHeader(c.cfg.MerchantKey, signature))
	req.Header.Set("Date", timestamp)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ninepay: http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ninepay: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ninepay: unexpected status %d: %s", resp.StatusCode, truncate(string(respBody), 256))
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("ninepay: decode response: %w (body: %s)", err, truncate(string(respBody), 256))
	}
	return nil
}

// MerchantKey exposes the configured merchant key for diagnostics.
// Does not expose secrets.
func (c *Client) MerchantKey() string { return c.cfg.MerchantKey }

// SecretKeyChecksum returns the IPN-verification key. Unexported by
// convention; provider.go reaches in via this package's webhook helper.
func (c *Client) secretKeyChecksum() string { return c.cfg.SecretKeyChecksum }

// encodeMultipart writes the params slice as multipart/form-data, in the
// caller's order. Returns the body buffer and the Content-Type header
// (which carries the random boundary). The order is preserved on the
// wire — multipart parts come out in WriteField call order — so a
// receiver replaying our parts back through CanonicalizeParams gets the
// same canonical string we signed.
//
// Body encoding is independent of the HMAC signing string: 9pay signs
// the canonical "k=v&k=v" form regardless of how the body is framed,
// per the Postman pre-request script.
func encodeMultipart(params []OrderedParam) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, p := range params {
		if err := w.WriteField(p.Key, p.Value); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}

func formatInt(v int64) string { return strconv.FormatInt(v, 10) }

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
