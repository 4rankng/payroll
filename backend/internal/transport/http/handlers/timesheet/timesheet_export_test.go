package timesheet

import (
	"testing"

	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
)

func reportFor(id uint) *domainServices.ProjectReportData {
	return &domainServices.ProjectReportData{Project: &domain.Project{ID: id, Name: "P"}}
}

// Regression: a partner calling GET /timesheets/payroll/report?atDate=...
// used to receive every active project, because that branch selects projects
// internally instead of scoping the query the way the date-range branch does.
func TestFilterReportDataForPartnerKeepsOnlyAllowedProjects(t *testing.T) {
	reportData := []*domainServices.ProjectReportData{reportFor(67), reportFor(15), reportFor(77)}

	kept := filterReportDataForPartner(reportData, []uint{67, 77})

	if len(kept) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(kept))
	}
	for _, rd := range kept {
		if rd.Project.ID != 67 && rd.Project.ID != 77 {
			t.Fatalf("partner received a project it cannot access: %d", rd.Project.ID)
		}
	}
}

func TestFilterReportDataForPartnerWithNoAccessReturnsNothing(t *testing.T) {
	reportData := []*domainServices.ProjectReportData{reportFor(67)}

	if kept := filterReportDataForPartner(reportData, nil); kept != nil {
		t.Fatalf("expected nil when the partner has no accessible project, got %d", len(kept))
	}
	if kept := filterReportDataForPartner(nil, []uint{67}); kept != nil {
		t.Fatalf("expected nil for an empty report, got %d", len(kept))
	}
}

func TestFilterReportDataForPartnerSkipsMalformedEntries(t *testing.T) {
	reportData := []*domainServices.ProjectReportData{
		nil,
		{Project: nil},
		reportFor(67),
	}

	kept := filterReportDataForPartner(reportData, []uint{67})

	if len(kept) != 1 || kept[0].Project.ID != 67 {
		t.Fatalf("expected only the well-formed entry, got %+v", kept)
	}
}
