package integration

import "testing"

func TestVendorPayType(t *testing.T) {
	cases := []struct{ in, want string }{
		{"phổ thường.ngày thường.tcn", "ngày_thường.TCN"},
		{"công nhân.ngày thường.hc", "ngày_thường.HC"},
		{"công nhân.ngày nghỉ.nn", "ngày_nghỉ.NN"},
		{"phổ thường.ngày lễ.08:00-18:00", "ngày_lễ.08:00-18:00"},
		{"ngày_thường.TCN", "ngày_thường.TCN"}, // already 2-part
		{"garbage", "garbage"},                 // not dot-separated: pass through
		{"", ""},
	}
	for _, c := range cases {
		if got := VendorPayType(c.in); got != c.want {
			t.Errorf("VendorPayType(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
