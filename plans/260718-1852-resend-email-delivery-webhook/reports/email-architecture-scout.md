# Email Architecture Scout

## Existing Flow

- `backend/internal/infra/email/resend_provider.go:23` returns Resend's `resp.Id` through `domain.EmailDeliveryResult.MessageID`.
- `backend/internal/app/services/notification/email_service.go:612` sends and publishes `EmailSentEvent` after provider acceptance.
- `backend/internal/app/services/notification/email_publisher.go:68` persists the ID in `notifications.resend_message_id`.
- `backend/internal/app/services/notification/email_service.go:328` serves those rows through the email-history API.
- `frontend/src/types/api/email.types.ts:36` already anticipates a `status` field, but the backend DTO does not provide it and the email-history UI does not render it.

## Gaps

- No delivery status, provider event timestamp, failure details, or indexed lookup by Resend email ID.
- No durable `svix-id` deduplication. Redis-only dedupe would lose auditability and raced events.
- Resend can call back before the notification record is committed, and notification persistence errors are currently logged without failing the send.
- OTP uses Resend directly and creates no email-history row; OTP tracking requires a separate privacy and correlation decision.

## Existing Patterns to Reuse

- Public static provider routes: `backend/internal/app/bootstrap/routes_disbursement.go`.
- Raw-body verification before parsing: `backend/internal/transport/http/handlers/disbursement/webhook.go:118`.
- Immutable webhook receipt/audit rows: wallet IPN domain and persistence packages.
- Repository-backed email history: `domain.NotificationRepository` and `infra/persistence/notification_repository.go`.

## Recommended Scope

- Track only emails already persisted in admin email history.
- Add a small durable Resend webhook inbox keyed by `svix_id`, without storing full payload or recipient addresses.
- Apply normalized current status to the matching notification using provider event time so older callbacks cannot regress newer state.
- Keep unmatched verified receipts for reconciliation; do not expose OTP recipients or bodies.
- Add current status to the existing API and both email-history surfaces.

