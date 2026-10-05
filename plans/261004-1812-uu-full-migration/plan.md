# UU PRO Full Migration — Plan of Record

Approved 2026-10-04. Full plan: `~/.claude/plans/migrate-our-frontend-to-adaptive-toast.md` (summary mirrored here; that file remains authoritative for rationale).

## Goal

Migrate the entire payroll frontend to Untitled UI PRO with superb, elegant UI/UX — theme may differ somewhat while keeping the overall feeling; layout/styling improvements during migration are acceptable. Binding: green brand identity, compact data-dense practicality, role structure, Vietnamese copy, WCAG 4.5:1, parity 1280/390/320, 44px targets, one-row controls, ≤2 dashboard cards. Icons migrate lucide → `@untitledui/icons` per-surface, glyphs chosen freely.

## Architecture

1. **Token bridge** — `tailwind.config.ts` + `src/styles/variables.css`: full UU v7 token set (TW3 syntax), UU values as source of truth; all role overrides at `:root` paired with `html.{admin,partner,employee}-route-active` selectors, **never `[data-*-ui]`-alone** (Radix portal transparency bug).
2. **Vendored UU (v7, via untitledui MCP)** — `src/components/base/<family>/`; TW4→TW3 ports (`data-[x]:`, `[&>[attr]]:`, `!`-prefix); Vietnamese labels threaded at vendoring time.
3. **Compat layer** — `src/components/ui/*` internals rewritten to UU while preserving props/classNames/exports 100%; `forwardRef` kept; Button `asChild` via `@radix-ui/react-slot`; app-specific variants (button success/info/warning/monochrome*, badge role variants) stay as cva entries.

**Engine-keep (restyle-in-place, Radix stays):** dialog/sheet, select, dropdown-menu, calendar/date-range-picker/month-picker, accordion (no UU equivalent), sonner, command/cmdk, sidebar. To be recorded in an ADR at W-final.

## Inventory (2026-10-04, commit 29cde20e)

- Vendored: `components/base/buttons/`, `components/foundations/featured-icon/`, `components/marketing/banners/banner-dual-action-brand-full-width.tsx`.
- `components/ui/`: 104 files; 513 files import `@/components/ui`; 249 use daisyUI `ct-`; 476 import lucide-react.
- Importers: button 255 · input 105 · dialog 78 · card 58 · select 51 · sonner 47 · dropdown-menu 18 · table 17 · tabs 13.
- `<Button asChild>` call sites: 0. RHF `register()`: 5 files.
- Known bugs to fix in W1: dead `in-data-input-wrapper:*` classes (vendored button); missing `utility-{gray,error,warning,success}-*` ladders (FeaturedIcon); missing `display-*` fontSize entries (cx.ts registers them in tailwind-merge).

## Waves → kanban cards (kanban/TODO/, created in W0)

| Wave | Scope | Card |
|---|---|---|
| W0 | Baselines + inventory + tickets | (this dir + reports) |
| W0b | Delete 9 dead ui files | folded into card 1 |
| W1 | Token bridge | 20261004_1-cau-hinh-token-uu-pro |
| W2 | Leaf statics (badge, card, separator, avatar, skeleton, label) | 20261004_2-nang-cap-phan-tinh-badge-card |
| W3 | Inputs (input, textarea) | 20261004_3-nang-cap-o-nhap-lieu |
| W4 | Button (255 importers) | 20261004_4-nang-cap-button |
| W5 | Overlays restyle (dialog, alert-dialog, sheet, popover, tooltip) | 20261004_5-nang-cap-overlay-dialog |
| W6 | Menus + select restyle | 20261004_6-nang-cap-menu-select |
| W7 | Form controls | 20261004_7-nang-cap-form-controls |
| W8 | Data display | 20261004_8-nang-cap-bang-du-lieu |
| W9 | Toast (sonner) | 20261004_9-nang-cap-toast |
| W10 | Auth surfaces | 20261004_10-chuyen-doi-trang-auth |
| W11 | Employee portal | 20261004_11-chuyen-doi-cong-nhan-vien |
| W12 | Partner (desktop+mobile) | 20261004_12-chuyen-doi-cong-doi-tac |
| W13 | Admin (desktop 18 + mobile 17, 2–3 sub-waves) | 20261004_13-chuyen-doi-quan-tri |
| W14 | Adv-partner + accountant | 20261004_14-chuyen-doi-adv-partner-ke-toan |
| W-final | Retirement + docs/ADR | 20261004_15-loai-bo-di-san-va-hoan-thien |

## Per-wave gate

