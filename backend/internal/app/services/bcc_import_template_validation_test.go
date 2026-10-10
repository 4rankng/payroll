package services

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// bccFixtureBytes reads a committed real-partner fixture workbook, skipping the
// test when the fixture is absent (fresh clone before fixture commit).
func bccFixtureBytes(t *testing.T, rel ...string) []byte {
	t.Helper()
	path := filepath.Join(append([]string{"..", "..", "..", "tests", "fixtures", "bcc"}, rel...)...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture unavailable: %v", err)
	}
	return data
}

// workbookBytes builds an in-memory .xlsx from rows, so the rejection path can be
// exercised without a fixture.
func workbookBytes(t *testing.T, sheet string, rows [][]any) []byte {
	t.Helper()
	f := excelize.NewFile()
	t.Cleanup(func() { _ = f.Close() })
	index, err := f.NewSheet(sheet)
	require.NoError(t, err)
	f.SetActiveSheet(index)
	require.NoError(t, f.DeleteSheet("Sheet1"))
	for i, row := range rows {
		for j, value := range row {
			cell, err := excelize.CoordinatesToCellName(j+1, i+1)
			require.NoError(t, err)
			require.NoError(t, f.SetCellValue(sheet, cell, value))
		}
	}
	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))
	return buf.Bytes()
}

// Every supported template family must pass the upload pre-flight, otherwise a
// working partner file would be rejected at the door.
func TestValidateBCCWorkbookTemplate_AcceptsSupportedTemplates(t *testing.T) {
	fixtures := [][]string{
		{"legacy", "eva_t08.xlsx"},
		{"multi_position", "pqc_haiphong.xlsx"},
		{"weekly_bcc", "lgd_weekly.xlsx"},
		{"weekly_payment", "tbd_weekly_payment.xlsx"},
		{"date_row", "bumhan_t08.xlsx"},
		{"eva06.xlsx"},
		{"thai_binh_duong.xlsx"},
	}
	for _, parts := range fixtures {
		t.Run(filepath.Join(parts...), func(t *testing.T) {
			assert.NoError(t, validateBCCWorkbookTemplate(bccFixtureBytes(t, parts...)))
		})
	}
}

// An unrelated spreadsheet (or a non-spreadsheet file) must be rejected with the
// uploader-facing message, and as a validation error so the handler answers with
// a 4xx instead of a 500.
func TestValidateBCCWorkbookTemplate_RejectsUnsupportedWorkbook(t *testing.T) {
	cases := map[string][]byte{
		"random sheet":   workbookBytes(t, "Sổ theo dõi xe", [][]any{{"STT", "Biển số"}, {1, "29A-12345"}}),
		"not a workbook": []byte("this is not an xlsx file at all"),
		"empty workbook": workbookBytes(t, "Sheet1", nil),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateBCCWorkbookTemplate(data)
			require.Error(t, err)
			assert.True(t, domain.IsValidationError(err), "want validation error, got %T: %v", err, err)
			assert.Equal(t, bccTemplateMismatchMessage, err.Error())
			assert.Contains(t, err.Error(), "Mẫu file chấm công không đúng")
		})
	}
}

// The rejection must happen before storage or the job queue is touched —
// proven with a zero-valued service whose repositories are nil, which would
// panic if any later step ran. This is the contract the upload endpoint's 4xx
// depends on: no asset, no background job, one clear message.
func TestAcceptUpload_RejectsUnsupportedWorkbookBeforeSideEffects(t *testing.T) {
	_, err := (&BCCImportService{}).AcceptUpload(
		context.Background(),
		bytes.NewReader(workbookBytes(t, "Sổ theo dõi xe", [][]any{{"STT", "Biển số"}, {1, "29A-12345"}})),
		"so-theo-doi.xlsx", 1, 1, "admin", "2026-10", false, "key-1",
	)
	require.Error(t, err)
	assert.True(t, domain.IsValidationError(err), "want validation error, got %T: %v", err, err)
	assert.Equal(t, bccTemplateMismatchMessage, err.Error())
	assert.Contains(t, err.Error(), "Mẫu file chấm công không đúng")
}

// The sheet NAME alone is enough for the legacy family, so a "BCC" sheet with
// unrelated columns passes the pre-flight; its structure is reported by the
// worker (which parses for real) rather than guessed at here.
func TestValidateBCCWorkbookTemplate_AcceptsOnSheetNameAlone(t *testing.T) {
	data := workbookBytes(t, "BCC", [][]any{{"a", "b", "c"}, {"x", "y", "z"}})
	assert.NoError(t, validateBCCWorkbookTemplate(data))
}
