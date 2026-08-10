package bulktransfer

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/payroll/excel"
	"api-server/internal/domain"
	pkgConstants "api-server/internal/pkg/constants"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

type recordingEventBus struct {
	domain.EventBus
	events []domain.DomainEvent
}

func (b *recordingEventBus) Publish(_ context.Context, events ...domain.DomainEvent) error {
	b.events = append(b.events, events...)
	return nil
}

func TestExportConfiguredLimitFailureDoesNotPersistTransactionCodes(t *testing.T) {
	bundle := seedStub()
	bundle.weeklyPercentage = 1
	bundle.timesheets = bundle.timesheets[:1]
	bundle.timesheets[0].Amount = 400_000_000
	bundle.timesheetByID = map[uint]*domain.Timesheet{
		bundle.timesheets[0].ID: bundle.timesheets[0],
	}

	excelService := excel.NewService(&stubSettings{pct: 1, limit: 400_000_000})
	planner := newPlannerWithStub(bundle)
	planner.excelService = excelService
	transactionCodes := &stubTransactionCodeRepo{}
	service := &ExportService{
		planner:             planner,
		transactionCodeRepo: transactionCodes,
		excelService:        excelService,
	}

	response, err := service.Export(context.Background(), &dto.ExportBulkTransferRequest{
		FromDate:  "2026-07-08",
		ToDate:    "2026-07-14",
		CreatedBy: 1,
	})

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Empty(t, transactionCodes.created)
}

func TestExportRefreshesBankEligibilityBeforeWorkbookAndAudit(t *testing.T) {
	withBackendWorkingDirectory(t)
	bundle := seedStub()
	bundle.timesheets[0].Date = time.Date(2026, time.July, 8, 0, 0, 0, 0, time.UTC)
	bundle.timesheets[1].Date = time.Date(2026, time.July, 14, 0, 0, 0, 0, time.UTC)
	bundle.beforeBatchRead = func() {
		bundle.employees[7].BankAccountStatus = domain.BankAccountStatusInvalid
	}

	excelService := excel.NewService(&stubSettings{pct: 1})
	planner := newPlannerWithStub(bundle)
	planner.excelService = excelService
	transactionCodes := &stubTransactionCodeRepo{}
	eventBus := &recordingEventBus{}
	service := &ExportService{
		planner:             planner,
		transactionCodeRepo: transactionCodes,
		excelService:        excelService,
		eventBus:            eventBus,
	}

	response, err := service.Export(context.Background(), &dto.ExportBulkTransferRequest{
		FromDate:  "2026-07-08",
		ToDate:    "2026-07-14",
		CreatedBy: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, response.IncludedCount)
	assert.Equal(t, 2, response.SkippedCount)
	assert.Empty(t, transactionCodes.created)

	workbook, err := excelize.OpenReader(bytes.NewReader(response.Data))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, workbook.Close()) })
	sequence, err := workbook.GetCellValue(pkgConstants.MBank_SheetName, "A3")
	require.NoError(t, err)
	assert.Empty(t, sequence)

	require.Len(t, eventBus.events, 1)
	event, ok := eventBus.events[0].(domain.BulkTransferFileExportedEvent)
	require.True(t, ok)
	assert.Zero(t, event.TransactionsCount)
	assert.Zero(t, event.TotalAmount)
	assert.Empty(t, event.ForecastOutcomeItems)
}

func withBackendWorkingDirectory(t *testing.T) {
	t.Helper()
	originalDirectory, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir("../../../../../"))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalDirectory))
	})
}
