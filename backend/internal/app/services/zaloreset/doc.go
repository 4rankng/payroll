// Package zaloreset implements the self-service Zalo-OTP password-reset flow
// for employee-role users whose only contact channel is a mobile number.
//
// A user requests a reset from the login page; the service looks up the mobile
// (scoped to role=employee), generates a 6-digit code, stores its hash in Redis
// under an opaque session id, and asynchronously dispatches the code via Zalo
// ZNS. The client then submits the session id + code + new password to confirm,
// which atomically consumes the session and updates the password + invalidates
// all existing sessions in one DB transaction.
//
// Security contract (mirrors the email passwordreset package):
//   - Anti-enumeration: RequestReset ALWAYS returns nil and dispatches a
//     structurally identical response for known and unknown mobiles. Not-found
//     and disabled-toggle paths perform a dummy Redis write to equalize timing.
//   - Single-use: Consume uses a Lua compare-and-delete so a code can succeed
//     at most once. A wrong code does NOT consume the session (user may retry;
//     the rate limiter bounds total attempts).
//   - Atomic password update: password + tokens_invalid_before commit together
//     in a single transaction (UpdatePasswordAndInvalidateSessions).
//   - Role gate: only role=employee is eligible. Admin/partner mobiles go down
//     the not-found path and are never enrolled in the Zalo flow.
//   - Hot toggle: RequestReset checks the admin-managed enabled flag on every
//     call (no redeploy to flip).
package zaloreset
