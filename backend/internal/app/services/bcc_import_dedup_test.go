package services

import (
	"testing"

	domainservices "api-server/internal/domain/services"
)

// entry is a terse constructor for the fields dedupBCCEntries keys on.
func dedupEntry(employee uint, date, hourType, dayType string, hours float64) domainservices.BulkCreateTimesheetEntry {
	return domainservices.BulkCreateTimesheetEntry{
		ProjectID:   54,
		EmployeeID:  employee,
		Date:        date,
		HoursWorked: hours,
		HourType:    hourType,
		DayType:     &dayType,
	}
}

// The EPE weekly template prices Sunday's NN and TCNN columns at the same
// 66,000 VND, and the project payrate has a single leaf at that rate, so both
// cells resolve to ngày nghỉ/ca ngày. Before dedup the second entry came back
// from BulkCreateTimesheets as an in-batch duplicate.
func TestDedupBCCEntriesMergesSameBucketSundayPair(t *testing.T) {
	entries := []domainservices.BulkCreateTimesheetEntry{
		dedupEntry(1210, "2026-09-06", "ca ngày", "ngày nghỉ", 8),
		dedupEntry(1210, "2026-09-06", "ca ngày", "ngày nghỉ", 0),
	}

	got := dedupBCCEntries(entries)

	if len(got) != 1 {
		t.Fatalf("expected the colliding pair to collapse to 1 entry, got %d", len(got))
	}
	if got[0].HoursWorked != 8 {
		t.Errorf("expected net hours 8 (NN=8 + TCNN=0), got %v", got[0].HoursWorked)
	}
}

// Summing, not first-wins: when both columns carry hours they are two tallies
// of work paid at one rate, so the bucket total is their sum. First-wins would
// silently drop the overtime.
func TestDedupBCCEntriesSumsTwoPositiveColumns(t *testing.T) {
	entries := []domainservices.BulkCreateTimesheetEntry{
		dedupEntry(1210, "2026-09-06", "ca ngày", "ngày nghỉ", 8),
		dedupEntry(1210, "2026-09-06", "ca ngày", "ngày nghỉ", 2),
	}

	got := dedupBCCEntries(entries)

	if len(got) != 1 {
		t.Fatalf("expected 1 merged entry, got %d", len(got))
	}
	if got[0].HoursWorked != 10 {
		t.Errorf("expected summed hours 10, got %v", got[0].HoursWorked)
	}
}

// An explicit 0 deletes the matching chờ duyệt row. Two zeros must merge to a
// surviving zero-hour entry, not vanish — dropping it would turn a deletion
// request into a no-op.
func TestDedupBCCEntriesPreservesDeletionIntent(t *testing.T) {
	entries := []domainservices.BulkCreateTimesheetEntry{
		dedupEntry(1210, "2026-09-06", "ca ngày", "ngày nghỉ", 0),
		dedupEntry(1210, "2026-09-06", "ca ngày", "ngày nghỉ", 0),
	}

	got := dedupBCCEntries(entries)

	if len(got) != 1 {
		t.Fatalf("expected the zero pair to collapse to 1 entry, got %d", len(got))
	}
	if got[0].HoursWorked != 0 {
		t.Errorf("expected the deletion request to stay at 0 hours, got %v", got[0].HoursWorked)
	}
}

// Weekday HC/TCN price differently (33,000 vs 49,500), resolve to separate
// buckets, and must both survive.
func TestDedupBCCEntriesKeepsDistinctBuckets(t *testing.T) {
	entries := []domainservices.BulkCreateTimesheetEntry{
		dedupEntry(1210, "2026-09-03", "ca ngày", "ngày thường", 7),
		dedupEntry(1210, "2026-09-03", "tăng ca", "ngày thường", 3),
		dedupEntry(1210, "2026-09-04", "ca ngày", "ngày thường", 8),
		dedupEntry(1211, "2026-09-03", "ca ngày", "ngày thường", 8),
	}

	got := dedupBCCEntries(entries)

	if len(got) != 4 {
		t.Fatalf("expected all 4 distinct entries to survive, got %d", len(got))
	}
}

// Merging must not reorder the surviving entries: the import reports errors by
// entry index, so a reshuffle would misattribute them to the wrong employee.
func TestDedupBCCEntriesPreservesFirstSeenOrder(t *testing.T) {
	entries := []domainservices.BulkCreateTimesheetEntry{
		dedupEntry(1210, "2026-09-03", "ca ngày", "ngày thường", 7),
		dedupEntry(1211, "2026-09-06", "ca ngày", "ngày nghỉ", 8),
		dedupEntry(1210, "2026-09-03", "ca ngày", "ngày thường", 1),
		dedupEntry(1212, "2026-09-04", "ca ngày", "ngày thường", 8),
	}

	got := dedupBCCEntries(entries)

	if len(got) != 3 {
		t.Fatalf("expected 3 entries after merging, got %d", len(got))
	}
	wantEmployees := []uint{1210, 1211, 1212}
	for i, want := range wantEmployees {
		if got[i].EmployeeID != want {
			t.Errorf("entry %d: expected employee %d, got %d", i, want, got[i].EmployeeID)
		}
	}
	if got[0].HoursWorked != 8 {
		t.Errorf("expected merged hours 8 on the first entry, got %v", got[0].HoursWorked)
	}
}

// Different projects never share a bucket even on the same employee/date/shift.
func TestDedupBCCEntriesSeparatesProjects(t *testing.T) {
	a := dedupEntry(1210, "2026-09-06", "ca ngày", "ngày nghỉ", 8)
	b := dedupEntry(1210, "2026-09-06", "ca ngày", "ngày nghỉ", 8)
	b.ProjectID = 10

	got := dedupBCCEntries([]domainservices.BulkCreateTimesheetEntry{a, b})

	if len(got) != 2 {
		t.Fatalf("expected entries from different projects to stay separate, got %d", len(got))
	}
}

// An in-batch collision means the file disagrees with itself. Reporting it as
// "Dữ liệu đã tồn tại" points partners at stored timesheets that are fine.
func TestSafeBulkFailureReasonDistinguishesInBatchDuplicate(t *testing.T) {
	inBatch := "Mục nhập trùng lặp trong yêu cầu: Nhân viên ID 1210, Dự án ID 54, " +
		"Ngày 2026-09-06, Loại giờ 'ca ngày' xuất hiện nhiều lần trong yêu cầu này"

	if got := safeBulkFailureReason(inBatch); got != "Tệp có nhiều ô chấm công trùng loại giờ cho cùng một ngày" {
		t.Errorf("in-batch duplicate got %q", got)
	}

	// The stored-conflict path reaches this function as the driver's unique-key
	// violation and must keep reporting existing data.
	stored := "Error 1062: duplicate key timesheets_unique"
	if got := safeBulkFailureReason(stored); got != "Dữ liệu đã tồn tại" {
		t.Errorf("stored-conflict duplicate got %q", got)
	}
}
