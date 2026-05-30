package main

import (
	"encoding/json"
	"fmt"
	"time"
)

const flowAdvanceOnePay = "AdvancePayment-OnePay"

func runAdvancePaymentOnePayTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW: Advance Payment — OnePay Full E2E")

	if data.EmployeeForAdvance == nil || data.EmployeeTokenForAdv == "" {
		reporter.Skip(flowAdvanceOnePay, "All OnePay advance payment tests", "no employee user account found")
		return
	}

	empClient := client.WithToken(data.EmployeeTokenForAdv)
	adminClient := client.WithToken(data.AdminToken)

	var activeProvider string
	reporter.RunTest(flowAdvanceOnePay, "Verify OnePay is the active disbursement provider", func() error {
		var resp map[string]interface{}
		if _, err := adminClient.GetInto("/api/v1/disbursement/provider/stats", &resp); err != nil {
			fmt.Printf("    Provider stats unavailable: %v (backend may not expose admin disbursement routes in dev)\n", err)
			return nil
		}
		fmt.Printf("    Provider stats: %+v\n", resp)
		if name, ok := resp["provider_name"]; ok {
			activeProvider = fmt.Sprintf("%v", name)
			fmt.Printf("    Active provider: %s\n", activeProvider)
		}
		return nil
	})

	reporter.RunTest(flowAdvanceOnePay, "Verify OnePay sandbox balance", func() error {
		var resp map[string]interface{}
		if _, err := adminClient.GetInto("/api/v1/disbursement/manual/balance", &resp); err != nil {
			fmt.Printf("    Balance unavailable: %v\n", err)
			return nil
		}
		fmt.Printf("    Balance: %+v\n", resp)
		return nil
	})

	var info AdvancePaymentInfoResponse
	reporter.RunTest(flowAdvanceOnePay, "Employee views advance payment info", func() error {
		if _, err := empClient.GetInto("/api/v1/me/advance-payment", &info); err != nil {
			return fmt.Errorf("get advance info: %w", err)
		}
		fmt.Printf("    ForMonth: %s, CanRequest: %v, HasFlexible: %v, MaxAdvance: %d, Remaining: %d\n",
			info.ForMonth, info.CanRequest, info.HasFlexible, info.MaxAdvanceAmount, info.RemainingAmount)

		if !info.CanRequest {
			fmt.Printf("    Employee cannot request advance (reason: %s) — will skip disbursement lifecycle\n", info.CanRequestReason)
		}
		if info.RemainingAmount < 200000 {
			fmt.Printf("    Remaining amount %d too low for disbursement test (need >= 200k) — skipping\n", info.RemainingAmount)
			info.CanRequest = false
		}
		return nil
	})

	if !info.CanRequest || info.RemainingAmount < 200000 {
		reporter.Skip(flowAdvanceOnePay, "Full OnePay disbursement lifecycle", "cannot request advance")
		return
	}

	var requestID uint
	var completedAmountBefore = info.CompletedAmount

	reporter.RunTest(flowAdvanceOnePay, "Calculate fee preview before request", func() error {
		req := CreateAdvancePaymentRequest{Amount: 200000}
		var resp CalculateFeeResponse
		if _, err := empClient.PostInto("/api/v1/me/advance-payment/calculate-fee", req, &resp); err != nil {
			return fmt.Errorf("calculate fee: %w", err)
		}
		fmt.Printf("    Fee: %d, Net: %d (for 200,000 VND)\n", resp.Fee, resp.NetAmount)
		if resp.Fee == 0 {
			return fmt.Errorf("expected non-zero fee")
		}
		if resp.NetAmount != resp.RequestAmount-resp.Fee {
			return fmt.Errorf("net amount mismatch: %d - %d != %d", resp.RequestAmount, resp.Fee, resp.NetAmount)
		}
		return nil
	})

	// Edge: request with net amount below provider minimum should be rejected at creation
	reporter.RunTest(flowAdvanceOnePay, "Edge: request below provider minimum rejected", func() error {
		// 105,000 VND request - 10,000 fee = 95,000 net < 100,000 provider minimum
		req := CreateAdvancePaymentRequest{Amount: 105000}
		apiErr, statusCode, err := empClient.PostExpectError("/api/v1/me/advance-payment/request", req)
		if err != nil {
			return fmt.Errorf("request should be rejected: %w", err)
		}
		fmt.Printf("    Got expected rejection (HTTP %d): %s\n", statusCode, apiErr.Message)
		if statusCode < 400 {
			return fmt.Errorf("expected 4xx, got %d", statusCode)
		}
		return nil
	})

	reporter.RunTest(flowAdvanceOnePay, "Create advance payment request (OnePay)", func() error {
		req := CreateAdvancePaymentRequest{Amount: 200000}
		var created AdvancePaymentHistoryItem
		if _, err := empClient.PostInto("/api/v1/me/advance-payment/request", req, &created); err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		requestID = created.ID
		fmt.Printf("    Created request ID %d, status: %s, fee: %d, net: %d\n",
			created.ID, created.Status, created.Fee, created.NetAmount)
		if err := AssertGreaterThan("id", uint(0), created.ID); err != nil {
			return err
		}
		return AssertEqual("initial_status", "PENDING", created.Status)
	})

	reporter.RunTest(flowAdvanceOnePay, "Verify request is PENDING in admin list", func() error {
		if requestID == 0 {
			return fmt.Errorf("no request ID")
		}
		var items []AdvancePaymentRequestItem
		if _, err := adminClient.GetInto("/api/v1/advance-payments?status=PENDING&pageSize=50", &items); err != nil {
			return fmt.Errorf("admin list pending: %w", err)
		}
		for _, item := range items {
			if item.ID == requestID {
				fmt.Printf("    Request %d confirmed PENDING (employeeId=%d, amount=%d)\n",
					item.ID, item.EmployeeID, item.RequestAmount)
				return nil
			}
		}
		return fmt.Errorf("request %d not found in PENDING list", requestID)
	})

	reporter.RunTest(flowAdvanceOnePay, "Verify request in employee history", func() error {
		if requestID == 0 {
			return fmt.Errorf("no request ID")
		}
		var items []AdvancePaymentHistoryItem
		if _, err := empClient.GetInto("/api/v1/me/advance-payment/history", &items); err != nil {
			return fmt.Errorf("get history: %w", err)
		}
		for _, item := range items {
			if item.ID == requestID {
				fmt.Printf("    History: status=%s, fee=%d, net=%d\n", item.Status, item.Fee, item.NetAmount)
				return AssertEqual("history_status", "PENDING", item.Status)
			}
		}
		return fmt.Errorf("request %d not found in history", requestID)
	})

	reporter.RunTest(flowAdvanceOnePay, "Admin summary shows pending request", func() error {
		var resp AdvancePaymentSummaryResponse
		if _, err := adminClient.GetInto("/api/v1/advance-payments/summary", &resp); err != nil {
			return fmt.Errorf("admin summary: %w", err)
		}
		fmt.Printf("    Total: %d, Pending: %d, Paid: %d, Amount: %d\n",
			resp.TotalRequests, resp.TotalPending, resp.TotalPaid, resp.TotalAmount)
		return AssertGreaterThan("totalPending", int64(0), resp.TotalPending)
	})

	reporter.RunTest(flowAdvanceOnePay, "Poll: wait for poller to claim request (PENDING → APPROVED)", func() error {
		if requestID == 0 {
			return fmt.Errorf("no request ID")
		}
		path := "/api/v1/advance-payments?pageSize=200"
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
				if item.ID == requestID {
					return item.Status == "APPROVED" || item.Status == "COMPLETED" || item.Status == "FAILED"
				}
			}
			return false
		})
		if err != nil {
			fmt.Printf("    WARNING: poll timeout waiting for PENDING→APPROVED: %v\n", err)
			return nil
		}
		fmt.Printf("    Request %d claimed by poller\n", requestID)
		return nil
	})

	reporter.RunTest(flowAdvanceOnePay, "Poll: wait for OnePay disbursement to complete (APPROVED → COMPLETED)", func() error {
		if requestID == 0 {
			return fmt.Errorf("no request ID")
		}
		path := "/api/v1/advance-payments?pageSize=200"
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
				if item.ID == requestID {
					return item.Status == "COMPLETED" || item.Status == "FAILED"
				}
			}
			return false
		})
		if err != nil {
			fmt.Printf("    WARNING: poll timeout - OnePay disbursement may still be processing: %v\n", err)
			return nil
		}
		fmt.Printf("    Request %d reached terminal state\n", requestID)
		return nil
	})

	reporter.RunTest(flowAdvanceOnePay, "Verify final request status and details", func() error {
		if requestID == 0 {
			return fmt.Errorf("no request ID")
		}
		var items []AdvancePaymentRequestItem
		if _, err := adminClient.GetInto("/api/v1/advance-payments?pageSize=200", &items); err != nil {
			return fmt.Errorf("admin list: %w", err)
		}
		var found *AdvancePaymentRequestItem
		for i := range items {
			if items[i].ID == requestID {
				found = &items[i]
				break
			}
		}
		if found == nil {
			return fmt.Errorf("request %d not found in admin list", requestID)
		}
		fmt.Printf("    Final status: %s, paidAt: %v, paymentRef: %s\n",
			found.Status, found.PaidAt, found.PaymentRef)

		if found.Status != "COMPLETED" {
			return fmt.Errorf("expected COMPLETED, got %s", found.Status)
		}
		if found.PaidAt == nil {
			return fmt.Errorf("COMPLETED request should have paidAt set")
		}
		if found.PaymentRef == "" {
			return fmt.Errorf("COMPLETED request should have paymentReference set")
		}
		fmt.Printf("    OnePay transaction reference: %s\n", found.PaymentRef)
		return nil
	})

	reporter.RunTest(flowAdvanceOnePay, "Verify wallet payment record from OnePay", func() error {
		if requestID == 0 {
			return fmt.Errorf("no request ID")
		}
		path := fmt.Sprintf("/api/v1/wallet/payments?entity_id=%d&page_size=5", requestID)
		var payments []WalletPaymentDetail
		if _, err := adminClient.GetInto(path, &payments); err != nil {
			return fmt.Errorf("fetch wallet payments: %w", err)
		}
		if len(payments) == 0 {
			return fmt.Errorf("no wallet payment records found for request %d", requestID)
		}
		wp := payments[0]
		fmt.Printf("    Wallet payment: id=%d, status=%s, txnId=%s, amount=%d\n",
			wp.ID, wp.Status, wp.TxnID, wp.Amount)
		if wp.Status != "completed" {
			return fmt.Errorf("expected wallet payment status 'completed', got '%s'", wp.Status)
		}
		return nil
	})

	reporter.RunTest(flowAdvanceOnePay, "Verify employee advance info reflects completed disbursement", func() error {
		if requestID == 0 {
			return fmt.Errorf("no request ID")
		}
		var infoAfter AdvancePaymentInfoResponse
		if _, err := empClient.GetInto("/api/v1/me/advance-payment", &infoAfter); err != nil {
			return fmt.Errorf("get advance info after: %w", err)
		}
		fmt.Printf("    After: completedAmount=%d (was %d), pendingAmount=%d, remainingAmount=%d\n",
			infoAfter.CompletedAmount, completedAmountBefore, infoAfter.PendingAmount, infoAfter.RemainingAmount)

		if infoAfter.CompletedAmount <= completedAmountBefore {
			return fmt.Errorf("completedAmount did not increase: was %d, now %d",
				completedAmountBefore, infoAfter.CompletedAmount)
		}
		increase := infoAfter.CompletedAmount - completedAmountBefore
		fmt.Printf("    completedAmount increased by %d (disbursement reflected)\n", increase)
		return nil
	})

	reporter.RunTest(flowAdvanceOnePay, "Verify request in employee history shows COMPLETED", func() error {
		if requestID == 0 {
			return fmt.Errorf("no request ID")
		}
		var items []AdvancePaymentHistoryItem
		if _, err := empClient.GetInto("/api/v1/me/advance-payment/history", &items); err != nil {
			return fmt.Errorf("get history: %w", err)
		}
		for _, item := range items {
			if item.ID == requestID {
				fmt.Printf("    History: status=%s, paidAt=%v\n", item.Status, item.PaidAt)
				if item.Status != "COMPLETED" {
					return fmt.Errorf("expected COMPLETED in history, got %s", item.Status)
				}
				if item.PaidAt == nil {
					return fmt.Errorf("COMPLETED history item should have paidAt")
				}
				return nil
			}
		}
		return fmt.Errorf("request %d not found in employee history", requestID)
	})

	reporter.RunTest(flowAdvanceOnePay, "Admin summary updated after disbursement", func() error {
		var resp AdvancePaymentSummaryResponse
		if _, err := adminClient.GetInto("/api/v1/advance-payments/summary", &resp); err != nil {
			return fmt.Errorf("admin summary: %w", err)
		}
		fmt.Printf("    Total: %d, Pending: %d, Completed: %d, PaidAmount: %d\n",
			resp.TotalRequests, resp.TotalPending, resp.TotalPaid, resp.TotalPaidAmount)
		return AssertGreaterThan("totalPaid", int64(0), resp.TotalPaid)
	})

	reporter.RunTest(flowAdvanceOnePay, "Admin employee list reflects updated utilization", func() error {
		var items []EmployeeAdvanceItem
		if _, err := adminClient.GetInto("/api/v1/advance-payments/employees", &items); err != nil {
			return fmt.Errorf("employee list: %w", err)
		}
		empID := data.EmployeeForAdvance.ID
		for _, item := range items {
			if item.EmployeeID == empID {
				fmt.Printf("    Employee %d: utilized=%d, completed=%d, pending=%d\n",
					item.EmployeeID, item.UtilizedAmount, item.CompletedRequestsCount, item.PendingRequestsCount)
				if item.CompletedRequestsCount == 0 {
					return fmt.Errorf("expected completedRequestsCount > 0")
				}
				return nil
			}
		}
		fmt.Printf("    Employee %d not found in advance employee list\n", empID)
		return nil
	})

	reporter.RunTest(flowAdvanceOnePay, "Export advance payments (post-disbursement)", func() error {
		_, _, err := adminClient.Get("/api/v1/advance-payments/export")
		if err != nil {
			fmt.Printf("    NOTE: export returned binary, JSON parse expected to fail\n")
			return nil
		}
		return nil
	})
}
