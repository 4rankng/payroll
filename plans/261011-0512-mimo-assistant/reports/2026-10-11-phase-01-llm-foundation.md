# Phase 1: LLM Foundation — Implementation Report

> Date: 2026-10-11 · Plan: `plans/261011-0512-mimo-assistant/` · Status: **DONE_WITH_CONCERNS** (spike blocked on user input; 4 pre-existing unrelated frontend test failures)

## Scope delivered

MiMo LLM foundation: provider client behind a port, admin-managed sealed API key settings, test-connection, model routing config, admin settings UI (desktop + mobile). Zero changes to phases 2–7 surfaces. Working tree left uncommitted for review.

## Red-team constraint coverage

| Constraint | Status | Evidence |
|---|---|---|
| #1 `mimo.api_key` registered in `secret.ProtectedKeys` (sealed + masked) | DONE | `pkg/secret/keys.go`; masked in ALL 4 settings GET shapes (handler test); sealed at rest (`enc:v1:…` in MySQL, plaintext-leak query = 0) |
| Medium: client in `internal/infra/llm/` behind a port | DONE | `llm.Client` interface in `infra/llm/types.go`; app service (`app/services/llm`) depends on the port only |
| Medium: budget/circuit breaker on shared key | DONE (minimal) | `infra/llm/guard.go`: concurrency semaphore (4) + per-user Redis fixed window (20/min), wired in `bootstrap/services/init.go` |
| Designed Vietnamese error states (missing/invalid key, timeout) | DONE (settings surface) | stable outcome codes `missing_key|invalid_key|timeout|rate_limited|unavailable|busy|error` (`dto/settings.go`); UI panels + button-enable rules in `MimoConnectionSection.tsx`, unit-tested |

## Files created

| File | Lines | Purpose |
|---|---|---|
| `backend/internal/infra/llm/types.go` | 252 | `Client` port + chat/tool-call/image/stream-chunk structs + `Config` |
| `backend/internal/infra/llm/client.go` | 278 | OpenAI-compatible HTTP client (`chat.completions`, `GET /models`), bounded timeouts (30s/attempt, 2 retries, jittered backoff), Vietnamese error mapping |
| `backend/internal/infra/llm/errors.go` | 127 | `ErrorKind` taxonomy + `APIError` + Vietnamese `MessageFor` |
| `backend/internal/infra/llm/models.go` | 76 | Allowlist `mimo-v2.6-flash`/`mimo-v2.6-pro`; `mimo-v2.5*` rejected with "đã ngừng hỗ trợ" error; `ResolveModel` |
| `backend/internal/infra/llm/guard.go` | 148 | `Semaphore`, `FixedWindowLimiter` (Redis), `MemoryLimiter`, `Guard` |
| `backend/internal/infra/llm/{client,models,guard}_test.go` | 602 | httptest request/response parsing, error mapping, retry semantics, allowlist, guards (miniredis) |
| `backend/internal/app/services/llm/service.go` | 164 | App service: settings-driven client construction, model routing, guard enforcement, fallback-model retry on model-not-found |
| `backend/internal/app/services/llm/service_test.go` | 254 | missing key, retired model rejection, fallback retry, per-user rate limit, fail-open limiter, busy, timeout |
| `backend/internal/transport/http/handlers/settings/settings_test.go` | 340 | Masking across every settings GET shape; test-connection success + designed failures |
| `frontend/src/components/settings/MimoConnectionSection.tsx` | 452 | Admin UI: "Khóa bí mật" masked field, model routing selects, save + test, designed error states |
| `frontend/src/components/settings/MimoConnectionSection.test.tsx` | 259 | 13 tests: missing-key state, create/update upsert, dirty-state button rules, success + all error codes |
| `qa/scripts/drive-mimo-settings.cjs` | 191 | Puppeteer click-through driver (desktop + 390/320 mobile) |

## Files modified

