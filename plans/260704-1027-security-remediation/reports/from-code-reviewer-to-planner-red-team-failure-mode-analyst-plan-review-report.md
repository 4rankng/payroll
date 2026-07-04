# Red-team plan review — Failure Mode Analyst

**Plan:** `plans/260704-1027-security-remediation/`
**Lens:** Murphy's Law — race conditions, data loss, cascading failures, recovery/deploy holes.
**Method:** trace every proposed change through actual callers/callees; reject any claim not backed by file:line evidence.
**Verdict:** Plan is NOT landable as written. Multiple findings either target the wrong code path, are already mitigated by existing guards (severity overstated), or omit the rewiring/error/deploy steps needed to make them real.

---

## Finding 1: H5 targets the wrong settlement path — the live money-in settlement is `SettlementUploadService.ProcessSettlementFileWithDedup` + `SettlementEventHandler.ApplySettlement`, NOT `FlexPaySettlementService.ProcessSettlementFile`

- **Severity:** Critical
- **Location:** Phase 1, "Architecture / H5" and "Related Code Files"
- **Flaw:** The plan asserts H5 wraps `ProcessSettlementFile`'s "status UPDATE + ledger create" in `db.Transaction` and frames the fix as "mirror the already-fixed timesheet path (`ApplySettlement`)". That framing is incoherent: `FlexPaySettlementService.ProcessSettlementFile` is the **advance-payment recon** path (`recon_settlemetn_handler.go:58`), and `ApplySettlement` is the **timesheet sao-ke** path (`settlement_event_handler.go:214`, called via `SettlementUploadService.ProcessSettlementFileWithDedup` at `upload_service.go:519`). They are two different flows writing to different tables. The audit itself flagged this in "Unresolved questions #1" (`lesson_sao_ke_upload_silent_settlement_drop` describes a synchronous `SettlementApplier`, commit `3beb3c6`) — the plan silently resolved that open question in the wrong direction without re-verifying wiring.
- **Failure scenario:** Cook implements H5 against `FlexPaySettlementService`. The actual production settlement upload (`/api/v1/settlements/upload` and `/api/v1/timesheets/.../settlement-upload`, both routed to `SettlementUploadService`) is untouched. The H5 money-integrity gap (if any remains after the synchronous `SettlementApplier` rewrite) stays open. CI passes because the new test targets the wrapped function; production still has the original gap. Worse, the file-hash unique index gets added to the wrong table (see Finding 2).
- **Evidence:**
  - `internal/transport/http/handlers/advance_payment/recon_settlemetn_handler.go:58` → `flexPaySettlementService.ProcessSettlementFile` (the path H5 targets)
  - `internal/transport/http/handlers/settlement/upload_settlement.go:76` + `internal/transport/http/handlers/timesheet/settlement_upload.go:50` → `settlementUploadService.ProcessSettlementFileWithDedup` (the live path H5 should target)
  - `internal/app/services/settlement/upload_service.go:484,519` → `s.settlementApplier.ApplySettlement(...)` (synchronous, one-tx-per-record, deadlock-retry — already what H5 claims to add)
  - `internal/infra/events/settlement_event_handler.go:201-214,435-437` — comment "invoked SYNCHRONOUSLY by the settlement-upload flow (SettlementApplier)" + "ApplySettlement is idempotent"
  - `internal/app/bootstrap/services/init.go:234` — `settlementUploadService.SetSettlementApplier(settlementEventHandler)` wires the synchronous path
- **Suggested fix:** Before any code, re-verify which path actually carries production settlement volume today. If `FlexPaySettlementService` is dead/legacy, H5 closes nothing — re-scope or drop. If both paths are live, H5 must specify each path's tx/idempotency status separately. The plan must stop using "mirror the already-fixed timesheet path" as the H5 design — the timesheet path IS the live path; the FlexPay path is a separate flow that needs its own analysis.

---

## Finding 2: H5's `settlement_file_hash` UNIQUE index — table unspecified, rollback undefined, and `make demo`/`make deploy` migration gotchas ignored

