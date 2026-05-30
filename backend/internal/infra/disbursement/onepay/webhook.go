package onepay

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"api-server/internal/domain/ports/infrastructure"
)

// parseAndVerifyWebhook extracts the raw body and headers from the payload
// map, verifies the OWS signature, checks timestamp freshness, decodes the
// IPN body, and returns a normalized WebhookEvent.
func parseAndVerifyWebhook(payload map[string]any, partnerID, partnerKey string, now func() time.Time) (*infrastructure.WebhookEvent, error) {
	if now == nil {
		now = time.Now
	}

	// ── 1. Extract raw body, headers, and full URL from payload map ───
	if payload == nil {
		return nil, ErrMalformedWebhook
	}

	rawBody, _ := payload["__body__"].(string)
	if rawBody == "" {
		return nil, ErrMalformedWebhook
	}
	body := []byte(rawBody)

	headersRaw := payload["__headers__"]
	if headersRaw == nil {
		return nil, ErrMalformedWebhook
	}
	headers, _ := headersRaw.(map[string]string)
	if headers == nil {
		return nil, ErrMalformedWebhook
	}

	// OnePay signs IPNs with the full callback URL, not just the path.
	fullURL, _ := payload["__url__"].(string)
	if fullURL == "" {
		fullURL = "https://" + lookupHeader(headers, "Host") + "/api/v1/webhooks/disbursement/1pay"
	}

	// ── 2. Extract required signature headers (case-insensitive) ──────
	xopDate := lookupHeader(headers, "X-OP-Date")
	xopAuth := lookupHeader(headers, "X-OP-Authorization")
	if xopDate == "" || xopAuth == "" {
		return nil, ErrMalformedWebhook
	}

	// ── 3. Parse X-OP-Date ────────────────────────────────────────────
	tm, err := time.Parse(DateLayout, xopDate)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid X-OP-Date: %v", ErrMalformedWebhook, err)
	}

	// ── 4. Check timestamp freshness ──────────────────────────────────
	xopExpiresStr := lookupHeader(headers, "X-OP-Expires")
	maxAge := int64(900) // default 15 minutes
	if xopExpiresStr != "" {
		if v, err := strconv.ParseInt(xopExpiresStr, 10, 64); err == nil && v > 0 {
			maxAge = v
		}
	}
	elapsed := now().UTC().Sub(tm.UTC()).Seconds()
	if elapsed < -float64(maxAge) || elapsed > float64(maxAge) {
		return nil, ErrExpiredWebhook
	}

	// ── 5. Verify OWS signature ───────────────────────────────────────
	if !VerifyIPN("PUT", fullURL, headers, body, partnerID, partnerKey) {
		return nil, ErrInvalidWebhookSignature
	}

	// ── 6. Decode the IPN body ────────────────────────────────────────
	var ipn IPNPayload
	if err := json.Unmarshal(body, &ipn); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON body: %v", ErrMalformedWebhook, err)
	}

	// ── 7. Translate state → TransferStatus ───────────────────────────
	status := translateState(ipn.State, ipn.ResponseCode)

	// ── 8. Build WebhookEvent ─────────────────────────────────────────
	amount, _ := ipn.Amount.Int64()

	return &infrastructure.WebhookEvent{
		Method:         "FUNDS_TRANSFER",
		RequestID:      ipn.FundsTransferID,
		ProviderRef:    ipn.TransactionID,
		Status:         status,
		Amount:         amount,
		BankRef:        "",
		FailureReason:  ipn.Message,
		RawErrorCode:   ipn.ResponseCode,
		DecodedPayload: body,
	}, nil
}
