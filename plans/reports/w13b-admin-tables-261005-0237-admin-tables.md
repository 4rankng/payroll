# W13b — Admin core table pages (Users / Projects / Employees / Timesheet, desktop + mobile)

Wave: W13 sub-wave 2 of the UU PRO full migration. Executed by `w13b-admin-tables`, 2026-10-05.
Branch `main`, uncommitted (rides the combined gate per wave protocol).

## Scope executed

8 owned page files; 7 modified, 1 needed no change:

| File | Changes |
|---|---|
| `pages/admin/TimesheetPage/index.tsx` | 12 lucide icons → UU (full retirement), `aria-hidden` on all decorative glyphs |
| `pages/admin/UsersPage/index.tsx` | `Loader2` → `RefreshCw05` (spinner class kept); `text-slate-800` → `text-foreground`; `border-slate-300` ×2 → `border-utility-gray-200` |
| `pages/admin/ProjectsPage/index.tsx` | `text-slate-800` → `text-foreground`; `border-slate-300` ×2 (skeleton) → `border-utility-gray-200` |
| `pages/admin/EmployeesPage/index.tsx` | `text-slate-800` → `text-foreground`; `border-slate-300` ×3 (skeleton + pagination strip) → `border-utility-gray-200` |
| `pages/mobile/admin/UsersPage/index.tsx` | Header icon `Users` → `Users01`; `SlidersHorizontal` → `FilterLines`; `X` → `XClose`; `Plus` kept (UU); dead stat fields removed (see below) |
| `pages/mobile/admin/ProjectsPage/index.tsx` | `Briefcase` → `Briefcase01`; `SlidersHorizontal` → `FilterLines`; `X` → `XClose`; Chevrons kept (UU); **daisyUI `ct-btn ct-btn-primary ct-btn-sm` + `normal-case` retired** (the only real `ct-` site in my scope); `Plus` kept |
| `pages/mobile/admin/EmployeesPage/index.tsx` | `Download` → `Download01`; `SlidersHorizontal` → `FilterLines`; `X` → `XClose`; `ArrowUp`/`ArrowDown` kept (UU names verified by probe); dead stat fields removed |
| `pages/mobile/admin/TimesheetPage/index.tsx` | No change needed: zero lucide, zero `ct-`, zero raw palette. Lives entirely on shared components (TimesheetPageHeaderMobile, PayrollControlCenter, TimesheetDisplaySection). |

Zero behavior/flow/route/hook/mutation/query-key/permission changes. All Vietnamese copy byte-preserved.

## Icon mapping (all verified — in-repo usage or two tsc probes against `@untitledui/icons` 0.0.22)

| lucide | UU | Note |
|---|---|---|
| CheckCheck | CheckDone01 | Duyệt hết |
| Undo2 | RefreshCw05 | Bỏ duyệt (free glyph per plan ruling; RefreshCw05 kept out of user-facing "retry" collision contexts) |
| ArrowRightLeft | Send01 | Chuyển lô |
| MoreVertical | DotsVertical | ⋯ menu trigger |
| FileUp (Nhập KQ) | Upload01 | |
| FileUp (Tải lên BCC) | UploadCloud01 | distinct glyph per item |
| FileDown (Tạo KQ CK) | FileCheck02 | result-file semantics |
| Banknote | Wallet01 | Chuyển OnePay |
| FileText | FileDownload01 | Xuất bảng công |
| History | ClockStopwatch | Lịch sử chuyển lô |
| FileSpreadsheet | Grid01 | Lịch sử BCC |
| Trash2 | Trash01 | Loại công |
| Loader2 | RefreshCw05 + `animate-spin` | desktop "Đang tải thêm..." |
| Users / Briefcase / Download / SlidersHorizontal / X | Users01 / Briefcase01 / Download01 / FilterLines / XClose | same partners pages already use |

Probe results (may save the next agent time): `Trash01/02, ArrowUp, ArrowDown, File02, UploadCloud01, FileCheck02, Repeat02, UserEdit, DotsVertical, DotsHorizontal, Menu01, Send01` exist; `MoreVertical/MoreHorizontal, Transfer01/02, Undo01/02, UserEdit01, UserSettings(01), Swap01/02, ArrowExchange01, Exchange01, RotateCcw, RotateLeft, GripVertical, ArrowUp01, ArrowDown01, ArrowRightLeft` do **not** exist in 0.0.22.

## Judgment calls

1. **Dead stat fields removed, not transplanted.** Mobile UsersPage + mobile EmployeesPage stats arrays carried `icon`/`color`/`bg` fields that `MobileStatStrip` never consumes (verified: its `MobileStatItem` contract is key/label/value/active/onClick). Swapping those lucide refs to UU would have added dead UU imports; removal is behavior-identical and kills the lucide seam for W-final. Live fields (`role`, `filter`) preserved.
2. **`btn-admin-primary` kept.** It is NOT a daisyUI `ct-` class — it's a live shared utility (`styles/utilities.css:227`, maps to `hsl(var(--primary))`) used by mobile admin pages outside my scope (AuditLog, Loans, PayrateEdit). Per-surface retirement of it would desynchronize sibling surfaces; left untouched.
3. **`normal-case` retired with `ct-btn`.** No global CSS uppercases buttons (checked `styles/*.css` — the uppercase rules target eyebrows/table-headers/kickers only), and ui/button has no uppercase. Verified non-load-bearing.
4. **`employees/details|list|filter|import` untouched.** Only `details/` exists; it is imported exclusively by non-owned surfaces (mobile admin DashboardPage, components/sheets, admin-dashboard) — my pages import none of it, so the "IF needed" condition never triggered.
5. **Desktop admin pages keep the W13a `AdminPageFrame` shell + W8 `ResponsiveTable` vocabulary** (engine-keep ruling; UU React Aria table is new-features-only). The W12 partner-page pattern informed token/icon choices only.

## Verification

- `pnpm exec tsc -p tsconfig.app.json --noEmit`: **exactly 1 error in the whole project, NOT in my files** — `pages/mobile/admin/PayrateEditPage/index.tsx(259,9)`: UU `Zap` passed into `MobileSubPageHeader` whose slot is still `icon: LucideIcon` (`components/shared/MobileSubPageHeader.tsx:12`). That file + that shared component are outside my ownership; this is the exact type-locked-lucide-seam class W13a's approved widening follow-up is closing mid-flight right now. My 7 files contribute zero errors.
- Scoped vitest (`--testTimeout=30000`): `workspace-parity.test.tsx` + `mobile/admin/EmployeesPage/index.test.tsx` — **2 files / 11 tests passed** (covers admin Projects desktop+mobile error/empty/pagination/deep-link parity and mobile Employees deep-link/status flows).
- Grep gates on my 8 files: 0 lucide imports, 0 `ct-` classes, 0 raw-palette classes (blue/red/emerald/amber/teal/slate/sky/indigo/gray-numeric).

## Concerns

1. **Combined gate is red on the shared tree** via the PayrateEditPage/MobileSubPageHeader seam above. Needs W13a's widening to reach `MobileSubPageHeader` (or that call site reverted to a lucide icon) before the wave commits can pass the full tsc gate. Flagging for the lead; not mine to fix.
2. `pages/partner/EmployeesPage/index.tsx` changed mid-session while I used it as the W12 reference (concurrent edits) — no impact on my files, noted for awareness.
3. Visual baselines for these surfaces will move (icon glyph changes) — expected and blessed per the wave brief; lead re-baselines after commit.
