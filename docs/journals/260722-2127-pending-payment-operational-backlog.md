# Pending approvals disappeared from the payment backlog

**Date**: 2026-07-22 21:27
**Severity**: High
**Component**: Timesheet summary metrics
**Status**: Resolved

## What Happened

The operational dashboard showed 123 timesheets waiting for approval but zero
employees and `0 ₫` waiting for payment. The frontend was rendering the backend
summary faithfully; the missing amount originated in the summary query.

## Root Cause

Commit `99dcd04` made the summary reuse the payroll-ready cohort: approved
timesheets whose payment status is pending or failed. That predicate is correct
for payment filtering and exports, but it excluded every pending-approval row
from the broader operational backlog.

## Decision

The two concepts now have separate named cohorts:

- Operational backlog: pending-approval or approved, with pending/failed payment.
- Payroll-ready: approved only, with pending/failed payment.

The summary uses the operational cohort. List filtering and bulk-transfer export
remain payroll-ready, so no unapproved timesheet becomes payable.

## Verification

The repository regression contains 123 pending-approval rows. It returns
`20,000` with the approved-only predicate and `1,250,000` with the operational
cohort, while paid and cancelled rows remain excluded. Race tests and vet passed
for the affected domain, repository, timesheet-service, and bulk-transfer
packages. The live API suite could not run because no backend was listening on
port 8080; the full repository suite also retains unrelated existing failures.
