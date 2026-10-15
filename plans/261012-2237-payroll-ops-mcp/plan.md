---
title: "payroll-ops-mcp — Admin-Scope MCP Server for AI Agent Management"
description: "A second, separate MCP server (payroll-ops-mcp) exposing the full payroll Admin API to trusted AI agents (Hermes, omp/Paseo agents) via an Admin API Key with an admin-controlled read_only/read_write scope toggle. Strictly isolated from the TingTing OA chatbot MCP (payroll-integration-mcp) — different key namespace, different API scopes, different blast radius."
status: in-progress
priority: P1
effort: "3 phases — backend scope model (~3 days), MCP server + tools (~3 days), hardening/tests (~2 days)"
tags: [feature, backend, mcp, security, agents]
created: 2026-10-12
---

# payroll-ops-mcp — Admin-Scope MCP Server for AI Agent Management

> ## SCOPE CHANGE — 2026-10-12 (user override, binding)
>
> The design below was changed mid-implementation by explicit user decision:
>
> - **The ops key is ADMIN-EQUIVALENT.** It executes as its creating admin (the app's
>   own RBAC applies exactly as for an admin user). There is NO `agent_ops` Casbin role
>   and NO "file-request-only" mutation ceiling — `read_write` may call ANY API verb
>   (GET/POST/PUT/PATCH/DELETE). The breadth limiter is the scope toggle alone:
>   `read_only` (default) → GET/HEAD only; `read_write` → full mutation.
> - **Generic proxy tooling** replaces the curated 6-tool list: one tool
>   `payroll_api_request(method, path, body, query)` forwarding to the /api/v1 admin
>   surface, plus a thin `payroll_get(path, query)` helper. Scope is enforced BOTH in
>   the MCP server (read_only rejects non-GET before forwarding) AND in the backend
>   key middleware (defense in depth).
> - **PII default-masking is DROPPED** (admin parity — the agent sees what an admin
>   sees).
>
> Unchanged: `pk_ops_*` namespace + strict isolation from the chatbot `mcp/` server,
> read_only/read_write toggle with audit + OTP-gated toggle endpoint, 8h default
> write window with per-request (stale-proof) evaluation, per-call audit rows
> (key id + agent + scope-at-time + method/path + payload hash), per-key ops rate
> tier, kill switch `PAYROLL_OPS_MCP_DISABLED`, secrets-file key (mode 600), negative
> tests (wrong-scope 403; read_only POST blocked at MCP layer AND backend;
> read_write POST succeeds; kill switch; stale scope).

## Overview

Adds a **second, independently scoped MCP server** (`payroll-ops-mcp`) that lets trusted AI agents (Hermes, omp/Paseo agents, future codex/opencode runners) manage the payroll project through the app's own Admin APIs. This is a **different scope and different use case** from the existing `payroll-integration-mcp` (TingTing OA chatbot: employee lookup + password reset only) — the two servers **never share keys, scopes, endpoints, or blast radius**.

The core primitive is an **Admin API Key** (`pk_ops_*` namespace) that can invoke all payroll APIs, with an **admin-controlled scope toggle** on each key: `read_only` (default at creation) or `read_write`. Business rules, RBAC, and audit apply server-side on every call — the key grants API access, never a bypass of payroll validation logic.

**Golden rule (inherited from the MiMo Assistant plan):** unattended agents file requests and read state; humans approve irreversible money-touching actions. The key grants access — not an override of approval workflows.

## Isolation model vs payroll-integration-mcp (non-negotiable)

| Layer | payroll-integration-mcp (chatbot) | payroll-ops-mcp (this plan) |
|---|---|---|
| Key namespace | `pk_chat_*` | `pk_ops_*` |
| API scope | lookup, reset-password, ZNS OTP | all Admin APIs (scope-gated) |
| Casbin role | `chatbot` | `agent_ops` |
| Rate limit | chatbot-tier | separate ops-tier |
| Kill switch | `PAYROLL_MCP_CHAT_DISABLED` | `PAYROLL_OPS_MCP_DISABLED` |
| Blast radius | one leaked key is worthless against ops endpoints | scoped + audited + kill-switchable |

