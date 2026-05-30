package settlement

import (
	"api-server/internal/app/services/settlement"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// Handler exposes settlement endpoints that operate on notification/email-history records.
type Handler struct {
	settlementUploadSvc *settlement.SettlementUploadService
	notificationRepo    domain.NotificationRepository
	clock               clock.Clock
}

// NewHandler constructs a new settlement Handler.
func NewHandler(
	settlementUploadSvc *settlement.SettlementUploadService,
	notificationRepo domain.NotificationRepository,
	clk clock.Clock,
) *Handler {
	if clk == nil {
		clk = clock.New()
	}
	return &Handler{
		settlementUploadSvc: settlementUploadSvc,
		notificationRepo:    notificationRepo,
		clock:               clk,
	}
}
