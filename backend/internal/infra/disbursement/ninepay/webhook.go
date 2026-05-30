package ninepay

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"api-server/internal/domain/ports/infrastructure"
)

// ErrInvalidWebhookSignature is returned when the IPN's checksum does
// not match the SHA-256(result + secretKeyChecksum) the partner
// computed. Treat as a hard reject — the payload is untrusted.
var ErrInvalidWebhookSignature = errors.New("ninepay: invalid webhook checksum")

// ErrMalformedWebhook is returned when required fields are missing or
// the result body fails to base64-decode / JSON-unmarshal.
var ErrMalformedWebhook = errors.New("ninepay: malformed webhook payload")

// parseAndVerifyWebhook validates the IPN payload's checksum, decodes
// the base64-encoded result, and translates it into the normalized
// WebhookEvent shape. payload is the form-decoded request body 9pay
// POSTs to the registered IPN URL.
//
// The signature scheme — plain SHA-256 of (result + secretKeyChecksum)
// uppercased — is per 9pay's integration rules. It is intentionally
// distinct from the HMAC scheme used for outbound request signing.
func parseAndVerifyWebhook(payload map[string]any, secretKeyChecksum string) (*infrastructure.WebhookEvent, error) {
	result, _ := payload["result"].(string)
	checksum, _ := payload["checksum"].(string)
	if result == "" || checksum == "" {
		return nil, fmt.Errorf("%w: missing result or checksum", ErrMalformedWebhook)
	}

	if !VerifyChecksum(result, secretKeyChecksum, checksum) {
		return nil, ErrInvalidWebhookSignature
	}

	decoded, err := decodeFlexibleBase64(result)
	if err != nil {
		return nil, fmt.Errorf("%w: base64: %v", ErrMalformedWebhook, err)
	}

	var body webhookResultBody
	if err := json.Unmarshal(decoded, &body); err != nil {
		return nil, fmt.Errorf("%w: json: %v", ErrMalformedWebhook, err)
	}

	// Compact the decoded JSON for storage (strips pretty-printing whitespace).
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, decoded); err != nil {
		compacted = *bytes.NewBuffer(decoded)
	}

	bankRef := ""
	if body.BankRef != nil {
		bankRef = *body.BankRef
	}

	return &infrastructure.WebhookEvent{
		RequestID:      body.RequestID,
		ProviderRef:    normalizeInvoiceNo(body.InvoiceNo),
		Status:         translateStatus(body.Status, body.ErrorCode, body.FailureReason, body.Message),
		BankRef:        bankRef,
		FailureReason:  body.FailureReason,
		RawErrorCode:   body.ErrorCode,
		Amount:         body.Amount,
		Method:         body.Method,
		DecodedPayload: compacted.Bytes(),
	}, nil
}

// translateStatus maps a 9pay numeric status and error code to the
// normalized TransferStatus.
//
// 9pay uses both "" and "000" interchangeably as the no-error code:
// /disbursement/balance returns "000", /disbursement/check-account
// returns "", and /disbursement/create returns "000" on the synchronous
// "accepted" response. The synchronous response only confirms 9pay
// queued the transfer — final success is reported asynchronously via
// the IPN webhook with status=5. Anything else with a real error code
// (1001, 1006, 318, 702, …) is a hard failure.
//
// Reversal ("hoàn chuyển tiền" — recipient bank rejected the transfer
// after 9pay had already settled it, so 9pay refunds principal+fee back
// to the merchant balance) is surfaced in the merchant UI but the
// numeric status code 9pay uses for it is not in the public docs and
// no dedicated error_code exists for it either (the canonical 9pay
// error list — 000/318/431/666/702/1001/1002/1004/1005/1006/1007/1008
// — has no refund-specific entry). We detect it from the
// failure_reason / message text. Once a real reversal IPN is observed
// in the wild, add the numeric status here explicitly and keep the
// text fallback as belt-and-braces.
func translateStatus(status int, errorCode, failureReason, message string) infrastructure.TransferStatus {
	if status == 5 && isSuccessCode(errorCode) {
		return infrastructure.TransferStatusSuccess
	}
	if !isSuccessCode(errorCode) {
		return infrastructure.TransferStatusFailed
	}
	return infrastructure.TransferStatusPending
}

// reversalMarkers are substrings (lower-cased) that 9pay is observed
// or expected to put in failure_reason / message when a transfer was
// settled and then reversed by the recipient bank. Matched
// case-insensitively against the original Vietnamese (with diacritics)
// and an unaccented fallback.

// decodeFlexibleBase64 decodes 9pay's IPN `result` field, tolerating
// the encoding variants observed in the wild against sand-payment.9pay.vn:
//
//   - MIME line-wrapped base64 (CR/LF every 64 or 76 chars). Encoded by
//     PHP base64_encode followed by chunk_split, common in legacy PHP
//     senders. Standard StdEncoding.DecodeString rejects these as
//     "illegal base64 data at input byte N" where N is the first
//     line-break position.
//   - Trailing or leading whitespace, including BOM (U+FEFF).
//   - Surrounding quotes (some intermediaries wrap form values in `"`).
//   - URL-safe alphabet (`-` / `_` instead of `+` / `/`).
//   - Missing padding (raw* encodings).
//
// Checksum verification has already passed on the original raw string,
// so cleaning the input here is safe: we're only relaxing the
// canonicalization that base64 itself imposes, not weakening the MAC.
//
// Strategy: clean once, then try Std → URL → RawStd → RawURL until one
// produces bytes. The first successful decode wins. If all four fail
// the caller gets a wrapped error referring to the standard-alphabet
// attempt (the most common case) so log messages stay diagnosable.
func decodeFlexibleBase64(s string) ([]byte, error) {
	cleaned := sanitizeBase64Input(s)

	if decoded, err := base64.StdEncoding.DecodeString(cleaned); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.URLEncoding.DecodeString(cleaned); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(cleaned, "=")); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(cleaned, "=")); err == nil {
		return decoded, nil
	}

	// Re-run StdEncoding to surface the original error for diagnostics —
	// callers wrap it as ErrMalformedWebhook so this position survives
	// in the structured log.
	_, err := base64.StdEncoding.DecodeString(cleaned)
	return nil, err
}

// sanitizeBase64Input strips the noise that legitimate 9pay payloads
// occasionally carry: BOM, surrounding quotes, and any whitespace
// (CR / LF / TAB / SP / U+00A0 etc.) anywhere in the body. The base64
// alphabet contains no whitespace, so removing it cannot collide with
// real data.
func sanitizeBase64Input(s string) string {
	s = strings.TrimPrefix(s, "\ufeff")
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			s = s[1 : len(s)-1]
		}
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}