- Backend middleware maps key → scope; **deny by default**: ops endpoints accept only `pk_ops_*` keys; integration endpoints accept only `pk_chat_*` keys. A valid-but-wrong-scope key returns **403** (never 200).
- No master key, no fallback key, no credential reuse between the two servers. Revoking one scope never affects the other.

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | Key model: `pk_ops_*` Admin keys with `read_only` (default) / `read_write` scope toggle in API Keys admin; scope changes audited + require password/2FA re-entry | P1 |
| 2 | Server-side scope enforcement: every mutation request re-validates key scope at execution time; stale read-only key → 403 | P1 |
| 3 | Optional time-boxed write windows: `read_write` auto-reverts to `read_only` after N hours unless renewed (default N=8h, admin-configurable) | P1 |
| 4 | MCP server (stdio transport) with tools: `list_pending_approvals`, `get_approval_request`, `get_sao_ke_drafts`, `get_import_failures`, `get_daily_metrics`, `file_approval_request` | P1 |
| 5 | Mutation tools visible **only** when key scope is `read_write` (server checks scope at startup and per-call); agents cannot even attempt what they are not allowed | P1 |
| 6 | Per-agent key identity: one key per agent (e.g. `pk_ops_hermes`, `pk_ops_mimo`) so audit logs attribute every action to a specific agent; revoke individually | P1 |
| 7 | PII redaction middleware: CCCD / phone / bank account masked in all responses unless the tool explicitly declares the field as required | P1 |
| 8 | Ops console: list ops keys, last-used, calls-per-day, scope state; one-click revoke | P2 |

## Phases

