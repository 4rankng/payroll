package excel

import (
	"testing"

	"golang.org/x/text/unicode/norm"
)

func TestNormHeader(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"  Họ và tên  ", "họ và tên"},
		{"Mã nhân viên", "mã nhân viên"},
		{"STT", "stt"},
		{"CCCD", "cccd"},
		{"Bộ phận", "bộ phận"},
	}
	for _, c := range cases {
		if got := normHeader(c.in); got != c.want {
			t.Errorf("normHeader(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestNormHeader_NFD_NFC verifies that Vietnamese text arriving in NFD form
// (common for Excel files saved on macOS) is treated as equivalent to the NFC
// form. Without this, dynamic header detection would silently fall back.
func TestNormHeader_NFD_NFC(t *testing.T) {
	nfc := "Nguyễn"
	nfd := norm.NFD.String(nfc)
	if nfc == nfd {
		t.Skip("test environment collapsed NFD/NFC; skipping")
	}
	if got := normHeader(nfd); got != "nguyễn" {
		t.Errorf("normHeader(NFD %q) = %q, want %q", nfd, got, "nguyễn")
	}
}
