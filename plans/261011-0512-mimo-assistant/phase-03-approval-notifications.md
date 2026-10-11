---
title: "Phase 3: Approval Notifications & Cards"
status: todo
---

# Phase 3: Approval Notifications & Cards

## Overview

Makes approvals visible: push + in-app notifications, Tổng Quan approval cards (admin desktop + mobile), approve/reject-with-note UI, and a partner-side "my requests" view.

## Requirements

- [ ] New request → push notification to admins via existing `PushService` (VAPID) + in-app notification; outcome → notify requester (`NotificationTypeApprovalRequest`/`ApprovalResult` constants already exist in `domain/notification.go` — free real estate)
- [ ] Tổng Quan card listing pending approval requests with approve/reject actions; reject opens note dialog (note optional-but-captured; used as retry guidance in phase 4). **Card is bounded**: top 5 + "Xem tất cả" deep link to a dedicated list surface — 12 overnight sao kê requests must not turn the dashboard into the page. Investigate `/admin/approvals` (currently renders `TimesheetPage`) first; the card must **aggregate across request types** (new `approval_requests` + existing `timesheet_edit_requests` where feasible) so "what's waiting on me" is one queue, not three
- [ ] Respect the binding **≤2 dashboard cards** constraint (UU migration plan): the approval surface lives inside the existing priority-list zone (`DashboardPriorityList` precedent, "Bảng công chờ duyệt") or replaces a card — never adds a third AI card alongside the digest + existing banner
- [ ] Escalation banner (retry-exhausted requests) on Tổng Quan
- [ ] Partner dashboard: "Yêu cầu của tôi" card — own requests + status + decision note
- [ ] Desktop + mobile for both roles; Vietnamese copy; 44px touch targets

## Implementation Steps

1. Backend notification wiring in `approval` service: `CreateNotificationWithContentType` (`notification/service.go`) on create/decision/escalation events.
2. Frontend type plumbing: add to `NotificationType` union (`types/api/notification.types.ts`), `notification-styles.ts`, `notification-navigation.ts` (deep link to the approval section). No new files per repo rule ("all notification types live here").
3. Admin dashboard: `ApprovalRequestsCard` into the bento grid of `pages/admin/DashboardPage/index.tsx` (priority-list precedent, `DashboardPriorityList` "Bảng công chờ duyệt") + `pages/mobile/admin/DashboardPage/index.tsx` disclosure section.
4. Decision UI: approve/reject modal with note textarea (precedent: note handlers in `pages/admin/AdvancePaymentsPage/index.tsx:232,244`).
5. Partner surfaces: card on `pages/partner/DashboardPage/index.tsx` + `pages/mobile/partner/DashboardPage/index.tsx`.
6. Verify existing `/admin/approvals` route semantics before reuse (timesheet approvals live there; `components/approvals/` kit appears unused — confirm, then reuse or replace); keep notification deep links valid.
7. Parity + unit tests (`*.parity.test.tsx` precedent, `NotificationItem.test.tsx` template).

## Related Code Files

- Create: `frontend/src/components/admin-dashboard/ApprovalRequestsCard.tsx`, `frontend/src/components/approvals/ApprovalDecisionModal.tsx`, partner "my requests" card component, `hooks/api/useApprovals.ts`
- Modify: `types/api/notification.types.ts`, `utils/notification-{styles,navigation}.ts`, both DashboardPage variants, `config/api.config.ts`, `services/api/` (new `approvals.service.ts`)

## Success Criteria

- [ ] Request → push arrives (web-push verified), card appears without reload (3-min polling bound documented)
- [ ] Approve/reject-with-note works from cards on admin desktop + mobile
- [ ] Partner sees own request status + decision note on desktop + mobile
- [ ] Parity tests green; 320px overflow check clean

## Risks

- Two FABs (bell + chat in phase 5) — settle placement now: desktop FAB cluster, mobile keeps bottom-nav bell; chat placement decided in phase 5.
- `components/approvals/` may be dead code — do not build on it until verified.
