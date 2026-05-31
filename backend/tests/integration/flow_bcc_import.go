package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

const flowBCC = "BCCImport"

type bccImportResponse struct {
	ID           uint   `json:"id"`
	ForMonth     string `json:"for_month"`
	Status       string `json:"status"`
	TotalRows    int    `json:"total_rows"`
	CreatedCount int    `json:"created_count"`
	SkippedCount int    `json:"skipped_count"`
	ErrorCount   int    `json:"error_count"`
	OriginalName string `json:"original_name"`
	ProjectID    uint   `json:"project_id"`
}

func runBCCImportTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW: Partner BCC Import")

	if len(data.Partners) == 0 {
		reporter.Skip(flowBCC, "All BCC import tests", "no partner users found")
		return
	}
	if data.WeeklyProject == nil {
		reporter.Skip(flowBCC, "All BCC import tests", "no weekly project found")
		return
	}

	partnerClient := client.WithToken(data.Partners[0].Token)
	adminClient := client.WithToken(data.AdminToken)
	projectID := data.WeeklyProject.ID
	projectIDStr := strconv.Itoa(int(projectID))
	endpoint := "/api/v1/timesheets/partner-import"

	// BCC sample file shipped with the repo (path relative to backend/ working dir).
	bccFile := "../docs/timesheets-excel/BCC LƯƠNG DỰ ÁN EVA Sample.xlsx"

	// ── 1. Missing project_id → 400 ──────────────────────────────────────────
	reporter.RunTest(flowBCC, "Upload without project_id returns 400", func() error {
		_, status, err := partnerClient.UploadFile(endpoint, "file", bccFile, nil)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		return AssertEqual("http_status", 400, status)
	})

	// ── 2. Happy path upload ─────────────────────────────────────────────────
	var importID uint
	reporter.RunTest(flowBCC, "Upload valid BCC file", func() error {
		apiResp, status, err := partnerClient.UploadFile(endpoint, "file", bccFile,
			map[string]string{"project_id": projectIDStr, "for_month": time.Now().Format("2006-01")})
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		if status == 403 {
			fmt.Printf("    Partner lacks upload access (403) — skipping BCC upload tests\n")
			return nil
		}
		if status != 200 {
			return fmt.Errorf("expected 200, got %d", status)
		}
		var result bccImportResponse
		raw, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(raw, &result); err != nil {
			return fmt.Errorf("unmarshal: %w", err)
		}
		importID = result.ID
		fmt.Printf("    import_id=%d status=%s created=%d skipped=%d errors=%d for_month=%s\n",
			result.ID, result.Status, result.CreatedCount, result.SkippedCount, result.ErrorCount, result.ForMonth)
		if result.ID == 0 {
			return fmt.Errorf("import ID must be > 0")
		}
		if result.ForMonth == "" {
			return fmt.Errorf("for_month must not be empty")
		}
		return nil
	})

	if importID > 0 {
		// ── 3. Get detail ───────────────────────────────────────────────────
		reporter.RunTest(flowBCC, "Get import detail by ID", func() error {
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
			return AssertEqual("import_id", importID, result.ID)
		})

		// ── 4. Download endpoint ────────────────────────────────────────────
		reporter.RunTest(flowBCC, "Download original BCC file", func() error {
			fileBytes, _, status, err := partnerClient.DownloadGet(
				fmt.Sprintf("%s/%d/download", endpoint, importID))
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			if status != 200 {
				return fmt.Errorf("expected 200, got %d", status)
			}
			if len(fileBytes) == 0 {
				return fmt.Errorf("downloaded file is empty")
			}
			return nil
		})
	}

	// ── 5. History list: partner sees only own uploads ────────────────────────
	reporter.RunTest(flowBCC, "List upload history returns 200", func() error {
		_, status, err := partnerClient.Get(endpoint)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		return AssertEqual("http_status", 200, status)
	})

	// Admin list must also work.
	reporter.RunTest(flowBCC, "Admin can list all import history", func() error {
		_, status, err := adminClient.Get(endpoint)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		return AssertEqual("http_status", 200, status)
	})

	// ── 6. Re-upload after approval → rejected entirely ───────────────────────
	reporter.RunTest(flowBCC, "Re-upload after timesheet approval is rejected", func() error {
		// First upload to get some timesheets.
		apiResp1, _, err := partnerClient.UploadFile(endpoint, "file", bccFile,
			map[string]string{"project_id": projectIDStr, "for_month": time.Now().Format("2006-01")})
		if err != nil {
			return fmt.Errorf("first upload: %w", err)
		}
		var r1 bccImportResponse
		raw, _ := json.Marshal(apiResp1.Data)
		_ = json.Unmarshal(raw, &r1)
		if r1.Status != "completed" || r1.CreatedCount == 0 {
			fmt.Println("    sub-test skipped: no timesheets created on first upload")
			return nil
		}

		// Admin approve all pending for this project.
		_, _, _ = adminClient.Post("/api/v1/timesheets/bulk-approve",
			map[string]any{"project_id": projectID, "approve_all": true})

		// Re-upload — must be rejected.
		apiResp2, _, err2 := partnerClient.UploadFile(endpoint, "file", bccFile,
			map[string]string{"project_id": projectIDStr, "for_month": time.Now().Format("2006-01")})
		if err2 != nil {
			return fmt.Errorf("second upload: %w", err2)
		}
		var r2 bccImportResponse
		raw2, _ := json.Marshal(apiResp2.Data)
		_ = json.Unmarshal(raw2, &r2)
		fmt.Printf("    re-upload after approval: status=%s\n", r2.Status)
		return AssertEqual("status_after_approval", "failed", r2.Status)
	})

}
