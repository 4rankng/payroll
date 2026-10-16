package main

import (
	"fmt"
	"strings"
	"time"
)

const flowMCPAPIKeys = "mcp-api-keys"

// mcpKeyCreateResponse mirrors POST /api/v1/admin/mcp-api-keys.
type mcpKeyCreateResponse struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Scope  string `json:"scope"`
	Key    string `json:"key"`
	Prefix string `json:"key_prefix"`
}

// mcpKeyScopeResponse mirrors PUT /api/v1/admin/mcp-api-keys/:id/scope.
type mcpKeyScopeResponse struct {
	ID             uint   `json:"id"`
	Scope          string `json:"scope"`
	ScopeUpdatedAt string `json:"scope_updated_at"`
}

// runMCPAPIKeyTests exercises the mcp_* chat-integration key channel: the
// admin scope toggle (read_only <-> full) on the SAME key with no
// revoke/recreate, next-call enforcement of the new scope (the transport gate
// re-reads the scope from the database on every request — no cache to
// invalidate), the closed scope vocabulary, revocation being terminal, and
// scope-toggle auditing.
func runMCPAPIKeyTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: MCP API keys (mcp_* scope toggle)")

	admin := client.WithToken(data.AdminToken)

	var key *mcpKeyCreateResponse

	// ── 1. Mint a read_only key ───────────────────────────────────────────
	reporter.RunTest(flowMCPAPIKeys, "create mcp key with read_only scope", func() error {
		var err error
		key, err = createMCPKey(admin, fmt.Sprintf("scope-toggle-%d", time.Now().UnixNano()), "read_only")
		if err != nil {
			return err
		}
		if !strings.HasPrefix(key.Key, "mcp_") {
			return fmt.Errorf("plaintext must be mcp_-prefixed, got %q…", key.Key[:4])
		}
		if key.Scope != "read_only" {
			return fmt.Errorf("scope must be read_only, got %q", key.Scope)
		}
		return nil
	})

	// The chat key authenticates with X-API-Key only (no X-MCP-Tool header).
	agent := func() *APIClient {
		return &APIClient{
			BaseURL:    client.BaseURL,
			HTTPClient: client.HTTPClient,
			Headers:    map[string]string{"X-API-Key": key.Key},
		}
	}

	// ── 2. read_only: GETs work (admin parity) ────────────────────────────
	reporter.RunTest(flowMCPAPIKeys, "read_only key reads an admin endpoint", func() error {
		resp, status, err := agent().Get("/api/v1/projects?pageSize=1")
		if err != nil {
			return err
		}
		if status != 200 || resp.Status != "success" {
			return fmt.Errorf("GET /projects via mcp key: HTTP %d %s", status, resp.Message)
		}
		return nil
	})

	// ── 3. read_only: mutations are blocked by the breadth gate ───────────
	reporter.RunTest(flowMCPAPIKeys, "read_only key POST is blocked (403)", func() error {
		body := map[string]any{"type": "general_request", "params": map[string]any{"summary": "mcp read_only probe"}}
		apiErr, status, err := agent().PostExpectError("/api/v1/approvals", body)
		if err != nil {
			return err
		}
		if status != 403 {
			return fmt.Errorf("want 403, got HTTP %d (%s)", status, apiErr.Message)
		}
		return nil
	})

	// ── 4. Toggle to full: same key id, no revoke/recreate ────────────────
	reporter.RunTest(flowMCPAPIKeys, "toggle to full keeps the same key and stamps scope_updated_at", func() error {
		var updated mcpKeyScopeResponse
		if _, err := admin.PutInto("/api/v1/admin/mcp-api-keys/"+fmt.Sprint(key.ID)+"/scope", map[string]any{
			"scope": "full",
		}, &updated); err != nil {
			return fmt.Errorf("toggle scope: %w", err)
		}
		if updated.ID != key.ID {
			return fmt.Errorf("toggle must keep the same key id: got %d, want %d", updated.ID, key.ID)
		}
		if updated.Scope != "full" {
			return fmt.Errorf("scope after toggle = %q, want full", updated.Scope)
		}
		if strings.TrimSpace(updated.ScopeUpdatedAt) == "" {
			return fmt.Errorf("scope_updated_at must be returned after a toggle, got %q", updated.ScopeUpdatedAt)
		}
		return nil
	})

	// ── 5. Next-call enforcement: the mutation now succeeds ───────────────
	reporter.RunTest(flowMCPAPIKeys, "next mutating call succeeds after the toggle (no cache staleness)", func() error {
		summary := fmt.Sprintf("mcp full mutation probe %d", time.Now().UnixNano())
		var created approvalFlowRequest
		resp, err := agent().PostInto("/api/v1/approvals", map[string]any{
			"type": "general_request", "params": map[string]any{"summary": summary},
		}, &created)
		if err != nil {
			return fmt.Errorf("file via full mcp key: %w", err)
		}
		if resp.Status != "success" || created.Status != "pending" {
			return fmt.Errorf("unexpected filing: %+v", created)
		}
		return nil
	})

	// ── 6. Flip back to read_only: the very next mutation is blocked ──────
	reporter.RunTest(flowMCPAPIKeys, "flip back to read_only blocks the next mutation", func() error {
		var updated mcpKeyScopeResponse
		if _, err := admin.PutInto("/api/v1/admin/mcp-api-keys/"+fmt.Sprint(key.ID)+"/scope", map[string]any{
			"scope": "read_only",
		}, &updated); err != nil {
			return fmt.Errorf("toggle scope back: %w", err)
		}
		body := map[string]any{"type": "general_request", "params": map[string]any{"summary": "mcp stale probe"}}
		apiErr, status, err := agent().PostExpectError("/api/v1/approvals", body)
		if err != nil {
			return err
		}
		if status != 403 {
			return fmt.Errorf("want 403 after flip, got HTTP %d (%s)", status, apiErr.Message)
		}
		return nil
	})

	// ── 7. Closed vocabulary: bogus scope values are rejected ─────────────
	reporter.RunTest(flowMCPAPIKeys, "scope toggle rejects values outside read_only/full", func() error {
		_, status, err := admin.Put("/api/v1/admin/mcp-api-keys/"+fmt.Sprint(key.ID)+"/scope", map[string]any{
			"scope": "revoked",
		})
		if err != nil {
			return err
		}
		if status != 400 {
			return fmt.Errorf("want 400 for bogus scope, got HTTP %d", status)
		}
		return nil
	})

	// ── 8. Scope toggles are audit-logged with old/new scope ──────────────
	reporter.RunTest(flowMCPAPIKeys, "scope toggles are audit-logged", func() error {
		if _, err := auditCount("mcp_api_key", "MCP_KEY_SCOPE_CHANGED", key.ID, 2); err != nil {
			return err
		}
		raw, err := queryPayrollScalar(fmt.Sprintf(
			"SELECT COUNT(*) FROM audit_logs WHERE action='MCP_KEY_SCOPE_CHANGED' AND entity_type='mcp_api_key' AND entity_id=%d AND message LIKE '%%old_scope=read_only new_scope=full%%'", key.ID))
		if err != nil {
			return err
		}
		if raw != "1" {
			return fmt.Errorf("audit rows carrying old/new scope: got %s, want 1", raw)
		}
		return nil
	})

	// ── 9. Revocation is terminal: no scope change afterwards ─────────────
	reporter.RunTest(flowMCPAPIKeys, "revoked mcp key cannot change scope (404) and is 401", func() error {
		resp, status, err := admin.Delete("/api/v1/admin/mcp-api-keys/" + fmt.Sprint(key.ID))
		if err != nil {
			return err
		}
		if status != 200 || resp.Status != "success" {
			return fmt.Errorf("revoke mcp key: HTTP %d %s", status, resp.Message)
		}
		_, status, err = admin.PutExpectError("/api/v1/admin/mcp-api-keys/"+fmt.Sprint(key.ID)+"/scope", map[string]any{
			"scope": "full",
		})
		if err != nil {
			return err
		}
		if status != 404 {
			return fmt.Errorf("want 404 toggling a revoked key, got HTTP %d", status)
		}
		_, status, err = agent().GetExpectError("/api/v1/projects?pageSize=1")
		if err != nil {
			return err
		}
		if status != 401 {
			return fmt.Errorf("want 401 after revoke, got HTTP %d", status)
		}
		return nil
	})
}

// createMCPKey mints a fresh mcp_* key via the admin API and returns it.
func createMCPKey(admin *APIClient, name string, scope string) (*mcpKeyCreateResponse, error) {
	var created mcpKeyCreateResponse
	resp, err := admin.PostInto("/api/v1/admin/mcp-api-keys", map[string]any{
		"name":  name,
		"scope": scope,
	}, &created)
	if err != nil {
		return nil, err
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("create mcp key: %s", resp.Message)
	}
	return &created, nil
}
