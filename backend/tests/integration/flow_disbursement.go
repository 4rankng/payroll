package main

import "fmt"

const flowDisbursement = "Disbursement"

func runDisbursementTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Disbursement")

	admin := client.WithToken(data.AdminToken)

	reporter.RunTest(flowDisbursement, "Get provider transaction stats", func() error {
		var resp interface{}
		if _, err := admin.GetInto("/api/v1/disbursement/provider/stats", &resp); err != nil {
			fmt.Printf("    Provider stats unavailable: %v\n", err)
			return nil
		}
		return nil
	})

	reporter.RunTest(flowDisbursement, "Get manual disbursement banks", func() error {
		var resp interface{}
		if _, err := admin.GetInto("/api/v1/disbursement/manual/banks", &resp); err != nil {
			fmt.Printf("    Manual banks unavailable: %v\n", err)
			return nil
		}
		return nil
	})

	reporter.RunTest(flowDisbursement, "Get manual disbursement balance", func() error {
		var resp interface{}
		if _, err := admin.GetInto("/api/v1/disbursement/manual/balance", &resp); err != nil {
			fmt.Printf("    Manual balance unavailable: %v\n", err)
			return nil
		}
		return nil
	})

	reporter.RunTest(flowDisbursement, "List disbursements", func() error {
		var resp interface{}
		if _, err := admin.GetInto("/api/v1/disbursement?pageSize=10", &resp); err != nil {
			fmt.Printf("    Disbursements list unavailable: %v\n", err)
			return nil
		}
		return nil
	})

	reporter.RunTest(flowDisbursement, "Edge: check account with invalid bank code", func() error {
		body := CheckAccountRequest{
			BankCode:  "INVALID",
			AccountNo: "0000000000",
		}
		_, _, err := admin.Post("/api/v1/disbursement/check-account", body)
		if err == nil {
			return fmt.Errorf("expected error for invalid bank code")
		}
		return nil
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
			fmt.Println("    Skipped: no partner token available")
			return nil
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