| File | Change |
|---|---|
| `backend/internal/pkg/secret/keys.go` | **CRITICAL fix**: `MimoAPIKey = "mimo.api_key"` registered in `ProtectedKeys` |
| `backend/internal/infra/persistence/settings_repository_secrets_test.go` | +2 tests: `mimo.api_key` sealed at rest + backfill; allowlist registration pinned |
| `backend/internal/app/services/config/settings_config.go` | `mimo.*` keys, defaults, typed accessors (authoritative reads, no caching of secrets, allowlist + URL validation on read), write-validation parsers |
| `backend/internal/app/services/config/settings_config_test.go` | +6 tests: defaults, authoritative key reads, retired-model fallback, base-URL validation, write validation |
| `backend/internal/app/services/config/settings_service.go` | `validateBusinessSetting` cases for `mimo.*` (write-time rejection of `mimo-v2.5*` with Vietnamese error) — **addition beyond the phase's file list, required by "rejected at config validation"** |
| `backend/internal/app/dto/settings.go` | `MimoTestConnectionResponse` + outcome codes; `Setting.masked/configured` already present |
| `backend/internal/transport/http/handlers/settings/settings.go` | `TestMimoConnection` handler (admin-only route) |
| `backend/internal/transport/http/handlers/settings.go` | Constructor wiring for the LLM service |
| `backend/internal/app/bootstrap/services/init.go` | LLM service + shared-key guard wiring (semaphore 4, 20 calls/min/user Redis window) |
| `backend/internal/app/bootstrap/container.go` | Handler wiring |
| `backend/internal/app/bootstrap/routes_admin.go` | `POST /api/v1/settings/mimo/test-connection` (admin-only; partner/employee grants are GET-only in `casbin_policy.csv`) |
| `frontend/src/types/api/settings.types.ts` | `masked/configured` on `Setting`; `MimoTestConnectionResult` + codes; `MIMO_SETTING_KEYS`; `MIMO_MODELS` |
| `frontend/src/config/api.config.ts` | `settings.mimoTestConnection` endpoint |
| `frontend/src/services/api/settings.service.ts` | `testMimoConnection()` |
| `frontend/src/hooks/api/useSettings.ts` | `useTestMimoConnection` mutation |
| `frontend/src/components/settings/SettingsTabList.tsx` | `SETTINGS_TABS.mimo` ("Trợ lý AI", Bot icon) |
| `frontend/src/pages/admin/SettingsPage/index.tsx` | MiMo `TabsContent` (desktop) |
| `frontend/src/pages/mobile/admin/SettingsPage/index.tsx` | MiMo `TabsContent` (mobile — parity) |
| `frontend/src/pages/admin/SettingsPage/settings-page-parity.test.tsx` | MimoConnectionSection mock + parity case (both views) |

## Test results (exact commands + outcomes)

| Command | Outcome |
|---|---|
| `cd backend && go vet ./...` | pass |
| `cd backend && go test ./... -race -cover` | **all packages pass, 0 failures** (`infra/llm` 88.2% cov, `handlers/settings` 34.5%, `pkg/secret` 91.8%) |
| `cd backend && go test ./internal/infra/llm/ ./internal/app/services/llm/ -race -count=3` | pass ×3 (a test-helper data race found & fixed during the run) |
| `make api-test` | **362 total \| 328 passed \| 0 failed \| 34 skipped** (skips pre-existing/env-dependent) |
| `cd frontend && pnpm lint` | pass (1 pre-existing warning in `qa/drive-payment-history-ledger.cjs`) |
| `cd frontend && pnpm type-check` | pass |
| `cd frontend && pnpm test:run` | 873/877 tests pass (171/175 files); **4 failures pre-existing & unrelated** (`src/components/ui/{avatar,checkbox,input,select}.test.tsx` UU-restyle class contracts — test + source both unmodified from HEAD, so identical at HEAD) |
| `cd frontend && pnpm vitest run src/components/settings/MimoConnectionSection.test.tsx … settings-page-parity.test.tsx` | 51/51 pass (13 new) |

## Designed error-state contract (settings surface)

