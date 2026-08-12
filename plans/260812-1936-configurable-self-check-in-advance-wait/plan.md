---
title: "Configurable self check-in advance wait"
description: "Make the post-checkout self-check-in advance wait configurable by Admin while retaining 24 hours as the safe default."
status: completed
priority: P1
effort: "small"
tags: [backend, frontend, settings, advance-payment, attendance]
created: 2026-08-12
---

# Configurable self check-in advance wait

## Overview

Replace the hard-coded 24-hour self-check-in earning hold with an Admin-managed,
whole-hour setting. Persist the default for both existing and new installations,
validate updates at the API boundary, and persist the configured deadline at
checkout for the scheduler and its overdue-credit recovery path.

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | Let Admin set the self-check-in advance wait in desktop and mobile Settings. | P1 |
| 2 | Apply the setting to future post-checkout scheduling while preserving each checkout's deadline for recovery. | P1 |
| 3 | Preserve a safe 24-hour fallback for missing or invalid persisted data. | P1 |

## Phases

| # | Phase | Status |
|---|-------|--------|
| 1 | [Phase 1: Persist, apply, and verify the wait](./phase-01-start.md) | Complete |

## Success Criteria

- [x] Existing databases receive `self_check_in_advance_hold_hours=24`; clean installs seed it too.
- [x] Admin can save an integer from 0 through 720 hours on desktop and mobile Settings.
- [x] Invalid API writes are rejected and missing/invalid reads use 24 hours.
- [x] Checkout persists the configured hold as an immutable deadline that both worker and recovery enforce.
- [x] Backend and frontend focused tests, type check, lint, and the project API test gate are run or its local-service limitation is reported.

<!-- slug: configurable-self-check-in-advance-wait -->
