# Admin flexible BCC import

## Scope

Add an Admin-only option to process flexible-pay employees from a BCC upload. It creates only missing timesheet records and never replaces or duplicates an existing record. Partner uploads retain the current behaviour.

## Plan

1. Carry the Admin-selected option through the upload request, idempotency fingerprint, and durable import job.
2. Apply missing-only handling to legacy, multi-position, and weekly BCC parsing paths.
3. Cover the permission and duplicate-safety contract with focused backend and frontend tests; run relevant checks.

## Acceptance criteria

- Only Admin desktop and mobile upload dialogs expose the option.
- Non-admin callers cannot enable it through the API.
- With the option enabled, flexible-pay rows create only when no matching timesheet exists; existing pending, approved, or paid data remains untouched.
- Default BCC imports preserve the current skip-for-flexible behaviour.
