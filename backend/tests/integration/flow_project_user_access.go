package main

import (
	"fmt"
	"net/http"
)

// PAYR-30 (id 705): project access guard matrix at the HTTP layer.
//
// Permission model:
//   - admin grants/revokes on any project (200), including revoking the
//     project creator (the creator then loses access entirely);
//   - partner-admin (project creator or holder of a project_users grant) may
//     grant/revoke only within their own project (403 elsewhere) and may not
//     revoke the creator (403, admin-only);
//   - granting management rights to a company-side role is an elevation
//     attempt and is rejected with 403.

const flowProjectUserAccess = "ProjectUserAccess"

type projectUserAccessFixture struct {
	admin      *APIClient
	creatorA   *APIClient // partner, created projectX
	managerB   *APIClient // partner, granted manager on projectX
	adminUID   uint
	creatorAID uint
	managerBID uint
	targetCID  uint
	projectX   uint // owned by creatorA — in scope for partner-admins
	projectY   uint // owned by admin — out of scope for partner-admins
}

type puaStep struct {
	name string
	run  func(*projectUserAccessFixture) error
}

func expectStatus(step string, got, want int) error {
	if got != want {
		return fmt.Errorf("%s: expected HTTP %d, got %d", step, want, got)
	}
	return nil
}

// puaSteps is the shared guard matrix: run through reporter.RunTest in the
// suite flow and through t.Run in the scoped guard test.
var puaSteps = []puaStep{
	{"PartnerCreator_Grant_Manager_InOwnProject_200", func(f *projectUserAccessFixture) error {
		_, status, err := f.creatorA.Post(fmt.Sprintf("/api/v1/projects/%d/users", f.projectX),
			map[string]uint{"user_id": f.managerBID})
		if err != nil {
			return fmt.Errorf("grant: %w", err)
		}
		return expectStatus("creator grants manager", status, http.StatusOK)
	}},
	{"PartnerManager_Grant_SameProject_200", func(f *projectUserAccessFixture) error {
		_, status, err := f.managerB.Post(fmt.Sprintf("/api/v1/projects/%d/users", f.projectX),
			map[string]uint{"user_id": f.targetCID})
		if err != nil {
			return fmt.Errorf("grant: %w", err)
		}
		return expectStatus("granted manager grants in same project", status, http.StatusOK)
	}},
	{"PartnerManager_Grant_CrossProject_403", func(f *projectUserAccessFixture) error {
		_, status, err := f.managerB.Post(fmt.Sprintf("/api/v1/projects/%d/users", f.projectY),
			map[string]uint{"user_id": f.targetCID})
		if err != nil {
			return fmt.Errorf("cross-project grant: %w", err)
		}
		return expectStatus("cross-project grant", status, http.StatusForbidden)
	}},
	{"PartnerManager_Grant_CompanySideTarget_403", func(f *projectUserAccessFixture) error {
		_, status, err := f.managerB.Post(fmt.Sprintf("/api/v1/projects/%d/users", f.projectX),
			map[string]uint{"user_id": f.adminUID})
		if err != nil {
			return fmt.Errorf("company-side target grant: %w", err)
		}
		return expectStatus("company-side target grant", status, http.StatusForbidden)
	}},
	{"PartnerManager_Revoke_SameProject_200", func(f *projectUserAccessFixture) error {
		_, status, err := f.managerB.Delete(fmt.Sprintf("/api/v1/projects/%d/users/%d", f.projectX, f.targetCID))
		if err != nil {
			return fmt.Errorf("revoke: %w", err)
		}
		return expectStatus("granted manager revokes in same project", status, http.StatusOK)
	}},
	{"PartnerManager_Revoke_CrossProject_403", func(f *projectUserAccessFixture) error {
		_, status, err := f.managerB.Delete(fmt.Sprintf("/api/v1/projects/%d/users/%d", f.projectY, f.targetCID))
		if err != nil {
			return fmt.Errorf("cross-project revoke: %w", err)
		}
		return expectStatus("cross-project revoke", status, http.StatusForbidden)
	}},
	{"PartnerManager_Revoke_Creator_403", func(f *projectUserAccessFixture) error {
		_, status, err := f.managerB.Delete(fmt.Sprintf("/api/v1/projects/%d/users/%d", f.projectX, f.creatorAID))
		if err != nil {
			return fmt.Errorf("partner-admin creator revoke: %w", err)
		}
		return expectStatus("partner-admin revokes creator", status, http.StatusForbidden)
	}},
	{"Admin_GrantRevoke_Baseline_200", func(f *projectUserAccessFixture) error {
		_, status, err := f.admin.Post(fmt.Sprintf("/api/v1/projects/%d/users", f.projectX),
			map[string]uint{"user_id": f.targetCID})
		if err != nil {
			return fmt.Errorf("admin grant: %w", err)
		}
		if err := expectStatus("admin grant", status, http.StatusOK); err != nil {
			return err
		}
		_, status, err = f.admin.Delete(fmt.Sprintf("/api/v1/projects/%d/users/%d", f.projectX, f.targetCID))
		if err != nil {
			return fmt.Errorf("admin revoke: %w", err)
		}
		return expectStatus("admin revoke", status, http.StatusOK)
	}},
	{"Admin_Revoke_Creator_200", func(f *projectUserAccessFixture) error {
		_, status, err := f.admin.Delete(fmt.Sprintf("/api/v1/projects/%d/users/%d", f.projectX, f.creatorAID))
		if err != nil {
			return fmt.Errorf("admin revoke creator: %w", err)
		}
		return expectStatus("admin revokes creator", status, http.StatusOK)
	}},
	{"Admin_Revoke_Creator_RemovesFromAccessList", func(f *projectUserAccessFixture) error {
		var users []struct {
			UserID uint `json:"user_id"`
		}
		if _, err := f.admin.GetInto(fmt.Sprintf("/api/v1/projects/%d/users", f.projectX), &users); err != nil {
			return fmt.Errorf("list project users: %w", err)
		}
		for _, u := range users {
			if u.UserID == f.creatorAID {
				return fmt.Errorf("revoked creator %d still listed as project manager", f.creatorAID)
			}
		}
		return nil
	}},
	{"RevokedCreator_LosesProjectAccess_403", func(f *projectUserAccessFixture) error {
		_, status, err := f.creatorA.GetExpectError(fmt.Sprintf("/api/v1/projects/%d", f.projectX))
		if err != nil {
			return fmt.Errorf("revoked creator project fetch: %w", err)
		}
		return expectStatus("revoked creator project fetch", status, http.StatusForbidden)
	}},
	{"RevokedCreator_CannotGrant_403", func(f *projectUserAccessFixture) error {
		_, status, err := f.creatorA.Post(fmt.Sprintf("/api/v1/projects/%d/users", f.projectX),
			map[string]uint{"user_id": f.targetCID})
		if err != nil {
			return fmt.Errorf("revoked creator grant: %w", err)
		}
		return expectStatus("revoked creator grant", status, http.StatusForbidden)
	}},
}

