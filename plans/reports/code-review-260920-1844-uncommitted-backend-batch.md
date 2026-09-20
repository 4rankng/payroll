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


## Tech-debt pass (2026-09-20, after `make api-test` on the local stack)

Running the integration suite against a local backend surfaced a set of real
defects that the earlier review had no reason to look at. All were reproduced
locally before the fix and re-verified after, and the suite's baseline is now a
clean signal.

| # | Defect | Evidence it was real | Fix |
|---|--------|----------------------|-----|
| 1 | Wrapped domain errors all answered 500 | `HandleDomainError` asserted the top-level type; services wrap with `%w`, so every wrapped validation/not-found fell to the 500 branch and polluted the 5xx counters the error-rate job watches | `errors.As` + translate the domain error (commit `61f58213`) |
| 2 | Unbalanced ledger block answered 500 | `POST /ledger/entries` with one debit leg returned `HTTP 500 ledger integrity violation` — caller input reported as a server fault | typed Vietnamese validation error → 400 (`ad911dcf`) |
| 3 | Unknown cron job answered 500 | Repository already returned a typed not-found; the handler discarded it | route through `HandleDomainError` → 404 (`f0f3368e`) |
| 4 | Nil bulk-transfer service panicked into an empty 500 | `GET /wallet/bulk-transfer/batches` returned 500 with an empty body (gin recovery) on any deployment without a disbursement provider; `estimate-fee` panicked the same way — a typed nil in an interface defeats `== nil` guards | leave the capability genuinely absent (`074c54ab`) |
| 5 | Re-uploaded FlexPay file was never reprocessed | `ErrTaskIDConflict` (finished task ID reused) was not tolerated, and the handler still answered "đang xử lý lại" | retry under a unique ID, and answer 500 when the enqueue really fails (`5d128d4a`) |
| 6 | Statement email answered 202 for a period it cannot report | Worker logged `không có dữ liệu payroll cho kỳ được chọn` while the API said the request was accepted: no email, no history row | validate the period before queueing → 400 (`8c330272`) |

**Behaviour change worth knowing:** sending a payroll statement for a period with
no eligible rows now answers 400 with the reason instead of 202. That is the point
of fix 6 (the admin could not previously tell that nothing was sent), but it is a
status-code change for that endpoint.

**Test suite:** provider-dependent flows now decide from
`GET /admin/settings/disbursement` (`registered_providers`) instead of the payout
settings flag, which can be on in a deployment that registered no provider; the
BCC import picks a weekly project the partner can actually write to; the ledger
flow creates a balanced block and asserts the unbalanced one is refused with 400;
the statement-email waits allow the worker the time it needs on a full dataset.
Result: 332 tests, 303 passed, 0 failed, 29 skipped with a reason (before: 22
failures mixing real defects with environment gaps).

**Still open, unchanged by this pass:** `LedgerService.ReverseEntry` writes the
mirror of a single entry through the unchecked `Create` path, so a reversal is
one-sided by construction. It nets out across the ledger (the original and its
mirror offset), which is why totals stay balanced, but it is the one remaining
write path that does not face the block-level double-entry guard. Deciding
whether reversals should be blocked, paired, or explicitly exempt is a financial
semantics call, not a cleanup.
