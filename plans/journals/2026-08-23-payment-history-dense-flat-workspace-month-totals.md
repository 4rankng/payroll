---
title: "Payment history: dense flat workspace + month totals"
date: 2026-08-23
summary: "Bank-transfer history page reworked into one flat workspace module — nested per-record cards removed, summary endpoint added (8501d3ce, d78e8c12)."
---

# Payment history: dense flat workspace + month totals

## What happened

- Backend: `GET bank transfer histories` now returns a `summary` (total_amount, transfer_count, employee_count) aggregated across all filtered results, not just the current page (`8501d3ce`). Powers the page's stat strip (Tổng đã chuyển / Bút toán / Nhân viên).
- Frontend: `/admin/payment-history` + partner variant reworked into a data-dense workspace — single-open accordion, 48px desktop rows, sticky column header, pageSize 50, month picker popover. Final pass removed the nested card design: record `<details>` rows are flat hairline-divided rows of the single workspace Card at every breakpoint; open accent is the inset emerald spine only (`d78e8c12`).
- Cleanup: dead `.admin-payment-history-{decoration,filters,record}` selectors in `src/styles/admin-daisy.css` were still card-styling records (radius + card bg + shadow) in the admin scope — deleted.
- Verification: 14/14 vitest, eslint + `tsc --noEmit` clean, targeted backend `go test` payroll package green; code-reviewer APPROVED, all 5 findings fixed (strict per-string `not.toContain` assertions, skeleton flat-class test, seam comment accuracy, dead CSS).

## Decision

- Flat rows replace per-record cards on mobile/narrow too — supersedes the "mobile keeps card layout" constraint recorded in `plans/260823-1525-bank-transfer-density/`. The workspace Card stays the single module surface; expanded panels stay flat tinted surfaces (`bg-slate-50/80`), not cards.
- jest-dom multi-class `not.toHaveClass` only fails when ALL classes are present — use per-string `not.toContain` when guarding against any single card-chrome class returning.

## Next steps

- Both commits are on local `main` only — not pushed, not deployed. Deploy from repo root with `make deploy` when ready (no migrations involved).
- understand-anything knowledge graph update was deferred until after these commits; it can run its structural check against `d78e8c12` next session.

> Historical work record — not durable authority. Prefer docs/specs/ADRs for current decisions.
