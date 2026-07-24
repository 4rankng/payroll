package domain

import "testing"

func TestTimesheetImportJobIsTerminal(t *testing.T) {
	for _, test := range []struct {
		status string
		want   bool
	}{
		{TimesheetImportStatusPending, false},
		{TimesheetImportStatusProcessing, false},
		{TimesheetImportStatusCompleted, true},
		{TimesheetImportStatusFailed, true},
	} {
		t.Run(test.status, func(t *testing.T) {
			job := TimesheetImportJob{Status: test.status}
			if got := job.IsTerminal(); got != test.want {
				t.Fatalf("IsTerminal() = %v, want %v", got, test.want)
			}
		})
	}
}
