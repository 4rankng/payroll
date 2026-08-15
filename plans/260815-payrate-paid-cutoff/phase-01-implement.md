---
phase: 1
title: "Implement"
status: in-progress
effort: "focused"
---

# Phase 1: Implement

## Overview

Implement the paid cutoff and atomic recalculation, then align the existing desktop and mobile editor guidance for Admin and Partner.

## Implementation Steps

1. Read the latest paid date at project scope and use it for dry-run validation and creation-date validation.
2. In the effective-dated payrate creation transaction, create the configuration, then recalculate rows dated on or after its start date only when `payment_status != paid` and `timesheet_status != approved`.
3. Preserve paid and approved rows even if they change concurrently by retaining the same predicates on each update.
4. Update the existing Admin/Partner desktop and mobile guidance to describe the paid cutoff and automatic update of mutable records.
5. Add a focused SQLite regression test for cutoff selection and recalculation.

## Success Criteria

- [ ] The earliest date is the day after the latest paid timesheet in the project.
- [ ] No paid or approved timesheet is modified.
- [ ] Later unpaid and unapproved timesheets receive the new payrate, rate, and amount.
- [ ] Desktop and mobile editors use consistent Vietnamese guidance.
