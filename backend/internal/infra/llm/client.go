package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Defaults for the provider client. Timeouts and retries are deliberately
// bounded: LLM calls share one paid key and several consumers (drafting, chat,
// vision, digest), so a hung provider must not pin goroutines or burn retries
// forever.
const (
	// DefaultBaseURL is the Xiaomi MiMo OpenAI-compatible endpoint.
	DefaultBaseURL = "https://token-plan-sgp.xiaomimimo.com/v1" // TokenPlan dedicated endpoint (api.xiaomimimo.com rejects TokenPlan keys)
	// DefaultHTTPTimeout bounds one HTTP attempt.
	DefaultHTTPTimeout = 30 * time.Second
	// DefaultMaxRetries is the number of retries after the first attempt, for
	// transient failures only (transport errors, 429, 5xx).
	DefaultMaxRetries = 2
	// DefaultRetryBaseDelay is the exponential-backoff base.
	DefaultRetryBaseDelay = 500 * time.Millisecond
	// maxErrorBody bounds how much provider error body is read.
	maxErrorBody = 8 << 10
	// maxResponseBody bounds how much of a success body is read. Completions
	// can be long; the cap exists to bound memory against a runaway response.
	maxResponseBody = 4 << 20
)

// sharedTransport is reused across clients so per-call client construction
// (the API key is read from settings per call) keeps connection pooling.
var sharedTransport http.RoundTripper = &http.Transport{
	MaxIdleConns:        16,
	MaxIdleConnsPerHost: 8,
	IdleConnTimeout:     90 * time.Second,
}

// HTTPClient is the concrete OpenAI-compatible client (MiMo chat.completions).
// Safe for concurrent use.
type HTTPClient struct {
	cfg    Config
	client *http.Client
}

// NewClient builds a client from cfg, filling in defaults for zero-valued
// knobs. The BaseURL must be an absolute http(s) URL; an invalid one is
// rejected at construction so a bad settings override fails loudly instead of
// producing confusing request errors.
func NewClient(cfg Config) (*HTTPClient, error) {
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, &APIError{
			Kind:    KindInvalidRequest,
			Message: "Địa chỉ API MiMo không hợp lệ — kiểm tra URL API trong Cài đặt.",
		}
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = DefaultHTTPTimeout
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	} else if cfg.MaxRetries == 0 {
		cfg.MaxRetries = DefaultMaxRetries
	}
	if cfg.RetryBaseDelay <= 0 {
		cfg.RetryBaseDelay = DefaultRetryBaseDelay
	}
	return &HTTPClient{
		cfg:    cfg,
		client: &http.Client{Transport: sharedTransport},
	}, nil
}

// providerErrorBody mirrors the OpenAI error envelope: {"error": {"message":
// "...", "type": "...", "code": "..."}}. Fields are optional in practice.
type providerErrorBody struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// modelsResponseBody mirrors GET /models: {"object": "list", "data": [{"id":
// "...", ...}]}. Unknown fields are ignored.
type modelsResponseBody struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// Chat runs one chat.completions call. Transient failures (transport errors,
// 429, 5xx) are retried with bounded exponential backoff; auth and payload
// errors are not retried — they will not fix themselves.
func (c *HTTPClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return nil, &APIError{Kind: KindInvalidRequest, Message: "Yêu cầu thiếu mô hình (model)."}
	}
	if len(req.Messages) == 0 {
		return nil, &APIError{Kind: KindInvalidRequest, Message: "Yêu cầu thiếu nội dung hội thoại."}
	}
	// The wire contract is non-streaming; SSE consumption is a later change.
	req.Stream = false

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, &APIError{Kind: KindInternal, Message: "Không thể đóng gói yêu cầu tới MiMo."}
	}

	resp, err := c.doWithRetry(ctx, http.MethodPost, "/chat/completions", payload)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var out ChatResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBody)).Decode(&out); err != nil {
		return nil, &APIError{Kind: KindInternal, Message: "MiMo trả về dữ liệu không hợp lệ."}
	}
	if len(out.Choices) == 0 {
		return nil, &APIError{Kind: KindInternal, Message: "MiMo không trả về kết quả nào."}
	}
	return &out, nil
}

