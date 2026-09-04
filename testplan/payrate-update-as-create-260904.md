# Test Plan — Payrate update-as-create split + closed-config editing

Date: 2026-09-04 · Scope: `UpdateEffectiveDatedPayrate` split semantics (`5e2783fc`),
plus the follow-up fix for editing ended (closed) configs. Project 73 local dev is
the reference data: `67 [26/07–14/08]` · `73 [15–21/08] CLOSED` (136 paid rows) ·
`87 [22/08→open]` (155 pending rows).

Legend: [A]=automated unit (sqlite harness) · [API]=curl vs :8080 · [UI]=browser on :3000
· [DB]=SQL assertion · status: ☑ pass / ⌀ N/A / ✗ fail

## Fixes under test (implement before executing)

- **F1 backend**: `UpdateEffectiveDatedPayrate` rejects any PUT on a config with
  `to_date` set (ended) with a dedicated clear message — replaces the two
  confusing errors ("đã có cấu hình... muộn hơn" for pre-clamped saves,
  "phải sau ngày chấm công..." for unchanged saves).
- **F2 admin UI**: edit page for an ended config — no floor pre-clamp (shows
  stored date), date field read-only, Save disabled, hint shown. Open configs
  keep today's behavior (pre-clamp to floor when stored < floor, editable).
- **F3 mobile UI**: same as F2 on `pages/mobile/admin/PayrateEditPage`.

## 1. Update semantics matrix (backend) — the model

| # | Case | Expected | Type | Status |
|---|------|----------|------|--------|
| 1.1 | Open config, from moves later, paid+pending linked | Split: old closed at from−1, new open row; paid rows stay on old (same payrate_id, rate, amount); mutable rows ≥ from re-linked to new + re-priced | A | ☑ |
| 1.2 | Open config, from moves later, no timesheets | Split still happens; recalc no-op | A | ☑ (dedicated test: TestPayrateTemporalServiceSplitsConfigWithNoLinkedTimesheets) |
| 1.3 | Open config, from unchanged, rates changed | In-place save; mutable rows in reign re-priced | A | ☑ (existing) |
| 1.4 | Open config, from moves earlier, ≥ floor | In-place; predecessor `to_date` synced to from−1; mutable rows ≥ from re-linked | A | ☑ (existing) |
| 1.5 | Open config, from earlier than floor | Reject: "phải sau ngày chấm công đã trả lương gần nhất" | A+API | ☑ (existing) |
| 1.6 | from == floor exactly (22/08 case) | Allowed (split) | API | ☑ verified on 73→87 |
| 1.7 | Ended config, from later / unchanged / earlier | Reject with F1 message, DB unchanged | A+API | ☑ |
| 1.8 | Mutable row's paytype missing in new rates | Whole tx rolls back (no partial split) | A | ☑ (existing rollback test) |
| 1.9 | Sibling config starts on/after submitted from | Reject conflicting-config | A | ☑ (existing) |
| 1.10 | from == another config's from (duplicate date) | Reject duplicate-date | A | ☑ (existing create-side; update-side via 1.9) |

## 2. Closed-config UX (F2/F3)

| # | Case | Expected | Type | Status |
|---|------|----------|------|--------|
| 2.1 | Open edit page of ENDED config 73 | Shows stored 15/08/2026 (NOT pre-clamped to 22/08); date read-only; Save disabled; hint visible | UI | ☑ |
| 2.2 | Open edit page of CURRENT config 87 | Shows 22/08/2026; date editable; no lock icon; Save enabled | UI | ☑ |
| 2.3 | Edit page, open config, stored from < floor | Pre-clamped to floor (existing behavior kept) | UI | ☑ (observed pre-fix) |
| 2.4 | Mobile edit page of ended config | Same as 2.1 | UI | ☑ |
| 2.5 | Rates matrix on ended config | Read-only (pointer-events none via ratesLocked path or isEnded) | UI | ☑ |

## 3. Validate dry-run contract (`POST /payrates/:id/validate`)

| # | Case | Expected | Type | Status |
|---|------|----------|------|--------|
| 3.1 | Update dry-run, from later than earliest linked | **ok** (linked cap gone — split handles it) | API | ☑ |
| 3.2 | Update dry-run, from < floor | error, min_value + suggested_value = floor (single direction, no ping-pong) | API | ☑ |
| 3.3 | Update dry-run, from ≥ floor | valid:true | API | ☑ |
| 3.4 | Create dry-run, from < floor | unchanged behavior (error + floor suggestion) | API | ☑ |
| 3.5 | Completed/cancelled project | all fields locked:true (unchanged) | API | ⌀ N/A locally (no completed project with payrates in dev data); projectBlocked branch untouched, verified by inspection |

