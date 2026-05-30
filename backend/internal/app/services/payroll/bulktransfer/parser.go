package bulktransfer

import (
	"fmt"
	"strings"

	"api-server/internal/domain"
	pkgConstants "api-server/internal/pkg/constants"
)

// ParsedRow is a value object representing a normalized bank row
type ParsedRow struct {
	AccountNumber  string
	AccountName    string
	Amount         string
	Description    string
	TransferStatus string
	TrackingData   string
}

// BankResultParser defines strategy for bank-specific result parsing
type BankResultParser interface {
	// Detect returns true if this parser can handle given file
	Detect(filename string, rows [][]string) bool
	// StartRow returns zero-based index of first data row
	StartRow() int
	// ParseRow maps a raw row to ParsedRow, applying len-guards
	ParseRow(row []string) ParsedRow
	// ValidateRow validates a parsed row and returns an error if invalid
	ValidateRow(parsed ParsedRow) error
	// AccountNumberColumn returns spreadsheet column letter for account number (for error reporting)
	AccountNumberColumn() string
}

// detectParser selects a parser for filename/rows or returns validation error
func (s *Service) detectParser(filename string, rows [][]string) (BankResultParser, error) {
	for _, p := range s.resultParsers {
		if p.Detect(filename, rows) {
			return p, nil
		}
	}
	return nil, domain.NewValidationError(
		fmt.Sprintf("Invalid file format. Expected %s prefix or valid MBank column structure", pkgConstants.MBankPrefix),
	)
}

// MBankParser implements BankResultParser for MBank exports
type MBankParser struct{}

func NewMBankParser() *MBankParser { return &MBankParser{} }

func (p *MBankParser) Detect(filename string, rows [][]string) bool {
	if strings.HasPrefix(filename, pkgConstants.MBankPrefix) {
		if len(rows) < 3 {
			return false
		}
		// Expect at least 6 columns on the third row (A-F)
		return len(rows[2]) >= 6
	}
	// Fallback by structure (no prefix)
	return len(rows) >= 3 && len(rows[2]) >= 6
}

func (p *MBankParser) StartRow() int { return 2 } // Row 3 (index 2)

func (p *MBankParser) ParseRow(row []string) ParsedRow {
	// Ensure length
	if len(row) < 8 {
		padded := make([]string, 8)
		copy(padded, row)
		row = padded
	}
	return ParsedRow{
		AccountNumber:  strings.TrimSpace(row[1]), // B
		AccountName:    strings.TrimSpace(row[2]), // C
		Amount:         strings.TrimSpace(row[4]), // E
		Description:    strings.TrimSpace(row[5]), // F
		TransferStatus: strings.TrimSpace(row[6]), // G
		TrackingData:   strings.TrimSpace(row[7]), // H
	}
}

func (p *MBankParser) ValidateRow(parsed ParsedRow) error {
	if parsed.AccountNumber == "" {
		return &ErrInsufficientData{Field: "account_number"}
	}
	if parsed.Description == "" {
		return &ErrInsufficientData{Field: "description"}
	}
	return nil
}

func (p *MBankParser) AccountNumberColumn() string { return "B" }
