package workers

import "testing"

func TestTransferTimesheetUpdateIdempotencyKeyIncludesCurrentRows(t *testing.T) {
	first := transferTimesheetUpdateIdempotencyKey("VFICb372866e", []uint{55653, 55654})
	reordered := transferTimesheetUpdateIdempotencyKey("VFICb372866e", []uint{55654, 55653})
	replaced := transferTimesheetUpdateIdempotencyKey("VFICb372866e", []uint{55655, 55656})

	if first != reordered {
		t.Fatalf("same target rows must have the same key: %q != %q", first, reordered)
	}
	if first == replaced {
		t.Fatalf("replaced target rows must have a new key: %q", first)
	}
}
