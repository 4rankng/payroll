package onepay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"api-server/internal/pkg/clock"
)

const (
	EndpointProduction = "https://onepay.vn"
	EndpointSandbox    = "https://mtf.onepay.vn"

	pathCustomers     = "/onepayout/api/v1/customers"
	pathFundsTransfer = "/onepayout/api/v1/accounts/%s/funds_transfers/%s"
	pathBalance       = "/onepayout/api/v1/accounts/%s"
)

// Named error codes that callers want to special-case.
const (
	ErrorCodeDuplicateTransactionID = "07"
	ErrorCodeDuplicateTxnRef        = "22"
	ErrorCodeNoBankProcess          = "19"
)

// Config holds everything Client needs at construction time.
type Config struct {
	PartnerID            string        // X-OP-Authorization Credential prefix
	PartnerKey           string        // HMAC secret
	AccountID            string        // path param for funds_transfers / balance
	Endpoint             string        // base URL (no trailing slash)
	RequestExpirySeconds int           // X-OP-Expires header value
	HTTPTimeout          time.Duration // 0 → 30s default
	TPS                  int           // 0 → 3 default (used by queue wrapper)
	QueueBuffer          int
}

// Client speaks the OnePay PayOut API.
type Client struct {
	cfg    Config
	http   *http.Client
	logger *slog.Logger
	now    func() time.Time // test injection
}

// NewClient builds a Client from the given Config. Returns an error if
// required credentials are blank — surfacing misconfiguration at boot
// rather than at first transfer.
func NewClient(cfg Config, logger *slog.Logger) (*Client, error) {
	if cfg.PartnerID == "" {
		return nil, fmt.Errorf("onepay: PartnerID is required")
	}
	if cfg.PartnerKey == "" {
		return nil, fmt.Errorf("onepay: PartnerKey is required")
	}
	if cfg.AccountID == "" {
		return nil, fmt.Errorf("onepay: AccountID is required")
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = EndpointProduction
	}
	cfg.Endpoint = strings.TrimRight(cfg.Endpoint, "/")
	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = 30 * time.Second
	}
	if cfg.RequestExpirySeconds == 0 {
		cfg.RequestExpirySeconds = 3600
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		cfg:    cfg,
		http:   &http.Client{Timeout: cfg.HTTPTimeout},
		logger: logger,
		now:    clock.Now,
	}, nil
}

// GetAccountInfo verifies a recipient bank account before transfer.
func (c *Client) GetAccountInfo(ctx context.Context, req AccountInfoRequest) (*AccountInfoResponse, error) {
	var out AccountInfoResponse
	if err := c.doSigned(ctx, http.MethodGet, pathCustomers, req.toQuery(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateFundsTransfer initiates a disbursement. The path includes
// account_id (config) and funds_transfer_id (idempotency key).
func (c *Client) CreateFundsTransfer(ctx context.Context, req FundsTransferRequest) (*FundsTransferResponse, error) {
	path := fmt.Sprintf(pathFundsTransfer, c.cfg.AccountID, req.FundsTransferID)
	var out FundsTransferResponse
	if err := c.doSigned(ctx, http.MethodPut, path, nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// InquiryFundsTransfer polls the status of a previously-created transfer.
func (c *Client) InquiryFundsTransfer(ctx context.Context, fundsTransferID string) (*InquiryResponse, error) {
	path := fmt.Sprintf(pathFundsTransfer, c.cfg.AccountID, fundsTransferID)
	var out InquiryResponse
	if err := c.doSigned(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetBalance fetches the outgoing-account balance.
// OnePay returns code 39 (NO_DATA_FOUND) when the account has no balance
// record — this is equivalent to a zero balance, not an error.
func (c *Client) GetBalance(ctx context.Context) (*BalanceResponse, error) {
	path := fmt.Sprintf(pathBalance, c.cfg.AccountID)
	var out BalanceResponse
	if err := c.doSigned(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.ResponseCode == "39" {
			c.logger.Info("onepay balance: no data found, returning zero balance",
				"account_id", c.cfg.AccountID,
			)
			return &BalanceResponse{AccountID: c.cfg.AccountID, Balance: json.Number("0")}, nil
		}
		return nil, err
	}
	return &out, nil
}

// PartnerID is exposed for diagnostics.
func (c *Client) PartnerID() string { return c.cfg.PartnerID }

// doSigned is the single place every OnePay HTTP request runs through.
func (c *Client) doSigned(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	body any,
	out any,
) error {
	fullURL := c.cfg.Endpoint + path
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("onepay: marshal body: %w", err)
		}
		bodyBytes = b
	}
	if len(query) > 0 {
		fullURL += "?" + canonicalQueryString(query)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("onepay: build request: %w", err)
	}

	now := c.now().UTC()
	xopDate := now.Format(DateLayout)
	xopExpires := strconv.Itoa(c.cfg.RequestExpirySeconds)

	headers := map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
		"X-OP-Date":    xopDate,
		"X-OP-Expires": xopExpires,
	}

	signedHeaders := []string{"accept", "content-type", "x-op-date", "x-op-expires"}

	// Sign
	canonical, err := CanonicalRequest(method, fullURL, headers, signedHeaders, bodyBytes)
	if err != nil {
		return fmt.Errorf("onepay: canonical request: %w", err)
	}
	sts := StringToSign(now, c.cfg.PartnerID, canonical)
	key := DerivedKey(c.cfg.PartnerKey, now)
	sig := Sign(sts, key)
	cred := Credential(c.cfg.PartnerID, now)
	auth := AuthorizationHeader(cred, strings.Join(signedHeaders, ";"), sig)

	c.logger.Info("onepay signing debug",
		"method", method,
		"full_url", fullURL,
		"partner_id", c.cfg.PartnerID,
		"cred", cred,
		"signed_headers", strings.Join(signedHeaders, ";"),
		"x_op_date", xopDate,
		"canonical_request", canonical,
		"string_to_sign", sts,
		"signature", sig,
		"auth_header", auth,
	)

	headers["X-OP-Authorization"] = auth
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	c.logger.Info("onepay request",
		"method", method,
		"url", fullURL,
		"x-op-date", xopDate,
		"body", string(bodyBytes),
	)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("onepay: http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	c.logger.Info("onepay response",
		"method", method,
		"url", fullURL,
		"status", resp.StatusCode,
		"body", string(respBody),
	)
	if resp.StatusCode == http.StatusOK {
		if out == nil {
			return nil
		}
		return json.Unmarshal(respBody, out)
	}

	// Non-200: try to parse as error envelope
	var errResp ErrorResponse
	if json.Unmarshal(respBody, &errResp) == nil && errResp.ResponseCode != "" {
		return &APIError{
			StatusCode:   resp.StatusCode,
			ResponseCode: errResp.ResponseCode,
			Name:         errResp.Name,
			Message:      errResp.Message,
			MessageVI:    errResp.MessageVI,
			State:        errResp.State,
		}
	}
	return fmt.Errorf("onepay: http %d: %s", resp.StatusCode, truncate(string(respBody), 256))
}

// APIError is a typed error for callers that want to special-case
// response codes (e.g. 07 DUPLICATE_TXN, 92 INVALID_AUTH_SIGNATURE).
type APIError struct {
	StatusCode   int
	ResponseCode string
	Name         string
	Message      string
	MessageVI    string
	State        string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("onepay: %s (%s): %s", e.ResponseCode, e.Name, e.Message)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
