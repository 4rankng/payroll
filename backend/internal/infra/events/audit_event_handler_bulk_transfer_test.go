package events

import (
	"context"
	"encoding/json"
	"testing"

	"api-server/internal/domain"
	asynqinfra "api-server/internal/infra/asynq"
	auditctx "api-server/internal/pkg/context"
)

// TestAuditHandler_BulkTransferFileExported_PersistsRow verifies that the
// AuditEventHandler enqueues an audit payload for BulkTransferFileExported
// with the right action, entity type, actor, and metadata fields.
func TestAuditHandler_BulkTransferFileExported_PersistsRow(t *testing.T) {
	enqueuer := &mockAuditEnqueuer{}
	handler := NewAuditEventHandler(enqueuer)

	ctx := auditctx.WithUserID(context.Background(), 42)
	ctx = auditctx.WithFullName(ctx, "Test Admin")

	event := domain.NewBulkTransferFileExportedEvent(
		ctx, 42,
		"MBANK_weekly_abcd1234",
		"weekly",
		"2026-04-28", "2026-05-04", "",
		17, 1_234_500,
		true,
		nil,
	)

	if err := handler.Handle(ctx, event); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(enqueuer.payloads) != 1 {
		t.Fatalf("expected 1 enqueued payload, got %d", len(enqueuer.payloads))
	}
	p := enqueuer.payloads[0]
	if p.Action != string(domain.AuditActionExport) {
		t.Fatalf("Action: got %s, want EXPORT", p.Action)
	}
	if p.EntityType != string(domain.EntityTypeBulkTransferFile) {
		t.Fatalf("EntityType: got %s, want bulk_transfer_file", p.EntityType)
	}
	if p.UserID != 42 {
		t.Fatalf("UserID: got %d, want 42", p.UserID)
	}
	if p.Message == "" {
		t.Fatalf("audit message empty")
	}

	// Metadata payload must include the export-specific fields admins query on.
	if p.MetadataJSON == "" {
		t.Fatalf("metadata not set")
	}
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(p.MetadataJSON), &meta); err != nil {
		t.Fatalf("metadata unmarshal: %v", err)
	}
	for _, key := range []string{"filename", "cycle", "transactions_count", "total_amount"} {
		if _, ok := meta[key]; !ok {
			t.Fatalf("metadata missing %q: %+v", key, meta)
		}
	}
}

// TestAuditHandler_BulkTransferResultImported_PersistsRow does the same for
// the IMPORT side.
func TestAuditHandler_BulkTransferResultImported_PersistsRow(t *testing.T) {
	enqueuer := &mockAuditEnqueuer{}
	handler := NewAuditEventHandler(enqueuer)

	ctx := auditctx.WithUserID(context.Background(), 7)
	ctx = auditctx.WithFullName(ctx, "Test Admin")

	event := domain.NewBulkTransferResultImportedEvent(
		ctx, 7, 99, "result.xlsx", 20, 18, 2, 9_000_000, false,
	)

	if err := handler.Handle(ctx, event); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(enqueuer.payloads) != 1 {
		t.Fatalf("expected 1 enqueued payload, got %d", len(enqueuer.payloads))
	}
	p := enqueuer.payloads[0]
	if p.Action != string(domain.AuditActionImport) {
		t.Fatalf("Action: got %s, want IMPORT", p.Action)
	}
	if p.EntityType != string(domain.EntityTypeBulkTransferFile) {
		t.Fatalf("EntityType: got %s, want bulk_transfer_file", p.EntityType)
	}
	if p.EntityID == nil || *p.EntityID != 99 {
		t.Fatalf("EntityID: want pointer to 99, got %v", p.EntityID)
	}

	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(p.MetadataJSON), &meta); err != nil {
		t.Fatalf("metadata unmarshal: %v", err)
	}
	for _, key := range []string{"asset_id", "filename", "completed", "failed", "total_amount", "is_duplicate"} {
		if _, ok := meta[key]; !ok {
			t.Fatalf("metadata missing %q: %+v", key, meta)
		}
	}
}

// TestAuditHandler_BulkTransferFileDownloaded_PersistsRow covers the P2
// re-download read-audit.
func TestAuditHandler_BulkTransferFileDownloaded_PersistsRow(t *testing.T) {
	enqueuer := &mockAuditEnqueuer{}
	handler := NewAuditEventHandler(enqueuer)

	ctx := auditctx.WithFullName(auditctx.WithUserID(context.Background(), 1), "Frank")
	event := domain.NewBulkTransferFileDownloadedEvent(ctx, 1, 50, 99, "x.xlsx")

	if err := handler.Handle(ctx, event); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(enqueuer.payloads) != 1 {
		t.Fatalf("expected 1 enqueued payload, got %d", len(enqueuer.payloads))
	}
	if enqueuer.payloads[0].Action != string(domain.AuditActionView) {
		t.Fatalf("expected VIEW action, got %s", enqueuer.payloads[0].Action)
	}
}

// Ensure mockAuditEnqueuer implements asynqinfra.AuditEnqueuer at compile time.
var _ asynqinfra.AuditEnqueuer = (*mockAuditEnqueuer)(nil)
