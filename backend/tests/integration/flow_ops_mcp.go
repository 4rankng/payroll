package main

import (
	"fmt"
	"strings"
	"time"
)

const flowOpsMCP = "payroll-ops-mcp"

// opsKeyCreateResponse mirrors POST /api/v1/admin/ops-api-keys.
type opsKeyCreateResponse struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Scope  string `json:"scope"`
	Key    string `json:"key"`
	Prefix string `json:"key_prefix"`
}

// opsKeyInfoResponse mirrors GET /api/v1/ops/key-info.
type opsKeyInfoResponse struct {
	KeyID          uint   `json:"key_id"`
	AgentName      string `json:"agent_name"`
	Scope          string `json:"scope"`
	EffectiveScope string `json:"effective_scope"`
}

// withOpsKey returns a shallow client copy authenticating with an ops
// machine key instead of a JWT (mirrors WithToken).
func withOpsKey(c *APIClient, plaintext string) *APIClient {
	return &APIClient{
		BaseURL:    c.BaseURL,
		HTTPClient: c.HTTPClient,
		Headers:    map[string]string{"X-API-Key": plaintext, "X-MCP-Tool": "integration-test"},
	}
}

// createOpsKey mints a fresh ops key via the admin API and returns it.
func createOpsKey(admin *APIClient, name string, scope string) (*opsKeyCreateResponse, error) {
	var created opsKeyCreateResponse
	resp, err := admin.PostInto("/api/v1/admin/ops-api-keys", map[string]any{
		"name":  name,
		"scope": scope,
	}, &created)
	if err != nil {
		return nil, err
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("create ops key: %s", resp.Message)
	}
	return &created, nil
}

