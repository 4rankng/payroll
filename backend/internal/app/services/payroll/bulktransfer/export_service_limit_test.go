package bulktransfer

import (
	"context"
	"testing"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/payroll/excel"
	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
