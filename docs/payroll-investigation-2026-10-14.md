# Payroll error-rate investigation — 2026-10-14

**Alert:** 06:00 ICT 14/10/2026, last 24h — `GET /api/v1/mcp` 100.00% (160/160), `GET /api/v1/users` 43.75% (28/64), `POST /api/v1/mcp` 11.89% (105/883).

**Companion document:** [`docs/incident-2026-10-14-payroll-errors.md`](incident-2026-10-14-payroll-errors.md) — full timeline, log excerpts, and fix details. This file is the condensed investigation record. All timestamps ICT (app clock; server clock is UTC — source of some confusion while correlating).

## 1. Error-rate anatomy (window 13/10 06:00 → 14/10 06:00 ICT, from prod `api_metrics`)

| Endpoint | Status | Count | When | Still occurring 14/10 evening? |
|---|---|---|---|---|
| GET /api/v1/mcp | 401 | 136 | 13/10 10:45–10:46 (8) + 17:51–18:06 (128) | No — stopped 18:06:02 |
| GET /api/v1/mcp | 405 | 26 | 13/10 18:06 onward, 2/h | **Yes — by design** |
| HEAD /api/v1/mcp | 401 / 405 | 22 / 26 | same windows | No / Yes |
| POST /api/v1/mcp | 401 | 105 | same storm as GET 401 | No |
| POST /api/v1/mcp | 200/202 | 431+17, all from 18:06 | zero successes before | Yes, healthy (~1–1.5/min) |
| GET /api/v1/users | 500 | 28 | 13/10 13:15:03–13:20:11 | No new 500s, **bug still latent** |

Global sweep: those 28 were the **only** ≥500 responses on any endpoint in the window.

## 2. Root causes

1. **GET+POST /mcp 401s — machine agent ran with no valid MCP key for ~20 h.** `/api/v1/mcp` (shipped 12/10 evening, `09731edc`) authenticates via `X-API-Key` against `mcp_api_keys` (SHA-256 hash, unrevoked, creator active → else 401). The table's **only row ever** is key "Hermes" (`mcp_zEcn7RDW…`, read_only, created by admin `frankng` 13/10 **17:43:37**). First 401s: 12/10 21:53 — correlates with the surface's **initial rollout**, not a later deploy or rotation. The agent host (159.26.115.182 = 2073/2096 of all /mcp traffic; remainder is internet scan noise) retried keyless until the operator minted Hermes (17:43) and installed it (~18:06). The "key expired" hypothesis is **disproved** — no key existed to expire. The separate TingHire integration key (`api_keys` id 3) was reactivated 13/10 and is healthy.
2. **GET/HEAD /mcp 405s (ongoing) — protocol-correct, monitoring artifact.** The handler is go-sdk Streamable HTTP with `Stateless: true`; the SDK returns 405 for standalone GET/HEAD SSE-stream requests by design (v1.8.0 `streamable.go` L138). The client's 30-min GET+HEAD liveness probes pass key auth and are answered 405 — counted as "errors" by api_metrics, so `GET /mcp` shows 100% forever on a healthy endpoint.
3. **GET /users 500s — unwhitelisted sort column reached SQL.** Commit `2be64b71` (13/10 10:11) added the computed `current_project_names` field and a sortable "Dự án" column; `ListWithFilters` only shape-sanitized `sortBy` before ORDER BY. Prod error: `Error 1054 (42S22): Unknown column 'current_project_names' in 'order clause'`. One admin (`anhbh`, 14.171.92.248 — 47/59 of GET /users traffic) triggered all 28 by clicking the sort. **Still live in prod** until the fix deploys.

## 3. Key evidence

- `api_metrics`/`api_endpoints` hourly aggregation: 401 storm 17:51:12–18:06:02; 405 cadence 2/h from 18:06:05 through 14/10 19:14; /users 500 window 13:15:03–13:20:11 with `user_id=2` on all 28.
- Gin access logs: storm IP 159.26.115.182; admin mcp-api-keys page loaded 17:47/17:53/18:03/18:05; the /users 500 requests carry `sortBy=current_project_names`.
- GORM error log: the exact MySQL 1054 line + `ListUsers "failed to list users with filters"` + 500 Gin line (quoted in companion doc §3.3).
- `mcp_api_keys`: single row (Hermes, created 13/10 17:43:37, never revoked). `api_keys` id 3 active, last_used 13/10 13:12.
- Success POSTs on 13/10 begin **only** at 18:06 (86 in the 18:00 hour, zero before) — no phantom earlier key needed.
- Repro on local builds: `main` → 500 on `sortBy=current_project_names`; fix branch → 200.

## 4. Recommended fixes

| Fix | Type | Status |
|---|---|---|
| Sort-column allowlist (`SanitizeSortColumnAgainst`) in users list + disable sorting on the computed frontend column | Code | **Done** on `fix/users-sort-computed-field-500` (`13cc22ef`), pushed, **not merged** — awaiting gate/merge approval |
| Exclude `/api/v1/mcp` GET/HEAD 405 from error-rate alerting (or stop the client's GET/HEAD probes / health-check via POST) | Monitoring | Proposed only |
| Key-rotation runbook: install new MCP key into the client together with minting (the 13/10 gap was 15 min; 12–13/10 was ~20 h) | Ops | Proposed only |
| Observability gaps: audit-log MCP-key create/revoke (currently unaudited — timeline had to be inferred); consider logging failed-key prefix on /mcp 401; `MCPKeyAuthMiddleware` doesn't stamp `last_used_at` | Code/ops | Proposed only |

**Data integrity: none affected.** All failures were auth/read-path (401 pre-handler, 405 transport, 500 on a read-only SELECT); no writes or money movement failed; wallet sync ran clean through the window.

## 5. Verification state

- Backend unit tests pass (`TestSanitizeSortColumnAgainst` et al.); frontend type-check + lint clean.
- Full integration suite vs branch build: 402 tests — auth-user flow **100% pass** incl. new regression case. 13 failures triaged as pre-existing/environmental (test payload omits now-required `CreateAPIKeyRequest.Scope` + 7 cascading auth checks; asset-binary storage; timesheet-seed cancel test). None of the failing flows call `/api/v1/users`; details in the commit message of `13cc22ef`.
- Prod claims are DB/API-verified from metrics tables + container logs; the admin-UI click path was not driven in a browser (rung 2, not rung 3).
