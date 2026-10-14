# Incident 2026-10-14: High error rates on /api/v1/mcp and /api/v1/users

**Alert:** 06:00 ICT 14/10/2026, last 24h — `GET /api/v1/mcp` 100.00% (160/160), `GET /api/v1/users` 43.75% (28/64), `POST /api/v1/mcp` 11.89% (105/883).
**Severity on triage:** low — no data integrity impact, no broad user-facing breakage. One latent admin-page bug (still live), one monitoring artifact (ongoing), one resolved key-configuration window (self-healed 13/10 18:06).
**Sources:** prod `api_metrics`/`api_endpoints` tables, `docker logs payroll-backend` (Gin access + GORM error logs), `mcp_api_keys`/`api_keys`/`audit_logs` tables, git history, code reading of the running build (`main` @ `00a8c563`).

> All timestamps below are **Asia/Ho_Chi_Minh (ICT)**. `api_metrics.called_at` and Gin log lines are ICT; the server clock is UTC. Alert counts vs measured counts differ by ≤2 requests at window boundaries (e.g. measured 162 GET /mcp errors vs alert 160); status anatomy is identical.

## 1. Status-code anatomy (alert window 13/10 06:00 → 14/10 06:00 ICT)

| Endpoint | Status | Count | Window | Still occurring at 14/10 19:25? |
|---|---|---|---|---|
| GET /api/v1/mcp | 401 | 136 | 13/10 10:45–10:46 (8) and 17:51–18:06 (128) | **No** — stopped 18:06:02 |
| GET /api/v1/mcp | 405 | 26 | 13/10 18:06 → 14/10 05:44 (2/h) | **Yes** — 2/h, by design |
| HEAD /api/v1/mcp | 401 | 22 | same storm as above | No |
| HEAD /api/v1/mcp | 405 | 26 | same 30-min cadence | Yes |
| POST /api/v1/mcp | 401 | 105 | 13/10 10:45 (8) + 17:51–18:06 (97) | **No** — stopped 18:06:02 |
| POST /api/v1/mcp | 200/202 | 431+17 from 13/10 18:06 onward; zero successes before | | Yes (healthy ~1–1.5/min) |
| GET /api/v1/users | 500 | 28 | 13/10 13:15:03–13:20:11 | No new 500s, but **bug still latent** |
| GET /api/v1/users | 200 | 36 | | Yes (healthy) |

Global sweep: those 28×500 were the **only** ≥500 responses on any endpoint in the whole window.

## 2. Timeline

| Time (ICT) | Event | Evidence |
|---|---|---|
| 12/10 22:46 | `09731edc` ships the new scoped API-key MCP server (`/api/v1/mcp`, Streamable HTTP, stateless) as part of the chatbot purge | git |
| 12/10 21:53 | First-ever /mcp errors: 2× POST 401 (client probing the fresh endpoint with no valid key) | api_metrics |
| 13/10 10:45:50–10:46:18 | 401 burst (8 GET + 8 POST + 2 HEAD) — MCP client retrying, still no valid key | api_metrics (container predates current one; access logs rotated away) |
| 13/10 11:27 | Backend deploy/restart (commits through `826c1aa4`, incl. `2be64b71` "add SDT and Du an columns" 10:11) | `docker inspect StartedAt` = 04:27:22Z |
| 13/10 13:15:03–13:20:11 | 28× GET /users **500** — admin `anhbh` (user 2, IP 14.171.92.248) clicks the new **Dự án** column sort repeatedly; switches back to `created_at` at 13:20:07 and gets 200s | Gin logs + api_metrics (`user_id=2` on all 28) |
| 13/10 17:43:37 | Admin `frankng` (user 1) mints MCP API key **"Hermes"** (`mcp_zEcn7RDW…`, read_only) — the first and only row ever in `mcp_api_keys` | `mcp_api_keys` table |
| 13/10 17:47–18:05 | Operator at 159.26.115.182 opens admin page `GET /api/v1/admin/mcp-api-keys` 4× (17:47, 17:53, 18:03, 18:05) | Gin logs |
| 13/10 17:51:12–18:06:02 | **401 storm**: the MCP client (same IP 159.26.115.182) retries GET+POST+HEAD every ~1–10 s with the old/absent key — 136 GET + 105 POST + 22 HEAD total | Gin logs + api_metrics |
| 13/10 18:06:02→05 | Hermes key pasted into the client: 401s stop dead; every POST from here is 200/202; GET+HEAD flip to **405** | api_metrics + Gin |
| 13/10 18:06 → now | Same client polls **GET+HEAD pairs every 30 min** (:14/:44 past the hour); each passes key auth and is answered **405 Method Not Allowed** by the stateless MCP handler | api_metrics (2/h through 14/10 19:14) |
| 14/10 06:00 | Alert fires; 24h lookback captures all of the above | — |

