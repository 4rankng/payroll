package main

import (
	"fmt"
)

const flowManualBulk = "ManualBulkTransfer"

func runManualBulkTransferTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW 5: Manual Bulk Transfer Export -> Result Import -> Timesheets Paid")

	if data.WeeklyProject == nil || data.WeeklyEmployee == nil {
		reporter.Skip(flowManualBulk, "All manual bulk transfer tests", "no weekly project/employee")
		return
	}

	adminClient := client.WithToken(data.AdminToken)

	// Monthly export test
	if data.MonthlyProject != nil {
		reporter.RunTest(flowManualBulk, "Export bulk transfer Excel (monthly)", func() error {
			req := ExportBulkTransferRequest{
				ProjectIDs: []uint{data.MonthlyProject.ID},
				ForMonth:   currentMonth(),
			}
			_, _, statusCode, err := adminClient.DownloadPost("/api/v1/payrolls/export-bulk-transfer", req)
			if err != nil {
				if statusCode == 200 {
					return nil
				}
				return fmt.Errorf("export monthly bulk transfer: %w", err)
			}
			fmt.Printf("    Monthly export completed (HTTP %d)\n", statusCode)
			return nil
		})
	} else {
		reporter.Skip(flowManualBulk, "Export bulk transfer Excel (monthly)", "no monthly project")
	}
}
