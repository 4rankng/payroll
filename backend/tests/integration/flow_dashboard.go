package main

import (
	"fmt"
)

const flowDashboard = "Dashboard"

func runDashboardTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Dashboard Read-Only")

	admin := client.WithToken(data.AdminToken)

	endpoints := []struct {
		name string
		path string
	}{
		{"Summary", "/api/v1/dashboard/summary"},
		{"Financial overview", "/api/v1/dashboard/financial-overview"},
		{"Financial", "/api/v1/dashboard/financial"},
		{"Salary distribution", "/api/v1/dashboard/salary-distribution"},
		{"Recent activities", "/api/v1/dashboard/recent-activities?pageSize=5"},
		{"Notifications", "/api/v1/dashboard/notifications"},
		{"New employees", "/api/v1/dashboard/new-employees"},
		{"Historical", "/api/v1/dashboard/historical"},
		{"Monthly financials", "/api/v1/dashboard/monthly-financials"},
		{"Employee activity", "/api/v1/dashboard/employee-activity"},
		{"Top paid employees", "/api/v1/dashboard/top-paid-employees"},
		{"Bank usage", "/api/v1/dashboard/bank-usage"},
		{"Bank usage by project", "/api/v1/dashboard/bank-usage/projects"},
		{"Project profitability", "/api/v1/dashboard/project-profitability"},
		{"Project weekly profit", "/api/v1/dashboard/project-weekly-profit"},
	}

	for _, ep := range endpoints {
		ep := ep
		reporter.RunTest(flowDashboard, "GET "+ep.name, func() error {
			var resp interface{}
			if _, err := admin.GetInto(ep.path, &resp); err != nil {
				return fmt.Errorf("%s: %w", ep.name, err)
			}
			fmt.Printf("    OK: %s\n", ep.name)
			return nil
		})
	}

	// Partner dashboard (requires partner token)
	if len(data.Partners) > 0 {
		partner := client.WithToken(data.Partners[0].Token)

		reporter.RunTest(flowDashboard, "Partner dashboard", func() error {
			var resp interface{}
			if _, err := partner.GetInto("/api/v1/dashboard/partner", &resp); err != nil {
				return fmt.Errorf("partner dashboard: %w", err)
			}
			fmt.Printf("    OK: Partner dashboard\n")
			return nil
		})

		reporter.RunTest(flowDashboard, "Partner employees", func() error {
			var resp interface{}
			if _, err := partner.GetInto("/api/v1/dashboard/partner/employees?type=active", &resp); err != nil {
				return fmt.Errorf("partner employees: %w", err)
			}
			fmt.Printf("    OK: Partner employees\n")
			return nil
		})
	}
}
