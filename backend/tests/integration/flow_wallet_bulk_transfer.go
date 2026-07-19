package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/xuri/excelize/v2"
)

const flowWalletBulk = "WalletBulkTransfer"

// runWalletBulkTransferTests exercises the Stage 2 wallet-page pipeline via
// the live HTTP API. Skips gracefully when the bulk-transfer endpoints aren't
// registered (no disbursement provider configured) — same pattern as the
// existing runBulkTransferTests.
func runWalletBulkTransferTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW: Wallet Bulk Transfer Pipeline (OnePay)")
	var knownBatchID uint64

	// Probe the upload endpoint — if it 404s, the whole flow is skipped.
	_, probeStatus, _ := client.Get("/api/v1/wallet/bulk-transfer/batches")
	if probeStatus == http.StatusNotFound {
		reporter.Skip(flowWalletBulk, "Endpoint availability",
			"/wallet/bulk-transfer/* not mounted (no disbursement provider) — skipping entire flow")
		return
	}

	// Test 1: List batches returns 200 + paginated shape.
	reporter.RunTest(flowWalletBulk, "List batches returns paginated response", func() error {
		var resp struct {
			Batches []struct {
				ID uint64 `json:"id"`
			} `json:"batches"`
			Total    int64 `json:"total"`
			Page     int   `json:"page"`
			PageSize int   `json:"page_size"`
		}
		_, err := client.GetInto("/api/v1/wallet/bulk-transfer/batches?page=1&page_size=10", &resp)
		if err != nil {
			return fmt.Errorf("list batches: %w", err)
		}
		if resp.Page != 1 {
			return fmt.Errorf("page: got %d, want 1", resp.Page)
		}
		if resp.PageSize != 10 {
			return fmt.Errorf("page_size: got %d, want 10", resp.PageSize)
		}
		if len(resp.Batches) > 0 {
			knownBatchID = resp.Batches[0].ID
		}
		return nil
	})

	// Test 2: Upload rejects a non-xlsx file (PDF renamed to .xlsx).
	reporter.RunTest(flowWalletBulk, "Upload rejects non-xlsx content", func() error {
		pdfBytes := []byte("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n") // PDF magic, not ZIP
		resp, err := uploadBulkTransferFile(client, "evil.pdf", "application/pdf", pdfBytes)
		if err != nil {
			return fmt.Errorf("upload evil pdf: %w", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			return fmt.Errorf("status: got %d, want 400", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(body, &errBody)
		if errBody.Error != "invalid_file_type" {
			return fmt.Errorf("error code: got %q, want invalid_file_type", errBody.Error)
		}
		return nil
	})

	// Test 3: Upload rejects a file exceeding 10MB.
	reporter.RunTest(flowWalletBulk, "Upload rejects file >10MB", func() error {
		oversized := bytes.Repeat([]byte{0x50, 0x4B, 0x03, 0x04}, (10<<20)/4+1) // >10MB with ZIP magic
		resp, err := uploadBulkTransferFile(client, "big.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", oversized)
		if err != nil {
			return fmt.Errorf("upload oversized: %w", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			return fmt.Errorf("status: got %d, want 400", resp.StatusCode)
		}
		return nil
	})

	// Test 4: Upload rejects an xlsx missing the SWIFT column.
	// (builds a minimal eMB_BulkPayment sheet without the Mã SWIFT column)
	reporter.RunTest(flowWalletBulk, "Upload rejects xlsx missing SWIFT column", func() error {
		xlsxBytes := buildXlsxWithoutSwiftColumn()
		resp, err := uploadBulkTransferFile(client, "no_swift.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", xlsxBytes)
		if err != nil {
			return fmt.Errorf("upload no_swift: %w", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			return fmt.Errorf("status: got %d, want 400", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(body, &errBody)
		if errBody.Error != "missing_swift_column" {
			return fmt.Errorf("error code: got %q, want missing_swift_column", errBody.Error)
		}
		return nil
	})

	// Note: a full happy-path test (upload → process → KQ) requires a live
	// OnePay provider + seeded employees with bank info + approved timesheets
	// exported via /admin/timesheet → "Chuyển OnePay". That sequence is
	// covered by the wallet_bulk_transfer_smoke_test.md runbook (manual).
	reporter.Skip(flowWalletBulk, "Full happy-path upload → process → KQ",
		"requires OnePay provider + seeded export file — see docs/runbooks/wallet-bulk-transfer-smoke-test.md")

	// Test 5: GetBatch returns 404 for a non-existent id.
	reporter.RunTest(flowWalletBulk, "GetBatch returns 404 for unknown id", func() error {
		_, status, err := client.Get("/api/v1/wallet/bulk-transfer/batches/99999999")
		if err != nil {
			return fmt.Errorf("get unknown batch: %w", err)
		}
		if status != http.StatusNotFound {
			return fmt.Errorf("status: got %d, want 404", status)
		}
		return nil
	})

	// Test 6: KQ download on unknown id returns 404.
	reporter.RunTest(flowWalletBulk, "KQ download returns 404 for unknown id", func() error {
		_, status, err := client.Get("/api/v1/wallet/bulk-transfer/batches/99999999/kq")
		if err != nil {
			return fmt.Errorf("download unknown kq: %w", err)
		}
		if status != http.StatusNotFound {
			return fmt.Errorf("status: got %d, want 404", status)
		}
		return nil
	})

	// Test 7: KQ scope is explicit and validated before generation.
	reporter.RunTest(flowWalletBulk, "KQ download rejects invalid scope", func() error {
		if knownBatchID == 0 {
			return fmt.Errorf("no existing wallet bulk batch available")
		}
		_, status, err := client.Get(fmt.Sprintf("/api/v1/wallet/bulk-transfer/batches/%d/kq?scope=invalid", knownBatchID))
		if err != nil {
			return fmt.Errorf("download invalid scope: %w", err)
		}
		if status != http.StatusBadRequest {
			return fmt.Errorf("status: got %d, want 400", status)
		}
		return nil
	})

	// Test 8: Casbin — partner role is denied.
	reporter.RunTest(flowWalletBulk, "Partner role denied by Casbin", func() error {
		if data == nil || len(data.Partners) == 0 {
			return fmt.Errorf("no partner user available — set up a partner to validate Casbin denial")
		}
		partnerClient := client.WithToken(data.Partners[0].Token)
		_, status, err := partnerClient.Get("/api/v1/wallet/bulk-transfer/batches")
		if err != nil {
			return fmt.Errorf("partner request: %w", err)
		}
		if status != http.StatusForbidden {
			return fmt.Errorf("partner status: got %d, want 403", status)
		}
		return nil
	})

	// Give the periodic sweepers a moment to register on first boot.
	time.Sleep(100 * time.Millisecond)
}

// uploadBulkTransferFile POSTs a multipart "file" to the upload endpoint.
func uploadBulkTransferFile(client *APIClient, filename, contentType string, content []byte) (*http.Response, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(content); err != nil {
		return nil, fmt.Errorf("write file content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequest("POST", client.BaseURL+"/api/v1/wallet/bulk-transfer/upload", &buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if client.Token != "" {
		req.Header.Set("Authorization", "Bearer "+client.Token)
	}
	return http.DefaultClient.Do(req)
}

// buildXlsxWithoutSwiftColumn constructs a valid .xlsx with the
// eMB_BulkPayment sheet but headers WITHOUT the Mã SWIFT column. Used to
// verify the parser's ErrMissingSwiftColumn path via the HTTP API.
//
// H1 fix: this builds the workbook in-process via excelize (the prior
// version returned an empty byte slice and the test would never reach
// ErrMissingSwiftColumn — it'd hit ErrInvalidSheet or open-xlsx error).
func buildXlsxWithoutSwiftColumn() []byte {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	const sheet = "eMB_BulkPayment"
	if idx, _ := f.GetSheetIndex("Sheet1"); idx == 0 {
		_ = f.SetSheetName("Sheet1", sheet)
	} else {
		_, _ = f.NewSheet(sheet)
	}

	// Headers WITHOUT "Mã SWIFT" — only 6 columns.
	headers := []string{
		"STT", "Số tài khoản", "Tên người thụ hưởng",
		"Ngân hàng thụ hưởng/Chi nhánh", "Số tiền", "Nội dung chuyển khoản",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		_ = f.SetCellValue(sheet, cell, h)
	}
	// One data row at row 3.
	data := []interface{}{1, "99990001", "Test A", "Quân đội (MB)", 1500000, "VFIC3ba3ec31LUONG"}
	for i, v := range data {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		_ = f.SetCellValue(sheet, cell, v)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil
	}
	return buf.Bytes()
}
