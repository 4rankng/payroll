package bootstrap

import (
	"api-server/internal/app/services/wallet_bulk"
	asynqinfra "api-server/internal/infra/asynq"
)

// walletBulkAuditAdapter bridges wallet_bulk.AuditEventEmitter to the asynq
// client. We need an adapter because the asynq client's method is named
// EnqueueWalletBulkAudit (to avoid colliding with the existing exported
// EnqueueAuditLogWrite which uses a different payload type).
type walletBulkAuditAdapter struct {
	client *asynqinfra.Client
}

func (a walletBulkAuditAdapter) EnqueueAuditLogWrite(p wallet_bulk.AuditLogPayload) error {
	if a.client == nil {
		return nil
	}
	return a.client.EnqueueWalletBulkAudit(p)
}
