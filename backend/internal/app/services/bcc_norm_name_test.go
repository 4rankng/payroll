package services

import (
	"testing"

	"golang.org/x/text/unicode/norm"
)

func TestBccNormName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"  Nguyễn Văn A  ", "nguyễn văn a"},
		{"Nguyen Van A", "nguyen van a"},
		{"", ""},
	}
	for _, c := range cases {
		if got := bccNormName(c.in); got != c.want {
			t.Errorf("bccNormName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestBccNormName_NFD_NFC: cross-check between STK and BCC sheets must
// tolerate the two Unicode normal forms Vietnamese text arrives in.
func TestBccNormName_NFD_NFC(t *testing.T) {
	nfc := "Trần Thị B"
	nfd := norm.NFD.String(nfc)
	if nfc == nfd {
		t.Skip("test env collapsed NFD/NFC")
	}
	if bccNormName(nfc) != bccNormName(nfd) {
		t.Errorf("NFC %q vs NFD %q should match", nfc, nfd)
	}
}
