# Model Spike Report — Phase 1 gate

> Date: 2026-10-11 · Endpoint: `https://token-plan-sgp.xiaomimimo.com/v1` (TokenPlan dedicated, Singapore) · Key: from `~/.zshrc`, never printed/stored · Raw responses: `qa/spike/*.json` (gitignored)

## Setup correction (root cause of the earlier 401)

The first spike run failed with `Invalid API Key` on all calls. Diagnosis, in order:
1. Wrong base URL — used the public API platform (`api.xiaomimimo.com/v1`) instead of the **TokenPlan dedicated endpoint** (`token-plan-sgp.xiaomimimo.com/v1`). TokenPlan keys are rejected by the public host (401, fast).
2. `mimo.base_url` setting then failed to save twice: first because I POSTed without the required `value_type` (`CreateSettingRequest.ValueType` is `binding:"required"`), then because a retry curl read the wrong token JSON path (`.data.token` vs `.data.access_token`).
3. With no override row, the running backend fell back to the old default and hit the public host. It reported `timeout`, **not** `invalid_key`. The classification logic is correct — `errorKindForStatus` maps 401/403 → `KindAuth`, and `retryable := 429 || >=500` means a 401 returns immediately (covered by client_test.go). So the app never received a 401: it hit a **DNS/connection failure** (this host is intermittently unresolvable from this environment — the same `ENOTFOUND` that killed both first-wave agents), and the bounded deadline expired before any HTTP response, surfacing as `KindTimeout` via `ContextError`. The misleading message is a *symptom of the environment*, not a code defect.

Fixes applied: `infra/llm/client.go` `DefaultBaseURL` → TokenPlan host; `mimo.base_url` seeded (with the required `value_type`). Post-fix `POST /settings/mimo/test-connection` → `success:true, code:"ok", latency_ms:111, model_count:8` (matches the 8-model TokenPlan catalog).

**Operational note:** `token-plan-sgp.xiaomimimo.com` resolves intermittently from this machine. Tests must never make live calls to it (use httptest/fakes); a build agent dying on `ENOTFOUND` is an environment flake, not a code signal.

## Results

| # | Sample | Model | Time | Outcome |
|---|--------|-------|------|---------|
| 1 | Vietnamese sao kê email draft | `v2.6-flash` | 7.16s | Good Vietnamese; numbers reproduced verbatim; **missed** the internal-vs-external framing conflict |
| 1b | same payload repeated (cache probe) | `v2.6-flash` | 27.10s | Different output; **cache effect inconclusive** — load variance dominates |
| 1c | same sample | `v2.6-pro` | 13.75s | Correct draft **plus** an explicit catch: "yêu cầu này là gửi ra ngoài (đối tác), khác phạm vi email nội bộ thường lệ" |
| 2 | Import failure diagnosis (3 errors) | `v2.6-flash` | 29.36s | Numbered per row, cause + fix each — fully usable |
| 2b | same sample | `v2.6-pro` | 15.42s | Same quality; **faster than flash** here |
| 3 | Tool-calling turn | `v2.6-flash` | 15.67s | `tool_calls: 1` → `get_import_status` — correctly **diagnoses before asking admin** |

## Verdict

**Model split: KEEP** (flash interactive / pro deliverables). Quality is NOT indistinguishable: on the drafting sample pro caught a contextual conflict flash missed, while diagnosis quality was equal — exactly the asymmetry the split was designed for.

**Latency: budget MISSED — streaming is now REQUIRED.** Full-response (non-streaming) latency ranged 7.2s–29.4s against a p95 target of ≤2s direct / ≤5s tool turns. Per `plan.md` the "measure-or-stream" gate triggers: SSE streaming becomes a mandatory fast-follow for the chat path, and chat must cap generation length (≤4 sentences, low `max_tokens`) rather than relying on the model to be brief.

**Caveats (do not over-read this data):** single samples per cell cannot establish p95 — the 7s→27s spread is load-dependent and overlaps both models. Prompt-cache benefit is unproven on latency (billing-side cache credits are a separate question). Production instrumentation remains the gate, not this spike.
## Spike results — 2026-10-11 post-recovery run (dedicated endpoint, 6/6 calls OK)

| Call | Model | Time | Notes |
|---|---|---|---|
| s1 email draft | flash | 12.0s | clean Vietnamese draft, all numbers verbatim |
| s1 repeat (cache probe) | flash | 14.2s | NO cache speedup; degraded output ("Kính gửi, kính gửi") — sampling variance is real |
| s1 email draft | pro | 28.9s | pedantic reframe (over-literal "internal email" reading); worse fit, 2.4× latency |
| s2 import diagnosis | flash | 13.7s | solid numbered per-row cause + fix |
| s2 import diagnosis | pro | 23.2s | same content, nicer markdown only |
| s3 tool turn | flash | 4.1s | **tool_calls=2** — function calling confirmed working |

**Verdict (phase-01 gate): collapse to `mimo-v2.6-flash` for ALL text surfaces.** Pro's only win was cosmetic markdown at 2.4× latency plus one quality regression. Vision/pro decision deferred to the phase-6 vision spike (no image sample run here).

**Latency evidence for phase 5:** full-text flash turns ran 12–14s (700 max_tokens, thinking on) vs 4.1s for pure tool selection. The p95 ≤2s direct budget therefore REQUIRES the short-output contract (≤4 sentences) and makes the measure-or-stream gate evidence-backed. Action item for chat implementation: investigate thinking-mode toggle / `max_completion_tokens` cap on interactive turns.
