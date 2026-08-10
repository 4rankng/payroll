package services

import (
	"context"
	"testing"

	"api-server/internal/domain"
)

func TestTimesheetReplacementDeletesOnlySuppressMatchingDuplicate(t *testing.T) {
	existing := &domain.Timesheet{ID: 42}

	if !isDuplicateTimesheetConflict(context.Background(), existing, 0) {
		t.Fatal("ordinary duplicate must remain a conflict")
	}

	replacementCtx := WithTimesheetReplacementDeletes(context.Background(), []uint{42})
	// The explicit filter remains safe even for a pooled read context.
	readCtx := context.WithValue(replacementCtx, domain.TransactionContextKey{}, nil)
	if isDuplicateTimesheetConflict(readCtx, existing, 0) {
		t.Fatal("row deleted by the active replacement must not block its replacement")
	}
	visible := filterTimesheetReplacementDeletes(readCtx, []*domain.Timesheet{existing, {ID: 99}})
	if len(visible) != 1 || visible[0].ID != 99 {
		t.Fatalf("validation reads must exclude only replacement-deleted rows, got %#v", visible)
	}

	otherReplacementCtx := WithTimesheetReplacementDeletes(context.Background(), []uint{99})
	if !isDuplicateTimesheetConflict(otherReplacementCtx, existing, 0) {
		t.Fatal("unrelated replacement rows must not weaken duplicate protection")
	}
}
