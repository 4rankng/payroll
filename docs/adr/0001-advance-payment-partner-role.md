# ADR 0001: Advance Payment Partner Role

## Status
Proposed

## Context
We need a dedicated role for external partners who manage the advance payment (FlexPay) pipeline. Currently only `admin` can access advance payment endpoints. The existing `partner` role is scoped to specific projects via `project_users` and has no advance payment access.

The `tester` role was added for 9Pay disbursement debugging but is unused in production.

## Decision

1. **Remove `tester` role** from the `users.role` enum and all code references.
2. **Add `adv_partner` role** — a new role in the enum: `enum('admin','partner','employee','adv_partner')`.
3. **Casbin policy** grants `adv_partner` read + import + export access to advance payment endpoints only. Financial actions (cancel, settle, upload-result, send-email) remain admin-only.
4. **No `project_users` scoping** — `adv_partner` sees all projects that have advance payment data.
5. **Shared frontend** — same admin panel, role-gated menus.

## Consequences

- Migration must alter the enum column. Any existing `tester` users must be reassigned or deleted before migration.
- Frontend needs to add `adv_partner` to role-based menu rendering.
- Casbin policy file gets a new block for `adv_partner`.
- `authorization.go` middleware needs no changes — `adv_partner` doesn't need project-level or employee-level access checks (no per-project scoping).
