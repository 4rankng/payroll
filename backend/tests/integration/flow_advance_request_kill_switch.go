package main

import (
	"fmt"
)

const flowAdvKillSwitch = "AdvanceRequestKillSwitch"

// runAdvanceRequestKillSwitchTests verifies the per-employee advance request
// kill switch ("tạm ngừng ứng lương"): admin disables an employee's
// advance_request_enabled flag, the employee's info endpoints report the
// paused state, request creation is rejected, and re-enabling restores the
// previous state. Leaves the employee re-enabled at the end.
func runAdvanceRequestKillSwitchTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW 6.7: Advance Request Kill Switch (tạm ngừng ứng lương)")

	if data.EmployeeForAdvance == nil || data.EmployeeTokenForAdv == "" {
		reporter.Skip(flowAdvKillSwitch, "All kill switch tests", "no employee user account found")
		return
	}

	empClient := client.WithToken(data.EmployeeTokenForAdv)
	adminClient := client.WithToken(data.AdminToken)
	employeeID := data.EmployeeForAdvance.ID

	var projectID uint

	reporter.RunTest(flowAdvKillSwitch, "Employees list finds the test employee's project", func() error {
		var items []EmployeeAdvanceItem
		if _, err := adminClient.GetInto("/api/v1/advance-payments/employees?pageSize=100", &items); err != nil {
			return fmt.Errorf("list employees: %w", err)
		}
		for _, it := range items {
			if it.EmployeeID == employeeID && it.Project != nil {
				projectID = it.Project.ID
				fmt.Printf("    employee %d -> project %d (%s), advance_request_enabled=%v\n",
					employeeID, projectID, it.Project.Name, it.Project.AdvanceRequestEnabled)
				return nil
			}
		}
		return fmt.Errorf("employee %d not found in advance-payments/employees", employeeID)
	})

	if projectID == 0 {
		reporter.Skip(flowAdvKillSwitch, "Remaining kill switch tests", "employee not in flexible employee list")
		return
	}

	togglePath := fmt.Sprintf("/api/v1/projects/%d/employees/%d/advance-request-enabled", projectID, employeeID)

	reporter.RunTest(flowAdvKillSwitch, "Admin disables advance requests for the employee", func() error {
		var resp map[string]any
		if _, err := adminClient.PatchInto(togglePath, map[string]bool{"advance_request_enabled": false}, &resp); err != nil {
			return fmt.Errorf("disable patch: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowAdvKillSwitch, "Employee advance info shows paused state", func() error {
		var info AdvancePaymentInfoResponse
		if _, err := empClient.GetInto("/api/v1/me/advance-payment", &info); err != nil {
			return fmt.Errorf("get info: %w", err)
		}
		if info.CanRequest {
			return fmt.Errorf("expected canRequest=false while paused")
		}
		if info.CanRequestTitle != "Tạm ngừng ứng lương" {
			return fmt.Errorf("canRequestTitle = %q, want %q", info.CanRequestTitle, "Tạm ngừng ứng lương")
		}
		if info.CanRequestReason == "" {
			return fmt.Errorf("expected non-empty canRequestReason while paused")
		}
		fmt.Printf("    paused: title=%q reason=%q\n", info.CanRequestTitle, info.CanRequestReason)
		return nil
	})

	reporter.RunTest(flowAdvKillSwitch, "Employee advance request is rejected while paused", func() error {
		req := CreateAdvancePaymentRequest{Amount: 100000}
		apiErr, statusCode, err := empClient.PostExpectError("/api/v1/me/advance-payment/request", req)
		if err != nil {
			return fmt.Errorf("request should be rejected while paused: %w", err)
		}
		fmt.Printf("    rejected (HTTP %d): %s\n", statusCode, apiErr.Message)
		return AssertContains("error message", apiErr.Message, "tạm ngừng")
	})

	reporter.RunTest(flowAdvKillSwitch, "Check-in advance info shows paused state (gate precedes window branches)", func() error {
		var info AdvancePaymentInfoResponse
		if _, err := empClient.GetInto("/api/v1/me/check-in-advance", &info); err != nil {
			return fmt.Errorf("get check-in info: %w", err)
		}
		if info.CanRequest {
			return fmt.Errorf("expected canRequest=false while paused")
		}
		if info.CanRequestTitle != "Tạm ngừng ứng lương" {
			return fmt.Errorf("canRequestTitle = %q, want %q (kill switch must win over window/not-enabled reasons)", info.CanRequestTitle, "Tạm ngừng ứng lương")
		}
		return nil
	})

	reporter.RunTest(flowAdvKillSwitch, "Employee token cannot toggle the kill switch", func() error {
		var resp map[string]any
		if _, err := empClient.PatchInto(togglePath, map[string]bool{"advance_request_enabled": true}, &resp); err == nil {
			return fmt.Errorf("employee PATCH should be rejected (401/403)")
		}
		return nil
	})

	reporter.RunTest(flowAdvKillSwitch, "Repeat disable is an idempotent no-op", func() error {
		var resp map[string]any
		if _, err := adminClient.PatchInto(togglePath, map[string]bool{"advance_request_enabled": false}, &resp); err != nil {
			return fmt.Errorf("repeat disable patch: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowAdvKillSwitch, "Admin re-enables and the paused state clears", func() error {
		var resp map[string]any
		if _, err := adminClient.PatchInto(togglePath, map[string]bool{"advance_request_enabled": true}, &resp); err != nil {
			return fmt.Errorf("re-enable patch: %w", err)
		}

		var info AdvancePaymentInfoResponse
		if _, err := empClient.GetInto("/api/v1/me/advance-payment", &info); err != nil {
			return fmt.Errorf("get info after re-enable: %w", err)
		}
		if info.CanRequestTitle == "Tạm ngừng ứng lương" {
			return fmt.Errorf("paused title still present after re-enable")
		}
		fmt.Printf("    re-enabled: canRequest=%v title=%q\n", info.CanRequest, info.CanRequestTitle)
		return nil
	})
}
