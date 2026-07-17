# Dream #14 — Cross-Project Memory Consolidation (pending apply)

**Run:** 2026-07-17 (Dream #14). **Author:** `/dream` invoked from the **payroll** project.

## Why this is a report, not applied edits

This `/dream` run was launched from the payroll session. The host's sensitive-file
permission hook **blocks all writes to other projects' `memory/` dirs** (verified:
both Edit-on-existing and Write-new-file against `~/.claude/projects/-Users-dev-Documents-projects-vfic-ats-ChatBot/memory/`
were denied). This is intentional — dream writes are project-scoped
(`feedback_dream_run_per_project`); the cross-project sweep completes by running
`/dream` once from inside each project.

The signal below was fully gathered (Phase 2) from recent transcripts but could not
be written. Each block gives the **exact target file and exact content to apply**,
so a future per-project `/dream` (or a manual paste) can merge it verbatim. The
payroll project itself was already current as of Dream #13 (today) and needed no
new entries this round.

---

## A. vfic-ats-ChatBot  (memory dir: `~/.claude/projects/-Users-dev-Documents-projects-vfic-ats-ChatBot/memory/`)

### A1. UPDATE  `2026-07-15-runtime-authority-gate-mutes-bot.md`  — append RESOLVED note (HIGH; current memory is misleading)

The file's "Fix options" presents rollback-or-seed as a pending protected decision.
It was actually resolved a third way. **Append this paragraph** after the line
ending `…[[bot-never-responds-2026-07-10-outage]].`:

> **RESOLVED 2026-07-15 (~23:17 UTC, image `a93e5331`) — NOT via either option above.**
> After investigation the user chose to **remove/bypass the fail-closed gate and redeploy**,
> rather than rollback or seed an installation. `webhooks.py` now calls `handle()` directly
> with `runtime_authority` possibly `None`; the bypass behavior is pinned by test
> `test_verified_webhook_dispatches_without_runtime_authority`
> (`backend/tests/.../test_webhooks.py:171-195`). The user first floated an admin toggle
> (session 53c8d5c4) then settled on the bypass path (session 3daaf078). Bot replying as of
> ~23:17 UTC; `installation_state` remains inactive / 0-rows but is now non-blocking. **The
> rule above still holds** — `0f89cedf` violated the gate+seed+guard-unit rule, and the
> cheapest recovery (chosen here) was to back the gate out. The seed/activate path (option 2)
> remains the correct way to use `resolve_active()` gating if it is ever re-introduced.
> (consolidated Dream #14 2026-07-17; supersedes the "PROTECTED — need approval" framing
> above, which is retained for the rationale only.)

### A2. UPDATE  `recent-fixes.md` §16  — refine root cause + mark fixed (HIGH; durable SQLAlchemy rule)

The §16 entry currently says root cause = "raw `op.execute CREATE TYPE`, still unfixed".
Replace the line beginning `- **State**: migration STILL unfixed in code…` with:

