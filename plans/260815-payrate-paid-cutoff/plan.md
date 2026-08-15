---
title: "Payrate cutoff and mutable timesheet recalculation"
description: "Allow a new payrate after the latest paid timesheet and recalculate only later mutable timesheets."
status: in-progress
priority: P1
branch: "main"
tags: []
blockedBy: []
blocks: []
created: "2026-08-15T06:51:22.247Z"
createdBy: "ck:plan"
source: skill
---

# Payrate cutoff and mutable timesheet recalculation

## Overview

Replace the current all-timesheet cutoff with a project-wide paid-timesheet cutoff. A configuration that starts on or after the day following the latest paid timesheet is valid; after it is saved, only timesheets dated on or after that start date that are both unpaid and unapproved are recalculated.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Implement](./phase-01-implement.md) | Pending |
| 2 | [Verify](./phase-02-verify.md) | Pending |

## Dependencies

No cross-plan dependencies. Paid and approved timesheets remain immutable; the existing public API response shape remains unchanged.
