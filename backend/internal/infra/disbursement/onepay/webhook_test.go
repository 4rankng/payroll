package onepay

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain/ports/infrastructure"
)

const testCallbackURL = "https://tingting.vip/api/v1/webhooks/disbursement/1pay"

// buildSignedIPNHeaders creates valid OWS signature headers for a given
// IPN body, partner key, and timestamp. Used by webhook tests.
func buildSignedIPNHeaders(t *testing.T, body []byte, partnerKey, partnerID string, tm time.Time) map[string]string {
	t.Helper()
	headers := map[string]string{
		"accept":       "application/json",
		"x-op-date":    tm.Format(DateLayout),
		"x-op-expires": "6000",
	}
	if len(body) > 0 {
		headers["content-type"] = "application/json"
		headers["content-length"] = fmt.Sprintf("%d", len(body))
	}
	signedHeaders := []string{"accept", "x-op-date", "x-op-expires"}

	// Mirror VerifyIPN: OnePay signs IPNs with their internal gateway URL.
	canonicalURI := uriEncode("http://localhost/payout-merchants/https/tingting.vip/443/api/v1/webhooks/disbursement/1pay", false)
	canonicalHeaders, signedHeadersNorm := buildCanonicalHeaders(headers, signedHeaders)
	if body == nil {
		body = []byte{}
	}
	bodyHashArr := sha256.Sum256(body)
	bodyHash := fmt.Sprintf("%x", bodyHashArr[:])
	canonical := strings.Join([]string{
		"PUT",
		canonicalURI,
		"",
		canonicalHeaders,
		signedHeadersNorm,
		bodyHash,
	}, "\n")

	sts := StringToSign(tm, partnerID, canonical)
	key := DerivedKey(partnerKey, tm)
	sig := Sign(sts, key)
	cred := Credential(partnerID, tm)
	headers["X-OP-Authorization"] = AuthorizationHeader(cred, strings.Join(signedHeaders, ";"), sig)
	return headers
}

func ipnBody(state string, fundsTransferID string) []byte {
	body, _ := json.Marshal(map[string]any{
		"transaction_id":    "TXN-IPN-001",
		"funds_transfer_id": fundsTransferID,
		"account_id":        testAccountID,
		"account_number":    "1023020330000",
		"holder_name":       "NGUYEN VAN A",
		"amount":            50000,
		"currency":          "VND",
		"state":             state,
		"response_code":     "0",
		"message":           "Success",
		"swift_code":        "VCBVNVVX",
		"create_time":       "20260108T112907Z",
		"update_time":       "20260108T112910Z",
	})
	return body
}

// toWebhookPayload converts body + headers into the map[string]any format
// that the VerifyAndParseWebhook port accepts. We stash headers and raw body
// under reserved keys so the parser can extract them.
func toWebhookPayload(body []byte, headers map[string]string) map[string]any {
	payload := map[string]any{
		"__body__":    string(body),
		"__headers__": headers,
		"__url__":     testCallbackURL,
	}
	// Also decode the body as a fallback for providers that read JSON directly
	var decoded map[string]any
	if json.Unmarshal(body, &decoded) == nil {
		for k, v := range decoded {
			payload[k] = v
		}
	}
	return payload
}

// ---------------------------------------------------------------------------
// Happy path — valid signature
// ---------------------------------------------------------------------------

func TestParseAndVerifyWebhook_ValidSignature(t *testing.T) {
	tm := mustParseDate(t, "20260108T112907Z")
	body := ipnBody("approved", "FT-IPN-001")
	headers := buildSignedIPNHeaders(t, body, testPartnerKey, testPartnerID, tm)

	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
	}, nil)
	c.now = func() time.Time { return tm }
	p := NewProvider(c, nil)

	payload := toWebhookPayload(body, headers)
	ev, err := p.VerifyAndParseWebhook(context.Background(), payload)
	if err != nil {
		t.Fatalf("VerifyAndParseWebhook: %v", err)
	}
	if ev.Status != infrastructure.TransferStatusSuccess {
		t.Errorf("Status = %q, want success (approved)", ev.Status)
	}
	if ev.RequestID != "FT-IPN-001" {
		t.Errorf("RequestID = %q", ev.RequestID)
	}
	if ev.ProviderRef != "TXN-IPN-001" {
		t.Errorf("ProviderRef = %q", ev.ProviderRef)
	}
}

