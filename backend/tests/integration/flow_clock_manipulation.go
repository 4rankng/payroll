package main

import (
	"encoding/json"
	"fmt"
	"time"
)

const flowClock = "ClockManipulation"

// runClockManipulationTests verifies the admin clock endpoint works correctly.
func runClockManipulationTests(client *APIClient, reporter *Reporter) {
	reporter.PrintSection("FLOW: Clock Manipulation")

	// Step 1: Get current server time
	reporter.RunTest(flowClock, "Get current server time", func() error {
		serverTime, err := GetServerTime(client)
		if err != nil {
			return fmt.Errorf("get server time: %w", err)
		}
		// Just verify we got a plausible time (after 2020)
		if serverTime.Year() < 2020 {
			return fmt.Errorf("implausible server time: %s", serverTime)
		}
		return nil
	})

	// Step 2: Set to a specific date
	var targetTime time.Time
	reporter.RunTest(flowClock, "Set server time to 2026-06-01", func() error {
		var err error
		targetTime, err = time.Parse(time.RFC3339, "2026-06-01T00:00:00+07:00")
		if err != nil {
			return err
		}
		return SetServerTime(client, targetTime)
	})

	// Step 3: Verify the set time stuck
	reporter.RunTest(flowClock, "Verify server time is 2026-06-01", func() error {
		gotTime, err := GetServerTime(client)
		if err != nil {
			return err
		}
		if !sameDay(gotTime, targetTime) {
			return fmt.Errorf("expected same day as %s, got %s",
				targetTime.Format("2006-01-02"), gotTime.Format("2006-01-02"))
		}
		return nil
	})

	// Step 4: Advance by 48h
	reporter.RunTest(flowClock, "Advance server time by 48h", func() error {
		return AdvanceServerTime(client, 48*time.Hour)
	})

	// Step 5: Verify time moved forward
	reporter.RunTest(flowClock, "Verify time is 2026-06-03 (after 48h advance)", func() error {
		expected := targetTime.Add(48 * time.Hour)
		gotTime, err := GetServerTime(client)
		if err != nil {
			return err
		}
		if !sameDay(gotTime, expected) {
			return fmt.Errorf("expected same day as %s, got %s",
				expected.Format("2006-01-02"), gotTime.Format("2006-01-02"))
		}
		return nil
	})

	// Step 6: Reset
	reporter.RunTest(flowClock, "Reset server time to real time", func() error {
		return ResetServerTime(client)
	})

	// Step 7: Verify reset — time should be close to real now and clock should no longer be fake
	reporter.RunTest(flowClock, "Verify time restored to real time", func() error {
		var result struct {
			Time   string `json:"time"`
			Unix   int64  `json:"unix"`
			IsFake bool   `json:"is_fake"`
		}
		resp, _, err := client.Get("/api/v1/admin/clock")
		if err != nil {
			return fmt.Errorf("get server time: %w", err)
		}
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("parse server time response: %w", err)
		}
		if result.IsFake {
			return fmt.Errorf("clock should not be fake after reset (is_fake=true)")
		}

		gotTime, err := time.Parse(time.RFC3339, result.Time)
		if err != nil {
			return fmt.Errorf("parse time: %w", err)
		}
		realNow := time.Now()
		diff := gotTime.Sub(realNow)
		if diff < 0 {
			diff = -diff
		}
		if diff > 5*time.Second {
			return fmt.Errorf("time diff too large after reset: %v (server: %s, real: %s)",
				diff, gotTime.Format(time.RFC3339), realNow.Format(time.RFC3339))
		}
		return nil
	})
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}
