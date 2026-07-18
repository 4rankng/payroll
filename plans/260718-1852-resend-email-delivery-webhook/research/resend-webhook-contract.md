# Resend Webhook Contract

## Summary

- `email.sent`: Resend accepted the send and will attempt delivery; not final delivery.
- `email.delivered`: recipient mail server accepted the message; final success signal.
- Track `email.delivery_delayed`, `email.bounced`, `email.failed`, and `email.suppressed` so non-delivery is not mislabeled as success.
- Delivery is at least once and ordering is not guaranteed. Deduplicate by `svix-id` and order state changes by provider event time.
- Verify the raw request body with `svix-id`, `svix-timestamp`, `svix-signature`, and a distinct webhook signing secret.
- Return `200 OK` only after the verified event is durably accepted.

## Payload Correlation

- Match the existing `notifications.resend_message_id` to `data.email_id`.
- Do not use `data.message_id` as the application correlation key; it is a separate provider message identifier.
- Record top-level `created_at` as the provider event time.

## Existing SDK Capability

The pinned `github.com/resend/resend-go/v2 v2.28.0` already exposes `client.Webhooks.Verify` with a five-minute timestamp tolerance. No new signature dependency is required.

## Official References

- https://resend.com/docs/webhooks/event-types
- https://resend.com/docs/webhooks/emails/sent
- https://resend.com/docs/webhooks/emails/delivered
- https://resend.com/docs/webhooks/emails/delivery-delayed
- https://resend.com/docs/webhooks/emails/bounced
- https://resend.com/docs/webhooks/emails/failed
- https://resend.com/docs/webhooks/emails/suppressed
- https://resend.com/docs/webhooks/verify-webhooks-requests
- https://resend.com/docs/webhooks/retries-and-replays

