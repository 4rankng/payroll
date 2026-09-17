package payroll

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"context"
	"testing"
	"time"

	appconfig "api-server/internal/app/services/config"
	domainServices "api-server/internal/domain/services"
	pkgConstants "api-server/internal/pkg/constants"

	"github.com/xuri/excelize/v2"
)

type fakeSettingsConfigProvider struct {
	bulkPercent float64
	feePercent  float64
}

func (f *fakeSettingsConfigProvider) GetWeeklyPaymentPercentage(ctx context.Context) float64 {
	return f.bulkPercent
}

func (f *fakeSettingsConfigProvider) GetWeeklyPaymentFeePercentage(ctx context.Context) float64 {
	return f.feePercent
}

func (f *fakeSettingsConfigProvider) GetTransferBankInfo(ctx context.Context) appconfig.TransferBankInfo {
	return appconfig.DefaultTransferBankInfo()
}

func TestBuildPayrollReportDataUsesPaidAmounts(t *testing.T) {
	provider := &fakeSettingsConfigProvider{
		bulkPercent: 0.7,
		feePercent:  0.02,
	}
	exporter := NewPayrollReportExporter(provider)

	paymentDate := time.Date(2025, time.October, 29, 10, 0, 0, 0, time.UTC)
	entries := []*domainServices.PayrollReportEntry{
		{
			PaymentDate:  &paymentDate,
			EmployeeName: "Dang Thi Huyen Trang",
			EmployeeCCCD: "031189014251",
			ProjectName:  "Magnetec",
			PaidAmount:   2_947_000,
			TimesheetIDs: []uint{101},
		},
		{
			PaymentDate:  &paymentDate,
			EmployeeName: "Dang Thi Huyen Trang",
			EmployeeCCCD: "031189014251",
			ProjectName:  "Magnetec",
			PaidAmount:   2_950_925,
			TimesheetIDs: []uint{102},
		},
	}

	data := exporter.BuildPayrollReportData(context.Background(), entries)
	if len(data.Rows) != 1 {
		t.Fatalf("expected single aggregated row, got %d", len(data.Rows))
	}

	amount, ok := data.Rows[0][5].(int64)
	if !ok {
		t.Fatalf("expected int64 amount, got %T", data.Rows[0][5])
	}

	expectedPaid := int64(2_947_000 + 2_950_925)
	if amount != expectedPaid {
		t.Fatalf("expected paid amount %d, got %d", expectedPaid, amount)
	}

	if len(data.TimesheetIDs) != 2 {
		t.Fatalf("expected 2 timesheet IDs, got %d", len(data.TimesheetIDs))
	}
	if data.TimesheetIDs[0] != 101 || data.TimesheetIDs[1] != 102 {
		t.Fatalf("unexpected timesheet IDs: %v", data.TimesheetIDs)
	}
}

func TestBuildPayrollReportDataWithSingleEntry(t *testing.T) {
	provider := &fakeSettingsConfigProvider{
		bulkPercent: 0.7,
		feePercent:  0.02,
	}
	exporter := NewPayrollReportExporter(provider)

	paymentDate := time.Date(2025, time.October, 29, 10, 0, 0, 0, time.UTC)
	entry := &domainServices.PayrollReportEntry{
		PaymentDate:  &paymentDate,
		EmployeeName: "Test Employee",
		EmployeeCCCD: "000000000",
		ProjectName:  "Test Project",
		PaidAmount:   5_893_125,
		TimesheetIDs: []uint{500},
	}

	data := exporter.BuildPayrollReportData(context.Background(), []*domainServices.PayrollReportEntry{entry})
	if len(data.Rows) != 1 {
		t.Fatalf("expected single aggregated row, got %d", len(data.Rows))
	}

	amount, ok := data.Rows[0][5].(int64)
	if !ok {
		t.Fatalf("expected int64 amount, got %T", data.Rows[0][5])
	}

	expectedPaid := int64(5_893_125)
	if amount != expectedPaid {
		t.Fatalf("expected paid amount %d, got %d", expectedPaid, amount)
	}

	if len(data.TimesheetIDs) != 1 {
		t.Fatalf("expected single timesheet ID, got %d", len(data.TimesheetIDs))
	}
	if data.TimesheetIDs[0] != 500 {
		t.Fatalf("unexpected timesheet ID: %d", data.TimesheetIDs[0])
	}
}

func TestGenerateExcelWritesConfiguredBankInfo(t *testing.T) {
	if _, err := os.Stat(pkgConstants.PayrollReportTemplatePath); err != nil {
		// Resolve template relative to the backend root for tests launched
		// from outside the module directory.
		_, thisFile, _, _ := runtime.Caller(0)
		backendRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))))
		origWd, wdErr := os.Getwd()
		if err := os.Chdir(backendRoot); err != nil {
			t.Skipf("cannot chdir to backend root: %v", err)
		}
		t.Cleanup(func() {
			if wdErr == nil {
				_ = os.Chdir(origWd)
			}
		})
	}
	provider := &fakeSettingsConfigProvider{bulkPercent: 0.7, feePercent: 0.02}
	exporter := NewPayrollReportExporter(provider)

	fromDate := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2026, time.August, 11, 0, 0, 0, 0, time.UTC)
	data := &PayrollReportExcelData{Rows: [][]interface{}{{1, "05/08/2026", "Nguyen Van A", "012345678901", "Dự án A", int64(1_500_000)}}}

	excelBytes, _, err := exporter.GenerateExcel(context.Background(), fromDate, toDate, data)
	if err != nil {
		t.Fatalf("generate excel: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(excelBytes))
	if err != nil {
		t.Fatalf("open generated excel: %v", err)
	}
	defer f.Close()

	sheet := f.GetSheetList()[0]
	for cell, want := range map[string]string{
		"E9":  "CONG TY TNHH MTV GPPM TING TING",
		"E10": "271866699",
		"E11": "Ngân hàng Quân đội (MB)",
	} {
		got, err := f.GetCellValue(sheet, cell)
		if err != nil {
			t.Fatalf("read %s: %v", cell, err)
		}
		if got != want {
			t.Errorf("%s = %q, want %q", cell, got, want)
		}
	}
}
