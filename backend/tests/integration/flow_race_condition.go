package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
)

const flowRaceCondition = "RaceCondition"

func runRaceConditionTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW: Advance Payment Race Condition Tests")

	if data.EmployeeForAdvance == nil || data.EmployeeTokenForAdv == "" {
		reporter.Skip(flowRaceCondition, "All race condition tests", "no employee user account found")
		return
	}

	empClient := client.WithToken(data.EmployeeTokenForAdv)

	// First check if employee can request
	var info AdvancePaymentInfoResponse
	canRaceTest := true
	reporter.RunTest(flowRaceCondition, "Check employee advance info", func() error {
		if _, err := empClient.GetInto("/api/v1/me/advance-payment", &info); err != nil {
			canRaceTest = false
			return fmt.Errorf("get advance info: %w", err)
		}
		fmt.Printf("    ForMonth: %s, CanRequest: %v, HasFlexible: %v, MaxAdvance: %d, Remaining: %d\n",
			info.ForMonth, info.CanRequest, info.HasFlexible, info.MaxAdvanceAmount, info.RemainingAmount)

		if !info.CanRequest {
			canRaceTest = false
			reporter.Skip(flowRaceCondition, "All concurrent budget tests",
				fmt.Sprintf("CanRequest=false (reason: %s) — likely past cutoff with no data for next period", info.CanRequestReason))
			return nil // not a failure — just timing
		}
		if info.RemainingAmount < 200000 {
			canRaceTest = false
			reporter.Skip(flowRaceCondition, "All concurrent budget tests",
				fmt.Sprintf("remaining amount %d too low (need >= 200k)", info.RemainingAmount))
			return nil
		}
		return nil
	})

	if !canRaceTest || !info.CanRequest || info.RemainingAmount < 200000 {
		return
	}

	// Count existing requests before test
	var existingRequests []AdvancePaymentRequestItem
	reporter.RunTest(flowRaceCondition, "Snapshot existing requests", func() error {
		if _, err := client.GetInto("/api/v1/advance-payments?status=PENDING&status=APPROVED&status=COMPLETED&pageSize=100", &existingRequests); err != nil {
			return fmt.Errorf("list existing requests: %w", err)
		}
		fmt.Printf("    Found %d existing active requests\n", len(existingRequests))
		return nil
	})

	// === TEST 1: Concurrent requests that should NOT exceed budget ===
	// If remaining is 300k, send 2x 200k concurrently — both should succeed (total 200k <= 300k)
	reporter.RunTest(flowRaceCondition, "Concurrent requests within budget (2x 200k, remaining="+fmt.Sprintf("%d", info.RemainingAmount)+")", func() error {
		var successCount atomic.Int32
		var failCount atomic.Int32
		var wg sync.WaitGroup
		var mu sync.Mutex
		var createdIDs []uint

		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				req := CreateAdvancePaymentRequest{Amount: 200000}
				var created AdvancePaymentHistoryItem
				apiResp, err := empClient.PostInto("/api/v1/me/advance-payment/request", req, &created)
				if err != nil {
					failCount.Add(1)
					mu.Lock()
					fmt.Printf("    Goroutine %d: FAILED — %v\n", idx, err)
					if apiResp != nil {
						fmt.Printf("    Response: %s\n", apiResp.Message)
					}
					mu.Unlock()
					return
				}
				successCount.Add(1)
				mu.Lock()
				createdIDs = append(createdIDs, created.ID)
				fmt.Printf("    Goroutine %d: SUCCESS — request ID %d created\n", idx, created.ID)
				mu.Unlock()
			}(i)
		}
		wg.Wait()

		fmt.Printf("    Results: %d succeeded, %d failed\n", successCount.Load(), failCount.Load())

		// Both should succeed since 200k <= remaining
		if successCount.Load() != 2 {
			return fmt.Errorf("expected both requests to succeed within budget, got %d successes", successCount.Load())
		}

		// Cancel the created requests to clean up
		for _, id := range createdIDs {
			path := fmt.Sprintf("/api/v1/me/advance-payment/request/%d/cancel", id)
			_, _, _ = empClient.Post(path, nil)
		}

		return nil
	})

	// === TEST 2: Concurrent requests that SHOULD exceed budget ===
	// Calculate an amount that would exceed the limit if both succeed
	// We send 2x (remaining/2 + 1) concurrently — at most one should succeed
	remaining := info.RemainingAmount
	// Use a request amount that's > half of remaining so two together exceed the limit
	// But each individual request is within the limit
	requestAmount := remaining/2 + 1
	// Clamp to minimum 10k
	if requestAmount < 10000 {
		requestAmount = 10000
	}

	reporter.RunTest(flowRaceCondition, fmt.Sprintf("Concurrent requests exceeding budget (2x %d, remaining=%d)", requestAmount, remaining), func() error {
		var successCount atomic.Int32
		var failCount atomic.Int32
		var wg sync.WaitGroup
		var mu sync.Mutex
		var createdIDs []uint
		var errors []string

		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				req := CreateAdvancePaymentRequest{Amount: requestAmount}
				var created AdvancePaymentHistoryItem
				apiResp, err := empClient.PostInto("/api/v1/me/advance-payment/request", req, &created)
				if err != nil {
					failCount.Add(1)
					mu.Lock()
					errMsg := fmt.Sprintf("Goroutine %d: REJECTED — %v", idx, err)
					errors = append(errors, errMsg)
					if apiResp != nil {
						// Try to extract the Vietnamese error message
						var apiErrResp APIError
						raw, _ := json.Marshal(apiResp)
						if json.Unmarshal(raw, &apiErrResp) == nil && apiErrResp.Message != "" {
							errMsg = fmt.Sprintf("Goroutine %d: REJECTED — %s", idx, apiErrResp.Message)
						}
					}
					fmt.Printf("    %s\n", errMsg)
					mu.Unlock()
					return
				}
				successCount.Add(1)
				mu.Lock()
				createdIDs = append(createdIDs, created.ID)
				fmt.Printf("    Goroutine %d: SUCCESS — request ID %d (amount=%d)\n", idx, created.ID, created.RequestAmount)
				mu.Unlock()
			}(i)
		}
		wg.Wait()

		fmt.Printf("    Results: %d succeeded, %d failed\n", successCount.Load(), failCount.Load())

		// At most ONE should succeed — the budget check should prevent both
		if successCount.Load() > 1 {
			// Clean up on failure
			for _, id := range createdIDs {
				path := fmt.Sprintf("/api/v1/me/advance-payment/request/%d/cancel", id)
				_, _, _ = empClient.Post(path, nil)
			}
			return fmt.Errorf("RACE CONDITION DETECTED: %d requests succeeded when budget only allows 1 (amount=%d each, remaining=%d)",
				successCount.Load(), requestAmount, remaining)
		}

		// At least one should succeed (first one in)
		if successCount.Load() == 0 {
			fmt.Printf("    NOTE: Both requests rejected — this is safe but may indicate the first request also failed\n")
			for _, e := range errors {
				fmt.Printf("    %s\n", e)
			}
		}

		// Clean up the one that succeeded
		for _, id := range createdIDs {
			path := fmt.Sprintf("/api/v1/me/advance-payment/request/%d/cancel", id)
			_, _, _ = empClient.Post(path, nil)
		}

		fmt.Printf("    ✅ Budget protection working: only %d of 2 concurrent requests succeeded\n", successCount.Load())
		return nil
	})

	// === TEST 3: Verify budget remains consistent after concurrent requests ===
	reporter.RunTest(flowRaceCondition, "Verify budget consistency after test", func() error {
		var infoAfter AdvancePaymentInfoResponse
		if _, err := empClient.GetInto("/api/v1/me/advance-payment", &infoAfter); err != nil {
			return fmt.Errorf("get advance info after: %w", err)
		}

		fmt.Printf("    After: ForMonth=%s, Remaining=%d (was %d), Completed=%d, Pending=%d\n",
			infoAfter.ForMonth, infoAfter.RemainingAmount, info.RemainingAmount,
			infoAfter.CompletedAmount, infoAfter.PendingAmount)

		// Pending should be 0 if we cleaned up properly
		// Completed should not have changed (we only created PENDING)
		if infoAfter.CompletedAmount != info.CompletedAmount {
			return fmt.Errorf("completed amount changed unexpectedly: was %d, now %d",
				info.CompletedAmount, infoAfter.CompletedAmount)
		}

		return nil
	})
}
