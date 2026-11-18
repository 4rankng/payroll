package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const flowPourinImport = "PourinWeeklyPaymentImport"

// pourinConfiguredRates mirrors the rates the dev/prod payrate config must
// carry for position "800000" before the POURIN upload runs.
const pourinHCPerHour = 800000
const pourinTCPerHour = 1200000

// runPourinTemplateImportTests exercises the POURIN full-month template
// variant of the weekly-payment family: identity columns shifted right by
// "Ngày đi làm"/"Ngày nghỉ việc", the salary header spelled
// "Vị trí lương/8h" (slash form), the tier stored under the #,##0 display
// format, and the project's payrate position named exactly by the raw tier
// value (800000 → position "800000"). Requires the weekly project's payrate
// config to carry positions "800000"/"900000" — seed them to enable.
func runPourinTemplateImportTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW: POURIN weekly payment template (full-month variant)")

	if len(data.Partners) == 0 {
		reporter.Skip(flowPourinImport, "All POURIN template tests", "no partner users found")
		return
	}
	if data.WeeklyProject == nil {
		reporter.Skip(flowPourinImport, "All POURIN template tests", "no weekly project found")
		return
	}

	partnerClient := client.WithToken(data.Partners[0].Token)
	adminClient := client.WithToken(data.AdminToken)
	weeklyProject, ok := partnerWeeklyProject(partnerClient, data.WeeklyProject)
	if !ok {
		reporter.Skip(flowPourinImport, "All POURIN template tests", "partner has no visible weekly project")
		return
	}
	projectID := weeklyProject.ID
	projectIDStr := strconv.Itoa(int(projectID))
	endpoint := "/api/v1/timesheets/partner-import"
	forMonth := "2026-07"

	// ── 0. Some config covering the import month must carry the positions ──────
	configReady := true
	reporter.RunTest(flowPourinImport, "Verify payrate config carries POURIN positions", func() error {
		var payrates []PayrateResponse
		if _, err := adminClient.GetInto(fmt.Sprintf("/api/v1/payrates?project_id=%d", projectID), &payrates); err != nil {
			configReady = false
			return fmt.Errorf("list payrates: %w", err)
		}
		if len(payrates) == 0 {
			configReady = false
			return fmt.Errorf("project %d has no payrate config", projectID)
		}
		for _, pr := range payrates {
			if pr.FromDate > forMonth+"-28" {
				continue // starts after every entry date — irrelevant to this upload
			}
			var rates map[string]map[string]map[string]float64
			if err := json.Unmarshal(pr.Rates, &rates); err != nil {
				continue
			}
			pos, ok := rates["800000"]["ngày thường"]
			if ok && pos["HC"] == pourinHCPerHour && pos["TC"] == pourinTCPerHour {
				fmt.Printf("    positions seeded in config from %s\n", pr.FromDate)
				return nil
			}
		}
		configReady = false
		fmt.Printf("    no config covering %s carries position \"800000\"/ngày thường HC=%d TC=%d — skipping flow\n", forMonth, pourinHCPerHour, pourinTCPerHour)
		return nil
	})
	if !configReady {
		reporter.Skip(flowPourinImport, "Upload POURIN template", "payrate config lacks positions 800000/900000 (seed them to enable)")
		return
	}

	pourinFile, err := buildPourinWeeklyPaymentFixture()
	if err != nil {
		reporter.RunTest(flowPourinImport, "Build synthetic POURIN fixture", func() error { return err })
		return
	}
	defer func() { _ = os.Remove(pourinFile) }()

	// ── 1. Upload the POURIN-style workbook ─────────────────────────────────────
	reporter.RunTest(flowPourinImport, "Upload POURIN template file", func() error {
		apiResp, status, err := partnerClient.UploadFile(endpoint, "file", pourinFile,
			map[string]string{"project_id": projectIDStr, "for_month": forMonth})
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		if status == 403 {
			return fmt.Errorf("partner lacks upload access (403)")
		}
		var result bccImportResponse
		raw, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(raw, &result); err != nil {
			return fmt.Errorf("unmarshal (status %d, body %s): %w", status, string(raw), err)
		}
		terminal, err := waitForBCCImport(partnerClient, endpoint, result.ID)
		if err != nil {
			return err
		}
		fmt.Printf("    import_id=%d status=%s created=%d errors=%d\n",
			terminal.ID, terminal.Status, terminal.CreatedCount, terminal.ErrorCount)
		if terminal.Status != "completed" {
			return fmt.Errorf("import status=%s, want completed", terminal.Status)
		}
		if terminal.CreatedCount < 2 {
			return fmt.Errorf("created=%d, want the HC and TC rows", terminal.CreatedCount)
		}
		return nil
	})

	// ── 2. Verify the HC row is priced at position 800000 ───────────────────────
	reporter.RunTest(flowPourinImport, "Verify HC row priced at position 800000", func() error {
		var timesheets []TimesheetWithDetailsResponse
		path := fmt.Sprintf("/api/v1/timesheets?project_ids=%d&fromDate=%s-27&toDate=%s-28&pageSize=500", projectID, forMonth, forMonth)
		if _, err := adminClient.GetInto(path, &timesheets); err != nil {
			return fmt.Errorf("fetch timesheets: %w", err)
		}
		for _, ts := range timesheets {
			if ts.Date == forMonth+"-27" && strings.EqualFold(ts.HourType, "HC") &&
				strings.HasPrefix(ts.PayType, "800000.") && ts.DayType == "ngày thường" &&
				ts.HoursWorked == 8 && ts.Amount == float64(8*pourinHCPerHour) {
				fmt.Printf("    Found HC row: employee=%s paytype=%s amount=%.0f\n", ts.EmployeeName, ts.PayType, ts.Amount)
				return nil
			}
		}
		return fmt.Errorf("no HC row on %s-27 priced under position 800000 (%d timesheets scanned)", forMonth, len(timesheets))
	})

	// ── 3. Verify the TC row: previously-dropped shift code now priced ──────────
	reporter.RunTest(flowPourinImport, "Verify TC row priced at position 800000", func() error {
		var timesheets []TimesheetWithDetailsResponse
		path := fmt.Sprintf("/api/v1/timesheets?project_ids=%d&fromDate=%s-27&toDate=%s-28&pageSize=500", projectID, forMonth, forMonth)
		if _, err := adminClient.GetInto(path, &timesheets); err != nil {
			return fmt.Errorf("fetch timesheets: %w", err)
		}
		for _, ts := range timesheets {
			if ts.Date == forMonth+"-28" && strings.EqualFold(ts.HourType, "TC") &&
				strings.HasPrefix(ts.PayType, "800000.") && ts.DayType == "ngày thường" &&
				ts.HoursWorked == 4 && ts.Amount == float64(4*pourinTCPerHour) {
				fmt.Printf("    Found TC row: employee=%s paytype=%s amount=%.0f\n", ts.EmployeeName, ts.PayType, ts.Amount)
				return nil
			}
		}
		return fmt.Errorf("no TC row on %s-28 priced under position 800000", forMonth)
	})

	// ── 4. Verify STK auto-created the employee with bank info ──────────────────
	reporter.RunTest(flowPourinImport, "Verify STK auto-created POURIN employee", func() error {
		var employees []EmployeeResponse
		path := fmt.Sprintf("/api/v1/employees?project_id=%d&pageSize=200", projectID)
		if _, err := adminClient.GetInto(path, &employees); err != nil {
			return fmt.Errorf("fetch employees: %w", err)
		}
		for _, emp := range employees {
			if emp.CCCD == "099260900602" && emp.BankAccountNumber == "100000900602" {
				return nil
			}
		}
		return fmt.Errorf("employee 099260900602 with STK bank info not found")
	})
}
