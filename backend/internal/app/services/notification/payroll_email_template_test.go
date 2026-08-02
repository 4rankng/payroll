package notification

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	serviceports "api-server/internal/domain/ports/services"
	emailinfra "api-server/internal/infra/email"
)

func TestRenderPayrollTemplateUsesResponsiveFinancialLayout(t *testing.T) {
	withBackendWorkingDirectory(t)

	htmlBody, err := renderPayrollTemplate(
		"15/08/2026",
		"2.069.229.050 đ",
		"41.384.581 đ",
		"2.110.613.631 đ",
	)
	if err != nil {
		t.Fatalf("render payroll email template: %v", err)
	}

	checks := []struct {
		name string
		want string
	}{
		{name: "mobile viewport", want: `<meta name="viewport" content="width=device-width,initial-scale=1">`},
		{name: "provider-compatible document width", want: `max-width:640px`},
		{name: "mobile breakpoint", want: `@media only screen and (max-width:480px)`},
		{name: "mobile outer gutter", want: `.email-canvas{padding:12px 8px!important;}`},
		{name: "fixed summary columns", want: `table-layout:fixed`},
		{name: "protected monetary values", want: `white-space:nowrap`},
		{name: "rendered total paid", want: `2.069.229.050 đ`},
		{name: "rendered fee", want: `41.384.581 đ`},
		{name: "rendered total collect", want: `2.110.613.631 đ`},
		{name: "rendered due date", want: `15/08/2026`},
		{name: "beneficiary", want: `CONG TY TNHH MTV GPPM TING TING`},
		{name: "account number", want: `283866888`},
		{name: "bank", want: `TECHCOMBANK`},
	}

	for _, check := range checks {
		if !strings.Contains(htmlBody, check.want) {
			t.Errorf("expected %s marker %q", check.name, check.want)
		}
	}

	if strings.Contains(htmlBody, "box-shadow") {
		t.Fatal("expected the operational email design to remain flat without decorative shadows")
	}
}

func TestPayrollTemplateStaysAlignedAfterProviderBranding(t *testing.T) {
	withBackendWorkingDirectory(t)

	reportDate := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.Local)
	reportBytes := []byte("xlsx-content")
	service := &EmailService{}
	message, err := service.buildPayrollReportMessage(
		&dto.SendPayrollReportEmailRequest{Recipients: []string{"recipient@example.com"}},
		reportDate,
		&serviceports.PayrollReportSummary{
			TotalAmount:  2_069_229_050,
			FeeAmount:    41_384_581,
			TotalWithFee: 2_110_613_631,
		},
		reportBytes,
	)
	if err != nil {
		t.Fatalf("build payroll report message: %v", err)
	}

	provider := emailinfra.NewSandboxProvider(slog.Default())
	if _, err := provider.Send(context.Background(), message); err != nil {
		t.Fatalf("send payroll email through sandbox provider: %v", err)
	}

	captured := provider.LastEmail()
	if captured == nil || captured.Message == nil {
		t.Fatal("expected sandbox provider to capture the delivered message")
	}
	deliveredHTML := captured.Message.HTMLBody
	if strings.Count(deliveredHTML, "tingting.vip/email-banner.jpg") != 1 {
		t.Fatal("expected exactly one canonical banner in delivered payroll email")
	}
	if !strings.Contains(deliveredHTML, `<table role="presentation" width="640"`) {
		t.Fatal("expected delivered document to retain the provider's 640px layout contract")
	}
	if !strings.Contains(deliveredHTML, `<img src="https://tingting.vip/email-banner.jpg?v=20260709" alt="TingTing Soft" width="640"`) {
		t.Fatal("expected delivered banner and document to use the same width")
	}
	if strings.Contains(deliveredHTML, `width="600"`) {
		t.Fatal("expected no stale 600px width after provider normalization")
	}
	if captured.Message.Kind != domain.EmailKindPayrollReport {
		t.Fatalf("delivered message kind = %q, want %q", captured.Message.Kind, domain.EmailKindPayrollReport)
	}
	if len(captured.Message.Attachments) != 1 {
		t.Fatalf("delivered attachment count = %d, want 1", len(captured.Message.Attachments))
	}
	attachment := captured.Message.Attachments[0]
	if attachment.Filename != "sao_ke_tt_2026-08-01.xlsx" || attachment.ContentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("unexpected delivered attachment metadata: %#v", attachment)
	}
	if !bytes.Equal(attachment.Content, reportBytes) {
		t.Fatal("expected delivered payroll attachment content to remain unchanged")
	}
	if captured.Message.Metadata["report_at_date"] != "2026-08-01" || captured.Message.Metadata["due_date"] == "" {
		t.Fatalf("unexpected delivered payroll metadata: %#v", captured.Message.Metadata)
	}
}

func withBackendWorkingDirectory(t *testing.T) {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	backendRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../.."))
	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(backendRoot); err != nil {
		t.Fatalf("change to backend root: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWorkingDirectory); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}
