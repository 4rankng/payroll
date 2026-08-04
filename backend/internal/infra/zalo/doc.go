// Package zalo implements the stateless Zalo Notification Service (ZNS) protocol
// client used by the employee password-reset flow.
//
// This package is a Go port of the proven PHP sender in the sister codebase
// tuyennhanvien.vn (wp-content/themes/vfic/inc/functions/functions-zns.php).
// Only the protocol is ported — credentials live in the payroll project's own
// OA ("Ting Ting Software Solution" / app "TingTing Soft").
//
// Endpoints:
//   - Send:    POST https://business.openapi.zalo.me/message/template
//   - OAuth:   POST https://oauth.zaloapp.com/v4/oa/access_token
//
// The package is intentionally free of DB/config imports. A Provider takes its
// credentials via the CredentialSource interface (implemented by the
// zaloconnect service in app/services/zaloconnect, which backs onto the
// generic settings table). This keeps the protocol layer unit-testable with a
// fake credential source and an httptest.Server standing in for Zalo.
//
// Token lifecycle:
//   - access_token is short-lived (typically ≤ 25h, often ~1h per Zalo docs).
//   - refresh_token is ONE-SHOT — every successful refresh rotates it, and the
//     new pair must be persisted atomically before any further call.
//   - Provider.mu serializes refreshes so two concurrent -124 retries cannot
//     double-spend the single-use refresh_token.
//
// Error handling: business errors from Zalo (e.g. -118 "no Zalo account",
// -115 "insufficient quota") are returned as SendResult.ErrorCode != 0, NOT as
// Go errors. Only transport/Go-level failures (network, malformed response,
// credential-source outage) surface as errors. Callers decide what to do with
// a non-zero ErrorCode.
package zalo
