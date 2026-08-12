# Configurable Self-Check-In Advance Wait

## Decision

Make the post-checkout wait before self-check-in earnings become advanceable an
Admin-managed, global whole-hour setting. The setting key is
`self_check_in_advance_hold_hours`; its default remains `24` hours.

## Scope

Included:

- Persisting and seeding the setting for existing and new databases.
- Validating Admin updates as canonical whole hours from `0` through `720`.
- Reading the setting authoritatively at checkout, then persisting the resulting
  quota-credit deadline for the worker and recovery sweep.
- Showing the same control on the desktop and mobile Admin Settings pages.
- Tests covering parsing, persistence validation, scheduling, recovery, and
  responsive Settings parity.

Excluded:

- Changing existing queued jobs after an Admin changes the value. Each checkout
  keeps its persisted deadline, and the recovery sweep uses that deadline rather
  than the current setting.
- Per-project, per-employee, or per-shift overrides.
- Changes to the day-10-to-month-end request window or the configured
  self-check-in advance percentage.

## Behavior

`0` means a completed checkout is eligible for quota credit immediately. A
positive value schedules quota credit that many hours after checkout. The
calculated deadline is persisted atomically with the checkout; both the worker
and recovery sweep enforce it, even if the Admin changes the setting later.

Missing, malformed, or out-of-range persisted values fail safe to 24 hours for
reads. Admin API writes are rejected rather than being silently coerced.

## Data Flow

```text
Admin Settings (desktop/mobile)
  -> settings API validation and transaction
  -> settings table: self_check_in_advance_hold_hours
  -> authoritative configuration accessor
  -> persisted checkout eligibility deadline
  -> worker and overdue-credit recovery
  -> advance-payment quota becomes available
```

## Verification

- Backend unit coverage proves 0, a configured non-default value, and invalid
  fallback behavior.
- Settings integration coverage proves the API rejects invalid values and saves
  a valid value.
- Frontend tests prove the card exists with the same label, bounds, and saved
  state on desktop and mobile.
- Type, lint, targeted backend, integration, graph update, and production
  health checks are run before release.
