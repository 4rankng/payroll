# Payment history: remove nested card design (flatten record rows)

Status: DONE (2026-08-23, ak-cook --auto)

Completion: all acceptance criteria met; reviewer APPROVED (3 minor + 2 nit
findings all fixed: per-string not.toContain assertions, skeleton flat-class
test added, seam comment corrected, dead admin-daisy.css card selectors
removed). Verification: 14/14 vitest, eslint + tsc --noEmit clean.
Scope: `frontend/src/components/payroll/BankTransferHistoryPageContent` UI only.
Supersedes the "Mobile unchanged: card layout" constraint of
`plans/260823-1525-bank-transfer-density/` — user now explicitly asks for the
nested card look to go.

## Outcome

On `/admin/payment-history` (and the partner variant sharing the component),
records render as flat hairline-divided rows of the single workspace module at
every breakpoint. No card-in-card.

## Constraints

- TingTing emerald/slate prod theme only; Vietnamese text.
- Keep single-open accordion, mobile auto-scroll, 44px touch targets, 20px
  mobile money value, sticky desktop header.
- Expanded panel stays a flat tinted surface (`bg-slate-50/80`) — already
  de-carded by the prior pass.
- No backend/API/schema changes; prior session's uncommitted backend work
  untouched.

## Acceptance criteria

1. Record `<details>` rows have no `rounded-xl`/`border`/drop-shadow card
   chrome at any breakpoint (mobile matches the existing xl: flat treatment).
2. Records container: unconditional `divide-y divide-slate-200/80`, no
   `space-y`/padding wrapper.
3. Open-state accent = inset emerald spine only
   (`open:shadow-[inset_4px_0_0_0_#059669]`); no `open:border-*`.
4. Workspace `Card` remains the single module surface (rounded mobile, squared
   xl) — unchanged.
5. Skeleton rows follow the flat rhythm (divide, no rounding/gaps).
6. Behavior tests stay green: single-open, collapse-on-page-change, filters,
   month picker; vitest + lint + type-check pass.
7. tester + code-reviewer subagent passes.

## Files

- `frontend/src/components/payroll/BankTransferHistoryPageContent.tsx` (edit)
- `frontend/src/components/payroll/BankTransferHistoryPageContent.test.tsx` (edit)

## Verification

- `pnpm exec vitest run src/components/payroll/BankTransferHistoryPageContent.test.tsx`
- `pnpm lint && pnpm type-check`
- tester + code-reviewer subagents

## Risks / rollback

- Mobile rows lose per-card separation; hairline dividers + open spine carry
  the structure. Single-file revert if the card look is wanted back.
