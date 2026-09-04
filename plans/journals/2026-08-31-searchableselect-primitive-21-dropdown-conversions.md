---
title: SearchableSelect primitive + 21 dropdown conversions
date: 2026-08-31
---

# SearchableSelect primitive + 21 dropdown conversions

2026-08-31 night, payroll. /ak:cook --auto 'all dropdown should have search function'. Scouted: repo had Popover+cmdk+vietnameseIncludes patterns (Multi/Async searchable dropdowns) but no single-select sync primitive; 33 files on plain Select. Scoped to dynamic/long-list dropdowns (21 instances, 17 surfaces); static enums kept Select. Built ui/searchable-select.tsx (shouldFilter=false + vietnameseIncludes; triggerId/triggerAriaLabel/contentAlign props emerged from review). 3 parallel devs converted disjoint clusters preserving parseInt/Number boundaries and 'all' pseudo-options. Adversarial review BLOCK->fixed (missing triggerId, swift_code dupes, prettier) then SHIP. Lessons: (1) Radix popover opens on pointerdown - fireEvent needs {button:0} in jsdom; (2) old test matched a mock artifact (mocked SelectItem rendered children as button) - rewrote to exercise real combobox; (3) cmdk renders option label+searchText as concatenated text, use regex finders; (4) tailwind-merge lets triggerClassName compact classes override primitive h-11. Gates: vitest 478/478, tsc/eslint clean, browser QA on history-sheet + page filters, deployed 81874cfb/9646a2b4 verified healthy.

> Historical work record — not durable authority. Prefer docs/specs/ADRs for current decisions.
