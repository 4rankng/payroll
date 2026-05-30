package bulktransfer

import (
	"testing"

	pkgConstants "api-server/internal/pkg/constants"

	"github.com/stretchr/testify/assert"
)

func TestParsedRow_Structure(t *testing.T) {
	row := ParsedRow{
		AccountNumber:  "123456",
		AccountName:    "John Doe",
		Amount:         "1000000",
		Description:    "Salary payment",
		TransferStatus: "Success",
		TrackingData:   "[1][100 101]",
	}

	assert.Equal(t, "123456", row.AccountNumber)
	assert.Equal(t, "John Doe", row.AccountName)
	assert.Equal(t, "1000000", row.Amount)
	assert.Equal(t, "Salary payment", row.Description)
	assert.Equal(t, "Success", row.TransferStatus)
	assert.Equal(t, "[1][100 101]", row.TrackingData)
}

func TestNewMBankParser(t *testing.T) {
	parser := NewMBankParser()
	assert.NotNil(t, parser)
	assert.IsType(t, &MBankParser{}, parser)
}

func TestMBankParser_Detect_WithPrefix(t *testing.T) {
	parser := NewMBankParser()

	rows := [][]string{
		{"header1"},
		{"header2"},
		{"col1", "col2", "col3", "col4", "col5", "col6"},
	}

	result := parser.Detect(pkgConstants.MBankPrefix+"test.xlsx", rows)
	assert.True(t, result)
}

func TestMBankParser_Detect_InsufficientRows(t *testing.T) {
	parser := NewMBankParser()

	rows := [][]string{
		{"header1"},
		{"header2"},
	}

	result := parser.Detect(pkgConstants.MBankPrefix+"test.xlsx", rows)
	assert.False(t, result)
}

func TestMBankParser_Detect_InsufficientColumns(t *testing.T) {
	parser := NewMBankParser()

	rows := [][]string{
		{"header1"},
		{"header2"},
		{"col1", "col2"},
	}

	result := parser.Detect(pkgConstants.MBankPrefix+"test.xlsx", rows)
	assert.False(t, result)
}

func TestMBankParser_Detect_ByStructure(t *testing.T) {
	parser := NewMBankParser()

	rows := [][]string{
		{"header1"},
		{"header2"},
		{"col1", "col2", "col3", "col4", "col5", "col6"},
	}

	result := parser.Detect("noprefix.xlsx", rows)
	assert.True(t, result)
}

func TestMBankParser_StartRow(t *testing.T) {
	parser := NewMBankParser()
	assert.Equal(t, 2, parser.StartRow())
}

func TestMBankParser_ParseRow_FullRow(t *testing.T) {
	parser := NewMBankParser()

	row := []string{
		"stt",
		"123456",
		"John Doe",
		"bank",
		"1000000",
		"Salary",
		"Success",
		"[1][100]",
	}

	result := parser.ParseRow(row)

	assert.Equal(t, "123456", result.AccountNumber)
	assert.Equal(t, "John Doe", result.AccountName)
	assert.Equal(t, "1000000", result.Amount)
	assert.Equal(t, "Salary", result.Description)
	assert.Equal(t, "Success", result.TransferStatus)
	assert.Equal(t, "[1][100]", result.TrackingData)
}

func TestMBankParser_ParseRow_ShortRow(t *testing.T) {
	parser := NewMBankParser()

	row := []string{
		"stt",
		"123456",
		"John",
	}

	result := parser.ParseRow(row)

	assert.Equal(t, "123456", result.AccountNumber)
	assert.Equal(t, "John", result.AccountName)
	assert.Equal(t, "", result.Amount)
	assert.Equal(t, "", result.Description)
}

func TestMBankParser_AccountNumberColumn(t *testing.T) {
	parser := NewMBankParser()
	assert.Equal(t, "B", parser.AccountNumberColumn())
}

func TestMBankParser_ParseRow_WithWhitespace(t *testing.T) {
	parser := NewMBankParser()

	row := []string{
		"stt",
		" 123456 ",
		" John Doe ",
		"bank",
		" 1000000 ",
		" Salary ",
		" Success ",
		" [1][100] ",
	}

	result := parser.ParseRow(row)

	// Verify whitespace is trimmed
	assert.Equal(t, "123456", result.AccountNumber)
	assert.Equal(t, "John Doe", result.AccountName)
	assert.Equal(t, "1000000", result.Amount)
	assert.Equal(t, "Salary", result.Description)
	assert.Equal(t, "Success", result.TransferStatus)
	assert.Equal(t, "[1][100]", result.TrackingData)
}
