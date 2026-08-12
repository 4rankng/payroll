package wallet_bulk

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
	domaintx "api-server/internal/domain/transactions"

	"github.com/xuri/excelize/v2"
)

// stringPtr helper for test row construction.
func stringPtr(s string) *string { return &s }
func uintPtr(u uint) *uint       { return &u }

// TestKQGenerator_HappyPath verifies the row-5 layout + status mapping + FT column.
func TestKQGenerator_HappyPath(t *testing.T) {
	rows := []BulkTransferRow{
		{OrderNo: 1, AccountNo: "99990001", AccountName: "NGUYEN A", Bank: "Quân đội (MB)", SwiftCode: "MBBEVNVX", Amount: 1_500_000, PaymentDetail: "VFICaaa1LUONG", VFICCode: "VFICaaa1"},
	}
	dataJSON, _ := json.Marshal(rows)
	batch := &domain.BulkTransferBatch{
		ID:       42,
		Filename: "Yeu_cau_chuyen_tien_weekly_20260719.xlsx",
		Data:     string(dataJSON),
	}

	// One completed row with a real FT number (IPN delivered).
	wpRows := []*domaintx.WalletPayment{
		{
			ID: 1, RequestID: "VFICaaa1", InvoiceNo: stringPtr("FT26199097048888"),
			RecipientName: "NGUYEN A", RecipientAccountNo: "99990001", RecipientBank: "MBBEVNVX",
			RequestedAmount: 1_500_000, Fee: 3850, Status: domaintx.StateCompleted,
			BulkTransferOrder: uintPtr(1),
		},
	}

	gen := NewKQExcelGenerator(func() time.Time { return time.Date(2026, 7, 19, 14, 30, 0, 0, time.UTC) })
	bytes, err := gen.Generate(context.Background(), batch, wpRows)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	f, err := excelize.OpenReader(strings.NewReader(string(bytes)), excelize.Options{UnzipSizeLimit: 10 << 20})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	// Row 5 column H = "Thành công"
	if v, _ := f.GetCellValue("data", "H5"); v != "Thành công" {
		t.Errorf("H5 status: got %q, want 'Thành công'", v)
	}
	// Row 5 column I = FT number
	if v, _ := f.GetCellValue("data", "I5"); v != "FT26199097048888" {
		t.Errorf("I5 FT: got %q, want FT26199097048888", v)
	}
	// Row 5 column D = Vietnamese display name from batch.data
	if v, _ := f.GetCellValue("data", "D5"); v != "Quân đội (MB)" {
		t.Errorf("D5 bank: got %q, want 'Quân đội (MB)'", v)
	}
	// Row 5 column G = fee
	if v, _ := f.GetCellValue("data", "G5"); v != "3850" {
		t.Errorf("G5 fee: got %q, want 3850", v)
	}
}

// TestKQGenerator_FTPendingFallback verifies "Đang chờ FT" when invoice_no still starts with VFIC.
func TestKQGenerator_FTPendingFallback(t *testing.T) {
	rows := []BulkTransferRow{
		{OrderNo: 1, Bank: "MB", VFICCode: "VFICaaa1"},
	}
	dataJSON, _ := json.Marshal(rows)
	batch := &domain.BulkTransferBatch{ID: 1, Data: string(dataJSON)}
	wpRows := []*domaintx.WalletPayment{
		{
			ID: 1, RequestID: "VFICaaa1", InvoiceNo: stringPtr("VFICaaa1"), // still VFIC
			Status: domaintx.StateCompleted, BulkTransferOrder: uintPtr(1), Fee: 3850,
		},
	}

	gen := NewKQExcelGenerator(time.Now)
	bytes, err := gen.Generate(context.Background(), batch, wpRows)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	f, err := excelize.OpenReader(strings.NewReader(string(bytes)), excelize.Options{UnzipSizeLimit: 10 << 20})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	if v, _ := f.GetCellValue("data", "I5"); v != "Đang chờ FT" {
		t.Errorf("I5: got %q, want 'Đang chờ FT'", v)
	}
	if v, _ := f.GetCellValue("data", "H5"); v != "Thành công" {
		t.Errorf("H5: got %q, want 'Thành công'", v)
	}
}

// TestKQGenerator_FailedRow verifies error_message in column I for failed rows.
func TestKQGenerator_FailedRow(t *testing.T) {
	rows := []BulkTransferRow{
		{OrderNo: 1, Bank: "MB", VFICCode: "VFICaaa1"},
	}
	dataJSON, _ := json.Marshal(rows)
	batch := &domain.BulkTransferBatch{ID: 1, Data: string(dataJSON)}
	wpRows := []*domaintx.WalletPayment{
		{
			ID: 1, RequestID: "VFICaaa1", Status: domaintx.StateFailed,
			ErrorMessage: stringPtr("Tài khoản không hợp lệ"), BulkTransferOrder: uintPtr(1), Fee: 3850,
		},
	}

	gen := NewKQExcelGenerator(time.Now)
	bytes, _ := gen.Generate(context.Background(), batch, wpRows)
	f, _ := excelize.OpenReader(strings.NewReader(string(bytes)), excelize.Options{UnzipSizeLimit: 10 << 20})
	defer func() { _ = f.Close() }()

	if v, _ := f.GetCellValue("data", "H5"); v != "Thất bại" {
		t.Errorf("H5: got %q, want 'Thất bại'", v)
	}
	if v, _ := f.GetCellValue("data", "I5"); v != "Tài khoản không hợp lệ" {
		t.Errorf("I5: got %q, want error message", v)
	}
}

func TestStatusToVietnamese(t *testing.T) {
	cases := []struct {
		s    domaintx.State
		want string
	}{
		{domaintx.StatePending, "Đang xử lý"},
		{domaintx.StateVerified, "Đang xử lý"},
		{domaintx.StateAuthorised, "Đang xử lý"},
		{domaintx.StateCompleted, "Thành công"},
		{domaintx.StateFailed, "Thất bại"},
		{domaintx.StateReversed, "Đã hoàn"},
	}
	for _, c := range cases {
		if got := statusToVietnamese(c.s); got != c.want {
			t.Errorf("statusToVietnamese(%q): got %q, want %q", c.s, got, c.want)
		}
	}
}
