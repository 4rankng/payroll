package ninepay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
)

type OrderedParam struct {
	Key, Value string
}

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

func stringToSign(method, uri, httpQuery, timestamp string) string {
	if httpQuery == "" {
		return strings.Join([]string{method, uri, timestamp}, "\n")
	}
	return strings.Join([]string{method, uri, timestamp, httpQuery}, "\n")
}

func sign(method, uri, httpQuery, timestamp string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(stringToSign(method, uri, httpQuery, timestamp)))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func IpnChecksum(result, secretKeyChecksum string) string {
	sum := sha256.Sum256([]byte(result + secretKeyChecksum))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func ChunkBase64(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 2*(len(s)/n+1))
	for i := 0; i < len(s); i += n {
		end := min(i+n, len(s))
		b.WriteString(s[i:end])
		b.WriteString("\r\n")
	}
	return b.String()
}
