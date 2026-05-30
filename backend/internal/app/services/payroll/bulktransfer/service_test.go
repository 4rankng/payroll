package bulktransfer

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"
	pkgConstants "api-server/internal/pkg/constants"

	"github.com/stretchr/testify/assert"
)

type fakeSettingsConfig struct {
	bulkPct float64
	feePct  float64
	partner string
}

func (f *fakeSettingsConfig) GetWeeklyPaymentPercentage(ctx context.Context) float64 {
	return f.bulkPct
}

func (f *fakeSettingsConfig) GetMonthlyPaymentPercentage(ctx context.Context) float64 {
	return f.bulkPct
}

func (f *fakeSettingsConfig) GetAdvanceCashFeePercentage(ctx context.Context) float64 {
	return f.feePct
}

func (f *fakeSettingsConfig) GetPartnerCompany(ctx context.Context) string {
	return f.partner
}

func (f *fakeSettingsConfig) GetPaymentPercentageForSchedule(ctx context.Context, schedule string) float64 {
	return f.bulkPct
}

func TestService_buildPaymentUpdatesPaid(t *testing.T) {
	settings := &fakeSettingsConfig{bulkPct: 0.7}
	service := &Service{settingsConfig: settings}

	timesheets := []*domain.Timesheet{
		{ID: 1, Amount: 1000},
		{ID: 2, Amount: 1500},
	}
	now := time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC)
	reference := "REF-123"

	updates, count := service.buildPaymentUpdates(context.Background(), timesheets, domain.PaymentStatusPaid, reference, now)

	assert.Len(t, updates, len(timesheets))
	assert.Equal(t, len(timesheets), count)

	for i, update := range updates {
		assert.NotNil(t, update.PaymentReference)
		assert.Equal(t, reference, *update.PaymentReference)
		assert.Equal(t, timesheets[i].ID, update.TimesheetID)
		assert.Equal(t, domain.PaymentStatusPaid, update.PaymentStatus)
		if assert.NotNil(t, update.PaymentDate) {
			assert.True(t, update.PaymentDate.Equal(now))
		}
		if assert.NotNil(t, update.PaidAmount) {
			expectedAmount := int64(float64(timesheets[i].Amount) * settings.bulkPct)
			assert.Equal(t, expectedAmount, *update.PaidAmount)
		}
	}
}

func TestService_buildPaymentUpdatesPending(t *testing.T) {
	settings := &fakeSettingsConfig{bulkPct: 0.7}
	service := &Service{settingsConfig: settings}

	timesheets := []*domain.Timesheet{
		{ID: 1, Amount: 1000},
	}
	now := time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC)
	reference := "REF-999"

	updates, count := service.buildPaymentUpdates(context.Background(), timesheets, domain.PaymentStatusPending, reference, now)

	assert.Len(t, updates, 1)
	assert.Equal(t, 1, count)
	update := updates[0]

	assert.NotNil(t, update.PaymentReference)
	assert.Equal(t, reference, *update.PaymentReference)
	assert.Nil(t, update.PaymentDate)
	assert.Nil(t, update.PaidAmount)
	assert.Equal(t, domain.PaymentStatusPending, update.PaymentStatus)
}

func TestBuildLedgerPlan(t *testing.T) {
	settings := &fakeSettingsConfig{
		feePct:  0.05,
		partner: "Partner Co",
	}
	ctx := context.Background()

	totalTransfer := 1000.49
	filename := "MBank-sample.xlsx"

	plan := BuildLedgerPlan(ctx, settings, totalTransfer, settings.partner, filename)

	assert.Equal(t, int64(1051), plan.ReceivableAmount)
	assert.Equal(t, int64(1000), plan.CashOutAmount)
	assert.Equal(t, plan.CashOutAmount, plan.RevenueOffset)
}

func TestLedgerPlanEntries(t *testing.T) {
	plan := LedgerPlan{
		Description:      "Test description",
		ReceivableAmount: 1200,
		CashOutAmount:    1000,
		RevenueOffset:    1000,
	}
	now := time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC)
	processedBy := uint(42)
	partnerCompany := "Partner Co"

	entries := plan.Entries(now, processedBy, partnerCompany)

	assert.Len(t, entries, 2)

	cashOut := entries[0]
	assert.Equal(t, now, cashOut.Date)
	assert.Equal(t, domain.AccountCash, cashOut.Account)
	assert.Equal(t, "Nhân viên", cashOut.Party)
	assert.Equal(t, int64(0), cashOut.Debit)
	assert.Equal(t, plan.CashOutAmount, cashOut.Credit)
	assert.Equal(t, processedBy, cashOut.CreatedBy)

	revenueOffset := entries[1]
	assert.Equal(t, now, revenueOffset.Date)
	assert.Equal(t, domain.AccountRevenue, revenueOffset.Account)
	assert.Equal(t, partnerCompany, revenueOffset.Party)
	assert.Equal(t, plan.RevenueOffset, revenueOffset.Debit)
	assert.Equal(t, int64(0), revenueOffset.Credit)
	assert.Equal(t, processedBy, revenueOffset.CreatedBy)
}

func TestService_detectParserMBank(t *testing.T) {
	service := &Service{
		resultParsers: []BankResultParser{NewMBankParser()},
	}

	rows := [][]string{
		{"header"},
		{"header2"},
		{"colA", "account", "name", "extra", "amount", "desc", "status", "tracking"},
	}

	parser, err := service.detectParser(pkgConstants.MBankPrefix+"file.xlsx", rows)
	assert.NoError(t, err)

	_, ok := parser.(*MBankParser)
	assert.True(t, ok)
	assert.Equal(t, 2, parser.StartRow())
}

func TestService_detectParserUnknown(t *testing.T) {
	service := &Service{
		resultParsers: []BankResultParser{NewMBankParser()},
	}

	rows := [][]string{
		{"header"},
	}

	_, err := service.detectParser("unknown.xlsx", rows)
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "Invalid file format")
	}
}

func TestMBankParser_ParseRowPadding(t *testing.T) {
	parser := NewMBankParser()
	raw := []string{"ignore", " 12345 ", " John Doe ", "unused", " 500000 ", " Salary ", " Success ", " ref "}

	row := parser.ParseRow(raw)

	assert.Equal(t, "12345", row.AccountNumber)
	assert.Equal(t, "John Doe", row.AccountName)
	assert.Equal(t, "500000", row.Amount)
	assert.Equal(t, "Success", row.TransferStatus)
	assert.Equal(t, "ref", row.TrackingData)
}

func TestMBankParser_ParseRowShortInput(t *testing.T) {
	parser := NewMBankParser()
	raw := []string{"only", " 12345 "}

	row := parser.ParseRow(raw)

	assert.Equal(t, "12345", row.AccountNumber)
	assert.Equal(t, "", row.AccountName)
	assert.Equal(t, "", row.Amount)
}
