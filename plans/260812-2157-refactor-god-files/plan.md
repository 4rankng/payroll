# Refactor Top-3 Backend God Files

> Behavior-preserving concern split. Move code, don't rewrite. Same inputs, same outputs.

## Context

Three application-service god files have low cohesion and impede review/onboarding:

| File | LOC | Funcs | Tests | Risk |
|------|-----|-------|-------|------|
| `internal/app/services/attendance/attendance_service.go` | 1703 | 45 | 7 test files + integration | Low |
| `internal/app/services/bcc_import_service.go` | 1252 | ~27 | service + helpers + 3 flows | Medium |
| `internal/app/services/bcc_import_weekly.go` | 1477 | ~15 | weekly + payrate + flow | Medium |

All targets are **unmodified** in the current tree. Scope chosen by user: top-3 backend.

## Goal & Non-Goals

**Goal:** each god file split into cohesive single-concern files in the **same package, same type** — pure file moves verified green per commit.

**Non-goals (explicit):**
- Type decomposition (splitting `AttendanceService`/`BCCImportService` via composition) — bigger wiring change, optional follow-up.
- Shrinking the two bcc weekly mega-functions via extract-method — separate riskier step (Phase 3b).
- Any behavior change. Refactor only.
- Cross-layer moves. All work stays inside `internal/app/services/`.

## Constraints (repo non-negotiables)

- Domain layer stays framework-free (N/A here — services only).
- `clock.Now()` preserved on any moved business-logic line.
- Cache invalidation stays after tx commit (moved code already follows this).
- Conventional commit format, no AI refs. One file-split per commit.

## Precondition

**Commit or stash current uncommitted work first** (~30 files: Zalo, wallet_bulk, bcc siblings). The refactor must land on a reviewable, isolated diff. `bcc_import_weekly.go` / `bcc_import_service.go` are clean, but sibling bcc files (`weekly_payment_parser.go`, `diagnose_bcc.go`) are dirty.

---

## Status

- **Phase 1 — DONE ✅** (commit `f16e232`). `attendance_service.go` 1703→98 LOC, split into 8 cohesive files (largest 352). Verified green.
- **Phase 2 — DONE ✅** (commit `88521f0`). `bcc_import_service.go` 1252→58 LOC core, split into 5 files (lifecycle/process/replacement/result + core). Independent code-reviewer multiset proof: every body line identical to original.
- **Phase 3 — DONE ✅** (commit `59d346c`). `bcc_import_weekly.go` 1477→1257 LOC; 13 helpers + `wbccRateKey` extracted to `bcc_import_weekly_rates.go`. Reviewer byte-diff: helpers SHA-match HEAD, mega-methods 0-diff.
- **Pushed to origin/main** 2026-08-12. Working tree clean.
- **Known limitation (deferred):** file-split fixed god *files*, not god *types* (`AttendanceService` still 45 methods, `processAssetData`/weekly mega-functions still long). Type decomposition + mega-function extract-method are optional follow-ups.

---

## Phase 1 — `attendance_service.go` split (lowest risk, proves pattern)

Same package `attendance`, methods stay on `*AttendanceService`. New files:

| New file | Functions moved |
|----------|-----------------|
| `attendance_checkin.go` | `CheckIn`, `AdminCreateCheckIn`, `AdminCheckInShifts`, `adminCheckInShifts`, `completeLegacyApprovedOpenAttendance`, `resolveShift`, `resolveProject`, `ResolveProjectID` |
| `attendance_checkout.go` | `CheckOut`, `CancelCurrentAttendance`, `ResolveCheckoutProjectID`, `validateGeofence` |
| `attendance_review.go` | `Approve`, `Reject`, `autoRejectReasonFor`, `AutoRejectIfExpired`, `AutoRejectSweep`, `resolveShiftForAttendance`, `normalizeLegacySalaryRejectReason` |
| `attendance_quota.go` | `CreditAttendanceQuota`, `CreditScheduledAttendanceQuota`, `creditAttendanceQuota`, `CreditOverduePendingQuota`, `selfCheckInAdvanceHoldDuration` |
| `attendance_query.go` | `GetTodayAttendance`, `GetByID`, `List`, `calculateEarningAmount` |
| `attendance_service.go` (slimmed) | struct def, `NewAttendanceService`, imports, type-level helpers |