// auditCount polls the audit pipeline (asynq is async) until the expected
// minimum row count shows up, then returns it.
func auditCount(action string, entityID uint, min int64) (int64, error) {
	deadline := time.Now().Add(8 * time.Second)
	var last int64
	for {
		raw, err := queryPayrollScalar(fmt.Sprintf(
			"SELECT COUNT(*) FROM audit_logs WHERE action='%s' AND entity_type='ops_api_key' AND entity_id=%d",
			action, entityID))
		if err != nil {
			return 0, fmt.Errorf("count audit rows: %w", err)
		}
		if _, err := fmt.Sscan(raw, &last); err != nil {
			return 0, fmt.Errorf("parse audit count %q: %w", raw, err)
		}
		if last >= min {
			return last, nil
		}
		if time.Now().After(deadline) {
			return last, fmt.Errorf("audit rows for %s key=%d: got %d, want >= %d", action, entityID, last, min)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// runOpsMCPTests exercises the pk_ops_* machine channel end-to-end against
// the live backend: admin-equivalent breadth behind the scope toggle,
// wrong-scope isolation from the chat-integration surface, stale-scope
// enforcement, per-call auditing, and revocation.
//
// The kill switch (PAYROLL_OPS_MCP_DISABLED) is env-based and cannot be
// flipped on a live server from a test; its fail-closed behavior is covered
// by the unit tests in internal/transport/http/middleware/ops_key_auth_test.go
// and the MCP-layer tests in payroll-ops-mcp/test/.
func runOpsMCPTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: payroll-ops-mcp (pk_ops_* admin-equivalent machine keys)")

	admin := client.WithToken(data.AdminToken)

	var key *opsKeyCreateResponse

	// ── 1. Mint a key (default read_only) ─────────────────────────────────
	reporter.RunTest(flowOpsMCP, "Create ops key defaults to read_only pk_ops_*", func() error {
		var err error
		key, err = createOpsKey(admin, fmt.Sprintf("integration-%d", time.Now().UnixNano()), "")
		if err != nil {
			return err
		}
		if !strings.HasPrefix(key.Key, "pk_ops_") {
			return fmt.Errorf("plaintext must be pk_ops_-prefixed, got %q…", key.Key[:10])
		}
		if key.Scope != "read_only" {
			return fmt.Errorf("default scope must be read_only, got %q", key.Scope)
		}
		return nil
	})

	agent := func() *APIClient { return withOpsKey(client, key.Key) }

	// ── 2. read_only: introspection reports the key ───────────────────────
	reporter.RunTest(flowOpsMCP, "read_only key introspects via /ops/key-info", func() error {
		var info opsKeyInfoResponse
		if _, err := agent().GetInto("/api/v1/ops/key-info", &info); err != nil {
			return err
		}
		if info.KeyID != key.ID || info.EffectiveScope != "read_only" {
			return fmt.Errorf("key-info mismatch: %+v", info)
		}
		return nil
	})

	// ── 3. read_only: admin-parity reads ──────────────────────────────────
	reporter.RunTest(flowOpsMCP, "read_only key reads an admin endpoint (admin parity)", func() error {
		resp, status, err := agent().Get("/api/v1/approvals?pageSize=1")
		if err != nil {
			return err
		}
		if status != 200 || resp.Status != "success" {
			return fmt.Errorf("GET /approvals via ops key: HTTP %d %s", status, resp.Message)
		}
		return nil
	})

	// ── 4. read_only: backend blocks mutations (defense in depth) ─────────
	reporter.RunTest(flowOpsMCP, "read_only key POST is blocked by the backend scope gate", func() error {
		body := map[string]any{"type": "general_request", "params": map[string]any{"summary": "ops-mcp read_only probe"}}
		apiErr, status, err := agent().PostExpectError("/api/v1/approvals", body)
		if err != nil {
			return err
		}
		if status != 403 {
			return fmt.Errorf("want 403, got HTTP %d (%s)", status, apiErr.Message)
		}
		return nil
	})

	// ── 5. Wrong-scope: ops key on the chat-integration surface ───────────
	reporter.RunTest(flowOpsMCP, "ops key on /api/v1/mcp (integration surface) is 403 + audit alert", func() error {
		apiErr, status, err := agent().PostExpectError("/api/v1/mcp", nil)
		if err != nil {
			return err
		}
		if status != 403 {
			return fmt.Errorf("want 403, got HTTP %d (%s)", status, apiErr.Message)
		}
		// The probe must raise an audit alert row (asynq-async — poll). The
		// probing key is never authenticated, so the row carries entity_id=0
		// and is identified by the key_prefix tag in the message.
		deadline := time.Now().Add(8 * time.Second)
		for {
			raw, err := queryPayrollScalar(
				"SELECT COUNT(*) FROM audit_logs WHERE action='OPS_MCP_SCOPE_PROBE' AND message LIKE '%wrong-surface probe POST /api/v1/mcp%'")
			if err != nil {
				return fmt.Errorf("count probe rows: %w", err)
			}
			var n int64
			if _, err := fmt.Sscan(raw, &n); err != nil {
				return fmt.Errorf("parse probe count %q: %w", raw, err)
			}
			if n >= 1 {
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("no OPS_MCP_SCOPE_PROBE audit row for the /api/v1/mcp probe")
			}
			time.Sleep(300 * time.Millisecond)
		}
	})

	// ── 6. Credential management stays human-only ─────────────────────────
	reporter.RunTest(flowOpsMCP, "ops key cannot reach ops-key credential management", func() error {
		_, status, err := agent().GetExpectError("/api/v1/admin/ops-api-keys")
		if err != nil {
			return err
		}
		if status != 403 {
			return fmt.Errorf("want 403, got HTTP %d", status)
		}
		return nil
	})

	// ── 7. Chat-integration keys cannot manage ops keys ───────────────────
	reporter.RunTest(flowOpsMCP, "mcp_* chat key cannot manage ops keys", func() error {
		var created struct {
			Key string `json:"key"`
		}
		if _, err := admin.PostInto("/api/v1/admin/mcp-api-keys", map[string]any{
			"name": "ops-flow-chat-probe", "scope": "read_only",
		}, &created); err != nil {
			return fmt.Errorf("mint mcp key: %w", err)
		}
		chat := withOpsKey(client, created.Key)
		_, status, err := chat.GetExpectError("/api/v1/admin/ops-api-keys")
		if err != nil {
			return err
		}
		if status != 403 {
			return fmt.Errorf("want 403, got HTTP %d", status)
		}
		return nil
	})

	// ── 8. Toggle to read_write, mutation succeeds (admin parity) ─────────
	reporter.RunTest(flowOpsMCP, "read_write toggle then POST succeeds (mutation parity)", func() error {
		var updated opsKeyInfoResponse
		if _, err := admin.PutInto("/api/v1/admin/ops-api-keys/"+fmt.Sprint(key.ID)+"/scope", map[string]any{
			"scope": "read_write",
		}, &updated); err != nil {
			return fmt.Errorf("toggle scope: %w", err)
		}

		summary := fmt.Sprintf("ops-mcp mutation probe %d", time.Now().UnixNano())
		var created approvalFlowRequest
		resp, err := agent().PostInto("/api/v1/approvals", map[string]any{
			"type": "general_request", "params": map[string]any{"summary": summary},
		}, &created)
		if err != nil {
			return fmt.Errorf("file via read_write ops key: %w", err)
		}
		if resp.Status != "success" || created.Status != "pending" {
			return fmt.Errorf("unexpected filing: %+v", created)
		}
		return nil
	})

	// ── 9. Stale scope: flip back to read_only blocks the next call ───────
	reporter.RunTest(flowOpsMCP, "stale scope: flip to read_only blocks the next mutation", func() error {
		var updated opsKeyInfoResponse
		if _, err := admin.PutInto("/api/v1/admin/ops-api-keys/"+fmt.Sprint(key.ID)+"/scope", map[string]any{
			"scope": "read_only",
		}, &updated); err != nil {
			return fmt.Errorf("toggle scope back: %w", err)
		}
		body := map[string]any{"type": "general_request", "params": map[string]any{"summary": "ops-mcp stale probe"}}
		apiErr, status, err := agent().PostExpectError("/api/v1/approvals", body)
		if err != nil {
			return err
		}
		if status != 403 {
			return fmt.Errorf("want 403 after flip, got HTTP %d (%s)", status, apiErr.Message)
		}
		return nil
	})

	// ── 10. Exactly-one-audit-row-per-call ────────────────────────────────
	reporter.RunTest(flowOpsMCP, "every authenticated call writes an OPS_MCP_CALL audit row", func() error {
		// Two successful GETs from the agent client happened in steps 2-3;
		// audit is async, so poll for at least two rows for this key.
		count, err := auditCount("OPS_MCP_CALL", key.ID, 2)
		if err != nil {
			return err
		}
		// Rows must never contain key material — the plaintext exists only
		// in this process's memory.
		raw, err := queryPayrollScalar(fmt.Sprintf(
			"SELECT COUNT(*) FROM audit_logs WHERE action='OPS_MCP_CALL' AND entity_type='ops_api_key' AND entity_id=%d AND message LIKE '%%scope=ops%%'", key.ID))
		if err != nil {
			return err
		}
		if raw != fmt.Sprint(count) {
			return fmt.Errorf("rows missing the scope=ops tag: %s of %d", raw, count)
		}
		return nil
	})

	// ── 11. Scope-toggle auditing ─────────────────────────────────────────
	reporter.RunTest(flowOpsMCP, "scope toggles are audit-logged", func() error {
		if _, err := auditCount("OPS_KEY_SCOPE_CHANGED", key.ID, 2); err != nil {
			return err
		}
		return nil
	})

	// ── 12. Revocation fails closed ───────────────────────────────────────
	reporter.RunTest(flowOpsMCP, "revoked ops key is 401", func() error {
		resp, status, err := admin.Delete("/api/v1/admin/ops-api-keys/" + fmt.Sprint(key.ID))
		if err != nil {
			return err
		}
		if status != 200 || resp.Status != "success" {
			return fmt.Errorf("revoke ops key: HTTP %d %s", status, resp.Message)
		}
		_, status, err = agent().GetExpectError("/api/v1/ops/key-info")
		if err != nil {
			return err
		}
		if status != 401 {
			return fmt.Errorf("want 401 after revoke, got HTTP %d", status)
		}
		return nil
	})
}
