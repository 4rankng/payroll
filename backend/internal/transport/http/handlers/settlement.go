package handlers

import (
	settlementHandler "api-server/internal/transport/http/handlers/settlement"

	"api-server/internal/app/services/settlement"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

type SettlementHandler struct {
	*settlementHandler.Handler
}

func NewSettlementHandler(
	settlementUploadSvc *settlement.SettlementUploadService,
	notificationRepo domain.NotificationRepository,
	clk clock.Clock,
) *SettlementHandler {
	return &SettlementHandler{
		Handler: settlementHandler.NewHandler(settlementUploadSvc, notificationRepo, clk),
	}
}
