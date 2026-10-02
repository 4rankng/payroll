package excelkit

import "testing"

// mbankListingRows replicates the 6-column outbound chuyenlo listing the
// MBank manual template produces: title row, header row, one data row.
func mbankListingRows() [][]string {
	return [][]string{
		{"", "DANH SÁCH GIAO DỊCH\n(LIST OF TRANSACTIONS)", "", "", "", ""},
		{
			"STT\n(Ord. No.)\n(1)",
			"Số tài khoản\n(Account No.)\n(2)",
			"Tên người thụ hưởng\n(Beneficiary)\n(3)",
			"Ngân hàng thụ hưởng/Chi nhánh\n(Beneficiary Bank)\n(4)",
			"Số tiền\n(Amount)\n(5)",
			"Nội dung chuyển khoản\n(Payment Detail)\n(6)",
		},
		{"1", "0989505854", "PHẠM DOÃN DŨNG", "Quân đội (MB)", "7583325", "VFIC931c9934"},
	}
}

// onepayListingRows replicates the 7-column OnePay eMB_BulkPayment listing,
// which inserts a SWIFT code column at E and shifts amount/payment detail
// right. The locator must resolve it without any positional knowledge.
func onepayListingRows() [][]string {
	return [][]string{
		{
			"STT\n(Ord. No.)",
			"Số tài khoản\n(Account No.)",
			"Tên người thụ hưởng\n(Beneficiary)",
			"Ngân hàng thụ hưởng/Chi nhánh\n(Beneficiary Bank)",
			"Mã SWIFT\n(SWIFT Code)",
			"Số tiền\n(Amount)",
			"Nội dung chuyển khoản\n(Payment Detail)",
		},
		{"1", "80001708644", "DOAN THI HONG", "NGÂN HÀNG TMCP HÀNG HẢI", "MBBEVNVX", "930000", "VFICa8ab43d8"},
	}
}

func TestLocateTransferListingHeaderFindsMBankShape(t *testing.T) {
	idx, cols, ok := LocateTransferListingHeader(mbankListingRows())
	if !ok {
		t.Fatal("expected the MBank listing header to be located")
	}
	if idx != 1 {
		t.Fatalf("header row index = %d, want 1", idx)
	}
	want := TransferListingCols{
		Order:          0,
		AccountNumber:  1,
		AccountName:    2,
		BankName:       3,
		Amount:         4,
		TransactionRef: 5,
	}
	if cols != want {
		t.Fatalf("column map = %+v, want %+v", cols, want)
	}
}

func TestLocateTransferListingHeaderFindsOnePayShape(t *testing.T) {
	_, cols, ok := LocateTransferListingHeader(onepayListingRows())
	if !ok {
		t.Fatal("expected the OnePay listing header to be located")
	}
	want := TransferListingCols{
		Order:          0,
		AccountNumber:  1,
		AccountName:    2,
		BankName:       3,
		Amount:         5,
		TransactionRef: 6,
	}
	if cols != want {
		t.Fatalf("column map = %+v, want %+v", cols, want)
	}
}

// The locator must reject a listing that drops the payment-detail column —
// without a transaction code the rows cannot be re-matched to a batch.
func TestLocateTransferListingHeaderRequiresTransactionRef(t *testing.T) {
	rows := mbankListingRows()
	rows[1][5] = "Ghi chú\n(Note)"

	if _, _, ok := LocateTransferListingHeader(rows); ok {
		t.Fatal("listing without a payment-detail column must not be detected")
	}
}

// A "Tổng số tiền" summary banner row carries the phrase "số tiền"; reading
// it as the amount header would silently read the grand total as the column.
func TestLocateTransferListingHeaderRejectsTotalRowAsAmount(t *testing.T) {
	rows := [][]string{
		{"Tổng số tiền\n(Total amount of transaction)", "399206690.0"},
		{
			"STT",
			"Số tài khoản",
			"Tên người thụ hưởng",
			"Số tiền",
			"Nội dung chuyển khoản",
		},
	}

	idx, cols, ok := LocateTransferListingHeader(rows)
	if !ok {
		t.Fatal("expected the real header row to be located")
	}
	if idx != 1 {
		t.Fatalf("header row index = %d, want 1 (total banner must not win)", idx)
	}
	if cols.Amount != 3 {
		t.Fatalf("amount column = %d, want 3", cols.Amount)
	}
}

// A result workbook (the KQ sheet) is not an outbound listing: it must never
// be mistaken for one, or a re-upload could be fed back through the converter.
func TestLocateTransferListingHeaderRejectsKQResult(t *testing.T) {
	rows := [][]string{
		{"STT", "Số tài khoản", "Tên người thụ hưởng", "Ngân hàng thụ hưởng", "Số tiền", "Nội dung", "Phí", "Trạng thái", "FT / Ghi chú"},
	}

	if _, _, ok := LocateTransferListingHeader(rows); ok {
		t.Fatal("KQ result layout must not be detected as an outbound listing")
	}
}
