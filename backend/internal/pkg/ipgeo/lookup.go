// Package ipgeo provides a lightweight IP-to-location lookup using the free ip-api.com service.
// No API key required. Results are cached in-memory to avoid hammering the upstream API.
package ipgeo

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// Location holds the geo result for an IP address.
type Location struct {
	Country string `json:"country"`
	City    string `json:"city"`
	Region  string `json:"region"`
}

// ipAPIResponse mirrors the fields we care about from ip-api.com.
type ipAPIResponse struct {
	Status      string `json:"status"`
	Country     string `json:"country"`
	RegionName  string `json:"regionName"`
	City        string `json:"city"`
	CountryCode string `json:"countryCode"`
}

var (
	cache   = make(map[string]*Location)
	cacheMu sync.RWMutex

	client = &http.Client{
		Timeout: 3 * time.Second,
	}
)

// isPrivate returns true for loopback / RFC-1918 / link-local addresses
// that ip-api.com cannot resolve.
func isPrivate(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return true
	}
	return parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast()
}

// Lookup resolves an IP address to a Location.
// Returns nil, nil for private/loopback addresses.
// Results are cached for the lifetime of the process.
func Lookup(ctx context.Context, ip string) (*Location, error) {
	if ip == "" || isPrivate(ip) {
		return nil, nil
	}

	// Check cache first.
	cacheMu.RLock()
	if loc, ok := cache[ip]; ok {
		cacheMu.RUnlock()
		return loc, nil
	}
	cacheMu.RUnlock()

	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,countryCode,regionName,city", ip)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var apiResp ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	if apiResp.Status != "success" {
		// ip-api returns "fail" for reserved/private ranges — treat as no data.
		return nil, nil
	}

	loc := &Location{
		Country: apiResp.Country,
		City:    apiResp.City,
		Region:  apiResp.RegionName,
	}

	cacheMu.Lock()
	cache[ip] = loc
	cacheMu.Unlock()

	return loc, nil
}
