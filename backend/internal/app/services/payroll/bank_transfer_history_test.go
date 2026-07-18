package payroll

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"api-server/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type bankHistoryFileRepoStub struct {
	files []*domain.BulkTransferFile
}

type bankHistoryProjectRepoStub struct {
	domain.ProjectRepository
	accessible []*domain.Project
	byID       map[uint]*domain.Project
	onList     func(domain.ProjectFilters)
}

func (s bankHistoryProjectRepoStub) List(_ context.Context, filters domain.ProjectFilters) ([]*domain.Project, error) {
	if s.onList != nil {
		s.onList(filters)
	}
	return s.accessible, nil
}

func (s bankHistoryProjectRepoStub) GetByIDs(context.Context, []uint) (map[uint]*domain.Project, error) {
	return s.byID, nil
}

type bankHistoryTransactionCodeRepoStub struct {
	domain.TransactionCodeRepository
	rows []*domain.TransactionCode
}

func (s bankHistoryTransactionCodeRepoStub) FindByCodes(context.Context, []string) ([]*domain.TransactionCode, error) {
	return s.rows, nil
}

func (s bankHistoryFileRepoStub) ListWeeklyForWorkMonth(context.Context, time.Time, time.Time) ([]*domain.BulkTransferFile, error) {
	return s.files, nil
}

func historyDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, clock.DefaultLocation)
}

func TestFixedWeeklyCycle(t *testing.T) {
	tests := []struct {
		from, to int
		want     int
		ok       bool
	}{
		{1, 7, 1, true},
		{8, 14, 2, true},
		{15, 21, 3, true},
		{22, 28, 4, true},
		{22, 31, 0, false},
		{29, 31, 0, false},
	}
	for _, test := range tests {
		got, ok := fixedWeeklyCycle(historyDate(2026, time.July, test.from), historyDate(2026, time.July, test.to))
		if got != test.want || ok != test.ok {
			t.Fatalf("%d-%d: got (%d,%v), want (%d,%v)", test.from, test.to, got, ok, test.want, test.ok)
		}
	}
}

func TestResolveWeeklyHistoryCycleFromTimesheets(t *testing.T) {
	workMonth := historyDate(2026, time.July, 1)
	weekly := &domain.CyclePayData{TimesheetIDs: []uint{10}}
	timesheets := map[uint]*domain.Timesheet{10: {ID: 10, Date: historyDate(2026, time.July, 23)}}

	cycle, fromDate, toDate, ok := resolveWeeklyHistoryCycle(&domain.BulkTransferFile{}, weekly, timesheets, workMonth)
	if !ok || cycle != 4 || fromDate.Day() != 22 || toDate.Day() != 28 {
		t.Fatalf("got cycle=%d from=%v to=%v ok=%v", cycle, fromDate, toDate, ok)
	}
}

func TestResolveWeeklyHistoryCycleExcludesDaysAfter28(t *testing.T) {
	workMonth := historyDate(2026, time.July, 1)
	weekly := &domain.CyclePayData{TimesheetIDs: []uint{10}}
	timesheets := map[uint]*domain.Timesheet{10: {ID: 10, Date: historyDate(2026, time.July, 29)}}

	_, _, _, ok := resolveWeeklyHistoryCycle(&domain.BulkTransferFile{}, weekly, timesheets, workMonth)
	if ok {
		t.Fatal("day 29 must not be assigned to a payroll cycle")
	}
}

func TestHistoryTransfersContainPaymentCodes(t *testing.T) {
	transfers := []dto.BankTransferHistoryTransfer{
		{TransferCode: "VFIC6d037214", BankReference: "FT26198846619959", Amount: 1548000},
		{TransferCode: "VFIC7a193042", BankReference: "FT26198940380850", Amount: 450000},
	}
	if !historyTransfersContain(transfers, "ft26198940380850") {
		t.Fatal("expected case-insensitive bank-reference match")
	}
	if !historyTransfersContain(transfers, "vfic6d037214") {
		t.Fatal("expected case-insensitive transfer-code match")
	}
}