## 4. API contract / response shapes

| # | Case | Expected | Type | Status |
|---|------|----------|------|--------|
| 4.1 | GET open & closed payrate | `PayrateResponse` shape unchanged; `from_date_locked:false`; closed one has toDate set | API | ☑ |
| 4.2 | PUT split success | 200; response = NEW config id, fromDate=new, toDate:null | API | ☑ |
| 4.3 | PUT ended config | 400 + F1 message (both from-later and from-unchanged shapes) | API | ☑ |
| 4.4 | DELETE config used by timesheets | still rejected (unchanged) | API | ☑ (existing suite) |
| 4.5 | Partner role: PUT config not created by partner | 403 (unchanged) | A | ☑ by inspection (role guard precedes all changed code) |

## 5. Data integrity after every mutating case

| # | Invariant | Method | Status |
|---|-----------|--------|--------|
| 5.1 | ≤ 1 open config per project (`to_date IS NULL`) | DB | ☑ |
| 5.2 | Contiguity: each predecessor `to_date` = successor `from − 1` day | DB | ☑ |
| 5.3 | Paid/approved rows: payrate_id, rate, amount never changed | DB (vs pre-state) | ☑ |
| 5.4 | Mutable rows ≥ new from: re-linked + amount = hours×rate (flexible: = rate) | DB | ☑ |
| 5.5 | No timesheet references missing payrate_id | DB | ☑ |

## 6. Full-browser end-to-end (the reported scenario)

| # | Case | Expected | Type | Status |
|---|------|----------|------|--------|
| 6.1 | Edit 87 (current), save unchanged → success, back to project page | UI | ☑ |
| 6.2 | Edit 87, move from later (e.g. 25/08), save → split; 5.1–5.5 hold; pending rows 22–24 stay on 87, 25+ on new | UI+DB | ☑ |
| 6.3 | Edit 87, set from < floor → red error + "Dùng giá trị" = floor; click → value applied; save succeeds | UI | ☑ |
| 6.4 | Edit 73 (ended) → read-only per 2.1 (no error reachable) | UI | ☑ |
| 6.5 | Project payrate list shows all three configs with correct ranges | UI | ☑ |

## 7. Regression gates

| # | Gate | Status |
|---|------|--------|
| 7.1 | `go build ./...` + `go vet` (3 pkgs) | ☑ |
| 7.2 | Unit: `go test ./internal/pkg/services/ ./internal/transport/http/handlers/... ./internal/app/services/payroll/...` | ☑ |
| 7.3 | Integration: `cd backend && go run ./tests/integration/` — 275/0 baseline | ☑ |
| 7.4 | Frontend `pnpm lint && pnpm type-check` (F2/F3 touch TSX) | ☑ |

## Execution notes

- 6.2 is destructive to the reference state: snapshot project-73 tables first,
  restore after (pattern: `_bak_` tables, drop when green).
- 6.3/6.6 mutate 87's from: same snapshot/restore pattern.
- UI cases run on `localhost:3000` with `auth_token` injected (frankng/Admin123).
- Mobile page: same URL under 375px viewport (ResponsivePage renders mobile component).

## Results (executed 2026-09-04, all against hot-reloaded :8080 + :3000)

- **4.3 initially FAILED**: ended-config PUT with unchanged from returned the
  floor error — the ended-guard ran after `validateEffectiveDateTx`. Fixed by
  reordering (load existing + ended-guard first). Both 4.3 shapes now return
  the dedicated message.
- 6.3 note: floor error + quick-fix verified end-to-end (apply → save → success).
- 6.2 invariants after browser split @25/08: 1 open config, contiguous
  timeline (73→21/08, 87→24/08, 89 from 25/08), 0 paid rows changed, 0
  dangling refs. State restored afterwards; bak tables dropped.
- 2.3 verified pre-fix (config 73 pre-split, open, stored 15/08 < floor →
  pre-clamped to 22/08); behavior intentionally retained for open configs,
  removed for ended ones.
- 6.5: timeline data verified via API/DB; project details sheet renders
  salary-period rows (Georim/SAM SUNG), not the raw config list.
- Gates: build+vet ✓ · unit suites ✓ (12 temporal tests incl. 3 new) ·
  integration 275/0/21 ✓ · eslint+tsc ✓