Private types (e.g. `parsedShift`) move to whichever file uses them, or a small `attendance_internal.go`.

**Verify:** `go build ./...` + `go test ./internal/app/services/attendance/... -race` (7 packages green).

## Phase 2 — `bcc_import_service.go` split

Same package, same `*BCCImportService` type. New files:

| New file | Functions moved |
|----------|-----------------|
| `bcc_import_lifecycle.go` | `AcceptUpload`, `ProcessUpload`, `ProcessPendingJob`, `processAssetData`, `RecoverPendingJobs`, `finalizeImportJob`, `completeSkippedBCCImport`, `updateAssetMetadata`, `prepareImportContext`, `ensureEmployeeUserAccount` |
| `bcc_import_replacement.go` | `planBCCReplacement`, `applyTimesheetReplacement`, `applyTimesheetReplacementPrepared`, `requireBCCImportApproval`, `bccEntryReplacementKey`, `bccTimesheetReplacementKey`, `importErrorsFromBulkFailures`, `safeBulkFailureReason` |
| `bcc_import_result.go` | `resultForAsset`, `PendingTerminalAudit`, `MarkTerminalAuditLogged`, `failedBCCImportResult`, `failWithImportErrors`, `safeWeeklyPaymentParseError`, `bccRequestFingerprint`, `stringPointer` |
| `bcc_import_service.go` (slimmed) | struct, `NewBCCImportService` |

**Verify:** `go build ./...` + `go test ./internal/app/services/... -run BCC` + integration flows `flow_bcc_import`.

## Phase 3 — `bcc_import_weekly.go` split

Extract the pure helper cluster; leave the two mega-orchestrators in place (flag for 3b).

| New file | Functions moved |
|----------|-----------------|
| `bcc_import_weekly_rates.go` | `shiftInConfig`, `buildShiftRatesForShift`, `canonicalBCCRateKeySegment`, `weeklyBCCWeekendFallbackKey`, `weeklyPaymentRateKey`, `buildWeeklyPaymentPositions`, `planWeeklyPaymentPositionCorrections`, `selectWeeklyPaymentPositionCorrections`, `getShiftTypes`, `determineDayType`, `findSTKRow`, `weeklyBCCMissingAssignmentError`, `isWeeklyBCCEmployeeBlocked` |
| `bcc_import_weekly.go` (slimmed) | `processWeeklyBCCUpload`, `processWeeklyPaymentUpload` (still long — see 3b) |

**Verify:** `go build ./...` + `go test ./internal/app/services/... -run Weekly` + integration `flow_bcc_weekly_import`, `flow_bcc_weekly_payment_import`.

### Phase 3b (optional, separate decision) — mega-function extract-method
`processWeeklyBCCUpload` (645 LOC) and `processWeeklyPaymentUpload` (586 LOC): extract named sub-steps. Higher risk — requires reading the bodies carefully and keeping params/threading identical. Propose separately after Phases 1–3 land green.

## Final Verification

- `cd backend && go build ./...`
- `cd backend && go test ./internal/app/services/... -race`
- `make api-test` (full integration suite — catches cross-module regressions)
- `go vet ./...`

## Risk & Rollback

- **Risk:** mis-grouping a private helper causes a compile error — caught immediately by `go build`. No runtime behavior change possible (pure moves, same package/type).
- **Rollback:** `git revert <commit>` per phase. Each phase is independent.
- **Weakest link:** Phase 3b (mega-functions) — deferred. Phases 1–3 are low-risk file moves.

## Commit Sequence

1. `refactor(attendance): split attendance_service.go by concern`
2. `refactor(bcc): split bcc_import_service.go by concern`
3. `refactor(bcc): extract weekly rate/position helpers`

Each gated on green tests before the next starts.
