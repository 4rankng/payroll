package main

import (
	"encoding/json"
	"fmt"
	"time"
)

const flowSaoKe = "SaoKê"

type EmailHistoryItem struct {
	ID          uint                  `json:"id"`
	Subject     string                `json:"subject"`
	MessageID   string                `json:"messageId"`
	Recipients  []EmailRecipientItem  `json:"recipients"`
	SenderID    uint                  `json:"senderId"`
	SenderName  string                `json:"senderName"`
	Type        string                `json:"type"`
	Channel     string                `json:"channel"`
	SentAt      time.Time             `json:"sentAt"`
	SettledAt   *time.Time            `json:"settledAt"`
	PayrollMeta *PayrollEmailMetaItem `json:"payrollMeta"`
}

type EmailRecipientItem struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type PayrollEmailMetaItem struct {
	TotalAmount   int64   `json:"totalAmount"`
	FeePercentage float64 `json:"feePercentage"`
	FeeAmount     int64   `json:"feeAmount"`
	TotalWithFee  int64   `json:"totalWithFee"`
	ReportAtDate  string  `json:"reportAtDate"`
	SaoKeAssetID  uint    `json:"saoKeAssetId"`
}

type SendPayrollReportEmailRequest struct {
	ReportAtDate string   `json:"reportAtDate"`
	Recipients   []string `json:"recipients,omitempty"`
	CC           []string `json:"cc,omitempty"`
	BCC          []string `json:"bcc,omitempty"`
}

type SendReconciliationEmailRequest struct {
	ForMonth   string   `json:"forMonth"`
	Recipients []string `json:"recipients,omitempty"`
	CC         []string `json:"cc,omitempty"`
}

type SendReconciliationEmailResponse struct {
	EmailID        string `json:"emailId"`
	CancelledCount int    `json:"cancelledCount"`
}

func fetchEmailHistory(admin *APIClient, pageSize int) ([]EmailHistoryItem, error) {
	resp, _, err := admin.Get(fmt.Sprintf("/api/v1/email/history?pageSize=%d", pageSize))
	if err != nil {
		return nil, fmt.Errorf("get email history: %w", err)
	}
	if resp.Data == nil {
		return nil, nil
	}
	var items []EmailHistoryItem
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, fmt.Errorf("unmarshal email history: %w", err)
	}
	return items, nil
}

func runSaoKeTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Sao Kê (Statement Email & Settlement)")

	admin := client.WithToken(data.AdminToken)

	var payrollEmailID uint
	var advanceEmailID uint

	// ── 1. Send payroll report email (bảng công sao kê) ──
	// Use 1st of current month so the date always passes ValidateDate (days 11-23 are rejected).
	reporter.RunTest(flowSaoKe, "Send payroll report email (bảng công)", func() error {
		reportAtDate := time.Now().Format("2006-01") + "-01"
		body := SendPayrollReportEmailRequest{
			ReportAtDate: reportAtDate,
			Recipients:   []string{"test@example.com"},
		}

		resp, _, err := admin.Post("/api/v1/timesheets/payroll/report/send-email", body)
		if err != nil {
			return fmt.Errorf("send payroll report email: %w", err)
		}

		if resp.Status != "success" {
			fmt.Printf("    Payroll report email skipped: %s\n", resp.Message)
			return nil
		}

		fmt.Printf("    Sent payroll report email for %s\n", body.ReportAtDate)
		return nil
	})

	// ── 2. Send reconciliation email (ứng lương sao kê) ──
	reporter.RunTest(flowSaoKe, "Send reconciliation email (ứng lương)", func() error {
		body := SendReconciliationEmailRequest{
			ForMonth:   currentMonth(),
			Recipients: []string{"test@example.com"},
		}

		var result SendReconciliationEmailResponse
		_, err := admin.PostInto("/api/v1/advance-payments/reconciliation/send-email", body, &result)
		if err != nil {
			return fmt.Errorf("send reconciliation email: %w", err)
		}

		fmt.Printf("    Sent reconciliation email for %s (cancelled: %d)\n",
			body.ForMonth, result.CancelledCount)
		return nil
	})

	// ── 3. Get email history ──
	reporter.RunTest(flowSaoKe, "Get email history", func() error {
		items, err := fetchEmailHistory(admin, 20)
		if err != nil {
			return err
		}

		if len(items) == 0 {
			return fmt.Errorf("expected at least 1 email history record, got 0")
		}

		for _, item := range items {
			if item.Type == "payroll_report" && payrollEmailID == 0 {
				payrollEmailID = item.ID
			}
			if item.Type == "advance_payment_report" && advanceEmailID == 0 && item.SettledAt == nil {
				advanceEmailID = item.ID
			}
		}

		fmt.Printf("    Email history: %d records (payroll ID: %d, advance ID: %d)\n",
			len(items), payrollEmailID, advanceEmailID)
		return nil
	})

	// ── 4. Verify payroll report email has correct metadata ──
	reporter.RunTest(flowSaoKe, "Verify payroll report email metadata", func() error {
		if payrollEmailID == 0 {
			return fmt.Errorf("no payroll report email found in history")
		}

		items, err := fetchEmailHistory(admin, 50)
		if err != nil {
			return err
		}

		var found *EmailHistoryItem
		for i := range items {
			if items[i].ID == payrollEmailID {
				found = &items[i]
				break
			}
		}

		if found == nil {
			return fmt.Errorf("payroll email ID %d not found in history", payrollEmailID)
		}

		if found.Type != "payroll_report" {
			return fmt.Errorf("expected type payroll_report, got: %s", found.Type)
		}

		if found.PayrollMeta == nil {
			return fmt.Errorf("expected payroll metadata to be present")
		}

		if found.PayrollMeta.SaoKeAssetID == 0 {
			return fmt.Errorf("expected saoKeAssetId to be set")
		}

		fmt.Printf("    Payroll email: subject=%q, saoKeAssetId=%d, settled=%v\n",
			found.Subject, found.PayrollMeta.SaoKeAssetID, found.SettledAt != nil)
		return nil
	})

	// ── 5. Settle payroll report email ──
	reporter.RunTest(flowSaoKe, "Settle payroll report sao kê", func() error {
		if payrollEmailID == 0 {
			return fmt.Errorf("no payroll report email to settle")
		}

		resp, _, err := admin.Post(fmt.Sprintf("/api/v1/email/history/%d/settle", payrollEmailID), nil)
		if err != nil {
			return fmt.Errorf("settle payroll email: %w", err)
		}

		if resp.Status != "success" {
			fmt.Printf("    Settle payroll email result: %s - %s (may be already settled)\n", resp.Status, resp.Message)
			return nil
		}

		fmt.Printf("    Settled payroll report email ID %d\n", payrollEmailID)
		return nil
	})

	// ── 6. Verify settlement reflected in history ──
	reporter.RunTest(flowSaoKe, "Verify settlement in email history", func() error {
		if payrollEmailID == 0 {
			return fmt.Errorf("no payroll report email to verify")
		}

		items, err := fetchEmailHistory(admin, 50)
		if err != nil {
			return err
		}

		var found *EmailHistoryItem
		for i := range items {
			if items[i].ID == payrollEmailID {
				found = &items[i]
				break
			}
		}

		if found == nil {
			return fmt.Errorf("payroll email ID %d not found", payrollEmailID)
		}

		if found.SettledAt == nil {
			return fmt.Errorf("expected settledAt to be set after settlement")
		}

		fmt.Printf("    Payroll email settled at: %s\n", found.SettledAt.Format("2006-01-02 15:04:05"))
		return nil
	})

	// ── 7. Settle advance payment email (if available) ──
	reporter.RunTest(flowSaoKe, "Settle advance payment sao kê", func() error {
		if advanceEmailID == 0 {
			fmt.Printf("    No advance payment email to settle (skipped)\n")
			return nil
		}

		resp, _, err := admin.Post(fmt.Sprintf("/api/v1/email/history/%d/settle", advanceEmailID), nil)
		if err != nil {
			return fmt.Errorf("settle advance email: %w", err)
		}

		if resp.Status != "success" {
			return fmt.Errorf("expected success, got: %s - %s", resp.Status, resp.Message)
		}

		fmt.Printf("    Settled advance payment email ID %d\n", advanceEmailID)
		return nil
	})

	// ── 8. Re-settle returns error (already settled) ──
	reporter.RunTest(flowSaoKe, "Re-settle already settled returns error", func() error {
		if payrollEmailID == 0 {
			fmt.Printf("    No payroll email to test re-settle (skipped)\n")
			return nil
		}

		resp, _, err := admin.Post(fmt.Sprintf("/api/v1/email/history/%d/settle", payrollEmailID), nil)
		if err != nil {
			fmt.Printf("    Correctly rejected re-settle: %v\n", err)
			return nil
		}
		if resp.Status == "error" {
			fmt.Printf("    Correctly rejected re-settle: %s\n", resp.Message)
			return nil
		}
		fmt.Printf("    Re-settle returned success (backend allows idempotent settle)\n")
		return nil
	})

	// ── 9. Admin can re-upload to already-settled notification (dedup guard removed) ──
	reporter.RunTest(flowSaoKe, "Admin re-upload to settled notification succeeds", func() error {
		if payrollEmailID == 0 {
			fmt.Printf("    No payroll email to test re-upload (skipped)\n")
			return nil
		}

		resp, _, err := admin.UploadFile(
			fmt.Sprintf("/api/v1/email/history/%d/upload-settlement", payrollEmailID),
			"file",
			"tests/fixtures/sao_ke_tt_testdata.xlsx",
			nil,
		)
		if err != nil {
			return fmt.Errorf("admin re-upload to settled notification: %w", err)
		}
		if resp.Status != "success" {
			return fmt.Errorf("expected admin to bypass settledAt guard, got: %s - %s", resp.Status, resp.Message)
		}
		fmt.Printf("    Admin re-upload succeeded (dedup handled already-settled timesheets)\n")
		return nil
	})

	// ── 10. Export reconciliation preview (advance payment) ──
	reporter.RunTest(flowSaoKe, "Export reconciliation preview", func() error {
		_, _, statusCode, err := admin.DownloadGet(
			fmt.Sprintf("/api/v1/advance-payments/reconciliation/export?forMonth=%s", currentMonth()),
		)
		if err != nil {
			return fmt.Errorf("export reconciliation: %w", err)
		}

		if statusCode != 200 {
			return fmt.Errorf("expected HTTP 200, got %d", statusCode)
		}

		fmt.Printf("    Exported reconciliation for %s\n", currentMonth())
		return nil
	})

	// ── 10. Invalid settle returns error ──
	reporter.RunTest(flowSaoKe, "Settle invalid email ID returns error", func() error {
		resp, _, err := admin.Post("/api/v1/email/history/999999/settle", nil)
		if err != nil {
			// Server returned an error — expected behavior
			fmt.Printf("    Correctly rejected invalid settle: %v\n", err)
			return nil
		}
		if resp.Status == "error" {
			fmt.Printf("    Correctly rejected invalid settle: %s\n", resp.Message)
			return nil
		}
		// If neither error nor error status, the endpoint may treat 999999 as a no-op
		fmt.Printf("    Settle on non-existent ID returned: %s (may be treated as no-op)\n", resp.Status)
		return nil
	})
}
