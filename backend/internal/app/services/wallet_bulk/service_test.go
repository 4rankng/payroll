package wallet_bulk

import (
	"testing"
)

func TestComputeContentHash_Deterministic(t *testing.T) {
	rows := []BulkTransferRow{
		{OrderNo: 1, AccountNo: "A1", AccountName: "N1", Bank: "B1", SwiftCode: "SW1", Amount: 100, PaymentDetail: "VFICaaa1", VFICCode: "VFICaaa1"},
		{OrderNo: 2, AccountNo: "A2", AccountName: "N2", Bank: "B2", SwiftCode: "SW2", Amount: 200, PaymentDetail: "VFICbbb2", VFICCode: "VFICbbb2"},
	}
	h1 := computeContentHash(rows)
	if h1 == "" {
		t.Fatal("expected non-empty hash")
	}
	// Same rows in different order → same hash (sort-by-VFIC normalization).
	shuffled := []BulkTransferRow{rows[1], rows[0]}
	h2 := computeContentHash(shuffled)
	if h1 != h2 {
		t.Errorf("hash differs after shuffle: %q vs %q", h1, h2)
	}
	// Different content → different hash.
	rows[0].Amount = 999
	h3 := computeContentHash(rows)
	if h1 == h3 {
		t.Errorf("hash unchanged after amount mutation")
	}
}

func TestComputeContentHash_Empty(t *testing.T) {
	// Empty input should still produce a stable hash (no panic).
	h := computeContentHash(nil)
	if h == "" {
		t.Fatal("expected non-empty hash for nil input")
	}
}

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"normal", "normal.xlsx", "normal.xlsx"},
		{"basename only", "path/to/file.xlsx", "file.xlsx"},
		{"windows path", "\\server\\share\\evil.xlsx", "evil.xlsx"},
		{"control char stripped", "name\x00bad.xlsx", "namebad.xlsx"},
		{"XSS chars stripped", "<script>.xlsx", "script.xlsx"},
		{"empty default", "", "bulk_transfer.xlsx"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sanitizeFilename(c.in)
			if got != c.want {
				t.Errorf("sanitize(%q): got %q, want %q", c.in, got, c.want)
			}
			if len(got) > 128 {
				t.Errorf("sanitize(%q): result exceeds 128 chars", c.in)
			}
		})
	}
}

func TestSanitizeFilename_LengthCap(t *testing.T) {
	long := "a"
	for i := 0; i < 300; i++ {
		long += "b"
	}
	got := sanitizeFilename(long + ".xlsx")
	if len(got) > 128 {
		t.Errorf("length: got %d, want ≤128", len(got))
	}
}
