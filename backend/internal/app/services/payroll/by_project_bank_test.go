package payroll

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	appconfig "api-server/internal/app/services/config"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	pkgConstants "api-server/internal/pkg/constants"

	"github.com/xuri/excelize/v2"
)

type stubBankProvider struct{}

func (stubBankProvider) GetTransferBankInfo(context.Context) appconfig.TransferBankInfo {
	return appconfig.TransferBankInfo{Holder: "TING TING TEST", Number: "99001122", Name: "Ngân hàng Kiểm thử (KT)"}
}

func TestByProjectExcelShowsConfiguredBank(t *testing.T) {
	if _, err := os.Stat(pkgConstants.PayrollReportByProjectTemplatePath); err != nil {
		_, thisFile, _, _ := runtime.Caller(0)
		backendRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))))
		origWd, wdErr := os.Getwd()
		if chdirErr := os.Chdir(backendRoot); chdirErr != nil {
			t.Skipf("template not found and cannot chdir to backend root: %v", chdirErr)
		}
		t.Cleanup(func() {
			if wdErr == nil {
				_ = os.Chdir(origWd)
			}
		})
	}
	if _, err := os.Stat(pkgConstants.PayrollReportByProjectTemplatePath); err != nil {
		t.Skipf("template not found after chdir: %v", err)
	}
	reportData := []*domainServices.ProjectReportData{
		{Project: &domain.Project{ID: 1, Name: "Dự án A"}, EmployeeCount: 1, TotalAmount: 500_000},
	}
	exporter := NewPayrollReportByProjectExporter(stubBankProvider{}, stubWeeklyFeeProvider{pct: 0.02})
	out, _, err := exporter.GenerateExcel(context.Background(), reportData, time.Date(2026, time.August, 19, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()
	for cell, want := range map[string]string{
		"E8":  "TING TING TEST",
		"E9":  "99001122",
		"E10": "Ngân hàng Kiểm thử (KT)",
	} {
		got, err := f.GetCellValue("Summary", cell)
		if err != nil {
			t.Fatalf("read %s: %v", cell, err)
		}
		if got != want {
			t.Errorf("Summary!%s = %q, want %q", cell, got, want)
		}
	}
}

// stubWeeklyFeeProvider feeds buildSummary a fixed weekly fee percentage.
type stubWeeklyFeeProvider struct {
	pct float64
}

func (s stubWeeklyFeeProvider) GetWeeklyPaymentFeePercentage(context.Context) float64 {
	return s.pct
}
