package services

import "testing"

func TestShiftInConfig(t *testing.T) {
	fr := map[string]int{
		"phổ thông.ngày thường.Com": 31000,
		"phổ thông.ngày thường.HC":  55000,
		"phổ thông.ngày lễ.HC":      0, // zero rate but still a configured shift
	}
	cases := []struct {
		shift string
		want  bool
	}{
		{"Com", true},
		{"com", true},   // case-insensitive
		{"HC", true},    // exists even where rate is 0
		{"hc", true},
		{"OT150", false},
	}
	for _, c := range cases {
		if got := shiftInConfig(fr, c.shift); got != c.want {
			t.Errorf("shiftInConfig(%q) = %v, want %v", c.shift, got, c.want)
		}
	}
	if shiftInConfig(nil, "HC") {
		t.Error("shiftInConfig(nil,...) should be false")
	}
}

func TestBuildShiftRatesForShift(t *testing.T) {
	fr := map[string]int{
		"phổ thông.ngày thường.Com": 31000,
		"phổ thông.ngày nghỉ.Com":   31000,
		"phổ thông.ngày thường.HC":  55000,
		"phổ thông.ngày lễ.HC":      0, // skipped (rate 0)
	}
	rates := buildShiftRatesForShift(fr, "com")
	if len(rates) != 2 {
		t.Fatalf("expected 2 Com rate keys, got %d: %v", len(rates), rates)
	}
	want := map[wbccRateKey]int{
		{position: "phổ thông", dayType: "ngày thường"}: 31000,
		{position: "phổ thông", dayType: "ngày nghỉ"}:   31000,
	}
	for k, v := range want {
		if got, ok := rates[k]; !ok || got != v {
			t.Errorf("rates[%v] = (%d, %v), want %d", k, got, ok, v)
		}
	}
}