func TestGetBankTransferHistoriesGroupsSplitReferencesScopesPartnerAndMatchesVietnameseName(t *testing.T) {
	ctrl := gomock.NewController(t)
	timesheetRepo := mocks.NewMockTimesheetRepository(ctrl)
	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)

	uploadedAt := time.Date(2026, time.July, 17, 20, 24, 0, 0, clock.DefaultLocation)
	rows := []dto.BulkTransferFileData{
		{EmployeeID: 82, ProjectID: 1, TransactionCode: "TX-1", Amount: 1_548_000, TransferStatus: "completed", BankTxnRef: "FT26198846619959"},
		{EmployeeID: 82, ProjectID: 1, TransactionCode: "TX-2", Amount: 450_000, TransferStatus: "completed", BankTxnRef: "FT26198940380850"},
		// Re-imported evidence for the same reference must not inflate the total.
		{EmployeeID: 82, ProjectID: 1, TransactionCode: "TX-3", Amount: 999, TransferStatus: "completed", BankTxnRef: " ft26198940380850 "},
		{EmployeeID: 82, ProjectID: 1, TransactionCode: "TX-4", Amount: 100_000, TransferStatus: "failed", BankTxnRef: "FAILED"},
		{EmployeeID: 99, ProjectID: 2, TransactionCode: "TX-5", Amount: 700_000, TransferStatus: "completed", BankTxnRef: "FT-UNRELATED"},
	}
	data, err := json.Marshal(rows)
	require.NoError(t, err)
	assetID := uint(7)
	cycle := "weekly"
	file := &domain.BulkTransferFile{ID: 5, Cycle: &cycle, AssetID: &assetID, Data: string(data), UploadedAt: &uploadedAt, CreatedAt: historyDate(2026, time.September, 1)}

	transactionRows := make([]*domain.TransactionCode, 0, 5)
	for index, code := range []string{"TX-1", "TX-2", "TX-3", "TX-4", "TX-5"} {
		employeeID, projectID := uint(82), uint(1)
		if code == "TX-5" {
			employeeID, projectID = 99, 2
		}
		codeData, marshalErr := json.Marshal(domain.TransactionCodeData{WeeklyPay: &domain.CyclePayData{
			TimesheetIDs: []uint{uint(index + 1)}, EmployeeID: employeeID, ProjectID: projectID,
		}})
		require.NoError(t, marshalErr)
		transactionRows = append(transactionRows, &domain.TransactionCode{Code: code, Data: codeData})
	}

	timesheetRepo.EXPECT().GetByIDs(gomock.Any(), gomock.Any()).Return([]*domain.Timesheet{
		{ID: 1, Date: historyDate(2026, time.July, 8)},
		{ID: 2, Date: historyDate(2026, time.July, 9)},
		{ID: 3, Date: historyDate(2026, time.July, 10)},
		{ID: 4, Date: historyDate(2026, time.July, 11)},
		{ID: 5, Date: historyDate(2026, time.July, 12)},
	}, nil).AnyTimes()
	employeeRepo.EXPECT().GetByIDs(gomock.Any(), []int64{82}).Return([]*domain.Employee{{ID: 82, Fullname: "LÒ THỊ MINH THU", CCCD: "031189014251"}}, nil).AnyTimes()
	projectRepo := bankHistoryProjectRepoStub{
		accessible: []*domain.Project{{ID: 1, Name: "Dự án A"}},
		byID:       map[uint]*domain.Project{1: {ID: 1, Name: "Dự án A"}},
		onList: func(filters domain.ProjectFilters) {
			require.NotNil(t, filters.AccessibleBy)
			require.Equal(t, uint(44), *filters.AccessibleBy)
			require.Equal(t, -1, filters.Limit)
		},
	}

	service := &PayrollService{
		timesheetRepo:           timesheetRepo,
		employeeRepo:            employeeRepo,
		projectRepo:             projectRepo,
		transactionCodeRepo:     bankHistoryTransactionCodeRepoStub{rows: transactionRows},
		bankTransferHistoryRepo: bankHistoryFileRepoStub{files: []*domain.BulkTransferFile{file}},
	}
	for _, search := range []string{"lo", "tX-1", "fT26198846619959"} {
		t.Run(search, func(t *testing.T) {
			result, err := service.GetBankTransferHistories(context.Background(), &dto.ListBankTransferHistoriesRequest{
				Month: "2026-07", Search: search, Page: 1, PageSize: 20,
			}, 44, "partner")
			require.NoError(t, err)
			require.Len(t, result.Data, 1)
			require.Equal(t, uint(82), result.Data[0].EmployeeID)
			require.Equal(t, "031189014251", result.Data[0].EmployeeCCCD)
			require.Equal(t, 2, result.Data[0].Cycle)
			require.Len(t, result.Data[0].Transfers, 2)
			require.Equal(t, "TX-1", result.Data[0].Transfers[0].TransferCode)
			require.Equal(t, "TX-2", result.Data[0].Transfers[1].TransferCode)
			require.Equal(t, "2026-07-17T20:24:00+07:00", result.Data[0].Transfers[0].PaidAt)
			require.Equal(t, "2026-07-17T20:24:00+07:00", result.Data[0].Transfers[1].PaidAt)
			require.Equal(t, int64(1_998_000), result.Data[0].TotalAmount)
		})
	}
}
