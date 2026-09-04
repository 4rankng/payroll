# BCC BUMHAN parse-failure — diagnosis

Source file: `BCC BUMHAN T09.2026 thợ phụ chốt ứng lương - Copy.xlsx` (attached by user, inspected directly with openpyxl — not committed to repo). Chat context: admin (HD) told Nguyễn Phương Anh the file is the shared template for 3 BUMHAN projects; app currently rejects/mis-parses it.

## 0. Executive summary

Two separate problems are tangled together in the chat:

1. **A code-level parser problem** — already fixed today, HEAD-of-branch. `date_row_bcc_parser.go` (commit `aed8a007`) now understands this exact template shape. Verify it's deployed/tested against the real file before calling this closed.
2. **A config/data problem, not a code bug (mostly)** — the project's payrate ("cấu hình lương") hourType leaf names don't match the shift-code text the BCC file uses (`NT`, `OT`, `T7`, `OT T7`, `CN`, `OT CN`). This is exactly what HD told Phương Anh to fix by hand in the chat. It's real and still open — nothing in the code auto-syncs payrate labels to a file's shift codes.
3. **One latent code bug found during this trace, not yet reported by the user**: when a shift-code label has no exact match in payrate, a fallback silently buckets `OT T7` / `OT CN` as generic overtime and bare `CN` as generic day-shift — losing the Saturday/Sunday premium without any error. This can produce **silently wrong pay**, not just a rejected import. See §3.
4. The `effective_from` lock error shown in the chat screenshot ("Ngày 2026-08-29 nằm sau bảng công... không thể muộn hơn 2026-08-15") is from **old code, already removed** (commits `26460dc7`→`5e2783fc`, further hardened by today's `cdb5c8e4`). It should not reproduce on current HEAD — needs a fresh repro to confirm, not a fix.

---

## 1. The uploaded file's actual structure

Workbook has **3 sheets**, only one of which is the current BCC format:

| Sheet | Content | Relevant? |
|---|---|---|
| `M1` | Current-format timesheet, Aug 21 – Sep 24 2026, the "BUMHAN" title | ✅ this is the real upload |
| `Truy lĩnh` | 2018 "BẢNG THANH TOÁN TIỀN LƯƠNG" payroll-payment table (BHXH/BHYT/BHTN deduction columns), dated Tháng 07/2018 | ❌ unrelated legacy sheet, carried along when the file was duplicated |
| `Công TrT T5` | 2018 "BẢNG CHẤM CÔNG THÁNG 05/2018", single-column-per-day-number layout (1–31), no NT/OT split | ❌ unrelated legacy sheet |

The filename itself is `...- Copy.xlsx` — Phương Anh duplicated an old multi-tab workbook and only edited the `M1` tab, leaving two stale 2018 sheets attached. This is the literal cause of "anyhow do" the boss referred to.

**`M1` sheet layout** (row-exact, verified with openpyxl `data_only=True`):

- Row 3: merged title "BẢNG CHẤM CÔNG DỰ ÁN BUMHAN" — generic, does **not** say which of the 3 BUMHAN projects this is for. Project identity is presumably chosen in the upload UI, not read from the file.
- Row 5: employee-info headers (`STT`, `Mã NV`, `Họ và tên`, `TK Ngân hàng`, `Ngân hàng`, `Mức lương /9h`, `Ngày thử việc`, `Ngày nghỉ việc`), then a day-block, then summary columns (`Tổng giờ công chính thức`, `OT 150%`, `T7+ OT T7`, `CN + OT CN`, `Ngày Lễ`, `Tổng công`, `Ghi chú`).
- Row 6: actual calendar dates as merged 2-column pairs (`2026-08-21`, `2026-08-22`, … `2026-09-24`) — a **rolling pay period spanning two calendar months**.
- Row 7: day-of-week token per pair (`6`=Fri, `7`=Sat, `CN`=Sun, `2..5`=Mon–Thu).
- Row 8: **shift-code label per individual sub-column, and this label changes depending on the day-of-week in row 7**:
  - Weekday (2,3,4,5,6) → `NT` (normal hours) / `OT` (overtime)
  - Saturday (7) → `T7` / `OT T7`
  - Sunday (CN) → `CN` / `OT CN`
- Rows 9+: one row per employee, hours (e.g. `9`, `1`, `8`, `2`, `4.5`) under the applicable shift-code column for that date.
- **No "Vị trí" (position) column anywhere in `M1`.** Every employee is a flat row.
- **Wage is NOT looked up by the parser from the file's `Mức lương /9h` column** — that per-employee inline number is present in the sheet but, per the code path traced below, the parser resolves pay via the *project's payrate config*, keyed by the row-8 shift label. The `Mức lương /9h` column is effectively cosmetic/informational to a human reader, not data the import pipeline consumes for rate resolution.

## 2. Format detection: confirmed this file routes correctly

`backend/internal/app/services/excel/format_detector.go` (`DetectFormat`) iterates all visible sheets and classifies each by fingerprint, in priority order `FormatLegacy > FormatWeeklyBCC > FormatWeeklyPayment > FormatMultiPosition > FormatDateRow`.

- `Truy lĩnh` and `Công TrT T5` have no full-date header row detectable by `parseExcelDate` in rows 2–10 (dates are stored as plain text/2018 layout, not date serials) → they fail `isDateRowBCCSheet` and every other fingerprint → correctly **excluded** from `DateRowSheets`.
- `M1` matches `isDateRowBCCSheet` (`format_detector.go:100-104`, fingerprint impl in `date_row_bcc_parser.go:250-278`): a date row with ≥3 full-date cells (rows 2–10) plus a shift-code row with ≥2 non-weekday short tokens directly below it.
- Result: `Format = FormatDateRow`, `DateRowSheets = ["M1"]`. The two legacy sheets do **not** break detection as currently coded.

So the multi-sheet-junk theory is **not** the live blocker (assuming HEAD is deployed) — worth confirming with a real re-upload, but the fingerprinting logic looks sound on inspection.

`ParseDateRowBCCFile` → `parseDateRowBCCSheet` (`date_row_bcc_parser.go:36-117`) then walks `M1`: finds the date row (`findDateRowAndCols`), the shift-code row directly below it (`findShiftCodeRow`), the employee-info columns (`findDateRowEmployeeCols`), and for every employee row emits one `BCCEntryData{DayNum, ShiftLabel, Hours, FullDate}` per non-zero cell in the day region (cols E–BH, i.e. 5–60). `ShiftLabel` is the **raw row-8 text**, e.g. `"NT"`, `"OT"`, `"T7"`, `"OT T7"`, `"CN"`, `"OT CN"` — untouched, not normalized to any internal enum at parse time.

This part (commit `aed8a007`, files with today's mtime) is new and was very likely written *in response to* the same BUMHAN file — it appears purpose-built for exactly this shape. No structural blocker was found here.

## 3. Where it actually breaks: label-keyed rate resolution

Downstream, `backend/internal/app/services/bcc_import_process.go:405-460` resolves each `BCCEntryData.ShiftLabel` to a VND rate. Chain (rateless-file path, i.e. no VND numbers embedded in the sheet — which matches `M1`, its cells are pure hour counts):

1. **`labelRateTarget(flatRates, entry.ShiftLabel)`** (`bcc_import_helpers.go:251-281`) — exact-match-first: canonicalizes the label (`canonicalBCCRateKeySegment`: NFC + Vietnamese-diacritic-fold, `bcc_import_weekly_rates.go:88-92`) and looks for a flattened payrate path `"position.dayType.hourType"` whose **`hourType` leaf equals the label exactly** (post-canonicalization). `flatRates` comes from the project's `Payrate.PayrateJSON` (`backend/internal/domain/payrate.go:371-386`, `Payrate.Flatten()`) — `hourType` is **free text the admin typed into the payrate editor**, not an enum. Multiple day-types can share the same label; `dayTypePriority` (`ngày thường` < `ngày nghỉ` < `ngày lễ`) breaks ties.
   - **This means: for the import to work, the project's payrate config must literally contain hourType leaves named `NT`, `OT`, `T7`, `OT T7`, `CN`, `OT CN`** (or whatever exact strings appear in row 8 of that partner's file). This is precisely what HD told Phương Anh to set up ("đổi tên ca ngày tăng ca cho đúng với ký hiệu trên file, ví dụ NT, OT, T7, OT T7, OT CN").
2. If step 1 finds no exact match, falls back to **`shiftLabelHourType(entry.ShiftLabel)`** (`bcc_import_helpers.go:109-124`) — a small fixed vocabulary built for a *different* partner template (Samsung SDS, see `lesson_bcc_partner_template_variants_2026-08-31`):
   - label contains `"OT"` (substring) → generic `"tăng ca"` bucket
   - label is exactly `CB`, `CN`, or `HC` (whole token) → generic `"ca ngày"` bucket
   - anything else → `false`, hard error `"không tìm thấy mức lương cho ca %s"`.

**Concrete effect on BUMHAN's exact 6 labels, if the payrate config does *not* have those exact leaf names:**

| File label | Step 1 (exact match) | Step 2 fallback | Result if payrate has no exact `NT`/`OT`/`T7 `/`OT T7`/`CN`/`OT CN` leaves |
|---|---|---|---|
| `NT` | fails | not `OT`, not `CB/CN/HC` | **hard error**, row rejected |
| `OT` | fails | contains `OT` → generic tăng ca | **silently mispaid** if a generic "tăng ca" bucket happens to exist; otherwise hard error |
| `T7` | fails | not `OT`, not `CB/CN/HC` | **hard error** |
| `OT T7` | fails | contains `OT` → generic tăng ca | **silently mispaid**: Saturday-OT premium collapses into ordinary OT rate |
| `CN` | fails | whole-token `CN` in the Samsung-vocabulary sense means something else (their "CN" ≠ "Chủ Nhật") → generic ca ngày | **silently mispaid**: Sunday work priced as an ordinary weekday, losing the Sunday premium — and produces **no error at all**, so nobody notices |
| `OT CN` | fails | contains `OT` → generic tăng ca | **silently mispaid**: Sunday-OT premium collapses into ordinary OT rate |

The `NT`/`T7` hard-error path is what's visibly "cannot parse" today. The `OT`/`OT T7`/`CN`/`OT CN` silent-fallback path is worse and not yet reported — worth calling out explicitly to whoever picks this up, since it produces wrong salary without surfacing anything to the admin. `shiftLabelHourType`'s doc comment (`bcc_import_helpers.go:101-108`) itself says it exists for *rateless* templates with a narrower vocabulary (`CB/CN/HC` vs `OT`) — it was not designed with BUMHAN's day-of-week-suffixed codes in mind, and the bare-`CN`-as-Sunday vs `CN`-as-Samsung's-code collision is a real ambiguity in the current fallback vocabulary, not a hypothetical.

## 4. Cross-project payrate divergence (3 BUMHAN projects)

Per the chat, HD observed the payrate ("cấu hình lương") differs across the 3 BUMHAN projects and wants them unified to the shape shown in the screenshot: `Vị trí` (position) × `Loại ngày` (Thường/Nghỉ/Lễ) × rate columns `CN đ/giờ`, `T7 đ/giờ`, `T7 TC đ/giờ`, `TC đ/giờ`.

This is consistent with §3's model: `PayrateConfiguration` is a free-form JSON tree flattened to `position.dayType.hourType → rate`. Whatever hourType leaf strings the admin enters in `PayrateEditPage` become the exact vocabulary the label-keyed resolver in §3 matches against. If the 3 projects' configs were built independently (different admins, different times) they likely have different hourType leaf spellings — some may already use `NT`/`OT`-style codes, others may use the generic "CA NGÀY"/"TĂNG CA" labels visible in the chat's screenshot 2, others something else again. **I did not query the DB to confirm the actual current leaf names per project** — that's a live-data check the fixing agent should do before touching anything (`payrates` table, `payrate_json` column, 3 BUMHAN project IDs).

This is a data-entry/config synchronization task, not a code bug — but if the intent is "the same BCC template should work for all 3 projects without per-project relabeling," the actual fix that removes the need for humans to keep 3 payrate configs' leaf-name vocab in lockstep with an Excel file forever is a product decision, not something to silently patch — flag it back to the user rather than assuming scope.

## 5. `effective_from` lock — already resolved history, not a live bug

Chat screenshot 4 shows: *"Ngày 2026-08-29 nằm sau bảng công đã gắn với cấu hình này (ngày đầu tiên: 2026-08-15). Ngày bắt đầu không thể muộn hơn 2026-08-15 vì đã có bảng công dùng cấu hình này."*

`git log -S "nằm sau bảng công"` shows this exact string was introduced in `26460dc7` ("feat(payrate): lock start date when timesheets are already linked") and **removed** in `5e2783fc` ("fix(payrate): split config instead of blocking later start-date updates"). It does not exist anywhere in current HEAD (`grep` across `backend/` and `frontend/src` — zero matches). Current validation lives in `backend/internal/transport/http/handlers/payrate_validate.go:314-342` (`applyCreateConstraints`) and produces a **different** message: `"Ngày %s có trước ngày trả lương gần nhất của dự án"` (a *before-the-paid-floor* rejection, not an *after-an-existing-config's-first-date* rejection) — semantically the opposite direction of lock. This was further hardened today by `cdb5c8e4` ("harden update-as-create flow after adversarial review") and `484c62e9` ("render ended configs read-only with a clear rejection message").

**This part of the chat is a historical symptom from earlier code, not a currently-reproducible bug.** Don't re-fix it — just confirm with a fresh repro against current HEAD/deployed build if the boss wants certainty, since the frontend (`PayrateEditPage/index.tsx`, currently has an uncommitted, unrelated `max-w-4xl` layout-width diff sitting in the working tree — not part of this issue) may still be running a stale bundle.

## 6. What's already fixed vs. what's still open

| Item | Status |
|---|---|
| Parser doesn't understand `M1`'s date-row + day-of-week-dependent shift-code layout | ✅ Fixed (`aed8a007`, `date_row_bcc_parser.go`) — needs a live re-upload test with the actual file, not yet verified end-to-end |
| Two stale 2018 sheets in the workbook confuse parsing | Not actually a bug given current fingerprinting (§2) — but worth confirming, and worth telling the user to stop shipping junk sheets regardless |
| Payrate hourType leaves don't match file's `NT/OT/T7/OT T7/CN/OT CN` codes | ❌ Open — config/data task for the admin, per-project, ×3 |
| Silent mispay when `OT`/`OT T7`/`OT CN`/`CN` fall through to the generic Samsung-style fallback vocabulary instead of hard-erroring | ❌ Open, newly identified, not yet reported by the user — recommend either tightening `shiftLabelHourType` to exclude ambiguous bare codes like `CN`, or removing the fallback entirely for `FormatDateRow`-sourced entries so any unmatched label hard-errors instead of silently mispricing |
| `effective_from` "date after existing config's first date" lock | ✅ Resolved historically (`5e2783fc`), further hardened today — needs a fresh repro to confirm on current build, not a code fix |
| 3 BUMHAN projects' payrate configs not synchronized | ❌ Open — data/product decision, not purely a code fix |

## Unresolved questions

1. Is `aed8a007`/`cdb5c8e4` actually deployed/running where Phương Anh is testing, or is she still hitting a pre-fix build? (Explains whether §5's old error is genuinely gone for her yet.)
2. Should `shiftLabelHourType`'s fallback be scoped only to the Samsung-style rateless format it was built for, and excluded for `FormatDateRow` imports (where an unmatched label should always hard-error, never silently substitute)? This is the highest-severity finding in this report (silent wrong pay) and needs a decision, not just a fix.
3. What are the 3 BUMHAN projects' current payrate `hourType` leaf names in the DB right now? (Live-data check needed before any config-sync work.)
4. Does the product intend "one BCC template usable across N projects" to mean payrate hourType vocab must always be kept in lockstep by hand, or should there be a per-project label-alias/mapping layer so the same file works without re-typing payrate leaf names per project? Out of scope for a bug fix — needs a user decision.
