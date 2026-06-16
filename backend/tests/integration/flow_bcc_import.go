package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	if _, err := os.Stat(bccFile); os.IsNotExist(err) {
		reporter.Skip(flowBCC, "All BCC file upload tests", "sample BCC Excel file not found: "+bccFile)
		return
	}

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

	// ── 2b. Boundary regression: importing a month whose day-1 equals the payrate's
	// effective date must NOT be rejected with "no active payrate". Guards a timezone
	// bug where monthStart was built in clock.Now().Location() (Asia/Ho_Chi_Minh) while
	// the MySQL driver uses loc=Local (time.Local = UTC in the prod scratch image),
	// shifting the day-1 boundary by 7h and failing `from_date <= monthStart`. The fix
	// builds monthStart in time.Local (bcc_import_service.go) and the prod image pins
	// TZ=Asia/Ho_Chi_Minh (Dockerfile). NOTE: the pre-fix failure only reproduces when
	// the host runs UTC; on a +07 dev host this asserts the happy path.
	reporter.RunTest(flowBCC, "Import not rejected on payrate effective-date boundary (TZ)", func() error {
		apiResp, status, err := partnerClient.UploadFile(endpoint, "file", bccFile,
			map[string]string{"project_id": projectIDStr, "for_month": time.Now().Format("2006-01")})
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		if status == 403 {
			fmt.Printf("    Partner lacks upload access (403) — skipping boundary test\n")
			return nil
		}
		if status != 200 {
			return fmt.Errorf("expected 200, got %d", status)
		}
		// A payrate-boundary failure returns HTTP 200 with Status="failed" and a message
		// carrying "không tìm thấy bảng lương ... no active payrate". Other import errors
		// (e.g. employee mismatch) must NOT trip this assertion.
		msg := apiResp.Message
		if strings.Contains(msg, "no active payrate") || strings.Contains(msg, "bảng lương") {
			return fmt.Errorf("import rejected on month/payrate boundary (TZ regression): %s", msg)
		}
		fmt.Printf("    boundary import ok: %s\n", msg)
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