- **Severity:** Critical
- **Location:** Phase 1, "Architecture / H5" and "Risk Assessment"
- **Flaw:** The plan says "add col nullable → backfill → add unique index in a follow-up migration (or `UNIQUE ... WHERE` partial index); deploy migration before backend" — but (a) never names the target table, (b) never specifies the rollback .down.sql, (c) MySQL does NOT support partial/filtered unique indexes (`UNIQUE ... WHERE` is Postgres syntax — this is a MySQL shop per `CLAUDE.md`), and (d) ignores two known prod gotchas: `lesson_apply_sql_migration_demo_prod` (manual DDL needed because `schema_migrations` is empty on demo+prod, run BEFORE backend deploys) and `lesson_drop_index_blocked_by_fk` (index drops blocked by FK on MySQL).
- **Failure scenario:** Cook writes `084_settlement_file_hash.up.sql` against the wrong table (because Finding 1 left the target ambiguous), or uses `CREATE UNIQUE INDEX ... WHERE` syntax that MySQL rejects. Migration half-applies on demo (no `schema_migrations` row, so re-runs on next `make demo-db` and fails on duplicate column). Backend deploy references the new column → container crash-loops on boot. Rollback: the plan provides no `.down.sql`, no runbook, and `golang-migrate down` against an FK-blocked index fails (`lesson_drop_index_blocked_by_fk`).
- **Evidence:**
  - Plan phase-01:47 — "deploy migration before backend" with no rollback path
  - `migrations/` listing — current max is `083`; no `settlement_file_hash` column exists anywhere (`grep settlement_file_hash` empty across `migrations/` and `internal/`)
  - MySQL does not support filtered unique indexes — the plan's "or `UNIQUE ... WHERE` partial index" is a Postgres-ism
