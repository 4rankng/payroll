# ADR-013: BCC Import "Add on Top" Supplement Mode

Date: 2026-11-11
Status: Accepted

## Context

The BCC import dedup system (import the same file repeatedly without creating
duplicate rows) intentionally protects finalized timesheets: any uploaded cell
whose (employee, date) key matches an approved/paid row is counted "bỏ qua"
(skipped) by `planBCCReplacement`. Partners hit this legitimately: after a week
is approved and paid (e.g. Samsung SDS week 1–7 Nov 2026, transaction 368),
they upload a supplement file intending to ADD hours, and all matched cells
are silently skipped.

## Decision

The upload dialog gains an explicit opt-in checkbox (`add_on_top` form field,
persisted on `timesheet_import_jobs`). When ticked:

- **Approved/paid rows are never mutated.** A positive-hours cell matching a
  protected row passes through the replacement plan marked `AddOnTop`, and
  bulk create inserts a NEW pending supplement row (same paytype, file hours)
  next to the finalized row. The paid row keeps its `amount` and
  `payment_status`; the delta goes through approval and the next payment run.
- **Pending rows still replace.** Re-uploading the same file with the tick
  remains deduped — drafts are replaced, never summed, so the tick cannot
  double draft hours.
- **Zero-hour cells (delete requests) keep the protected skip.** Deleting
  approved/paid hours is not an addition, and the bulk path refuses paid-row
  deletes anyway.
- No database unique constraint exists on (project, employee, date, paytype);
  supplement rows rely on the in-batch `existingByPaytypeMap` bypass in
  `BulkCreateTimesheetsInTransaction` and the 24h/day cap, which now counts
  existing finalized hours for add-mode keys (`willBeModified` ignores
  AddOnTop requests).

## Consequences

- Two rows can exist for one (employee, date, paytype): the finalized
  original plus a pending supplement. Grouping UIs and payment rollups sum
  hours, so totals stay correct; the supplement is separately auditable.
- The partner-visible "bỏ qua" count drops to only zero-cell and flexible
  skips when the tick is used; supplement creations count under "tạo mới".
- Ticking is the uploader's acknowledgement that the file's hours are
  incremental additions, not corrections. Corrections to finalized rows
  remain an admin action (reset/reject), unchanged by this feature.
