# Live Local-Dev E2E — Excel Fixture Upload Sweep (2026-09-06)

**Goal:** run every file in `testplan/excelfixture/` through the real HTTP
endpoint of the live local dev backend (post excel-refactor, commits
`f39cdb72..7836efb7` on `main`), proving the full
`uploadguard → AcceptUpload → asynq worker → parse → DB` pipeline on real
partner files — not just the in-process unit/golden tests.

**Environment (verified before writing this plan)**

- Backend: live `backend/tmp/main` (air) PID 75891, `:8080`; MySQL
  `payroll-mysql` + Redis `payroll-redis` in Docker; DB is a prod-restored
  sandbox — mutating it is in-bounds.
- Auth: admin `frankng` / `Admin123` (`POST /api/v1/auth/login`), same defaults
  the integration suite uses. Admin bypasses the partner project-permission
  gate in the handler.
- Endpoint: `POST /api/v1/timesheets/partner-import` — multipart `file`,
  `project_id`, `for_month=2026-08`; required header `Idempotency-Key`.
  Accept → 202 + `data.id`; poll `GET /api/v1/timesheets/partner-import/{id}`
  until `status` ∈ {completed, failed}.
- Each upload uses a fresh Idempotency-Key; uploads run sequentially (per
  project+month `active_scope_key` uniqueness forbids concurrency anyway).

## Cases

All 5 fixtures are T08.2026 → `for_month=2026-08`. Expected routing column
comes from the characterization suite (`routing_characterization_test.go` /
goldens), which the live run cross-checks only via observable outcome — the
API response does not expose the detected format, so routing is evidenced by
successful completion on the correct processor plus sane counts.

| # | Fixture (testplan/excelfixture/) | Project (id) | Expected routing | Accept | Terminal | Total/Created/Skipped/Err | Verdict |
|---|----------------------------------|--------------|------------------|--------|----------|---------------------------|---------|
| 1 | BCC BUMHAN Thang8.xlsx (612 KB) | BUMHAN THỢ LẮP RÁP (73)¹ | date-row (winner over legacy detect; "BCC"-named sheet) | 202 | failed | 17 / 0 / 0 / 107 | ROUTE OK — blocked by payrate config state (F1) |
| 2 | BCC+LGD+Mr+Đức+-31.07+-+lương+tuần+up.xlsx | LGDISPLAY (70)² | weekly BCC | 202 | completed | 122 / 0 / 134 / 0 | PASS (on correct project) |
| 3 | BCC+LƯƠNG+TUẦN+KỲ+4+DỰ+ÁN+EVA+T08.2026.xlsx | EVA KCN Vsip (10) | legacy | 202 | completed | 42 / 0 / 1092 / 0 | PASS |
| 4 | BCC+LƯƠNG+TUẦN+KỲ+4+PQC+HẢI+PHÒNG+THÁNG+08.2026.xlsx | PQC Hải Phòng (7) | multi-position | 202 | completed | 18 / 0 / 82 / 0 | PASS |
| 5 | BCC+THÁI+BÌNH+DƯƠNG+LƯƠNG+TUẦN+KỲ+4+THÁNG+08.2026.xlsx | Thái Bình Dương (74) | weekly payment | 202 | completed | 29 / 45 / 116 / 0 | PASS (full create path) |

Import job ids 543–548 (6 uploads: BUMHAN first targeted 56, re-run targeted 73).
All accepts: HTTP 202, zero guard rejections, zero 5xx. Server PID 75891 @
`7836efb7`, clean tree throughout.

## Results (filled 2026-09-06 ~11:30)

**Skipped-count semantics.** The "before" snapshot query in this plan was
broken (wrong column name, `date` not `work_date`), but post-state + created
timestamps reconstruct it: projects 7/10/56 carried prod-restored **approved**
T08 timesheets (2037/429/865 rows). The import's replacement contract keeps
approved rows and skips them — EVA's 1092 and PQC's 82 skips are that exact
semantic, with `errors=0`. TBD (74) had no approved T08 coverage for the
file's dates, so it exercised the full create path: 45 timesheets created.