func setupProjectUserAccessFixture(base *APIClient, adminToken, prefix string) (*projectUserAccessFixture, func(), error) {
	f := &projectUserAccessFixture{admin: base.WithToken(adminToken)}

	var adminProfile struct {
		ID uint `json:"id"`
	}
	if _, err := f.admin.GetInto("/api/v1/auth/me", &adminProfile); err != nil {
		return nil, nil, fmt.Errorf("admin profile: %w", err)
	}
	f.adminUID = adminProfile.ID

	password := "TestPass123!"
	usernames := map[string]string{
		"creatorA": prefix + "_pua_a",
		"managerB": prefix + "_pua_b",
		"targetC":  prefix + "_pua_c",
	}
	userIDs := map[string]uint{}
	clients := map[string]*APIClient{}
	for key, username := range usernames {
		var created UserResponse
		if _, err := f.admin.PostInto("/api/v1/users", CreateUserRequest{
			Username: username,
			Password: password,
			Fullname: "ITest PUA " + key,
			Role:     "partner",
		}, &created); err != nil {
			return nil, nil, fmt.Errorf("create %s: %w", key, err)
		}
		userIDs[key] = created.ID
		userClient := NewAPIClient(base.BaseURL)
		if _, err := userClient.Login(username, password); err != nil {
			return nil, nil, fmt.Errorf("login %s: %w", key, err)
		}
		clients[key] = userClient
	}
	f.creatorAID, f.managerBID, f.targetCID = userIDs["creatorA"], userIDs["managerB"], userIDs["targetC"]
	f.creatorA, f.managerB = clients["creatorA"], clients["managerB"]

	var projectX ProjectResponse
	if _, err := f.creatorA.PostInto("/api/v1/projects", CreateProjectRequest{
		ClientName: prefix + " PUA Client",
		Name:       prefix + " PUA X",
		Code:       prefix + "PUAX",
	}, &projectX); err != nil {
		return nil, nil, fmt.Errorf("create projectX: %w", err)
	}
	f.projectX = projectX.ID

	var projectY ProjectResponse
	if _, err := f.admin.PostInto("/api/v1/projects", CreateProjectRequest{
		ClientName: prefix + " PUA Client",
		Name:       prefix + " PUA Y",
		Code:       prefix + "PUAY",
	}, &projectY); err != nil {
		return nil, nil, fmt.Errorf("create projectY: %w", err)
	}
	f.projectY = projectY.ID

	teardown := func() {
		for _, id := range []uint{f.projectX, f.projectY} {
			if id != 0 {
				_, _, _ = f.admin.Delete(fmt.Sprintf("/api/v1/projects/%d", id))
			}
		}
		for _, id := range []uint{f.creatorAID, f.managerBID, f.targetCID} {
			if id != 0 {
				_, _, _ = f.admin.Delete(fmt.Sprintf("/api/v1/users/%d", id))
			}
		}
	}
	return f, teardown, nil
}

func runProjectUserAccessTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Project User Access Guards (PAYR-30)")

	fixture, teardown, err := setupProjectUserAccessFixture(client, data.AdminToken, cfg.UniquePrefix())
	if err != nil {
		reporter.RunTest(flowProjectUserAccess, "Setup fixture", func() error { return err })
		return
	}
	defer teardown()

	for _, step := range puaSteps {
		step := step
		reporter.RunTest(flowProjectUserAccess, step.name, func() error {
			return step.run(fixture)
		})
	}
}
