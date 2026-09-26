package main

import (
	"fmt"
	"time"
)

const flowIntegrationAPI = "IntegrationAPI"

// uniqueMobile returns a syntactically-valid Vietnamese mobile that is very
// unlikely to be assigned, so the "unknown phone" path is deterministic.
func uniqueMobile() string {
	n := time.Now().UnixNano()
	d := n % 10
	if d == 5 {
		d = 7
	}
	return fmt.Sprintf("09%d%07d", d, n%10_000_000)
}

// runIntegrationResetTests exercises the API-key-authenticated chatbot channel:
// admin key management plus the integration endpoint guards. The full three-step
// reset cannot be driven cross-process (the sandbox OTP only exists in the
// server process), so that path is proven by the unit tests and a manual smoke.
func runIntegrationResetTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Integration API (chatbot API-key channel)")

	admin := client.WithToken(data.AdminToken)

	var createdID uint
	var key string

	reporter.RunTest(flowIntegrationAPI, "Admin creates an API key", func() error {
		var out CreateAPIKeyResponse
		if _, err := admin.PostInto("/api/v1/admin/api-keys", CreateAPIKeyRequest{Name: "itest chatbot"}, &out); err != nil {
			return err
		}
		if out.Key == "" {
			return fmt.Errorf("create returned an empty plaintext key")
		}
		if out.KeyPrefix != out.Key[:12] {
			return fmt.Errorf("key_prefix %q != first 12 of key %q", out.KeyPrefix, out.Key[:12])
		}
		createdID = out.ID
		key = out.Key
		return nil
	})

	machine := NewAPIClient(cfg.BaseURL)
	machine.Headers = map[string]string{"X-API-Key": key}

	reporter.RunTest(flowIntegrationAPI, "Integration call without X-API-Key returns 401", func() error {
		anon := NewAPIClient(cfg.BaseURL)
		_, code, err := anon.PostExpectError("/api/v1/integration/password-reset/otp", IntegrationOTPRequest{Phone: "0987654321"})
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("status %d, want 401", code)
		}
		return nil
	})

	reporter.RunTest(flowIntegrationAPI, "Integration call with garbage key returns 401", func() error {
		bad := NewAPIClient(cfg.BaseURL)
		bad.Headers = map[string]string{"X-API-Key": "ttk_garbage"}
		_, code, err := bad.PostExpectError("/api/v1/integration/password-reset/otp", IntegrationOTPRequest{Phone: "0987654321"})
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("status %d, want 401", code)
		}
		return nil
	})

	reporter.RunTest(flowIntegrationAPI, "OTP for unknown phone reports account_not_found", func() error {
		var out IntegrationOTPResponse
		if _, err := machine.PostInto("/api/v1/integration/password-reset/otp", IntegrationOTPRequest{Phone: uniqueMobile()}, &out); err != nil {
			return err
		}
		if out.Found || out.OTPSent {
			return fmt.Errorf("found=%v otp_sent=%v, want false/false", out.Found, out.OTPSent)
		}
		if out.FailureReason == nil || *out.FailureReason != "account_not_found" {
			return fmt.Errorf("failure_reason = %v, want account_not_found", out.FailureReason)
		}
		return nil
	})

	reporter.RunTest(flowIntegrationAPI, "OTP with a malformed phone returns 400", func() error {
		_, code, err := machine.PostExpectError("/api/v1/integration/password-reset/otp", IntegrationOTPRequest{Phone: "034090005032"})
		if err != nil {
			return err
		}
		if code != 400 {
			return fmt.Errorf("status %d, want 400", code)
		}
		return nil
	})

	reporter.RunTest(flowIntegrationAPI, "Verify with a garbage session returns 401 or 429", func() error {
		_, code, err := machine.PostExpectError("/api/v1/integration/password-reset/verify", map[string]string{
			"session_id": "garbage-session",
			"code":       "123456",
		})
		if err != nil {
			return err
		}
		if code != 401 && code != 429 {
			return fmt.Errorf("status %d, want 401 or 429", code)
		}
		return nil
	})

	reporter.RunTest(flowIntegrationAPI, "Employee lookup for unknown phone reports found=false", func() error {
		var out struct {
			Found bool `json:"found"`
		}
		if _, err := machine.PostInto("/api/v1/integration/employee/lookup", IntegrationOTPRequest{Phone: uniqueMobile()}, &out); err != nil {
			return err
		}
		if out.Found {
			return fmt.Errorf("found=true, want false")
		}
		return nil
	})

	reporter.RunTest(flowIntegrationAPI, "Revoked key can no longer authenticate", func() error {
		if createdID == 0 {
			return fmt.Errorf("no key created")
		}
		if _, code, err := admin.Delete(fmt.Sprintf("/api/v1/admin/api-keys/%d", createdID)); err != nil {
			return err
		} else if code != 200 {
			return fmt.Errorf("revoke status %d, want 200", code)
		}
		_, code, err := machine.PostExpectError("/api/v1/integration/password-reset/otp", IntegrationOTPRequest{Phone: "0987654321"})
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("post-revoke status %d, want 401", code)
		}
		return nil
	})

	if len(data.Partners) == 0 {
		reporter.Skip(flowIntegrationAPI, "Non-admin cannot create keys", "no partner token discovered")
		return
	}
	reporter.RunTest(flowIntegrationAPI, "Non-admin cannot create keys (403)", func() error {
		partner := client.WithToken(data.Partners[0].Token)
		_, code, err := partner.PostExpectError("/api/v1/admin/api-keys", CreateAPIKeyRequest{Name: "nope"})
		if err != nil {
			return err
		}
		if code != 403 {
			return fmt.Errorf("status %d, want 403", code)
		}
		return nil
	})
}
