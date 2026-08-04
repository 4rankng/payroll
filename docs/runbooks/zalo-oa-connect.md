# Runbook: Connect Zalo ZNS for Employee Password Reset

This runbook walks an admin through connecting the Zalo OA so employees can
reset their passwords via Zalo ZNS OTP. **No shell or SQL access required** —
everything is done through the admin UI.

## Prerequisites

1. A verified **Zalo OA** ("Ting Ting Software Solution" / app "TingTing Soft")
   with the ZNS (Zalo Notification Service) permission enabled.
2. The **template `617976`** (`OTP-ZNS-v1`) must be **approved** (status
   "Đã duyệt"). It was submitted 2026-08-04 and takes 2–3 business days for Zalo
   review. Check the OA Console → Quản lý mẫu ZBS. Prod go-live is blocked until
   this flips; dev/test can proceed in sandbox (Zalo returns `-127`, treated as
   non-fatal).

## Steps

### 1. Open the Zalo settings tab

Navigate to **Admin → Cài đặt → Zalo ZNS** (`/admin/settings?tab=zalo`).

### 2. Register the OAuth callback URL

In the Zalo OA Console, under the app/permission settings, add the callback URL
shown in the settings tab as an allowed `redirect_uri`:

```
https://tingting.vip/admin/settings?tab=zalo
```

> **Why a frontend URL, not an API URL?** Zalo does a full browser redirect to
> this URL with `?code=...&state=...` appended. Our API uses header-based JWT
> auth (`Authorization: Bearer`), not cookies — so a direct redirect to an API
> endpoint would arrive with no auth header and get 401. By redirecting to the
> SPA route, the SPA loads, reads `code`+`state` from the URL, and POSTs them
> to `POST /api/v1/admin/zalo/oauth/callback` via axios (which attaches the
> admin's JWT from localStorage).

Use the **Copy** button next to the Callback URL field in the settings tab to
copy it exactly. The scheme + host + path must match what's registered in the
OA Console.

### 3. Enter credentials

Fill in:
- **App ID** — from the OA Console app settings.
- **Secret Key** — from the OA Console (this is write-only; the field is blank
  after save for security).
- **Template ID** — `617976` (pre-filled; change only if using a different template).

Click **Lưu thông tin**. The status badge should flip to "Chưa kết nối"
(configured but not connected).

### 4. Connect (OAuth)

Click **Kết nối Zalo**. Your browser redirects to Zalo's permission page.
Authorize the app. Zalo redirects back to the settings tab with
`?zalo_connected=1` and the status badge flips to "Đã kết nối" with a live
token expiry.

> If you see `?zalo_error=...`, the connect failed. Common causes:
> - Callback URL not registered in the OA Console (step 2).
> - App ID / Secret Key incorrect (re-enter and save).
> - The `state` expired (5-min TTL) — just click Kết nối Zalo again.

### 5. Enable the feature

Toggle **Bật tính năng** to ON. A confirmation dialog appears when disabling
(employees won't be able to reset via Zalo until re-enabled). The toggle is
**hot** — it takes effect on the next request, no redeploy.

### 6. Smoke test

As an employee (or a test employee account):
1. Go to `/login` → click **Quên mật khẩu?**
2. Enter the employee's registered **mobile number**.
3. You should be navigated to `/zalo-reset-password` — check the Zalo inbox for
   the OTP message (it carries `otp_code`, the employee's name, and "Hết hạn
   (phút)").
4. Enter the 6-digit code + a new password → login with the new password.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| Status: "Lỗi" with `-124` | Access token expired + refresh failed | Click **Làm mới token**; if that fails, re-run Kết nối Zalo (step 4). |
| Status: "Lỗi" with `-118` | The employee's phone has no linked Zalo account | Not a code issue — the user must install Zalo / link their number, or use the email channel. |
| Status: "Lỗi" with `-115`/`-137` | Insufficient ZBS balance | Top up the ZBS account in the OA Console. |
| Status: "Lỗi" with `-131` | Template not approved yet | Wait for Zalo review (2–3 business days). |
| Status: "Lỗi" with `-127` | Sandbox mode — template test only sends to OA admins | Expected in dev; use an OA-admin Zalo account for testing. |
| OTP never arrives but status is "Đang hoạt động" | Zalo delivered but user didn't see it / phone has no Zalo | Check the user's Zalo app; have them search the OA name. |

## Disaster recovery

If the refresh_token is lost (e.g. exhausted by a crash mid-refresh, or the
settings row is corrupted), the admin cannot refresh and the connection is dead.

**Option A — Re-OAuth (preferred):**
1. In the settings tab, click **Kết nối Zalo** again. The old dead refresh_token
   is replaced by a fresh pair on success.

**Option B — Re-seed from env (if the settings row is deleted):**
1. Set the `ZALO_*` env vars to known-good values.
2. Delete the `zalo.credentials` and `zalo.enabled` rows from the `settings`
   table (SQL access required).
3. Restart api-server — `SeedFromEnvIfEmpty` repopulates the rows.
4. Re-run the connect flow (step 4 above) to get fresh OAuth tokens.

**Option C — Direct DB write (last resort):**
```sql
UPDATE settings SET value = '<json-with-new-tokens>' WHERE `key` = 'zalo.credentials';
```
Document the reason in an incident note; never use routinely.