- **Suggested fix:** (1) Name the exact target table in H5. (2) Drop the partial-index alternative (MySQL can't do it) — if nullable-hash rows must coexist with uniqueness, use a generated column or a separate dedup table. (3) Mandate a paired `.down.sql` and an idempotent `apply_sql_migration_demo_prod`-style runbook entry. (4) Add a CI step that runs the migration up→down→up on a fresh MySQL container.

---

## Finding 3: H6 dedup is TOCTOU-unsafe and largely redundant — the worker is ALREADY idempotent via `Initiate` + `isDuplicateRequestIDError` + `HasPendingForRecipient`. H6's "High → double-pay" severity is overstated.

- **Severity:** High (severity-overstatement + false-confidence risk)
- **Location:** Phase 1, "Architecture / H6" + plan.md finding map
- **Flaw:** The plan proposes `HasNonTerminalForAdvanceRequest` then `Enqueue`, returning 409 if a non-terminal `wallet_payment` exists. That check has a TOCTOU window (plan admits no DB constraint). But trace the worker: `DisbursementExecuteWorker.ProcessJob` calls `walletPaymentService.Initiate` which (a) calls `HasPendingForRecipient` (`wallet_payment_service.go:163`), (b) on duplicate `request_id` catches `isDuplicateRequestIDError` and re-fetches the existing row (`:199-204`), and (c) the worker returns nil if the row is already past pending (`disbursement_execute_worker.go:118-122`) or returns `ErrDuplicatePaymentInProgress` as terminal (`:109-113`). So even if two retry requests both pass the plan's pre-Enqueue check and both enqueue, the second worker invocation exits cleanly without creating a second payment. The plan's success criterion "two concurrent retries → one 200, one 409; no second asynq task enqueued" is the wrong bar — the real invariant is "no double-pay", which already holds.
- **Failure scenario:** Two concurrent admin retries both pass `HasNonTerminalForAdvanceRequest` (TOCTOU), both enqueue. The plan's stated guarantee ("no second asynq task enqueued") is violated — but no double-pay occurs because the worker's `Initiate` dedup catches it. The plan gives false confidence that the pre-check is the protection layer; if a future cook "simplifies" by removing the worker-level dedup (believing the handler check is sufficient), double-pay becomes possible. Severity is mis-stated as High money-loss when the actual residual risk is "two enqueued tasks, one no-op."
- **Evidence:**
  - `internal/app/services/disbursement/wallet_payment_service.go:163-169` — `HasPendingForRecipient` already returns `ErrDuplicatePaymentInProgress`
  - `internal/app/services/disbursement/wallet_payment_service.go:192-204` — `Create` catches duplicate `request_id`, re-fetches existing row
  - `internal/app/workers/disbursement_execute_worker.go:99-122` — Initiate idempotent; row past pending → return nil
  - `internal/app/workers/disbursement_execute_worker.go:109-113` — `ErrDuplicatePaymentInProgress` → terminal, no retry
- **Suggested fix:** Re-state H6's severity accurately: residual risk = duplicate asynq task (wasted worker cycle, log noise), NOT double-pay. Either (a) drop H6 to Medium and add the pre-check purely for UX (409 tells admin "already in flight"), OR (b) if the goal is watertight dedup, the plan must specify the actual race-safe mechanism: a `UNIQUE(advance_request_id) WHERE status NOT IN (terminal)` partial index is impossible on MySQL, so the real fix is an asynq `UniqueTTL` on the task or a row-level `SELECT ... FOR UPDATE` in the handler. The plan specifies neither.

---

## Finding 4: H7 status-allowlist will break the documented `failed → completed` reconciliation override unless `failed` is in the allowlist — and the plan defers the enumeration to "coordinate with owner," a blocking handoff buried in implementation

- **Severity:** High
- **Location:** Phase 1, "Architecture / H7" + "Risk Assessment / H7"
- **Flaw:** `MarkReconciled` is called exclusively from `WalletPaymentService.ReconcilePayment` (`wallet_payment_service.go:711-777`). That function's entire purpose is to override status in cases the FSM cannot express — most importantly `failed → completed` when reconciliation contradicts the IPN (`:720-727`, with a `Warn` log "reconciliation override — failed → completed"). If H7 narrows the precondition to a generic "legal prior states" list and the cook forgets to include `failed`, every legitimate recon-driven `failed → completed` flip returns `ErrNotFound` (the repo returns that when RowsAffected==0, `tx_wallet_payment_repository.go:288-290`), and `ReconcilePayment` returns an error → the **reconcile worker fails the task and asynq retries indefinitely**, or worse, silently no-ops the reconciliation and the books stay wrong.
- **Failure scenario:** Reconciliation worker polls OnePay, finds a payment the provider says succeeded but our row is `failed` (e.g. IPN was lost, prior transient error). With H7 applied naively (allowlist = `{authorised, verified, completed}`), `MarkReconciled(id, completed)` against a `failed` row updates 0 rows → `ErrNotFound` → reconcile worker logs error and either retries forever or skips. The money was paid by the provider; our books say failed; the override that exists specifically to fix this no longer works.
- **Evidence:**
  - `internal/app/services/disbursement/wallet_payment_service.go:719-727` — `case StateFailed: if reconSuccess { ... MarkReconciled(..., StateCompleted, ...) }`
  - `internal/app/services/disbursement/wallet_payment_service.go:700-710` — comment block documents `failed + recon success → override to completed` as the explicit designed behavior
  - `internal/infra/persistence/tx_wallet_payment_repository.go:283-290` — `WHERE id = ?` only; `RowsAffected == 0 → ErrNotFound`
  - All 6 call sites are in `wallet_payment_service.go:724,734,748,757,765` — `MarkReconciled` has zero callers outside `ReconcilePayment`
- **Suggested fix:** The allowlist MUST be enumerated BEFORE implementation, with sign-off, as: `allowedPriorStates = {failed, authorised, verified, completed}` (i.e. every state `ReconcilePayment` actually dispatches from). Promote "enumerate exact legal transitions" from a buried implementation step to a plan-level prerequisite; until that enumeration exists and is reviewed against `ReconcilePayment`'s switch, H7 is not safe to land. Add a regression test that exercises each transition in the `ReconcilePayment` switch end-to-end.

---

## Finding 5: M9's nonce store has no specified wiring point, and the OnePay adapter has no Redis dependency — `parseAndVerifyWebhook` would need a signature change OR the nonce check moves to the handler, and the plan says neither

- **Severity:** High
- **Location:** Phase 2, "Architecture / M9" + "Implementation Steps / Step 4"
- **Flaw:** The plan says "`SETNX ipn:nonce:<jti> 1 EX 600`; reject if already exists." But `parseAndVerifyWebhook` (`internal/infra/disbursement/onepay/webhook.go`) has zero Redis/cache imports and operates purely on the parsed payload map — adding Redis means either (a) changing the function signature (cascading change to the `Provider.VerifyAndParseWebhook` interface, the 9Pay adapter, and the disbursement `WebhookHandler.Receive` caller), or (b) putting the nonce check in the disbursement webhook handler before/after `provider.VerifyAndParseWebhook`. The plan picks neither. Also, the IPN has no `jti` field — the `IPNPayload` struct uses `FundsTransferID`/`TransactionID`; the nonce key must be derived from one of those, not a non-existent `jti`.
- **Failure scenario:** Cook reads "add nonce store" and either (a) skips it because the wiring point is unclear (M9 silently unimplemented, "tests pass" because nothing asserts the Redis call), or (b) wires it into `WebhookHandler.Receive` but BEFORE `VerifyAndVerifyWebhook`, where a Redis outage causes the handler to fail-closed and reject ALL OnePay IPNs — blocking every legitimate disbursement confirmation until Redis recovers (cascading to reconcile-worker backlog and stuck `authorised` rows). Plan specifies no fail-open/fail-closed policy.
- **Evidence:**
  - `internal/infra/disbursement/onepay/webhook.go:1-10` — imports: only `encoding/json`, `fmt`, `strconv`, `time`, `infrastructure`. No cache/Redis.
  - `internal/infra/disbursement/onepay/webhook.go:15` — function signature `parseAndVerifyWebhook(payload map[string]any, partnerID, partnerKey string, now func() time.Time)` — no ctx, no Redis handle
  - `internal/infra/disbursement/onepay/provider.go:108` — `parseAndVerifyWebhook(payload, p.client.cfg.PartnerID, p.client.cfg.PartnerKey, p.client.now)` — the only call site
  - `internal/transport/http/handlers/disbursement/webhook.go:107` — `provider.VerifyAndParseWebhook(...)` — the handler-level site where Redis could be injected
- **Suggested fix:** Pick ONE site in the plan and state it explicitly. Recommended: nonce check in `WebhookHandler.Receive` AFTER `VerifyAndParseWebhook` succeeds, keyed on `event.RequestID + event.ProviderRef` (NOT `jti`). Specify fail-open (log + alert, accept IPN) so a Redis outage does not block disbursement reconciliation. The signature-change path (injecting Redis into the adapter) is rejected: it would force the same change on 9Pay's `parseAndVerifyWebhook`.

---

## Finding 6: M9's "server-clock ±60s" uses the wrong clock source — `parseAndVerifyWebhook` defaults `now` to `time.Now()`, NOT `clock.Now()`. Phase 4 cites `lesson_timezone_loc_local_prod_utc`; Phase 2 does not.

- **Severity:** Medium
- **Location:** Phase 2, "Architecture / M9" + "Implementation Steps / Step 3"
- **Flaw:** The plan says "server-clock ±60s; reject negative skew; cap expires" but the verification function uses `now = time.Now` as the default (`webhook.go:17`) and the provider injects `p.client.now`. The plan never audits whether `p.client.now` is `clock.Now()` (Asia/Ho_Chi_Minh) or `time.Now()`. `lesson_timezone_loc_local_prod_utc` documents that prod scratch containers run UTC while the DSN forces `time.Local` to Asia/Ho_Chi_Minh — so the "server clock" the plan references is ambiguous and could be off by 7 hours from the timestamp OnePay embeds in `X-OP-Date`. Tightening the window from ±900s to ±60s while the clock source is unverified will mass-reject legitimate IPNs in prod.
- **Failure scenario:** `p.client.now` resolves to a UTC clock on prod; OnePay `X-OP-Date` is in UTC (also unverified in the plan); window is tightened to ±60s; an NTP skew of >60s between prod and OnePay's clock rejects every IPN. Disbursements sit in `authorised` forever; the reconcile poller becomes the only status mechanism (per `lesson_onepay_ipn_investigation`), and any disbursement OnePay marked failed but we never learned about → money already debited, books say pending.
- **Evidence:**
  - `internal/infra/disbursement/onepay/webhook.go:17` — `now = time.Now` default
  - `internal/infra/disbursement/onepay/provider.go:108` — `p.client.now` injected (plan does not verify this value)
  - Phase 4 plan:27 — explicitly references `lesson_timezone_loc_local_prod_utc` and `clock.Now()`; Phase 2 has no such reference
- **Suggested fix:** Before tightening the window, the plan must (a) grep-verify `p.client.now` is `clock.Now()` and (b) log the `elapsed` distribution for 24h in prod at the current ±900s window before reducing it. Add an explicit fail-safe: if `elapsed` exceeds 2× the configured window for >5% of IPNs in an hour, auto-widen the window and alert.

---

## Finding 7: M8 amount cross-check site is ambiguous — IPN processing is async (`EnqueueIPNProcess`), so "in the IPN handler" could mean the webhook Receive handler OR the asynq IPN processor. Wrong site = no protection.

- **Severity:** High
- **Location:** Phase 2, "Architecture / M8" + "Implementation Steps / Step 1"
- **Flaw:** Plan says "in the IPN handler, after parsing, assert `event.Amount == persistedRequest.Amount`." But `WebhookHandler.Receive` does NOT apply the IPN to the FSM directly in the happy path — it calls `h.asynqClient.EnqueueIPNProcess(...)` (`webhook.go:152`) and only falls back to `applyIPNSync` if asynq fails (`:162-174`). The amount check must therefore run in the **asynq IPN processor** to cover the production path. If the cook puts the check in `Receive`, a forged IPN with a tampered amount still gets enqueued and applied by the worker — the protection is theater.
- **Failure scenario:** Attacker forges an IPN (signature oracle risk noted in audit Low-bullets) with amount=1 VND on a 10M VND disbursement. The cross-check in `Receive` loads the persisted request, sees mismatch, returns non-2xx — but asynq is the happy path; the forged payload was already parsed and the `WalletIPN` audit row created. If the cook instead places the check in `Receive` BEFORE `VerifyAndParseWebhook`, signature verification hasn't run yet and the check itself becomes a signature oracle. If the check is placed in the asynq processor, it covers prod — but the plan doesn't name that file.
- **Evidence:**
  - `internal/transport/http/handlers/disbursement/webhook.go:107` — `provider.VerifyAndParseWebhook` (parse site)
  - `internal/transport/http/handlers/disbursement/webhook.go:126-149` — `WalletIPN` audit row created BEFORE any amount validation
  - `internal/transport/http/handlers/disbursement/webhook.go:151-172` — async enqueue is the happy path; sync fallback only on asynq failure
- **Suggested fix:** Name the asynq IPN processor file as the M8 check site. Order: (1) signature verify, (2) parse, (3) load persisted request by `event.RequestID`/`event.ProviderRef`, (4) amount + currency cross-check, (5) apply via `RecordIPN`. Reject on mismatch with non-2xx + CRITICAL alert. Drop the audit-row write until AFTER the cross-check passes (currently it's written before, leaking a forged payload into audit logs).

---

## Finding 8: H3 `gps_at` freshness check requires changing `GeoReading` or `validateGeofence` signature — the plan edits `attendance_service.go:294-320` without noting that `validateGeofence` doesn't receive `gps_at` today

- **Severity:** Medium
- **Location:** Phase 4, "Architecture / H3" + "Related Code Files"
- **Flaw:** The plan says "in the check-in path (`attendance_service.go` 294-320 + handler `attendance.go` 153-212), cross-check `gps_at` against `clock.Now()`." But `validateGeofence(project, reading)` (line 294) takes a `domain.GeoReading` — and `GeoReading` has `Lat`, `Lng`, `Accuracy` only (confirmed by `:301-302` reading `reading.Accuracy`, `:307` reading `reading.Lat/Lng`). There is no `gps_at` field. Adding the freshness check means either (a) extending `GeoReading` with a `TakenAt time.Time` field (rippling through every caller and test), or (b) adding a new parameter to `validateGeofence`. The plan describes neither.
- **Failure scenario:** Cook starts H3, opens `validateGeofence`, discovers `gps_at` isn't on the struct. Either improvises (passing `gps_at` as a separate arg, breaking the existing 6-call shape) or — worse — pulls `gps_at` from the request DTO at the call site and compares to `clock.Now()` without going through `validateGeofence`, leaving a parallel validation path that the next cook doesn't know about. Either way the test "stale gps_at rejected" passes against the new code, but legacy check-in callers (auto-reject re-check-in path, admin override if not retired) bypass the freshness check.
- **Evidence:**
  - `internal/app/services/attendance/attendance_service.go:294` — `validateGeofence(project *domain.Project, reading domain.GeoReading)` — `GeoReading` carries no timestamp
  - `internal/app/services/attendance/attendance_service.go:301-302,307,311` — only `reading.Accuracy`, `reading.Lat`, `reading.Lng` accessed
- **Suggested fix:** Specify in the plan: extend `domain.GeoReading` with `TakenAt time.Time` (or add a sibling `GeoReadingWithTimestamp`), audit every caller (check-in, check-out, auto-reject re-check-in, `LogDeviceAttempt`), and pass the timestamp through. If admin override is retired (M1) before H3 lands, scope the change to just check-in/check-out to reduce blast radius.

---

## Finding 9: Phase 3 H10 — "delete line 119" is line-number-fragile; the policy file is 158 lines and the grant line is identifiable only by content. Casbin caches policy in-process; restart coordination unspecified.

- **Severity:** Medium
- **Location:** Phase 3, "Architecture / H10" + "Implementation Steps / Step 3"
- **Flaw:** The plan says "delete `configs/casbin_policy.csv:119`" citing the audit's line number. The actual file is 158 lines (verified) and the audit cited lines 118-119 — line numbers in this file have shifted before. The grant is `p, adv_partner, /api/v1/users/*, PUT, allow` (verified by content match). Worse, `lesson_partner_403_edit_requests` documents that Casbin caches policy in-process and a **server restart is required** for policy changes to take effect — the plan says "Restart policy load (Casbin caches — server restart)" but never says whether the restart must coincide with the deploy, nor what happens if a stale process keeps serving with the old policy between deploy and restart.
- **Failure scenario:** Between the time the new image deploys (with the policy line deleted) and the Casbin-enforcer being re-initialized, in-flight requests still match the cached policy. More concretely: if a cook deletes by line number and the file has shifted, they delete the wrong line — possibly the scoped `/api/v1/adv-partner/users/*, *, allow` grant — and adv_partner loses all self-service user management in prod with no test catching it (CI tests the deleted-line case, not the deleted-content case).
- **Evidence:**
  - `backend/configs/casbin_policy.csv` — file is 158 lines, contains `p, adv_partner, /api/v1/users/*, PUT, allow` (verified by grep)
  - `lesson_partner_403_edit_requests` — "deny-overrides model, keyMatch2 no prefix-match, server restart needed"
- **Suggested fix:** Mandate content-match deletion (`grep -n 'adv_partner, /api/v1/users/\*, PUT, allow'`), not line number. Add a unit test that loads the CSV directly and asserts the line is gone AND that the scoped `/api/v1/adv-partner/users/*` grant still resolves. Document that the deploy must include the backend restart (it will, since it's a new container, but state it).

---

## Finding 10: No deploy/rollback runbook for the migration-before-backend ordering, and no monitoring/rollback criterion specified for any phase

- **Severity:** Medium (deploy hole)
- **Location:** plan.md "Verification posture" + all phase "Risk Assessment" sections
- **Flaw:** The plan's entire deploy/rollback guidance is: "after merge, `cd backend && make push && make deploy`, then re-run the `verify-fix-deployed-prod` timestamp check." There is no rollback runbook for any phase, no monitoring criterion ("if X metric spikes, roll back"), and no coordination note for the migration-before-backend ordering required by H5 (Finding 2). `lesson_apply_sql_migration_demo_prod` and `lesson_demo_deploy_gotchas` are both directly relevant and neither is cited. H3's ±60s window tightening (Finding 6) and M12's fail-closed both need staged rollout / rejection-rate monitors — the plan says "env-configurable threshold + rejection-rate monitor" for M9 but not for H3 or M12.
- **Failure scenario:** H5 migration applies cleanly on demo, backend deploy references the new column, but the new backend image fails to start for an unrelated reason (e.g. M12 fail-closed rejects boot because a schedule is missing in demo env). Migration is already applied; rollback requires the `.down.sql` that doesn't exist; demo is down. Or: H3 ±60s lands, prod field devices have 90s clock skew, check-in rejection rate spikes from 0.1% to 8%, no monitor fires, drivers cannot check in for a full shift before someone notices.
- **Evidence:**
  - plan.md:70-72 — entire deploy section
  - Phase docs: every "Risk Assessment" lists mitigations but none specifies rollback steps, monitoring queries, or rollback thresholds
- **Suggested fix:** Add a "Deploy & Rollback" section to plan.md covering: (a) migration ordering + paired `.down.sql` requirement (Finding 2), (b) staged rollout for H3/M9 window-tightening with explicit rollback threshold (e.g. "if check-in rejection rate > 1% for 30 min, set H3_GPS_WINDOW_S=300 and redeploy"), (c) M12 fail-closed behind a flag for first deploy (the plan's own Risk Assessment mentions this but doesn't make it a step), (d) cite `lesson_apply_sql_migration_demo_prod` and `lesson_demo_deploy_gotchas`.

---

## Summary of must-fix-before-land items

1. Finding 1 (Critical): re-verify H5 target path. If `FlexPaySettlementService` is legacy, H5 closes nothing.
2. Finding 2 (Critical): name the H5 target table, drop the Postgres-only partial index, mandate `.down.sql` + idempotent apply runbook.
3. Finding 4 (High): enumerate H7 allowed-prior-states BEFORE implementation; ensure `failed → completed` is preserved.
4. Finding 5 (High): pick the M9 nonce-site explicitly; specify fail-open on Redis outage.
5. Finding 7 (High): name the M8 check site (asynq IPN processor, not webhook Receive).

Status: DONE
