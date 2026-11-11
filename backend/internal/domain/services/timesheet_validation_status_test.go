package services

import (
	"context"
	"testing"

	"api-server/internal/domain"
)

// The add-on-top supplement flow must pass the duplicate check against the
// exact row it deliberately sits next to, while every other existing row at
// the same key and replacement-deleted rows keep their current behavior.
func TestIsDuplicateTimesheetConflictSupplementCoexist(t *testing.T) {
	existing := &domain.Timesheet{ID: 93036}
	ctx := context.Background()

	if !isDuplicateTimesheetConflict(ctx, existing, 0) {
		t.Fatal("existing row must be a duplicate conflict without any marker")
	}

	coexistCtx := WithTimesheetSupplementCoexist(ctx, []uint{93036})
	if isDuplicateTimesheetConflict(coexistCtx, existing, 0) {
		t.Fatal("marked row must be allowed as the supplement's coexist target")
	}

	other := &domain.Timesheet{ID: 93037}
	if !isDuplicateTimesheetConflict(coexistCtx, other, 0) {
		t.Fatal("unmarked rows must stay duplicate conflicts under the coexist marker")
	}

	deleteCtx := WithTimesheetReplacementDeletes(ctx, []uint{93036})
	if isDuplicateTimesheetConflict(deleteCtx, existing, 0) {
		t.Fatal("replacement-deleted rows must stay allowed")
	}
}
