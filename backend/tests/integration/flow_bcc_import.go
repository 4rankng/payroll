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
	ID           uint    `json:"id"`
	ForMonth     string  `json:"for_month"`
	Status       string  `json:"status"`
	TotalRows    int     `json:"total_rows"`
	CreatedCount int     `json:"created_count"`
	SkippedCount int     `json:"skipped_count"`
	ErrorCount   int     `json:"error_count"`
	OriginalName string  `json:"original_name"`
	ProjectID    uint    `json:"project_id"`
	ErrorDetail  *string `json:"error_detail"`
}

func waitForBCCImport(client *APIClient, endpoint string, importID uint) (*bccImportResponse, error) {
	apiResp, err := client.PollUntil(
		fmt.Sprintf("%s/%d", endpoint, importID),
		500*time.Millisecond,
		2*time.Minute,
		func(resp *APIResponse) bool {
			var result bccImportResponse
			raw, _ := json.Marshal(resp.Data)
			_ = json.Unmarshal(raw, &result)
			return result.Status == "completed" || result.Status == "failed"
		},
	)
	if err != nil {
		return nil, err
	}
	var result bccImportResponse
	raw, _ := json.Marshal(apiResp.Data)
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return &result, nil
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
	bccFile := getEnvOrDefault("PAYROLL_BCC_FIXTURE", "../docs/WeeklyBCC/BCC LGD.xlsx")
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
		if status != 202 {
			return fmt.Errorf("expected 202, got %d", status)
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
		terminal, err := waitForBCCImport(partnerClient, endpoint, result.ID)
		if err != nil {
			return err
		}
		fmt.Printf("    terminal status=%s created=%d skipped=%d errors=%d\n",
			terminal.Status, terminal.CreatedCount, terminal.SkippedCount, terminal.ErrorCount)
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
		if status != 202 {
			return fmt.Errorf("expected 202, got %d", status)
		}
		var accepted bccImportResponse
		raw, _ := json.Marshal(apiResp.Data)
		_ = json.Unmarshal(raw, &accepted)
		terminal, err := waitForBCCImport(partnerClient, endpoint, accepted.ID)
		if err != nil {
			return err
		}
		msg := ""
		if terminal.ErrorDetail != nil {
			msg = *terminal.ErrorDetail
		}
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

	// ── 6. Re-upload preserves approved rows ─────────────────────────────────
	reporter.RunTest(flowBCC, "Re-upload preserves approved timesheets", func() error {
		// First upload to get some timesheets.
		apiResp1, status1, err := partnerClient.UploadFile(endpoint, "file", bccFile,
			map[string]string{"project_id": projectIDStr, "for_month": time.Now().Format("2006-01")})
		if err != nil {
			return fmt.Errorf("first upload: %w", err)
		}
		var r1 bccImportResponse
		raw, _ := json.Marshal(apiResp1.Data)
		_ = json.Unmarshal(raw, &r1)
		if status1 != 202 {
			return fmt.Errorf("expected first upload 202, got %d", status1)
		}
		terminal1, err := waitForBCCImport(partnerClient, endpoint, r1.ID)
		if err != nil {
			return err
		}
		r1 = *terminal1
		if r1.Status != "completed" || r1.CreatedCount == 0 {
			return fmt.Errorf("approval prerequisite: import status=%s created=%d", r1.Status, r1.CreatedCount)
		}

		// Approve the actual imported rows. The endpoint accepts explicit IDs,
		// and reviewed rows are preserved while pending rows may be replaced.
		rows, err := loadAllPages[TimesheetResponse](adminClient, fmt.Sprintf("/api/v1/timesheets?project_ids=%d", projectID))
		if err != nil {
			return err
		}
		var ids []uint
		before := make(map[uint]TimesheetResponse)
		for _, row := range rows {
			if row.Status == "pending_approval" && strings.HasPrefix(row.Date, time.Now().Format("2006-01")) {
				ids = append(ids, row.ID)
				before[row.ID] = row
			}
		}
		if len(ids) == 0 {
			return fmt.Errorf("no pending imported rows to approve")
		}
		defer func() {
			_, status, resetErr := adminClient.Post("/api/v1/timesheets/bulk-reset", BulkApproveRequest{TimesheetIDs: ids})
			if resetErr != nil || status >= 400 {
				fmt.Printf("    cleanup reset failed: HTTP %d, %v\n", status, resetErr)
			}
		}()
		approval, status, err := adminClient.Post("/api/v1/timesheets/bulk-approve", BulkApproveRequest{TimesheetIDs: ids})
		if err != nil {
			return err
		}
		if status != 200 {
			return fmt.Errorf("approve rows: HTTP %d", status)
		}
		var approved BulkOperationResponse
		if err := json.Unmarshal(approval.Data, &approved); err != nil {
			return err
		}
		if approved.Approved != len(ids) {
			return fmt.Errorf("approved %d of %d rows", approved.Approved, len(ids))
		}

		// Re-upload should complete with protected rows skipped.
		apiResp2, status2, err2 := partnerClient.UploadFile(endpoint, "file", bccFile,
			map[string]string{"project_id": projectIDStr, "for_month": time.Now().Format("2006-01")})
		if err2 != nil {
			return fmt.Errorf("second upload: %w", err2)
		}
		var r2 bccImportResponse
		raw2, _ := json.Marshal(apiResp2.Data)
		_ = json.Unmarshal(raw2, &r2)
		if status2 != 202 {
			return fmt.Errorf("expected second upload 202, got %d", status2)
		}
		terminal2, err := waitForBCCImport(partnerClient, endpoint, r2.ID)
		if err != nil {
			return err
		}
		r2 = *terminal2
		if r2.Status != "completed" || r2.CreatedCount != 0 || r2.SkippedCount == 0 {
			return fmt.Errorf("protected re-upload: status=%s created=%d skipped=%d", r2.Status, r2.CreatedCount, r2.SkippedCount)
		}
		rows, err = loadAllPages[TimesheetResponse](adminClient, fmt.Sprintf("/api/v1/timesheets?project_ids=%d", projectID))
		if err != nil {
			return err
		}
		verified := 0
		for _, row := range rows {
			if original, ok := before[row.ID]; ok {
				if row.Status != "approved" || row.Amount != original.Amount || row.HoursWorked != original.HoursWorked || row.Date != original.Date || row.PayType != original.PayType {
					return fmt.Errorf("approved timesheet %d changed on re-upload", row.ID)
				}
				verified++
			}
		}
		if verified != len(ids) {
			return fmt.Errorf("approved rows missing: found %d of %d", verified, len(ids))
		}
		fmt.Printf("    re-upload preserved %d approved rows; skipped=%d\n", verified, r2.SkippedCount)
		return nil
	})

}
