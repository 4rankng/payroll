package main

import (
	"encoding/json"
	"fmt"
)

const flowIntegrationZaloToken = "IntegrationZaloToken"

// runIntegrationZaloTokenTests exercises the chatbot's pull lane for the OA
// access token: the 401 guards and, with a valid machine key, the envelope
// shape for both server states (200 with a token, or 503 when payroll holds
// no usable token yet — the dev DB may legitimately be unconfigured).
func runIntegrationZaloTokenTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Integration Zalo Token (chatbot pull lane)")

	admin := client.WithToken(data.AdminToken)

	var keyID uint
	var key string
	reporter.RunTest(flowIntegrationZaloToken, "Admin creates an API key for the flow", func() error {
		var out CreateAPIKeyResponse
		if _, err := admin.PostInto("/api/v1/admin/api-keys", CreateAPIKeyRequest{Name: "itest zalo token"}, &out); err != nil {
			return err
		}
		if out.Key == "" {
			return fmt.Errorf("create returned an empty plaintext key")
		}
		keyID, key = out.ID, out.Key
		return nil
	})
	defer func() {
		if keyID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/admin/api-keys/%d", keyID))
		}
	}()

	reporter.RunTest(flowIntegrationZaloToken, "Token without X-API-Key returns 401", func() error {
		anon := NewAPIClient(cfg.BaseURL)
		_, code, err := anon.GetExpectError("/api/v1/integration/zalo/token")
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("status %d, want 401", code)
		}
		return nil
	})

	reporter.RunTest(flowIntegrationZaloToken, "Token with garbage key returns 401", func() error {
		bad := NewAPIClient(cfg.BaseURL)
		bad.Headers = map[string]string{"X-API-Key": "ttk_garbage"}
		_, code, err := bad.GetExpectError("/api/v1/integration/zalo/token")
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("status %d, want 401", code)
		}
		return nil
	})

	if key == "" {
		reporter.Skip(flowIntegrationZaloToken, "Authenticated token pull", "API key creation failed")
		return
	}
	machine := NewAPIClient(cfg.BaseURL)
	machine.Headers = map[string]string{"X-API-Key": key}

	reporter.RunTest(flowIntegrationZaloToken, "Token pull with a valid key returns a coherent envelope", func() error {
		resp, code, err := machine.Get("/api/v1/integration/zalo/token")
		if err != nil {
			return err
		}
		switch code {
		case 200:
			var out struct {
				AccessToken string `json:"access_token"`
			}
			if resp.Status != "success" {
				return fmt.Errorf("envelope status = %q, want success", resp.Status)
			}
			if err := json.Unmarshal(resp.Data, &out); err != nil {
				return fmt.Errorf("unmarshal data: %w", err)
			}
			if out.AccessToken == "" {
				return fmt.Errorf("200 with an empty access_token")
			}
			return nil
		case 503:
			if resp.Status != "error" {
				return fmt.Errorf("503 envelope status = %q, want error", resp.Status)
			}
			if resp.Message == "" {
				return fmt.Errorf("503 without the Vietnamese remediation message")
			}
			return nil
		default:
			return fmt.Errorf("status %d, want 200 or 503", code)
		}
	})
}
