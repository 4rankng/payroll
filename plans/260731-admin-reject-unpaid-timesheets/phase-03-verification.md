---
phase: 3
title: verification
status: completed
effort: medium
---

# Phase 3: verification

## Overview

Run focused, repository-wide, responsive, authorization, and adversarial checks.

## Implementation Steps

1. Run focused Go and Vitest suites first.
2. Run backend race/unit tests, frontend lint/type-check/build, and `make api-test` when the live integration environment is available.
3. Verify authenticated Admin at 1280px, 390px, and 320px, including long project names, keyboard/focus behavior, and success/zero/error states.
4. Verify Partner action absence and API denial.
5. Run code review, `git diff --check`, and `graphify update .`.

## Success Criteria

- [x] Focused behavior and authorization tests pass.
- [x] Frontend quality gates pass without new warnings/errors.
- [x] Responsive evidence covers all required widths and Partner absence.
- [x] Skipped integration checks are reported explicitly.
