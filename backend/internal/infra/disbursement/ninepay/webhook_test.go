package ninepay

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"api-server/internal/domain/ports/infrastructure"
)

func sealWebhook(t *testing.T, body string, secret string) (resultB64, checksum string) {
	t.Helper()
	resultB64 = base64.StdEncoding.EncodeToString([]byte(body))
	sum := sha256.Sum256([]byte(resultB64 + secret))
	checksum = strings.ToUpper(hex.EncodeToString(sum[:]))
	return
}

func TestParseAndVerifyWebhook_Success(t *testing.T) {
	body := `{
		"status": 5,
		"error_code": "",
		"message": "ok",
		"method": "DISBURSEMENT",
		"payment_no": "99887766",
		"amount": 50000,
		"description": "salary",
		"bank_code": "VCB",
		"account_name": "NGUYEN VAN A",
		"account_no": "1023020330000",
		"created_at": "2026-05-06 09:00:00",
		"request_id": "REQ-001"
	}`
	const secret = "ipn-secret"
	result, checksum := sealWebhook(t, body, secret)

	ev, err := parseAndVerifyWebhook(map[string]any{
		"result":   result,
		"checksum": checksum,
		"version":  "v1",
	}, secret)
	if err != nil {
		t.Fatalf("parseAndVerifyWebhook: %v", err)
	}

	if ev.Status != infrastructure.TransferStatusSuccess {
		t.Errorf("Status = %q, want success", ev.Status)
	}
	if ev.RequestID != "REQ-001" {
		t.Errorf("RequestID = %q", ev.RequestID)
	}
	if ev.ProviderRef != "99887766" {
		t.Errorf("ProviderRef = %q", ev.ProviderRef)
	}
	if ev.Amount != 50000 {
		t.Errorf("Amount = %d", ev.Amount)
	}
	if ev.Method != "DISBURSEMENT" {
		t.Errorf("Method = %q", ev.Method)
	}
}

func TestParseAndVerifyWebhook_RejectsTamperedChecksum(t *testing.T) {
	const secret = "ipn-secret"
	result, checksum := sealWebhook(t, `{"status":5}`, secret)
	tampered := checksum[:len(checksum)-2] + "00"

	_, err := parseAndVerifyWebhook(map[string]any{
		"result":   result,
		"checksum": tampered,
	}, secret)
	if !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Fatalf("err = %v, want ErrInvalidWebhookSignature", err)
	}
}

func TestParseAndVerifyWebhook_RejectsWrongSecret(t *testing.T) {
	result, checksum := sealWebhook(t, `{"status":5}`, "real-secret")
	_, err := parseAndVerifyWebhook(map[string]any{
		"result":   result,
		"checksum": checksum,
	}, "different-secret")
	if !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Fatalf("err = %v, want ErrInvalidWebhookSignature", err)
	}
}

func TestParseAndVerifyWebhook_MissingFields(t *testing.T) {
	_, err := parseAndVerifyWebhook(map[string]any{"result": "x"}, "s")
	if !errors.Is(err, ErrMalformedWebhook) {
		t.Fatalf("err = %v, want ErrMalformedWebhook", err)
	}
	_, err = parseAndVerifyWebhook(map[string]any{"checksum": "x"}, "s")
	if !errors.Is(err, ErrMalformedWebhook) {
		t.Fatalf("err = %v, want ErrMalformedWebhook", err)
	}
}

func TestParseAndVerifyWebhook_BadBase64(t *testing.T) {
	bad := "!!!not-base64!!!"
	const secret = "s"
	sum := sha256.Sum256([]byte(bad + secret))
	checksum := strings.ToUpper(hex.EncodeToString(sum[:]))

	_, err := parseAndVerifyWebhook(map[string]any{"result": bad, "checksum": checksum}, secret)
	if !errors.Is(err, ErrMalformedWebhook) {
		t.Fatalf("err = %v, want ErrMalformedWebhook", err)
	}
}

