package onepay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"
)

// mustParseDate parses a DateLayout string or fails the test.
func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(DateLayout, s)
	if err != nil {
		t.Fatalf("mustParseDate(%q): %v", s, err)
	}
	return tm
}

// ---------------------------------------------------------------------------
// Credential
// ---------------------------------------------------------------------------

func TestCredential_Format(t *testing.T) {
	cases := []struct {
		partnerID string
		date      string // DateLayout input
		want      string
	}{
		{
			partnerID: "PARTNER01",
			date:      "20260108T112907Z",
			want:      "PARTNER01/20260108/onepay/onepayout/ows1_request",
		},
		{
			partnerID: "MERCHANT99",
			date:      "20260101T000000Z",
			want:      "MERCHANT99/20260101/onepay/onepayout/ows1_request",
		},
	}
	for _, tc := range cases {
		t.Run(tc.partnerID+"/"+tc.date, func(t *testing.T) {
			tm := mustParseDate(t, tc.date)
			got := Credential(tc.partnerID, tm)
			if got != tc.want {
				t.Fatalf("Credential = %q, want %q", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AuthorizationHeader
// ---------------------------------------------------------------------------

func TestAuthorizationHeader_Format(t *testing.T) {
	got := AuthorizationHeader("PARTNER01/20260108/onepay/onepayout/ows1_request", "accept;x-op-date", "abc123sig=")
	want := "OWS1-HMAC-SHA256 Credential=PARTNER01/20260108/onepay/onepayout/ows1_request,SignedHeaders=accept;x-op-date,Signature=abc123sig="
	if got != want {
		t.Fatalf("AuthorizationHeader = %q, want %q", got, want)
	}
}

func TestAuthorizationHeader_NoSpacesAfterCommas(t *testing.T) {
	got := AuthorizationHeader("cred", "sh", "sig")
	if strings.Contains(got, ", ") {
		t.Fatalf("AuthorizationHeader must NOT have spaces after commas, got %q", got)
	}
	if !strings.HasPrefix(got, "OWS1-HMAC-SHA256 ") {
		t.Fatalf("AuthorizationHeader must start with algorithm prefix, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// CanonicalRequest
// ---------------------------------------------------------------------------

func TestCanonicalRequest_GET_NoBody(t *testing.T) {
	// GET request with no body: body hash is SHA-256 of empty string.
	headers := map[string]string{
		"accept":       "application/json",
		"x-op-date":    "20260108T112907Z",
		"x-op-expires": "6000",
	}
	signedHeaders := []string{"accept", "x-op-date", "x-op-expires"}

	got, err := CanonicalRequest("GET", "https://mtf.onepay.vn/onepayout/api/v1/accounts/ACC01", headers, signedHeaders, nil)
	if err != nil {
		t.Fatalf("CanonicalRequest: %v", err)
	}

	// Must have exactly 6 lines: METHOD, URI, QUERY, HEADERS, SIGNED_HEADERS, BODY_HASH
	// +1 for the trailing newline in canonical headers (blank line before signed-headers-list).
	lines := strings.Split(got, "\n")
	// 3 (method,uri,qs) + N signed header lines + 1 (blank from trailing \n) + 2 (signed-headers-list, body-hash)
	expectedLines := 3 + len(signedHeaders) + 1 + 2
	if len(lines) != expectedLines {
		t.Fatalf("expected %d lines in canonical request, got %d:\n%s", expectedLines, len(lines), got)
	}
	if lines[0] != "GET" {
		t.Errorf("METHOD line = %q, want GET", lines[0])
	}
	// Empty query string
	if lines[2] != "" {
		t.Errorf("QUERY line = %q, want empty", lines[2])
	}
	// SHA-256 of empty string is well-known
	const emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if lines[len(lines)-1] != emptySHA256 {
		t.Errorf("body hash = %q, want %q", lines[len(lines)-1], emptySHA256)
	}
}

func TestCanonicalRequest_PUT_WithBody(t *testing.T) {
	body := []byte(`{"swift_code":"VCBVNVVX","amount":"50000"}`)
	headers := map[string]string{
		"accept":         "application/json",
		"content-type":   "application/json",
		"content-length": fmt.Sprintf("%d", len(body)),
		"x-op-date":      "20260108T112907Z",
		"x-op-expires":   "6000",
	}
	signedHeaders := []string{"accept", "content-type", "content-length", "x-op-date", "x-op-expires"}

	got, err := CanonicalRequest("PUT", "https://mtf.onepay.vn/onepayout/api/v1/accounts/ACC01/funds_transfers/FT001", headers, signedHeaders, body)
	if err != nil {
		t.Fatalf("CanonicalRequest: %v", err)
	}

	lines := strings.Split(got, "\n")
	expectedLines := 3 + len(signedHeaders) + 1 + 2
	if len(lines) != expectedLines {
		t.Fatalf("expected %d lines, got %d:\n%s", expectedLines, len(lines), got)
	}
	if lines[0] != "PUT" {
		t.Errorf("METHOD = %q, want PUT", lines[0])
	}
	// Body hash is deterministic SHA-256 of the body bytes
	bodyHash := fmt.Sprintf("%x", sha256.Sum256(body))
	if lines[len(lines)-1] != bodyHash {
		t.Errorf("body hash = %q, want %q", lines[len(lines)-1], bodyHash)
	}
}

func TestCanonicalRequest_SortsHeaders(t *testing.T) {
	// Headers must be sorted by lowercase name in canonical form.
	headers := map[string]string{
		"x-op-expires": "6000",
		"accept":       "application/json",
		"x-op-date":    "20260108T112907Z",
	}
	signedHeaders := []string{"accept", "x-op-date", "x-op-expires"}

	got, err := CanonicalRequest("GET", "https://mtf.onepay.vn/path", headers, signedHeaders, nil)
	if err != nil {
		t.Fatalf("CanonicalRequest: %v", err)
	}

	lines := strings.Split(got, "\n")
	// Canonical headers start at line index 3, sorted by name
	// 3 headers: accept, x-op-date, x-op-expires
	if !strings.HasPrefix(lines[3], "accept:") {
		t.Errorf("first header = %q, want accept first", lines[3])
	}
	if !strings.HasPrefix(lines[4], "x-op-date:") {
		t.Errorf("second header = %q, want x-op-date second", lines[4])
	}
	if !strings.HasPrefix(lines[5], "x-op-expires:") {
		t.Errorf("third header = %q, want x-op-expires third", lines[5])
	}
}

func TestCanonicalRequest_QueryStringSorted(t *testing.T) {
	// Query string keys must be sorted ASCII-lex with %20 encoding.
	headers := map[string]string{
		"accept":       "application/json",
		"x-op-date":    "20260108T112907Z",
		"x-op-expires": "6000",
	}
	signedHeaders := []string{"accept", "x-op-date", "x-op-expires"}

	got, err := CanonicalRequest("GET", "https://mtf.onepay.vn/path?zebra=z&alpha=a&mango=m", headers, signedHeaders, nil)
	if err != nil {
		t.Fatalf("CanonicalRequest: %v", err)
	}

	lines := strings.Split(got, "\n")
	if lines[2] != "alpha=a&mango=m&zebra=z" {
		t.Errorf("query string = %q, want sorted", lines[2])
	}
}

// ---------------------------------------------------------------------------
// StringToSign
// ---------------------------------------------------------------------------

func TestStringToSign_Format(t *testing.T) {
	tm := mustParseDate(t, "20260108T112907Z")
	canonical := "GET\n/path\n\nheaders\nsigned\nbodyhash"
	got := StringToSign(tm, "TESTPARTNER", canonical)

	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Fatalf("StringToSign should have 4 lines, got %d:\n%s", len(lines), got)
	}
	if lines[0] != "OWS1-HMAC-SHA256" {
		t.Errorf("line 1 = %q, want algorithm", lines[0])
	}
	if lines[1] != "20260108T112907Z" {
		t.Errorf("line 2 = %q, want X-OP-Date", lines[1])
	}
	if lines[2] != "20260108/onepay/onepayout/ows1_request" {
		t.Errorf("line 3 = %q, want credential scope (date/region/service/terminator)", lines[2])
	}
	// Line 4 is SHA-256 hex of the canonical request
	expected := fmt.Sprintf("%x", sha256.Sum256([]byte(canonical)))
	if lines[3] != expected {
		t.Errorf("line 4 = %q, want %q", lines[3], expected)
	}
}

// ---------------------------------------------------------------------------
// DerivedKey
// ---------------------------------------------------------------------------

func TestDerivedKey_Chain(t *testing.T) {
	// Verify the key derivation chain:
	//   kDate    = HMAC-SHA256("OWS1" + partnerKey, yyyyMMdd)
	//   kRegion  = HMAC-SHA256(kDate, "onepay")
	//   kService = HMAC-SHA256(kRegion, "onepayout")
	//   kSigning = HMAC-SHA256(kService, "ows1_request")
	partnerKey := "test-partner-secret"
	tm := mustParseDate(t, "20260108T112907Z")
	dateStr := "20260108"

	kDate := hmacSHA256([]byte("OWS1"+partnerKey), dateStr)
	kRegion := hmacSHA256(kDate, "onepay")
	kService := hmacSHA256(kRegion, "onepayout")
	kSigning := hmacSHA256(kService, "ows1_request")

	got := DerivedKey(partnerKey, tm)
	if string(got) != string(kSigning) {
		t.Fatalf("DerivedKey mismatch:\n  got  %x\n  want %x", got, kSigning)
	}
}

func TestDerivedKey_DifferentDateProducesDifferentKey(t *testing.T) {
	key1 := DerivedKey("secret", mustParseDate(t, "20260108T000000Z"))
	key2 := DerivedKey("secret", mustParseDate(t, "20260109T000000Z"))
	if string(key1) == string(key2) {
		t.Fatal("different dates should produce different derived keys")
	}
}

func TestDerivedKey_DifferentSecretProducesDifferentKey(t *testing.T) {
	tm := mustParseDate(t, "20260108T000000Z")
	key1 := DerivedKey("secret-a", tm)
	key2 := DerivedKey("secret-b", tm)
	if string(key1) == string(key2) {
		t.Fatal("different secrets should produce different derived keys")
	}
}

// ---------------------------------------------------------------------------
// Sign
// ---------------------------------------------------------------------------

func TestSign_ProducesValidHex(t *testing.T) {
	sts := "OWS1-HMAC-SHA256\n20260108T112907Z\nTESTPARTNER/20260108/onepay/onepayout/ows1_request\nsomehash"
	key := DerivedKey("test-secret", mustParseDate(t, "20260108T112907Z"))
	sig := Sign(sts, key)

	decoded, err := hex.DecodeString(sig)
	if err != nil {
		t.Fatalf("Sign output is not valid hex: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("HMAC-SHA256 output should be 32 bytes, got %d", len(decoded))
	}
}

func TestSign_Deterministic(t *testing.T) {
	sts := "test-string-to-sign"
	key := []byte("deterministic-key")
	sig1 := Sign(sts, key)
	sig2 := Sign(sts, key)
	if sig1 != sig2 {
		t.Fatalf("Sign should be deterministic, got %q and %q", sig1, sig2)
	}
}

func TestSign_DifferentKeyProducesDifferentSignature(t *testing.T) {
	sts := "test-string-to-sign"
	sig1 := Sign(sts, []byte("key-a"))
	sig2 := Sign(sts, []byte("key-b"))
	if sig1 == sig2 {
		t.Fatal("different keys should produce different signatures")
	}
}

func TestSign_MatchesManualHMAC(t *testing.T) {
	sts := "test-string-to-sign"
	key := []byte("test-key")

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(sts))
	expected := hex.EncodeToString(mac.Sum(nil))

	got := Sign(sts, key)
	if got != expected {
		t.Fatalf("Sign = %q, want %q", got, expected)
	}
}

// ---------------------------------------------------------------------------
// VerifyIPN
// ---------------------------------------------------------------------------

func TestVerifyIPN_ValidSignature(t *testing.T) {
	// Build a synthetic IPN with known values, sign it with full URL, then verify.
	partnerKey := "ipn-verify-secret"
	tm := mustParseDate(t, "20260108T112907Z")
	body := []byte(`{"transaction_id":"TXN001","funds_transfer_id":"FT001","state":"approved"}`)
	fullURL := "https://pay.1stop.app/api/v1/webhooks/disbursement/1pay"

	headers := map[string]string{
		"accept":         "application/json",
		"x-op-date":      tm.Format(DateLayout),
		"x-op-expires":   "6000",
		"content-type":   "application/json",
		"content-length": fmt.Sprintf("%d", len(body)),
	}
	signedHeaders := []string{"accept", "x-op-date", "x-op-expires"}

	// OnePay's IPN signer uses their internal gateway URL as the canonical URI;
	// VerifyIPN has the exact form hardcoded. Mirror it here so the test signs
	// what VerifyIPN expects.
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

	sts := StringToSign(tm, "TESTPARTNER", canonical)
	key := DerivedKey(partnerKey, tm)
	sig := Sign(sts, key)
	cred := Credential("TESTPARTNER", tm)

	headers["X-OP-Authorization"] = AuthorizationHeader(cred, strings.Join(signedHeaders, ";"), sig)

	// Verify should succeed
	if !VerifyIPN("PUT", fullURL, headers, body, "TESTPARTNER", partnerKey) {
		t.Fatal("VerifyIPN rejected a valid signature")
	}
}

func TestVerifyIPN_InvalidSignature(t *testing.T) {
	headers := map[string]string{
		"x-op-date":          "20260108T112907Z",
		"x-op-expires":       "6000",
		"X-OP-Authorization": "OWS1-HMAC-SHA256 Credential=x,SignedHeaders=x,Signature=TAMPERED",
	}
	body := []byte(`{"state":"approved"}`)

	if VerifyIPN("PUT", "https://pay.1stop.app/api/v1/webhooks/disbursement/1pay", headers, body, "partner", "secret") {
		t.Fatal("VerifyIPN accepted an invalid signature")
	}
}

func TestVerifyIPN_MissingAuthHeader(t *testing.T) {
	headers := map[string]string{
		"x-op-date":    "20260108T112907Z",
		"x-op-expires": "6000",
	}
	body := []byte(`{}`)

	if VerifyIPN("PUT", "https://pay.1stop.app/api/v1/webhooks/disbursement/1pay", headers, body, "partner", "secret") {
		t.Fatal("VerifyIPN should reject when X-OP-Authorization is missing")
	}
}

// ---------------------------------------------------------------------------
// Fixture-driven tests — skip until we have real sandbox fixtures
// ---------------------------------------------------------------------------

func TestCanonicalRequest_SandboxFixture(t *testing.T) {
	t.Skip("awaiting OnePay sandbox fixture")
}

func TestSign_SandboxFixture(t *testing.T) {
	t.Skip("awaiting OnePay sandbox fixture")
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
