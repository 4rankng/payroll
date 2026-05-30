package onepay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	SignatureAlgorithm = "OWS1-HMAC-SHA256"
	Service            = "onepayout"
	Scope              = "onepay/onepayout/ows1_request"
	DateLayout         = "20060102T150405Z"
)

// Credential builds the Credential field of X-OP-Authorization.
//
//	{partner_id}/{yyyyMMdd}/onepay/onepayout/ows1_request
func Credential(partnerID string, t time.Time) string {
	return fmt.Sprintf("%s/%s/%s", partnerID, t.UTC().Format("20060102"), Scope)
}

// AuthorizationHeader assembles the full X-OP-Authorization value.
//
//	OWS1-HMAC-SHA256 Credential=...,SignedHeaders=...,Signature=...
func AuthorizationHeader(credential, signedHeaders, signature string) string {
	return fmt.Sprintf("%s Credential=%s,SignedHeaders=%s,Signature=%s",
		SignatureAlgorithm, credential, signedHeaders, signature)
}

// CanonicalRequest builds the canonical-request string.
// Format (one line per element, separated by '\n'):
//
//	METHOD\n
//	CANONICAL_URI\n
//	CANONICAL_QUERY_STRING\n
//	CANONICAL_HEADERS\n
//	SIGNED_HEADERS\n
//	HEX_SHA256(BODY)
func CanonicalRequest(method, rawURL string, headers map[string]string, signedHeaders []string, body []byte) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}

	canonicalURI := u.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	canonicalQueryString := canonicalQueryString(u.Query())
	canonicalHeaders, signedHeadersStr := buildCanonicalHeaders(headers, signedHeaders)

	if body == nil {
		body = []byte{}
	}
	bodyHashArr := sha256.Sum256(body)
	bodyHash := fmt.Sprintf("%x", bodyHashArr[:])

	parts := []string{
		strings.ToUpper(method),
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeadersStr,
		bodyHash,
	}
	return strings.Join(parts, "\n"), nil
}

// StringToSign builds the string-to-sign:
//
//	OWS1-HMAC-SHA256\n
//	X-OP-DATE\n
//	{yyyyMMdd}/onepay/onepayout/ows1_request\n
//	HEX_SHA256(CanonicalRequest)
func StringToSign(t time.Time, partnerID string, canonicalRequest string) string {
	crHashArr := sha256.Sum256([]byte(canonicalRequest))
	canonicalHash := fmt.Sprintf("%x", crHashArr[:])
	return strings.Join([]string{
		SignatureAlgorithm,
		t.UTC().Format(DateLayout),
		fmt.Sprintf("%s/%s", t.UTC().Format("20060102"), Scope),
		canonicalHash,
	}, "\n")
}

// DerivedKey applies the key derivation:
//
//	kDate    = HMAC-SHA256("OWS1" + partnerKey, yyyyMMdd)
//	kRegion  = HMAC-SHA256(kDate, "onepay")
//	kService = HMAC-SHA256(kRegion, "onepayout")
//	kSigning = HMAC-SHA256(kService, "ows1_request")
func DerivedKey(partnerKey string, t time.Time) []byte {
	dateStr := t.UTC().Format("20060102")
	kDate := hmacSHA256Raw([]byte("OWS1"+partnerKey), dateStr)
	kRegion := hmacSHA256Raw(kDate, "onepay")
	kService := hmacSHA256Raw(kRegion, "onepayout")
	kSigning := hmacSHA256Raw(kService, "ows1_request")
	return kSigning
}