// TestParseAndVerifyWebhook_LineWrappedBase64 covers the byte-676/688/692
// production failure: 9pay (or some senders proxying through PHP
// chunk_split) emit MIME-style line-wrapped base64. The checksum verifies
// against the wrapped bytes, but base64.StdEncoding.DecodeString rejects
// the embedded \r\n. The fix is decodeFlexibleBase64, which strips
// whitespace before decoding.
func TestParseAndVerifyWebhook_LineWrappedBase64(t *testing.T) {
	// Build an IPN payload large enough to span multiple wrap lines.
	body := `{
		"status": 5,
		"error_code": "",
		"message": "ok",
		"method": "DISBURSEMENT",
		"payment_no": "99887766",
		"amount": 50000,
		"description": "salary disbursement for employee with a longer description",
		"bank_code": "BIDV",
		"account_name": "NGUYEN VAN A",
		"account_no": "1023020330000",
		"created_at": "2026-05-07 09:00:00",
		"request_id": "REQ-LINEWRAP-001",
		"failure_reason": ""
	}`
	const secret = "ipn-secret"
	raw := base64.StdEncoding.EncodeToString([]byte(body))

	// Inject CRLF every 76 chars (PHP chunk_split default).
	wrapped := wrapAt(raw, 76, "\r\n")
	if !strings.Contains(wrapped, "\r\n") {
		t.Fatalf("test fixture must contain CRLF wraps, got %q", wrapped)
	}

	// Checksum is computed over the WRAPPED string — that's the byte
	// sequence on the wire, and the prior raw-string SHA256 path would
	// otherwise reject this payload before we even reached base64 decode.
	sum := sha256.Sum256([]byte(wrapped + secret))
	checksum := strings.ToUpper(hex.EncodeToString(sum[:]))

	ev, err := parseAndVerifyWebhook(map[string]any{
		"result":   wrapped,
		"checksum": checksum,
	}, secret)
	if err != nil {
		t.Fatalf("parseAndVerifyWebhook on line-wrapped base64: %v", err)
	}
	if ev.RequestID != "REQ-LINEWRAP-001" {
		t.Errorf("RequestID = %q", ev.RequestID)
	}
	if ev.Status != infrastructure.TransferStatusSuccess {
		t.Errorf("Status = %q, want success", ev.Status)
	}
}

// TestDecodeFlexibleBase64_Variants exercises every encoding variant the
// flexible decoder is expected to accept. Run as a single table so any
// regression flags clearly which variant broke.
func TestDecodeFlexibleBase64_Variants(t *testing.T) {
	plain := []byte(`{"status":5,"request_id":"R-1"}`)
	std := base64.StdEncoding.EncodeToString(plain)
	urlSafe := base64.URLEncoding.EncodeToString(plain)
	rawStd := base64.RawStdEncoding.EncodeToString(plain)

	cases := []struct {
		name string
		in   string
	}{
		{"std with padding (baseline)", std},
		{"std with leading whitespace", "  \t" + std},
		{"std with trailing whitespace", std + "\n  "},
		{"std with embedded crlf wrap", wrapAt(std, 8, "\r\n")},
		{"std with embedded lf wrap", wrapAt(std, 8, "\n")},
		{"std wrapped in double quotes", `"` + std + `"`},
		{"std with bom prefix", "\ufeff" + std},
		{"url-safe alphabet", urlSafe},
		{"raw std (no padding)", rawStd},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := decodeFlexibleBase64(c.in)
			if err != nil {
				t.Fatalf("decodeFlexibleBase64 failed: %v (input=%q)", err, c.in)
			}
			if string(out) != string(plain) {
				t.Errorf("decode mismatch:\n  got  %q\n  want %q", out, plain)
			}
		})
	}
}

// TestDecodeFlexibleBase64_RejectsTrueGarbage confirms the decoder still
// errors on input that is not base64 in any variant.
func TestDecodeFlexibleBase64_RejectsTrueGarbage(t *testing.T) {
	if _, err := decodeFlexibleBase64("!!!definitely-not-base64-at-all!!!"); err == nil {
		t.Fatal("expected error for garbage input, got nil")
	}
}