// ---------------------------------------------------------------------------
// Tampered body → ErrInvalidWebhookSignature
// ---------------------------------------------------------------------------

func TestParseAndVerifyWebhook_TamperedBody(t *testing.T) {
	tm := mustParseDate(t, "20260108T112907Z")
	body := ipnBody("approved", "FT-001")
	headers := buildSignedIPNHeaders(t, body, testPartnerKey, testPartnerID, tm)

	// Tamper with the body after signing
	tamperedBody := []byte(strings.ReplaceAll(string(body), "approved", "failed"))

	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
	}, nil)
	c.now = func() time.Time { return tm }
	p := NewProvider(c, nil)

	payload := toWebhookPayload(tamperedBody, headers)
	_, err := p.VerifyAndParseWebhook(context.Background(), payload)
	if !isWebhookError(err, ErrInvalidWebhookSignature) {
		t.Fatalf("err = %v, want ErrInvalidWebhookSignature", err)
	}
}

// ---------------------------------------------------------------------------
// Wrong partner key → ErrInvalidWebhookSignature
// ---------------------------------------------------------------------------

func TestParseAndVerifyWebhook_WrongPartnerKey(t *testing.T) {
	tm := mustParseDate(t, "20260108T112907Z")
	body := ipnBody("approved", "FT-001")
	headers := buildSignedIPNHeaders(t, body, testPartnerKey, testPartnerID, tm)

	// Create a provider with a DIFFERENT key
	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: "wrong-partner-key", AccountID: testAccountID,
	}, nil)
	c.now = func() time.Time { return tm }
	p := NewProvider(c, nil)

	payload := toWebhookPayload(body, headers)
	_, err := p.VerifyAndParseWebhook(context.Background(), payload)
	if !isWebhookError(err, ErrInvalidWebhookSignature) {
		t.Fatalf("err = %v, want ErrInvalidWebhookSignature", err)
	}
}

// ---------------------------------------------------------------------------
// Expired X-OP-Date → ErrExpiredWebhook
// ---------------------------------------------------------------------------

func TestParseAndVerifyWebhook_ExpiredDate(t *testing.T) {
	oldTime := mustParseDate(t, "20200101T000000Z")
	body := ipnBody("approved", "FT-001")
	headers := buildSignedIPNHeaders(t, body, testPartnerKey, testPartnerID, oldTime)

	// Provider uses current time for expiry check
	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
	}, nil)
	p := NewProvider(c, nil)

	payload := toWebhookPayload(body, headers)
	_, err := p.VerifyAndParseWebhook(context.Background(), payload)
	if !isWebhookError(err, ErrExpiredWebhook) {
		t.Fatalf("err = %v, want ErrExpiredWebhook", err)
	}
}

// ---------------------------------------------------------------------------
// Missing headers → ErrMalformedWebhook
// ---------------------------------------------------------------------------

