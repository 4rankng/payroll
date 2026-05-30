package services

import (
	infraServices "api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	"api-server/internal/infra/events"
	"api-server/internal/infra/persistence"
)

type InfrastructureServices struct {
	CacheService       *infraServices.CacheService
	TransactionManager *infraServices.TransactionManager
	EventBus           domain.EventBus
}

func InitializeInfrastructure(db *persistence.Database, redis *persistence.RedisClient) (*InfrastructureServices, domain.EventBus) {
	cacheService := infraServices.NewCacheService(redis)
	eventBus := events.NewWorkerPoolEventBus(100, 10000)
	transactionManager := infraServices.NewTransactionManager(db.DB)

	return &InfrastructureServices{
		CacheService:       cacheService,
		TransactionManager: transactionManager,
		EventBus:           eventBus,
	}, eventBus
}
