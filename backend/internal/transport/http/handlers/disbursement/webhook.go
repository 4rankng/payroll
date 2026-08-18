package disbursement

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/domain/wallet"
	asynqinfra "api-server/internal/infra/asynq"
	onepayinfra "api-server/internal/infra/disbursement/onepay"
	"api-server/internal/infra/persistence"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// WebhookHandler dispatches inbound IPN callbacks to the matching
// provider for signature verification and parsing. Routes are
// per-provider (e.g. /webhooks/disbursement/9pay) so the provider name
// comes from the URL, not the body — which means we can verify with
// the right key even before unmarshaling the payload.
type WebhookHandler struct {
	registry       *disbursement.Registry
	providerTxs    *disbursement.WalletPaymentService
	asynqClient    *asynqinfra.Client
	walletIPNRepo  wallet.WalletIPNRepository
	redisClient    *persistence.RedisClient
	logger         *slog.Logger
	providerLogger *slog.Logger
}

func NewWebhookHandler(registry *disbursement.Registry, providerTxs *disbursement.WalletPaymentService, asynqClient *asynqinfra.Client, walletIPNRepo wallet.WalletIPNRepository, redisClient *persistence.RedisClient, logger *slog.Logger, providerLogger *slog.Logger) *WebhookHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &WebhookHandler{registry: registry, providerTxs: providerTxs, asynqClient: asynqClient, walletIPNRepo: walletIPNRepo, redisClient: redisClient, logger: logger, providerLogger: providerLogger}
}

// ipnNonceTTL bounds the replay-detection window. A callback re-delivered
// inside this window with the same (RequestID, ProviderRef) is treated as a
// replay and rejected; after it expires, a repeat is allowed through (matching
// the FSM's own terminal-state idempotency for late re-delivery).
const ipnNonceTTL = 10 * time.Minute

// checkIPNNonce returns true if this IPN is fresh (first time we see this
// request_id/provider_ref within the TTL), false if it is a replay. On Redis
// error it returns true (fail-open): a Redis outage must not block disbursement
// reconcile — the FSM's terminal-state guard still makes re-application a no-op.
func (h *WebhookHandler) checkIPNNonce(ctx context.Context, providerName, requestID, providerRef string) bool {
	if h.redisClient == nil || requestID == "" {
		return true
	}
	key := fmt.Sprintf("ipn:nonce:%s:%s:%s", providerName, requestID, providerRef)
	ok, err := h.redisClient.SetNX(ctx, key, 1, ipnNonceTTL).Result()
	if err != nil {
		h.logProviderEvent(slog.LevelWarn, "disbursement: nonce check failed-open (redis error)",
			"provider", providerName, "request_id", requestID, "error", err)
		return true
	}
	return ok
}

// logProviderEvent logs a webhook event to the payment-gateway file logger.
func (h *WebhookHandler) logProviderEvent(level slog.Level, msg string, args ...any) {
	if h.providerLogger != nil {
		switch level {
		case slog.LevelWarn:
			h.providerLogger.Warn(msg, args...)
		default:
			h.providerLogger.Info(msg, args...)
		}
	}
}

// providerCtxKey is the gin.Context key used to pass the provider name from
// per-provider route registration (ReceiveFrom) into the shared Receive handler.
const providerCtxKey = "disbursement_provider"

// ReceiveFrom returns a gin.HandlerFunc bound to a specific provider name.
// Used for per-provider route registration (e.g. PUT /9pay, PUT /1pay) where
// the route path is a literal segment rather than a :provider parameter.
func (h *WebhookHandler) ReceiveFrom(providerName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(providerCtxKey, providerName)
		h.Receive(c)
	}
}

