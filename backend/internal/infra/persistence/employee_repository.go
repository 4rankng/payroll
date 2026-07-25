package persistence

import (
	"context"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
	query_builders "api-server/internal/infra/persistence/query_builders"

	"gorm.io/gorm"
)

type EmployeeRepository struct {
	*BaseRepository
	queryBuilder        *query_builders.EmployeeQueryBuilder
	statisticsBuilder   *query_builders.EmployeeStatisticsBuilder
	projectQueryBuilder *query_builders.EmployeeProjectQueryBuilder
	errorHandler        *common.RepoErrorHandler
	batchProcessor      *common.BatchProcessor
}

func NewEmployeeRepository(db *Database) domain.EmployeeRepository {
	return &EmployeeRepository{
		BaseRepository:      NewBaseRepository(db),
		queryBuilder:        query_builders.NewEmployeeQueryBuilder(db.DB),
		statisticsBuilder:   query_builders.NewEmployeeStatisticsBuilder(db.DB),
		projectQueryBuilder: query_builders.NewEmployeeProjectQueryBuilder(db.DB),
		errorHandler:        common.NewRepoErrorHandler(),
		batchProcessor:      common.NewBatchProcessor(common.DefaultBatchConfig()),
	}
}

func (r *EmployeeRepository) dbForContext(ctx context.Context) *gorm.DB {
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.DB.WithContext(ctx)
}