### Finding F1 — BUMHAN T08: payrate labels vs config window (not a regression)

- First run (project 56 "BUMHAN") failed 161× "Không thể tạo bảng chấm công"
  (default bucket) — root cause: the file's 17 workers are assigned to 73
  (BUMHAN THỢ LẮP RÁP) / 78 (PA BUMHAN), not 56. Correct domain rejection.
- Re-run (project 73, where all 17 are assigned + payrate chain covers T08)
  failed 107× "không tìm thấy mức lương cho ca NT / ca T7": the file's hour
  labels are `NT`/`T7`, but the payrate row active for its dates 08-22..28
  (`payrates.id=73`, window 08-15→08-28) is keyed `ca ngày/ca đêm/tăng ca`;
  `NT`/`TC` keys only take effect 08-29 (`payrates.id=78`). The 09-04
  "update-as-create splits config" reshape (known event) left this August
  file unresolvable under the current config.
- Non-regression evidence: the failing loop is `bcc_import_process.go` legacy
  resolution; window-diff vs `401f3093` (pre-refactor) shows only the
  documented Phase-3 helper extractions (`crossCheckSTKName`,
  `planMonthReplacement`) — resolution loop unchanged; parse layer locked by
  the byte-identical golden for this exact file.

### Finding F2 — LGD weekly: RESOLVED — fixture belongs to LGDISPLAY (70), not LGD (58)

- Original run (project 58 "LGD") failed `weeklyBCCConfigError`: sheet shift
  `OT390` absent from 58's payrate keys. **Read-only prod check (2026-09-06):
  prod project 58 has the byte-identical config — the file fails on prod too;
  not a restore artifact and not a regression** (`shiftInConfig` untouched by
  the refactor at identical lines; parse golden-locked).
- "LGD" in the filename is LG**DISPLAY** (project 70), whose payrate carries
  the `HC/OT30…OT390` shift keys — and which owns all 2,357 OT-keyed
  timesheets since July (project 58: zero). Re-run against 70 with
  `for_month=2026-07` (file week ends 31.07): **completed, 122 rows,
  0 errors, 134 approved-skips** (import job 554).
- Pattern note: partner filenames use short ambiguous project aliases
  (BUMHAN→THỢ LẮP RÁP 73, LGD→LGDISPLAY 70) — resolve the target project from
  `project_employees`/payrate keys, not the filename.

¹ Correct target established from `project_employees` coverage (17/17 file
workers on 73, payrate windows spanning the file dates; 78's payrates start
2026-09-04, after the file's dates).
² Correct target established from payrate shift keys (70 carries
`HC/OT30…OT390`; 58's config lacks `OT390` on prod and local alike) and
`timesheets` shift-key history (2,357 OT-keyed rows on 70 since July, zero on
58). Uploaded with `for_month=2026-07` — the file's week ends 31.07.

## Pass criteria (per case)

1. Upload accepted: HTTP 202 (guard must NOT reject these real files — all
   .xlsx, well under the 20 MiB cap). Any 4xx from the guard = FAIL.
2. Job reaches terminal state within 120 s; `status=completed` = PASS.
   `status=failed` = FAIL (capture `error_detail` for diagnosis).
3. Counts recorded as observed. Non-zero `skipped`/`error_count` is NOT an
   automatic failure on this DB: it is prod-restored data, so name cross-check
   warnings ("có thể sai CCCD"), payrate gaps, or already-approved rows can
   legitimately skip — record and explain each.
4. No 5xx anywhere in accept or poll.

## Snapshot evidence

- `timesheets` row counts per project for 2026-08 before vs after (replacement
  semantics: pending rows for the month are replaced, approved rows kept).
- `backend/logs/app.log` scanned for each import id (ERROR lines attributable
  to the run are findings, not noise).
- Server PID recorded at start; if air rebuilds mid-run (background
  `/code-review --fix` may touch code), the affected case is re-run and noted.

## Out of scope

- Guard negative paths (oversized / wrong-magic) — covered by
  `uploadguard_test.go`; this sweep is the happy-path real-file matrix.
- Frontend UI flows; format-string assertions (locked by goldens).