`pnpm lint` → `pnpm build` → `pnpm test:run` → `pnpm test:e2e` → visual diff vs baseline (only intended deltas, blessed) → owner manual QA → card moves TODO → IN_PROGRESS → DEV_COMPLETED → QA_TESTED (owner-verified). One wave = focused commit(s) on `main`; rollback = single `git revert`. Stage by explicit path (parallel sessions/teammates). No push/deploy without the owner's word.

## Rulings (binding for all waves)

- **tailwind-merge stays v2.** Upstream UU v7 pairs with twMerge v3; the CLI auto-bumps it (major, touches every `cx()` call). All vendored files are cx-ported to the repo's v2 `cx`. Vendored dirs are style references, never runtime deps of the compat layer.
- **`untitledui add` CLI:** only with owner approval per component family; run `git status` immediately after and revert any out-of-scope change (it overwrites `cx.ts`, `is-react-component.ts`, and existing `base/**` files). Use `pnpm dlx` — plain `npx` fails with EOVERRIDE on this machine.
- **Compat layer stays native-first where contracts demand it:** inputs/textarea keep native elements + forwardRef (RHF `register()`); UU error state via `aria-[invalid=true]:` (TW3 has no `aria-invalid:` variant); role-scoped surfaces use `bg-card`/`border-input`, not the vendored `bg-primary` (emerald collision here).
- **Dead primitives:** 7 deleted in W1 (context-menu, menubar, navigation-menu, hover-card, carousel, resizable, aspect-ratio). `drawer.tsx` (1 importer) and `scroll-area.tsx` (5) are alive — the design-agent report overstated; they restyle in their own waves.

- 2026-10-04: **W3 delivered + reviewed (approved, uncommitted):** ui/input.tsx + ui/textarea.tsx restyled to UU tokens, contracts 100% (native elements, forwardRef for RHF, exact variant unions, `lang="vi-VN"`, h-11/sm:h-9); error state via `aria-[invalid=true]:border-utility-error-300` + outline-error; 11/11 contract tests; tsc green. Vendoring incident → rulings above.
- 2026-10-04: **W4 delivered + reviewed (approved, uncommitted):** ui/button.tsx restated in UU vocabulary inline (NOT importing the vendored module — preserves native type/form/ref semantics for 255 call sites and keeps react-aria-components out of the 1.46MB entry chunk; sync-note header added). All 11 variants × 5 sizes preserved; app-specific variants (success/info/warning/monochrome*) kept; disabled moved opacity→UU tokens; ghost hover → bg-secondary_hover; solids verbatim (brand/error/warning/success-solid). bg-primary collision (shadcn emerald vs UU gray surface) handled via unambiguous families. 8/8 new tests, 44/44 existing button-pinned tests, tsc green. `info` variant keeps bg-sky-600 interim (bridge has no UU blue — W7+ decision).
- 2026-10-04: W2 mid-flight healthy (6 files churning; vendored avatar/badges import only repo-safe modules; dot-icon.tsx is a real dependency of vendored badges, not a stray).

- 2026-10-04: **W2 reviewed mid-flight (approved pending final report):** badge → UU soft-tint chips on the W1 utility ladders (bridge gaps documented: info/admin keep legacy palette); avatar → utility-gray surface + UU inner contrast outline, compound shape unchanged; card keeps variant prop + compact padding. All contracts intact.
- 2026-10-04: **Revised commit/freeze sequence (from visual-baseline forensics):** the visual harness is fixed (skeleton waits, notification pinning, volatile masks, login retry; regeneration 24/24) but the :3000 dev server hot-reloads our own uncommitted edits mid-run — commits land FIRST (W1→W2→W3→W4 back-to-back), then one frozen-tree baseline regeneration (which becomes the working reference for W5+; the d9ba00a9 baselines stay in git as the true pre-migration record), then compare-twice for 24/24. Rule: never run the visual suite while any session edits frontend/src.

- 2026-10-04: **W2 delivered (accepted):** badge/avatar vendored (v7, TW3-ported); card/separator/skeleton/label restyled on the bridge. tsc green project-wide, 19/19 new contract tests, Tailwind-CLI emission check (no silent fallbacks). Rulings: no utility-blue ladder for now (info/admin keep legacy palette); PRE-EXISTING WCAG gaps escalated to owner (info 4.1:1, manager 3.8:1 vs the 4.5:1 gate — candidate fix with the future blue-ladder addition).
- 2026-10-04: All four primitive waves delivered → combined gate → commits W1→W2→W3→W4 back-to-back (explicit paths; teammate reports ride their wave commits) → freeze tree → visual regeneration + compare-twice → harness commit.