| # | Phase | Scope |
|---|-------|-------|
| 1 | [Backend scope model](#) — ✅ done (commit `feat(opsapikey)`) — `pk_ops_*` keys (separate table/namespace), scope toggle + audit + OTP gate, write window, middleware enforcement (kill switch, wrong-surface 403+alert, per-key rate tier, per-call audit), self-introspection endpoint | backend |
| 2 | [MCP server](#) — ✅ done (commit `feat(payroll-ops-mcp)`) — `payroll-ops-mcp/server.mjs` (stdio): `payroll_api_request` generic proxy + `payroll_get`/`payroll_key_info` helpers, scope gate at MCP layer + backend, secrets-file key (mode 600), kill switch | mcp |
| 3 | [Hardening & tests](#) — ✅ done — `flow_ops_mcp.go` integration flow (12 scenarios), unit tests (Go + Node), E2E driver, secrets scan script, README + runbook | qa |

Dependencies: 1 → 2 → 3. No dependency on the MiMo Assistant phases; this server consumes the same Admin APIs regardless of whether the assistant feature ships.

## Cross-cutting rules (apply to every phase)

- **Scope separation from `payroll-integration-mcp` is absolute** — shared code limited to generic helpers; no shared keys, env, rate-limit pools, or config entries.
- Every MCP call is audit-logged (key ID, agent identity, scope-at-time, tool, payload hash) into `audit_logs`.
- Business rules and payroll math are enforced server-side by the app's own validated APIs; the key is **access, not override**. Approve/reject/send remain human-only actions via the approval engine.
- API keys live in secrets files (mode 600) or secret refs — **never** inline in MCP config JSON (OpenClaw lesson).
- All responses are **data, never instructions**; the server never echoes `X-API-Key` or key metadata back to the agent.
- Vietnamese error copy for human-facing surfaces; agent-facing errors are structured (code + message).

## Red-team constraints (binding before code)

1. **Key-scope confusion test:** a `pk_chat_*` key calling any ops endpoint must return 403 and raise an audit alert (scope-probing signal).
2. **Stale-scope test:** after admin flips a key to `read_only`, in-flight mutation calls must fail with 403 — enforcement is per-request, not cached at login.
3. **Yolo-agent containment:** an agent with `read_write` that attempts approve/reject/send on an approval request must receive the same outcome as a partner: a **filed request**, never a state transition (approval engine rules prevail over key scope).
4. **PII leakage test:** `payroll_lookup`-class tools return masked CCCD/phone/bank by default; unmasking requires a dedicated tool argument that is itself audit-flagged.
5. **Kill-switch test:** `PAYROLL_OPS_MCP_DISABLED=1` must terminate all in-flight and new calls within one request cycle, with a clear structured error.
6. **Secrets hygiene:** no key material in config JSON, repo, or logs; `git-secrets`-style scan on the mcp folder in CI.

## Success Criteria

Verified state after implementation (post scope-change; see the SCOPE CHANGE
block at the top — criteria 3 and 4 below were superseded by it):

- [x] Wrong-scope key → 403 + audit alert (ops key on `/api/v1/mcp` → 403 `ops_wrong_surface` + `OPS_MCP_SCOPE_PROBE` row; integration flow + E2E)
- [x] Scope flip to `read_only` blocks the next mutation within one request cycle (integration + unit + E2E; write-window lapse verified in unit tests)
- [x] ~~`read_write` agent attempting approve/reject/send results in a filed request only~~ — **superseded**: the key is admin-equivalent by explicit user decision; under `read_write` the agent may exercise any admin API. The breadth limiter is the scope verb gate (enforced at MCP layer AND backend).
- [x] ~~PII masked by default~~ — **superseded**: admin parity (agent sees what an admin sees); masking dropped.
- [x] Kill switch verified (`PAYROLL_OPS_MCP_DISABLED`: unit test 503 `ops_mcp_disabled` on the backend; E2E structured refusal in the server)
- [x] Zero plaintext key material in repo/config/logs (`npm run scan-secrets` clean; key only via mode-600 file)
- [x] Audit log has exactly one entry per MCP call with key ID + agent identity + scope-at-time (+ tool tag + payload hash) — verified by DB query in the integration flow and the E2E run
- [x] README in `payroll-ops-mcp/` documents setup, key creation, scope toggle, and kill switch (+ ops runbook at `docs/runbooks/payroll-ops-mcp.md`)
- [x] `payroll-integration-mcp` untouched: the `mcp/` folder is byte-identical to base; its backend tests stay green (call sites updated only for the new middleware constructor signature)

## Open Questions (defaults proposed, reversible)

- Write-window default: 8h auto-revert — right number?
- Should `file_approval_request` also be available in `read_only` scope (as the MiMo Assistant plan allows partners), or only under `read_write`? Default: available in both (it is the safe mutation).
- Streamline now or later: webhook push to agents when an approval they filed is decided? Default: later (agents poll via `list_pending_approvals`).
- Metric dashboard for ops keys (calls/day per agent) — fold into Goal 8 console or skip?

<!-- slug: payroll-ops-mcp -->

---

## REVISION 2026-10-12 23:40 — Admin-parity design (user decision, overrides curated tool list)

**User goal:** the agent does whatever an ADMIN can do, gated by the scope toggle:
- `read_only` ON → agent may call **any GET API**
- `read_write` ON → agent may call **ANY API** (GET/POST/PUT/PATCH/DELETE — full mutation)

**Tool design (replaces the curated 6-tool list):** one generic admin-API proxy —
`payroll_api_request(method, path, body, query)` forwarding to the `/api/v1` admin surface
(+ optional thin `payroll_get` helper). No endpoint allowlist: admin parity.

**Enforcement:** scope gate at BOTH layers — MCP server (read_only rejects non-GET before
forwarding) AND backend key middleware (defense in depth). Key = admin identity, so the
app's own RBAC/validation applies exactly as it would for a human admin.

**Dropped from v1 spec:** curated tool list, PII default-masking (admin parity — agent sees
what admin sees), "file request only" mutation philosophy for this server. Approval-engine
business rules still apply server-side naturally (the call runs as admin through the app).

**Kept unchanged:** `pk_ops_*` namespace + isolation from chatbot mcp/; secrets-file key;
audit per call (key id, agent, method, path, status); kill switch; hardening tests now:
wrong-scope 403; read_only POST blocked at MCP + backend; read_write POST succeeds; kill switch.