| Outcome code | User sees | Buttons enabled |
|---|---|---|
| `missing_key` | amber: "Chưa có Khóa bí mật" + "Nhập Khóa bí mật và bấm \"Lưu cấu hình\"…" | Lưu enabled; Kiểm tra **disabled** |
| `invalid_key` | red: "Khóa API không hợp lệ" + re-paste hint | both enabled (save new key / retry) |
| `timeout` | amber: "MiMo không phản hồi kịp" + retry hint | both enabled |
| `rate_limited` / `busy` | amber: over-budget + retry hint | both enabled |
| `unavailable` | red: "Dịch vụ MiMo đang gián đoạn… Các nghiệp vụ khác không bị ảnh hưởng." | both enabled |
| `ok` | green: "Kết nối MiMo thành công — độ trễ X ms" | both enabled |

Sao kê/chat/digest surfaces (phases 4–7) reuse the same backend outcome codes.

## Pending user input

1. **Model spike (flash vs pro verdict)** — needs a **real MiMo API key** (never stored in code/tests; QA used `mimo-sk-qa-fake-*` and is sealed at rest in the dev DB). Once provided: run 3 Vietnamese samples on both models, record latency + quality, tick the spike gate in `phase-01-llm-foundation.md`.
2. Business baseline inputs listed in the red-team report (sao kê cycle cost, top-3 partner import questions) remain open for phases 5–6.

## Verification coverage

| Claim / bug | Rung | Evidence | Not covered |
|---|---|---|---|
| Key saved + masked, desktop | UI DRIVEN | `qa/screenshots/mimo-03-desktop-saved.png`, `mimo-04-desktop-test-result.png`; DOM toast `"Đã lưu cấu hình MiMo"`; `qa/mimo-settings-driver.log` | partner/employee roles (masking proven API-side + unit tests instead), staging env |
| Key saved + masked, mobile (390) | UI DRIVEN | `qa/screenshots/mimo-05-mobile-390.png`; DOM assertion `mobile section present = true`; button height 44px logged | partner mobile, landscape |
| 320px no horizontal overflow | UI DRIVEN | `qa/screenshots/mimo-06-mobile-320.png`; driver log `320px horizontal overflow = false` | long company-name strings in this section |
| Test-connection success/failure + Vietnamese messaging | UI DRIVEN | Live MiMo API answered `Invalid API Key` to the fake key → designed `invalid_key` panel (screenshot + quoted DOM in driver log) | **success path (rung 0 for latency/success UI): fake key only — real-key success needs user input** |
| API key masked in ALL settings GET shapes (all roles) | DB/API VERIFIED | `handlers/settings/settings_test.go` drives list/by-id/by-key/active through real service stack (asserts `value:null, masked:true, configured:true`, no plaintext in body); `GET /settings/key/mimo.api_key` live response quoted in driver log | partner/employee tokens not exercised live (covered by Casbin GET-only grants + unit test) |
| Key sealed at rest (red-team Critical #1) | DB/API VERIFIED | MySQL `SELECT … WHERE key='mimo.api_key'` → `enc:v1:ZbHy…` ciphertext; `plaintext_leak = 0`; `settings_repository_secrets_test.go` | key-rotation-at-rest via UI (update path writes sealed — covered by unit test, not clicked) |
| Audit row on key change, values never logged | DB/API VERIFIED | `audit_logs` rows 16485 (CREATE) / 16490 (UPDATE) `metadata: {"key": "mimo.api_key"}` — no value in message/metadata | audit UI rendering of these rows |
| `mimo-v2.5*` rejected at config validation + call time | CODE-READ ONLY (UI not driven) | Go tests (`models_test.go`, `settings_config_test.go`) — typing a retired model into the UI select is impossible (allowlist options) but not clicked | UI surface for invalid persisted values (read path falls back to default — unit tested only) |
| Per-user rate limit + concurrency guard | DB/API VERIFIED | Go tests with miniredis/fakes (`guard_test.go`, `service_test.go`); wired in `init.go` | load behavior under real traffic, Redis-down fail-open in production |
| `make api-test` regression gate green | DB/API VERIFIED | 328 passed / 0 failed against live backend on :8080 | 34 skipped scenarios (env-dependent, pre-existing) |
| 4 failing `components/ui/*` tests pre-exist this change | CODE-READ ONLY | `git status`: neither `ui/*.test.tsx` nor `ui/*.tsx` modified; failures are `focus-visible:outline` class contracts in untouched files | not re-verified at HEAD in a clean checkout |
