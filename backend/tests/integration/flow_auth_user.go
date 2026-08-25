package main

import (
	"fmt"
	"net/http"
	"time"
)

const flowAuth = "AuthUser"

func runAuthUserTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Auth & User Management")

	admin := client.WithToken(data.AdminToken)
	prefix := cfg.UniquePrefix()

	// --- Profile tests ---

	reporter.RunTest(flowAuth, "Get current admin profile", func() error {
		var me UserResponse
		if _, err := admin.GetInto("/api/v1/auth/me", &me); err != nil {
			return fmt.Errorf("get profile: %w", err)
		}
		if err := AssertEqual("username", cfg.AdminUsername, me.Username); err != nil {
			return err
		}
		return AssertEqual("role", "admin", me.Role)
	})

	reporter.RunTest(flowAuth, "Update admin profile", func() error {
		body := map[string]interface{}{"fullname": "Admin ITest Updated"}
		if _, _, err := admin.Put("/api/v1/auth/me", body); err != nil {
			return fmt.Errorf("update profile: %w", err)
		}
		var me UserResponse
		if _, err := admin.GetInto("/api/v1/auth/me", &me); err != nil {
			return fmt.Errorf("get profile after update: %w", err)
		}
		return AssertEqual("fullname", "Admin ITest Updated", me.Fullname)
	})

	reporter.RunTest(flowAuth, "Restore admin fullname", func() error {
		body := map[string]interface{}{"fullname": "Frank Ng"}
		_, _, err := admin.Put("/api/v1/auth/me", body)
		return err
	})

	// Wrong current_password must be a 400 validation error, not 401: the
	// session is valid and only the submitted field is wrong. A 401 makes the
	// SPA's global interceptor treat the typo as session expiry and log the
	// user out mid-change — the captive force-change dialog's core error path.
	reporter.RunTest(flowAuth, "Edge: change password with wrong current password returns 400", func() error {
		body := map[string]interface{}{
			"current_password": "WrongCurrent@9",
			"new_password":     "NewValid@2026x",
		}
		apiErr, status, err := admin.PostExpectError("/api/v1/auth/change-password", body)
		if err != nil {
			return fmt.Errorf("expected error response: %w", err)
		}
		if status != http.StatusBadRequest {
			return fmt.Errorf("expected 400 (validation), got %d (msg=%s)", status, apiErr.Message)
		}
		fmt.Printf("    Wrong current password rejected with %d: %s\n", status, apiErr.Message)
		return nil
	})

	// --- Auth edge cases ---

	reporter.RunTest(flowAuth, "Edge: login with wrong password", func() error {
		tmpClient := NewAPIClient(client.BaseURL)
		_, err := tmpClient.Login(cfg.AdminUsername, "WrongPassword123")
		if err == nil {
			return fmt.Errorf("expected login to fail with wrong password")
		}
		fmt.Printf("    Login correctly rejected: %v\n", err)
		return nil
	})

	reporter.RunTest(flowAuth, "Edge: access protected route without token", func() error {
		tmpClient := NewAPIClient(client.BaseURL)
		_, statusCode, err := tmpClient.Get("/api/v1/projects")
		if err == nil && statusCode < 400 {
			return fmt.Errorf("expected error accessing protected route without token, got HTTP %d", statusCode)
		}
		fmt.Printf("    Access correctly rejected: %v\n", err)
		return nil
	})

	// --- User CRUD ---

	var testUserID uint
	mobileSuffix := time.Now().UnixNano() % 10_000_000
	originalMobile := fmt.Sprintf("090%07d", mobileSuffix)
	updatedMobile := fmt.Sprintf("091%07d", mobileSuffix)

	reporter.RunTest(flowAuth, "Create test user", func() error {
		body := CreateUserRequest{
			Username: prefix + "_user",
			Password: "TestPass123!",
			Fullname: "ITest Partner User",
			Email:    prefix + "@itest.local",
			Role:     "partner",
			Mobile:   originalMobile,
		}
		var resp UserResponse
		if _, err := admin.PostInto("/api/v1/users", body, &resp); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		testUserID = resp.ID
		if err := AssertEqual("mobile", originalMobile, resp.Mobile); err != nil {
			return err
		}
		fmt.Printf("    Created user ID %d, username: %s\n", resp.ID, resp.Username)
		return AssertGreaterThan("id", uint(0), resp.ID)
	})

	defer func() {
		if testUserID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/users/%d", testUserID))
		}
	}()

	reporter.RunTest(flowAuth, "List users", func() error {
		var users []UserResponse
		if _, err := admin.GetInto("/api/v1/users?pageSize=10", &users); err != nil {
			return fmt.Errorf("list users: %w", err)
		}
		fmt.Printf("    Found %d users\n", len(users))
		return AssertSliceMinLen("users", len(users), 1)
	})

	reporter.RunTest(flowAuth, "Get user by ID", func() error {
		if testUserID == 0 {
			return fmt.Errorf("no test user ID")
		}
		var resp UserResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/users/%d", testUserID), &resp); err != nil {
			return fmt.Errorf("get user: %w", err)
		}
		if err := AssertEqual("username", prefix+"_user", resp.Username); err != nil {
			return err
		}
		return AssertEqual("mobile", originalMobile, resp.Mobile)
	})

	reporter.RunTest(flowAuth, "Update user fullname", func() error {
		if testUserID == 0 {
			return fmt.Errorf("no test user ID")
		}
		body := map[string]interface{}{"fullname": "ITest Updated Name"}
		if _, _, err := admin.Put(fmt.Sprintf("/api/v1/users/%d", testUserID), body); err != nil {
			return fmt.Errorf("update user: %w", err)
		}
		var resp UserResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/users/%d", testUserID), &resp); err != nil {
			return fmt.Errorf("get after update: %w", err)
		}
		return AssertContains("fullname", resp.Fullname, "Updated Name")
	})

	reporter.RunTest(flowAuth, "Update and clear partner mobile", func() error {
		if testUserID == 0 {
			return fmt.Errorf("no test user ID")
		}
		body := map[string]interface{}{"mobile": updatedMobile}
		if _, statusCode, err := admin.Put(fmt.Sprintf("/api/v1/users/%d", testUserID), body); err != nil {
			return fmt.Errorf("update mobile: %w", err)
		} else if statusCode >= 400 {
			return fmt.Errorf("update mobile returned HTTP %d", statusCode)
		}
		var resp UserResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/users/%d", testUserID), &resp); err != nil {
			return fmt.Errorf("get after mobile update: %w", err)
		}
		if err := AssertEqual("mobile", updatedMobile, resp.Mobile); err != nil {
			return err
		}
		if _, statusCode, err := admin.Put(fmt.Sprintf("/api/v1/users/%d", testUserID), map[string]interface{}{"mobile": ""}); err != nil {
			return fmt.Errorf("clear mobile: %w", err)
		} else if statusCode >= 400 {
			return fmt.Errorf("clear mobile returned HTTP %d", statusCode)
		}
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/users/%d", testUserID), &resp); err != nil {
			return fmt.Errorf("get after mobile clear: %w", err)
		}
		return AssertEqual("mobile", "", resp.Mobile)
	})

	reporter.RunTest(flowAuth, "Get user summary", func() error {
		var resp UserSummaryResponse
		if _, err := admin.GetInto("/api/v1/users/summary", &resp); err != nil {
			return fmt.Errorf("user summary: %w", err)
		}
		fmt.Printf("    Total: %d, Admins: %d, Partners: %d\n",
			resp.TotalUsers, resp.TotalAdmins, resp.TotalPartners)
		return AssertGreaterThan("total_users", int64(0), resp.TotalUsers)
	})

	reporter.RunTest(flowAuth, "Reset user password", func() error {
		if testUserID == 0 {
			return fmt.Errorf("no test user ID")
		}
		body := ResetPasswordRequest{Password: "NewPass123!"}
		if _, _, err := admin.Post(fmt.Sprintf("/api/v1/users/%d/reset-password", testUserID), body); err != nil {
			return fmt.Errorf("reset password: %w", err)
		}
		// Verify login with new password
		tmpClient := NewAPIClient(client.BaseURL)
		if _, err := tmpClient.Login(prefix+"_user", "NewPass123!"); err != nil {
			return fmt.Errorf("login with new password: %w", err)
		}
		fmt.Printf("    Password reset and login verified\n")
		return nil
	})

	reporter.RunTest(flowAuth, "Delete test user", func() error {
		if testUserID == 0 {
			return fmt.Errorf("no test user ID")
		}
		if _, _, err := admin.Delete(fmt.Sprintf("/api/v1/users/%d", testUserID)); err != nil {
			return fmt.Errorf("delete user: %w", err)
		}
		testUserID = 0
		fmt.Printf("    Test user deleted\n")
		return nil
	})

	// --- Edge cases ---

	reporter.RunTest(flowAuth, "Edge: create user with short password", func() error {
		body := CreateUserRequest{
			Username: prefix + "_short",
			Password: "abc",
			Fullname: "ITest Short Pass",
		}
		_, statusCode, err := admin.PostExpectError("/api/v1/users", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})

	reporter.RunTest(flowAuth, "Edge: create user with duplicate username", func() error {
		body := CreateUserRequest{
			Username: cfg.AdminUsername,
			Password: "TestPass123!",
			Fullname: "ITest Duplicate",
		}
		_, statusCode, err := admin.PostExpectError("/api/v1/users", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})

	reporter.RunTest(flowAuth, "Edge: get non-existent user", func() error {
		_, statusCode, _ := admin.Get("/api/v1/users/999999")
		if statusCode < 400 {
			return fmt.Errorf("expected error for non-existent user, got HTTP %d", statusCode)
		}
		return nil
	})

	reporter.RunTest(flowAuth, "Edge: update non-existent user", func() error {
		body := map[string]interface{}{"fullname": "Ghost"}
		_, statusCode, _ := admin.Put("/api/v1/users/999999", body)
		if statusCode < 400 {
			return fmt.Errorf("expected error updating non-existent user, got HTTP %d", statusCode)
		}
		return nil
	})
}
