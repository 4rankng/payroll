# Bank-transfer history: density pass + single-open expanded state

Status: DONE (2026-08-23) — criteria implemented and green (14/14 component
tests). "Mobile unchanged: card layout" constraint superseded same day by
plans/260823-1542-payment-history-flat-records/ (user asked to remove the
nested card design).
Scope: `frontend/src/components/payroll/BankTransferHistoryContent` UI only. No backend/API/schema changes.

## Outcome

Turn the expand-on-click bank-transfer history list into a data-dense desktop table
with an unmistakable single-open expanded record, per the accepted design
instruction set (A.1–A.7, B.1–B.6).

## Constraints

- Mobile unchanged: 44px touch targets, card layout, 20px primary money value.
- TingTing emerald/slate prod theme only (no gold, no new palettes).
- Desktop = xl: breakpoint (1280px).

## Acceptance criteria

1. A1 Row height: summary `xl:min-h-[48px] xl:py-1.5` (was 64px/py-2).
2. A2 Avatar: desktop badge shrinks to 20px (`xl:h-5 xl:w-5`, 12px icon).
3. A3 Identity inline on desktop: name · CCCD · projects on one line (xl:flex,
   `·` separators), stacked layout kept on mobile.
4. A4 Numerals: `Thực nhận` desktop 15px→13px (mobile 20px kept); payment-date
   column right-aligned on xl to match the numeric edge.
5. A5 Sticky header: header row `xl:sticky xl:top-0 xl:z-10 xl:bg-slate-50/95
   xl:backdrop-blur`; Card gets `xl:overflow-visible xl:rounded-none`
   (overflow-hidden ancestor would disable sticky; squaring removes corner
   artifacts). Desktop scrollport = `main.admin-main` (overflow-auto).
6. A6 Filter bar: wrapper `xl:py-1.5` (controls already `min-h-11 sm:min-h-9`).
7. A7 Page size default 20 → 50 (existing pagination option).
8. B1 Merged open card: panel divider `group-open/record:border-t-0` so the open
   summary + panel read as one continuous card.
9. B2 Spine: delivered by the existing details-level
   `open:shadow-[inset_4px_0_0_0_#059669]` which already runs summary→panel
   once the B1 seam is removed; an extra 2px tick would duplicate it.
10. B3 Identity chip: `↳ {employee_name}` 10px emerald-700, top-left of panel,
    desktop only.
11. B4 Inner header tint: TransferReferences header row `bg-emerald-100/60`
    (outer stays slate).
12. B5 Single-open accordion: controlled `openKey` state at page level, reset on
    filter/page/pageSize change; summary click preventDefault + state toggle.
13. B6 Mobile auto-scroll: on open below xl, `scrollIntoView({block:'start'})`
    guarded by matchMedia + optional call (jsdom-safe), `scroll-mt-3` offset.
14. Tests: update class assertions, add single-open + pageSize-50 + chip tests;
    vitest + lint + type-check green; independent tester + code-reviewer pass.

## Files

- `frontend/src/components/payroll/BankTransferHistoryPageContent.tsx` (edit)
- `frontend/src/components/payroll/BankTransferHistoryPageContent.test.tsx` (edit)

## Verification

- `pnpm exec vitest run src/components/payroll/BankTransferHistoryPageContent.test.tsx`
- `pnpm lint && pnpm type-check`
- tester + code-reviewer subagents (ak-cook mandatory delegation)

## Risks / rollback

- Sticky header depends on no overflow-hidden ancestor between header and
  `main.admin-main`; if a partner-layout ancestor clips, sticky silently no-ops
  (harmless degradation).
- Single-file revert restores prior state; no data-layer touchpoints.
