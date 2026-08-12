package workers

import (
	"context"

	"api-server/internal/app/services/zaloconnect"
)

// FlexPaySalaryNotificationWorker is the Asynq boundary for durable salary ZNS.
type FlexPaySalaryNotificationWorker struct {
	service *zaloconnect.SalaryNotificationDeliveryService
}

func NewFlexPaySalaryNotificationWorker(service *zaloconnect.SalaryNotificationDeliveryService) *FlexPaySalaryNotificationWorker {
	return &FlexPaySalaryNotificationWorker{service: service}
}

func (w *FlexPaySalaryNotificationWorker) ProcessJob(ctx context.Context, notificationID uint) error {
	if w == nil || w.service == nil {
		return nil
	}
	return w.service.Deliver(ctx, notificationID)
}

func (w *FlexPaySalaryNotificationWorker) Recover(ctx context.Context) error {
	if w == nil || w.service == nil {
		return nil
	}
	return w.service.Recover(ctx)
}
