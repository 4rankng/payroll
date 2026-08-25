package workers

import "testing"

func TestIsPermanentAccountCheckFailure(t *testing.T) {
	cases := []struct {
		code string
		want bool
	}{
		{"name_mismatch", true},
		{"", false},
		{"preflight_validation", false}, // malformed request, handled at its own layer
		{"account_not_found", false},    // provider-specific, kept retryable until classified
		{"E26", false},                  // raw provider code, not classified as permanent
		{"NAME_MISMATCH", false},        // case-sensitive: only the code we emit matches
	}
	for _, tc := range cases {
		if got := isPermanentAccountCheckFailure(tc.code); got != tc.want {
			t.Errorf("isPermanentAccountCheckFailure(%q) = %v, want %v", tc.code, got, tc.want)
		}
	}
}
