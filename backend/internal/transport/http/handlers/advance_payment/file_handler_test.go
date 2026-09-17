package advance_payment

import (
	"context"
	"encoding/json"
	"testing"

	"api-server/internal/domain"
)

type saoKeNotificationRepo struct {
	domain.NotificationRepository
	created *domain.Notification
}

func (r *saoKeNotificationRepo) Create(_ context.Context, notification *domain.Notification) error {
	r.created = notification
	return nil
}

func TestSaoKeExportNotificationPreservesAuthenticatedSender(t *testing.T) {
	repo := &saoKeNotificationRepo{}
	handler := &AdvancePaymentHandler{notificationRepo: repo}
	// The export's asset stores the authenticated user that uploaded it.
	asset := &domain.Asset{ID: 81, UploadedBy: 99}
	handler.createSaoKeExportNotification(context.Background(), asset.UploadedBy, asset.ID, 250000, "2026-09")

	if repo.created == nil {
		t.Fatal("expected a statement history notification")
	}
	if repo.created.SenderID != asset.UploadedBy {
		t.Fatalf("sender = %d, want authenticated exporter %d", repo.created.SenderID, asset.UploadedBy)
	}
	if repo.created.Type != domain.NotificationTypeAdvancePaymentReport || repo.created.Channel != domain.NotificationChannelEmail {
		t.Fatalf("unexpected notification category: %+v", repo.created)
	}
	if repo.created.Metadata == nil {
		t.Fatal("expected statement asset metadata")
	}
	var metadata domain.PayrollEmailMetadata
	if err := json.Unmarshal([]byte(*repo.created.Metadata), &metadata); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if metadata.SaoKeAssetID == nil || *metadata.SaoKeAssetID != asset.ID || metadata.TotalAmount != 250000 || metadata.ReportAtDate != "2026-09" {
		t.Fatalf("unexpected statement metadata: %+v", metadata)
	}
}
