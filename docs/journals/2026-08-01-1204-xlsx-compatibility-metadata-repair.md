# XLSX Compatibility Metadata Repair

**Date**: 2026-08-01 12:04 +08
**Severity**: Medium
**Component**: Payroll Excel templates
**Status**: Resolved

## What Happened

Opening the payroll workbook in Excel produced the content-recovery popup instead of a clean file open. The failure was not in the visible cells; it was in the workbook XML metadata on the `Summary` sheet, where `mc:Ignorable` referenced `x14ac`, `xr2`, and `xr3` without matching namespace declarations. Excel treated that as broken compatibility metadata and offered to repair the file on open.

The fix was intentionally narrow: repair the template metadata, not the sheet layout. The workbook was reduced to `mc:Ignorable="xr"` on the affected sheet so the declared ignorable prefix actually matches the namespace present in the part.

## The Brutal Truth

This was a dumb XML-level trap. The sheet looked fine until Excel decided to be strict, and then the whole file became suspect because one attribute lied about its own namespaces. That is exactly the kind of invisible workbook corruption that wastes time because the UI failure looks dramatic while the real bug is one bad string in a ZIP entry.

## Technical Details

- Root cause: `Summary` sheet compatibility metadata declared stale ignorable prefixes: `x14ac`, `xr2`, and `xr3`.
- Repair: narrow the template metadata to `mc:Ignorable="xr"` instead of carrying dead prefixes.
- Regression coverage: the new XML test went red on the broken metadata and green after the repair by asserting every `mc:Ignorable` prefix has a matching `xmlns:` declaration.
- Validation: the repaired workbook opened cleanly in Excel for Mac and LibreOffice after the metadata fix.
- API-level integration could not be re-run because the local backend on `:8080` was not running.

## What We Tried

1. Verified the workbook-level failure rather than guessing from the visible spreadsheet layout.
2. Added a focused XML regression test to catch undeclared ignorable prefixes before Excel does.
3. Repaired only the compatibility metadata instead of regenerating the workbook or touching the cell content.

## Root Cause Analysis

The workbook carried stale Office compatibility cruft from a prior export path. Nobody cleaned up the `mc:Ignorable` list when the workbook contents changed, so the metadata outlived the namespaces it referenced. Excel Mac exposed the problem immediately; LibreOffice confirmed the file was not clean.

## Lessons Learned

- XML metadata can break a workbook even when every cell looks correct.
- If a template has `mc:Ignorable`, it needs a regression test, not trust.
- Narrow the repair to the broken workbook part; do not “fix” this by changing visible sheet structure unless the layout is also wrong.

## Next Steps

Keep the XML compatibility regression in place so the next export path that drags in stale `mc:` prefixes fails in tests instead of in Excel. Re-run the API integration once the local backend is actually up on `:8080`.
