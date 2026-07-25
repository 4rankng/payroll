package main

import (
	"fmt"
)

const flowPasswordReset = "PasswordReset"

// runPasswordResetTests exercises the self-service password-reset endpoints.
//
// Per Red Team C3, the cross-process harness cannot retrieve the emailed token
// (the SandboxProvider lives in the server process). So this integration test
// covers ONLY the no-token contracts:
//   - anti-enumeration: request returns 200 for known AND unknown emails
//   - confirm with a garbage token → 401
//   - confirm with a missing token → 400
//
// The single-use, transactional, and session-invalidation guarantees are
// covered by the unit tests in internal/app/services/passwordreset/.
func runPasswordResetTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Password Reset (email)")

	anonymous := NewAPIClient(client.BaseURL)

	// Resolve the admin's real email via /auth/me so the known-email test is real.
	admin := client.WithToken(data.AdminToken)
	var adminProfile struct {
		Email string `json:"email"`
	}
	if _, err := admin.GetInto("/api/v1/auth/me", &adminProfile); err != nil {
		reporter.RunTest(flowPasswordReset, "Setup: resolve admin email", func() error {
			return fmt.Errorf("get admin profile for email: %w", err)
		})
		// Without the email we still run the contract tests that don't need it.
		adminProfile.Email = ""
	}

	// Anti-enumeration: known email returns 200.
	if adminProfile.Email != "" {
		reporter.RunTest(flowPasswordReset, "Request reset for known email returns 200", func() error {
			body := map[string]interface{}{"email": adminProfile.Email}
			_, status, err := anonymous.Post("/api/v1/auth/password-reset/request", body)
			if err != nil {
				return fmt.Errorf("request reset (known): %w", err)
			}
			if status != 200 {
				return fmt.Errorf("known email: status = %d, want 200", status)
			}
			return nil
		})
	}

	// Anti-enumeration: unknown email returns the SAME 200.
	reporter.RunTest(flowPasswordReset, "Request reset for unknown email returns same 200 (anti-enumeration)", func() error {
		body := map[string]interface{}{"email": "definitely-not-real-" + cfg.UniquePrefix() + "@example.com"}
		_, status, err := anonymous.Post("/api/v1/auth/password-reset/request", body)
		if err != nil {
			return fmt.Errorf("request reset (unknown): %w", err)
		}
		if status != 200 {
			return fmt.Errorf("unknown email: status = %d, want 200 (anti-enumeration)", status)
		}
		return nil
	})

	// Confirm with a garbage token → 401 (invalid/expired).
	reporter.RunTest(flowPasswordReset, "Confirm with garbage token returns 401", func() error {
		body := map[string]interface{}{
			"token":        "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA-fake",
			"new_password": "SomeNewStrong!2026",
		}
		_, status, err := anonymous.Post("/api/v1/auth/password-reset/confirm", body)
		if err != nil {
			return fmt.Errorf("confirm garbage: unexpected transport error: %w", err)
		}
		if status != 401 {
			return fmt.Errorf("garbage token: status = %d, want 401", status)
		}
		return nil
	})

	// Confirm with a malformed body (missing token) → 400.
	reporter.RunTest(flowPasswordReset, "Confirm with missing token returns 400", func() error {
		body := map[string]interface{}{"new_password": "SomeNewStrong!2026"}
		_, status, err := anonymous.Post("/api/v1/auth/password-reset/confirm", body)
		if err != nil {
			return fmt.Errorf("confirm missing token: unexpected transport error: %w", err)
		}
		if status != 400 {
			return fmt.Errorf("missing token: status = %d, want 400", status)
		}
		return nil
	})
}
