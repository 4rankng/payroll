package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// SetServerTime sets the test server's internal clock to the given time
// via the admin clock endpoint.
func SetServerTime(client *APIClient, t time.Time) error {
	body := map[string]string{"time": t.Format(time.RFC3339)}
	resp, statusCode, err := client.Post("/api/v1/admin/clock/set", body)
	if err != nil {
		return fmt.Errorf("set server time: %w", err)
	}
	if statusCode != 200 {
		return fmt.Errorf("set server time failed: status %d, body: %s", statusCode, string(resp.Data))
	}
	return nil
}

// AdvanceServerTime advances the server clock by the given duration.
func AdvanceServerTime(client *APIClient, d time.Duration) error {
	body := map[string]string{"duration": d.String()}
	resp, statusCode, err := client.Post("/api/v1/admin/clock/advance", body)
	if err != nil {
		return fmt.Errorf("advance server time: %w", err)
	}
	if statusCode != 200 {
		return fmt.Errorf("advance server time failed: status %d, body: %s", statusCode, string(resp.Data))
	}
	return nil
}

// ResetServerTime resets the server clock to real system time.
func ResetServerTime(client *APIClient) error {
	resp, statusCode, err := client.Post("/api/v1/admin/clock/reset", nil)
	if err != nil {
		return fmt.Errorf("reset server time: %w", err)
	}
	if statusCode != 200 {
		return fmt.Errorf("reset server time failed: status %d, body: %s", statusCode, string(resp.Data))
	}
	return nil
}

// GetServerTime returns the server's current clock time.
func GetServerTime(client *APIClient) (time.Time, error) {
	var result struct {
		Time   string `json:"time"`
		Unix   int64  `json:"unix"`
		IsFake bool   `json:"is_fake"`
	}
	resp, _, err := client.Get("/api/v1/admin/clock")
	if err != nil {
		return time.Time{}, fmt.Errorf("get server time: %w", err)
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return time.Time{}, fmt.Errorf("parse server time response: %w", err)
	}
	return time.Parse(time.RFC3339, result.Time)
}
