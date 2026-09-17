package services

import (
	"testing"
	"time"

	domainservices "api-server/internal/domain/services"
)

func TestEarliestEntryDateByEmployee(t *testing.T) {
	entries := []domainservices.BulkCreateTimesheetEntry{
		{EmployeeID: 7, Date: "2026-09-12", HoursWorked: 8},
		{EmployeeID: 7, Date: "2026-09-10", HoursWorked: 7},
		{EmployeeID: 7, Date: "2026-09-08", HoursWorked: 0}, // deletion request: no coverage requirement
		{EmployeeID: 9, Date: "2026-09-14", HoursWorked: 2.5},
		{EmployeeID: 9, Date: "bad-date", HoursWorked: 1}, // unparseable: skipped, not fatal
	}

	got := earliestEntryDateByEmployee(entries)

	if len(got) != 2 {
		t.Fatalf("expected 2 employees, got %d (%v)", len(got), got)
	}
	if want := time.Date(2026, 9, 10, 0, 0, 0, 0, time.Local); !got[7].Equal(want) {
		t.Fatalf("employee 7 earliest = %v, want %v (zero-hour cells must not count)", got[7], want)
	}
	if want := time.Date(2026, 9, 14, 0, 0, 0, 0, time.Local); !got[9].Equal(want) {
		t.Fatalf("employee 9 earliest = %v, want %v (bad dates skipped)", got[9], want)
	}
}

func TestEarliestEntryDateByEmployeeEmpty(t *testing.T) {
	if got := earliestEntryDateByEmployee(nil); len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}