> - **REFINED ROOT CAUSE + FIX (2026-07-15, consolidated Dream #14 2026-07-17):** the real
>   trigger is the enum declared as the **generic `sa.Enum(name=..., create_type=False)`**
>   rather than the dialect-specific **`postgresql.ENUM(..., create_type=False)`**. The
>   generic `Enum` silently ignores `create_type=False` and emits a spurious empty
>   `CREATE TYPE ... AS ENUM ()` via the SQLAlchemy `before_create` event during
>   `create_table` — that is the statement the retry loop trips on. Fix: swap to
>   `postgresql.ENUM` (import already exists at migration line 10). Verified by offline DDL
>   render (`alembic upgrade 0038->0040 --sql`): 0 empty `AS ENUM ()`, exactly 2 real
>   `CREATE TYPE ... AS ENUM (...)` with values. Migration `0023` (already on prod) already
>   uses the correct `postgresql.ENUM` form — treat that as the convention. **Durable rule:**
>   in Postgres migrations use `postgresql.ENUM(..., create_type=False)` for enum columns,
>   never the generic `sa.Enum`, when relying on `create_type=False` to suppress type creation.

### A3. NEW  `ingestion-template-recruitment-product-advisory.md`  — feature + dead-code surface (MEDIUM)

> ---
> name: ingestion-template-recruitment-product-advisory
> description: 2026-07-14 multi-vertical ingestion-template feature (recruitment + product_advisory packs) ships with a large dead-code surface — domain_tools/chatbot.paths are test-only, kb_version threading is inert in prod
> metadata:
>   node_type: memory
>   type: project
>   originSessionId: dream-consolidation-2026-07-17
> ---
>
> The ingestion-template / multi-vertical / fact-repository feature (untracked WIP reviewed
> across sessions 64e00086 + 9ca3b4d0) lands `backend/app/capabilities/recruitment/` and
> `product_advisory/` packs. **Live-state finding:** the entire `domain_tools` +
> `chatbot.paths` layer (`path_a_structured`/`path_b_faq`/`get_benefits`/`get_working_hours`/
> `get_job_requirements`/`get_faq_entry`/`get_job_locations`) is **referenced ONLY by tests —
> not wired into the live runner**. `publish_reviewed_recruitment_contract` (the sole caller of
> `publish_contract(kb_version_id=...)`) is itself uncalled → **`kb_version` threading is inert
> in production**. Both packs hardcode `runtime_ready=False` (`recruitment/definition.py:47`,
> `product_advisory/definition.py:15`).
>
> Code-review fixes shipped with the WIP: `_extract_artifact` KeyError guard, `update_draft`
> stale-preview clearing, frontend builder restructure (non-persisting preview +
> create-on-publish), source-level guard for `active_kb_version_id is None`. Verified: 94
> pytest + 2 new assertions, ruff clean, tsc exit 0, blast-radius sweep 185 passed / 14
> skipped / 0 failed.
>
> **Related decision (same review):** graph/service layer isolation forbids cross-layer constant
> sharing; for volatile markers duplicated across `runner.py` ↔ `paths.py` the chosen approach was
> documented duplication (sync-comments) rather than forcing coupling through `app.core` — the
> markers are currently dead code and port-injection would over-engineer a non-live path.

### A4. UPDATE  `preferences.md`  — append local-first rule (MEDIUM)

Append after the 2026-07-12 actionable-direction bullet:

> - [2026-07-15] **Check the local working tree before claiming a file/repo is unavailable.**
>   During the runtime-authority-gate recovery the user snapped *"wtf fuck? repo is here in
>   local, wtf are you talkin about"* (session 3daaf078) after the agent asserted something was
>   missing. Scout the local checkout first; do not assume code/data is remote-only or absent.
>   Reinforces the actionable-direction rule above and the global scout-first contract.

### A5. ChatBot MEMORY.md index — bump header (Phase 4)

Change `Last consolidated: 2026-07-15 (Dream #12).` header to note **Dream #14 2026-07-17**:
runtime-authority-gate RESOLVED via gate removal (`a93e5331`); recent-fixes §16 refined
(`sa.Enum` → `postgresql.ENUM`); +new `ingestion-template-recruitment-product-advisory`;
preferences += local-first rule. (No size risk — index already compact.)

### Already captured (NO action — subagent re-reported these, but they're in memory)
- Facade parity guard (26-method mirror test) → already in `2026-07-14-outbox-channel-facade-drift-outage.md` line 22.
- Deploy-verify budget 60s→100s → already in `deploy-verify-gate-false-fail.md` line 14.
- Naive-datetime outbox root cause → already in `2026-07-14-outbox-naive-datetime-outage.md`.

---

## B. nepocorp  (memory dir: `~/.claude/projects/-Users-dev-Documents-projects-nepocorp/memory/`)

### B1. UPDATE  `preferences.md`  — append demo-DB rule (HIGH; strong user correction)

Append after the last bullet:

> - **Demo DB is additive + UPDATE only — NEVER wipe** (2026-07-15, HIGH). When fixing/seeding
>   demo data on `vantai.tingting.vip`, INSERT new rows or UPDATE existing rows in place. NEVER
>   `TRUNCATE`/`DROP`/recreate the demo DB. Verbatim: *"dont fucking wip the existing db in demo,
>   you just add on, or update, dont fucking wipe"* (session 66b40d79). Broader than the prod
>   rule — this explicitly ALLOWS UPDATEs to existing demo rows (prod forbids data-row changes
>   entirely, see payroll's `feedback_no_prod_db`).

### B2. NEW  `drizzle-0107-onboarding-collision.md`  — deploy-blocking migration lesson (HIGH)

> ---
> name: drizzle-0107-onboarding-collision
> description: 2026-07-14 drizzle migration 0107 lacked IF NOT EXISTS, collided on prod with a manually-applied knowledge_chunks table and atomically rolled back the whole onboarding batch (0104–0106); prod applied 94→98 after idempotency fix
> metadata:
>   node_type: memory
>   type: project
>   originSessionId: dream-consolidation-2026-07-17
> ---
>
> `backend/drizzle/0107_fantastic_star_brand.sql` shipped with `CREATE TABLE "knowledge_chunks"`
> (no `IF NOT EXISTS`) and collided on prod with a `knowledge_chunks` table that had been applied
> manually outside the drizzle journal. drizzle applies the pending batch **atomically**, so this
> single error rolled back the legitimate onboarding migrations 0104–0106 with it. Fix: made `0107`
> idempotent (`CREATE TABLE IF NOT EXISTS` + `IF NOT EXISTS` on the source index). Committed in
> `512d0b99 feat(onboarding): add various documents for onboarding process`; re-deployed via
> `make deploy-backend`; prod applied-count went **94 → 98**. New tables: `onboarding_events`,
> `onboarding_progress`, `onboarding_progress_tasks`, `onboarding_tasks`, `onboarding_status`,
> `knowledge_chunks` (+ `agent_turn_metrics` columns).
>
> **Cleanup flag (unresolved):** `backend/drizzle/0104_faq_knowledge_base.sql` and
> `0106_faq_operational_expansion.sql` exist on disk but are **NOT entries in
> `meta/_journal.json`** (the journal lists `0104_parallel_guardian` / `0106_onboarding_events`
> instead — the onboarding migration won the index collision). `0106` does `INSERT`s into
> `faq_entries` (FAQ content expansion) that may be missing on prod. These orphan files will
> confuse the next `drizzle-kit generate` and reproduce this collision class — needs cleanup.
>
> Same journal-drift family as `drizzle-migrate-timestamp-not-hash.md` and
> `vantai-migration-desync-agent-tables.md`.

### B3. NEW  `shared-dist-rebuild-for-exports.md`  — backend can't see new shared exports until rebuilt (MEDIUM)

> ---
> name: shared-dist-rebuild-for-exports
> description: adding an export to shared/src is not enough — backend resolves @tingting/shared via shared/dist/index.js and tsx watch doesn't rebuild it; run pnpm --filter @tingting/shared build or you get "does not provide an export named X"
> metadata:
>   node_type: memory
>   type: project
>   originSessionId: dream-consolidation-2026-07-17
> ---
>
> Adding/modifying an export in `shared/src/` is not enough for the backend to see it. Backend
> resolves `@tingting/shared` via `shared/dist/index.js` (`package.json` `"main": "dist/index.js"`),
> and `tsx watch` does **not** pick up `dist/` changes. Symptom:
> `SyntaxError: The requested module '@tingting/shared' does not provide an export named
> 'appSettingsSchema'` even though the source is correct. Fix:
> `pnpm --filter @tingting/shared build` (runs `tsc && node fix-esm-imports.js`). Companion to
> `shared-re-export-barrel.md` (which covers the barrel re-export TS2724 trap, not this dist-build
> step).

### B4. nepocorp MEMORY.md index — bump header + add entries (Phase 4)

Header currently reads `Last consolidated: 2026-07-08` (stale — topic files already go to
2026-07-16). Bump to **Dream #14 2026-07-17** and add three index rows for B1/B2/B3. Also add
to Quick Reference: **"Demo DB (`vantai`): additive + UPDATE only, never wipe"**.

### Held (not persisted — needs user confirmation)
- "nepocorp uses feature branches" (sessions ran on `feat/onboarding-orchestration`). Likely
  just this one feature; existing memory already notes "PRs used for larger batched changes".
> , so not durable enough to record as a convention. Confirm with user first.

---

## C. payroll  (current project — already current, no new entries this round)

- 10 of 11 recent payroll transcripts were autonomous `/dream` runs consolidating the **ChatBot**
  project; correctly filtered out (no cross-project writes into payroll memory).
- The one real payroll session (`f60a7da1`) produced the manual-partial-settlement /
  `revenue_paid` finding — **already captured** in `lesson_settlement_oversell_silent_skip.md`
  (line 14, written earlier today in Dream #13). No new entry needed.
