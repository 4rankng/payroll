package ninepay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
)

func TestCanonicalizeParams_SortsAlphabetically(t *testing.T) {
	// The 9pay Postman pre-request script sorts keys with
	// Object.keys(params).sort() before joining. Insertion order is
	// irrelevant — only the sorted order is signed.
	got := CanonicalizeParams([]OrderedParam{
		{Key: "zebra", Value: "z"},
		{Key: "alpha", Value: "a"},
		{Key: "mango", Value: "m"},
	})
	want := "alpha=a&mango=m&zebra=z"
	if got != want {
		t.Fatalf("CanonicalizeParams = %q, want %q (sorted alphabetically)", got, want)
	}
}

func TestCanonicalizeParams_PostmanCheckAccountFixture(t *testing.T) {
	// Locked against the Postman collection's check-account example.
	// The script sorts keys: account_no, account_type, bank_code, request_id.
	params := []OrderedParam{
		{Key: "request_id", Value: "1996359677"},
		{Key: "bank_code", Value: "9PAY"},
		{Key: "account_no", Value: "0888523111"},
		{Key: "account_type", Value: "0"},
	}
	got := CanonicalizeParams(params)
	want := "account_no=0888523111&account_type=0&bank_code=9PAY&request_id=1996359677"
	if got != want {
		t.Fatalf("CanonicalizeParams = %q, want %q", got, want)
	}
}

func TestCanonicalizeParams_URLEncodesValues(t *testing.T) {
	// encodeURIComponent + replace %20 with + — spaces become '+', not %20.
	got := CanonicalizeParams([]OrderedParam{
		{Key: "description", Value: "Test chi ho"},
	})
	want := "description=Test+chi+ho"
	if got != want {
		t.Fatalf("CanonicalizeParams = %q, want %q (spaces as +)", got, want)
	}
}

func TestCanonicalizeParams_PreservesJSEncodeURIComponentSpecials(t *testing.T) {
	// encodeURIComponent leaves !*'() unencoded. url.QueryEscape encodes
	// them, so our wrapper must un-escape these five back to literal.
	got := CanonicalizeParams([]OrderedParam{
		{Key: "v", Value: "!*'()"},
	})
	want := "v=!*'()"
	if got != want {
		t.Fatalf("CanonicalizeParams = %q, want %q (JS encodeURIComponent specials)", got, want)
	}
}

func TestCanonicalizeParams_EmptyInput(t *testing.T) {
	if got := CanonicalizeParams(nil); got != "" {
		t.Fatalf("CanonicalizeParams(nil) = %q, want empty", got)
	}
	if got := CanonicalizeParams([]OrderedParam{}); got != "" {
		t.Fatalf("CanonicalizeParams(empty) = %q, want empty", got)
	}
}

func TestStringToSign_POST_FourLines(t *testing.T) {
	// POST: METHOD\nURI\nTIMESTAMP\nQUERY (four lines, no trailing newline).
	got := StringToSign("POST", "https://sand-payment.9pay.vn/disbursement/check-account",
		"account_no=0888523111&account_type=0&bank_code=9PAY&request_id=1996359677",
		"1715050000")
	want := "POST\nhttps://sand-payment.9pay.vn/disbursement/check-account\n1715050000\naccount_no=0888523111&account_type=0&bank_code=9PAY&request_id=1996359677"
	if got != want {
		t.Fatalf("StringToSign(POST) = %q, want %q", got, want)
	}
}

func TestStringToSign_GET_NoQueryThreeLines(t *testing.T) {
	// GET /disbursement/balance has no params. The Postman script emits
	// exactly three lines (METHOD\nURI\nTIMESTAMP) — no trailing \n,
	// no empty fourth line.
	got := StringToSign("GET", "https://sand-payment.9pay.vn/disbursement/balance", "", "1715050000")
	want := "GET\nhttps://sand-payment.9pay.vn/disbursement/balance\n1715050000"
	if got != want {
		t.Fatalf("StringToSign(GET, empty query) = %q, want %q", got, want)
	}
}

