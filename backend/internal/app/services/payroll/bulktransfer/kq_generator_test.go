package bulktransfer

import (
	"bytes"
	"context"
	"mime/multipart"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/excelkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubListingConverter serves a fixed outbound listing to the generator.
type stubListingConverter struct {
	rows [][]string
}

func (s *stubListingConverter) ProcessExcelFile(*multipart.FileHeader) (*ExcelFile, error) {
	return &ExcelFile{FileName: "chuyenlo.xlsx", Data: stubExcelData{}}, nil
}

func (s *stubListingConverter) GetSheetNames(*ExcelFile) []string { return []string{"eMB_BulkPayment"} }

func (s *stubListingConverter) GetRowsFromSheet(*ExcelFile, string) ([][]string, error) {
	return s.rows, nil
}

type stubExcelData struct{}

func (stubExcelData) Close() error { return nil }

// mbankChuyenLoRows is the shape of a real chuyenlo.xlsx: a title row, the
// bilingual header row, then one transfer line.
func mbankChuyenLoRows() [][]string {
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
		{"2", "80001708644", "DOAN THI HONG", "Hàng hải (MSB)", "1,500,000", "VFICa8ab43d8"},
	}
}

func fixedClock() func() time.Time {
	t := time.Date(2026, time.October, 2, 15, 4, 5, 0, time.UTC)
	return func() time.Time { return t }
}

func generateKQ(t *testing.T, rows [][]string, filename string) *KQResultFile {
	t.Helper()
	gen := NewKQResultGenerator(&stubListingConverter{rows: rows}, fixedClock())
	result, err := gen.GenerateFromTransferListing(
		context.Background(),
		&multipart.FileHeader{Filename: filename},
	)
	require.NoError(t, err)
	return result
}

// readGeneratedRows reopens the generated workbook the way the result
// importer does, so the test exercises the real round trip.
func readGeneratedRows(t *testing.T, data []byte) [][]string {
	t.Helper()
	f, err := excelkit.OpenReader(bytesReader(data))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	rows, err := f.GetRows(excelkit.KQSheetName)
	require.NoError(t, err)
	return rows
}

// TestGenerateFromTransferListing_ReimportsCleanly is the core guarantee:
// a KQ file generated from a chuyenlo file must come back through the result
// importer with every line completed and its transaction code intact — that
// is what lets the admin upload it again like a file from the bank.
func TestGenerateFromTransferListing_ReimportsCleanly(t *testing.T) {
	result := generateKQ(t, mbankChuyenLoRows(), "chuyenlo.xlsx")

	assert.Equal(t, 2, result.TotalCount)
	assert.Equal(t, int64(7_583_325+1_500_000), result.TotalAmount)
	assert.Equal(t, "KQ_CK_20261002_150405.xlsx", result.Filename)

	factory := NewResultStrategyFactory(NewRowParser())
	strategy, err := factory.DetectStrategy(readGeneratedRows(t, result.ExcelBytes))
	require.NoError(t, err)

	// The KQ layout must resolve to the flat transaction-code strategy, not
	// the MBank statement strategy — a misdetection would read banner cells
	// as transaction codes.
	assert.Equal(t, "TransactionCodeStrategy", strategy.Name())

	parsed, err := strategy.ParseRows(context.Background(), readGeneratedRows(t, result.ExcelBytes))
	require.NoError(t, err)
	require.Len(t, parsed, 2)

	assert.Equal(t, "0989505854", parsed[0].AccountNumber)
	assert.Equal(t, "PHẠM DOÃN DŨNG", parsed[0].AccountName)
	assert.Equal(t, "Quân đội (MB)", parsed[0].BankName)
	assert.Equal(t, float64(7_583_325), parsed[0].Amount)
	assert.Equal(t, "VFIC931c9934", parsed[0].TransactionCode)
	assert.Equal(t, ResultStatusCompleted, parsed[0].Status)

	// Thousands separators in the source listing must not lose the amount.
	assert.Equal(t, float64(1_500_000), parsed[1].Amount)
	assert.Equal(t, "VFICa8ab43d8", parsed[1].TransactionCode)
	assert.Equal(t, ResultStatusCompleted, parsed[1].Status)
}

