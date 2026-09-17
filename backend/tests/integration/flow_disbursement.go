package main

import (
	"fmt"
	"net/http"
)

const flowDisbursement = "Disbursement"

func runDisbursementTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Disbursement")

	admin := client.WithToken(data.AdminToken)

	reporter.RunTest(flowDisbursement, "Get provider transaction stats", func() error {
		var resp interface{}
		if _, err := admin.GetInto("/api/v1/admin/provider-transactions/stats", &resp); err != nil {
			return fmt.Errorf("provider stats: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowDisbursement, "Get manual disbursement banks", func() error {
		var resp interface{}
		if _, err := admin.GetInto("/api/v1/admin/manual-disbursement/banks", &resp); err != nil {
			return fmt.Errorf("manual banks: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowDisbursement, "Get manual disbursement balance", func() error {
		var resp interface{}
		if _, err := admin.GetInto("/api/v1/admin/manual-disbursement/balance", &resp); err != nil {
			return fmt.Errorf("manual balance: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowDisbursement, "List disbursements", func() error {
		var resp interface{}
		if _, err := admin.GetInto("/api/v1/admin/manual-disbursement?pageSize=10", &resp); err != nil {
			return fmt.Errorf("disbursements list: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowDisbursement, "Edge: check account with invalid bank code", func() error {
		body := CheckAccountRequest{
			BankCode:  "INVALID",
			AccountNo: "0000000000",
		}
		_, status, err := admin.PostExpectError("/api/v1/admin/manual-disbursement/check-account", body)
		if err != nil {
			return fmt.Errorf("check malformed account: %w", err)
		}
		return AssertEqual("status", http.StatusBadRequest, status)
	})

	reporter.RunTest(flowDisbursement, "Employee account lookup requires an employee", func() error {
		_, statusCode, err := admin.Post(
			"/api/v1/admin/manual-disbursement/employee-account-check",
			EmployeeAccountCheckRequest{},
		)
		if err != nil {
			return err
		}
		if statusCode != 400 {
			return fmt.Errorf("expected 400 for missing employee_id, got %d", statusCode)
		}
		return nil
	})

	reporter.RunTest(flowDisbursement, "Employee account lookup is admin only", func() error {
		if len(data.Partners) == 0 || data.Partners[0].Token == "" {
			return fmt.Errorf("partner fixture is required for authorization test")
		}
		partner := client.WithToken(data.Partners[0].Token)
		_, statusCode, err := partner.Post(
			"/api/v1/admin/manual-disbursement/employee-account-check",
			EmployeeAccountCheckRequest{EmployeeID: 1},
		)
		if err != nil {
			return err
		}
		if statusCode != 403 {
			return fmt.Errorf("expected 403 for partner lookup, got %d", statusCode)
		}
		return nil
	})
}
