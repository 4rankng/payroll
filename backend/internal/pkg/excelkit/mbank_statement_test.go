package excelkit

import (
	"testing"

	"golang.org/x/text/unicode/norm"
)

// statementHeaderRow replicates the column-header row of the MBank
// "Kết quả giao dịch" bulk-transfer statement (merged cells appear as the
// leading cell followed by empty continuations, exactly like excelize
// GetRows output).
func statementHeaderRow() []string {
	row := make([]string, 25)
	row[0] = " STT"
	row[2] = " SỐ TÀI KHOẢN\n (Account No)"
	row[5] = " TÊN ĐƠN VỊ THỤ HƯỞNG\n (Beneficiary Name)"
	row[6] = " NGÂN HÀNG THỤ HƯỞNG\n (Beneficiary Bank)"
	row[7] = " SỐ TIỀN GIAO DỊCH\n (Amount)"
	row[9] = " CHI TIẾT THANH TOÁN\n (Payment Details)"
	row[12] = " PHÍ (CHƯA GỒM VAT)\n (Charge not VAT)"
	row[15] = " HÌNH THỨC CHUYỂN KHOẢN\n (Payment Method)"
	row[16] = " TRẠNG THÁI GIAO DỊCH\n (Transaction status)"
	row[17] = " TRẠNG THÁI HẠCH TOÁN\n (Account deduction status)"
	row[18] = " GHI CHÚ\n (Note)"
	row[23] = " BÚT TOÁN\n (FT number)"
	return row
}

// statementBannerRows replicates the banner block above the header, including
// the applicant-info rows that reuse phrases like "Số Tài khoản" and
// "Tổng số tiền" — the locator must not mistake them for the header.
func statementBannerRows() [][]string {
	rows := make([][]string, 11)
	for i := range rows {
		rows[i] = make([]string, 25)
	}
	rows[1][4] = "THÔNG TIN CHI TIẾT GIAO DỊCH LÔ"
	rows[1][14] = "NGÂN HÀNG TMCP QUÂN ĐỘI"
	rows[6][0] = "Đơn vị/Người chuyển\n(Applicant)"
	rows[6][5] = "CTY TNHH MTV GP PHAN MEM TING TING"
	rows[6][11] = "Số tham chiếu (Ref No.)"
	rows[6][15] = "202609102419883866"
	rows[7][0] = "Số Tài khoản(A/C No.)"
	rows[7][5] = "271866699"
	rows[7][6] = "Loại tiền \n(Currency)"
	rows[7][7] = "VND"
	rows[7][10] = "Tổng số tiền \n(Total amount of transaction)"
	rows[7][15] = "399206690.0"
	rows[9][11] = "Số giao dịch trong lô \n(The number of transaction)"
	rows[9][15] = "166"
	rows[10][0] = " BẢNG CHI TIẾT GIAO DỊCH TRONG LÔ"
	return rows
}

func TestLocateMBankStatementHeaderFindsHeaderAndColumns(t *testing.T) {
	rows := statementBannerRows()
	rows = append(rows, statementHeaderRow())
	data := make([]string, 25)
	data[0] = "1"
	data[2] = "80001708644"
	data[5] = "DOAN THI HONG"
	data[6] = "NGÂN HÀNG TMCP HÀNG HẢI VIỆT NAM"
	data[7] = "930000.0"
	data[9] = "VFICa8ab43d8"
	data[12] = "0.0"
	data[15] = "Nhanh 247"
	data[16] = "Thành công"
	data[17] = "Đã trừ tiền"
	data[23] = "FT26253098070986"
	rows = append(rows, data)

	idx, cols, ok := LocateMBankStatementHeader(rows)
	if !ok {
		t.Fatal("expected header row to be located")
	}
	if idx != len(rows)-2 {
		t.Fatalf("header row index = %d, want %d", idx, len(rows)-2)
	}
	want := MBankStatementCols{
		AccountNumber:   2,
		AccountName:     5,
		BankName:        6,
		Amount:          7,
		TransactionCode: 9,
		Status:          16,
		BankRef:         23,
	}
	if cols != want {
		t.Fatalf("column map = %+v, want %+v", cols, want)
	}
}

func TestLocateMBankStatementHeaderMatchesNFDForm(t *testing.T) {
	rows := statementBannerRows()
	header := statementHeaderRow()
	for i, cell := range header {
		header[i] = norm.NFD.String(cell)
	}
	rows = append(rows, header)

	_, _, ok := LocateMBankStatementHeader(rows)
	if !ok {
		t.Fatal("NFD header form should still be located")
	}
}

func TestLocateMBankStatementHeaderRejectsLegacyFlatLayout(t *testing.T) {
	// Legacy "Kết quả chuyển tiền theo bảng kê" export: short headers, data
	// from row 5, transaction code in column F, status in column H.
	rows := [][]string{
		{"Bảng kê chuyển tiền"},
		{""},
		{""},
		{"STT", "Số TK", "Tên khách hàng", "Ngân hàng", "Số tiền", "Mã giao dịch", "", "Trạng thái", "Tham chiếu"},
		{"1", "80001708644", "DOAN THI HONG", "MSB", "930000", "VFICa8ab43d8", "", "Thành công", "FT26253098070986"},
	}

	if _, _, ok := LocateMBankStatementHeader(rows); ok {
		t.Fatal("legacy flat layout must not be detected as MBank statement")
	}
}

func TestLocateMBankStatementHeaderRejectsBannerOnly(t *testing.T) {
	if _, _, ok := LocateMBankStatementHeader(statementBannerRows()); ok {
		t.Fatal("banner rows without the column header must not be detected")
	}
}

func TestLocateMBankStatementHeaderBankRefNotStolenByStatusColumn(t *testing.T) {
	// If the accounting-status column is ever renamed to include "bút toán",
	// it must not claim the FT-reference column slot.
	rows := statementBannerRows()
	header := statementHeaderRow()
	header[17] = " TRẠNG THÁI BÚT TOÁN"
	rows = append(rows, header)

	_, cols, ok := LocateMBankStatementHeader(rows)
	if !ok {
		t.Fatal("header should still be located")
	}
	if cols.BankRef != 23 {
		t.Fatalf("bank ref column = %d, want 23 (the FT number column)", cols.BankRef)
	}
	if cols.Status != 16 {
		t.Fatalf("status column = %d, want 16", cols.Status)
	}
}
