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

## Status log

- 2026-10-04: Plan approved. W0 in progress — kanban cards + visual-baseline harness delegated; inventory written (this file).
- 2026-10-04: Bundle baseline (exit 0): main chunk 1,463.95 kB (gzip 412.52 kB); pdfmake chunk 2,214.79 kB (gzip 1,050.77 kB); 154 precache entries; build 1m35s.
- 2026-10-04: W1 token gap verified against built CSS ground truth — see `token-gap-w1.md` (error/warning/success/secondary solids, tertiary, fg-*, full utility ladders, featured-icon fg tokens, destructive icon tokens, display-* sizes all missing; only bg-brand-solid + transition-inherit-all resolve today).
- 2026-10-04: Untitled UI MCP vendoring flow validated (v7, CLI command + auto-placement under `src/components/base/<family>/`).
- 2026-10-04: Fix commit `d71d52b2` — repaired 2 pre-existing type errors in UU groundwork (banner `icon = <Flag05 />`; cx.ts `display-*` moved from `theme.text` to `classGroups["font-size"]`). All four gates green at baseline.
- 2026-10-04: FOUND BUG (affects every e2e wave gate): the backend CORS whitelist allows only `http://localhost:3000` (vite.config dev port); preflight from `127.0.0.1:5173`/`localhost:5173` → 403, so authenticated browser flows from the main Playwright config's origin fail with "Thông tin đăng nhập không hợp lệ". QA creds verified valid via direct API login. Mitigation for this migration: `frontend/playwright.visual.config.ts` serves the visual suite from `localhost:3000`. Main `playwright.config.ts` likely broken for all login-dependent specs on this machine — aligning it to :3000 is a proposed follow-up awaiting owner word.
- 2026-10-04: 15 kanban cards regenerated via the teammate's `/tmp/gen_cards.py` after a rename accident destroyed them (zsh `BASH_REMATCH` is empty — the loop overwrote 14 cards into one file); filename convention now matches the repo (`20261004_<n>-<slug>.docx`). No net loss.
