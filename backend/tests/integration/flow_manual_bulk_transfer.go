package main

import (
	"fmt"
	"mime"
	"net/http"
	"strings"
)

const flowManualBulk = "ManualBulkTransfer"

const (
	bulkTransferXLSXContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	bulkTransferZIPContentType  = "application/zip"
)

func assertBulkTransferDownload(body []byte, headers http.Header, statusCode int) error {
	if statusCode != http.StatusOK {
		return fmt.Errorf("expected HTTP %d, got %d", http.StatusOK, statusCode)
	}
	if len(body) == 0 {
		return fmt.Errorf("expected non-empty export body")
	}

	disposition := headers.Get("Content-Disposition")
	_, dispositionParams, err := mime.ParseMediaType(disposition)
	if err != nil {
		return fmt.Errorf("parse Content-Disposition %q: %w", disposition, err)
	}
	filename := dispositionParams["filename"]
	if filename == "" {
		return fmt.Errorf("Content-Disposition missing filename: %q", disposition)
	}

	contentType, _, err := mime.ParseMediaType(headers.Get("Content-Type"))
	if err != nil {
		return fmt.Errorf("parse Content-Type %q: %w", headers.Get("Content-Type"), err)
	}

	extension := strings.ToLower(filename)
	switch contentType {
	case bulkTransferXLSXContentType:
		if !strings.HasSuffix(extension, ".xlsx") {
			return fmt.Errorf("XLSX response filename must end in .xlsx, got %q", filename)
		}
	case bulkTransferZIPContentType:
		if !strings.HasSuffix(extension, ".zip") {
			return fmt.Errorf("ZIP response filename must end in .zip, got %q", filename)
		}
	default:
		return fmt.Errorf("unexpected bulk transfer Content-Type %q for filename %q", contentType, filename)
	}

	return nil
}

func runManualBulkTransferTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW 5: Manual Bulk Transfer Export -> Result Import -> Timesheets Paid")

	if data.WeeklyProject == nil || data.WeeklyEmployee == nil {
		reporter.Skip(flowManualBulk, "All manual bulk transfer tests", "no weekly project/employee")
		return
	}

	adminClient := client.WithToken(data.AdminToken)

	// Monthly export test
	if data.MonthlyProject != nil {
		reporter.RunTest(flowManualBulk, "Export bulk transfer file (monthly)", func() error {
			req := ExportBulkTransferRequest{
				ProjectIDs: []uint{data.MonthlyProject.ID},
				ForMonth:   currentMonth(),
			}
			body, headers, statusCode, err := adminClient.DownloadPost("/api/v1/payrolls/export-bulk-transfer", req)
			if err != nil {
				return fmt.Errorf("export monthly bulk transfer: %w", err)
			}
			if err := assertBulkTransferDownload(body, headers, statusCode); err != nil {
				return fmt.Errorf("validate monthly bulk transfer export: %w", err)
			}
			fmt.Printf("    Monthly export completed (HTTP %d, %s)\n", statusCode, headers.Get("Content-Type"))
			return nil
		})
	} else {
		reporter.Skip(flowManualBulk, "Export bulk transfer file (monthly)", "no monthly project")
	}
}
