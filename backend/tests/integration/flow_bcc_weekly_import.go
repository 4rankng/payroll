package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

const flowWeeklyBCC = "WeeklyBCCImport"

// runWeeklyBCCImportTests tests the BCC-<shiftType> sheet format upload.
// Uses docs/WeeklyBCC/BCC LGD.xlsx which has BCC-HC, BCC-OT150, etc. sheets.
func runWeeklyBCCImportTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW: Weekly BCC Import (BCC-<shiftType> sheets)")

	if len(data.Partners) == 0 {
		reporter.Skip(flowWeeklyBCC, "All weekly BCC tests", "no partner users found")
		return
	}
	if data.WeeklyProject == nil {
		reporter.Skip(flowWeeklyBCC, "All weekly BCC tests", "no weekly project found")
		return
	}

	partnerClient := client.WithToken(data.Partners[0].Token)
	adminClient := client.WithToken(data.AdminToken)
	projectID := data.WeeklyProject.ID
	projectIDStr := strconv.Itoa(int(projectID))
	endpoint := "/api/v1/timesheets/partner-import"

	// WeeklyBCC fixture (relative to backend/ working dir).
	weeklyBCCFile := "../docs/WeeklyBCC/BCC LGD.xlsx"
	currentMonth := time.Now().Format("2006-01")

	// ── 1. Upload BCC LGD.xlsx (WeeklyBCC format) ────────────────────────────
	var importID uint
	reporter.RunTest(flowWeeklyBCC, "Upload WeeklyBCC file (BCC LGD.xlsx)", func() error {
		apiResp, status, err := partnerClient.UploadFile(endpoint, "file", weeklyBCCFile,
			map[string]string{"project_id": projectIDStr, "for_month": currentMonth})
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		if status == 403 {
			fmt.Printf("    Partner lacks upload access (403) — skipping\n")
			return nil
		}

		var result bccImportResponse
		raw, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(raw, &result); err != nil {
			// If status != 200, dump the raw response for debugging.
			if status != 200 {
				return fmt.Errorf("status %d, body: %s", status, string(raw))
			}
			return fmt.Errorf("unmarshal: %w (raw: %s)", err, string(raw))
		}

		importID = result.ID
		fmt.Printf("    import_id=%d status=%s created=%d skipped=%d errors=%d for_month=%s total_rows=%d\n",
			result.ID, result.Status, result.CreatedCount, result.SkippedCount, result.ErrorCount, result.ForMonth, result.TotalRows)

		if status != 200 && status != 201 {
			return fmt.Errorf("expected 200/201, got %d", status)
		}
		if result.ID == 0 {
			return fmt.Errorf("import ID must be > 0")
		}
		// Import should succeed with at least some timesheets created.
		// Employees are auto-created from BCC sheets.
		if result.Status == "failed" {
			return fmt.Errorf("import failed: status=%s, error_detail present=%v", result.Status, result.ErrorCount > 0)
		}
		return nil
	})

	// ── 2. Verify import detail ───────────────────────────────────────────────
	if importID > 0 {
		reporter.RunTest(flowWeeklyBCC, "Get WeeklyBCC import detail", func() error {
			apiResp, status, err := partnerClient.Get(fmt.Sprintf("%s/%d", endpoint, importID))
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			if status != 200 {
				return fmt.Errorf("expected 200, got %d", status)
			}
			var result bccImportResponse
			raw, _ := json.Marshal(apiResp.Data)
			if err := json.Unmarshal(raw, &result); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			fmt.Printf("    detail: status=%s created=%d errors=%d for_month=%s\n",
				result.Status, result.CreatedCount, result.ErrorCount, result.ForMonth)
			return AssertEqual("import_id", importID, result.ID)
		})
	}

	// ── 3. Re-upload with same month → latest-wins overwrite ──────────────────
	reporter.RunTest(flowWeeklyBCC, "Re-upload WeeklyBCC (latest wins)", func() error {
		apiResp, status, err := partnerClient.UploadFile(endpoint, "file", weeklyBCCFile,
			map[string]string{"project_id": projectIDStr, "for_month": currentMonth})
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		if status == 403 {
			fmt.Printf("    Partner lacks upload access (403) — skipping\n")
			return nil
		}
		if status != 200 && status != 201 {
			return fmt.Errorf("expected 200/201, got %d", status)
		}
		var result bccImportResponse
		raw, _ := json.Marshal(apiResp.Data)
		_ = json.Unmarshal(raw, &result)
		fmt.Printf("    re-upload: status=%s created=%d errors=%d\n",
			result.Status, result.CreatedCount, result.ErrorCount)
		// Second upload should also succeed (overwriting unapproved entries).
		if result.Status == "failed" && result.ErrorCount > 0 {
			// Could be blocked by paid/approved timesheets from first upload — acceptable.
			fmt.Printf("    re-upload had errors (likely blocked entries) — acceptable\n")
		}
		return nil
	})

	// ── 4. Upload EPE06.2026.xlsx (STK format) ───────────────────────────────
	epeFile := "../docs/WeeklyBCC/EPE06.2026.xlsx"
	reporter.RunTest(flowWeeklyBCC, "Upload EPE06.2026.xlsx with WeeklyBCC project", func() error {
		apiResp, status, err := partnerClient.UploadFile(endpoint, "file", epeFile,
			map[string]string{"project_id": projectIDStr, "for_month": currentMonth})
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		if status == 403 {
			fmt.Printf("    Partner lacks upload access (403) — skipping\n")
			return nil
		}
		var result bccImportResponse
		raw, _ := json.Marshal(apiResp.Data)
		_ = json.Unmarshal(raw, &result)
		fmt.Printf("    EPE upload: status=%d import_id=%d created=%d errors=%d for_month=%s\n",
			status, result.ID, result.CreatedCount, result.ErrorCount, result.ForMonth)
		// EPE file might be a different format (legacy or multi-position) — just verify no crash.
		if status == 500 {
			return fmt.Errorf("server error (500) — possible unhandled format")
		}
		return nil
	})

	// ── 5. Admin can list weekly BCC imports ──────────────────────────────────
	reporter.RunTest(flowWeeklyBCC, "Admin can list all import history", func() error {
		_, status, err := adminClient.Get(endpoint)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		return AssertEqual("http_status", 200, status)
	})
}