func TestParseAndVerifyWebhook_MissingHeaders(t *testing.T) {
	body := ipnBody("approved", "FT-001")

	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
	}, nil)
	p := NewProvider(c, nil)

	cases := []struct {
		name    string
		headers map[string]string
	}{
		{"no X-OP-Date", map[string]string{"X-OP-Authorization": "sig"}},
		{"no X-OP-Authorization", map[string]string{"X-OP-Date": "20260108T112907Z"}},
		{"empty headers", map[string]string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := toWebhookPayload(body, tc.headers)
			_, err := p.VerifyAndParseWebhook(context.Background(), payload)
			if !isWebhookError(err, ErrMalformedWebhook) {
				t.Fatalf("err = %v, want ErrMalformedWebhook", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// State mapping in webhook
// ---------------------------------------------------------------------------

func TestParseAndVerifyWebhook_StateReverted(t *testing.T) {
	tm := mustParseDate(t, "20260108T112907Z")
	body := ipnBody("reverted", "FT-REV-001")
	headers := buildSignedIPNHeaders(t, body, testPartnerKey, testPartnerID, tm)

	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
	}, nil)
	c.now = func() time.Time { return tm }
	p := NewProvider(c, nil)

	ev, err := p.VerifyAndParseWebhook(context.Background(), toWebhookPayload(body, headers))
	if err != nil {
		t.Fatalf("VerifyAndParseWebhook: %v", err)
	}
	if ev.Status != infrastructure.TransferStatusReversed {
		t.Errorf("Status = %q, want reversed", ev.Status)
	}
}

func TestParseAndVerifyWebhook_StatePending(t *testing.T) {
	tm := mustParseDate(t, "20260108T112907Z")
	body := ipnBody("pending", "FT-PEND-001")
	headers := buildSignedIPNHeaders(t, body, testPartnerKey, testPartnerID, tm)

	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
	}, nil)
	c.now = func() time.Time { return tm }
	p := NewProvider(c, nil)

	ev, err := p.VerifyAndParseWebhook(context.Background(), toWebhookPayload(body, headers))
	if err != nil {
		t.Fatalf("VerifyAndParseWebhook: %v", err)
	}
	if ev.Status != infrastructure.TransferStatusPending {
		t.Errorf("Status = %q, want pending", ev.Status)
	}
}

func TestParseAndVerifyWebhook_StateApproved(t *testing.T) {
	tm := mustParseDate(t, "20260108T112907Z")
	body := ipnBody("approved", "FT-APP-001")
	headers := buildSignedIPNHeaders(t, body, testPartnerKey, testPartnerID, tm)

	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
	}, nil)
	c.now = func() time.Time { return tm }
	p := NewProvider(c, nil)

	ev, err := p.VerifyAndParseWebhook(context.Background(), toWebhookPayload(body, headers))
	if err != nil {
		t.Fatalf("VerifyAndParseWebhook: %v", err)
	}
	if ev.Status != infrastructure.TransferStatusSuccess {
		t.Errorf("Status = %q, want success (approved)", ev.Status)
	}
}

// ---------------------------------------------------------------------------
// Fuzz / adversarial inputs don't panic
// ---------------------------------------------------------------------------

func TestParseAndVerifyWebhook_AdversarialInputs(t *testing.T) {
	c, _ := NewClient(Config{
		PartnerID: testPartnerID, PartnerKey: testPartnerKey, AccountID: testAccountID,
	}, nil)
	p := NewProvider(c, nil)

	cases := []struct {
		name    string
		payload map[string]any
	}{
		{"nil payload", nil},
		{"empty payload", map[string]any{}},
		{"garbage body", map[string]any{"__body__": "!!not-json!!", "__headers__": map[string]string{"X-OP-Date": "x", "X-OP-Authorization": "x"}}},
		{"null body", map[string]any{"__body__": "null", "__headers__": map[string]string{"X-OP-Date": "x", "X-OP-Authorization": "x"}}},
		{"malformed auth", map[string]any{"__body__": `{"state":"approved"}`, "__headers__": map[string]string{"X-OP-Date": "20260108T112907Z", "X-OP-Authorization": "not-ows1-format"}}},
		{"binary body", map[string]any{"__body__": "\x00\x01\x02\xff", "__headers__": map[string]string{"X-OP-Date": "x", "X-OP-Authorization": "x"}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Must not panic
			_, _ = p.VerifyAndParseWebhook(context.Background(), tc.payload)
		})
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func isWebhookError(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}
	return err.Error() == target.Error() || strings.Contains(err.Error(), target.Error())
}