## 3. Root causes

### 3.1 GET+POST /mcp 401s — machine client ran without a valid MCP key for ~20 h
`/api/v1/mcp` authenticates via `X-API-Key` validated by `MCPKeyAuthMiddleware` → `mcpapikey.Service.Authenticate` (SHA-256 hash lookup in `mcp_api_keys`, key must be unrevoked, creator active; all failures indistinguishable 401 — `constants.MsgMCPKeyInvalidVN`).

The prime hypothesis ("MCP integration API key missing/expired after the 13/10 reminder") is **half right**: the key was indeed missing, but not *expired* — **no MCP key existed at all** until Hermes was minted 13/10 17:43:37 (it is the only row the table has ever had; no `revoked_at`, creator `frankng` active). The `/api/v1/mcp` surface shipped 12/10 evening with zero keys; the agent client at 159.26.115.182 ran keyless/stale until the operator minted Hermes (17:43) and installed it into the client (18:06). The unrelated **integration** API key (TingHire, `api_keys` id 3, `timesheet_read`) was reactivated 13/10 (`23bb7c8f`) and is healthy (`last_used_at` 13/10 13:12) — the reminder did not flag an expired prod key.

First occurrence: 12/10 21:53 — correlates with the **initial rollout** of the scoped MCP surface (`09731edc`), not with a later deploy or key rotation.

**Not one broken client hammering broadly:** since the 13/10 11:27 deploy, 2073 of 2096 /mcp requests came from 159.26.115.182 (the Hermes host); the rest are single internet scans (GCP/odd ranges) that got standard 401s. POST /mcp shares this exact root cause — same storm, same fix point in time.

### 3.2 GET/HEAD /mcp 405s (ongoing) — protocol-correct rejection, a monitoring artifact
The MCP handler is `mcp.NewStreamableHTTPHandler(..., &mcp.StreamableHTTPOptions{Stateless: true})` (`routes_mcp.go` / `mcp/server.go`). Per the go-sdk (v1.8.0, `streamable.go`: "In Stateless mode, GET and DELETE requests return 405 Method Not Allowed"), standalone SSE-stream GETs are unsupported by design and **must** be refused — the MCP transport treats 405 as "no server-initiated streams" (clients in non-strict mode handle it). The Hermes client's 30-minute GET+HEAD liveness probes therefore register as endpoint errors in `api_metrics` (any status ≥ 400 counts), producing a permanent "GET /mcp 100% errors" signal while the endpoint is actually healthy.

### 3.3 GET /users 500s — unwhitelisted sort field reached SQL ORDER BY
Exact prod error (Gin/GORM, 13/10 13:15:03):

```
Error 1054 (42S22): Unknown column 'current_project_names' in 'order clause'
SELECT * FROM `users` WHERE `users`.`deleted_at` IS NULL ORDER BY current_project_names ASC LIMIT ?
ERROR user.(*UserService).ListUsers "failed to list users with filters"
[GIN] 13:15:03 | 500 | GET "/api/v1/users?page=1&pageSize=20&sortBy=current_project_names&sortOrder=asc&offset=0"
```

`2be64b71` (13/10 10:11) added the computed `current_project_names` DTO field (batched post-query lookup) and a sortable "Dự án" column, but `UserRepository.ListWithFilters` only *shape*-sanitizes `sortBy` (`common.SanitizeSortColumn` — regex, no existence check) before interpolating it into `ORDER BY`. `users` has no such column → MySQL 1054 → 500. **The bug is still live in prod** (`main` @ `00a8c563` contains no fix); it recurs the moment any admin sorts that column. Verified reproducible on a local build of `main`.

Caller: one admin (`anhbh`, Viettel IP 14.171.92.248) — 47 of 59 GET /users? since the deploy; the 28 failures were all this one user clicking the broken sort. Not broad.

## 4. Data integrity

**None affected.** All three failure modes are auth/read-path: 401 (rejected before handlers), 405 (rejected by transport), 500 on a read-only `SELECT` (rolled back by GORM, nothing written). The 24h sweep shows zero other 5xx; wallet sync/reconciliation ran clean through the window (logs at 13:15 show `Wallet balance in sync`). No payroll, ledger, wallet, or disbursement operation failed.

## 5. Fixes