// Receive handles inbound disbursement webhook callbacks. The provider name
// is resolved from the gin context (set by ReceiveFrom) or the :provider
// path parameter (backward compat).
func (h *WebhookHandler) Receive(c *gin.Context) {
	providerName := c.Param("provider")
	if providerName == "" {
		if v, ok := c.Get(providerCtxKey); ok {
			providerName = v.(string)
		}
	}
	if providerName == "" {
		response.BadRequest(c, "provider path parameter is required")
		return
	}

	provider, ok := h.registry.ByName(providerName)
	if !ok {
		response.NotFound(c, "unknown disbursement provider: "+providerName)
		return
	}

	rawBody, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))

	payload, err := decodeWebhookPayload(c)
	if err != nil {
		h.logProviderEvent(slog.LevelWarn, "disbursement: webhook decode failed",
			"provider", providerName,
			"error", err)
		response.BadRequest(c, "invalid webhook payload")
		return
	}

	payload["__body__"] = string(rawBody)
	payload["__headers__"] = extractHeaders(c)
	payload["__url__"] = requestFullURL(c)

	event, err := provider.VerifyAndParseWebhook(c.Request.Context(), payload)
	if err != nil {
		h.logProviderEvent(slog.LevelWarn, "disbursement: webhook rejected",
			"provider", providerName,
			"error", err,
			"client_ip", c.ClientIP())
		response.BadRequest(c, "webhook rejected")
		return
	}

	h.logProviderEvent(slog.LevelInfo, "disbursement: webhook accepted",
		"provider", providerName,
		"request_id", event.RequestID,
		"provider_ref", event.ProviderRef,
		"status", event.Status,
		"amount", event.Amount,
		"raw_error_code", event.RawErrorCode,
	)

	// Replay guard: reject a re-delivered IPN inside the nonce window. Keyed on
	// (provider, request_id, provider_ref) — the payload has no jti, so these
	// real fields are the dedup identity. Fail-open on Redis error so a Redis
	// outage doesn't block disbursement reconcile (the FSM still idempotently
	// no-ops a late re-delivery into a terminal row).
	if !h.checkIPNNonce(c.Request.Context(), providerName, event.RequestID, event.ProviderRef) {
		h.logProviderEvent(slog.LevelWarn, "disbursement: replayed IPN rejected (nonce hit)",
			"provider", providerName,
			"request_id", event.RequestID,
			"provider_ref", event.ProviderRef)
		writeWebhookAcknowledgement(c, providerName)
		return
	}

	var ipnRecordID uint64
	if h.walletIPNRepo != nil {
		var failureReason *string
		if event.FailureReason != "" {
			failureReason = &event.FailureReason
		}
		ipn := &wallet.WalletIPN{
			Provider:         providerName,
			InvoiceNo:        event.ProviderRef,
			RequestID:        event.RequestID,
			Status:           string(event.Status),
			Amount:           event.Amount,
			RawErrorCode:     event.RawErrorCode,
			FailureReason:    failureReason,
			RawPayload:       rawPayloadBytes(event, payload),
			ProcessingStatus: "pending",
		}
		id, err := h.walletIPNRepo.Create(c.Request.Context(), ipn)
		if err != nil {
			h.logProviderEvent(slog.LevelWarn, "disbursement: failed to record IPN audit", "provider", providerName, "error", err)
		} else {
			ipnRecordID = id
		}
	}

	if h.asynqClient != nil {
		err := h.asynqClient.EnqueueIPNProcess(asynqinfra.IPNProcessPayload{
			Provider:      providerName,
			InvoiceNo:     event.ProviderRef,
			RequestID:     event.RequestID,
			Status:        string(event.Status),
			Amount:        event.Amount,
			RawErrorCode:  event.RawErrorCode,
			FailureReason: event.FailureReason,
			IPNRecordID:   ipnRecordID,
		})
		if err != nil {
			h.logProviderEvent(slog.LevelWarn, "disbursement: failed to enqueue IPN — falling back to sync",
				"provider", providerName,
				"invoice_no", event.ProviderRef,
				"request_id", event.RequestID,
				"error", err)
			h.applyIPNSync(c, providerName, event)
		}
		writeWebhookAcknowledgement(c, providerName)
		return
	}

	h.applyIPNSync(c, providerName, event)
	writeWebhookAcknowledgement(c, providerName)
}

