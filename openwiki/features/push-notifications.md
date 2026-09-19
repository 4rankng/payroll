---
type: feature
title: Web Push Notifications
description: Web Push subscription storage (VAPID), the notification fan-out path, the PWA service worker that receives the push, and the relationship with in-app and email channels.
tags: [feature, push, web-push, vapid, pwa, notification]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-4310adf7e826f078b900920f
    resource: repo://backend/internal/app/services/notification/employee_notification_service.go
  - id: openwiki-source-a32d7a799824994abad1c7ec
    resource: repo://backend/internal/app/services/push/push_service.go
  - id: openwiki-source-dc86f34e915ec816019a57db
    resource: repo://backend/internal/domain/push_subscription.go
  - id: openwiki-source-94818d82bdf61ffa53d6fc57
    resource: repo://frontend/src/sw.ts
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Feature: Web Push Notifications

The web-push subsystem delivers lightweight messages to a user's browser even when the PWA tab is closed. It coexists with in-app notifications (served over the API) and email (Resend in production). The push path is for time-sensitive pings — FlexPay approval, timesheet reminder, salary cycle update — while in-app notifications own the persistent history and email owns the long-form receipt.

## Subscription storage

`internal/domain/push_subscription.go` defines `PushSubscription`:

```
push_subscriptions(
  id, user_id, endpoint, p256dh, auth,
  device_type, created_at, updated_at
)
```

- `UNIQUE INDEX (user_id, endpoint)` — one row per (user, push endpoint).
- `device_type` is `web` (default), `android`, or `ios`. The web path uses Web Push with VAPID keys; native device types use the same record with platform-specific payload encoding.
- The repository contract (`PushSubscriptionRepository`): `Create`, `GetByUserID`, `DeleteByEndpoint`, `DeleteByUserID`. Subscriptions are looked up per user, not globally.

`internal/app/services/push/push_service.go` (`PushService`) wraps the repository and the VAPID config (`config.NotificationConfig`). It exposes `Subscribe`, `Unsubscribe`, `SendToUser` (fan out to every subscription for the user), and platform-specific payload shaping. The VAPID library is `github.com/SherClockHolmes/webpush-go`.

## Notification fan-out

`internal/app/services/notification/` is the broader notification package. It owns:

- `employee_notification_service.go` — per-employee notification assembly (in-app + push).
- `email_service.go` / `email_publisher.go` — email channel.
- `payment_cycle_publisher.go` — payment-cycle events that include push fan-out.
- `payroll_report_adapter.go` — payroll email/push bridge.
- `loan_repayment_reminder_service.go` — loan reminder scheduling.
- `bank_info_e2e_test.go` — end-to-end test fixture for the bank-info notification path.

The fan-out runs after the originating transaction commits. The pattern is the same as everywhere else: emit a domain event, the event handler claims the work, then the notification service assembles the message and calls the push service (and the email publisher, and the in-app store).

The wallet settlement event, the FlexPay approval event, and the timesheet import completion event all have notification handlers. See `integrations/event-bus.md` for the event handler registry.

## PWA service worker

`frontend/src/sw.ts` is the service worker that receives the push. The PWA stack (vite-plugin-pwa + Workbox) caches the app shell, and the service worker handles `push` events from the browser's Push API.

The end-to-end shape:

1. The frontend registers a service worker, calls `pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: vapidKey })`, and posts the resulting subscription to `/api/v1/push/subscribe` (handled by `PushService.Subscribe`).
2. The backend stores the subscription keyed by the user ID.
3. When an event handler decides to send a notification, it iterates the user's subscriptions and calls `webpush.SendNotification` for each, signing with the VAPID private key.
4. The browser wakes the service worker; the service worker renders a `Notification` with the payload's title/body/icon/tag and routes clicks back to the relevant in-app page.

Stale or expired endpoints return 404/410 from the push service and are removed via `DeleteByEndpoint`. This keeps the subscription table healthy without manual cleanup.

## Where it lives

- **Backend**:
  - `internal/domain/push_subscription.go` — entity + repository.
  - `internal/domain/notification.go` — the broader notification entity (in-app).
  - `internal/app/services/push/push_service.go` — VAPID send + subscribe/unsubscribe.
  - `internal/app/services/notification/` — fan-out and assembly.
  - `internal/infra/email/` — email channel adapter.
- **Frontend**:
  - `frontend/src/sw.ts` — service worker push handler.
  - `frontend/public/manifest.webmanifest` (PWA manifest) and the workbox-generated service worker registration in `frontend/src/main.tsx`.
  - The settings UI where the user grants notification permission.

## Operational notes

- VAPID keys are configured via `config.NotificationConfig` and must be present in production. The build fails fast if the keys are missing in any environment where push is enabled.
- Failed deliveries (4xx/5xx from the push gateway, or 404/410 from the browser) are removed from the store; transient failures are retried by the originating event handler.
- The push channel is best-effort. Authoritative delivery — payroll confirmations, FlexPay settlement — also writes an in-app notification so the user sees the message next time they open the app.

See `frontend/shared-platform.md` for the broader PWA stack (workbox, manifest, offline cache) and `architecture/infrastructure.md` for the email and notification adapter layer.