func TestSign_PostmanCheckAccountFixture(t *testing.T) {
	// End-to-end fixture from the Postman collection.
	// Inputs picked deterministic so the literal expected sig can be
	// regenerated via openssl.
	const (
		method    = "POST"
		uri       = "https://sand-payment.9pay.vn/disbursement/check-account"
		timestamp = "1715050000"
	)
	secret := []byte("test-secret-key-fixed")

	canonical := CanonicalizeParams([]OrderedParam{
		{Key: "request_id", Value: "1996359677"},
		{Key: "bank_code", Value: "9PAY"},
		{Key: "account_no", Value: "0888523111"},
		{Key: "account_type", Value: "0"},
	})

	got := Sign(method, uri, canonical, timestamp, secret)

	// Independent reference computation matching the Postman recipe:
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(method + "\n" + uri + "\n" + timestamp + "\n" + canonical))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if got != want {
		t.Fatalf("Sign(check-account fixture) = %q, want %q", got, want)
	}
	// Locks in the literal output. Regenerate via:
	//   printf 'POST\nhttps://sand-payment.9pay.vn/disbursement/check-account\n1715050000\naccount_no=0888523111&account_type=0&bank_code=9PAY&request_id=1996359677' | \
	//     openssl dgst -sha256 -hmac 'test-secret-key-fixed' -binary | base64
	const expectedSignature = "IUK/PFufU0MIw7i0TGT0cHBRO0oUZrOmLmRfYc/6w/4="
	if got != expectedSignature {
		t.Fatalf("Sign literal = %q, want %q", got, expectedSignature)
	}
}

func TestSign_PostmanBalanceFixture_GETHasNoQueryLine(t *testing.T) {
	// GET /disbursement/balance: 3-line message, no query.
	const (
		method    = "GET"
		uri       = "https://sand-payment.9pay.vn/disbursement/balance"
		timestamp = "1715050000"
	)
	secret := []byte("test-secret-key-fixed")

	got := Sign(method, uri, "", timestamp, secret)

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(method + "\n" + uri + "\n" + timestamp))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if got != want {
		t.Fatalf("Sign(balance fixture) = %q, want %q", got, want)
	}
}

func TestSign_UsesStandardBase64WithPadding(t *testing.T) {
	// CryptoJS.enc.Base64.stringify produces standard base64 with '+'/'/' and '=' padding.
	got := Sign("POST", "https://sand-payment.9pay.vn/x", "k=v", "1700000000", []byte("secret"))

	if _, err := base64.StdEncoding.DecodeString(got); err != nil {
		t.Fatalf("Sign output %q is not valid standard base64: %v", got, err)
	}
	// HMAC-SHA256 = 32 bytes → 44 base64 chars (with padding).
	if len(got) != 44 {
		t.Fatalf("Sign output length = %d, want 44", len(got))
	}
}

func TestSign_DifferentSecretProducesDifferentSignature(t *testing.T) {
	a := Sign("POST", "https://sand-payment.9pay.vn/x", "k=v", "1", []byte("a"))
	b := Sign("POST", "https://sand-payment.9pay.vn/x", "k=v", "1", []byte("b"))
	if a == b {
		t.Fatalf("expected different signatures for different secrets")
	}
}

func TestAuthorizationHeader_NoSpacesAfterCommas(t *testing.T) {
	// Postman header value is literal:
	//   Signature Algorithm=HS256,Credential={{client_id}},SignedHeaders=,Signature={{signature}}
	got := AuthorizationHeader("MERCH123", "SIG456")
	want := "Signature Algorithm=HS256,Credential=MERCH123,SignedHeaders=,Signature=SIG456"
	if got != want {
		t.Fatalf("AuthorizationHeader = %q, want %q", got, want)
	}
}

func TestVerifyChecksum_ValidAndInvalid(t *testing.T) {
	result := "abc123base64result"
	checkSecret := "secret-checksum-key"

	sum := sha256.Sum256([]byte(result + checkSecret))
	valid := strings.ToUpper(hex.EncodeToString(sum[:]))

	if !VerifyChecksum(result, checkSecret, valid) {
		t.Fatalf("VerifyChecksum rejected a valid checksum")
	}
	if !VerifyChecksum(result, checkSecret, strings.ToLower(valid)) {
		t.Fatalf("VerifyChecksum should be case-insensitive on the received value")
	}
	if VerifyChecksum(result, checkSecret, valid+"00") {
		t.Fatalf("VerifyChecksum accepted a corrupted checksum")
	}
	if VerifyChecksum(result, "wrong-secret", valid) {
		t.Fatalf("VerifyChecksum accepted with wrong secret")
	}
}
