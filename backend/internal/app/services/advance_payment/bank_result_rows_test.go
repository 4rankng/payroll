package advance_payment

import "testing"

func mbankStatementRows() [][]string {
	rows := make([][]string, 0, 15)
	for range 11 {
		rows = append(rows, make([]string, 25))
	}
	rows[1][4] = "THÔNG TIN CHI TIẾT GIAO DỊCH LÔ"
	rows[7][0] = "Số Tài khoản(A/C No.)"
	rows[7][10] = "Tổng số tiền"

	header := make([]string, 25)
	header[0] = " STT"
	header[2] = " SỐ TÀI KHOẢN\n (Account No)"
	header[5] = " TÊN ĐƠN VỊ THỤ HƯỞNG\n (Beneficiary Name)"
	header[6] = " NGÂN HÀNG THỤ HƯỞNG\n (Beneficiary Bank)"
	header[7] = " SỐ TIỀN GIAO DỊCH\n (Amount)"
	header[9] = " CHI TIẾT THANH TOÁN\n (Payment Details)"
	header[16] = " TRẠNG THÁI GIAO DỊCH\n (Transaction status)"
	header[23] = " BÚT TOÁN\n (FT number)"
	rows = append(rows, header)

	mk := func(stt, code, status, ref string) []string {
		row := make([]string, 25)
		row[0] = stt
		row[2] = "80001708644"
		row[5] = "NGUYEN VAN A"
		row[9] = code
		row[16] = status
		row[23] = ref
		return row
	}
	rows = append(rows,
		mk("1", "VFICaa000001", "Thành công", "FT26253098070986"),
		mk("2", "VFICbb000002", "Thất bại", "Lỗi: sai thông tin"),
		mk("", "", "", ""), // padded empty row
	)
	return rows
}

func TestExtractBankResultRowsMBankStatement(t *testing.T) {
	rows := extractBankResultRows(mbankStatementRows())
	if len(rows) != 2 {
		t.Fatalf("extracted %d rows, want 2 (banner + empty rows skipped)", len(rows))
	}

	first := rows[0]
	if first.TxCode != "VFICaa000001" || first.Row != 13 {
		t.Errorf("first row = %+v, want code VFICaa000001 at sheet row 13", first)
	}
	if first.Status != "Thành công" {
		t.Errorf("success status should pass through as written, got %q", first.Status)
	}
	if first.PaymentRef != "FT26253098070986" {
		t.Errorf("payment ref = %q, want FT26253098070986", first.PaymentRef)
	}

	second := rows[1]
	if second.Status != "THẤT BẠI" {
		t.Errorf("statement casing \"Thất bại\" must normalize to THẤT BẠI, got %q", second.Status)
	}
}

func TestNormalizeStatementStatusFailClosed(t *testing.T) {
	cases := map[string]string{
		"Thất bại":   "THẤT BẠI",
		"FAILED":     "THẤT BẠI",
		"Đang xử lý": "THẤT BẠI", // unrecognized wording must never complete a payment
		"Chờ xử lý":  "THẤT BẠI",
		" từ chối ":  "THẤT BẠI",
		"":           "THẤT BẠI", // blank status cell
		"Thành công": "Thành công",
		" Success ":  "Success",
	}
	for in, want := range cases {
		if got := normalizeStatementStatus(in); got != want {
			t.Errorf("normalizeStatementStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExtractBankResultRowsLegacyFlat(t *testing.T) {
	// Legacy flat export: data from row 3, F=code, G=status, I=ref.
	rowsIn := [][]string{
		{"Kết quả chuyển tiền"},
		{"STT", "Số TK", "Tên", "Ngân hàng", "Số tiền", "Mã GD", "Trạng thái", "", "Tham chiếu"},
		{"1", "80001708644", "NGUYEN VAN A", "MSB", "930000", "VFICaa000001", "FAILED", "", "FT26253098070986"},
		{"2", "80001708645", "NGUYEN VAN B", "MSB", "940000", "VFICbb000002", "SUCCESS", "", "FT26253435344053"},
		{"3", "", "", "", "", "", "", "", ""}, // no code → skipped
	}

	rows := extractBankResultRows(rowsIn)
	if len(rows) != 2 {
		t.Fatalf("extracted %d rows, want 2", len(rows))
	}
	if rows[0].TxCode != "VFICaa000001" || rows[0].Status != "FAILED" || rows[0].PaymentRef != "FT26253098070986" {
		t.Errorf("legacy row 0 = %+v", rows[0])
	}
	if rows[0].Row != 3 {
		t.Errorf("legacy row number = %d, want 3", rows[0].Row)
	}
	if rows[1].Status != "SUCCESS" {
		t.Errorf("legacy statuses must pass through untouched, got %q", rows[1].Status)
	}
}
