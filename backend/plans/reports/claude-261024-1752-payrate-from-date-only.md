# Payrate flow: from_date-only rewrite — session report (24/10/2026)

Ticket: PAYR-3 (backend, → QA), PAYR-4 (frontend, → QA), PAYR-5 (verification, Todo, blocked by 3+4).
Plan: ~/.claude/plans/tighten-improve-the-flow-spicy-cosmos.md (approved). Webchat consults: Gemini 24/10 ×2 (entry-flow design; from_date-only verdict).

## Outcome

The payrate change flow is rebuilt on the owner's from_date-only ruling: a config's end is implied by its successor (resolution: latest `from_date <= work date` — unchanged), and everything that materialized `to_date` — the source of every incident artifact — is deleted.

### Backend (PAYR-3)
- Create = pure INSERT, must start strictly after the latest config; structured error `PAYRATE_NOT_LATEST` with `latest_id`/`latest_from_date` context.
- Update = in-place on the latest row only (replaced rows are view-only, error names the successor); kept guards: paid floor, sibling-order check, earliest-linked-timesheet bound, `recalculateMutableTimesheets` (aborts on missing paytype).
- Delete = pure soft-delete; approved/paid links hard-block (`PAYRATE_IMMUTABLE_LINKED`); mutable-only links re-price under the covering config in the same tx; timeline heals by resolution.
- Deleted: `splitUpdateTx`, `closePayrateTx`, `synchronizePredecessorEndDateTx`, `manageActivePayrates`, delete's predecessor re-open, dead `POST /api/v1/payrates`, plus 7 unused repo/port methods.
- Reads: all `to_date` predicates removed; resolution + current/upcoming queries gain `id DESC` tiebreaker; `IsActiveOnDate` checks from_date only.
- Responses derive `toDate` (successor − 1; null for latest) in `AttachDisplayEndDates` — wire contract unchanged.
- Write-time key discipline: `NormalizedKeys` (trim + NFC, diacritics preserved) rejects Vietnamese fold-collisions; wired into create/update/dry-run.
- Dry-run: lost-position warning, key-drift warning (case variants vs latest config), `mutable_timesheets_affected` count, non-latest/sibling mirrors.
- Migration 137 (idempotent, guarded): drops `uq_payrates_one_open_per_project` + `open_project_id`; codifies `live_from_date` + `uq_payrates_live_project_from_date` for fresh envs. `to_date`/`chk_payrates_date_order` drop later (after one pay cycle). **Deploy order: migration BEFORE the new build.**

### Frontend (PAYR-4)
- Carry-forward: create seeds a deep clone of the latest config; min date = latest + 1 with Vietnamese note; generic template only for first-ever config.
- Confirmation step before creates/rate changes: diff summary (changed cells, added/removed positions, key-drift warnings, timesheets to re-price). Date-only corrections still save directly.
- Failed mutations surface inline (were silently swallowed).
- `PayrateConfigTab` shows all configs: active, upcoming (editable — it is the latest), collapsed "Cấu hình khác" with two-step delete (`useDeletePayRate` now used).
- Editors normalize keys (trim+NFC) and block fold-duplicates; JSON tab removed (desktop + mobile).
- Dead code deleted (verified zero imports): PayrateJsonEditor, ConfigurationHistory, PositionInput, PositionRateForm, QuickTemplateSelector, RateTypeSelector, usePayrateValidation, utils/payrate-test.ts. Kept: in-file convenience hooks in usePayRates.ts (zero importers, low risk).

## Verification
- `go test ./...` exit 0 (97 packages), incl. rewritten temporal tests (in-place/create/delete-heal/block), domain NormalizedKeys, AttachDisplayEndDates order test.
- `pnpm type-check` exit 0; `pnpm lint` 0 errors.
- Live drive on dev (api-server :8081, migration applied to dev DB): 13-step curl flow — every designed behavior confirmed, including the two rejections and the healing delete. Bug found live and fixed: derived-end-date walk assumed the wrong sort; fixed order-agnostically + regression test.
- Fixtures: project `PAYRATE-FLOW-TEST` (214000000071) with configs 10-01/10-15; dev passwords reset to `Admin123` (repo-documented dev state).

## Follow-ups found during the wave
1. `make reset-passwords` uses root + `MYSQL_ROOT_PASSWORD` which is empty in this dev container — target fails; I ran its steps manually as `payroll_user`. Fix the Makefile.
2. QA-driven UI browser run + integration scenarios in `tests/integration/` remain (PAYR-5).
3. After one pay cycle on prod: migration to drop `to_date` + `chk_payrates_date_order`.
4. Backend not committed yet (working tree, main). Commit on owner's word.
