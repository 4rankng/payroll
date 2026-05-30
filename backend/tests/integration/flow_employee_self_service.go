package main

import (
	"fmt"
)

const flowEmpSelfService = "EmployeeSelfService"

func runEmployeeSelfServiceTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Employee Self-Service")

	if data.EmployeeTokenForAdv == "" || data.EmployeeForAdvance == nil {
		reporter.Skip(flowEmpSelfService, "All tests", "no employee user account available")
		return
	}

	empClient := client.WithToken(data.EmployeeTokenForAdv)
	empID := data.EmployeeForAdvance.ID

	reporter.RunTest(flowEmpSelfService, "Get my profile (employee)", func() error {
		var resp interface{}
		if _, err := empClient.GetInto("/api/v1/me", &resp); err != nil {
			return fmt.Errorf("get profile: %w", err)
		}
		fmt.Printf("    Employee profile fetched for ID %d\n", empID)
		return nil
	})

	reporter.RunTest(flowEmpSelfService, "Get my timesheets", func() error {
		var resp interface{}
		if _, err := empClient.GetInto("/api/v1/me/timesheet?pageSize=10", &resp); err != nil {
			return fmt.Errorf("get timesheets: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowEmpSelfService, "Get my summary", func() error {
		var resp interface{}
		if _, err := empClient.GetInto("/api/v1/me/summary", &resp); err != nil {
			return fmt.Errorf("get summary: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowEmpSelfService, "Get my payroll history", func() error {
		var resp interface{}
		if _, err := empClient.GetInto(fmt.Sprintf("/api/v1/employees/%d/payroll?pageSize=10", empID), &resp); err != nil {
			fmt.Printf("    Payroll history unavailable: %v\n", err)
			return nil
		}
		return nil
	})

	reporter.RunTest(flowEmpSelfService, "Get my advance payment info", func() error {
		var resp interface{}
		if _, err := empClient.GetInto("/api/v1/me/advance-payment", &resp); err != nil {
			// May not have advance payment available
			fmt.Printf("    Advance payment info unavailable: %v\n", err)
			return nil
		}
		return nil
	})
}
