# Code Review: uncommitted backend batch vs HEAD (6af555b2)

- Date: 2026-09-20 18:44, mode `auto --fix`
- Scope: 41 modified + 25 untracked files (+1809/−181), all uncommitted on `main`
- Reviewer: standards-reviewer sub-agent (opus); fixes applied in controller session

## Spec

Skipped — no spec exists. No commits, no issue references, no matching plan under
`plans/` for the diff's work streams (mobile identity, reachability, settings
secrets, aggregate recompute, collation migrations). If one exists, point at it
and re-run `/code-review <sha>`.

## Standards

Reviewer's report (lightly cleaned, line refs as of review time):

**Hard violation — FIXED**

1. `disbursement_execute_worker.go:282` — preflight rejection passed
   `err.Error()` (English, e.g. "onepay: Amount must be at least 100000 VND")
   straight into `NotifyAdvancePaymentFailed`, reaching the employee push. Violates
   CLAUDE.md rule 4 / code-standards "UI Text": all user-facing text Vietnamese.

**Judgement calls**

2. ~~Settings secret backfill never wired~~ — **not reproducible**: `container.go:141`
   calls `EncryptProtectedValues` in the production DI container, best-effort with
   boot-continue on failure. Reviewer missed the caller. No action.
3. `pkg/secret/default.go:7` imports `infra/observability` — real layering inversion
   (pkg→infra), but `slog.SetDefault` is never called, so swapping to plain `slog`
   would silently reroute the degraded-mode warning from `logs/app.log` to stderr.
   Left as-is; follow-up options: inject a logger into `secret.Default()` resolution
   or relocate resolution into infra. Report-only.
4. `employee_repository_reachability.go:125` — unbounded full-cohort query, Go-side
   pagination, List+Count double scan; breaches performance.md "paginate everything".
   Code comments acknowledge the small-table rationale. Follow-up: single-pass
   classify+count. Report-only.
5. `bulk_transfer_worker.go:194` — project-total recompute after `BulkUpdatePaymentStatus`
   outside a shared tx, errors logged and swallowed; totals eventually consistent via
   `RecomputeAllProjects`, which has no scheduler wiring. Report-only; recommend
   documenting the reconciliation dependency or wiring a scheduled recompute.
6. ~~Frontend CCCD guard duplicated~~ — **not reproducible**: both sites call the same
   shared `validateMobileNotCCCD`; only the presentation wiring differs (inline form
   error vs save-time toast). Not duplicated logic. No action.
7. Migration 111: 19 × `ALTER TABLE ... CONVERT TO` are full table rebuilds —
   **FIXED** by adding an operational note to the migration header (maintenance
   window, per-table DML blocking, largest table size).
8. `NewSettingsRepository` returns the concrete type — deliberate and documented at
   `settings_repository.go:24-26` (bootstrap needs `EncryptProtectedValues`). No action.

**Confirmed compliant:** ADR-007 cache invalidation ordering; domain purity
(`pkg/phone` stdlib-only, `employee_reachability.go` clean); secret cipher design
(AES-256-GCM, random nonce, fail-closed `ErrKeyUnavailable`, byte-guarded CAS backfill);
package-level `clock.Now()` usage matches the 378-site repo precedent.

## Fixes applied

| # | File | Change |
|---|------|--------|
| 1 | `backend/internal/app/workers/disbursement_execute_worker.go` | New `employeeFacingDetail(code, detail)`: `preflight_validation` → fixed Vietnamese message; provider messages pass through. Used at the `NotifyAdvancePaymentFailed` call in `failAdvanceRequest`. Raw English stays in wallet_payment `RawMessage` + logs. |
| 2 | `backend/internal/app/workers/disbursement_execute_worker_test.go` | New `TestApplyPermanentProviderFailure_VietnameseEmployeeDetail` guarding the translation + provider passthrough. |
| 3 | `backend/migrations/111_normalize_collations.up.sql` | Header operational note: full table rebuilds block DML; run prod in a maintenance window. |

## Verification

- `go build ./...` exit 0; `go vet` clean; `gofmt` clean
- `go test ./internal/app/workers/ -count=1` — ok
- Full backend suite: `go test ./... -count=1` — exit 0, zero failures
- Frontend `pnpm type-check` + `pnpm lint` — exit 0 (no frontend changes made)

## Follow-up outcomes (2026-09-20, after this review)

Findings 3/4/5 were re-examined and closed as follows.

**Finding 3 — `internal/pkg/secret` importing `internal/infra/observability`: WONT FIX
by design.** The import is the repo-wide convention for the `internal/pkg` layer, not a
defect local to this package: `internal/pkg/retry/retry.go`,
`internal/pkg/tenantqueue/queue.go`, `internal/pkg/db/{helper,batch_helper,temporal_helper}.go`
and `internal/pkg/services/payrate_temporal_service.go` all call `observability.GetLogger()`.
Rewriting only `secret` would leave two logging seams in one layer, and the reviewer's own
note records the cost of the alternative: plain `slog` reroutes the degraded-mode warning
out of `logs/app.log` to stderr. A pkg-layer logging port is a repo-wide refactor, not a
follow-up to this batch.

**Finding 4 — reachability query shape: FIXED (`8e40c480`), then the whole feature was
REMOVED (`c1ee22ee`).** The fix made one repository call read the cohort once, classify it
once, and return the page plus the cohort counters together. Reviewing it then showed the
report had **no HTTP route, no handler and no frontend caller**: its only consumer was its
own tests. Wiring it as designed would also have exposed name, CCCD, mobile, projects and
salary for every active employee to every partner, because the casbin policy grants
`partner` and `adv_partner` all of `/api/v1/employees/*` while neither the service nor the
repository scopes rows by the caller's accessible employees. The report was therefore
deleted rather than wired: domain states/signals/classifier/report types, the repository and
its query, the service, both test files, the port method, its mock and the one message
constant. `idx_employees_mobile` and `FlexPaySalaryNotificationSuppressed` stay — both have
independent consumers. Recoverable with `git revert c1ee22ee` if it is wanted later as an
admin-only ops screen with explicit row scoping.

**Finding 5 — recompute scheduling: FIXED (`0aa17446`).** `RecomputeAllProjects` is now
registered as the daily `recompute_project_aggregates` job (03:00 `Asia/Ho_Chi_Minh`) in
`registerSchedulerJobs`, with the service built once in the container. The swallowed
per-project error in the bulk transfer worker now has an automatic recovery path on top of
the manual `recompute-project-aggregates` subcommand. Deployed and verified on production:
scheduler logs `Job registered | recompute_project_aggregates 0 3 * * *` and
`cron_job_status` row 7572 is seeded and enabled.

## Open item surfaced while closing the above — RESOLVED by removal

- The Zalo reachability report (`42c161f9`) was service- and repository-only: no HTTP
  handler, no route, no frontend caller, so ops could not see it. Decision (2026-09-20): do
  not carry an unreferenced second write path for contact data — the feature is deleted in
  `c1ee22ee`.
- The data gap it described is NOT fixed by the deletion and has no detector now: 588 of
  1440 active employees have no mobile value and 23 had their most recent ZNS send rejected
  with -118, so payouts and notifications for those employees have no delivery channel.
