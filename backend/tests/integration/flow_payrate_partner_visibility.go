package main

import (
	"fmt"
	"net/http"
)

const flowPayratePartnerVisibility = "PayratePartnerVisibility"

// runPayratePartnerVisibilityTests guards the fix for the bug where a partner
// account could not see existing (admin-created) payrates for a project.
//
// Root cause: ListPayrates used to scope partner results by CreatedBy, hiding
// payrates created by other users (admins). Payrates are project-level config,
// so access is now gated by project membership (ProjectPermissionService).
//
// Coverage:
//   - POSITIVE: a partner sees the SAME payrates as admin (including those
//     created by other users) for a project they can access.
//   - NEGATIVE: a partner is denied (403) when listing payrates for a project
//     they cannot access (guards against an IDOR regression).
func runPayratePartnerVisibilityTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Payrate partner visibility")

	if len(data.Partners) == 0 {
		reporter.Skip(flowPayratePartnerVisibility, "All tests", "no partner user available")
		return
	}

	admin := client.WithToken(data.AdminToken)
	partner := client.WithToken(data.Partners[0].Token)
	partnerID := data.Partners[0].ID

	// Projects the partner can access (created / shared / employee-assigned).
	var partnerProjects []ProjectResponse
	if _, err := partner.GetInto("/api/v1/projects?pageSize=100", &partnerProjects); err != nil {
		reporter.Skip(flowPayratePartnerVisibility, "All tests", fmt.Sprintf("could not list partner projects: %v", err))
		return
	}
	accessible := make(map[uint]bool, len(partnerProjects))
	for _, p := range partnerProjects {
		accessible[p.ID] = true
	}

	// POSITIVE: find an accessible project that has payrates created by another
	// user (e.g. admin). The partner must see all of them — before the fix this
	// list was empty because of the CreatedBy filter.
	var positiveProjectID uint
	var adminRateCount int
	for _, p := range partnerProjects {
		var rates []PayrateResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/payrates?project_id=%d&pageSize=100", p.ID), &rates); err != nil {
			continue
		}
		hasOtherCreator := false
		for _, r := range rates {
			if r.CreatedBy != partnerID {
				hasOtherCreator = true
				break
			}
		}
		if hasOtherCreator {
			positiveProjectID = p.ID
			adminRateCount = len(rates)
			break
		}
	}

	if positiveProjectID == 0 {
		reporter.Skip(flowPayratePartnerVisibility, "Partner sees admin payrates", "no accessible project with other-user-created payrates")
	} else {
		reporter.RunTest(flowPayratePartnerVisibility, "Partner sees admin payrates for accessible project", func() error {
			var partnerRates []PayrateResponse
			if _, err := partner.GetInto(fmt.Sprintf("/api/v1/payrates?project_id=%d&pageSize=100", positiveProjectID), &partnerRates); err != nil {
				return fmt.Errorf("partner listing payrates for project %d: %w", positiveProjectID, err)
			}
			if len(partnerRates) != adminRateCount {
				return fmt.Errorf("project %d: partner sees %d payrates, admin sees %d (regression: partner missing other-user-created payrates)", positiveProjectID, len(partnerRates), adminRateCount)
			}
			sawOther := false
			for _, r := range partnerRates {
				if r.CreatedBy != partnerID {
					sawOther = true
					break
				}
			}
			if !sawOther {
				return fmt.Errorf("project %d: partner does not see payrates created by another user", positiveProjectID)
			}
			fmt.Printf("    Partner sees all %d payrates (incl. other-user-created) for project %d\n", adminRateCount, positiveProjectID)
			return nil
		})
	}

	// NEGATIVE: a project the partner cannot access must return 403.
	var inaccessibleProjectID uint
	for _, p := range data.Projects {
		if !accessible[p.ID] {
			inaccessibleProjectID = p.ID
			break
		}
	}

	if inaccessibleProjectID == 0 {
		reporter.Skip(flowPayratePartnerVisibility, "Partner denied inaccessible project", "no inaccessible project found")
	} else {
		reporter.RunTest(flowPayratePartnerVisibility, "Partner denied payrates for inaccessible project", func() error {
			_, statusCode, err := partner.GetExpectError(fmt.Sprintf("/api/v1/payrates?project_id=%d", inaccessibleProjectID))
			if err != nil {
				return fmt.Errorf("request for inaccessible project %d: %w", inaccessibleProjectID, err)
			}
			return AssertEqual("status", http.StatusForbidden, statusCode)
		})
	}
}
