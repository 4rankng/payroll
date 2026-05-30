package bulktransfer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"api-server/internal/app/dto"
)

// ChecksumCalculator handles checksum calculation for bulk transfer files
type ChecksumCalculator struct{}

// NewChecksumCalculator creates a new ChecksumCalculator instance
func NewChecksumCalculator() *ChecksumCalculator {
	return &ChecksumCalculator{}
}

// CalculateFromData calculates checksum from BulkTransferFileData array
// Extracts all transaction_codes, sorts them ascending, concatenates, and returns SHA256 hash
func (c *ChecksumCalculator) CalculateFromData(dataArray []dto.BulkTransferFileData) (string, error) {
	if len(dataArray) == 0 {
		return "", fmt.Errorf("empty data array")
	}

	// Extract transaction codes
	transactionCodes := make([]string, 0, len(dataArray))
	for _, data := range dataArray {
		if data.TransactionCode == "" {
			return "", fmt.Errorf("missing transaction_code in data")
		}
		transactionCodes = append(transactionCodes, data.TransactionCode)
	}

	// Sort ascending
	sort.Strings(transactionCodes)

	// Concatenate
	concatenated := strings.Join(transactionCodes, "")

	// Calculate SHA256 hash
	hash := sha256.Sum256([]byte(concatenated))
	return hex.EncodeToString(hash[:]), nil
}

// CalculateFromJSON calculates checksum from JSON string
func (c *ChecksumCalculator) CalculateFromJSON(dataJSON string) (string, error) {
	dataArray, err := dto.ParseBulkTransferFileData(dataJSON)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	return c.CalculateFromData(dataArray)
}

// CalculateFromTransactionCodes calculates checksum from transaction codes directly
func (c *ChecksumCalculator) CalculateFromTransactionCodes(transactionCodes []string) (string, error) {
	if len(transactionCodes) == 0 {
		return "", fmt.Errorf("empty transaction codes")
	}

	// Sort ascending
	sorted := make([]string, len(transactionCodes))
	copy(sorted, transactionCodes)
	sort.Strings(sorted)

	// Concatenate
	concatenated := strings.Join(sorted, "")

	// Calculate SHA256 hash
	hash := sha256.Sum256([]byte(concatenated))
	return hex.EncodeToString(hash[:]), nil
}
