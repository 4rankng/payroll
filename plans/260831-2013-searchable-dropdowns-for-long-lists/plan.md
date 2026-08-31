---
title: "Searchable dropdowns for long lists"
description: "Type-to-search for every dropdown whose option list can grow long, via one reusable SearchableSelect primitive"
status: pending
priority: P2
effort: "1d"
tags: [frontend, ui, dropdown]
created: 2026-08-31
---

# Searchable searchability: scope note

The request "all dropdown should have search function" is scoped by evidence:
search belongs where option lists are **dynamic or long** (projects, employees,
users, lenders, banks, accounts, transaction types). Static enums (priority,
status, role, hour-type — 3–6 fixed options) keep plain `Select` — a search box
there is noise. This boundary is a stated product call, not a shortcut: the
full instance list lives in phase-02 with per-file classification, and any
instance the user wants added beyond it is a one-line swap once the primitive
exists.

# Searchable dropdowns for long lists

## Overview

Dropdowns with long option lists (37 projects, hundreds of employees/accounts)
force scrolling. The repo already has the building blocks — Popover + Command
(`cmdk`) + `vietnameseIncludes` normalization, used by `MultiSearchableDropdown`
(multi-select) and `AsyncSearchableDropdown` (async) — but no single-select
sync primitive, so 33 files still use plain `Select` for long lists.

Two phases:

1. Build **`SearchableSelect`** — single-select, sync options, client-side
   filter with `vietnameseIncludes` (diacritic-insensitive: "phan phong" finds
   "Phòng ban"), following the existing MultiSearchableDropdown styling.
2. Convert every long-list dropdown to it (instance table in phase-02).

User decisions this session (2026-08-31): `--auto`; scope = long/dynamic-list
dropdowns; static enums keep `Select`.

## Constraints

- No backend changes; no new dependencies (`cmdk`, `vietnameseIncludes` exist).
- Match existing primitives' look: `Button variant="outline" role="combobox"`,
  h-11 touch targets, Vietnamese labels, `Pop modal` where in-sheet.
- Type-safe: value is `string` (ids stringified at call sites).
- Static enums stay `Select` (stated boundary).
- The UploadHistorySheet project dropdown ships in this sweep — its 37-option
  Select is what prompted the request.

## Non-goals

- No redesign of `Select` itself; plain Select remains for enums.
- No async/server-side search here (AsyncSearchableDropdown covers that).
- No keyboard-shortcut or multi-select changes to existing primitives.

## Phases

| # | Phase | Status |
|---|---|--------|
| 1 | [Build SearchableSelect primitive](./phase-01-build-searchableselect-primitive.md) | Todo |
| 2 | [Convert long-list dropdowns](./phase-02-convert-long-list-dropdowns.md) | Todo |

## Success Criteria

- [ ] `SearchableSelect` renders, filters diacritic-insensitively, selects,
      shows empty state; unit tests pass.
- [ ] Every long-list dropdown in the phase-02 table uses it; short enums
      untouched.
- [ ] `pnpm lint` (eslint + tsc) green; no regression in the converted
      surfaces (browser-verified: filter, select, empty state, sheet-context
      modal behavior).
- [ ] Commit(s) follow `ui(...)`/`feat(...)` conventional format.

## Key evidence (scout 2026-08-31)

- Repo primitives: `ui/command.tsx` (cmdk), `ui/popover.tsx`,
  `ui/multi-searchable-dropdown.tsx` (pattern source),
  `ui/async-searchable-dropdown.tsx`, `utils/vietnameseNormalization.ts`
  (`vietnameseIncludes`).
- 33 files import plain `Select`; dynamic-list instances identified by
  `.map(` over projects/users/employees/lenders/banks/accounts/txn-types.
- `BCCUploadModal`'s custom `ProjectCombobox` already searchable (excluded).
- Vitest + RTL available (`employee-multi-selector.test.tsx` exists).

## Risks

- **Value-type drift** (number vs string ids) — primitive is string-typed;
  call sites stringify; code-reviewer watches for `Number()`/`parseInt`
  regressions at consumers.
- **Radix popover inside Sheet/Dialog** — use `modal` Popover where the
  instance sits in a Sheet/Dialog context (established pattern). Devs verify
  the 3 sheet-context instances in browser.
- **cmdk `value` normalization** — cmdk lowercases/normalizes by default; pass
  `vietnameseIncludes`-based filtering via `shouldFilter={false}` +
  self-managed query state (same as AsyncSearchableDropdown) so diacritics
  filter correctly.
