package advance_payment

import (
	"fmt"
	"strings"

	"api-server/internal/app/services/advance_payment"
	"api-server/internal/app/services/flex_pay"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/app/services/notification"
	"api-server/internal/domain"
	"api-server/internal/domain/services"
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

// formatCurrencyVN formats a number with Vietnamese currency format (dots as thousands separator)
func formatCurrencyVN(amount int64) string {
	s := fmt.Sprintf("%d", amount)
	negative := false
	if len(s) > 0 && s[0] == '-' {
		negative = true
		s = s[1:]
	}
	var result strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteString(".")
		}
		result.WriteRune(r)
	}
	value := result.String()
	if negative {
		value = "-" + value
	}
	return value
}
