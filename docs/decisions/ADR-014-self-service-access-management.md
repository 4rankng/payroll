# ADR-014: Self-Service Access Management Without Approval

**Date:** 2026-11-24
**Status:** Accepted

## Context

Partners need to manage employees and projects they did not create. The
duplicate-CCCD dialog ("Nhân viên với CCCD này đã tồn tại trong hệ thống")
offered a "Yêu cầu quản lý" action whose mental model implied a request that
someone must approve, and project-management asks had no first-class path at
all — they degraded into the generic approval engine (`general_request`)
where they sat in the admin queue. The product owner ruled on 2026-11-24:

> "user can request to manage employee and manage project without the need to
> get approve, it is not needed"

and, in the same ruling, extended self-service to removal: a user may also ask
to be removed from a project and from employee management, granted instantly.

## Decision

Employee and project access management is fully self-service for partners.
Requesting is granting — there is no approval state, queue, or decision step
anywhere in this flow:

| Action | Endpoint | Effect |
|--------|----------|--------|
| Claim employee | `POST /employees/:id/request-access` | `employee_users` row, self-granted, instant |
| Release employee | `POST /employees/:id/release-access` | Drops own relation, instant |
| Claim project | `POST /projects/:id/request-access` | `project_users` row, self-granted, instant |
| Release project | `POST /projects/:id/release-access` | Drops own grant, instant |

Guards that remain (service layer, not approvals):

- Only partner-role users claim/release via self-service; admins use the
  explicit grant/revoke paths.
- Creators cannot self-release their own employee or project — intrinsic
  ownership stays (project owner-privilege strip remains admin-only, PAYR-30).
- All four endpoints are idempotent.
- Every action publishes the same domain events as explicit grant/revoke
  (`EmployeeAccessGrantedEvent`, `EmployeeAccessRevokedEvent`,
  `ProjectAccessGrantedEvent`, `ProjectAccessRevokedEvent`) so the audit log
  records who requested, when, and what was granted/removed.

UI contract: after the action the result is stated plainly ("Đã cấp quyền quản
lý…" / "Đã loại bạn khỏi quản lý…") and the employee/project appears in (or
disappears from) the caller's list immediately — list caches invalidate in the
same mutation. A released project closes the details sheet because the caller
can no longer read it.

The generic approval engine is untouched. Its lanes — sao kê/payroll email
filing, send_reminder, void_timesheet, and the chat-assistant `general_request`
catch-all — are different flows and keep their queues and Duyệt/Từ chối
decisions. No approval queue or decision UI exists for employee/project
management requests, and none may be reintroduced without superseding this
ADR.

## Consequences

- Positive: zero-latency access for partners; no admin bottleneck on
  onboarding; a single uniform mental model ("yêu cầu là được") across claim
  and release.
- Positive: audit parity with explicit grants — the log answers "ai yêu cầu,
  lúc nào, được gì" without any new table.
- Neutral: partners can add themselves to any project's management list and
  drop themselves from any relation they hold. Access revocation from others
  stays with admins/creators/managers, so the blast radius of self-service is
  bounded to the caller's own access.
- Neutral: the per-project authorization middleware exempts
  `/request-access` and `/release-access`; the service layer re-checks role,
  creator protection, and idempotency on every call (same pattern as the
  pre-existing employee `/request-access` exemption).

## Alternatives Considered

- Keep an approval queue for management requests — rejected by the product
  owner: the approval step adds latency with no control benefit, since the
  requester is a trusted partner and removal only shrinks their own access.
- Route project-management asks through `general_request` approvals — rejected:
  execution stayed manual after approval; no first-class grant ever happened.
- Confirm-free release (no ConfirmDialog) — rejected for release actions only:
  release is destructive to the caller's own access, so a confirm click guards
  against accidental drops while remaining instant.
