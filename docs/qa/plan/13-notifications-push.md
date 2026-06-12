# 13 — Notifications & Push

**Priority**: P2 | **API Prefix**: `/api/v1/notifications` | **Risk Level**: Low

## Business Rules

- Notifications support both in-app and push (Web Push VAPID)
- Push notifications supported in Safari and Chrome
- Admin can create custom notifications targeting all admins
- Employee transfer notifications sent automatically
- Unread count tracked per user
- Mark all as read available
- Title is required for custom notifications

## State Machine

```
Notification: [unread] → (mark read) → [read]
```

## Test Scenarios

### F23 — Notifications

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F23-01 | Get unread count | Happy | GET `/notifications/unread-count` | 200, count of unread notifications |
| F23-02 | Create custom notification (to all admins) | Happy | POST `/notifications` with title, message | 201, notification sent to all admin users |
| F23-03 | List notifications | Happy | GET `/notifications` | 200, paginated notification list |
| F23-04 | Get unread notifications | Happy | GET `/notifications?status=unread` | 200, filtered unread list |
| F23-05 | Mark all as read | Happy | POST `/notifications/mark-all-read` | 200, all notifications marked read |
| F23-06 | Create notification without title | Negative | POST without title field | 400, title required |
| F23-07 | Create notification with empty title | Negative | POST with title="" | 400, title required |
| F23-08 | Push notification delivery | Integration | Create notification → check push delivery | Web Push notification received |
| F23-09 | Unread count updates after marking read | Integration | Mark all read → GET unread-count | Count = 0 |

## Automated Test Reference

- Integration test: `backend/tests/integration/flow_notification.go`
