package onepay

import (
	"testing"
)

func TestMessageVI_KnownCodes(t *testing.T) {
	cases := []struct {
		code string
		want string
	}{
		{"00", "Thành công"},
		{"07", "Giao dịch bị trùng"},
		{"15", "Thông tin tài khoản không hợp lệ"},
		{"19", "Không tìm thấy ngân hàng xử lý"},
		{"40", "Quá hạn mức tối thiểu trên lần giao dịch"},
		{"41", "Quá hạn mức tối đa trên lần giao dịch"},
		{"92", "Chữ ký xác thực không hợp lệ"},
		{"100", "Hệ thống tạm thời gián đoạn"},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			got := MessageVI(tc.code)
			if got != tc.want {
				t.Errorf("MessageVI(%q) = %q, want %q", tc.code, got, tc.want)
			}
		})
	}
}

func TestMessageVI_EmptyCode(t *testing.T) {
	got := MessageVI("")
	if got != "" {
		t.Errorf("MessageVI(empty) = %q, want empty string", got)
	}
}

func TestMessageVI_UnknownCode(t *testing.T) {
	got := MessageVI("999")
	if got == "" {
		t.Error("MessageVI(unknown) should return fallback, not empty")
	}
	// Fallback should include the code
	if !contains(got, "999") {
		t.Errorf("fallback should contain code '999', got %q", got)
	}
}

func TestMessageVI_AllCriticalCodesExist(t *testing.T) {
	// From the spec: all critical codes must exist in the map.
	criticalCodes := []string{"00", "07", "15", "19", "40", "41", "92", "100"}
	for _, code := range criticalCodes {
		t.Run("exists/"+code, func(t *testing.T) {
			msg, ok := ErrorMessagesVI[code]
			if !ok {
				t.Errorf("code %q missing from ErrorMessagesVI", code)
			}
			if msg == "" {
				t.Errorf("code %q has empty message", code)
			}
		})
	}
}

func TestErrorMessagesVI_CoversAllSpecCodes(t *testing.T) {
	// Verify every code from the PDF table has an entry.
	specCodes := []string{
		"00", "01", "02", "03", "04", "05", "06", "07", "08", "09",
		"10", "12", "13", "14", "15", "16", "17", "18", "19", "20",
		"21", "22", "30", "31", "32", "34", "38", "39", "40", "41",
		"42", "45", "50", "70", "71", "72", "73", "80", "81", "82",
		"83", "86", "87", "88", "89", "90", "91", "92", "93", "94",
		"95", "96", "97", "98", "99", "100", "101", "102", "ZZ",
	}
	for _, code := range specCodes {
		t.Run("spec/"+code, func(t *testing.T) {
			msg, ok := ErrorMessagesVI[code]
			if !ok {
				t.Errorf("code %q from PDF spec missing from ErrorMessagesVI", code)
			}
			if msg == "" {
				t.Errorf("code %q has empty message", code)
			}
		})
	}
}

// contains is a simple string contains helper (avoids importing strings for one use).
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
