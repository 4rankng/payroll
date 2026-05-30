package ninepay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// AuthorizationHeader formats the Authorization value 9pay's gateway
// expects on every signed request. Format taken verbatim from the 9pay
// Postman collection's request templates:
//
//	Signature Algorithm=HS256,Credential=<merchantKey>,SignedHeaders=,Signature=<sig>
//
// Comma separators carry NO trailing space — the Postman header value
// is "Signature Algorithm=HS256,Credential={{client_id}},SignedHeaders=,Signature={{signature}}".
// SignedHeaders is intentionally empty.
func AuthorizationHeader(merchantKey, signature string) string {
	return fmt.Sprintf(
		"Signature Algorithm=HS256,Credential=%s,SignedHeaders=,Signature=%s",
		merchantKey, signature,
	)
}

// OrderedParam is a single key/value pair in a request's parameter
// list. Despite the name, the slice's order is irrelevant to signing —
// CanonicalizeParams sorts keys alphabetically before emitting the
// canonical string. The struct shape is kept for callsite ergonomics
// and to preserve param order on the multipart wire.
type OrderedParam struct {
	Key   string
	Value string
}

// StringToSign builds the canonical request string per the 9pay
// Postman collection's pre-request script:
//
//	POST: METHOD + "\n" + URI + "\n" + TIMESTAMP + "\n" + QUERY
//	GET (no params): METHOD + "\n" + URI + "\n" + TIMESTAMP
//
// When httpQuery is empty (GET /disbursement/balance has no params),
// the trailing "\n" + query is omitted entirely — three lines, no
// trailing newline. This matches the Postman script:
//
//	var message = "GET" + "\n" + END_POINT + "/disbursement/balance" + "\n" + time;
//
// uri is the full URL including scheme, e.g.
// "https://payment.9pay.vn/disbursement/create". The docs example at
// https://developers.9pay.vn/danh-sach-api/quy-tac-tich-hop shows the
// full URL with scheme in the URI slot, and the Postman pre-request
// script concatenates END_POINT (which holds the base URL with scheme)
// with the path.
func StringToSign(method, uri, httpQuery, timestamp string) string {
	if httpQuery == "" {
		return strings.Join([]string{method, uri, timestamp}, "\n")
	}
	return strings.Join([]string{method, uri, timestamp, httpQuery}, "\n")
}

// Sign computes the base64-encoded HMAC-SHA256 of the canonical
// request string per 9pay's Postman pre-request script:
//
//	stringToSign := METHOD + "\n" + URI + "\n" + TIMESTAMP [+ "\n" + QUERY]
//	signature    := base64(HMAC-SHA256(stringToSign, merchantSecret))
//
// timestamp must match the value sent in the Date header (10-digit
// unix seconds). httpQuery is CanonicalizeParams(...) output (or
// empty for GETs with no body). uri is host+path.
//
// Output uses standard base64 with padding (CryptoJS.enc.Base64.stringify
// in the Postman script), NOT base64url.
func Sign(method, uri, httpQuery, timestamp string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(StringToSign(method, uri, httpQuery, timestamp)))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// CanonicalizeParams renders the parameter list as the canonical query
// string the Postman script feeds into the HMAC:
//
//	Object.keys(params).sort().map(k => encodeURIComponent(k) + '=' + encodeURIComponent(v)).join('&').replace(/%20/g, '+')
//
// In Go terms:
//   - Keys sorted ascending (lexicographic, byte-wise)
//   - Each key and value URL-encoded with encodeURIComponent semantics
//   - Spaces emitted as '+', not '%20'
//   - Pairs joined with '&'
//
// Empty values are kept (the JS Postman script does not skip them).
// Returns "" for empty input.
func CanonicalizeParams(params []OrderedParam) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	values := make(map[string]string, len(params))
	for _, p := range params {
		keys = append(keys, p.Key)
		values[p.Key] = p.Value
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, encodeURIComponent(k)+"="+encodeURIComponent(values[k]))
	}
	return strings.Join(parts, "&")
}

// encodeURIComponent matches JavaScript's encodeURIComponent followed
// by .replace(/%20/g, "+"), which is what the 9pay Postman script does.
//
// Go's url.QueryEscape is close — spaces already become '+' — but it
// also percent-encodes !*'() which encodeURIComponent leaves alone.
// We undo those five so our canonical string matches the JS reference
// byte-for-byte. The 9pay HMAC has no tolerance for any difference.
func encodeURIComponent(s string) string {
	encoded := url.QueryEscape(s)
	r := strings.NewReplacer(
		"%21", "!",
		"%27", "'",
		"%28", "(",
		"%29", ")",
		"%2A", "*",
	)
	return r.Replace(encoded)
}

// VerifyChecksum implements 9pay's IPN/webhook checksum validation.
// Per the integration rules:
//
//	expected := strtoupper(sha256(result + secretKeyCheckSum))
//
// Note this is plain SHA-256 (not HMAC) and uses a separate
// secretKeyCheckSum, distinct from the merchant secret used to sign
// outbound requests. result is the raw base64 string from the IPN
// payload (do not decode it before hashing).
func VerifyChecksum(result, secretKeyCheckSum, received string) bool {
	sum := sha256.Sum256([]byte(result + secretKeyCheckSum))
	expected := strings.ToUpper(hex.EncodeToString(sum[:]))
	return hmac.Equal([]byte(expected), []byte(strings.ToUpper(received)))
}