- 2026-10-04: **Combined gate green → wave commits landed back-to-back:** `036c2e39` (W1 token bridge + 7 dead primitives), `53495a4a` (W2 leaf statics), `d0f4ab08` (W3 inputs), `42e045a9` (W4 button). Gates: lint ✓, vitest 158 files / 751 tests ✓ (38 new contract tests), build ✓. Tree frozen for the visual certification window (regenerate + compare-twice); harness + snapshots commit after 24/24-twice. Kanban cards 1–4 → DEV_COMPLETED (owner QA pending → QA_TESTED).

- 2026-10-04: **Visual harness certified against rendering nondeterminism** (PerformanceObserver quiet-window: 2.5s zero network before capture; skeleton class eliminated, 24/24 regeneration, 17/24 byte-stable over 16 min). Residual 7 flips = live QA-DB drift (payment aggregates mutated between runs). **Ruling: pin the ~8–10 volatile endpoints with CAPTURED real-response fixtures** (the approved plan's "deterministic data" clause; pins apply to visual-baseline only; documented in-spec). Then regen + compare-twice → commit spec + snapshots + fixtures.
- 2026-10-04: Local browser verification pass (owner-authorized): users/settings pages render the UU restyle cleanly (buttons, soft chips, inputs, density preserved). Dashboard desktop+mobile hangs on the KNOWN backend perf bug (dashboard/summary + bank-usage/projects stall; 44 other calls 200) — pre-existing, on record as a follow-up; harness will stub these two.

- 2026-10-04: **Waves W5–W7 delivered, reviewed, committed:** `50efc462` overlays (dialog/alert-dialog/sheet/popover/tooltip — header strip → brand-section_subtle, engine Radix, portal rule preserved; tooltip rounded-lg deviation accepted as UU-authentic), `db7b834d` menus+select (trigger = W3 input vocabulary; `--radix-select-*`/popper/animation hooks byte-identical; FilterPill noted as raw-Radix composer), `457661f3` form controls (checkbox/radio real border over ring to preserve call-site border tints — BCC amber; tabs pill idiom recolored smallest-delta; switch/slider/progress/accordion on brand+utility tokens). Combined gate: lint ✓, vitest **169 files / 800 tests** ✓, build ✓. Frozen window OPEN for the harness's regen + compare-twice; W8/W9/W10 spawns held until it closes.

- 2026-10-04: **W0 visual harness CERTIFIED** — frozen-tree sequence 24/24 regeneration + 24/24 compare ×2 (regen 10m→4.2m: the endpoint pins removed the slow-endpoint waits entirely). Committed as `584f5363` (spec + 24 certified snapshots + 31 captured-real fixtures + reports). From here: compare mode only after each wave; per-surface --update-snapshots only for intentional changes; normal QA activity no longer breaks the harness. W8+W9 (data display + toast) and W10 (auth surfaces — first native-UU surface wave, harness login selectors preserved as a hard constraint) spawned in parallel; kanban cards 8–10 IN_PROGRESS, 5 remain in TODO.

- 2026-10-05: **W10 committed** `e24c5fef` (5 auth screens native-UU; e2e auth.spec renewed 18/18 ×2 under the visual config — main-config origin remains CORS-blocked, owner decision pending). Login testids/placeholders preserved for the harness.
- 2026-10-05: **W12 + W14 delivered/accepted (uncommitted, ride the combined gate):** partner 9 surfaces + shell (ct- retirement, de-CAPS eyebrows, --partner-* untouched) and adv-partner/accountant (aria-invalid error channel, 3 new accountant contract tests). Findings: 3 legacy template-era e2e specs (projects/employees/timesheet) are red-by-construction → owner DELETE-at-W-final recommendation; mobile-partner claim-flow parity gap = pre-existing product decision (out of scope); type-locked lucide seams found.
- 2026-10-05: **W13a delivered/accepted (uncommitted):** admin shell surgical restyle (sidebar e2e PO contracts intact), mobile dashboard month bar de-daisyUI'd, settings alerts on utility-error ladder. Follow-up APPROVED and in flight: widen shared icon-slot props (LucideIcon → ComponentType<SVGProps>) across 8 shared components + complete deferred UU icon swaps — closes the lucide seam class for W-final.

- 2026-10-05: **Endgame executed.** W13b `9b1a340e` + W13c `ee27d588` (admin table + financial/utility pages, 59 files) — combined gate 174/814 vitest, employee e2e 14/14, build ✓. Owner-decision fixes: `635eaa7a` WCAG accent fix (info/admin/manager + button info → 700-level steps, 4.5:1 cleared; utility-blue ladder added to the bridge) and `5a7e5452` (main Playwright config → localhost:3000; 3 red-by-construction legacy specs deleted). W-final `8c4bf525`: 20 dead files deleted, 6 zero-importer Radix packages pruned, daisyUI `ct-` classes at ZERO (12-file tail sweep + lead's color-utility fixes), lucide 476→366 files (remainder documented — dep stays), bundle +2.25%, ADR-012 written + indexed, frontend/CLAUDE.md style guide + project ruling updated. Final certification window open.

## Status log

- 2026-10-04: Plan approved. W0 in progress — kanban cards + visual-baseline harness delegated; inventory written (this file).
- 2026-10-04: Bundle baseline (exit 0): main chunk 1,463.95 kB (gzip 412.52 kB); pdfmake chunk 2,214.79 kB (gzip 1,050.77 kB); 154 precache entries; build 1m35s.
- 2026-10-04: W1 token gap verified against built CSS ground truth — see `token-gap-w1.md` (error/warning/success/secondary solids, tertiary, fg-*, full utility ladders, featured-icon fg tokens, destructive icon tokens, display-* sizes all missing; only bg-brand-solid + transition-inherit-all resolve today).
- 2026-10-04: Untitled UI MCP vendoring flow validated (v7, CLI command + auto-placement under `src/components/base/<family>/`).
- 2026-10-04: Fix commit `d71d52b2` — repaired 2 pre-existing type errors in UU groundwork (banner `icon = <Flag05 />`; cx.ts `display-*` moved from `theme.text` to `classGroups["font-size"]`). All four gates green at baseline.
- 2026-10-04: FOUND BUG (affects every e2e wave gate): the backend CORS whitelist allows only `http://localhost:3000` (vite.config dev port); preflight from `127.0.0.1:5173`/`localhost:5173` → 403, so authenticated browser flows from the main Playwright config's origin fail with "Thông tin đăng nhập không hợp lệ". QA creds verified valid via direct API login. Mitigation for this migration: `frontend/playwright.visual.config.ts` serves the visual suite from `localhost:3000`. Main `playwright.config.ts` likely broken for all login-dependent specs on this machine — aligning it to :3000 is a proposed follow-up awaiting owner word.
- 2026-10-04: 15 kanban cards regenerated via the teammate's `/tmp/gen_cards.py` after a rename accident destroyed them (zsh `BASH_REMATCH` is empty — the loop overwrote 14 cards into one file); filename convention now matches the repo (`20261004_<n>-<slug>.docx`). No net loss.
- 2026-10-04: **W0 committed** as `d9ba00a9` (visual-baseline harness + 24 snapshots + playwright.visual.config.ts + plan-of-record docs). Note: `kanban/` is gitignored by design — cards stay untracked. First determinism survey: 14/24 stable, 10 flaky (dashboards/timesheets: live "Cập nhật" clock, pending-approvals row, count badges) — mask-and-verify cycle delegated to the harness author.
- 2026-10-04: **W1 in progress** — token bridge applied to `tailwind.config.ts` (UU error/warning/success/secondary-solid/tertiary/secondary_alt flat families, fg extensions incl. brand/error/warning/success, full utility-{gray,brand,error,warning,success} ladders, featured-icon-light-fg-*, destructive button icon tokens, outline error, display-* compact 20–40px), dead `in-data-input-wrapper:*` classes removed from vendored button, FeaturedIcon gray surface repointed `bg-tertiary` → `bg-utility-gray-100` (adaptation comment). Static gates running; expected visual delta: previously-unstyled FeaturedIcon/banner tokens now render (intended, will re-baseline with note).

- 2026-10-05: **RELEASED TO PRODUCTION.** Pre-deploy QA battery green on localhost:3000 (lint/tsc ✓; vitest 173 files / 813 tests ✓ — first run was the known load-flake at load ~49, passed with `--testTimeout=30000`; auth e2e 18/18 ✓; employee e2e 14/14 ✓; visual compare 24/24 ✓; build ✓) + manual visual sweep clean (partner dash, admin users desktop+mobile, login post-`a470432b`; admin dashboard skeletons = the documented backend stall, not frontend). Pushed `e149a39f..a470432b` (27 commits) to origin/main; `make deploy` exit 0 (frontend+backend images pushed, containers recreated on tingting.vip). Prod verified: new bundle live (entry `index.s1DT8C_s.js` — deploy-pipeline env differs from local dist hash; vendor chunks/CSS identical to the QA'd build), sw.js 200, backend auth probe returns proper JSON 401. 15 kanban cards remain DEV_COMPLETED pending owner manual QA → QA_TESTED.
