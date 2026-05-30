package payroll

import (
	"context"
	"testing"
	"time"

	domainServices "api-server/internal/domain/services"
)

type fakeSettingsConfigProvider struct {
	bulkPercent float64
	feePercent  float64
}

func (f *fakeSettingsConfigProvider) GetWeeklyPaymentPercentage(ctx context.Context) float64 {
	return f.bulkPercent
}

func (f *fakeSettingsConfigProvider) GetAdvanceCashFeePercentage(ctx context.Context) float64 {
	return f.feePercent
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
