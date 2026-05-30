package onepay

// Sentinel errors for webhook/IPN verification.
// Full implementation in TASK-031 (webhook parser).

import "errors"

var (
	ErrInvalidWebhookSignature = errors.New("onepay: invalid webhook signature")
	ErrExpiredWebhook          = errors.New("onepay: expired webhook timestamp")
	ErrMalformedWebhook        = errors.New("onepay: malformed webhook payload")
)
