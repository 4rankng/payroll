package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPayrate_ValidatePayrateConfig(t *testing.T) {
	pr := &Payrate{
		ProjectID: 1,
		FromDate:  time.Now().AddDate(0, 0, -1), // yesterday
		Payrate:   PayrateConfiguration(`{"regular": {"rate": 100000}}`),
	}

	err := pr.ValidatePayrateConfig()
	assert.NoError(t, err)

	// Test with empty configuration
	pr.Payrate = PayrateConfiguration("")
	err = pr.ValidatePayrateConfig()
	assert.Error(t, err)
}

func TestPayrate_IsValid(t *testing.T) {
	now := time.Now()
	pr := &Payrate{
		ProjectID: 1,
		FromDate:  now.AddDate(0, 0, -1), // yesterday
		Payrate:   PayrateConfiguration(`{"regular": {"rate": 100000}}`),
	}

	err := pr.IsValid()
	assert.NoError(t, err)

	// Test with invalid ProjectID
	pr.ProjectID = 0
	err = pr.IsValid()
	assert.Error(t, err)
}

func TestPayrate_IsActiveOnDate(t *testing.T) {
	now := time.Now()
	pr := &Payrate{
		FromDate: now.AddDate(0, -1, 0), // 1 month ago
	}

	// Active today
	assert.True(t, pr.IsActiveOnDate(now))

	// Active forever after from_date: there is no end date — a config runs
	// until its successor's fromDate (migration 138 dropped to_date)
	futureDate := now.AddDate(0, 2, 0) // 2 months from now
	assert.True(t, pr.IsActiveOnDate(futureDate))

	// Not active before from_date
	pastDate := now.AddDate(0, -2, 0) // 2 months ago
	assert.False(t, pr.IsActiveOnDate(pastDate))
}

func TestPayrateConfiguration_NormalizedKeys(t *testing.T) {
	// Keys are trimmed and NFC-normalized; diacritics are preserved for display.
	normalized, err := PayrateConfiguration("{\"Lái xe nâng \": {\"ngày thường\": {\"ca ngày\": 10000}}}").NormalizedKeys()
	assert.NoError(t, err)

	var got map[string]interface{}
	assert.NoError(t, json.Unmarshal(normalized, &got))
	assert.Contains(t, got, "Lái xe nâng")

	// Sibling keys colliding after the Vietnamese lookup fold are rejected.
	_, err = PayrateConfiguration(
		"{\"Lái xe nâng\": {\"ngày thường\": {\"ca ngày\": 10000}}, \"lái xe nâng\": {\"ngày thường\": {\"ca ngày\": 20000}}}",
	).NormalizedKeys()
	assert.Error(t, err)
}

func TestPayrate_GetRateValue(t *testing.T) {
	config := `{"regular": 100000, "overtime": 150000}`
	pr := &Payrate{
		Payrate: PayrateConfiguration(config),
	}

	// Test getting regular rate
	rate, err := pr.GetRateValue("regular")
	assert.NoError(t, err)
	assert.Equal(t, 100000, rate)

	// Test getting overtime rate
	rate, err = pr.GetRateValue("overtime")
	assert.NoError(t, err)
	assert.Equal(t, 150000, rate)

	// Test getting non-existent rate
	rate, err = pr.GetRateValue("nonexistent")
	assert.Error(t, err)
	assert.Equal(t, 0, rate)
}

func TestPayrate_GetAllRates(t *testing.T) {
	config := `{"regular": 100000, "overtime": 150000}`
	pr := &Payrate{
		Payrate: PayrateConfiguration(config),
	}

	rates := pr.GetAllRates()
	assert.Equal(t, PayrateConfiguration(config), rates)
}

// TestNormalizeVietnamese: diacritic-stripping must fold Đ/đ (U+0110/U+0111)
// to D/d. These are precomposed letters that NFD does not decompose and that
// carry no unicode.Mn combining mark, so without an explicit replacement the
// doc-promised "Ca đêm" -> "ca dem" mapping would actually yield "ca đem" and
// night-shift payrate lookups would miss. Mirrors the onepay.toASCII fix.
func TestNormalizeVietnamese(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Ngày lễ", "ngay le"},
		{"TC 150%", "tc 150%"},
		{"Ca đêm", "ca dem"},
		{"Đặng Thị Huệ", "dang thi hue"},
		{"HĐ-001", "hd-001"},
	}
	for _, c := range cases {
		if got := normalizeVietnamese(c.in); got != c.want {
			t.Errorf("normalizeVietnamese(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestPayrateConfiguration_RateLookup: a lookup built once must resolve with
// the same exact-then-Vietnamese-folded matching as GetRate, so re-pricing
// loops can parse the JSON once per config without changing behavior.
func TestPayrateConfiguration_RateLookup(t *testing.T) {
	config := PayrateConfiguration(`{"Tất cả":{"Ngày thường":50000,"Ngày lễ":100000},"Lái xe nâng":{"Ngày thường":60000}}`)

	lookup, err := config.RateLookup()
	if err != nil {
		t.Fatalf("RateLookup() returned error: %v", err)
	}

	// Exact match
	v, err := lookup.Get("Tất cả.Ngày thường")
	if err != nil || v != 50000 {
		t.Errorf("exact match: got (%d, %v), want 50000, nil", v, err)
	}

	// Folded fallback: case and diacritics differences resolve to the same rate
	v, err = lookup.Get("tất cả.ngày lễ")
	if err != nil || v != 100000 {
		t.Errorf("folded fallback: got (%d, %v), want 100000, nil", v, err)
	}

	// Missing path errors
	if _, err := lookup.Get("Tất cả.Ca đêm"); err == nil {
		t.Error("missing path: want error, got nil")
	}

	// Invalid JSON errors
	if _, err := PayrateConfiguration(`not json`).RateLookup(); err == nil {
		t.Error("invalid JSON: want error, got nil")
	}
}
