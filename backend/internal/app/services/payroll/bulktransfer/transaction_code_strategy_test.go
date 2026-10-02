package bulktransfer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// excelize trims trailing empty cells, so a result line whose FT / note
// column (I) is blank comes back with 8 columns instead of 9. Gating parsing
// on len(row) >= 9 dropped such lines silently — the upload then failed with
// "no valid transaction rows found" even though the file was well-formed.
func TestTransactionCodeStrategy_ParsesRowWithBlankTrailingColumns(t *testing.T) {
	rows := [][]string{
		{"Kết quả chuyển tiền - chuyenlo.xlsx"},
		{"Mã lô: WB12"},
		{"Ngày: 02/10/2026 15:04:05"},
		{"STT", "Số tài khoản", "Tên người thụ hưởng", "Ngân hàng thụ hưởng", "Số tiền", "Nội dung", "Phí", "Trạng thái", "FT / Ghi chú"},
		// Column I omitted — no FT number recorded.
		{"1", "0989505854", "PHẠM DOÃN DŨNG", "Quân đội (MB)", "7583325", "VFIC931c9934", "0", "Thành công"},
	}

	parsed, err := NewTransactionCodeStrategy().ParseRows(context.Background(), rows)
	require.NoError(t, err)
	require.Len(t, parsed, 1)

	assert.Equal(t, "VFIC931c9934", parsed[0].TransactionCode)
	assert.Equal(t, ResultStatusCompleted, parsed[0].Status)
	assert.Empty(t, parsed[0].BankTxnRef)
	assert.Empty(t, parsed[0].ErrorMessage)
}

// A genuinely empty row must still be skipped, not turned into a phantom
// line once the length gate is gone.
func TestTransactionCodeStrategy_SkipsEmptyRow(t *testing.T) {
	rows := [][]string{
		{"title"},
		{"ref"},
		{"date"},
		{"STT", "Số tài khoản", "Tên người thụ hưởng", "Ngân hàng thụ hưởng", "Số tiền", "Nội dung", "Phí", "Trạng thái", "FT / Ghi chú"},
		{},
	}

	_, err := NewTransactionCodeStrategy().ParseRows(context.Background(), rows)
	require.Error(t, err)
}