// The importer finds the batch by hashing every transaction code in the file.
// Codes dropped or reordered during conversion would silently match nothing,
// so the generated file must carry exactly the source's code set.
func TestGenerateFromTransferListing_PreservesTransactionCodes(t *testing.T) {
	result := generateKQ(t, mbankChuyenLoRows(), "chuyenlo.xlsx")

	source := NewChecksumCalculator()
	sourceCodes := []string{"VFIC931c9934", "VFICa8ab43d8"}
	sourceSum, err := source.CalculateFromTransactionCodes(sourceCodes)
	require.NoError(t, err)

	strategy := NewTransactionCodeStrategy()
	generatedSum, err := strategy.GetLookupKey(context.Background(), readGeneratedRows(t, result.ExcelBytes))
	require.NoError(t, err)

	assert.Equal(t, sourceSum, generatedSum)
}

// A listing pads rows with totals and blanks. Those must not become KQ lines,
// because an empty transaction code would fail the importer's batch lookup.
func TestGenerateFromTransferListing_SkipsPaddedRows(t *testing.T) {
	rows := mbankChuyenLoRows()
	rows = append(rows,
		[]string{"Tổng", "", "", "", "9083325", ""},
		[]string{},
		nil,
	)

	result := generateKQ(t, rows, "chuyenlo.xlsx")

	assert.Equal(t, 2, result.TotalCount)
	assert.Equal(t, int64(7_583_325+1_500_000), result.TotalAmount)
}

// The generator is only useful when its output re-imports, so a file with no
// transferable line must fail loudly instead of emitting an empty workbook.
func TestGenerateFromTransferListing_RejectsEmptyListing(t *testing.T) {
	gen := NewKQResultGenerator(&stubListingConverter{rows: mbankChuyenLoRows()[:2]}, fixedClock())

	_, err := gen.GenerateFromTransferListing(
		context.Background(),
		&multipart.FileHeader{Filename: "chuyenlo.xlsx"},
	)

	require.Error(t, err)
	var domainErr *domain.DomainError
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "VALIDATION_ERROR", domainErr.Type)
}

// A workbook that is not an outbound listing (missing the payment-detail
// column) must be rejected as a validation error, not silently converted.
func TestGenerateFromTransferListing_RejectsNonListingWorkbook(t *testing.T) {
	rows := [][]string{
		{"STT", "Số tài khoản", "Tên người thụ hưởng", "Số tiền"},
		{"1", "0989505854", "PHẠM DOÃN DŨNG", "7583325"},
	}

	gen := NewKQResultGenerator(&stubListingConverter{rows: rows}, fixedClock())

	_, err := gen.GenerateFromTransferListing(
		context.Background(),
		&multipart.FileHeader{Filename: "khong-phai-chuyenlo.xlsx"},
	)

	require.Error(t, err)
	var domainErr *domain.DomainError
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "VALIDATION_ERROR", domainErr.Type)
}

// parseTransferListingAmount: bank listings write plain digits, but
// hand-edited files carry separators, a currency suffix, or nothing at all.
func TestParseTransferListingAmount(t *testing.T) {
	cases := map[string]int64{
		"7583325":     7_583_325,
		"7,583,325":   7_583_325,
		"7 583 325":   7_583_325,
		"1.500.000 đ": 1_500_000,
		"930000 VND":  930_000,
		"":            0,
		"abc":         0,
	}
	for input, want := range cases {
		assert.Equal(t, want, parseTransferListingAmount(input), "input %q", input)
	}
}

// bytesReader wraps the generated workbook bytes for excelize.
func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }
