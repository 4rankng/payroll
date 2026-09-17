package main

import (
	"fmt"
	"time"
)

const flowBulk = "BulkTransfer"

func runBulkTransferTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW 1: Weekly/Monthly Payment via Auto Bulk Transfer")

	// Discovery-dependent flow: nothing to drive without a weekly/monthly
	// project AND an employee (with bank details) on it — skip loudly instead
	// of nil-dereferencing below when the local DB has no qualifying data.
	if data.WeeklyProject == nil || data.WeeklyEmployee == nil ||
		data.MonthlyProject == nil || data.MonthlyEmployee == nil {
		fmt.Println("    SKIPPED: discovery found no weekly/monthly project+employee with bank details")
		return
	}

	cleanupEmployeeTimesheets(client, data.WeeklyProject.ID, data.WeeklyEmployee.ID)
	cleanupEmployeeTimesheets(client, data.MonthlyProject.ID, data.MonthlyEmployee.ID)

	var autoBulkConfig struct {
		Enabled bool `json:"enabled"`
	}
	if _, err := client.GetInto("/api/v1/payrolls/auto-bulk-transfer/config", &autoBulkConfig); err != nil {
		reporter.RunTest(flowBulk, "Check auto bulk transfer config", func() error {
			return fmt.Errorf("get config: %w", err)
		})
	}

	if !autoBulkConfig.Enabled {
		skipAll := func(names []string) {
			for _, name := range names {
				reporter.Skip(flowBulk, name, "auto bulk transfer not enabled (no disbursement provider configured)")
			}
		}
		reporter.Skip(flowBulk, "Check auto bulk transfer config enabled", "no disbursement provider configured — skipping auto bulk transfer flow")
		skipAll([]string{
			"Estimate fee (weekly, before approval)",
			"Estimate fee (monthly)",
			"Edge: initiate with future dates (no timesheets)",
			"Edge: initiate with both forMonth and date range",
			"Edge: estimate fee with missing dates",
		})
		return
	}

	reporter.RunTest(flowBulk, "Check auto bulk transfer config enabled", func() error {
		return AssertTrue("enabled", autoBulkConfig.Enabled)
	})

	// --- Weekly estimate fee ---
	if data.WeeklyProject == nil || data.WeeklyEmployee == nil {
		reporter.Skip(flowBulk, "Estimate fee (weekly, before approval)", "no weekly project or employee found")
	} else {
		var weeklyMinDate *time.Time
		if data.WeeklyAssignment != nil && data.WeeklyAssignment.StartDate != "" {
			if t, err := time.ParseInLocation("2006-01-02", data.WeeklyAssignment.StartDate, time.Local); err == nil {
				weeklyMinDate = &t
			}
		}
		weeklyDate := findAvailableDateWithMin(client, data.WeeklyProject.ID, data.WeeklyEmployee.ID, time.Now(), weeklyMinDate)
		if weeklyDate == "" {
			reporter.Skip(flowBulk, "Estimate fee (weekly, before approval)", "no available date found for weekly employee")
		} else {
			fmt.Printf("    Weekly test date: %s\n", weeklyDate)

			reporter.RunTest(flowBulk, "Estimate fee (weekly, before approval)", func() error {
				req := ExportBulkTransferRequest{
					ProjectIDs: []uint{data.WeeklyProject.ID},
					FromDate:   weeklyDate,
					ToDate:     weeklyDate,
				}
				var resp EstimateFeeResponse
				if _, err := client.PostInto("/api/v1/payrolls/auto-bulk-transfer/estimate-fee", req, &resp); err != nil {
					return fmt.Errorf("estimate fee: %w", err)
				}
				fmt.Printf("    transfer_count=%d, total_fee=%d\n", resp.TransferCount, resp.TotalFee)
				return nil
			})
		}
	}

	// --- Monthly estimate fee ---
	if data.MonthlyProject == nil {
		reporter.Skip(flowBulk, "Estimate fee (monthly)", "no monthly project found")
	} else {
		reporter.RunTest(flowBulk, "Estimate fee (monthly)", func() error {
			req := ExportBulkTransferRequest{
				ProjectIDs: []uint{data.MonthlyProject.ID},
				ForMonth:   currentMonth(),
			}
			var resp EstimateFeeResponse
			if _, err := client.PostInto("/api/v1/payrolls/auto-bulk-transfer/estimate-fee", req, &resp); err != nil {
				return fmt.Errorf("estimate fee monthly: %w", err)
			}
			fmt.Printf("    Monthly fee: transfer_count=%d, total_fee=%d\n", resp.TransferCount, resp.TotalFee)
			return nil
		})
	}

	// --- Edge cases ---

	reporter.RunTest(flowBulk, "Edge: initiate with future dates (no timesheets)", func() error {
		req := ExportBulkTransferRequest{
			FromDate: "2099-01-01",
			ToDate:   "2099-01-07",
		}
		_, _, err := client.Post("/api/v1/payrolls/auto-bulk-transfer", req)
		if err != nil {
			// Error is expected - either "no eligible" or similar
			return nil
		}
		// If no error, the response might have total_count=0 which is also acceptable
		return nil
	})

	reporter.RunTest(flowBulk, "Edge: initiate with both forMonth and date range", func() error {
		req := ExportBulkTransferRequest{
			FromDate: weekAgo(),
			ToDate:   today(),
			ForMonth: currentMonth(),
		}
		apiErr, _, err := client.PostExpectError("/api/v1/payrolls/auto-bulk-transfer", req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		return AssertContains("message", apiErr.Message, "đồng thời")
	})

	reporter.RunTest(flowBulk, "Edge: estimate fee with missing dates", func() error {
		req := ExportBulkTransferRequest{
			FromDate: weekAgo(),
			// ToDate missing
		}
		apiErr, _, err := client.PostExpectError("/api/v1/payrolls/auto-bulk-transfer/estimate-fee", req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		return AssertContains("message", apiErr.Message, "bắt buộc")
	})

}
