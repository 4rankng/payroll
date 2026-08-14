# Runbook: Connect Zalo ZNS for Employee Password Reset

This runbook walks an admin through connecting the Zalo OA so employees can
reset their passwords via Zalo ZNS OTP. **No shell or SQL access required** —
everything is done through the admin UI.

## Prerequisites

1. A verified **Zalo OA** ("Ting Ting Software Solution" / app "TingTing Soft")
   with the ZNS (Zalo Notification Service) permission enabled.
2. The **template `619684`** (`OTP-ZNS-v2`) must be **approved** (status
   "Đã duyệt"). It was submitted 2026-08-04 and takes 2–3 business days for Zalo
   review. Check the OA Console → Quản lý mẫu ZBS. Prod go-live is blocked until
   this flips; dev/test can proceed in sandbox (Zalo returns `-127`, treated as
   non-fatal).

## Steps

### 1. Open the Zalo settings tab

Navigate to **Admin → Cài đặt → Zalo ZNS** (`/admin/settings?tab=zalo`).

### 2. Enter credentials

Fill in:
- **App ID** — from the OA Console app settings.
- **Secret Key** — from the OA Console (this is write-only; the field is blank
  after save for security).
- **Access Token** and **Refresh Token** — retrieve them as one pair for this
  OA and app. Do not use a refresh token that has already been exchanged by
  the OA Console or another system.

Treat that rotating refresh-token chain as single-writer state. Do not paste
the same pair into Payroll and ChatBot (or any other service): whichever system
refreshes first advances the chain and leaves the other system holding an
invalid token. Systems that must use the same OA need one shared token authority
or separate Zalo app/OA authorizations whose refresh chains are independent.

Click **Lưu cấu hình**. Payroll immediately exchanges the pasted refresh token
and stores the successor token pair before reporting **Đã kết nối**. If Zalo
rejects the refresh token, save fails immediately; Payroll does not wait until
the access token expires roughly one day later to reveal the broken pair.

Payroll coordinates every refresh-token exchange through Redis, so concurrent
API instances wait for the winning exchange and then re-read its stored pair.
Credential status and error updates use compare-and-swap persistence: a stale
request cannot put an already-consumed refresh token back into the settings.
These safeguards apply inside Payroll only; they do not make a token pair safe
to share with another service.

### 3. Validate the channel

Enter a controlled recipient number and select **Gửi OTP mẫu**. This verifies
the real ZNS send path without enabling either service.

**Kiểm tra cấu hình đã lưu** deliberately performs another coordinated token
exchange without sending a message. Normally this is unnecessary immediately
after saving because save already validated and rotated the pair.

### 4. Enable the required service

Toggle **Bật tính năng** to ON. A confirmation dialog appears when disabling
(employees won't be able to reset via Zalo until re-enabled). The toggle is
**hot** — it takes effect on the next request, no redeploy.

### 5. Smoke test

As an employee (or a test employee account):
1. Go to `/login` → click **Quên mật khẩu?**
2. Enter the employee's registered **mobile number**.
3. You should be navigated to `/zalo-reset-password` — check the Zalo inbox for
   the OTP message (it carries the six-digit `otp`).
4. Enter the 6-digit code + a new password → login with the new password.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| Send fails with `-124` | Zalo rejected the access token and the paired refresh token could not recover it | Paste a newly issued access/refresh pair, save it, then send one OTP test. |
| Save or refresh validation returns `-14014` | Zalo rejected the refresh token (expired, previously exchanged, or for a different OA/app) | This is an invalid-token condition, not a temporary outage. Obtain a newly issued pair from the same OA/app and save it again. A failed save leaves the previously stored connection unchanged. |
| Payroll returns `-14014` while another app still sends normally | The same rotating token pair may be stored in more than one system; the other system advanced the chain | Choose one token owner or provision independent Zalo authorizations, then paste a newly issued pair only into its intended owner. |
| Save or refresh validation says Zalo cannot be reached, or returns an OAuth HTTP/response error other than `-14014` | Temporary OAuth transport or upstream-response failure | Wait a few minutes and retry the same operation. Do not replace a known-good pair solely because of a temporary upstream failure. |
| Status: "Lỗi" with `-118` | The employee's phone has no linked Zalo account | Not a code issue — the user must install Zalo / link their number, or use the email channel. |
| Status: "Lỗi" with `-115`/`-137` | Insufficient ZBS balance | Top up the ZBS account in the OA Console. |
| Status: "Lỗi" with `-131` | Template not approved yet | Wait for Zalo review (2–3 business days). |
| Status: "Lỗi" with `-127` | Sandbox mode — template test only sends to OA admins | Expected in dev; use an OA-admin Zalo account for testing. |
| OTP never arrives but status is "Đang hoạt động" | Zalo delivered but user didn't see it / phone has no Zalo | Check the user's Zalo app; have them search the OA name. |

## Disaster recovery

If the refresh_token is lost (e.g. exhausted by a crash mid-refresh, or the
settings row is corrupted), the admin cannot refresh and the connection is dead.

Payroll renews its Redis lease while exchanging and storing tokens, but Zalo's
external exchange and MySQL cannot form one transaction. A process crash,
ambiguous timeout, or database failure after Zalo consumes the refresh token
can still require the recovery steps below.

**Option A — Replace the token pair (preferred):**
1. Obtain a newly issued access/refresh pair for the same OA and app.
2. Paste both values in the settings tab and click **Lưu cấu hình**.
3. Send one OTP test before enabling a service.

**Option B — Re-seed from env (if the settings row is deleted):**
1. Set the `ZALO_*` env vars to known-good values.
2. Delete the `zalo.credentials` and `zalo.enabled` rows from the `settings`
   table (SQL access required).
3. Restart api-server — `SeedFromEnvIfEmpty` repopulates the rows.
4. Paste a newly issued access/refresh pair through the settings tab.

**Option C — Direct DB write (last resort):**
```sql
UPDATE settings SET value = '<json-with-new-tokens>' WHERE `key` = 'zalo.credentials';
```
Document the reason in an incident note; never use routinely.

## Production handoff after a code deployment

A code deployment does not provide or validate credentials. After the deployed
Payroll API is available, an authorized administrator must obtain **one fresh
access/refresh pair for Payroll's own Zalo App ID**, save it through Payroll,
and complete one controlled OTP test. Do not reuse a pair issued for ChatBot or
another system. This runbook describes the required operator action; it is not
evidence that a deployment or live Zalo test has occurred.
