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
	timesheetDates := map[uint]time.Time{10: historyDate(2026, time.July, 23)}

	cycle, fromDate, toDate, ok := resolveWeeklyHistoryCycle(&domain.BulkTransferFile{}, weekly, timesheetDates, workMonth)
	if !ok || cycle != 4 || fromDate.Day() != 22 || toDate.Day() != 28 {
		t.Fatalf("got cycle=%d from=%v to=%v ok=%v", cycle, fromDate, toDate, ok)
	}
}

func TestResolveWeeklyHistoryCycleExcludesDaysAfter28(t *testing.T) {
	workMonth := historyDate(2026, time.July, 1)
	weekly := &domain.CyclePayData{TimesheetIDs: []uint{10}}
	timesheetDates := map[uint]time.Time{10: historyDate(2026, time.July, 29)}

	_, _, _, ok := resolveWeeklyHistoryCycle(&domain.BulkTransferFile{}, weekly, timesheetDates, workMonth)
	if ok {
		t.Fatal("day 29 must not be assigned to a payroll cycle")
	}
}

// TestResolveWeeklyHistoryCycleFromPersistedFromDate covers the Phase A fast
// path: CyclePayData carries FromDate, so the cycle resolves via KyFromWorkDay
// WITHOUT consulting timesheetDates and WITHOUT fixedWeeklyCycle (which would
// reject off-boundary export ranges — red-team F2).
func TestResolveWeeklyHistoryCycleFromPersistedFromDate(t *testing.T) {
	workMonth := historyDate(2026, time.July, 1)
	tests := []struct {
		name     string
		day      int // FromDate day-of-month (off-boundary allowed)
		wantKy   int
		wantFrom int // expected canonical cycle start day
		wantTo   int // expected canonical cycle end day
	}{
		{"canonical Ky1 (day 1)", 1, 1, 1, 7},
		{"off-boundary Ky1 (day 3, red-team F2)", 3, 1, 1, 7},
		{"off-boundary Ky2 (day 10)", 10, 2, 8, 14},
		{"canonical Ky3 (day 15)", 15, 3, 15, 21},
		{"canonical Ky4 (day 22)", 22, 4, 22, 28},
		{"off-boundary Ky4 (day 25)", 25, 4, 22, 28},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fromDate := historyDate(2026, time.July, tc.day)
			weekly := &domain.CyclePayData{FromDate: &fromDate, TimesheetIDs: []uint{999}}
			// timesheetDates intentionally has NO entry for 999 — the persisted
			// path must resolve without consulting it.
			cycle, from, to, ok := resolveWeeklyHistoryCycle(&domain.BulkTransferFile{}, weekly, map[uint]time.Time{}, workMonth)
			require.True(t, ok, "expected cycle to resolve from persisted FromDate")
			require.Equal(t, tc.wantKy, cycle, "cycle mismatch")
			require.Equal(t, tc.wantFrom, from.Day(), "fromDate day mismatch")
			require.Equal(t, tc.wantTo, to.Day(), "toDate day mismatch")
		})
	}
}

// TestResolveWeeklyHistoryCyclePerEntryMixedKy covers red-team F4: a file with
// Ky1 and Ky3 entries must resolve each entry to its own cycle, not the first
// entry's cycle. Per-entry resolution from weeklyData.FromDate handles this.
func TestResolveWeeklyHistoryCyclePerEntryMixedKy(t *testing.T) {
	workMonth := historyDate(2026, time.July, 1)
	ky1From := historyDate(2026, time.July, 1)
	ky3From := historyDate(2026, time.July, 15)
	ky1 := &domain.CyclePayData{FromDate: &ky1From, TimesheetIDs: []uint{1}}
	ky3 := &domain.CyclePayData{FromDate: &ky3From, TimesheetIDs: []uint{2}}

	cycle1, _, _, ok1 := resolveWeeklyHistoryCycle(&domain.BulkTransferFile{}, ky1, map[uint]time.Time{}, workMonth)
	require.True(t, ok1)
	require.Equal(t, 1, cycle1, "Ky1 entry must resolve to cycle 1")

	cycle3, _, _, ok3 := resolveWeeklyHistoryCycle(&domain.BulkTransferFile{}, ky3, map[uint]time.Time{}, workMonth)
	require.True(t, ok3)
	require.Equal(t, 3, cycle3, "Ky3 entry must resolve to cycle 3 (not the file's first entry's cycle)")
}

