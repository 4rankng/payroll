package main

import (
	"fmt"
)

const flowAudit = "Audit"

func runAuditTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Audit Logs")

	admin := client.WithToken(data.AdminToken)

	reporter.RunTest(flowAudit, "List audit logs", func() error {
		var resp interface{}
		if _, err := admin.GetInto("/api/v1/audit/logs?pageSize=10", &resp); err != nil {
			return fmt.Errorf("list audit logs: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowAudit, "List audit logs with date filter", func() error {
		var resp interface{}
		path := fmt.Sprintf("/api/v1/audit/logs?pageSize=10&fromDate=%s&toDate=%s", weekAgo(), today())
		if _, err := admin.GetInto(path, &resp); err != nil {
			return fmt.Errorf("filtered logs: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowAudit, "Edge: get non-existent audit log", func() error {
		_, statusCode, _ := admin.Get("/api/v1/audit/logs/999999")
		if statusCode < 400 {
			return fmt.Errorf("expected error for non-existent audit log")
		}
		return nil
	})

}
