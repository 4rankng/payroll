package bulktransfer

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"api-server/internal/app/dto"
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
		{OrderNo: 1, AccountNo: "99990001", AccountName: "NGUYEN TEST A", Bank: "Quân đội (MB)", SwiftCode: "MBBEVNVX", Amount: 1_500_000, PaymentDetail: "VFIC3ba3ec31", VFICCode: "VFIC3ba3ec31"},
		{OrderNo: 2, AccountNo: "99990002", AccountName: "NGUYEN TEST B", Bank: "Vietcombank", SwiftCode: "BFTVVNVX", Amount: 2_500_000, PaymentDetail: "VFIC4cd5ef62", VFICCode: "VFIC4cd5ef62"},
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

// TestBuildRowsWithSwift_PaymentDetailIsBareVFIC guards against the
// INVALID_PARAMETERS bug: OnePay rejects remarks containing hyphens
// (sandbox/onepay/handler.go:hasHyphen), and a verbose PaymentDetail
// like "CT VFICxxx Lâm Văn Bách Hilex - KCN nomura" picks up the hyphen
// from a project name and fails the transfer. PaymentDetail MUST be the
// bare VFIC code only — reconciliation keys off the VFIC code anyway.
//
// This test deliberately uses a hyphenated project name ("Hilex - KCN nomura")
// so any future regression that re-introduces the verbose format fails here.
func TestBuildRowsWithSwift_PaymentDetailIsBareVFIC(t *testing.T) {
	bankRepo := &stubBankRepo{byCode: map[string]*domain.Bank{
		"VCB": {ID: 2, BankCode: "VCB", SwiftCode: "BFTVVNVX", BranchName: "Vietcombank"},
	}}
	txnRepo := &stubTransactionCodeRepo{}
	e := NewOnePayExporter(nil, bankRepo, txnRepo, nil)

	const vfic = "VFICe1f37125"
	data := &excel.BulkTransferData{
		EmployeeProjectAmounts: map[excel.EmployeeProjectKey]int64{
			{EmployeeID: 1, ProjectID: 10}: 1_192_500,
		},
		EmployeeProjectTimesheets: map[excel.EmployeeProjectKey][]uint{
			{EmployeeID: 1, ProjectID: 10}: {101},
		},
		EmployeeData: map[uint]excel.Employee{
			1: {ID: 1, Fullname: "Lâm Văn Bách", BankAccountNumber: "1043541997", BankAccountName: "Lâm Văn Bách", Bank: &excel.Bank{ID: 2, BankCode: "VCB", BranchName: "Vietcombank"}},
		},
		// Hyphenated project name — this is what triggered the original bug.
		ProjectData: map[uint]excel.Project{10: {ID: 10, Name: "Hilex - KCN nomura"}},
		TransactionCodes: map[excel.EmployeeProjectKey]string{
			{EmployeeID: 1, ProjectID: 10}: vfic,
		},
	}

	rows, _, _, err := e.buildRowsWithSwift(context.Background(), data, "weekly")
	if err != nil {
		t.Fatalf("buildRowsWithSwift: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	if rows[0].PaymentDetail != vfic {
		t.Errorf("PaymentDetail: got %q, want bare VFIC %q (no hyphens, no name/project suffix)",
			rows[0].PaymentDetail, vfic)
	}
	if rows[0].VFICCode != vfic {
		t.Errorf("VFICCode: got %q, want %q", rows[0].VFICCode, vfic)
	}
}

// TestCycleLabelVN covers the Vietnamese cycle label used in the empty-pool
// EMPTY_RESULT message.
func TestCycleLabelVN(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{string(domain.PaymentScheduleWeekly), "tuần"},
		{string(domain.PaymentScheduleMonthly), "tháng"},
		{"", "đã chọn"},
		{"unknown", "đã chọn"},
	}
	for _, tc := range tests {
		if got := cycleLabelVN(tc.in); got != tc.want {
			t.Errorf("cycleLabelVN(%q): got %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestBuildEmptySwiftMessage_BucketsByReason verifies the EMPTY_RESULT
// message combines the three skip counts: pre-validation missing
// account/name, post-validation missing bank linkage, and unresolved SWIFT.
func TestBuildEmptySwiftMessage_BucketsByReason(t *testing.T) {
	plan := &ExportPlan{
		ValidatedData: &excel.BulkTransferValidationResult{
			SkippedCount: 3, // pre-validation: missing account/name
		},
	}
	skipped := []OnePaySkippedEmployee{
		{EmployeeID: 1, Reason: skipReasonMissingBankInfo},
		{EmployeeID: 2, Reason: skipReasonMissingBankInfo},
		{EmployeeID: 3, Reason: skipReasonUnresolvedSwift},
		{EmployeeID: 4, Reason: skipReasonUnresolvedSwift},
		{EmployeeID: 5, Reason: skipReasonUnresolvedSwift},
	}
	msg := buildEmptySwiftMessage(plan, skipped)

	// 2 missing bank info, 3 unresolved SWIFT, 3 pre-validation.
	if !strings.Contains(msg, "2 nhân viên thiếu thông tin ngân hàng") {
		t.Errorf("missing bank info count: msg=%q", msg)
	}
	if !strings.Contains(msg, "3 nhân viên chưa có mã SWIFT") {
		t.Errorf("unresolved SWIFT count: msg=%q", msg)
	}
	if !strings.Contains(msg, "3 nhân viên bị bỏ qua trước đó") {
		t.Errorf("pre-validation count: msg=%q", msg)
	}
}

// TestBuildEmptySwiftMessage_NilPlan guards against a nil plan (defensive —
// Export never passes nil but the helper must not panic if it ever does).
func TestBuildEmptySwiftMessage_NilPlan(t *testing.T) {
	msg := buildEmptySwiftMessage(nil, nil)
	if !strings.Contains(msg, "0 nhân viên") {
		t.Errorf("expected zeroed counts, got %q", msg)
	}
}

// TestExport_EmptyPool_ReturnsEmptyResultError exercises the full Export
// pipeline with a stub planner whose pool contains zero eligible timesheets.
// Asserts the returned error is the typed EMPTY_RESULT domain error so the
// handler maps it to HTTP 422 (not 500).
//
// Note: even with an empty pool, the planner returns a non-nil (but empty)
// ValidatedData, so the code path that fires is the post-SWIFT-resolution
// zero-rows branch. Either branch produces EMPTY_RESULT — the test only
// asserts the error type and that the message is the actionable VN form.
func TestExport_EmptyPool_ReturnsEmptyResultError(t *testing.T) {
	// Empty stub bundle — no timesheets, no employees, no projects.
	bundle := &stubRepoBundle{
		employees:        map[uint]*domain.Employee{},
		projects:         map[uint]*domain.Project{},
		assignments:      map[excel.EmployeeProjectKey]*domain.ProjectEmployee{},
		timesheetByID:    map[uint]*domain.Timesheet{},
		weeklyPercentage: 1.0,
	}
	planner := newPlannerWithStub(bundle)

	bankRepo := &stubBankRepo{byCode: map[string]*domain.Bank{}}
	txnRepo := &stubTransactionCodeRepo{}
	exporter := NewOnePayExporter(planner, bankRepo, txnRepo, nil)

	_, err := exporter.Export(context.Background(), &dto.ExportBulkTransferRequest{
		FromDate: "2026-07-08",
		ToDate:   "2026-07-14",
	})

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !domain.IsEmptyResultError(err) {
		t.Fatalf("expected EMPTY_RESULT domain error, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "Không có") {
		t.Errorf("expected actionable VN message, got: %v", err)
	}
}

// TestExport_AllSkipped_ReturnsEmptyResultError exercises the path where the
// planner produces validated data, but every candidate is skipped during
// SWIFT resolution (missing bank info / unresolved SWIFT). The returned
// error must be EMPTY_RESULT and carry the SWIFT-resolution message.
func TestExport_AllSkipped_ReturnsEmptyResultError(t *testing.T) {
	bundle := seedStub() // 1 valid employee (emp 7), 1 missing account (emp 9)
	planner := newPlannerWithStub(bundle)

	// bankRepo returns nil for the only bank code "MB" → unresolved_swift for emp 7.
	bankRepo := &stubBankRepo{byCode: map[string]*domain.Bank{}}
	txnRepo := &stubTransactionCodeRepo{}
	exporter := NewOnePayExporter(planner, bankRepo, txnRepo, nil)

	_, err := exporter.Export(context.Background(), &dto.ExportBulkTransferRequest{
		FromDate: "2026-07-08",
		ToDate:   "2026-07-14",
	})

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !domain.IsEmptyResultError(err) {
		t.Fatalf("expected EMPTY_RESULT domain error, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "sau khi đối chiếu SWIFT") {
		t.Errorf("expected SWIFT resolution message, got: %v", err)
	}
	// txnRepo must NOT have been called — we surface EMPTY_RESULT before persisting.
	if len(txnRepo.created) != 0 {
		t.Errorf("CreateBatch must not run on empty export; got %d codes", len(txnRepo.created))
	}
}
