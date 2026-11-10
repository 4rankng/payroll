package main

import (
	"fmt"
	"net/http"

	middleware "api-server/internal/transport/http/middleware"
)

const flowRoleEnforcement = "RoleEnforcement"

// runRoleEnforcementTests pins PAYR-37 item 5: authorization is decided
// server-side (Casbin, deny-by-default) on EVERY protected route — the
// client-stored role is display data only. Two accounts of different roles
// cross-call each other's routes and every denial must come from the backend.
// It also pins that the RBAC verdict is identical over the new HttpOnly
// cookie channel (item 4) as over the legacy Bearer header.
func runRoleEnforcementTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Role Enforcement (server-side RBAC pin)")

	admin := client.WithToken(data.AdminToken)
	prefix := cfg.UniquePrefix()

	// Fixture: one employee account (fresh, unique prefix) to pair against
	// the admin. Partner accounts come from the configured pool.
	username := prefix + "_rbac_emp"
	password := "TestPass123!"
	var created UserResponse
	if _, err := admin.PostInto("/api/v1/users", CreateUserRequest{
		Username: username,
		Password: password,
		Fullname: "ITest RBAC Employee",
		Role:     "employee",
	}, &created); err != nil {
		reporter.Record(TestResult{Flow: flowRoleEnforcement, Name: "create employee fixture", Status: StatusFail, Error: fmt.Errorf("create employee: %w", err).Error()})
		return
	}
	defer func() { _, _, _ = admin.Delete(fmt.Sprintf("/api/v1/users/%d", created.ID)) }()

	empClient := NewAPIClient(client.BaseURL)
	if _, err := empClient.Login(username, password); err != nil {
		reporter.Record(TestResult{Flow: flowRoleEnforcement, Name: "login employee fixture", Status: StatusFail, Error: fmt.Errorf("login employee: %w", err).Error()})
		return
	}

	partnerName := cfg.PartnerUsernames[0]
	partnerClient := NewAPIClient(client.BaseURL)
	if _, err := partnerClient.Login(partnerName, cfg.CommonPassword); err != nil {
		reporter.Record(TestResult{Flow: flowRoleEnforcement, Name: "login partner fixture", Status: StatusFail, Error: fmt.Errorf("login partner %s: %w", partnerName, err).Error()})
		return
	}

	// --- Cross-role matrix (Bearer channel) ---

	crossRoleCases := []struct {
		name   string
		client *APIClient
		method string
		path   string
	}{
		{"employee calls admin user list", empClient, http.MethodGet, "/api/v1/users"},
		{"employee calls admin user create", empClient, http.MethodPost, "/api/v1/users"},
		{"employee calls settings write (admin-only)", empClient, http.MethodPost, "/api/v1/settings"},
		{"employee calls approval decision (explicit deny)", empClient, http.MethodPost, "/api/v1/approvals/99999/approve"},
		{"employee calls cron toggle (deny-by-default)", empClient, http.MethodPut, "/api/v1/cron-jobs/x/toggle"},
		{"employee calls zns admin list (deny-by-default)", empClient, http.MethodGet, "/api/v1/admin/zns-notifications"},
		{"partner calls admin user create", partnerClient, http.MethodPost, "/api/v1/users"},
		{"partner calls admin api-key create", partnerClient, http.MethodPost, "/api/v1/admin/api-keys"},
		{"partner calls cron toggle (deny-by-default)", partnerClient, http.MethodPut, "/api/v1/cron-jobs/x/toggle"},
		{"partner calls db export start (deny-by-default)", partnerClient, http.MethodPost, "/api/v1/db-export"},
	}

	for _, tc := range crossRoleCases {
		reporter.RunTest(flowRoleEnforcement, "RBAC deny: "+tc.name, func() error {
			var (
				status int
				apiErr *APIError
				err    error
			)
			switch tc.method {
			case http.MethodPost:
				apiErr, status, err = tc.client.PostExpectError(tc.path, map[string]string{})
			case http.MethodPut:
				apiErr, status, err = tc.client.PutExpectError(tc.path, map[string]string{})
			case http.MethodGet:
				apiErr, status, err = tc.client.GetExpectError(tc.path)
			case http.MethodDelete:
				apiErr, status, err = tc.client.DeleteExpectError(tc.path)
			}
			if err != nil {
				return fmt.Errorf("expected an error response: %w", err)
			}
			if status != http.StatusForbidden {
				return fmt.Errorf("expected 403, got %d (msg=%s)", status, apiErr.Message)
			}
			return nil
		})
	}

	// Positive pins: the same accounts still reach what their role allows —
	// proving the 403s above are policy decisions, not broken auth.
	reporter.RunTest(flowRoleEnforcement, "RBAC allow: employee reaches /auth/me", func() error {
		var me UserResponse
		if _, err := empClient.GetInto("/api/v1/auth/me", &me); err != nil {
			return fmt.Errorf("employee /auth/me: %w", err)
		}
		return AssertEqual("role", "employee", me.Role)
	})

	reporter.RunTest(flowRoleEnforcement, "RBAC allow: employee reads settings (explicit grant)", func() error {
		if _, status, err := empClient.Get("/api/v1/settings"); err != nil || status >= 400 {
			return fmt.Errorf("employee settings read: status=%d err=%v", status, err)
		}
		return nil
	})

	reporter.RunTest(flowRoleEnforcement, "RBAC allow: admin reaches user list", func() error {
		if _, status, err := admin.Get("/api/v1/users"); err != nil || status >= 400 {
			return fmt.Errorf("admin user list: status=%d err=%v", status, err)
		}
		return nil
	})

	// --- Cookie channel (PAYR-37 item 4): same RBAC verdicts, cookie-only ---

	empCookie := NewAPIClient(client.BaseURL)
	reporter.RunTest(flowRoleEnforcement, "Cookie login sets HttpOnly session cookie", func() error {
		_, cookies, err := empCookie.LoginCookie(username, password)
		if err != nil {
			return err
		}
		for _, c := range cookies {
			if c.Name != middleware.AuthCookieName {
				continue
			}
			if !c.HttpOnly {
				return fmt.Errorf("session cookie must be HttpOnly")
			}
			if c.SameSite != http.SameSiteLaxMode {
				return fmt.Errorf("session cookie SameSite must be Lax, got %v", c.SameSite)
			}
			if c.Path != "/" {
				return fmt.Errorf("session cookie Path must be /, got %q", c.Path)
			}
			if c.MaxAge <= 0 {
				return fmt.Errorf("session cookie must be persistent (Max-Age>0), got %d", c.MaxAge)
			}
			return nil
		}
		return fmt.Errorf("no %s cookie in login response", middleware.AuthCookieName)
	})

	reporter.RunTest(flowRoleEnforcement, "Cookie-only session authenticates /auth/me", func() error {
		var me UserResponse
		if _, err := empCookie.WithCookie(empCookie.CookieToken).GetInto("/api/v1/auth/me", &me); err != nil {
			return fmt.Errorf("cookie /auth/me: %w", err)
		}
		return AssertEqual("role", "employee", me.Role)
	})

	reporter.RunTest(flowRoleEnforcement, "RBAC deny over cookie channel: employee calls admin user list", func() error {
		apiErr, status, err := empCookie.WithCookie(empCookie.CookieToken).GetExpectError("/api/v1/users")
		if err != nil {
			return fmt.Errorf("expected an error response: %w", err)
		}
		if status != http.StatusForbidden {
			return fmt.Errorf("expected 403, got %d (msg=%s)", status, apiErr.Message)
		}
		return nil
	})

	reporter.RunTest(flowRoleEnforcement, "Bearer auto-upgrade: legacy header request mints session cookie", func() error {
		// Simulate a cached old-SPA client: Bearer header only, no cookie.
		// LoginCookie is not used so the client carries the token as a
		// Bearer header (doRequest attaches it from .Token).
		legacy := NewAPIClient(client.BaseURL)
		if _, err := legacy.Login(username, password); err != nil {
			return fmt.Errorf("legacy login: %w", err)
		}
		resp, err := legacy.doRequest(http.MethodGet, "/api/v1/auth/me", nil)
		if err != nil {
			return fmt.Errorf("legacy /auth/me request: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		for _, c := range resp.Cookies() {
			if c.Name == middleware.AuthCookieName && c.Value == legacy.Token && c.HttpOnly {
				return nil
			}
		}
		return fmt.Errorf("Bearer-authenticated request did not auto-mint the session cookie (cookies=%d)", len(resp.Cookies()))
	})

	reporter.RunTest(flowRoleEnforcement, "Edge: garbage session cookie rejected with 401", func() error {
		_, status, err := empCookie.WithCookie("not-a-real-jwt").Get("/api/v1/auth/me")
		if err != nil {
			return fmt.Errorf("unexpected transport error: %w", err)
		}
		if status != http.StatusUnauthorized {
			return fmt.Errorf("expected 401 for garbage cookie, got %d", status)
		}
		return nil
	})

	reporter.RunTest(flowRoleEnforcement, "Cookie logout blacklists the session", func() error {
		logoutClient := NewAPIClient(client.BaseURL)
		if _, _, err := logoutClient.LoginCookie(username, password); err != nil {
			return fmt.Errorf("cookie login: %w", err)
		}
		if _, status, err := logoutClient.WithCookie(logoutClient.CookieToken).Post("/api/v1/auth/logout", nil); err != nil || status >= 400 {
			return fmt.Errorf("cookie logout: status=%d err=%v", status, err)
		}
		// The same cookie must now be dead (jti blacklisted).
		_, status, err := logoutClient.WithCookie(logoutClient.CookieToken).Get("/api/v1/auth/me")
		if err != nil {
			return fmt.Errorf("unexpected transport error: %w", err)
		}
		if status != http.StatusUnauthorized {
			return fmt.Errorf("expected 401 after cookie logout, got %d", status)
		}
		return nil
	})
}
