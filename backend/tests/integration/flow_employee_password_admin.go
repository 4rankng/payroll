package main

import (
	"fmt"

	employeesvc "api-server/internal/app/services/employee"
)

const flowEmployeePasswordAdmin = "EmployeePasswordAdmin"

// runEmployeePasswordAdminTests covers the admin/partner password-reset
// endpoint PUT /api/v1/employees/:id/change-password.
//
// This endpoint previously had no coverage at all — the only change-password
// flow in the suite was the self-service /auth/change-password — so a
// regression here shipped unnoticed. These tests pin the contracts behind the
// reported "success shown but the password did not change" symptom:
//
//   - reset returns 200 AND the NEW password actually authenticates;
//   - the OLD password no longer authenticates;
//   - a token minted under the OLD password stops working (session
//     invalidation — the half that was silently broken, because the write used
//     a full-row Save that never advanced tokens_invalid_before);
//   - a weak password is rejected and does not disturb the working one;
//   - a partner without access is refused AND the stored password is untouched;
//   - a missing employee yields an error, never a 200 with nothing written.
func runEmployeePasswordAdminTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Employee Password Reset (Admin/Partner)")

	admin := client.WithToken(data.AdminToken)

	const newPassword = "Ch@ngeMe2026"
	// A freshly created employee's account carries the default employee
	// password, not the suite's shared CommonPassword — that is the credential
	// the reset supersedes.
	const preResetPassword = employeesvc.DefaultEmployeePassword

	var (
		testEmpID uint
		username  string
	)

	// ── Setup: create an employee with a usable mobile ──────────────────────
	reporter.RunTest(flowEmployeePasswordAdmin, "Setup: create employee to reset", func() error {
		// A mobile unique to this run. A mobile shared with another employee
		// makes identity.Resolver fail closed, so the employee could not log in
		// at all and the assertions below would measure the wrong thing.
		body := CreateEmployeeRequest{
			Fullname: "PW Reset " + cfg.UniquePrefix(),
			CCCD:     "99" + cfg.UniquePrefix() + "000",
			Mobile:   uniqueMobile(),
		}
		var resp EmployeeResponse
		if _, err := admin.PostInto("/api/v1/employees", body, &resp); err != nil {
			return fmt.Errorf("create employee: %w", err)
		}
		testEmpID = resp.ID

		var detail EmployeeDetailedResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/employees/%d", resp.ID), &detail); err != nil {
			return fmt.Errorf("get employee detail: %w", err)
		}
		if detail.Username == nil || *detail.Username == "" {
			return fmt.Errorf("employee %d has no linked user account; cannot test password reset", resp.ID)
		}
		username = *detail.Username

		fmt.Printf("    Created employee ID %d (username %s)\n", detail.ID, username)
		return AssertGreaterThan("id", uint(0), detail.ID)
	})

	defer func() {
		if testEmpID == 0 {
			return
		}
		if _, _, err := admin.Delete(fmt.Sprintf("/api/v1/employees/%d", testEmpID)); err != nil {
			fmt.Printf("    cleanup: delete employee %d: %v\n", testEmpID, err)
		}
	}()

	if testEmpID == 0 || username == "" {
		// Setup failed; every assertion below depends on it.
		return
	}

	changePath := fmt.Sprintf("/api/v1/employees/%d/change-password", testEmpID)

	// ── The employee can authenticate with the pre-reset password ───────────
	var oldToken string
	reporter.RunTest(flowEmployeePasswordAdmin, "Setup: employee logs in with the pre-reset password", func() error {
		tmp := NewAPIClient(client.BaseURL)
		loginResp, err := tmp.Login(username, preResetPassword)
		if err != nil {
			return fmt.Errorf("pre-reset login for %s: %w", username, err)
		}
		if loginResp.AccessToken == "" {
			return fmt.Errorf("pre-reset login returned no token")
		}
		oldToken = loginResp.AccessToken
		return nil
	})

	// ── The core contract: reset → new password works ──────────────────────
	reporter.RunTest(flowEmployeePasswordAdmin, "Admin resets password and the NEW password authenticates", func() error {
		body := map[string]string{"new_password": newPassword}
		resp, status, err := admin.Put(changePath, body)
		if err != nil {
			return fmt.Errorf("reset password: %w", err)
		}
		if status != 200 {
			return fmt.Errorf("reset status = %d, want 200 (message: %s)", status, resp.Message)
		}

		tmp := NewAPIClient(client.BaseURL)
		if _, err := tmp.Login(username, newPassword); err != nil {
			return fmt.Errorf("login with the new password failed: %w", err)
		}
		fmt.Printf("    New password authenticates\n")
		return nil
	})

	// ── The superseded password must be dead ───────────────────────────────
	reporter.RunTest(flowEmployeePasswordAdmin, "The OLD password no longer authenticates", func() error {
		tmp := NewAPIClient(client.BaseURL)
		if _, err := tmp.Login(username, preResetPassword); err == nil {
			return fmt.Errorf("login with the old password still succeeded — reset did not take effect")
		}
		return nil
	})

	// ── Session invalidation: the pre-reset token must stop working ────────
	//
	// This is the assertion that failed before the fix. The reset wrote the new
	// hash but left tokens_invalid_before untouched, so any token already issued
	// under the old password kept working until it expired on its own.
	reporter.RunTest(flowEmployeePasswordAdmin, "Session issued under the OLD password is invalidated", func() error {
		if oldToken == "" {
			return fmt.Errorf("no pre-reset token captured")
		}
		emp := NewAPIClient(client.BaseURL).WithToken(oldToken)
		var me UserResponse
		if _, err := emp.GetInto("/api/v1/auth/me", &me); err == nil {
			return fmt.Errorf("a token minted before the password reset still authenticates — " +
				"tokens_invalid_before was not advanced, so the previous credential stays live")
		}
		fmt.Printf("    Pre-reset token correctly rejected\n")
		return nil
	})

	// ── Weak passwords are still rejected, and change nothing ──────────────
	reporter.RunTest(flowEmployeePasswordAdmin, "Weak new password is rejected and leaves the password intact", func() error {
		body := map[string]string{"new_password": "weak"}
		_, status, err := admin.Put(changePath, body)
		if err != nil {
			return fmt.Errorf("put weak password: %w", err)
		}
		if status < 400 {
			return fmt.Errorf("weak password accepted with status %d, want >= 400", status)
		}

		// The rejected attempt must not have changed the working password.
		tmp := NewAPIClient(client.BaseURL)
		if _, err := tmp.Login(username, newPassword); err != nil {
			return fmt.Errorf("a rejected weak password changed the stored password: %w", err)
		}
		return nil
	})

	// ── Partner RBAC ───────────────────────────────────────────────────────
	reporter.RunTest(flowEmployeePasswordAdmin, "Partner without access is refused and changes nothing", func() error {
		if len(data.Partners) == 0 {
			return fmt.Errorf("no partner fixture available to exercise partner RBAC")
		}
		partner := data.Partners[0]
		pc := NewAPIClient(client.BaseURL).WithToken(partner.Token)

		body := map[string]string{"new_password": "PartnerTry@2026"}
		resp, status, err := pc.Put(changePath, body)
		if err != nil {
			return fmt.Errorf("partner reset attempt: %w", err)
		}

		switch status {
		case 403:
			// The refusal must not have touched the stored password.
			tmp := NewAPIClient(client.BaseURL)
			if _, err := tmp.Login(username, newPassword); err != nil {
				return fmt.Errorf("a refused partner reset changed the password: %w", err)
			}
			fmt.Printf("    Partner refused (403), password unchanged\n")
			return nil
		case 200:
			// The partner had legitimate access; the reset must then work.
			tmp := NewAPIClient(client.BaseURL)
			if _, err := tmp.Login(username, "PartnerTry@2026"); err != nil {
				return fmt.Errorf("partner reset returned 200 but the password does not work: %w", err)
			}
			// Restore the known-good password for any later assertions.
			if _, _, err := admin.Put(changePath, map[string]string{"new_password": newPassword}); err != nil {
				return fmt.Errorf("restore password after partner reset: %w", err)
			}
			fmt.Printf("    Partner had access, reset applied and verified\n")
			return nil
		default:
			return fmt.Errorf("unexpected status %d (message: %s)", status, resp.Message)
		}
	})

	// ── A missing employee must error, never 200-with-nothing-written ──────
	reporter.RunTest(flowEmployeePasswordAdmin, "Missing employee returns an error, not a silent success", func() error {
		body := map[string]string{"new_password": newPassword}
		resp, status, err := admin.Put("/api/v1/employees/999999999/change-password", body)
		if err != nil {
			return fmt.Errorf("reset for missing employee: %w", err)
		}
		if status < 400 {
			return fmt.Errorf("reset for a non-existent employee returned %d, want >= 400", status)
		}
		if resp.Message == "" {
			return fmt.Errorf("error response carried no message")
		}
		return nil
	})
}
