// Package zaloconnect implements the admin-managed Zalo OA connection layer.
//
// It persists the Zalo OAuth credentials + the runtime feature toggle in the
// existing generic settings table (two rows: "zalo.enabled" boolean and
// "zalo.credentials" JSON), implements zalo.CredentialSource for the infra/zalo
// Provider, and orchestrates the admin OAuth v4 connect flow (state mint +
// callback code-exchange).
//
// This package is the runtime authority: env vars (ZALO_*) seed the settings
// rows on first boot only; thereafter the admin UI (via this service) is the
// single source of truth. Toggling zalo.enabled or rotating tokens takes effect
// immediately — no redeploy.
//
// Security:
//   - Secret fields (secret_key, access_token, refresh_token) are NEVER returned
//     by GetStatus — it produces a masked view for the admin UI.
//   - The OAuth state param is 256-bit, single-use (Redis GETDEL), binding the
//     connect flow to the admin session that initiated it (CSRF defense).
package zaloconnect
