package notification

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"api-server/internal/app/dto"
	appconfig "api-server/internal/app/services/config"
	"api-server/internal/config"
	serviceports "api-server/internal/domain/ports/services"
	emailinfra "api-server/internal/infra/email"
)

type stubBankProvider struct{}

func (stubBankProvider) GetTransferBankInfo(context.Context) appconfig.TransferBankInfo {
	return appconfig.TransferBankInfo{Holder: "TING TING TEST", Number: "99001122", Name: "Ngân hàng Kiểm thử (KT)"}
}

func TestPayrollEmailShowsConfiguredBankEndToEnd(t *testing.T) {
	withBackendWorkingDirectory(t)
	service := NewEmailService(config.NotificationConfig{}, emailinfra.NewSandboxProvider(slog.Default()), nil, nil, nil, nil, nil, stubBankProvider{}, slog.Default())
	msg, err := service.buildPayrollReportMessage(
		context.Background(),
		&dto.SendPayrollReportEmailRequest{Recipients: []string{"r@example.com"}},
		time.Date(2026, time.August, 19, 0, 0, 0, 0, time.Local),
		&serviceports.PayrollReportSummary{TotalAmount: 1_000_000, FeeAmount: 20_000, TotalWithFee: 1_020_000},
		[]byte("xlsx"),
	)
	if err != nil {
		t.Fatalf("build message: %v", err)
	}
	for _, want := range []string{"TING TING TEST", "99001122", "Ngân hàng Kiểm thử (KT)"} {
		if !strings.Contains(msg.HTMLBody, want) {
			t.Errorf("HTML body missing %q", want)
		}
		if !strings.Contains(msg.TextBody, want) {
			t.Errorf("text body missing %q", want)
		}
	}
	if strings.Contains(msg.HTMLBody, "TECHCOMBANK") || strings.Contains(msg.HTMLBody, "283866888") {
		t.Fatal("stale Techcombank data leaked into email")
	}
}
