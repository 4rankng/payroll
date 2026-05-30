package persistence

import (
	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
	query_builders "api-server/internal/infra/persistence/query_builders"
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
