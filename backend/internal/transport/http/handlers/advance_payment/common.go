package advance_payment

import (
	"api-server/internal/app/services/advance_payment"
	"api-server/internal/app/services/flex_pay"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/app/services/notification"
	"api-server/internal/domain"
	"api-server/internal/domain/services"
	"api-server/internal/domain/wallet"
	asynqinfra "api-server/internal/infra/asynq"
	"api-server/internal/infra/storage"
	"api-server/internal/pkg/clock"
)

// AdvancePaymentHandler handles HTTP requests for advance payments
type AdvancePaymentHandler struct {
	service                       *advance_payment.Service
	assetRepo                     domain.AssetRepository
	asynqClient                   *asynqinfra.Client
	importProgressSvc             *advance_payment.ImportProgressService
	fileStorage                   storage.FileStorage
	flexPayReconciliationService  *services.FlexPayReconciliationService
	flexPayReconciliationExporter *flex_pay.FlexPayReconciliationExporter
	flexPaySettlementService      *flex_pay.FlexPaySettlementService
	emailService                  *notification.EmailService
	auditService                  *infrastructure.AuditService
	notificationRepo              domain.NotificationRepository
	walletPaymentRepo             wallet.WalletPaymentRepository
	clock                         clock.Clock
}

// NewAdvancePaymentHandler creates a new advance payment handler
func NewAdvancePaymentHandler(
	service *advance_payment.Service,
	assetRepo domain.AssetRepository,
	asynqClient *asynqinfra.Client,
	importProgressSvc *advance_payment.ImportProgressService,
	fileStorage storage.FileStorage,
	flexPayReconciliationService *services.FlexPayReconciliationService,
	flexPayReconciliationExporter *flex_pay.FlexPayReconciliationExporter,
	flexPaySettlementService *flex_pay.FlexPaySettlementService,
	emailService *notification.EmailService,
	auditService *infrastructure.AuditService,
	notificationRepo domain.NotificationRepository,
	walletPaymentRepo wallet.WalletPaymentRepository,
	clk clock.Clock,
) *AdvancePaymentHandler {
	if clk == nil {
		clk = clock.New()
	}
	return &AdvancePaymentHandler{
		service:                       service,
		assetRepo:                     assetRepo,
		asynqClient:                   asynqClient,
		importProgressSvc:             importProgressSvc,
		fileStorage:                   fileStorage,
		flexPayReconciliationService:  flexPayReconciliationService,
		flexPayReconciliationExporter: flexPayReconciliationExporter,
		flexPaySettlementService:      flexPaySettlementService,
		emailService:                  emailService,
		auditService:                  auditService,
		notificationRepo:              notificationRepo,
		walletPaymentRepo:             walletPaymentRepo,
		clock:                         clk,
	}
}

func calculatePercentage(totalRows, processedRows int) int {
	if totalRows == 0 {
		return 0
	}
	percentage := float64(processedRows) / float64(totalRows) * 100
	return int(percentage)
}