func writeWebhookAcknowledgement(c *gin.Context, providerName string) {
	if providerName == "1pay" {
		c.JSON(http.StatusOK, onepayinfra.IPNAck{ErrorCode: "0", Message: "Success"})
		return
	}
	response.Success(c, gin.H{"received": true}, "webhook accepted")
}

func (h *WebhookHandler) applyIPNSync(c *gin.Context, providerName string, event *infrastructure.WebhookEvent) {
	if h.providerTxs == nil {
		return
	}
	_, err := h.providerTxs.RecordIPN(c.Request.Context(), disbursement.IPNResult{
		Status:        event.Status,
		InvoiceNo:     event.ProviderRef,
		RequestID:     event.RequestID,
		Amount:        event.Amount,
		RawErrorCode:  event.RawErrorCode,
		RawMessage:    "",
		FailureReason: event.FailureReason,
		Source:        domaintx.ResolutionSourceIPN,
	})
	switch {
	case err == nil:
	case errors.Is(err, domaintx.ErrNotFound):
		h.logProviderEvent(slog.LevelWarn, "disbursement: webhook references unknown invoice",
			"provider", providerName,
			"invoice_no", event.ProviderRef,
			"request_id", event.RequestID,
			"status", event.Status)
	default:
		var terminalErr domaintx.TerminalStateError
		if errors.As(err, &terminalErr) {
			h.logProviderEvent(slog.LevelInfo, "disbursement: webhook ignored — row already in terminal state (idempotency)",
				"provider", providerName,
				"invoice_no", event.ProviderRef,
				"current_state", terminalErr.State,
				"trigger", terminalErr.Trigger)
		} else {
			h.logProviderEvent(slog.LevelWarn, "disbursement: webhook FSM apply failed",
				"provider", providerName,
				"invoice_no", event.ProviderRef,
				"error", err)
		}
	}
}

// rawPayloadBytes returns the decoded webhook payload for storage.
// When the provider already decoded the payload (e.g. 9pay base64),
// use that directly. Otherwise fall back to the raw form-decoded map.
func rawPayloadBytes(event *infrastructure.WebhookEvent, payload map[string]any) []byte {
	if len(event.DecodedPayload) > 0 {
		return event.DecodedPayload
	}
	return persistence.MarshalPayload(payload)
}

func decodeWebhookPayload(c *gin.Context) (map[string]any, error) {
	contentType := strings.ToLower(c.GetHeader("Content-Type"))

	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, err
		}
		return parseFormPreservingPlus(string(body))
	}

	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := c.Request.ParseForm(); err != nil {
			return nil, err
		}
		out := make(map[string]any, len(c.Request.PostForm))
		for k, v := range c.Request.PostForm {
			if len(v) > 0 {
				out[k] = v[0]
			}
		}
		return out, nil
	}

	var out map[string]any
	if err := c.ShouldBindJSON(&out); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, errors.New("empty body")
	}
	return out, nil
}

func extractHeaders(c *gin.Context) map[string]string {
	headers := make(map[string]string, len(c.Request.Header))
	for k, v := range c.Request.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	return headers
}

// requestFullURL reconstructs the full URL from the incoming request,
// needed because OnePay signs IPN callbacks with the full callback URL.
func requestFullURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host + c.Request.URL.RequestURI()
}

func parseFormPreservingPlus(body string) (map[string]any, error) {
	out := map[string]any{}
	for _, pair := range strings.Split(body, "&") {
		if pair == "" {
			continue
		}
		eq := strings.IndexByte(pair, '=')
		var rawKey, rawVal string
		if eq < 0 {
			rawKey = pair
		} else {
			rawKey = pair[:eq]
			rawVal = pair[eq+1:]
		}
		key, err := url.PathUnescape(rawKey)
		if err != nil {
			return nil, fmt.Errorf("decode form key: %w", err)
		}
		val, err := url.PathUnescape(rawVal)
		if err != nil {
			return nil, fmt.Errorf("decode form value for %q: %w", key, err)
		}
		if _, exists := out[key]; !exists {
			out[key] = val
		}
	}
	return out, nil
}
