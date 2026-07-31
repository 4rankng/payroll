---
title: "Sandbox account holder resolution"
date: "2026-07-31 10:55"
severity: "Medium"
component: "OnePay sandbox holder-name lookup"
status: "Resolved"
---

# Sandbox account holder resolution

## What Happened

We fixed the OnePay sandbox regression where the mock account-info endpoint always returned `BANK CONFIRMED NAME`. That came from `a1555fc`, and it poisoned the local mismatch path by making the provider look confidently wrong for almost every real employee.

The corrected flow now resolves the stored `bank_account_name` by `account_number + swift_code`, falls back to `fullname` for legacy rows, and only uses `MOCK_ONEPAY_HOLDER_NAME` when the operator sets it explicitly. If the database lookup fails, the sandbox returns `500`; if there is no matching row, it returns an empty holder name instead of inventing one.

## The Brutal Truth

This was a self-inflicted regression. We shipped a literal test value into the default path and then acted surprised when the local warning flow became unusable. The frustrating part is that the fix was not clever, just overdue: stop lying about the holder name and read the actual record.

## Technical Details

- Regression target: `backend/sandbox/onepay/handler.go`
- Old behavior: hard-coded `BANK CONFIRMED NAME`
- New behavior: `bank_account_name` lookup by account + swift, `fullname` fallback, explicit env override only
- Error handling: DB errors now surface as HTTP `500`; `sql.ErrNoRows` returns `""`
- Related config/docs updated in `backend/docker-compose.dev.yml` and `backend/sandbox/README.md`

## What We Tried

- Red/green regression on the sandbox holder-name path
- Sandbox, `race`, provider, and employee test runs
- Local sandbox rebuild and health check
- API suite run, which still had 8 unrelated failures that were outside this fix
- Reviewer corrections that pushed us away from the literal default and toward the DB-backed lookup

## Root Cause Analysis

The root cause was a bad default baked into the mock. `a1555fc` optimized for forcing the warning dialog and ignored the real persisted authority path, so the mock stopped reflecting the data model and started contradicting it.

## Lessons Learned

Never make a mismatch flow depend on a hard-coded fake name if the real system already stores the authoritative value. Use the database first, keep the override explicit, and make missing data empty instead of fabricated.

## Next Steps

Keep an eye on the 15-minute cached invalid verdicts. If that cache is still masking the updated state after a fix, it needs a shorter invalidation path or a clearer refresh contract, not another literal in the mock.
