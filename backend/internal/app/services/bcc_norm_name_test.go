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

// TestBccNormNameLoose: diacritic-stripped form of two names that differ only
// by an accent mark (e.g. "Thì" vs "Thị") must compare equal, so the STK
// cross-check can accept common data-entry typos. Two names that differ in
// letters after diacritic removal (e.g. "Lò" vs "Lê") must NOT compare equal —
// that is a real mismatch we want to surface.
func TestBccNormNameLoose(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		// Same letters, different diacritic — should match after stripping.
		{"Lò Thì Dương", "Lò Thị Dương", true},
		{"Nguyễn Văn A", "Nguyễn Văn Á", true},
		{"Trần Thị B", "tran thi b", true}, // loose is also lowercase + trim

		// Different letters — should NOT match even after stripping.
		{"Lò Thị Dương", "Lê Thị Dương", false}, // Lò vs Lê: different base char
		{"Lò Thị Dương", "Lò Thị Dung", false},  // Dương vs Dung: stripped differs

		// Empty / whitespace.
		{"", "", true},
	}
	for _, c := range cases {
		got := bccNormNameLoose(c.a) == bccNormNameLoose(c.b)
		if got != c.want {
			t.Errorf("bccNormNameLoose(%q) vs bccNormNameLoose(%q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
