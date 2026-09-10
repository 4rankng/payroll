package bulktransfer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"api-server/internal/pkg/excelkit"
)

// mbankStatementRows builds an in-memory replica of the MBank "Kết quả giao
// dịch" bulk-transfer statement (banner block, column header, data rows) in
// the shape excelize GetRows returns.
func mbankStatementRows() [][]string {
	rows := make([][]string, 0, 16)
	for range 11 {
		rows = append(rows, make([]string, 25))
	}
	rows[1][4] = "THÔNG TIN CHI TIẾT GIAO DỊCH LÔ"
	rows[6][0] = "Đơn vị/Người chuyển"
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
	header[17] = " TRẠNG THÁI HẠCH TOÁN\n (Account deduction status)"
	header[23] = " BÚT TOÁN\n (FT number)"
	rows = append(rows, header)

	mk := func(stt, account, name, bank, amount, code, status, ref string) []string {
		row := make([]string, 25)
		row[0] = stt
		row[2] = account
		row[5] = name
		row[6] = bank
		row[7] = amount
		row[9] = code
		row[16] = status
		row[23] = ref
		return row
	}
	rows = append(rows,
		mk("1", "80001708644", "NGUYEN VAN A", "NGÂN HÀNG TMCP HÀNG HẢI VIỆT NAM", "930000.0", "VFICaa000001", "Thành công", "FT26253098070986"),
		mk("2", "19023465392011", "TRAN THI B", "NGÂN HÀNG TMCP KỸ THƯƠNG VIỆT NAM", "1050000.0", "VFICbb000002", "Thất bại", "Lỗi: sai thông tin thụ hưởng"),
		mk("3", "296608866", "LE VAN C", "NGÂN HÀNG TMCP QUÂN ĐỘI", "2214000.0", "VFICcc000003", "Thành công", "FT26253435344053"),
	)
	return rows
}

func TestMBankStatementStrategyParsesNewTemplate(t *testing.T) {
	strategy := NewMBankStatementStrategy()
	rows := mbankStatementRows()

	parsed, err := strategy.ParseRows(t.Context(), rows)
	if err != nil {
		t.Fatalf("ParseRows failed: %v", err)
	}
	if len(parsed) != 3 {
		t.Fatalf("parsed %d rows, want 3", len(parsed))
	}

	first := parsed[0]
	if first.TransactionCode != "VFICaa000001" {
		t.Errorf("transaction code = %q, want VFICaa000001", first.TransactionCode)
	}
	if first.AccountNumber != "80001708644" || first.AccountName != "NGUYEN VAN A" {
		t.Errorf("account fields mismatch: %+v", first)
	}
	if first.Amount != 930000 {
		t.Errorf("amount = %v, want 930000", first.Amount)
	}
	if first.Status != ResultStatusCompleted {
		t.Errorf("status = %q, want completed", first.Status)
	}
	if first.BankTxnRef != "FT26253098070986" || first.ErrorMessage != "" {
		t.Errorf("completed row should carry the bank ref, got ref=%q err=%q", first.BankTxnRef, first.ErrorMessage)
	}

	second := parsed[1]
	if second.Status != ResultStatusFailed {
		t.Errorf("status = %q, want failed", second.Status)
	}
	if second.ErrorMessage == "" || second.BankTxnRef != "" {
		t.Errorf("failed row should carry the error message, got ref=%q err=%q", second.BankTxnRef, second.ErrorMessage)
	}
}

func TestMBankStatementStrategyGetLookupKey(t *testing.T) {
	strategy := NewMBankStatementStrategy()
	rows := mbankStatementRows()

	key, err := strategy.GetLookupKey(t.Context(), rows)
	if err != nil {
		t.Fatalf("GetLookupKey failed: %v", err)
	}

	want, err := NewChecksumCalculator().CalculateFromTransactionCodes([]string{"VFICaa000001", "VFICbb000002", "VFICcc000003"})
	if err != nil {
		t.Fatalf("reference checksum failed: %v", err)
	}
	if key != want {
		t.Errorf("lookup key = %q, want %q", key, want)
	}
}