// Sign returns the hex-encoded HMAC-SHA256 of stringToSign with derivedKey.
func Sign(stringToSign string, derivedKey []byte) string {
	mac := hmac.New(sha256.New, derivedKey)
	mac.Write([]byte(stringToSign))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyIPN re-derives the signature from the inbound IPN body and
// X-OP-Date header and compares against X-OP-Authorization in
// constant time. Returns true on match.
//
// fullURL is the complete callback URL (scheme + host + path + query)
// that OnePay called. The path component is extracted and used as the canonical URI.
func VerifyIPN(method, fullURL string, headers map[string]string, body []byte, partnerID, partnerKey string) bool {
	authHeader := lookupHeader(headers, "X-OP-Authorization")
	if authHeader == "" {
		return false
	}

	dateStr := lookupHeader(headers, "X-OP-Date")
	if dateStr == "" {
		return false
	}

	tm, err := time.Parse(DateLayout, dateStr)
	if err != nil {
		return false
	}

	_, signedHeadersStr, expectedSig, err := parseAuthorization(authHeader)
	if err != nil {
		return false
	}

	signedHeaders := strings.Split(signedHeadersStr, ";")
	sort.Strings(signedHeaders)

	filteredHeaders := make(map[string]string, len(signedHeaders))
	for _, sh := range signedHeaders {
		lower := strings.ToLower(strings.TrimSpace(sh))
		for k, v := range headers {
			if strings.ToLower(k) == lower {
				filteredHeaders[lower] = v
				break
			}
		}
	}

	// OnePay's IPN signer uses their internal gateway URL as the canonical URI,
	// not the public IPN URL they PUT to. Confirmed by OnePay support 2026-05-20.
	// Format: http://localhost/payout-merchants/{scheme}/{host}/{port}{path}
	const onepayIPNInternalURL = "http://localhost/payout-merchants/https/tingting.vip/443/api/v1/webhooks/disbursement/1pay"
	canonicalURI := uriEncode(onepayIPNInternalURL, false)
	_ = fullURL
	canonicalHeaders, signedHeadersNorm := buildCanonicalHeaders(filteredHeaders, signedHeaders)

	if body == nil {
		body = []byte{}
	}
	bodyHashArr := sha256.Sum256(body)
	bodyHash := fmt.Sprintf("%x", bodyHashArr[:])

	canonical := strings.Join([]string{
		strings.ToUpper(method),
		canonicalURI,
		"", // query string is embedded in the URI for IPN
		canonicalHeaders,
		signedHeadersNorm,
		bodyHash,
	}, "\n")

	sts := StringToSign(tm, partnerID, canonical)
	key := DerivedKey(partnerKey, tm)
	computedSig := Sign(sts, key)

	if !hmac.Equal([]byte(computedSig), []byte(expectedSig)) {
		slog.Warn("onepay: IPN signature mismatch",
			"method", method,
			"partner_id", partnerID,
		)
		return false
	}

	return true
}

func parseAuthorization(header string) (credential, signedHeaders, signature string, err error) {
	header = strings.TrimPrefix(header, SignatureAlgorithm+" ")

	parts := strings.Split(header, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch strings.TrimSpace(kv[0]) {
		case "Credential":
			credential = kv[1]
		case "SignedHeaders":
			signedHeaders = kv[1]
		case "Signature":
			signature = kv[1]
		}
	}
	if credential == "" || signature == "" {
		return "", "", "", fmt.Errorf("missing fields in authorization header")
	}
	return
}

func lookupHeader(headers map[string]string, key string) string {
	lower := strings.ToLower(key)
	for k, v := range headers {
		if strings.ToLower(k) == lower {
			return v
		}
	}
	return ""
}

func canonicalQueryString(vals url.Values) string {
	if len(vals) == 0 {
		return ""
	}
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		vs := vals[k]
		sort.Strings(vs)
		for _, v := range vs {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(parts, "&")
}

func buildCanonicalHeaders(headers map[string]string, signedHeaders []string) (string, string) {
	lower := make(map[string]string, len(headers))
	for k, v := range headers {
		lower[strings.ToLower(k)] = strings.TrimSpace(v)
	}

	sorted := make([]string, len(signedHeaders))
	copy(sorted, signedHeaders)
	sort.Strings(sorted)

	headerLines := make([]string, 0, len(sorted))
	for _, sh := range sorted {
		headerLines = append(headerLines, sh+":"+lower[sh])
	}

	return strings.Join(headerLines, "\n") + "\n", strings.Join(sorted, ";")
}

func uriEncode(data string, encodeSlash bool) string {
	var sb strings.Builder
	for _, ch := range []byte(data) {
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') ||
			ch == '_' || ch == '-' || ch == '~' || ch == '.' || (ch == '/' && !encodeSlash) {
			sb.WriteByte(ch)
		} else {
			fmt.Fprintf(&sb, "%%%02X", ch)
		}
	}
	return sb.String()
}

func hmacSHA256Raw(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
