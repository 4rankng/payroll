---
title: "Phase 1: Persist, apply, and verify the wait"
status: completed
---

# Phase 1: Start

## Overview

Ship the Admin-managed self-check-in advance wait safely across the persistence,
attendance, API, and responsive Settings boundaries.

## Requirements

- [x] Add a number setting with a migration and a seed default of 24 hours.
- [x] Accept only canonical whole hours from 0 through 720; use 24 hours for missing or invalid reads.
- [x] Persist the authoritative hold calculation at checkout; make the worker and overdue-credit sweep enforce that deadline.
- [x] Add the reusable Settings form state and the matching desktop/mobile cards.
- [x] Cover backend parsing/scheduling and frontend form/page parity.

## Implementation Steps

1. Add the setting key, parsing/validation rules, migration, and fresh-install seed.
2. Replace the hard-coded hold with the runtime setting at checkout and persist its deadline for scheduled and recovery credit.
3. Extend the shared Admin Settings form, then render the same card in desktop and mobile Settings.
4. Add focused backend, integration, and frontend regression coverage.
5. Run targeted checks, broad relevant gates, an implementation review, and production release checks.

## Todo

- [x] Persistence and runtime accessor
- [x] Attendance scheduling and recovery
- [x] Responsive Admin Settings UI
- [x] Tests and release validation

## Success Criteria

The configured delay is durable, validated, respected by every credit path, and
visible/editable with the same behavior on desktop and mobile Admin Settings.
