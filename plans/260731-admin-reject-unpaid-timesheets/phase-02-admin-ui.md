---
phase: 2
title: admin-ui
status: completed
effort: medium
---

# Phase 2: admin-ui

## Overview

Add one shared responsive form dialog and wire it into separate Admin desktop/mobile action surfaces.

## Implementation Steps

1. Add API types, service method, TanStack mutation, and complete timesheet/dashboard cache invalidation.
2. Build the dialog from existing project selector, inclusive date-range picker, textarea, and destructive confirmation primitives.
3. Show an exact server-backed affected count before final confirmation where practical; always re-evaluate scope during mutation.
4. Add the action to the desktop Admin overflow menu and mobile Admin action sheet with minimum 44px touch targets.
5. Keep Partner surfaces untouched and add action-absence tests.

## Success Criteria

- [x] Required fields and reversed dates are blocked with Vietnamese messages.
- [x] Desktop and mobile submit identical scoped payloads and show actual result count.
- [x] Failed submissions retain form values for retry.
- [x] Partner never sees the action.
