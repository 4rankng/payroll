package bulktransfer

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"api-server/internal/app/services/payroll/excel"
	"api-server/internal/domain"

	"github.com/xuri/excelize/v2"
)

// stubBankRepo is a tiny in-memory BankRepository for exporter tests.
// Embeds the interface so we only override FindByBankCode (the only method
// the exporter actually calls).
type stubBankRepo struct {
	domain.BankRepository
	byCode map[string]*domain.Bank
	err    error
}

func (s *stubBankRepo) FindByBankCode(_ context.Context, code string) (*domain.Bank, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.byCode[code], nil
}

// stubTransactionCodeRepo captures CreateBatch calls so tests can assert VFIC persistence.
type stubTransactionCodeRepo struct {
	created []*domain.TransactionCode
	err     error
}

func (s *stubTransactionCodeRepo) Create(context.Context, *domain.TransactionCode) error { return nil }
func (s *stubTransactionCodeRepo) GetByCode(context.Context, string) (*domain.TransactionCode, error) {
	return nil, nil
}
func (s *stubTransactionCodeRepo) GetAllCodes(context.Context) (map[string]struct{}, error) {
	return nil, nil
}
func (s *stubTransactionCodeRepo) CreateBatch(ctx context.Context, tcs []*domain.TransactionCode) error {
	if s.err != nil {
		return s.err
	}
	s.created = append(s.created, tcs...)
	return nil
}
func (s *stubTransactionCodeRepo) FindByCodes(context.Context, []string) ([]*domain.TransactionCode, error) {
	return nil, nil
}
func (s *stubTransactionCodeRepo) UpdateFileIDByCodes(context.Context, []string, uint) error {
	return nil
}

