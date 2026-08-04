// Package zaloconnect implements the admin-managed Zalo OA connection layer.
//
// It persists the Zalo OA credentials + the runtime feature toggle in the
// existing generic settings table (two rows: "zalo.enabled" boolean and
// "zalo.credentials" JSON), implements zalo.CredentialSource for the infra/zalo
// Provider, and exposes the admin SaveCredentials / TestSend / RefreshNow
// actions. The admin pastes all four OA fields (app_id, secret, access_token,
// refresh_token) directly into the settings UI — there is no OAuth
// authorization-code flow. Token rotation happens automatically via the
// refresh_token grant inside the Provider on each Send.
//
// This package is the runtime authority: env vars (ZALO_*) seed the settings
// rows on first boot only; thereafter the admin UI (via this service) is the
// single source of truth. Toggling zalo.enabled or rotating tokens takes effect
// immediately — no redeploy.
//
// Security:
//   - Secret fields (secret_key, access_token, refresh_token) are NEVER returned
//     by GetStatus — it produces a masked view for the admin UI.
package zaloconnect