// TestResolveWeeklyHistoryCycleZeroValueFromDate covers red-team F-MEDIUM-1:
// a Go zero-value time.Time (year 1) must fall through to the legacy path
// rather than producing a bogus cycle. The Year()==workMonth.Year() guard
// rejects it.
func TestResolveWeeklyHistoryCycleZeroValueFromDate(t *testing.T) {
	workMonth := historyDate(2026, time.July, 1)
	zero := time.Time{}
	weekly := &domain.CyclePayData{FromDate: &zero, TimesheetIDs: []uint{10}}
	// Legacy path: timesheet date present → resolves via timesheet lookup.
	timesheetDates := map[uint]time.Time{10: historyDate(2026, time.July, 23)}

	cycle, _, _, ok := resolveWeeklyHistoryCycle(&domain.BulkTransferFile{}, weekly, timesheetDates, workMonth)
	require.True(t, ok, "zero-value FromDate must fall through to timesheet path")
	require.Equal(t, 4, cycle, "timesheet on day 23 → Ky4")
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
	// Phase A: all codes carry persisted FromDate, so the union fetch is skipped.
	// Each code's FromDate determines its cycle (TX-1..TX-4 → Ky2 day 8-11;
	// TX-5 → Ky2 day 10). All resolve to Ky2.
	fromDate := historyDate(2026, time.July, 8)
	toDate := historyDate(2026, time.July, 14)
	for index, code := range []string{"TX-1", "TX-2", "TX-3", "TX-4", "TX-5"} {
		employeeID, projectID := uint(82), uint(1)
		if code == "TX-5" {
			employeeID, projectID = 99, 2
		}
		codeData, marshalErr := json.Marshal(domain.TransactionCodeData{WeeklyPay: &domain.CyclePayData{
			TimesheetIDs: []uint{uint(index + 1)}, EmployeeID: employeeID, ProjectID: projectID,
			FromDate: &fromDate, ToDate: &toDate, CycleNum: 2,
		}})
		require.NoError(t, marshalErr)
		transactionRows = append(transactionRows, &domain.TransactionCode{Code: code, Data: codeData})
	}

	// Phase A: GetTimesheetDatesByIDs MUST NOT be called when all codes carry FromDate.
	timesheetRepo.EXPECT().GetTimesheetDatesByIDs(gomock.Any(), gomock.Any()).Times(0)
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
			// Summary aggregates the SEARCH-FILTERED result set, not the raw scan.
			require.Equal(t, int64(1_998_000), result.Summary.TotalAmount)
			require.Equal(t, 2, result.Summary.TransferCount)
			require.Equal(t, 1, result.Summary.EmployeeCount)
		})
	}
}

