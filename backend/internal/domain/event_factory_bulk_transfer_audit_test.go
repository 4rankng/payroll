package domain

import (
	"context"
	"strings"
	"testing"

	auditctx "api-server/internal/pkg/context"
)

// TestBulkTransferFileExportedEvent_FieldsAndAudit verifies that
// NewBulkTransferFileExportedEvent populates the actor, action, entity type,
// and produces a non-empty Vietnamese audit message — the audit handler
// silently drops events with an empty message, which is what made the export
// gap invisible before.
func TestBulkTransferFileExportedEvent_FieldsAndAudit(t *testing.T) {
	const actorID uint = 42
	ctx := auditctx.WithUserID(context.Background(), actorID)
	ctx = auditctx.WithFullName(ctx, "Test Admin")

	event := NewBulkTransferFileExportedEvent(
		ctx,
		actorID,
		"MBANK_weekly_abcd1234",
		"weekly",
		"2026-04-28", "2026-05-04", "",
		17,
		1_234_500,
		true,
		nil,
	)

	if event.UserID() != actorID {
		t.Fatalf("UserID: got %d, want %d", event.UserID(), actorID)
	}
	if event.GetAction() != AuditActionExport {
		t.Fatalf("Action: got %s, want EXPORT", event.GetAction())
	}
	if event.GetEntityType() != EntityTypeBulkTransferFile {
		t.Fatalf("EntityType: got %s, want bulk_transfer_file", event.GetEntityType())
	}
	msg := event.GetAuditMessage()
	if msg == "" {
		t.Fatalf("AuditMessage is empty — audit handler will silently skip this event")
	}
	if !strings.Contains(msg, "Test Admin") {
		t.Fatalf("AuditMessage missing actor: %q", msg)
	}
	if !strings.Contains(msg, "1.234.500") {
		t.Fatalf("AuditMessage missing total amount: %q", msg)
	}
	if event.Filename != "MBANK_weekly_abcd1234" || event.Cycle != "weekly" || event.TransactionsCount != 17 || event.TotalAmount != 1_234_500 {
		t.Fatalf("payload fields not preserved: %+v", event)
	}
}

// TestBulkTransferResultImportedEvent_FieldsAndAudit verifies the IMPORT audit
// emit for the result file upload path, including the duplicate flag.
func TestBulkTransferResultImportedEvent_FieldsAndAudit(t *testing.T) {
	const actorID uint = 7
	ctx := auditctx.WithUserID(context.Background(), actorID)
	ctx = auditctx.WithFullName(ctx, "Test Admin")

	event := NewBulkTransferResultImportedEvent(
		ctx, actorID, 99, "result_2026-05-04.xlsx",
		20, 18, 2, 9_000_000, false,
	)

	if event.GetAction() != AuditActionImport {
		t.Fatalf("Action: got %s, want IMPORT", event.GetAction())
	}
	if event.GetEntityType() != EntityTypeBulkTransferFile {
		t.Fatalf("EntityType: got %s, want bulk_transfer_file", event.GetEntityType())
	}
	if event.AggregateID() != 99 {
		t.Fatalf("AggregateID: got %d, want 99 (asset_id)", event.AggregateID())
	}
	// User-facing format: "<actor> nhập file kết quả chuyển lô tạm ứng,
	// tổng giao dịch <total> đ". Filename moves to event payload / audit
	// metadata, not the user-visible message.
	wantMsg := "Test Admin nhập file kết quả chuyển lô tạm ứng, tổng giao dịch 9.000.000 ₫"
	if got := event.GetAuditMessage(); got != wantMsg {
		t.Fatalf("AuditMessage:\n got:  %q\n want: %q", got, wantMsg)
	}
	if event.IsDuplicate {
		t.Fatalf("IsDuplicate should be false for fresh upload")
	}

	dup := NewBulkTransferResultImportedEvent(ctx, actorID, 99, "result.xlsx", 0, 0, 0, 0, true)
	if !dup.IsDuplicate {
		t.Fatalf("IsDuplicate flag not propagated")
	}
}

// TestBulkTransferAuditEvents_ActorFallsBackToParam ensures the actor is
// preserved even when ctx has no user — same robustness pattern as the
// LedgerEntryCreated fix from session 49d16e28.
func TestBulkTransferAuditEvents_ActorFallsBackToParam(t *testing.T) {
	const actorID uint = 555

	exp := NewBulkTransferFileExportedEvent(context.Background(), actorID, "f", "weekly", "", "", "", 0, 0, true, nil)
	if exp.UserID() != actorID {
		t.Fatalf("export actor lost: got %d, want %d", exp.UserID(), actorID)
	}

	imp := NewBulkTransferResultImportedEvent(context.Background(), actorID, 1, "f", 0, 0, 0, 0, false)
	if imp.UserID() != actorID {
		t.Fatalf("import actor lost: got %d, want %d", imp.UserID(), actorID)
	}

	dl := NewBulkTransferFileDownloadedEvent(context.Background(), actorID, 1, 2, "f")
	if dl.UserID() != actorID {
		t.Fatalf("download actor lost: got %d, want %d", dl.UserID(), actorID)
	}

	pe := NewPayrollHistoriesExportedEvent(context.Background(), actorID, "2026-01-01", "2026-01-31", "f.xlsx")
	if pe.UserID() != actorID {
		t.Fatalf("payroll-export actor lost: got %d, want %d", pe.UserID(), actorID)
	}
}
