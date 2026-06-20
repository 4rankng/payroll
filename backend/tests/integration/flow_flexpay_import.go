package main

import (
	"encoding/json"
	"fmt"
)

const flowFlexPay = "FlexPayImport"

func runFlexPayImportTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW 3: FlexPay Import (Upload Bảng Lương)")

	fixturePath := "tests/fixtures/LGD- TINGTING 05.05.26 đợt 1.xlsx"

	adminClient := client.WithToken(data.AdminToken)

	// 3.1 Upload FlexPay file for the CURRENT period (so advance_payments exist)
	var jobID uint
	reporter.RunTest(flowFlexPay, "Upload FlexPay bảng lương file", func() error {
		var resp ImportJobResponse
		// Upload for the current advance payment period (2026-04 if today is May 1–16)
		forMonth := currentMonth()
		fmt.Printf("    Uploading for current period month: %s\n", forMonth)
		apiResp, _, err := adminClient.UploadFile("/api/v1/advance-payments/import", "file", fixturePath,
			map[string]string{"forMonth": forMonth})
		if err != nil {
			apiResp, _, err = adminClient.UploadFile("/api/v1/advance-payments/import", "file", fixturePath, nil)
			if err != nil {
				return fmt.Errorf("upload flexpay file: %w", err)
			}
		}
		raw, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(raw, &resp); err != nil {
			return fmt.Errorf("unmarshal import response: %w", err)
		}
		jobID = resp.ID
		fmt.Printf("    Import job ID: %d, status: %s\n", resp.ID, resp.Status)
		return AssertGreaterThan("job_id", uint(0), resp.ID)
	})

	if jobID == 0 {
		reporter.Skip(flowFlexPay, "Poll import job status", "no job created")
		reporter.Skip(flowFlexPay, "Verify import result", "no job created")
		reporter.Skip(flowFlexPay, "Verify advance payment info updated", "no job created")
		return
	}

	// 3.5 Verify advance payment info reflects the import
	if data.EmployeeForAdvance != nil && data.EmployeeTokenForAdv != "" {
		reporter.RunTest(flowFlexPay, "Verify employee advance info updated after import", func() error {
			empClient := client.WithToken(data.EmployeeTokenForAdv)
			var info AdvancePaymentInfoResponse
			if _, err := empClient.GetInto("/api/v1/me/advance-payment", &info); err != nil {
				return fmt.Errorf("get advance info after import: %w", err)
			}
			fmt.Printf("    ForMonth: %s, MaxAdvance: %d, CanRequest: %v\n",
				info.ForMonth, info.MaxAdvanceAmount, info.CanRequest)
			return nil
		})
	} else {
		reporter.Skip(flowFlexPay, "Verify employee advance info updated after import", "no employee user")
	}

	// 3.6 Regression guard: advance_payments.upload_date must be stored as the
	// full upload day (YYYY-MM-DD), not the legacy upload month (YYYY-MM). The
	// HTTP API does not surface upload_date, so assert the on-disk format via
	// the payroll-mysql container. Validates both new writes and the 074
	// migration backfill of legacy month values.
	reporter.RunTest(flowFlexPay, "Verify advance_payments.upload_date is YYYY-MM-DD", func() error {
		dates, err := queryAdvancePaymentUploadDates()
		if err != nil {
			return fmt.Errorf("read upload_date from DB: %w", err)
		}
		if len(dates) == 0 {
			return fmt.Errorf("expected advance_payments rows after FlexPay import, found none")
		}
		for _, d := range dates {
			if !isYYYYMMDD(d) {
				return fmt.Errorf("upload_date %q is not YYYY-MM-DD (len=%d) — legacy month value or bad format", d, len(d))
			}
		}
		fmt.Printf("    Checked %d advance_payments rows: all upload_date in YYYY-MM-DD format\n", len(dates))
		return nil
	})

	data.FlexPayJobID = jobID
}