// wrapAt inserts sep every n runes — used to reproduce MIME chunk_split
// style line-wrapped base64 payloads in tests.
func wrapAt(s string, n int, sep string) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i += n {
		end := i + n
		if end > len(s) {
			end = len(s)
		}
		b.WriteString(s[i:end])
		if end < len(s) {
			b.WriteString(sep)
		}
	}
	return b.String()
}

func TestIsSuccessCode(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", true},
		{"000", true},
		{"0", false},
		{"1004", false},
		{"702", false},
	}
	for _, c := range cases {
		if got := isSuccessCode(c.in); got != c.want {
			t.Errorf("isSuccessCode(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestTranslateStatus(t *testing.T) {
	cases := []struct {
		name          string
		status        int
		code          string
		failureReason string
		message       string
		want          infrastructure.TransferStatus
	}{
		{"success-empty-code", 5, "", "", "", infrastructure.TransferStatusSuccess},
		{"success-000-code", 5, "000", "", "", infrastructure.TransferStatusSuccess},
		{"failed-account-not-found", 2, "1001", "", "", infrastructure.TransferStatusFailed},
		{"failed-duplicate", 6, "702", "", "", infrastructure.TransferStatusFailed},
		{"failed-data-invalid", 6, "318", "", "", infrastructure.TransferStatusFailed},
		{"pending-zero-status", 0, "", "", "", infrastructure.TransferStatusPending},
		{"pending-no-error-code", 2, "", "", "", infrastructure.TransferStatusPending},
		{"pending-000-code", 2, "000", "", "", infrastructure.TransferStatusPending},
		{"reversed-failure-reason-vi", 2, "", "Hoàn chuyển tiền do tài khoản không hợp lệ", "", infrastructure.TransferStatusReversed},
		{"reversed-message-vi", 2, "", "", "Giao dịch đã hoàn tiền", infrastructure.TransferStatusReversed},
		{"reversed-unaccented", 2, "", "Hoan chuyen tien", "", infrastructure.TransferStatusReversed},
		{"reversed-english", 2, "", "Transaction reversed by beneficiary bank", "", infrastructure.TransferStatusReversed},
		{"reversed-takes-precedence-over-error-code", 2, "999", "Hoàn chuyển", "", infrastructure.TransferStatusReversed},
		{"failed-reason-not-reversal", 2, "1004", "Bank account invalid", "", infrastructure.TransferStatusFailed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := translateStatus(c.status, c.code, c.failureReason, c.message); got != c.want {
				t.Errorf("translateStatus(%d, %q, %q, %q) = %q, want %q",
					c.status, c.code, c.failureReason, c.message, got, c.want)
			}
		})
	}
}

func TestParseAndVerifyWebhook_Reversed(t *testing.T) {
	body := `{
		"status": 2,
		"error_code": "",
		"failure_reason": "Hoàn chuyển tiền - tài khoản người nhận không hợp lệ",
		"message": "Refund processed",
		"method": "DISBURSEMENT",
		"payment_no": "485436557681313",
		"amount": 10000,
		"description": "salary",
		"bank_code": "VCB",
		"account_name": "NGUYEN VAN A",
		"account_no": "1023020330000",
		"created_at": "2026-05-07 11:00:00",
		"request_id": "REVERSAL-001"
	}`
	const secret = "ipn-secret"
	result, checksum := sealWebhook(t, body, secret)

	ev, err := parseAndVerifyWebhook(map[string]any{
		"result":   result,
		"checksum": checksum,
	}, secret)
	if err != nil {
		t.Fatalf("parseAndVerifyWebhook: %v", err)
	}

	if ev.Status != infrastructure.TransferStatusReversed {
		t.Errorf("Status = %q, want reversed", ev.Status)
	}
	if ev.RequestID != "REVERSAL-001" {
		t.Errorf("RequestID = %q", ev.RequestID)
	}
	if ev.ProviderRef != "485436557681313" {
		t.Errorf("ProviderRef = %q", ev.ProviderRef)
	}
	if ev.FailureReason == "" {
		t.Error("FailureReason should be propagated for reversal IPN")
	}
}