### Applied on branch `fix/users-sort-computed-field-500` (NOT merged, per incident protocol)
1. `backend/internal/infra/persistence/common/sort.go` — new `SanitizeSortColumnAgainst(sortBy, allowed, default)` allowlist sanitizer.
2. `backend/internal/infra/persistence/user_repository.go` — `ListWithFilters` now orders only by real `users` columns (`id, username, email, fullname, role, mobile, cccd, last_login, created_at, updated_at`); anything else (incl. `current_project_names`, `password`) falls back to `created_at`. Kills the 500 and stops non-sortable columns from ever reaching ORDER BY.
3. `frontend/src/config/user-table-columns.tsx` — `enableSorting: false` on the computed "Dự án" column (the server cannot sort it; the header was a live grenade).
4. Tests: `sort_test.go` unit cases (computed field, injection, case sensitivity); integration case in `flow_auth_user.go` (`GET /users?sortBy=current_project_names` must be 200). Verified on a local build of the branch: 200 (was 500), `sortBy=password` → 200 fallback, `sortBy=last_login` → 200 sorted.

### Proposed for prod (require approval — nothing applied)
- **Deploy the branch above** (only outstanding user-facing defect).
- **Monitoring:** exclude `/api/v1/mcp` GET/HEAD 405 from error-rate alerts (or alert on `status ≥ 400 AND NOT (path=/api/v1/mcp AND method IN (GET,HEAD) AND status=405)`). Otherwise "GET /mcp 100% errors" fires forever on a healthy endpoint. Alternatively/additionally: stop the client's 30-min GET+HEAD probes (they probe for a stream the stateless server will never open) or point its health check at `POST` initialize or `/api/healthz`.
- **Key-rotation runbook:** when rotating MCP keys, install the new key into the client *before/with* minting (this storm was a 15-min retry window between mint 17:43 and install 18:06; on 12–13/10 the window was ~20 h).
- **Observability gaps found:** MCP-key admin actions (create/revoke) are not written to `audit_logs` (timeline had to be inferred from table state + page loads); `/mcp` 401 responses carry no caller identity (no key prefix logged — by design, but consider logging the failed key *prefix*). `MCPKeyAuthMiddleware` does not stamp `last_used_at` (Hermes shows 18:50 from the REST branch only), so key-usage dashboards undercount.

## 6. Risk assessment

| Signal | User-facing? | Verdict |
|---|---|---|
| GET/POST /mcp 401 storm (12/10 21:53 → 13/10 18:06) | Only the machine agent (Hermes) — no human UI affected | Resolved; transient misconfiguration during rollout |
| GET/HEAD /mcp 405 (13/10 18:06 → ongoing) | No — protocol-correct, client functions normally over POST | **Monitoring-only artifact**; will keep the alert hot until the rule or the probe changes |
| GET /users 500 (13/10 13:15–13:20) | Yes — admin user list breaks when sorting "Dự án" | **Latent bug still in prod**; fix ready on branch, needs deploy |

## Verification coverage

| Claim | Rung | Evidence | Not covered |
|---|---|---|---|
| /users 500 anatomy + SQL error | DB/API VERIFIED | prod GORM log line quoted above; api_metrics rows; Gin access lines | — |
| /mcp 401 storm window, counts, single client | DB/API VERIFIED | api_metrics hourly rows; Gin logs (159.26.115.182 = 2073/2096) | the 10:45 burst's access logs (previous container, rotated); UA strings (not logged) |
| Key-not-expired finding | DB VERIFIED | `mcp_api_keys` single row Hermes (created 13/10 17:43:37, never revoked); `api_keys` id 3 active | who deleted/never-created earlier keys is unknowable (no audit rows) |
| 405 = stateless-by-design | CODE-READ + prod metrics | go-sdk v1.8.0 `streamable.go` L138; prod 405 cadence 2/h post-18:06 | none |
| Fix works (sortBy=current_project_names → 200) | DB/API VERIFIED (local build of branch, driven via curl with real login) | curl transcript: 500 on `main` build, 200 on fix build | browser click-through of the admin UI not driven (frontend change is a column-config flag, type-checked + linted) |
| Integration test case | RUN — full suite (402 tests) vs branch build | auth-user flow 100% PASS incl. new regression case (log line: `PASS List users sorted by computed field falls back to default`) | 13 unrelated suite failures are pre-existing/environmental: 3× test omits now-required `CreateAPIKeyRequest.Scope` (+7 cascading 401/403 checks), asset-binary storage missing, timesheet-seed-dependent cancel test. None of the failing flows call `/api/v1/users`; not re-verified on a separate main build |
