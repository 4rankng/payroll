package bulktransfer

import (
	"fmt"
)

// ResultStrategyFactory creates and detects appropriate result processing strategies
type ResultStrategyFactory struct {
	legacyStrategy          *LegacyResultStrategy
	transactionCodeStrategy *TransactionCodeStrategy
}

// NewResultStrategyFactory creates a new strategy factory
func NewResultStrategyFactory(rowParser *RowParser) *ResultStrategyFactory {
	return &ResultStrategyFactory{
		legacyStrategy:          NewLegacyResultStrategy(rowParser),
		transactionCodeStrategy: NewTransactionCodeStrategy(),
	}
}

// DetectStrategy analyzes the file structure and returns the appropriate strategy
// Only uses transaction_code strategy (H2/H3 legacy format has been deprecated)
func (f *ResultStrategyFactory) DetectStrategy(rows [][]string) (ResultProcessingStrategy, error) {
	if len(rows) < 2 {
		return nil, fmt.Errorf("insufficient rows to detect strategy")
	}

	// Use transaction code strategy as the only supported format
	if f.transactionCodeStrategy.Detect(rows) {
		return f.transactionCodeStrategy, nil
	}

	// Default to transaction code strategy
	return f.transactionCodeStrategy, nil
}
