package settlement

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"api-server/internal/domain"
	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/infra/disbursement/onepay"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestParseOnePayFeeReport(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, buildOnePayFeeWorkbook().Write(&buf))

	report, issues, err := parseOnePayFeeReport(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	require.Empty(t, issues)
	require.NotNil(t, report)

	require.Equal(t, "VFICPO", report.summary.MerchantID)
	require.Equal(t, "2026-06-01", report.summary.PeriodFrom)
	require.Equal(t, "2026-06-30", report.summary.PeriodTo)
	require.Equal(t, 2, report.summary.TransactionCount)
	require.Equal(t, int64(3850), report.summary.FeePerTransaction)
	require.Equal(t, int64(7700), report.summary.TotalFee)
	require.Equal(t, int64(300000), report.summary.DetailTotalAmount)
	require.Equal(t, "ONEPAY-FEE:VFICPO:2026-06", report.summary.ImportReference)
}

func TestParseOnePayFeeReportDetectsCountMismatch(t *testing.T) {
	t.Parallel()

	wb := buildOnePayFeeWorkbook()
	require.NoError(t, wb.SetCellInt("PHI THANG", "C12", 3))

	var buf bytes.Buffer
	require.NoError(t, wb.Write(&buf))

	_, issues, err := parseOnePayFeeReport(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	require.NotEmpty(t, issues)
	require.Equal(t, "transaction_count_mismatch", issues[0].Code)
}

func TestParseMoney(t *testing.T) {
	t.Parallel()

	require.Equal(t, int64(3850), parseMoney("3850"))
	require.Equal(t, int64(3850), parseMoney("3,850"))
	require.Equal(t, int64(3850), parseMoney("3.850"))
	require.Equal(t, int64(3850), parseMoney("3850.00"))
}

func TestValidateWalletPaymentsMatchesReportBankToStoredSwiftCode(t *testing.T) {
	t.Parallel()

	invoiceOne := "OP001"
	invoiceTwo := "OP002"
	service := &OnePayFeeImportService{
		walletPayments: &fakeOnePayFeeWalletPayments{
			byRequestID: map[string]*domaintx.WalletPayment{
				"tt001": {
					RequestID:          "tt001",
					InvoiceNo:          &invoiceOne,
					Provider:           onepay.ProviderName,
					Status:             domaintx.StateCompleted,
					RequestedAmount:    100000,
					Fee:                3850,
					RecipientName:      "Nguyen Van A",
					RecipientAccountNo: "00112233",
					RecipientBank:      "MSCBVNVX",
				},
				"tt002": {
					RequestID:          "tt002",
					InvoiceNo:          &invoiceTwo,
					Provider:           onepay.ProviderName,
					Status:             domaintx.StateCompleted,
					RequestedAmount:    200000,
					Fee:                3850,
					RecipientName:      "Tran Van B",
					RecipientAccountNo: "00445566",
					RecipientBank:      "VTCBVNVX",
				},
			},
		},
		bankRepo: &fakeOnePayFeeBankRepo{
			banks: []*domain.Bank{
				{BranchName: "Quân đội (MB)", BankCode: "MB", SwiftCode: "MSCBVNVX"},
				{BranchName: "Kỹ Thương (TCB)", BankCode: "TCB", SwiftCode: "VTCBVNVX"},
			},
		},
	}
	report := &onePayFeeReport{
		details: []onePayFeeReportDetail{
			{
				row:                    2,
				fundTransferID:         "tt001",
				opTransactionID:        "OP001",
				beneficiaryAccount:     "00112233",
				beneficiaryAccountName: "NGUYEN VAN A",
				beneficiaryBank:        "MB",
				amount:                 100000,
			},
			{
				row:                    3,
				fundTransferID:         "tt002",
				opTransactionID:        "OP002",
				beneficiaryAccount:     "00445566",
				beneficiaryAccountName: "TRAN VAN B",
				beneficiaryBank:        "TECHCOMBANK",
				amount:                 200000,
			},
		},
	}

	issues := service.validateWalletPayments(context.Background(), report)

	require.Empty(t, issues)
	require.Equal(t, int64(7700), report.summary.AppRecordedFeeTotal)
}

func buildOnePayFeeWorkbook() *excelize.File {
	wb := excelize.NewFile()
	defaultSheet := wb.GetSheetName(0)
	_ = wb.SetSheetName(defaultSheet, "PHI THANG")
	_, _ = wb.NewSheet("GD")

	_ = wb.SetCellValue("PHI THANG", "A5", "Từ ngày 01/06/2026 Đến ngày 30/06/2026")
	_ = wb.SetCellValue("PHI THANG", "A6", "CÔNG TY CỔ PHẦN QUỐC TẾ THƯƠNG MẠI VÀ DỊCH VỤ VIỆT PHÁP- PO - VFICPO")
	_ = wb.SetCellValue("PHI THANG", "A11", "STT")
	_ = wb.SetCellValue("PHI THANG", "B11", "Thời gian")
	_ = wb.SetCellValue("PHI THANG", "C11", "SLGD")
	_ = wb.SetCellValue("PHI THANG", "D11", "Phí XLGD")
	_ = wb.SetCellValue("PHI THANG", "E11", "Tổng phí")
	_ = wb.SetCellValue("PHI THANG", "A12", 1)
	_ = wb.SetCellValue("PHI THANG", "B12", "Tháng 06.2026")
	_ = wb.SetCellValue("PHI THANG", "C12", 2)
	_ = wb.SetCellValue("PHI THANG", "D12", 3850)
	_ = wb.SetCellFormula("PHI THANG", "E12", "=C12*D12")

	headers := []any{
		"No",
		"Merchant ID",
		"Merchant Account",
		"Merchant Name",
		"Merchant Fund Transfer ID",
		"OP Transaction ID",
		"Bank Trans ID",
		"Create Date",
		"Fund Transfer Date",
		"Update Date",
		"Currency",
		"Beneficiary Account",
		"Beneficiary Account Name",
		"Beneficiary Bank",
		"Amount",
		"Remark",
		"State",
	}
	_ = wb.SetSheetRow("GD", "A1", &headers)
	row2 := []any{1, "VFICPO", "666999764888", "VFIC MANPOWER", "tt001", "OP001", "BANK001", "06-06-2026 12:26 PM", "06-06-2026 12:26 PM", "06-06-2026 12:26 PM", "VND", "00112233", "NGUYEN VAN A", "MB", 100000, "tt001", "Approved"}
	row3 := []any{2, "VFICPO", "666999764888", "VFIC MANPOWER", "tt002", "OP002", "BANK002", "07-06-2026 11:04 AM", "07-06-2026 11:04 AM", "07-06-2026 11:04 AM", "VND", "00445566", "TRAN VAN B", "MSB", 200000, "tt002", "Approved"}
	_ = wb.SetSheetRow("GD", "A2", &row2)
	_ = wb.SetSheetRow("GD", "A3", &row3)

	return wb
}

type fakeOnePayFeeWalletPayments struct {
	byRequestID map[string]*domaintx.WalletPayment
}

func (r *fakeOnePayFeeWalletPayments) Create(context.Context, *domaintx.WalletPayment) error {
	return nil
}

func (r *fakeOnePayFeeWalletPayments) GetByID(context.Context, uint64) (*domaintx.WalletPayment, error) {
	return nil, domaintx.ErrNotFound
}

func (r *fakeOnePayFeeWalletPayments) GetByRequestID(_ context.Context, requestID string) (*domaintx.WalletPayment, error) {
	row, ok := r.byRequestID[requestID]
	if !ok {
		return nil, domaintx.ErrNotFound
	}
	return row, nil
}

func (r *fakeOnePayFeeWalletPayments) GetByProviderInvoiceNo(context.Context, string, string) (*domaintx.WalletPayment, error) {
	return nil, domaintx.ErrNotFound
}

func (r *fakeOnePayFeeWalletPayments) GetByTxnID(context.Context, string) (*domaintx.WalletPayment, error) {
	return nil, domaintx.ErrNotFound
}

func (r *fakeOnePayFeeWalletPayments) ListRecent(context.Context, int) ([]*domaintx.WalletPayment, error) {
	return nil, nil
}

func (r *fakeOnePayFeeWalletPayments) UpdateExpected(context.Context, uint64, int64, domaintx.UpdatePatch) error {
	return nil
}

func (r *fakeOnePayFeeWalletPayments) MarkReconciled(context.Context, uint64, domaintx.State, time.Time) error {
	return nil
}

func (r *fakeOnePayFeeWalletPayments) ListByStatuses(context.Context, []domaintx.State, time.Time, time.Time) ([]*domaintx.WalletPayment, error) {
	return nil, nil
}

func (r *fakeOnePayFeeWalletPayments) StatsByErrorCode(context.Context, time.Time, time.Time) ([]domaintx.ErrorCodeStat, error) {
	return nil, nil
}

func (r *fakeOnePayFeeWalletPayments) StatsByStatus(context.Context, time.Time, time.Time) ([]domaintx.StatusStat, error) {
	return nil, nil
}

func (r *fakeOnePayFeeWalletPayments) ListByBatchID(context.Context, string) ([]*domaintx.WalletPayment, error) {
	return nil, nil
}

func (r *fakeOnePayFeeWalletPayments) ListByProviderAndCreatedRange(context.Context, string, time.Time, time.Time) ([]*domaintx.WalletPayment, error) {
	return nil, nil
}

func (r *fakeOnePayFeeWalletPayments) ListStaleAuthorised(context.Context, string, time.Time, int) ([]*domaintx.WalletPayment, error) {
	return nil, nil
}

func (r *fakeOnePayFeeWalletPayments) HasPendingForRecipient(context.Context, string, string, string) (bool, error) {
	return false, nil
}

func (r *fakeOnePayFeeWalletPayments) HasNonTerminalByEntityID(context.Context, uint64) (bool, error) {
	return false, nil
}

// Bulk-transfer worker helpers (Phase 3 of wallet bulk transfer pipeline).
// No-op stubs — the OnePay fee import flow doesn't touch bulk batches.
func (r *fakeOnePayFeeWalletPayments) UpdateBulkBatchLink(context.Context, uint64, uint64, uint, string) error {
	return nil
}
func (r *fakeOnePayFeeWalletPayments) CountByBatchAndStatuses(context.Context, uint64, []domaintx.State) (int64, error) {
	return 0, nil
}
func (r *fakeOnePayFeeWalletPayments) SumFeeByBatchAndStatuses(context.Context, uint64, []domaintx.State) (int64, error) {
	return 0, nil
}
func (r *fakeOnePayFeeWalletPayments) ListByBatchIDOrdered(context.Context, uint64) ([]*domaintx.WalletPayment, error) {
	return nil, nil
}
func (r *fakeOnePayFeeWalletPayments) IncrementSweeperRetry(context.Context, uint64) (uint, error) {
	return 0, nil
}

type fakeOnePayFeeBankRepo struct {
	banks []*domain.Bank
}

func (r *fakeOnePayFeeBankRepo) Create(context.Context, *domain.Bank) error { return nil }

func (r *fakeOnePayFeeBankRepo) GetByID(context.Context, uint) (*domain.Bank, error) {
	return nil, errors.New("not found")
}

func (r *fakeOnePayFeeBankRepo) Update(context.Context, *domain.Bank) error { return nil }

func (r *fakeOnePayFeeBankRepo) Delete(context.Context, uint) error { return nil }

func (r *fakeOnePayFeeBankRepo) List(context.Context, domain.BankFilters) ([]*domain.Bank, error) {
	return r.banks, nil
}

func (r *fakeOnePayFeeBankRepo) Count(context.Context, domain.BankFilters) (int64, error) {
	return int64(len(r.banks)), nil
}

func (r *fakeOnePayFeeBankRepo) SearchByBranchName(_ context.Context, searchTerm string, limit int) ([]*domain.Bank, error) {
	var matches []*domain.Bank
	for _, bank := range r.banks {
		if normalizeName(bank.BranchName) == normalizeName(searchTerm) {
			matches = append(matches, bank)
			if limit > 0 && len(matches) >= limit {
				break
			}
		}
	}
	return matches, nil
}

func (r *fakeOnePayFeeBankRepo) FindByBankCode(_ context.Context, bankCode string) (*domain.Bank, error) {
	for _, bank := range r.banks {
		if normalizeText(bank.BankCode) == normalizeText(bankCode) {
			return bank, nil
		}
	}
	return nil, errors.New("not found")
}

func (r *fakeOnePayFeeBankRepo) FindBySwiftCode(_ context.Context, swiftCode string) (*domain.Bank, error) {
	for _, bank := range r.banks {
		if normalizeText(bank.SwiftCode) == normalizeText(swiftCode) {
			return bank, nil
		}
	}
	return nil, errors.New("not found")
}
