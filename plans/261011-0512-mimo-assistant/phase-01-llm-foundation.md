---
title: "Phase 1: LLM Foundation"
status: done-with-concerns
---

# Phase 1: LLM Foundation

## Overview

Provider-agnostic LLM client + admin-managed Xiaomi MiMo API key + model selection spike. Nothing user-facing ships here; this is the substrate phases 4–6 build on.

## Requirements

- [x] Admin can enter/update the MiMo API key in settings (masked field), desktop + mobile
- [x] Key stored via the settings table AND registered in `pkg/secret/keys.go` `ProtectedKeys` (sealed at rest — the zaloconnect precedent is **sealed**, not plaintext; `settings_repository_secrets_test.go` proves the mechanism) — masked in every API response (`secret.IsProtectedKey`), value never logged, audit row on change
- [x] Concrete HTTP client lives in `internal/infra/llm/` behind a port interface (pattern: `internal/infra/zalo/` + `internal/adapters/`); app services depend on the port only — the "provider swap in an afternoon" is a seam, not a slogan
- [x] Per-user rate limit + spend/concurrency circuit breaker on the shared key (4 consumers: drafting, chat, vision, digest — one runaway loop must not starve the rest); runtime model-override setting + fallback model ID (vendor renamed its model family within 10 days — allowlist redeploys are the failure mode)
- [x] Designed Vietnamese error states specified per surface (settings, sao kê preview, chat, digest) for: missing/invalid key, model timeout mid-draft, LLM outage during unattended 08:00 retry — what the user sees, which buttons stay enabled *(settings surface designed + implemented in this phase; sao kê/chat/digest surfaces are phase 4–7 deliverables and reuse the same outcome codes)*
- [x] "Test connection" admin action: validates key against `https://api.xiaomimimo.com/v1/models` and reports latency
- [x] `AIClient` interface: `Chat(ctx, req)` with function calling, streaming-capable, image input (base64), per-call `model` param; OpenAI-compatible `chat.completions` so any provider swaps in an afternoon
- [x] Model routing config: `mimo-v2.6-flash` default, `mimo-v2.6-pro` for deliverables; `mimo-v2.5*` rejected at config validation
- [ ] Spike gate executed and its verdict recorded in this file's Success Criteria — **PENDING USER INPUT: needs a real MiMo API key** (the spike runs 3 Vietnamese samples on flash vs pro; a fake key cannot produce latency/quality data)

## Implementation Steps

1. Backend: create `internal/app/services/llm/` — `client.go` (HTTP client, timeouts, retries, error mapping to Vietnamese), `types.go` (chat request/response, tool-call structs), `models.go` (allowed model IDs). *(Superseded by red-team placement constraint: the HTTP client lives in `internal/infra/llm/` behind the `llm.Client` port; `internal/app/services/llm/` holds the app service — model routing, guards, settings-driven client construction.)*
2. Settings: add `mimo.api_key` (+ `mimo.base_url` override, `mimo.default_model`) via `services/config/settings_service.go` accessors; GET returns masked value only. *(Done in `settings_config.go` — also `mimo.fallback_model`.)*
3. Handler + routes: extend `handlers/settings/settings.go` and the `/api/v1/settings` group in `routes_admin.go` with `POST /settings/mimo/test-connection`.
4. Frontend: `components/settings/MimoConnectionSection.tsx` (secret-field pattern from `ZaloConnectionSection.tsx`, "Khóa bí mật" masked input, save + test buttons); register in `SettingsTabList.tsx` (`SETTINGS_TABS`); wire desktop `pages/admin/SettingsPage/index.tsx` AND mobile `pages/mobile/admin/SettingsPage/index.tsx`.
5. Hooks: reuse `useSettingByKey`/`useUpdateSetting` (`hooks/api/useSettings.ts`); add test-connection mutation. *(Read path uses `useSettings({search: 'mimo'})` — an all-miss `useMultipleSettings` batch reads as a load failure on fresh environments; empty list is the designed "Chưa cấu hình" state.)*
6. Spike: run 3 Vietnamese samples (email draft, import diagnosis, multi-tool turn) on flash and pro — record latency + quality verdict below. **PENDING USER INPUT (real API key).**

## Related Code Files

- Create: `backend/internal/app/services/llm/{client,types,models}.go`, `frontend/src/components/settings/MimoConnectionSection.tsx`
- Modify: `backend/internal/app/services/config/settings_config.go`, `backend/internal/transport/http/handlers/settings/settings.go`, `backend/internal/app/bootstrap/routes_admin.go`, `frontend/src/components/settings/SettingsTabList.tsx`, both SettingsPage variants
- Template for tests: `frontend/src/components/settings/ApiKeysSection.test.tsx`

## Success Criteria

- [x] Key saved + masked on desktop and mobile settings (both verified) — UI-driven 2026-10-11, see `reports/2026-10-11-phase-01-llm-foundation.md` (screenshots `qa/screenshots/mimo-*.png`)
- [x] Test connection returns success/failure with Vietnamese messaging — live MiMo API rejected the QA fake key with `Invalid API Key` → designed `invalid_key` state rendered
- [x] Unit tests: client request/response parsing, masked GET, model validation
- [ ] Spike verdict recorded: ☐ flash-only (collapse) ☐ keep flash/pro split — **PENDING USER INPUT: real MiMo API key required to run the flash-vs-pro spike**

## Risks

- Settings-table secrets: zaloconnect stores credentials JSON in settings rows — mirror its handling exactly; flag plaintext-at-rest residual risk to the business rather than inventing a second secret system (KISS).
- Chat naming collision: an external "TingTing chatbot" integration already exists (`internal/app/services/integration/`) — name this module `llm`/`llmchat`, never "chatbot service".
