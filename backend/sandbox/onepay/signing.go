package onepay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	sandboxAlgo    = "OWS1-HMAC-SHA256"
	sandboxService = "onepayout"
	sandboxScope   = "onepay/onepayout/ows1_request"
)

func deriveSigningKey(partnerKey, date string) []byte {
	kDate := hmacRaw([]byte("OWS1"+partnerKey), date)
	kRegion := hmacRaw(kDate, "onepay")
	kService := hmacRaw(kRegion, sandboxService)
	kSigning := hmacRaw(kService, "ows1_request")
	return kSigning
}

func hmacRaw(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func hexSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func buildCanonicalRequest(method, reqPath, queryString string, headers http.Header, signedHeaders []string, body []byte) string {
	canonicalHeaders, signedHeadersStr := buildCanonicalHeadersFromHTTP(headers, signedHeaders)
	parts := []string{
		strings.ToUpper(method), reqPath, queryString,
		canonicalHeaders, signedHeadersStr, hexSHA256(body),
	}
	return strings.Join(parts, "\n")
}

func buildCanonicalHeadersFromHTTP(headers http.Header, signedHeaders []string) (string, string) {
	lower := make(map[string]string)
	for k, vs := range headers {
		lower[strings.ToLower(k)] = strings.TrimSpace(vs[0])
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

// VerifyOWSSignature checks the OWS1 signature on an inbound request.
func VerifyOWSSignature(r *http.Request, partnerID, partnerKey string) error {
	authHeader := r.Header.Get("X-OP-Authorization")
	if authHeader == "" {
		return fmt.Errorf("missing X-OP-Authorization header")
	}
	dateHeader := r.Header.Get("X-OP-Date")
	if dateHeader == "" {
		return fmt.Errorf("missing X-OP-Date header")
	}
	if len(dateHeader) < 8 {
		return fmt.Errorf("invalid X-OP-Date header: too short")
	}
	date := dateHeader[:8]

	cred, signedHeadersStr, sig, err := parseAuth(authHeader)
	if err != nil {
		return fmt.Errorf("parse authorization: %w", err)
	}

	expectedCred := fmt.Sprintf("%s/%s/%s", partnerID, date, sandboxScope)
	if cred != expectedCred {
		return fmt.Errorf("credential mismatch: expected %s got %s", expectedCred, cred)
	}

	body := []byte{}
	if r.Method == http.MethodPut || r.Method == http.MethodPost {
		if b, ok := r.Context().Value(CtxKeyBody).([]byte); ok {
			body = b
		}
	}

	signedHeaders := strings.Split(signedHeadersStr, ";")
	if signedHeadersStr == "" {
		signedHeaders = nil
	}

	queryString := canonicalQS(r.URL.Query())
	cr := buildCanonicalRequest(r.Method, r.URL.Path, queryString, r.Header, signedHeaders, body)

	scope := fmt.Sprintf("%s/%s", date, sandboxScope)
	sts := strings.Join([]string{sandboxAlgo, dateHeader, scope, hexSHA256([]byte(cr))}, "\n")

	key := deriveSigningKey(partnerKey, date)
	expectedSig := hex.EncodeToString(hmacRaw(key, sts))

	if !hmac.Equal([]byte(expectedSig), []byte(sig)) {
		return fmt.Errorf("signature mismatch: want %s got %s", expectedSig, sig)
	}
	return nil
}

func parseAuth(header string) (credential, signedHeaders, signature string, err error) {
	header = strings.TrimPrefix(header, sandboxAlgo+" ")
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

func canonicalQS(vals url.Values) string {
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

func signOutboundOWS(method, fullURL string, body []byte, partnerID, partnerKey string, now time.Time) (opDate, opExpires, opAuth string) {
	timestamp := now.UTC().Format("20060102T150405Z")
	date := now.UTC().Format("20060102")
	cred := fmt.Sprintf("%s/%s/%s", partnerID, date, sandboxScope)

	signedHeaders := []string{"accept", "x-op-date", "x-op-expires"}
	headers := http.Header{}
	headers.Set("Accept", "application/json")
	headers.Set("X-OP-Date", timestamp)
	headers.Set("X-OP-Expires", "6000")

	canonicalURI := sandboxURIEncode(fullURL, false)
	canonicalHeaders, signedHeadersNorm := buildCanonicalHeadersFromHTTP(headers, signedHeaders)
	cr := strings.Join([]string{
		strings.ToUpper(method), canonicalURI, "",
		canonicalHeaders, signedHeadersNorm, hexSHA256(body),
	}, "\n")

	scope := fmt.Sprintf("%s/%s", date, sandboxScope)
	sts := strings.Join([]string{sandboxAlgo, timestamp, scope, hexSHA256([]byte(cr))}, "\n")

	key := deriveSigningKey(partnerKey, date)
	sig := hex.EncodeToString(hmacRaw(key, sts))

	signedHeadersStr := "accept;x-op-date;x-op-expires"
	auth := fmt.Sprintf("%s Credential=%s,SignedHeaders=%s,Signature=%s",
		sandboxAlgo, cred, signedHeadersStr, sig)

	return timestamp, "6000", auth
}

func sandboxURIEncode(data string, encodeSlash bool) string {
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

type ctxBodyKeyType struct{}

var CtxKeyBody = ctxBodyKeyType{}
