# Configurable Chuyển lô workbook limit

**Date**: 2026-07-26 18:36 +08
**Severity**: High
**Component**: Manual MBank Chuyển lô export / Admin Settings
**Status**: Resolved

## What Happened

We replaced the hardcoded manual Chuyển lô workbook ceiling with the persisted `bulk_transfer_workbook_limit_vnd` setting and seeded the default at `400000000`. The export path now captures that threshold once, uses strict `< limit` partitioning, and refuses to write transaction codes if workbook generation fails. Both Admin Settings routes now expose the limit as a full-VND control.

The review feedback was useful and annoying in the right way: it forced the money path to stop trusting cache-aside reads for a financial control, made the seed rollback-safe, and exposed where the OnePay/export paths were too loosely coupled.

## The Brutal Truth

This was the kind of bug that looks small until you remember it is a payroll boundary. A stale cached setting or a sloppy numeric parser would have let the next export use the wrong ceiling and nobody would notice until a workbook split or overflow behaved badly. That is a stupid place to be with real money. The UI also needed a proper retry state instead of silently pretending the setting existed.

## Technical Details

- Added an authoritative DB read for `bulk_transfer_workbook_limit_vnd` so a successful Admin update is visible to the next export without waiting on cache invalidation.
- Validation now only accepts canonical base-10 integers, rejects decimals/whitespace/overflow, and falls back to `400000000` when the stored value is missing or corrupt.
- The workbook limit is captured once per export and reused through Excel generation and partitioning; `400000000` is the default, and `400000000` itself is rejected because the rule is strict `< limit`.
- Generation now happens before persistence, so a failed workbook does not leave transaction-code writes behind.
- OnePay is isolated from the manual MBank workbook limit path instead of reading a control it should never need.
- The Settings UI now retries load failures at the page level on both desktop and mobile, and the VND card keeps canonical integer state with 44px controls.

## What We Tried

- Seeded the setting through both migration and bootstrap seeder, then made the down migration delete only the exact untouched seed row.
- Added focused tests for parser boundaries, generation-failure no-write behavior, exact-key settings loading, and desktop/mobile page parity.
- Updated the QA plan to describe the configurable strict threshold instead of the old hardcoded limit.

## Root Cause Analysis

The original design assumed a fixed ceiling and let that assumption leak across config, export, and UI. That was the real mistake. Once the limit became user-configurable, the export path had to stop relying on cached read-side convenience and start treating the setting like an accounting boundary. Review caught the parts that would have made this brittle: stale reads, non-canonical numeric input, and seed deletion that could have stomped an admin edit.

## Lessons Learned

- Money thresholds need authoritative reads, not “probably fresh” caches.
- Canonical integer validation is cheaper than recovering from a bogus VND string later.
- If generation can fail, persistence must wait.
- If a setting is manual-MBank-specific, keep it out of OnePay paths.
- Page-level retry is not optional for shared settings screens; a blank field is a lie.

## Next Steps

- Keep watching for any follow-up export split regressions or admin-edit persistence issues.
- Run browser QA later when the authenticated environment is available; it was not claimed in this step.
- Treat any future threshold change as a full contract change, not a UI-only tweak.