// TestConnection validates the API key with GET /models and reports latency —
// a read-only call that never spends completion tokens.
func (c *HTTPClient) TestConnection(ctx context.Context) (*TestConnectionResult, error) {
	start := time.Now()
	resp, err := c.doWithRetry(ctx, http.MethodGet, "/models", nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var body modelsResponseBody
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBody)).Decode(&body); err != nil {
		return nil, &APIError{Kind: KindInternal, Message: "MiMo trả về dữ liệu không hợp lệ."}
	}

	result := &TestConnectionResult{
		Latency:    time.Since(start),
		LatencyMS:  time.Since(start).Milliseconds(),
		ModelCount: len(body.Data),
	}
	for _, model := range body.Data {
		result.Models = append(result.Models, model.ID)
		if model.ID == DefaultModel {
			result.DefaultModelPresent = true
		}
	}
	return result, nil
}

// doWithRetry performs one HTTP attempt per call with bounded retries. Each
// attempt gets its own timeout so one slow provider response cannot consume the
// whole retry budget in a single read.
func (c *HTTPClient) doWithRetry(ctx context.Context, method, path string, payload []byte) (*http.Response, error) {
	attempts := 1 + c.cfg.MaxRetries
	var lastErr error

	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			delay := c.backoff(attempt)
			select {
			case <-ctx.Done():
				return nil, ContextError(ctx)
			case <-time.After(delay):
			}
		}

		resp, retryable, err := c.doOnce(ctx, method, path, payload)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !retryable {
			return nil, err
		}
	}
	return nil, lastErr
}

// doOnce performs a single HTTP attempt. retryable reports whether the failure
// is transient (a later attempt could succeed).
func (c *HTTPClient) doOnce(ctx context.Context, method, path string, payload []byte) (*http.Response, bool, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, c.cfg.HTTPTimeout)
	defer cancel()

	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(attemptCtx, method, c.cfg.BaseURL+path, body)
	if err != nil {
		return nil, false, &APIError{Kind: KindInternal, Message: "Không thể tạo yêu cầu tới MiMo."}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if errors.Is(attemptCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
			return nil, true, &APIError{Kind: KindTimeout, Message: MessageFor(KindTimeout)}
		}
		if ctx.Err() != nil {
			return nil, false, ContextError(ctx)
		}
		// Transport-level failure (DNS, connection reset, …) — transient.
		return nil, true, &APIError{Kind: KindUnavailable, Message: MessageFor(KindUnavailable)}
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, false, nil
	}

	apiErr := parseErrorBody(resp)
	retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
	return nil, retryable, apiErr
}

// parseErrorBody drains the error response into a classified APIError and
// closes the body. The provider's own message is appended (bounded) by
// newAPIError.
func parseErrorBody(resp *http.Response) *APIError {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	_ = resp.Body.Close()

	var parsed providerErrorBody
	providerMsg := ""
	if err := json.Unmarshal(raw, &parsed); err == nil && parsed.Error.Message != "" {
		providerMsg = parsed.Error.Message
	} else if text := strings.TrimSpace(string(raw)); text != "" && !strings.HasPrefix(text, "{") {
		providerMsg = text
	}

	kind := errorKindForStatus(resp.StatusCode)
	if kind == KindInternal {
		providerMsg = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, providerMsg)
	}
	return newAPIError(kind, resp.StatusCode, providerMsg, resp.Header.Get("X-Request-Id"))
}

// ContextError maps a cancelled/deadlined caller context to the matching
// classified error so the UI can tell "our timeout" from "caller gave up".
func ContextError(ctx context.Context) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return &APIError{Kind: KindTimeout, Message: MessageFor(KindTimeout)}
	}
	return ctx.Err()
}

// backoff computes the sleep before retry `attempt` (1-based): exponential
// with ±25% jitter, capped at 5s so the bounded retry budget stays bounded.
func (c *HTTPClient) backoff(attempt int) time.Duration {
	delay := c.cfg.RetryBaseDelay * time.Duration(1<<uint(attempt-1))
	const maxDelay = 5 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}
	jitter := 1 + (rand.Float64()-0.5)*0.5 //nolint:gosec // timing jitter, not security
	return time.Duration(float64(delay) * jitter)
}
