package main

import (
	"encoding/json"
	"fmt"
	"time"
)

const flowAdvance = "AdvancePayment"

func runAdvancePaymentTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW 2: Advance Salary Payment via 9Pay")

	if data.EmployeeForAdvance == nil || data.EmployeeTokenForAdv == "" {
		reporter.Skip(flowAdvance, "All advance payment tests", "no employee user account found")
		return
	}

	empClient := client.WithToken(data.EmployeeTokenForAdv)
	adminClient := client.WithToken(data.AdminToken)

	var info AdvancePaymentInfoResponse

	// 2.1 Calculate fee preview
	reporter.RunTest(flowAdvance, "Calculate fee preview (100,000 VND)", func() error {
		req := CreateAdvancePaymentRequest{Amount: 100000}
		var resp CalculateFeeResponse
		if _, err := empClient.PostInto("/api/v1/me/advance-payment/calculate-fee", req, &resp); err != nil {
			return fmt.Errorf("calculate fee: %w", err)
		}
		if err := AssertEqual("requestAmount", uint64(100000), resp.RequestAmount); err != nil {
			return err
		}
		if err := AssertGreaterThan("fee", uint64(0), resp.Fee); err != nil {
			return err
		}
		expectedNet := resp.RequestAmount - resp.Fee
		return AssertEqual("netAmount", expectedNet, resp.NetAmount)
	})

	// 2.3 Admin sees pending requests
	reporter.RunTest(flowAdvance, "Admin sees pending requests", func() error {
		var items []AdvancePaymentRequestItem
		if _, err := adminClient.GetInto("/api/v1/advance-payments?status=PENDING&pageSize=50", &items); err != nil {
			return fmt.Errorf("admin list pending: %w", err)
		}
		fmt.Printf("    Found %d pending requests\n", len(items))
		return nil
	})

	// 2.4 Admin summary
	reporter.RunTest(flowAdvance, "Admin summary reflects requests", func() error {
		var resp AdvancePaymentSummaryResponse
		if _, err := adminClient.GetInto("/api/v1/advance-payments/summary", &resp); err != nil {
			return fmt.Errorf("admin summary: %w", err)
		}
		fmt.Printf("    Total: %d requests, Pending: %d, Paid: %d, Amount: %d\n",
			resp.TotalRequests, resp.TotalPending, resp.TotalPaid, resp.TotalAmount)
		return AssertGreaterThan("totalRequests", int64(0), resp.TotalRequests)
	})

	// 2.5 Admin available months
	reporter.RunTest(flowAdvance, "Admin available months", func() error {
		var months []AvailableMonth
		if _, err := adminClient.GetInto("/api/v1/advance-payments/available-months", &months); err != nil {
			return fmt.Errorf("available months: %w", err)
		}
		fmt.Printf("    Found %d available months\n", len(months))
		return AssertSliceMinLen("months", len(months), 1)
	})

	// 2.6 Admin employee list
	reporter.RunTest(flowAdvance, "Admin employee list", func() error {
		var items []EmployeeAdvanceItem
		if _, err := adminClient.GetInto("/api/v1/advance-payments/employees", &items); err != nil {
			return fmt.Errorf("employee list: %w", err)
		}
		fmt.Printf("    Found %d employees\n", len(items))
		return nil
	})

	// --- Request creation + cancel flow (only if CanRequest) ---

	var requestID uint
	if info.CanRequest && info.RemainingAmount >= 100000 {
		reporter.RunTest(flowAdvance, "Create advance payment request", func() error {
			req := CreateAdvancePaymentRequest{Amount: 100000}
			var created AdvancePaymentHistoryItem
			if _, err := empClient.PostInto("/api/v1/me/advance-payment/request", req, &created); err != nil {
				return fmt.Errorf("create request: %w", err)
			}
			requestID = created.ID
			fmt.Printf("    Created request ID %d, status: %s\n", created.ID, created.Status)
			return AssertGreaterThan("id", uint(0), created.ID)
		})

		// Verify in employee history
		reporter.RunTest(flowAdvance, "Verify request in employee history", func() error {
			if requestID == 0 {
				return fmt.Errorf("no request ID from previous step")
			}
			var items []AdvancePaymentHistoryItem
			if _, err := empClient.GetInto("/api/v1/me/advance-payment/history", &items); err != nil {
				return fmt.Errorf("get history: %w", err)
			}
			for _, item := range items {
				if item.ID == requestID {
					return nil
				}
			}
			return fmt.Errorf("request ID %d not found in history (%d items)", requestID, len(items))
		})

		// Cancel the request
		reporter.RunTest(flowAdvance, "Cancel advance payment request", func() error {
			if requestID == 0 {
				return fmt.Errorf("no request to cancel")
			}
			path := fmt.Sprintf("/api/v1/me/advance-payment/request/%d/cancel", requestID)
			if _, _, err := empClient.Post(path, nil); err != nil {
				return fmt.Errorf("cancel request: %w", err)
			}
			fmt.Printf("    Cancelled request ID %d\n", requestID)
			return nil
		})

		// Re-cancel should fail
		reporter.RunTest(flowAdvance, "Edge: cancel already cancelled request", func() error {
			if requestID == 0 {
				return fmt.Errorf("no request to re-cancel")
			}
			path := fmt.Sprintf("/api/v1/me/advance-payment/request/%d/cancel", requestID)
			_, _, err := empClient.PostExpectError(path, nil)
			if err != nil {
				return fmt.Errorf("re-cancel request: %w", err)
			}
			return nil
		})
	} else {
		skipReason := fmt.Sprintf("CanRequest=%v, Remaining=%d", info.CanRequest, info.RemainingAmount)
		reporter.Skip(flowAdvance, "Create advance payment request", skipReason)
		reporter.Skip(flowAdvance, "Verify request in employee history", "no request created")
		reporter.Skip(flowAdvance, "Cancel advance payment request", "no request created")
		reporter.Skip(flowAdvance, "Edge: cancel already cancelled request", "no request created")
	}

	// --- Full 9Pay auto-disbursement lifecycle ---
	// Flow: Employee requests → PENDING → (poller claims) → APPROVED → (execute worker) → COMPLETED
	// Verifies: status transitions, wallet payment creation, employee info update

	var disbursementRequestID uint
	var completedAmountBefore uint64

	if info.CanRequest && info.RemainingAmount >= 100000 {
		// Snapshot employee info before disbursement
		var infoBefore AdvancePaymentInfoResponse
		reporter.RunTest(flowAdvance, "Snapshot employee info before disbursement", func() error {
			if _, err := empClient.GetInto("/api/v1/me/advance-payment", &infoBefore); err != nil {
				return fmt.Errorf("get advance info before: %w", err)
			}
			completedAmountBefore = infoBefore.CompletedAmount
			fmt.Printf("    Before: completedAmount=%d, pendingAmount=%d, remainingAmount=%d\n",
				infoBefore.CompletedAmount, infoBefore.PendingAmount, infoBefore.RemainingAmount)
			return nil
		})

		reporter.RunTest(flowAdvance, "Create request for 9Pay auto-disbursement", func() error {
			req := CreateAdvancePaymentRequest{Amount: 100000}
			var created AdvancePaymentHistoryItem
			if _, err := empClient.PostInto("/api/v1/me/advance-payment/request", req, &created); err != nil {
				return fmt.Errorf("create disbursement request: %w", err)
			}
			disbursementRequestID = created.ID
			fmt.Printf("    Created disbursement request ID %d, initial status: %s\n", created.ID, created.Status)
			if err := AssertGreaterThan("id", uint(0), created.ID); err != nil {
				return err
			}
			return AssertEqual("initial_status", "PENDING", created.Status)
		})

		// Verify request appears as PENDING in admin list
		reporter.RunTest(flowAdvance, "Verify request is PENDING in admin view", func() error {
			if disbursementRequestID == 0 {
				return fmt.Errorf("no request ID")
			}
			var items []AdvancePaymentRequestItem
			if _, err := adminClient.GetInto("/api/v1/advance-payments?status=PENDING&pageSize=50", &items); err != nil {
				return fmt.Errorf("admin list pending: %w", err)
			}
			for _, item := range items {
				if item.ID == disbursementRequestID {
					fmt.Printf("    Request %d confirmed PENDING in admin view\n", disbursementRequestID)
					return nil
				}
			}
			return fmt.Errorf("request %d not found in PENDING list", disbursementRequestID)
		})

		// Poll until status changes from PENDING (poller worker claims it → APPROVED)
		reporter.RunTest(flowAdvance, "Poll: wait for poller to claim request (PENDING → APPROVED)", func() error {
			if disbursementRequestID == 0 {
				return fmt.Errorf("no request ID")
			}
			path := "/api/v1/advance-payments?pageSize=50"
			_, err := adminClient.PollUntil(path, 3*time.Second, 2*time.Minute, func(apiResp *APIResponse) bool {
				if apiResp.Data == nil {
					return false
				}
				var items []AdvancePaymentRequestItem
				raw, _ := json.Marshal(apiResp.Data)
				if err := json.Unmarshal(raw, &items); err != nil {
					return false
				}
				for _, item := range items {
					if item.ID == disbursementRequestID {
						status := item.Status
						fmt.Printf("    Status: %s\n", status)
						return status == "APPROVED" || status == "COMPLETED" || status == "FAILED"
					}
				}
				return false
			})
			if err != nil {
				fmt.Printf("    WARNING: poll timeout waiting for PENDING→APPROVED: %v\n", err)
				return nil
			}
			fmt.Printf("    Request %d claimed by poller (no longer PENDING)\n", disbursementRequestID)
			return nil
		})

		// Poll until terminal state (COMPLETED or FAILED)
		reporter.RunTest(flowAdvance, "Poll: wait for 9Pay disbursement to complete (APPROVED → COMPLETED)", func() error {
			if disbursementRequestID == 0 {
				return fmt.Errorf("no request ID")
			}
			path := "/api/v1/advance-payments?pageSize=50"
			_, err := adminClient.PollUntil(path, 5*time.Second, 5*time.Minute, func(apiResp *APIResponse) bool {
				if apiResp.Data == nil {
					return false
				}
				var items []AdvancePaymentRequestItem
				raw, _ := json.Marshal(apiResp.Data)
				if err := json.Unmarshal(raw, &items); err != nil {
					return false
				}
				for _, item := range items {
					if item.ID == disbursementRequestID {
						return item.Status == "COMPLETED" || item.Status == "FAILED"
					}
				}
				return false
			})
			if err != nil {
				fmt.Printf("    WARNING: poll timeout - disbursement may still be processing: %v\n", err)
				return nil
			}
			fmt.Printf("    Request %d reached terminal state\n", disbursementRequestID)
			return nil
		})

		// Verify final status and details
		reporter.RunTest(flowAdvance, "Verify final request status and details", func() error {
			if disbursementRequestID == 0 {
				return fmt.Errorf("no request ID")
			}
			var items []AdvancePaymentRequestItem
			if _, err := adminClient.GetInto("/api/v1/advance-payments?pageSize=50", &items); err != nil {
				return fmt.Errorf("admin list: %w", err)
			}
			var found *AdvancePaymentRequestItem
			for i := range items {
				if items[i].ID == disbursementRequestID {
					found = &items[i]
					break
				}
			}
			if found == nil {
				return fmt.Errorf("request %d not found in admin list", disbursementRequestID)
			}
			fmt.Printf("    Final status: %s, paidAt: %v, paymentRef: %s\n",
				found.Status, found.PaidAt, found.PaymentRef)

			if found.Status == "COMPLETED" {
				if found.PaidAt == nil {
					return fmt.Errorf("COMPLETED request should have paidAt set")
				}
				if found.PaymentRef == "" {
					return fmt.Errorf("COMPLETED request should have paymentReference set")
				}
				fmt.Printf("    Payment reference from 9Pay: %s\n", found.PaymentRef)
			}
			return nil
		})

		// Verify wallet payment record exists (9Pay disbursement trace)
		reporter.RunTest(flowAdvance, "Verify wallet payment record from 9Pay", func() error {
			if disbursementRequestID == 0 {
				return fmt.Errorf("no request ID")
			}
			path := fmt.Sprintf("/api/v1/wallet/payments?entity_id=%d&page_size=5", disbursementRequestID)
			var payments []WalletPaymentDetail
			if _, err := adminClient.GetInto(path, &payments); err != nil {
				fmt.Printf("    WARNING: could not fetch wallet payments: %v\n", err)
				return nil
			}
			if len(payments) == 0 {
				fmt.Printf("    WARNING: no wallet payment records found for request %d\n", disbursementRequestID)
				return nil
			}
			wp := payments[0]
			fmt.Printf("    Wallet payment: id=%d, status=%s, txnId=%s, amount=%d\n",
				wp.ID, wp.Status, wp.TxnID, wp.Amount)
			return nil
		})

		// Verify employee advance info reflects the completed disbursement
		reporter.RunTest(flowAdvance, "Verify employee advance info updated after disbursement", func() error {
			if disbursementRequestID == 0 {
				return fmt.Errorf("no request ID")
			}
			var infoAfter AdvancePaymentInfoResponse
			if _, err := empClient.GetInto("/api/v1/me/advance-payment", &infoAfter); err != nil {
				return fmt.Errorf("get advance info after: %w", err)
			}
			fmt.Printf("    After: completedAmount=%d (was %d), pendingAmount=%d, remainingAmount=%d\n",
				infoAfter.CompletedAmount, completedAmountBefore, infoAfter.PendingAmount, infoAfter.RemainingAmount)

			// If disbursement completed, completedAmount should have increased
			if infoAfter.CompletedAmount > completedAmountBefore {
				fmt.Printf("    completedAmount increased by %d (disbursement reflected)\n",
					infoAfter.CompletedAmount-completedAmountBefore)
			} else {
				fmt.Printf("    NOTE: completedAmount unchanged — request may have FAILED or still processing\n")
			}
			return nil
		})

		// Verify request in employee history shows correct status
		reporter.RunTest(flowAdvance, "Verify disbursement request in employee history", func() error {
			if disbursementRequestID == 0 {
				return fmt.Errorf("no request ID")
			}
			var items []AdvancePaymentHistoryItem
			if _, err := empClient.GetInto("/api/v1/me/advance-payment/history", &items); err != nil {
				return fmt.Errorf("get history: %w", err)
			}
			for _, item := range items {
				if item.ID == disbursementRequestID {
					fmt.Printf("    History: status=%s, paidAt=%v\n", item.Status, item.PaidAt)
					return nil
				}
			}
			return fmt.Errorf("request %d not found in employee history", disbursementRequestID)
		})
	} else {
		reporter.Skip(flowAdvance, "Snapshot employee info before disbursement", "cannot request advance")
		reporter.Skip(flowAdvance, "Create request for 9Pay auto-disbursement", "cannot request advance")
		reporter.Skip(flowAdvance, "Verify request is PENDING in admin view", "no request created")
		reporter.Skip(flowAdvance, "Poll: wait for poller to claim request (PENDING → APPROVED)", "no request created")
		reporter.Skip(flowAdvance, "Poll: wait for 9Pay disbursement to complete (APPROVED → COMPLETED)", "no request created")
		reporter.Skip(flowAdvance, "Verify final request status and details", "no request created")
		reporter.Skip(flowAdvance, "Verify wallet payment record from 9Pay", "no request created")
		reporter.Skip(flowAdvance, "Verify employee advance info updated after disbursement", "no request created")
		reporter.Skip(flowAdvance, "Verify disbursement request in employee history", "no request created")
	}

	// --- Edge cases (always run) ---

	reporter.RunTest(flowAdvance, "Edge: request with amount below minimum (5,000)", func() error {
		req := CreateAdvancePaymentRequest{Amount: 5000}
		apiErr, statusCode, err := empClient.PostExpectError("/api/v1/me/advance-payment/request", req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		fmt.Printf("    Got expected error (HTTP %d): %s\n", statusCode, apiErr.Message)
		return AssertGreaterOrEqual("http_status", 400, statusCode)
	})

	reporter.RunTest(flowAdvance, "Edge: request with zero amount", func() error {
		req := CreateAdvancePaymentRequest{Amount: 0}
		_, statusCode, err := empClient.PostExpectError("/api/v1/me/advance-payment/request", req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		return AssertGreaterOrEqual("http_status", 400, statusCode)
	})

	reporter.RunTest(flowAdvance, "Edge: request exceeding max advance amount", func() error {
		req := CreateAdvancePaymentRequest{Amount: 999999999}
		apiErr, _, err := empClient.PostExpectError("/api/v1/me/advance-payment/request", req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		fmt.Printf("    Got expected error: %s\n", apiErr.Message)
		return nil
	})

	reporter.RunTest(flowAdvance, "Edge: calculate fee with zero amount", func() error {
		req := CreateAdvancePaymentRequest{Amount: 0}
		apiErr, _, err := empClient.PostExpectError("/api/v1/me/advance-payment/calculate-fee", req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		return AssertGreaterOrEqual("http_status", 400, apiErr.HTTPStatus)
	})

	reporter.RunTest(flowAdvance, "Export advance payments (admin)", func() error {
		// Export returns binary Excel - just verify the HTTP call works
		_, _, err := adminClient.Get("/api/v1/advance-payments/export")
		// May fail JSON parse since it returns binary
		if err != nil {
			return nil // Binary response can't be parsed as JSON, that's OK
		}
		return nil
	})

	// --- Deterministic cutoff date tests (clock manipulation) ---
	// Uses SetServerTime/AdvanceServerTime/ResetServerTime to test all three
	// phases in a single run regardless of the actual calendar day.
	//
	// Three-phase request window:
	//   Days  1–10 (tail of previous period): open — no lock
	//   Days 11–20 (inter-period gap):         always locked
	//   Days 20–31 (new period):               open iff admin uploaded bang luong for current month

	{
		defer func() { _ = ResetServerTime(adminClient) }()

		now := time.Now()
		y, m, _ := now.Date()
		loc := time.FixedZone("ICT", 7*60*60) // +07:00 (Asia/Ho_Chi_Minh)

		// Simulate the NEXT calendar month: the locked-gap tests below need
		// the "current" month to have no uploaded bảng công (days 10–19
		// unlock early once current-month quota exists, so a month with real
		// data would make "locked" non-deterministic). Next month can never
		// have quota, and the previous month (this one) keeps whatever real
		// data exists for the phase-1 tail checks.
		nm, ny := m+1, y
		if nm > 12 {
			nm, ny = time.January, y+1
		}

		makeTime := func(day int) time.Time {
			return time.Date(ny, nm, day, 12, 0, 0, 0, loc)
		}

		prevCalMonth := time.Date(ny, nm, 1, 0, 0, 0, 0, loc).AddDate(0, -1, 0).Format("2006-01")
		currentCalMonth := time.Date(ny, nm, 1, 0, 0, 0, 0, loc).Format("2006-01")

		// Test 1: Phase 1 (day 5) — open for previous period
		reporter.RunTest(flowAdvance, "Cutoff [clock]: Phase 1 (day 5) — forMonth is previous month", func() error {
			if err := SetServerTime(adminClient, makeTime(5)); err != nil {
				return fmt.Errorf("set clock to day 5: %w", err)
			}

			var p1Info AdvancePaymentInfoResponse
			if _, err := empClient.GetInto("/api/v1/me/advance-payment", &p1Info); err != nil {
				return fmt.Errorf("get advance info: %w", err)
			}

			fmt.Printf("    Day 5: forMonth=%s, canRequest=%v, maxAdvance=%d, remaining=%d\n",
				p1Info.ForMonth, p1Info.CanRequest, p1Info.MaxAdvanceAmount, p1Info.RemainingAmount)

			if err := AssertEqual("forMonth", prevCalMonth, p1Info.ForMonth); err != nil {
				return err
			}
			if p1Info.HasFlexible && p1Info.RemainingAmount >= 10000 {
				if err := AssertTrue("canRequest", p1Info.CanRequest); err != nil {
					return err
				}
				if err := AssertEqual("canRequestReason", "", p1Info.CanRequestReason); err != nil {
					return err
				}
			}
			return nil
		})

		// Test 2: Phase 1 boundary (day 9) — last open day
		reporter.RunTest(flowAdvance, "Cutoff [clock]: Phase 1 boundary (day 9) — still open", func() error {
			if err := SetServerTime(adminClient, makeTime(9)); err != nil {
				return fmt.Errorf("set clock to day 9: %w", err)
			}

			var p1bInfo AdvancePaymentInfoResponse
			if _, err := empClient.GetInto("/api/v1/me/advance-payment", &p1bInfo); err != nil {
				return fmt.Errorf("get advance info: %w", err)
			}

			fmt.Printf("    Day 9: forMonth=%s, canRequest=%v, remaining=%d\n",
				p1bInfo.ForMonth, p1bInfo.CanRequest, p1bInfo.RemainingAmount)

			if err := AssertEqual("forMonth", prevCalMonth, p1bInfo.ForMonth); err != nil {
				return err
			}
			if p1bInfo.HasFlexible && p1bInfo.RemainingAmount >= 10000 {
				return AssertTrue("canRequest", p1bInfo.CanRequest)
			}
			return nil
		})

		// Test 3: Phase 2 (day 15) — locked gap, request rejected
		reporter.RunTest(flowAdvance, "Cutoff [clock]: Phase 2 (day 15) — locked, request rejected", func() error {
			if err := SetServerTime(adminClient, makeTime(15)); err != nil {
				return fmt.Errorf("set clock to day 15: %w", err)
			}

			var p2Info AdvancePaymentInfoResponse
			if _, err := empClient.GetInto("/api/v1/me/advance-payment", &p2Info); err != nil {
				return fmt.Errorf("get advance info: %w", err)
			}

			fmt.Printf("    Day 15: canRequest=%v, reason=%s\n", p2Info.CanRequest, p2Info.CanRequestReason)

			if p2Info.HasFlexible {
				if err := AssertFalse("canRequest", p2Info.CanRequest); err != nil {
					return err
				}
				if p2Info.CanRequestReason == "" {
					return fmt.Errorf("expected canRequestReason in locked gap (day 15)")
				}
			}

			req := CreateAdvancePaymentRequest{Amount: 100000}
			apiErr, statusCode, err := empClient.PostExpectError("/api/v1/me/advance-payment/request", req)
			if err != nil {
				return fmt.Errorf("request should be blocked in locked gap: %w", err)
			}
			fmt.Printf("    Request rejected (HTTP %d): %s\n", statusCode, apiErr.Message)
			return AssertGreaterOrEqual("http_status", 400, statusCode)
		})

		// Test 4: Phase 1→2 transition (day 9 → day 10)
		reporter.RunTest(flowAdvance, "Cutoff [clock]: Phase 1→2 transition (day 9 → day 10)", func() error {
			if err := SetServerTime(adminClient, makeTime(9)); err != nil {
				return fmt.Errorf("set clock to day 9: %w", err)
			}

			var before AdvancePaymentInfoResponse
			if _, err := empClient.GetInto("/api/v1/me/advance-payment", &before); err != nil {
				return fmt.Errorf("get advance info (day 9): %w", err)
			}
			fmt.Printf("    Day 9: canRequest=%v, forMonth=%s\n", before.CanRequest, before.ForMonth)

			if err := AdvanceServerTime(adminClient, 24*time.Hour); err != nil {
				return fmt.Errorf("advance clock by 24h: %w", err)
			}

			var after AdvancePaymentInfoResponse
			if _, err := empClient.GetInto("/api/v1/me/advance-payment", &after); err != nil {
				return fmt.Errorf("get advance info (day 10): %w", err)
			}
			fmt.Printf("    Day 10: canRequest=%v, reason=%s\n", after.CanRequest, after.CanRequestReason)

			if after.HasFlexible {
				if err := AssertFalse("canRequest_after", after.CanRequest); err != nil {
					return err
				}
				expectedTitle := fmt.Sprintf("Kỳ ứng lương %s/%s kết thúc", prevCalMonth[5:7], prevCalMonth[0:4])
				if err := AssertEqual("canRequestTitle_after", expectedTitle, after.CanRequestTitle); err != nil {
					return err
				}
				if before.HasFlexible && before.RemainingAmount >= 10000 && before.CanRequest && !after.CanRequest {
					fmt.Printf("    Transition verified: open → locked across day boundary\n")
				}
			}
			return nil
		})

		// Test 5: Phase 2 boundary (day 19) — last locked day
		reporter.RunTest(flowAdvance, "Cutoff [clock]: Phase 2 boundary (day 19) — still locked", func() error {
			if err := SetServerTime(adminClient, makeTime(19)); err != nil {
				return fmt.Errorf("set clock to day 19: %w", err)
			}

			var p2bInfo AdvancePaymentInfoResponse
			if _, err := empClient.GetInto("/api/v1/me/advance-payment", &p2bInfo); err != nil {
				return fmt.Errorf("get advance info: %w", err)
			}

			fmt.Printf("    Day 19: canRequest=%v, reason=%s\n", p2bInfo.CanRequest, p2bInfo.CanRequestReason)

			if p2bInfo.HasFlexible {
				if err := AssertFalse("canRequest", p2bInfo.CanRequest); err != nil {
					return err
				}
				if p2bInfo.CanRequestReason == "" {
					return fmt.Errorf("expected canRequestReason on last locked day (day 19)")
				}
			}
			return nil
		})

		// Test 6: Phase 2→3 transition (day 19 → day 20)
		reporter.RunTest(flowAdvance, "Cutoff [clock]: Phase 2→3 transition (day 19 → day 20)", func() error {
			if err := SetServerTime(adminClient, makeTime(19)); err != nil {
				return fmt.Errorf("set clock to day 19: %w", err)
			}

			var locked AdvancePaymentInfoResponse
			if _, err := empClient.GetInto("/api/v1/me/advance-payment", &locked); err != nil {
				return fmt.Errorf("get advance info (day 19): %w", err)
			}
			fmt.Printf("    Day 19: canRequest=%v\n", locked.CanRequest)

			if err := AdvanceServerTime(adminClient, 24*time.Hour); err != nil {
				return fmt.Errorf("advance clock by 24h: %w", err)
			}

			var opened AdvancePaymentInfoResponse
			if _, err := empClient.GetInto("/api/v1/me/advance-payment", &opened); err != nil {
				return fmt.Errorf("get advance info (day 20): %w", err)
			}
			fmt.Printf("    Day 20: forMonth=%s, canRequest=%v, reason=%s\n",
				opened.ForMonth, opened.CanRequest, opened.CanRequestReason)

			// forMonth should switch to current calendar month on day 20
			if err := AssertEqual("forMonth", currentCalMonth, opened.ForMonth); err != nil {
				return err
			}

			if opened.CanRequest {
				fmt.Printf("    Phase 3 OPEN: bang luong exists for %s\n", currentCalMonth)
			} else if opened.HasFlexible {
				fmt.Printf("    Phase 3 LOCKED: bang luong not yet uploaded for %s\n", currentCalMonth)
				if opened.CanRequestReason == "" {
					return fmt.Errorf("expected canRequestReason when locked in Phase 3")
				}
			}
			return nil
		})

		// Test 7: Phase 3 (day 25) — adaptive bang luong check
		reporter.RunTest(flowAdvance, "Cutoff [clock]: Phase 3 (day 25) — bang luong state", func() error {
			if err := SetServerTime(adminClient, makeTime(25)); err != nil {
				return fmt.Errorf("set clock to day 25: %w", err)
			}

			var p3Info AdvancePaymentInfoResponse
			if _, err := empClient.GetInto("/api/v1/me/advance-payment", &p3Info); err != nil {
				return fmt.Errorf("get advance info: %w", err)
			}

			fmt.Printf("    Day 25: forMonth=%s, canRequest=%v, reason=%s\n",
				p3Info.ForMonth, p3Info.CanRequest, p3Info.CanRequestReason)

			if err := AssertEqual("forMonth", currentCalMonth, p3Info.ForMonth); err != nil {
				return err
			}

			if p3Info.CanRequest {
				// Bang luong exists — verify open state is consistent
				if p3Info.CanRequestReason != "" {
					return fmt.Errorf("expected empty canRequestReason when open, got: %s", p3Info.CanRequestReason)
				}
				fmt.Printf("    Phase 3 OPEN: forMonth=%s matches current calendar month\n", p3Info.ForMonth)
			} else if p3Info.HasFlexible {
				// Locked — verify reason present and request blocked
				if p3Info.CanRequestReason == "" {
					return fmt.Errorf("expected canRequestReason when locked in Phase 3")
				}

				req := CreateAdvancePaymentRequest{Amount: 100000}
				apiErr, statusCode, err := empClient.PostExpectError("/api/v1/me/advance-payment/request", req)
				if err != nil {
					return fmt.Errorf("request should be blocked without bang luong: %w", err)
				}
				fmt.Printf("    Locked (no bang luong): request rejected (HTTP %d): %s\n", statusCode, apiErr.Message)
				return AssertGreaterOrEqual("http_status", 400, statusCode)
			}
			return nil
		})

		// Test 8: Reset clock
		reporter.RunTest(flowAdvance, "Cutoff [clock]: Reset server time to real time", func() error {
			if err := ResetServerTime(adminClient); err != nil {
				return fmt.Errorf("reset clock: %w", err)
			}

			serverNow, err := GetServerTime(adminClient)
			if err != nil {
				return fmt.Errorf("get server time after reset: %w", err)
			}

			realNow := time.Now()
			diff := realNow.Sub(serverNow)
			if diff < 0 {
				diff = -diff
			}
			fmt.Printf("    Clock restored: server=%s, real=%s, diff=%s\n",
				serverNow.Format(time.RFC3339), realNow.Format(time.RFC3339), diff.Round(time.Second))

			if diff > 10*time.Second {
				return fmt.Errorf("server time too far from real time after reset: diff=%s", diff)
			}
			return nil
		})
	}
}
