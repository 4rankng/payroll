package bulktransfer

import (
	"fmt"
)

// ResultStrategyFactory creates and detects appropriate result processing strategies
type ResultStrategyFactory struct {
	legacyStrategy          *LegacyResultStrategy
	transactionCodeStrategy *TransactionCodeStrategy
	mbankStatementStrategy  *MBankStatementStrategy
}

// NewResultStrategyFactory creates a new strategy factory
func NewResultStrategyFactory(rowParser *RowParser) *ResultStrategyFactory {
	return &ResultStrategyFactory{
		legacyStrategy:          NewLegacyResultStrategy(rowParser),
		transactionCodeStrategy: NewTransactionCodeStrategy(),
		mbankStatementStrategy:  NewMBankStatementStrategy(),
	}
}

// DetectStrategy analyzes the file structure and returns the appropriate strategy
// H2/H3 legacy format has been deprecated; transaction_code remains the default
func (f *ResultStrategyFactory) DetectStrategy(rows [][]string) (ResultProcessingStrategy, error) {
	if len(rows) < 2 {
		return nil, fmt.Errorf("insufficient rows to detect strategy")
	}

	// MBank "Kết quả giao dịch" statement must be checked before the flat
	// transaction_code format, whose positional parsing would otherwise
	// read banner cells as transaction codes.
	if f.mbankStatementStrategy.Detect(rows) {
		return f.mbankStatementStrategy, nil
	}

	// Use transaction code strategy as the only supported format
	if f.transactionCodeStrategy.Detect(rows) {
		return f.transactionCodeStrategy, nil
	}

	// Default to transaction code strategy
	return f.transactionCodeStrategy, nil
}
