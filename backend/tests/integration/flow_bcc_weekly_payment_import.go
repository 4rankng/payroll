package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const flowWeeklyPayment = "WeeklyPaymentImport"

// runWeeklyPaymentImportTests tests the Thai Binh Duong weekly payment format.
// Builds a synthetic workbook with complete employee, bank, and salary tier data.
func runWeeklyPaymentImportTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW: Weekly Payment Import (Thai Binh Duong format)")

	if len(data.Partners) == 0 {
		reporter.Skip(flowWeeklyPayment, "All weekly payment tests", "no partner users found")
		return
	}
	if data.WeeklyProject == nil {
		reporter.Skip(flowWeeklyPayment, "All weekly payment tests", "no weekly project found")
		return
	}

	partnerClient := client.WithToken(data.Partners[0].Token)
	adminClient := client.WithToken(data.AdminToken)
	projectID := data.WeeklyProject.ID
	projectIDStr := strconv.Itoa(int(projectID))
	endpoint := "/api/v1/timesheets/partner-import"

	// Build a complete synthetic fixture. The historical workbook contains
	// payroll rows without CCCDs and cannot drive a successful import.
	weeklyPaymentFile, err := buildWeeklyPaymentFixture()
	if err != nil {
		reporter.RunTest(flowWeeklyPayment, "Build synthetic weekly payment fixture", func() error { return err })
		return
	}
	defer os.Remove(weeklyPaymentFile)
	// Use 2026-07 for the Thai Binh Duong fixture (matches the file title)
	forMonth := "2026-07"

	// ── 1. Upload Thai Binh Duong.xlsx (WeeklyPayment format) ────────────────────
	var importID uint
	var uploadForbidden bool
	reporter.RunTest(flowWeeklyPayment, "Upload synthetic WeeklyPayment file", func() error {
		apiResp, status, err := partnerClient.UploadFile(endpoint, "file", weeklyPaymentFile,
			map[string]string{"project_id": projectIDStr, "for_month": forMonth})
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		if status == 403 {
			fmt.Printf("    Partner lacks upload access (403) — skipping\n")
			uploadForbidden = true
			return nil
		}

		var result bccImportResponse
		raw, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(raw, &result); err != nil {
			if status != 200 {
				return fmt.Errorf("status %d, body: %s", status, string(raw))
			}
			return fmt.Errorf("unmarshal: %w (raw: %s)", err, string(raw))
		}

		importID = result.ID
		fmt.Printf("    import_id=%d status=%s created=%d skipped=%d errors=%d for_month=%s total_rows=%d\n",
			result.ID, result.Status, result.CreatedCount, result.SkippedCount, result.ErrorCount, result.ForMonth, result.TotalRows)

		if status != 202 {
			return fmt.Errorf("expected 202, got %d", status)
		}
		if result.ID == 0 {
			return fmt.Errorf("import ID must be > 0")
		}
		terminal, err := waitForBCCImport(partnerClient, endpoint, result.ID)
		if err != nil {
			return err
		}
		result = *terminal
		// Import should succeed with at least some timesheets created.
		// Employees are auto-created from BCC sheets.
		if result.Status == "failed" {
			return fmt.Errorf("import failed: status=%s, error_detail present=%v", result.Status, result.ErrorCount > 0)
		}
		if result.CreatedCount == 0 {
			return fmt.Errorf("expected at least 1 timesheet created, got %d", result.CreatedCount)
		}
		return nil
	})
	if uploadForbidden {
		for _, name := range []string{
			"Verify per-tier rate resolution (520HC)",
			"Verify per-tier normal-day rate (520NN)",
			"Verify STK sheet auto-creates employees",
			"Verify day uniqueness (no duplicate days)",
			"Re-upload same file (latest wins)",
			"Import with wrong forMonth (should fail)",
		} {
			reporter.Skip(flowWeeklyPayment, name, "partner lacks BCC upload permission")
		}
		return
	}

	// ── 2. Verify per-tier rate resolution ────────────────────────────────────────
	reporter.RunTest(flowWeeklyPayment, "Verify per-tier rate resolution (520HC)", func() error {
		if importID == 0 {
			return fmt.Errorf("no import ID from previous test")
		}

		// Fetch timesheets created from the import
		var timesheets []TimesheetWithDetailsResponse
		path := fmt.Sprintf("/api/v1/timesheets?project_ids=%d&pageSize=200", projectID)
		if _, err := adminClient.GetInto(path, &timesheets); err != nil {
			return fmt.Errorf("fetch timesheets: %w", err)
		}

		// Look for a weekday entry (should have HourType=HC, DayType="ngày thường")
		found := false
		for _, ts := range timesheets {
			if strings.EqualFold(ts.HourType, "HC") && ts.Date == forMonth+"-29" && strings.HasPrefix(ts.PayType, "520.") && ts.DayType == "ngày thường" && ts.HoursWorked == 8 && ts.Amount > 0 {
				found = true
				fmt.Printf("    Found weekday entry: employee=%s date=%s hour_type=%s day_type=%s hours=%.0f\n",
					ts.EmployeeName, ts.Date, ts.HourType, ts.DayType, ts.HoursWorked)
				break
			}
		}
		if !found {
			return fmt.Errorf("expected to find weekday entry with HC/8h, got %d timesheets", len(timesheets))
		}
		return nil
	})

	// ── 3. Verify weekly-payment NN shift keeps the fixed normal-day type ───────────
	reporter.RunTest(flowWeeklyPayment, "Verify per-tier normal-day rate (520NN)", func() error {
		if importID == 0 {
			return fmt.Errorf("no import ID from previous test")
		}

		// Fetch timesheets
		var timesheets []TimesheetWithDetailsResponse
		path := fmt.Sprintf("/api/v1/timesheets?project_ids=%d&pageSize=200", projectID)
		if _, err := adminClient.GetInto(path, &timesheets); err != nil {
			return fmt.Errorf("fetch timesheets: %w", err)
		}

		// Weekly-payment templates always resolve through "ngày thường"; NN is the
		// row-10 shift code, not a signal to change the day-type branch.
		found := false
		for _, ts := range timesheets {
			if strings.EqualFold(ts.HourType, "NN") && ts.Date == forMonth+"-30" && strings.HasPrefix(ts.PayType, "520.") && ts.DayType == "ngày thường" && ts.HoursWorked == 8 && ts.Amount > 0 {
				found = true
				fmt.Printf("    Found NN entry: employee=%s date=%s hour_type=%s day_type=%s hours=%.0f\n",
					ts.EmployeeName, ts.Date, ts.HourType, ts.DayType, ts.HoursWorked)
				break
			}
		}
		if !found {
			return fmt.Errorf("expected to find normal-day entry with NN/8h, got %d timesheets", len(timesheets))
		}
		return nil
	})

	// ── 4. Verify STK auto-creates employees ─────────────────────────────────────────
	reporter.RunTest(flowWeeklyPayment, "Verify STK sheet auto-creates employees", func() error {
		if importID == 0 {
			return fmt.Errorf("no import ID from previous test")
		}

		// Fetch employees for the project
		var employees []EmployeeResponse
		path := fmt.Sprintf("/api/v1/employees?project_id=%d&pageSize=100", projectID)
		if _, err := adminClient.GetInto(path, &employees); err != nil {
			return fmt.Errorf("fetch employees: %w", err)
		}

		// Look for employees with bank info (from STK sheet)
		foundWithBank := 0
		for _, emp := range employees {
			if emp.CCCD == "099260900501" && emp.BankAccountNumber == "100000900501" && emp.BankAccountName != "" {
				foundWithBank++
			}
		}
		fmt.Printf("    Found %d employees with bank info in project\n", foundWithBank)
		if foundWithBank == 0 {
			return fmt.Errorf("expected at least 1 employee with bank info from STK sheet")
		}
		return nil
	})

	// ── 5. Verify day uniqueness (no duplicate days from sheet 520) ───────────────────
	reporter.RunTest(flowWeeklyPayment, "Verify day uniqueness (no duplicate days)", func() error {
		if importID == 0 {
			return fmt.Errorf("no import ID from previous test")
		}

		// Fetch timesheets for the first employee
		var timesheets []TimesheetWithDetailsResponse
		path := fmt.Sprintf("/api/v1/timesheets?project_ids=%d&pageSize=200", projectID)
		if _, err := adminClient.GetInto(path, &timesheets); err != nil {
			return fmt.Errorf("fetch timesheets: %w", err)
		}

		// Group by employee and date
		empDateSet := make(map[string]map[string]bool)
		for _, ts := range timesheets {
			if empDateSet[ts.EmployeeName] == nil {
				empDateSet[ts.EmployeeName] = make(map[string]bool)
			}
			if empDateSet[ts.EmployeeName][ts.Date] {
				return fmt.Errorf("duplicate timesheet found: employee=%s date=%s", ts.EmployeeName, ts.Date)
			}
			empDateSet[ts.EmployeeName][ts.Date] = true
		}

		// Verify at least one employee has multiple days
		maxDays := 0
		for _, dates := range empDateSet {
			if len(dates) > maxDays {
				maxDays = len(dates)
			}
		}
		fmt.Printf("    Max days per employee: %d (no duplicates found)\n", maxDays)
		if maxDays < 2 {
			return fmt.Errorf("expected at least one employee with 2+ days, got max %d", maxDays)
		}
		return nil
	})

	// ── 6. Re-upload (latest wins) ─────────────────────────────────────────────────
	reporter.RunTest(flowWeeklyPayment, "Re-upload same file (latest wins)", func() error {
		apiResp, _, err := partnerClient.UploadFile(endpoint, "file", weeklyPaymentFile,
			map[string]string{"project_id": projectIDStr, "for_month": forMonth})
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}

		var result bccImportResponse
		raw, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(raw, &result); err != nil {
			return fmt.Errorf("unmarshal: %w", err)
		}

		fmt.Printf("    Re-upload import_id=%d status=%s\n", result.ID, result.Status)

		terminal, err := waitForBCCImport(partnerClient, endpoint, result.ID)
		if err != nil {
			return err
		}
		if terminal.Status != "completed" || terminal.CreatedCount != 2 {
			return fmt.Errorf("re-upload status=%s created=%d; expected both synthetic rows", terminal.Status, terminal.CreatedCount)
		}

		// Verify no duplicate timesheets exist
		var timesheets []TimesheetWithDetailsResponse
		path := fmt.Sprintf("/api/v1/timesheets?project_ids=%d&pageSize=500", projectID)
		if _, err := adminClient.GetInto(path, &timesheets); err != nil {
			return fmt.Errorf("fetch timesheets: %w", err)
		}

		// Check for duplicates (same employee, same date)
		empDateCount := make(map[string]int)
		for _, ts := range timesheets {
			key := fmt.Sprintf("%s|%s", ts.EmployeeName, ts.Date)
			empDateCount[key]++
			if empDateCount[key] > 1 {
				return fmt.Errorf("duplicate timesheet after re-upload: employee=%s date=%s", ts.EmployeeName, ts.Date)
			}
		}

		fmt.Printf("    No duplicates found after re-upload (%d timesheets total)\n", len(timesheets))
		return nil
	})

	// ── 7. Import with wrong forMonth (should fail) ───────────────────────────────────
	reporter.RunTest(flowWeeklyPayment, "Import with wrong forMonth (should fail)", func() error {
		wrongMonth := "2026-02" // February (different from July in fixture)
		apiResp, status, err := partnerClient.UploadFile(endpoint, "file", weeklyPaymentFile,
			map[string]string{"project_id": projectIDStr, "for_month": wrongMonth})
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}

		var result bccImportResponse
		raw, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(raw, &result); err != nil {
			// If we get a parse error, check if status indicates an error response
			if status >= 400 {
				fmt.Printf("    Import rejected with status %d (expected for month mismatch)\n", status)
				return nil
			}
			return fmt.Errorf("unmarshal: %w", err)
		}

		terminal, err := waitForBCCImport(partnerClient, endpoint, result.ID)
		if err != nil {
			return fmt.Errorf("wrong-month import did not reach a terminal result: %w", err)
		}

		result = *terminal
		if result.Status != "failed" {
			return fmt.Errorf("expected import to fail with wrong forMonth, got status=%s", result.Status)
		}
		fmt.Printf("    Import failed as expected: status=%s\n", result.Status)
		return nil
	})
}
