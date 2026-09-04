# 2026-09-04 — Payrate from_date_locked fix: live browser verification (550e1cf7)

Follow-up verification session for the locked "Từ ngày" field fix committed as
`550e1cf7` (fix(payrate): unlock effective_from when a valid start date still
exists). Backend-only change: `fromDateLocked()` now locks only when the legal
window `[paid floor, earliest linked timesheet]` is genuinely empty
(`startDateWindowEmpty`), instead of locking whenever the *stored* date sat
outside that window. Admin + mobile payrate edit pages both consume
`from_date_locked`, so both were fixed by the one change.

## Live walkthrough on localhost:3000 (fresh repro, this session)

1. Seeded exact broken state: project `1000005` / payrate `85`
   (`from_date = 2026-08-29`) / linked **unpaid** timesheet `62517` on
   `2026-08-15` → stored start *after* its own earliest linked timesheet.
2. `GET /api/v1/payrates/85` → `from_date_locked: false` (pre-fix: `true`).
3. Browser (agent-browser, `auth_token` injected into localStorage):
   edit page rendered 29/8/2026 with **zero `lucide-lock` icons** and no
   `disabled`/`readOnly` on the date inputs.
4. Save → original error reproduced verbatim: "Ngày 2026-08-29 nằm sau bảng
   công đã gắn với cấu hình này (ngày đầu tiên: 2026-08-15)." Field stayed
   editable — pre-fix this same state froze the input so the "Dùng giá trị"
   link was useless.
5. "Dùng giá trị: 2026-08-15" → date corrected, banner cleared. Save →
   success, navigated back to `/admin/projects`. API confirmed persisted
   `fromDate: 2026-08-15`.
6. All synthetic rows hard-deleted (0 residue); tests re-run green
   (`TestStartDateWindowEmpty`, 4 subtests incl. the regression case).

## Gotchas hit while seeding

- `payrates.live_from_date` and `open_project_id` are **generated columns**
  (3105 error if inserted) — insert only real columns.
- Timesheet `paytype` must be a full composite key
  (`phổ thông.ngày thường.ca ngày`), else the re-pricing guard rejects the
  save with "bảng công … có loại lương không tồn tại trong cấu hình mới".
  Hand-rolled SQL seeds must copy a real paytype, not invent `'Ca ngày'`.
- Local admin login for walkthroughs: `frankng` / `Admin123`
  (`make reset-passwords` convention), token at `POST /api/v1/auth/login`,
  SPA reads `localStorage.auth_token`.

## State at close

- `550e1cf7` local on `main`, ahead of `origin/main` by 1 — **not pushed**.
- Unrelated uncommitted work left untouched: `max-w-4xl` layout widening in
  `frontend/src/pages/admin/PayrateEditPage/index.tsx`.
- Still unresolved (separate issue): how production data legitimately reached
  `from_date` > earliest linked timesheet — all normal write paths reject it;
  suspects are import/older code paths.

## Round 2 (same day): update-as-create split — user-directed

User hit the remaining dead-end on REAL data (project/payrate 73): config
`[15/08→open]` with paid rows 15–21/08 and pending 22–28/08 all linked.
Floor demanded ≥22/08, linked-cap demanded ≤15/08 — validate suggested 15/08
and 22/08 at each other, no legal date. `from_date_locked` was `true` (my
round-1 fix's "genuinely empty window" case — correct under the old model,
wrong product behavior).

**User decision: "when update, treat it as create new payrate config."**

- `UpdateEffectiveDatedPayrate`: from-date moving LATER now splits — old row
  closed at from−1, new open row inserted (trim before insert; one-open-per-
  project), mutable (unpaid+unapproved) rows ≥ from re-linked + re-priced.
  Closed configs can't be split (explicit guard). In-place (≤) path unchanged.
- `/validate` update dry-run: linked-timesheet constraint deleted; only the
  paid floor applies (create parity).
- `from_date_locked` → constant `false` (split always offers a legal move);
  `fromDateLocked`/`startDateWindowEmpty` removed. Frontend already pre-clamps
  to the paid floor on load, so the edit page opens showing the target date.
- PUT response returns the NEW config id after a split (frontend navigates
  back regardless).

Verified on the real data, twice (API curl loop, then restored + full browser
walkthrough): 73 → `[15–21/08]` closed with its 136 paid rows untouched;
new config `[22/08→open]` with all 155 pending rows re-linked; 0 pricing
mismatches vs backup. Temporal tests: split test + closed-config guard test
added; all existing temporal/handler/payroll tests pass. Integration suite
`go run ./tests/integration/`: **275/0** (21 skipped).
