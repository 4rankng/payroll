package domain

import (
	"context"
	"testing"
	"time"

	auditctx "api-server/internal/pkg/context"
)

// TestLedgerEntryCreatedEvent_ActorFromEntityWhenCtxEmpty reproduces the
// bulk-transfer audit-log leak: the BulkTransferTransactionWorker creates
// ledger entries from a context.Background() (the event-bus worker strips the
// originating ctx), so the audit emit can no longer pull the actor from ctx.
//
// Before the fix, NewLedgerEntryCreatedEvent used getUserIDFromContext(ctx) as
// the actor and silently produced ActorUserID=0 — which then flowed into
// audit_logs.user_id=0. The fix is to fall back to entry.CreatedBy, which the
// caller (LedgerService.CreateEntries) already populates with the real actor.
func TestLedgerEntryCreatedEvent_ActorFromEntityWhenCtxEmpty(t *testing.T) {
	const uploaderUserID uint = 643

	entry := &LedgerEntry{
		ID:        100,
		Date:      time.Now(),
		Account:   AccountCash,
		Party:     "Test Party",
		Debit:     1_000_000,
		Credit:    0,
		CreatedBy: uploaderUserID,
	}

	event := NewLedgerEntryCreatedEvent(context.Background(), entry)

	if got := event.UserID(); got != uploaderUserID {
		t.Fatalf("LedgerEntryCreated audit actor leaked: got UserID=%d, want %d (entry.CreatedBy)", got, uploaderUserID)
	}
}

// TestLedgerEntryCreatedEvent_ContextWinsOverEntity verifies that when both ctx
// and entry have actors, ctx still wins — the request-time actor is more
// specific than entity.CreatedBy (which may be the original creator on update).
func TestLedgerEntryCreatedEvent_ContextWinsOverEntity(t *testing.T) {
	const ctxUser uint = 11
	const entryCreator uint = 22

	entry := &LedgerEntry{
		ID:        101,
		Date:      time.Now(),
		Account:   AccountCash,
		Party:     "Test Party",
		Debit:     500,
		Credit:    0,
		CreatedBy: entryCreator,
	}

	ctx := auditctx.WithUserID(context.Background(), ctxUser)
	event := NewLedgerEntryCreatedEvent(ctx, entry)

	if got := event.UserID(); got != ctxUser {
		t.Fatalf("ctx user should win: got UserID=%d, want %d", got, ctxUser)
	}
}

// TestTransactionCreatedEventWithAudit_ActorFromTxnWhenCtxEmpty mirrors the
// ledger leak for the TransactionCreated emit on the same bulk-transfer path
// (transaction_service.createTransactionInContext is called with the worker's
// empty ctx).
func TestTransactionCreatedEventWithAudit_ActorFromTxnWhenCtxEmpty(t *testing.T) {
	const uploaderUserID uint = 643

	txn := &Transaction{
		ID:              200,
		Amount:          5_000_000,
		TransactionType: TransactionTypeRevenue,
		Description:     "Bulk transfer revenue",
		CreatedBy:       uploaderUserID,
	}

	event := NewTransactionCreatedEventWithAudit(context.Background(), txn, "test message")

	if got := event.UserID(); got != uploaderUserID {
		t.Fatalf("TransactionCreated audit actor leaked: got UserID=%d, want %d (txn.CreatedBy)", got, uploaderUserID)
	}
}
