package persistence

import (
	"sync"

	"api-server/internal/domain"
	query_builders "api-server/internal/infra/persistence/query_builders"
)

// LedgerEntryRepository implements domain.LedgerEntryRepository.
// Logic is split across:
//   - ledger_repository_crud.go      — write operations (Create, Delete, duplicates)
//   - ledger_repository_queries.go   — read operations (Get*, List, Count, Search)
//   - ledger_repository_balance.go   — balance calculations and recalculation
//   - ledger_repository_analytics.go — reporting and chart data
type LedgerEntryRepository struct {
	*BaseRepository
	balanceMutex sync.RWMutex // protects concurrent balance calculations
	queryBuilder *query_builders.LedgerQueryBuilder
}

// accountAgg is a shared GORM scan target for per-account debit/credit aggregations.
type accountAgg struct {
	Account     string `gorm:"column:account"`
	TotalDebit  int64  `gorm:"column:total_debit"`
	TotalCredit int64  `gorm:"column:total_credit"`
}

// NewLedgerEntryRepository creates a new LedgerEntryRepository.
func NewLedgerEntryRepository(db *Database) domain.LedgerEntryRepository {
	return &LedgerEntryRepository{
		BaseRepository: NewBaseRepository(db),
		queryBuilder:   query_builders.NewLedgerQueryBuilder(db.DB),
	}
}