// TestBuildRowsWithSwift_HappyPath: 2 employees with valid bank info → 2 rows
// with SWIFT codes resolved from the bank repo.
func TestBuildRowsWithSwift_HappyPath(t *testing.T) {
	bankRepo := &stubBankRepo{byCode: map[string]*domain.Bank{
		"MB":  {ID: 1, BankCode: "MB", SwiftCode: "MBBEVNVX", BranchName: "Quân đội (MB)"},
		"VCB": {ID: 2, BankCode: "VCB", SwiftCode: "BFTVVNVX", BranchName: "Vietcombank"},
	}}
	txnRepo := &stubTransactionCodeRepo{}
	e := NewOnePayExporter(nil, bankRepo, txnRepo, nil)

	data := &excel.BulkTransferData{
		EmployeeProjectAmounts: map[excel.EmployeeProjectKey]int64{
			{EmployeeID: 1, ProjectID: 10}: 1_500_000,
			{EmployeeID: 2, ProjectID: 10}: 2_500_000,
		},
		EmployeeProjectTimesheets: map[excel.EmployeeProjectKey][]uint{
			{EmployeeID: 1, ProjectID: 10}: {100, 101},
			{EmployeeID: 2, ProjectID: 10}: {200},
		},
		EmployeeData: map[uint]excel.Employee{
			1: {ID: 1, Fullname: "Nguyen Test A", BankAccountNumber: "99990001", BankAccountName: "NGUYEN TEST A", Bank: &excel.Bank{ID: 1, BankCode: "MB", BranchName: "Quân đội (MB)"}},
			2: {ID: 2, Fullname: "Nguyen Test B", BankAccountNumber: "99990002", BankAccountName: "NGUYEN TEST B", Bank: &excel.Bank{ID: 2, BankCode: "VCB", BranchName: "Vietcombank"}},
		},
		ProjectData: map[uint]excel.Project{10: {ID: 10, Name: "Project Alpha"}},
		TransactionCodes: map[excel.EmployeeProjectKey]string{
			{EmployeeID: 1, ProjectID: 10}: "VFIC3ba3ec31",
			{EmployeeID: 2, ProjectID: 10}: "VFIC4cd5ef62",
		},
	}

	rows, skipped, txnCodes, err := e.buildRowsWithSwift(context.Background(), data, "weekly")
	if err != nil {
		t.Fatalf("buildRowsWithSwift: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if len(skipped) != 0 {
		t.Fatalf("expected 0 skipped, got %d: %+v", len(skipped), skipped)
	}
	if len(txnCodes) != 2 {
		t.Fatalf("expected 2 txn codes, got %d", len(txnCodes))
	}
	if rows[0].SwiftCode != "MBBEVNVX" {
		t.Errorf("row 0 swift: got %q, want MBBEVNVX", rows[0].SwiftCode)
	}
	if rows[0].VFICCode != "VFIC3ba3ec31" {
		t.Errorf("row 0 vfic: got %q", rows[0].VFICCode)
	}
	if rows[1].SwiftCode != "BFTVVNVX" {
		t.Errorf("row 1 swift: got %q, want BFTVVNVX", rows[1].SwiftCode)
	}
}

// TestBuildRowsWithSwift_SkipPaths: missing_bank_info + unresolved_swift.
func TestBuildRowsWithSwift_SkipPaths(t *testing.T) {
	bankRepo := &stubBankRepo{byCode: map[string]*domain.Bank{
		"MB": {ID: 1, BankCode: "MB", SwiftCode: "MBBEVNVX", BranchName: "MB"},
		// "NONE" intentionally absent — employee 3's lookup returns nil
	}}
	txnRepo := &stubTransactionCodeRepo{}
	e := NewOnePayExporter(nil, bankRepo, txnRepo, nil)

	data := &excel.BulkTransferData{
		EmployeeProjectAmounts: map[excel.EmployeeProjectKey]int64{
			{EmployeeID: 1, ProjectID: 10}: 1_500_000, // has bank
			{EmployeeID: 2, ProjectID: 10}: 2_500_000, // missing bank_id
			{EmployeeID: 3, ProjectID: 10}: 3_500_000, // bank_code "NONE" doesn't exist in repo
		},
		EmployeeData: map[uint]excel.Employee{
			1: {ID: 1, Fullname: "Has Bank", BankAccountNumber: "99990001", BankAccountName: "HAS BANK", Bank: &excel.Bank{ID: 1, BankCode: "MB", BranchName: "MB"}},
			2: {ID: 2, Fullname: "No BankID", BankAccountNumber: "99990002", BankAccountName: "NO BANK", Bank: nil},
			3: {ID: 3, Fullname: "Bad BankCode", BankAccountNumber: "99990003", BankAccountName: "BAD", Bank: &excel.Bank{ID: 3, BankCode: "NONE", BranchName: "None"}},
		},
		ProjectData: map[uint]excel.Project{10: {ID: 10, Name: "P"}},
		TransactionCodes: map[excel.EmployeeProjectKey]string{
			{EmployeeID: 1, ProjectID: 10}: "VFICaaaa1111",
			{EmployeeID: 2, ProjectID: 10}: "VFICbbbb2222",
			{EmployeeID: 3, ProjectID: 10}: "VFICcccc3333",
		},
	}

	rows, skipped, txnCodes, err := e.buildRowsWithSwift(context.Background(), data, "weekly")
	if err != nil {
		t.Fatalf("buildRowsWithSwift: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if len(skipped) != 2 {
		t.Fatalf("expected 2 skipped, got %d", len(skipped))
	}
	if len(txnCodes) != 1 {
		t.Fatalf("expected 1 txn code, got %d", len(txnCodes))
	}

	reasons := map[uint]string{}
	for _, s := range skipped {
		reasons[s.EmployeeID] = s.Reason
	}
	if reasons[2] != skipReasonMissingBankInfo {
		t.Errorf("emp 2 reason: got %q, want %q", reasons[2], skipReasonMissingBankInfo)
	}
	if reasons[3] != skipReasonUnresolvedSwift {
		t.Errorf("emp 3 reason: got %q, want %q", reasons[3], skipReasonUnresolvedSwift)
	}
}

// TestBuildRowsWithSwift_BankRepoError: when FindByBankCode errors, the
// employee is skipped (not failed) — surface in warning list, admin re-exports.
func TestBuildRowsWithSwift_BankRepoError(t *testing.T) {
	bankRepo := &stubBankRepo{err: errors.New("db down")}
	txnRepo := &stubTransactionCodeRepo{}
	e := NewOnePayExporter(nil, bankRepo, txnRepo, nil)

	data := &excel.BulkTransferData{
		EmployeeProjectAmounts: map[excel.EmployeeProjectKey]int64{
			{EmployeeID: 1, ProjectID: 10}: 1_500_000,
		},
		EmployeeData: map[uint]excel.Employee{
			1: {ID: 1, Fullname: "A", BankAccountNumber: "1", BankAccountName: "A", Bank: &excel.Bank{ID: 1, BankCode: "MB"}},
		},
		ProjectData:     map[uint]excel.Project{10: {ID: 10, Name: "P"}},
		TransactionCodes: map[excel.EmployeeProjectKey]string{{EmployeeID: 1, ProjectID: 10}: "VFICxxx"},
	}
	rows, skipped, _, err := e.buildRowsWithSwift(context.Background(), data, "weekly")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows on bank lookup failure, got %d", len(rows))
	}
	if len(skipped) != 1 || skipped[0].Reason != skipReasonUnresolvedSwift {
		t.Errorf("expected 1 skip with unresolved_swift, got %+v", skipped)
	}
}

// TestGenerateOnePayExcel_LayoutAndColumns: verify the workbook layout
// (header row 2, data row 3, 7 columns, SWIFT at column E).
func TestGenerateOnePayExcel_LayoutAndColumns(t *testing.T) {
	rows := []OnePayExportRow{
		{OrderNo: 1, AccountNo: "99990001", AccountName: "NGUYEN TEST A", Bank: "Quân đội (MB)", SwiftCode: "MBBEVNVX", Amount: 1_500_000, PaymentDetail: "VFIC3ba3ec31LUONGT1", VFICCode: "VFIC3ba3ec31"},
		{OrderNo: 2, AccountNo: "99990002", AccountName: "NGUYEN TEST B", Bank: "Vietcombank", SwiftCode: "BFTVVNVX", Amount: 2_500_000, PaymentDetail: "VFIC4cd5ef62LUONGT2", VFICCode: "VFIC4cd5ef62"},
	}
	b, err := GenerateOnePayExcel(rows)
	if err != nil {
		t.Fatalf("GenerateOnePayExcel: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(b), excelize.Options{UnzipSizeLimit: 10 << 20})
	if err != nil {
		t.Fatalf("open generated xlsx: %v", err)
	}
	defer f.Close()

	if idx, _ := f.GetSheetIndex("eMB_BulkPayment"); idx < 0 {
		t.Fatalf("sheet eMB_BulkPayment not found")
	}

	// Header row 2 — verify the SWIFT header at column E.
	swiftHeader, err := f.GetCellValue("eMB_BulkPayment", "E2")
	if err != nil {
		t.Fatalf("get E2: %v", err)
	}
	if !strings.Contains(swiftHeader, "SWIFT") {
		t.Errorf("E2 header: got %q, want it to contain 'SWIFT'", swiftHeader)
	}

	// Data row 3.
	if swiftVal, _ := f.GetCellValue("eMB_BulkPayment", "E3"); swiftVal != "MBBEVNVX" {
		t.Errorf("E3 swift: got %q, want MBBEVNVX", swiftVal)
	}
	if amountVal, _ := f.GetCellValue("eMB_BulkPayment", "F3"); amountVal != "1500000" {
		t.Errorf("F3 amount: got %q, want 1500000", amountVal)
	}
	// Data row 4.
	if swiftVal2, _ := f.GetCellValue("eMB_BulkPayment", "E4"); swiftVal2 != "BFTVVNVX" {
		t.Errorf("E4 swift: got %q, want BFTVVNVX", swiftVal2)
	}
}

func TestBuildOnePayFilename(t *testing.T) {
	ts := time.Date(2026, 7, 19, 14, 30, 45, 0, time.UTC)
	got := buildOnePayFilename("weekly", ts)
	want := "Yeu_cau_chuyen_tien_weekly_20260719_143045.xlsx"
	if got != want {
		t.Errorf("filename: got %q, want %q", got, want)
	}
}