// TestGetBankTransferHistoriesSummaryAggregatesAllScopedEmployees proves the
// summary spans every accessible employee-cycle, not just the current page,
// and stays consistent with the deduped per-item totals.
func TestGetBankTransferHistoriesSummaryAggregatesAllScopedEmployees(t *testing.T) {
	ctrl := gomock.NewController(t)
	timesheetRepo := mocks.NewMockTimesheetRepository(ctrl)
	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)

	uploadedAt := time.Date(2026, time.July, 17, 20, 24, 0, 0, clock.DefaultLocation)
	rows := []dto.BulkTransferFileData{
		{EmployeeID: 82, ProjectID: 1, TransactionCode: "TX-1", Amount: 1_548_000, TransferStatus: "completed", BankTxnRef: "FT26198846619959"},
		{EmployeeID: 82, ProjectID: 1, TransactionCode: "TX-2", Amount: 450_000, TransferStatus: "completed", BankTxnRef: "FT26198940380850"},
		{EmployeeID: 99, ProjectID: 2, TransactionCode: "TX-5", Amount: 700_000, TransferStatus: "completed", BankTxnRef: "FT-UNRELATED"},
	}
	data, err := json.Marshal(rows)
	require.NoError(t, err)
	assetID := uint(7)
	cycle := "weekly"
	file := &domain.BulkTransferFile{ID: 5, Cycle: &cycle, AssetID: &assetID, Data: string(data), UploadedAt: &uploadedAt, CreatedAt: historyDate(2026, time.September, 1)}

	fromDate := historyDate(2026, time.July, 8)
	toDate := historyDate(2026, time.July, 14)
	transactionRows := make([]*domain.TransactionCode, 0, 3)
	for index, code := range []string{"TX-1", "TX-2", "TX-5"} {
		employeeID, projectID := uint(82), uint(1)
		if code == "TX-5" {
			employeeID, projectID = 99, 2
		}
		codeData, marshalErr := json.Marshal(domain.TransactionCodeData{WeeklyPay: &domain.CyclePayData{
			TimesheetIDs: []uint{uint(index + 1)}, EmployeeID: employeeID, ProjectID: projectID,
			FromDate: &fromDate, ToDate: &toDate, CycleNum: 2,
		}})
		require.NoError(t, marshalErr)
		transactionRows = append(transactionRows, &domain.TransactionCode{Code: code, Data: codeData})
	}

	timesheetRepo.EXPECT().GetTimesheetDatesByIDs(gomock.Any(), gomock.Any()).Times(0)
	employeeRepo.EXPECT().GetByIDs(gomock.Any(), gomock.Any()).Return([]*domain.Employee{
		{ID: 82, Fullname: "LÒ THỊ MINH THU", CCCD: "031189014251"},
		{ID: 99, Fullname: "NGUYỄN VĂN B", CCCD: "031000001111"},
	}, nil).AnyTimes()
	projectRepo := bankHistoryProjectRepoStub{
		accessible: []*domain.Project{{ID: 1, Name: "Dự án A"}, {ID: 2, Name: "Dự án B"}},
		byID:       map[uint]*domain.Project{1: {ID: 1, Name: "Dự án A"}, 2: {ID: 2, Name: "Dự án B"}},
	}

	service := &PayrollService{
		timesheetRepo:           timesheetRepo,
		employeeRepo:            employeeRepo,
		projectRepo:             projectRepo,
		transactionCodeRepo:     bankHistoryTransactionCodeRepoStub{rows: transactionRows},
		bankTransferHistoryRepo: bankHistoryFileRepoStub{files: []*domain.BulkTransferFile{file}},
	}

	// Page size 1: summary must still cover BOTH employee-cycles.
	result, err := service.GetBankTransferHistories(context.Background(), &dto.ListBankTransferHistoriesRequest{
		Month: "2026-07", Page: 1, PageSize: 1,
	}, 44, "partner")
	require.NoError(t, err)
	require.Len(t, result.Data, 1)
	require.Equal(t, int64(2), result.Pagination.TotalRecords)
	require.Equal(t, int64(2_698_000), result.Summary.TotalAmount)
	require.Equal(t, 3, result.Summary.TransferCount)
	require.Equal(t, 2, result.Summary.EmployeeCount)
}

// TestGetBankTransferHistoriesSummaryZeroWhenEmpty covers the no-data path.
func TestGetBankTransferHistoriesSummaryZeroWhenEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	timesheetRepo := mocks.NewMockTimesheetRepository(ctrl)
	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	projectRepo := bankHistoryProjectRepoStub{}

	service := &PayrollService{
		timesheetRepo:           timesheetRepo,
		employeeRepo:            employeeRepo,
		projectRepo:             projectRepo,
		transactionCodeRepo:     bankHistoryTransactionCodeRepoStub{},
		bankTransferHistoryRepo: bankHistoryFileRepoStub{},
	}

	result, err := service.GetBankTransferHistories(context.Background(), &dto.ListBankTransferHistoriesRequest{
		Month: "2026-07", Page: 1, PageSize: 20,
	}, 44, "partner")
	require.NoError(t, err)
	require.Empty(t, result.Data)
	require.Equal(t, dto.BankTransferHistorySummary{}, result.Summary)
}
