package services

import (
	"context"
	"testing"

	"api-server/internal/domain"
)

func TestSetInitialTimesheetStatusForBulk_RequiresApprovalForBCCImport(t *testing.T) {
	service := &TimesheetDomainService{}

	t.Run("admin BCC import remains pending approval", func(t *testing.T) {
		timesheet := &domain.Timesheet{}
		if err := service.SetInitialTimesheetStatusForBulk(context.Background(), timesheet, 42, "admin", true); err != nil {
			t.Fatalf("SetInitialTimesheetStatusForBulk() error = %v", err)
		}
		if timesheet.Status != domain.TimesheetStatusPendingApproval {
			t.Fatalf("status = %q, want %q", timesheet.Status, domain.TimesheetStatusPendingApproval)
		}
		if timesheet.ApprovedBy != nil || timesheet.ApprovedAt != nil {
			t.Fatal("BCC-imported timesheet must not have approval metadata")
		}
	})

	t.Run("manual admin entry remains auto-approved", func(t *testing.T) {
		timesheet := &domain.Timesheet{}
		if err := service.SetInitialTimesheetStatusForBulk(context.Background(), timesheet, 42, "admin", false); err != nil {
			t.Fatalf("SetInitialTimesheetStatusForBulk() error = %v", err)
		}
		if timesheet.Status != domain.TimesheetStatusApproved {
			t.Fatalf("status = %q, want %q", timesheet.Status, domain.TimesheetStatusApproved)
		}
		if timesheet.ApprovedBy == nil || *timesheet.ApprovedBy != 42 || timesheet.ApprovedAt == nil {
			t.Fatal("manual admin entry must retain approval metadata")
		}
	})
}
