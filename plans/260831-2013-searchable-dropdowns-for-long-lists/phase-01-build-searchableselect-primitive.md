---
phase: 1
title: "Build SearchableSelect primitive"
status: todo
priority: P1
effort: "2h"
dependencies: []
---

# Phase 1: Build SearchableSelect primitive

## Overview

One reusable single-select searchable dropdown, closing the gap between
`MultiSearchableDropdown` (multi) and `AsyncSearchableDropdown` (async).

## Requirements

- Popover + Command (cmdk), `shouldFilter={false}` + self-managed query so
  `vietnameseIncludes` drives filtering (diacritic-insensitive both directions).
- Props: `value: string | undefined`, `onChange`, `options: {value, label,
  disabled?, searchText?}[]`, `placeholder`, `searchPlaceholder`,
  `emptyMessage`, `disabled`, `triggerClassName`, `contentClassName`,
  `modal?` (default true).
- Trigger: `Button variant="outline" role="combobox"` + ChevronDown/
  ChevronsUpDown, h-11, truncated label of selected option; shows placeholder
  when empty; label lookup by value each render.
- List: max-h with CommandList, Check icon on selected, CommandEmpty message.
- Vietnamese labels; matches MultiSearchableDropdown styling (outline button,
  `justify-between`, w-full).

## Implementation Steps

1. `frontend/src/components/ui/searchable-select.tsx` following
   `multi-searchable-dropdown.tsx` lines 1–80 for structure/styling.
2. `frontend/src/components/ui/searchable-select.test.tsx` — vitest + RTL:
   renders options; filters "phan phong" → "Phòng ban" (diacritic-insensitive);
   filters via searchText; selects → onChange + closes; disabled option not
   selectable; empty message on no match; keeps selected label on trigger.

## Success Criteria

- [x] Primitive + tests written; `pnpm lint` green; `pnpm test:run --
      searchable-select` green.
