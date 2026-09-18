package bulktransfer

import (
	"api-server/internal/app/services/payroll/excel"
	"bytes"
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestOnePayExportUsesPlannedPercentageForWorkbookAndTransferCode(t *testing.T) {
	for _, cycle := range []string{"weekly", "monthly"} {
		t.Run(cycle, func(t *testing.T) {
			bundle := seedStub()
			for _, assignment := range bundle.assignments {
				assignment.PaymentSchedule = cycle
			}
			planner := newPlannerWithStub(bundle)
			planner.periodCalculator = NewPeriodCalculator(planner.projectRepo)
			bundle.weeklyPercentage = .7
			planner.excelService = excel.NewService(&stubSettings{pct: .7})
			codes := &stubTransactionCodeRepo{}
			exporter := NewOnePayExporter(planner, &stubBankRepo{byCode: map[string]*domain.Bank{
				"MB": {ID: 1, BankCode: "MB", SwiftCode: "MBBEVNVX"},
			}}, codes, nil)
			request := &dto.ExportBulkTransferRequest{FromDate: "2026-07-08", ToDate: "2026-07-14"}
			if cycle == "monthly" {
				request = &dto.ExportBulkTransferRequest{ForMonth: "2026-07", ProjectIDs: []uint{12}}
			}
			result, err := exporter.Export(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, int64(5_600_000), result.TransferAmount, "8 million gross at 70 percent")
			require.Len(t, codes.created, 1)
			var code domain.TransactionCodeData
			require.NoError(t, json.Unmarshal(codes.created[0].Data, &code))
			require.Equal(t, result.TransferAmount, code.GetAmount())
			require.NotNil(t, code.GetTransferAmountSnapshot())
			require.Equal(t, result.TransferAmount, *code.GetTransferAmountSnapshot())
			workbook, err := excelize.OpenReader(bytes.NewReader(result.ExcelBytes))
			require.NoError(t, err)
			defer func() { _ = workbook.Close() }()
			amount, err := workbook.GetCellValue(onePaySheetName, "F3")
			require.NoError(t, err)
			require.Equal(t, "5600000", amount)
		})
	}
}

func TestOnePayExportRejectsInvalidPaymentPercentages(t *testing.T) {
	exporter := NewOnePayExporter(nil, nil, nil, nil)
	for _, percentage := range []float64{0, -1, 1.1, math.NaN(), math.Inf(1)} {
		_, _, _, err := exporter.buildRowsWithSwift(context.Background(), &excel.BulkTransferData{}, "weekly", time.Time{}, time.Time{}, percentage)
		require.Error(t, err)
	}
}