func TestResultStrategyFactoryRoutesNewAndOldTemplates(t *testing.T) {
	factory := NewResultStrategyFactory(NewRowParser())

	newStrategy, err := factory.DetectStrategy(mbankStatementRows())
	if err != nil {
		t.Fatalf("DetectStrategy failed: %v", err)
	}
	if _, ok := newStrategy.(*MBankStatementStrategy); !ok {
		t.Fatalf("new template routed to %s, want MBankStatementStrategy", newStrategy.Name())
	}

	// Old flat "bảng kê" layout: data from row 5, F=code, H=status, I=ref.
	oldRows := [][]string{
		{"Bảng kê chuyển tiền"},
		{""},
		{""},
		{"STT", "Số TK", "Tên", "Ngân hàng", "Số tiền", "Mã GD", "", "Trạng thái", "Tham chiếu"},
		{"1", "80001708644", "NGUYEN VAN A", "MSB", "930000", "VFICaa000001", "", "Thành công", "FT26253098070986"},
	}
	oldStrategy, err := factory.DetectStrategy(oldRows)
	if err != nil {
		t.Fatalf("DetectStrategy failed: %v", err)
	}
	if _, ok := oldStrategy.(*TransactionCodeStrategy); !ok {
		t.Fatalf("old template routed to %s, want TransactionCodeStrategy", oldStrategy.Name())
	}

	// And the old strategy still parses it.
	parsed, err := oldStrategy.ParseRows(t.Context(), oldRows)
	if err != nil {
		t.Fatalf("legacy parse failed: %v", err)
	}
	if len(parsed) != 1 || parsed[0].TransactionCode != "VFICaa000001" || parsed[0].Status != ResultStatusCompleted {
		t.Fatalf("legacy parse mismatch: %+v", parsed)
	}
}

// TestMBankStatementStrategyRealFile runs the strategy against real bank
// statement exports. Point MBANK_RESULT_REAL_FILE at the file (or a
// directory of .xlsx files) to run; real beneficiary data is never
// committed to the repo.
func TestMBankStatementStrategyRealFile(t *testing.T) {
	path := os.Getenv("MBANK_RESULT_REAL_FILE")
	if path == "" {
		t.Skip("MBANK_RESULT_REAL_FILE not set")
	}

	var files []string
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.IsDir() {
		matches, err := filepath.Glob(filepath.Join(path, "*.xlsx"))
		if err != nil {
			t.Fatalf("glob %s: %v", path, err)
		}
		for _, m := range matches {
			// Skip Excel owner/lock temp files ("~$book.xlsx")
			if !strings.HasPrefix(filepath.Base(m), "~$") {
				files = append(files, m)
			}
		}
	} else {
		files = []string{path}
	}
	if len(files) == 0 {
		t.Fatalf("no .xlsx files under %s", path)
	}

	strategy := NewMBankStatementStrategy()
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			f, err := excelkit.OpenFile(file)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			defer func() { _ = f.Close() }()

			sheets := f.GetSheetList()
			rows, err := f.GetRows(sheets[0])
			if err != nil {
				t.Fatalf("read rows: %v", err)
			}

			if !strategy.Detect(rows) {
				t.Fatalf("statement header not detected in %s", file)
			}
			parsed, err := strategy.ParseRows(t.Context(), rows)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			completed, failed := 0, 0
			for _, r := range parsed {
				if !strings.HasPrefix(r.TransactionCode, "VFIC") {
					t.Errorf("row %d: transaction code %q lacks VFIC prefix", r.STT, r.TransactionCode)
				}
				if r.Status == ResultStatusCompleted {
					completed++
				} else {
					failed++
				}
			}
			t.Logf("parsed %d rows: %d completed, %d failed", len(parsed), completed, failed)
		})
	}
}
